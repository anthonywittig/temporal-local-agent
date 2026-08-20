package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
}

type ollamaChatResponse struct {
	Message Message `json:"message"`
	Error   string  `json:"error"`
}

// CompleteChat sends the conversation to Ollama and returns the assistant's
// next message.
func (a *Activities) CompleteChat(ctx context.Context, history []Message) (string, error) {
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

	body, err := json.Marshal(ollamaChatRequest{Model: model, Messages: history})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling ollama: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, respBody)
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("ollama error: %s", parsed.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, respBody)
	}
	return parsed.Message.Content, nil
}
