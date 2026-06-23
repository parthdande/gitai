package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// chatCompletionRequest mirrors the OpenAI chat completions request body.
type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse mirrors the relevant parts of the OpenAI response.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Generate sends the prompt to the configured OpenAI-compatible endpoint
// and returns the model's text response.
func (c *Client) Generate(prompt, systemPrompt string) (string, error) {
	if c.Model == "" {
		return "", fmt.Errorf("model is required (set model in config or MODEL env var)")
	}
	if c.APIBase == "" {
		return "", fmt.Errorf("api_base is required (set api_base in config or API_BASE env var)")
	}

	// Build the full URL: ensure base ends with /v1/
	baseURL := c.APIBase
	if baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	// If base doesn't already include /v1, append it.
	if baseURL[len(baseURL)-4:] != "/v1/" && baseURL[len(baseURL)-4:] != "/v1" {
		baseURL += "v1/"
	}
	url := baseURL + "chat/completions"

	messages := make([]chatMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: prompt})

	body, err := json.Marshal(chatCompletionRequest{
		Model:    c.Model,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("API returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}
