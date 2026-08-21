package chat

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Workflow runs one chat session. Conversation history is workflow state:
// user messages arrive as updates, the LLM call happens in an activity, and
// the assistant's reply is returned as the update result. When Temporal
// suggests the event history is getting large, the workflow continues-as-new,
// carrying the conversation forward.
func Workflow(ctx workflow.Context, state State) error {
	if state.History == nil {
		state.History = []Message{{
			Role:    "system",
			Content: "You are a helpful assistant. Keep replies concise. " +
				"When a message with role \"tool\" appears, it is the result of a tool you called: " +
				"answer the user's question directly using that result.",
		}}
	}

	ended := false

	if err := workflow.SetUpdateHandler(ctx, UpdateSendMessage,
		func(ctx workflow.Context, userText string) (string, error) {
			state.History = append(state.History, Message{Role: "user", Content: userText})

			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 5 * time.Minute,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval: time.Second,
					MaximumAttempts: 3,
				},
			}
			actCtx := workflow.WithActivityOptions(ctx, ao)
			a := &Activities{}

			// Agent loop: the model either answers or requests tool calls.
			// Each tool call runs as its own activity, so it is durable and
			// visible in the workflow's event history.
			const maxToolRounds = 5
			for round := 0; ; round++ {
				var assistant Message
				err := workflow.ExecuteActivity(actCtx, a.CompleteChat, state.History).Get(ctx, &assistant)
				if err != nil {
					return "", err
				}
				state.History = append(state.History, assistant)

				if len(assistant.ToolCalls) == 0 {
					return assistant.Content, nil
				}
				if round >= maxToolRounds {
					return "", temporal.NewNonRetryableApplicationError(
						"model exceeded max tool rounds", "TooManyToolRounds", nil)
				}

				for _, tc := range assistant.ToolCalls {
					var result string
					switch tc.Function.Name {
					case ToolGetCurrentTime:
						if err := workflow.ExecuteActivity(actCtx, a.GetCurrentTime).Get(ctx, &result); err != nil {
							return "", err
						}
					case ToolListDirectory:
						path, _ := tc.Function.Arguments["path"].(string)
						if err := workflow.ExecuteActivity(actCtx, a.ListDirectory, path).Get(ctx, &result); err != nil {
							return "", err
						}
					default:
						result = "error: unknown tool " + tc.Function.Name
					}
					state.History = append(state.History, Message{
						Role:     "tool",
						ToolName: tc.Function.Name,
						Content:  result,
					})
				}
			}
		},
	); err != nil {
		return err
	}

	if err := workflow.SetQueryHandler(ctx, QueryHistory, func() ([]Message, error) {
		return state.History, nil
	}); err != nil {
		return err
	}

	endCh := workflow.GetSignalChannel(ctx, SignalEndChat)
	workflow.Go(ctx, func(ctx workflow.Context) {
		endCh.Receive(ctx, nil)
		ended = true
	})

	if err := workflow.Await(ctx, func() bool {
		return ended || workflow.GetInfo(ctx).GetContinueAsNewSuggested()
	}); err != nil {
		return err
	}

	// Don't continue-as-new or exit while an update is mid-flight.
	if err := workflow.Await(ctx, func() bool { return workflow.AllHandlersFinished(ctx) }); err != nil {
		return err
	}

	if ended {
		return nil
	}
	return workflow.NewContinueAsNewError(ctx, Workflow, state)
}
