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
			Content: "You are a helpful assistant. Keep replies concise.",
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
			var reply string
			err := workflow.ExecuteActivity(
				workflow.WithActivityOptions(ctx, ao),
				(&Activities{}).CompleteChat, state.History,
			).Get(ctx, &reply)
			if err != nil {
				return "", err
			}

			state.History = append(state.History, Message{Role: "assistant", Content: reply})
			return reply, nil
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
