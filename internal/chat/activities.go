package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Activities holds dependencies for activity implementations.
type Activities struct {
	// BaseURL of the Ollama server; defaults to http://localhost:11434.
	BaseURL string
	// Model name to run; defaults to $OLLAMA_MODEL or "qwen3:14b".
	Model string
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Think    bool      `json:"think"`
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
	{
		Type: "function",
		Function: toolFunction{
			Name:        ToolListDirectory,
			Description: "List the files and subdirectories in a directory.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "The directory to list. Defaults to the current working directory.",
					},
				},
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
		model = "qwen3:14b"
	}

	// Thinking is off by default: qwen3's reasoning phase multiplies latency
	// several times over for chat-sized questions. Non-thinking models ignore
	// the field. Set OLLAMA_THINK=true to trade speed for more careful answers.
	body, err := json.Marshal(ollamaChatRequest{
		Model:    model,
		Messages: history,
		Think:    os.Getenv("OLLAMA_THINK") == "true",
		Tools:    tools,
	})
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

// maxDirEntries caps list_directory output so a huge directory can't blow up
// the model's context (or the workflow's event history).
const maxDirEntries = 200

// ListDirectory is the list_directory tool. Expected problems (missing or
// unreadable directory) are returned as the tool result rather than an error,
// so the model can see what went wrong and recover instead of the activity
// retrying a call that will never succeed.
func (a *Activities) ListDirectory(ctx context.Context, path string) (string, error) {
	if path == "" {
		path = "."
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "error: " + err.Error(), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "contents of %s:\n", path)
	for i, entry := range entries {
		if i == maxDirEntries {
			fmt.Fprintf(&b, "... and %d more entries\n", len(entries)-maxDirEntries)
			break
		}
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		b.WriteString(name + "\n")
	}
	if len(entries) == 0 {
		b.WriteString("(empty)\n")
	}
	return b.String(), nil
}
