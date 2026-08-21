// Package chat contains the Temporal workflow and activities for a chat
// session backed by a locally running LLM (Ollama).
package chat

const (
	// TaskQueue is the Temporal task queue both the worker and clients use.
	TaskQueue = "local-agent-chat"

	// UpdateSendMessage is a workflow update: send a user message, get the
	// assistant's reply back as the update result.
	UpdateSendMessage = "send-message"

	// QueryHistory returns the full conversation history.
	QueryHistory = "history"

	// SignalEndChat asks the workflow to finish.
	SignalEndChat = "end-chat"

	// ToolGetCurrentTime is the tool name the model uses to ask for the
	// current date and time.
	ToolGetCurrentTime = "get_current_time"
)

// Message is one turn in the conversation, matching Ollama's chat format.
type Message struct {
	Role      string     `json:"role"` // "system", "user", "assistant", or "tool"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // set on assistant messages requesting tools
	ToolName  string     `json:"tool_name,omitempty"`  // set on "tool" (result) messages
}

// ToolCall is a tool invocation requested by the model.
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction names the tool and carries its JSON arguments.
type ToolCallFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// State is the workflow's durable state. It is passed to continue-as-new so a
// long conversation can span multiple workflow runs.
type State struct {
	History []Message `json:"history"`
}
