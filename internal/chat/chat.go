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
)

// Message is one turn in the conversation, matching Ollama's chat format.
type Message struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

// State is the workflow's durable state. It is passed to continue-as-new so a
// long conversation can span multiple workflow runs.
type State struct {
	History []Message `json:"history"`
}
