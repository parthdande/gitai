package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// defaultHTTPTimeout is the maximum time an API call is allowed to take.
const defaultHTTPTimeout = 120 * time.Second

// chatCompletionRequest mirrors the OpenAI chat completions request body.
type chatCompletionRequest struct {
	Model              string        `json:"model"`
	Messages           []chatMessage `json:"messages"`
	Stream             bool          `json:"stream"`
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

// streamChunk mirrors a single SSE chunk from a streaming response.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
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

	// Estimate prompt size to provide better diagnostics
	promptSize := len(prompt) + len(systemPrompt)

	// Try with the requested thinking setting.
	result, err := c.doGenerate(ctx, prompt, systemPrompt, model, thinking)

	// If thinking was ON and it failed, retry silently with thinking OFF.
	if err != nil && thinking {
		fmt.Fprintf(os.Stderr, "Thinking mode failed (%v), falling back to non-thinking...\n", err)
		return c.doGenerate(ctx, prompt, systemPrompt, model, false)
	}

	// If the error mentions context length, provide a helpful suggestion.
	if err != nil && isContextLengthError(err) {
		return "", fmt.Errorf(
			"context length exceeded (prompt ~%d bytes). "+
				"This diff is too large for a single API call. "+
				"gitai will automatically chunk it into smaller pieces on the next run.\n\n"+
				"To reduce diff size:\n"+
				"  - Commit smaller changes more frequently\n"+
				"  - Avoid including build artifacts (dist/, node_modules/)\n"+
				"  - Avoid committing lock files or auto-generated code",
			promptSize,
		)
	}

	// If timeout, suggest the issue might be a large diff.
	if err != nil && isTimeoutError(err) {
		return "", fmt.Errorf(
			"request timed out (prompt ~%d bytes). "+
				"The diff may be too large — try committing smaller batches of changes.",
			promptSize,
		)
	}

	return result, err
}

// doGenerate performs a single API call (no fallback logic).
// Attempts streaming first, falls back to non-streaming if streaming fails.
func (c *Client) doGenerate(ctx context.Context, prompt, systemPrompt, model string, thinking bool) (string, error) {
	baseURL := c.buildURL()
	url := baseURL + "chat/completions"

	messages := c.buildMessages(prompt, systemPrompt)
	templateKwargs := c.buildTemplateKwargs(thinking)

	body, err := json.Marshal(chatCompletionRequest{
		Model:              model,
		Messages:           messages,
		Stream:             true, // Try streaming first
		ChatTemplateKwargs: templateKwargs,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Try streaming response first
	result, streamErr := c.doStreamedRequest(ctx, url, body)
	if streamErr == nil {
		return result, nil
	}

	// Streaming failed — fall back to non-streaming (some APIs don't support it)
	fmt.Fprintf(os.Stderr, "Streaming not supported, falling back to non-streamed response...\n")

	bodyNonStream, _ := json.Marshal(chatCompletionRequest{
		Model:              model,
		Messages:           messages,
		Stream:             false,
		ChatTemplateKwargs: templateKwargs,
	})

	return c.doNonStreamedRequest(ctx, url, bodyNonStream)
}

// buildURL normalizes the API base URL.
func (c *Client) buildURL() string {
	baseURL := c.APIBase
	if baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	if baseURL[len(baseURL)-4:] != "/v1/" && baseURL[len(baseURL)-4:] != "/v1" {
		baseURL += "v1/"
	}
	return baseURL
}

// buildMessages constructs the message list from system + user prompts.
func (c *Client) buildMessages(prompt, systemPrompt string) []chatMessage {
	messages := make([]chatMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: prompt})
	return messages
}

// buildTemplateKwargs returns the thinking mode kwargs or nil.
func (c *Client) buildTemplateKwargs(thinking bool) *struct {
	EnableThinking bool `json:"enable_thinking"`
} {
	if thinking {
		return &struct {
			EnableThinking bool `json:"enable_thinking"`
		}{EnableThinking: true}
	}
	return nil
}

// doStreamedRequest sends the request and reads an SSE stream response.
func (c *Client) doStreamedRequest(ctx context.Context, url string, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("streaming request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Read SSE stream
	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	// Increase buffer size for large tokens (default is 64KB, we need more)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE lines start with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// End of stream marker
		if data == "[DONE]" {
			break
		}

		// Parse the JSON chunk
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // Skip malformed chunks
		}

		// Extract and accumulate content
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				result.WriteString(choice.Delta.Content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	return result.String(), nil
}

// doNonStreamedRequest sends a standard non-streaming request.
func (c *Client) doNonStreamedRequest(ctx context.Context, url string, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses with better diagnostics.
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		bodyStr := string(respBody)

		// Try to extract a more specific error from common API formats
		switch resp.StatusCode {
		case http.StatusRequestEntityTooLarge:
			return "", fmt.Errorf("request body too large (%d bytes). Your diff exceeds the API's payload limit. Try committing smaller batches", len(body))
		case http.StatusRequestTimeout:
			return "", fmt.Errorf("request timeout from server. The diff may be too large for the configured timeout")
		case http.StatusTooManyRequests:
			return "", fmt.Errorf("rate limited by the API. Please wait and try again")
		default:
			return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, bodyStr)
		}
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

// isContextLengthError checks if the error is related to exceeding context limits.
func isContextLengthError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "context") && strings.Contains(msg, "length") ||
		strings.Contains(msg, "model's max context length") ||
		strings.Contains(msg, "exceeds") && strings.Contains(msg, "context") ||
		strings.Contains(msg, "max_tokens") ||
		strings.Contains(msg, "too long")
}

// isTimeoutError checks if the error is a timeout.
func isTimeoutError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "i/o timeout")
}