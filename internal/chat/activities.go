package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Activities holds dependencies for activity implementations.
type Activities struct {
	// BaseURL of the Ollama server; defaults to http://localhost:11434.
	BaseURL string
	// Model name to run; defaults to $OLLAMA_MODEL or "llama3.2".
	Model string
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []tool    `json:"tools,omitempty"`
}

type tool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// tools advertised to the model on every request. Each one the model calls is
// executed by the workflow as its own activity.
var tools = []tool{
	{
		Type: "function",
		Function: toolFunction{
			Name:        ToolGetCurrentTime,
			Description: "Get the current date and time, including the timezone.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	},
}

type ollamaChatResponse struct {
	Message Message `json:"message"`
	Error   string  `json:"error"`
}

// CompleteChat sends the conversation to Ollama and returns the assistant's
// next message, which either answers the user or requests tool calls.
func (a *Activities) CompleteChat(ctx context.Context, history []Message) (Message, error) {
	baseURL := a.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := a.Model
	if model == "" {
		model = os.Getenv("OLLAMA_MODEL")
	}
	if model == "" {
		model = "llama3.2"
	}

	body, err := json.Marshal(ollamaChatRequest{Model: model, Messages: history, Tools: tools})
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("calling ollama: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, err
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Message{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, respBody)
	}
	if parsed.Error != "" {
		return Message{}, fmt.Errorf("ollama error: %s", parsed.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, respBody)
	}
	return parsed.Message, nil
}

// GetCurrentTime is the get_current_time tool. It runs as an activity: the
// workflow itself must be deterministic, so "what time is it" has to be
// answered out here, where the result gets recorded in the event history.
func (a *Activities) GetCurrentTime(ctx context.Context) (string, error) {
	return time.Now().Format("Monday, January 2, 2006, 3:04 PM MST"), nil
}
