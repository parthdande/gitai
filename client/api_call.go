package client

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

// defaultHTTPTimeout is the maximum time an API call is allowed to take.
const defaultHTTPTimeout = 120 * time.Second

// chatCompletionRequest mirrors the OpenAI chat completions request body.
type chatCompletionRequest struct {
	Model              string        `json:"model"`
	Messages           []chatMessage `json:"messages"`
	ChatTemplateKwargs *struct {
		EnableThinking bool `json:"enable_thinking"`
	} `json:"chat_template_kwargs,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse mirrors the relevant parts of the API response.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// httpClient returns the HTTP client to use for API calls.
// Reuses the embedded client if set, otherwise creates one with a default timeout.
func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	// Lazy-init a client with a 120s timeout so we never hang forever.
	c.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	return c.HTTPClient
}

// Generate sends the prompt to the configured OpenAI-compatible endpoint.
//
//   - ctx:        context for cancellation (e.g. Ctrl+C) and timeouts
//   - model:      which model to use for this call (overrides client.Model if different)
//   - thinking:   whether to enable extended thinking mode
//
// If thinking is true and the API returns an error, Generate automatically
// retries one time with thinking disabled and returns that result instead.
// This way the user never sees a hard failure just because the model doesn't
// support thinking mode.
func (c *Client) Generate(ctx context.Context, prompt, systemPrompt, model string, thinking bool) (string, error) {
	if model == "" {
		model = c.Model // fallback to client-level model
	}
	if model == "" {
		return "", fmt.Errorf("model is required (set model in config or MODEL env var)")
	}
	if c.APIBase == "" {
		return "", fmt.Errorf("api_base is required (set api_base in config or API_BASE env var)")
	}

	// Try with the requested thinking setting.
	result, err := c.doGenerate(ctx, prompt, systemPrompt, model, thinking)

	// If thinking was ON and it failed, retry silently with thinking OFF.
	if err != nil && thinking {
		fmt.Fprintf(os.Stderr, "Thinking mode failed (%v), falling back to non-thinking...\n", err)
		return c.doGenerate(ctx, prompt, systemPrompt, model, false)
	}

	return result, err
}

// doGenerate performs a single API call (no fallback logic).
// This is the internal workhorse — Generate() calls this.
func (c *Client) doGenerate(ctx context.Context, prompt, systemPrompt, model string, thinking bool) (string, error) {
	// Build the full URL: ensure base ends with /v1/
	baseURL := c.APIBase
	if baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	if baseURL[len(baseURL)-4:] != "/v1/" && baseURL[len(baseURL)-4:] != "/v1" {
		baseURL += "v1/"
	}
	url := baseURL + "chat/completions"

	// Build the message list (system prompt first, then user prompt).
	messages := make([]chatMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: prompt})

	// Only include chat_template_kwargs when thinking is enabled.
	var templateKwargs *struct {
		EnableThinking bool `json:"enable_thinking"`
	}
	if thinking {
		templateKwargs = &struct {
			EnableThinking bool `json:"enable_thinking"`
		}{EnableThinking: true}
	}

	// Serialize the request body.
	body, err := json.Marshal(chatCompletionRequest{
		Model:              model,
		Messages:           messages,
		ChatTemplateKwargs: templateKwargs,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request with context support (Ctrl+C will cancel).
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	// Reuse the HTTP client for connection pooling (not a new one each call).
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses.
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Decode and return the model's response.
	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("API returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}
