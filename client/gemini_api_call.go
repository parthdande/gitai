package client

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

const defaultModel = "gemini-3.1-flash-lite"

// GeminiAPI sends inputText to the Gemini model and returns the generated response.
func (g *Gemini) GeminiAPI(inputText, systemPrompt string) (string, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: g.APIKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create GenAI client: %w", err)
	}

	model := g.Model
	if model == "" {
		model = defaultModel
	}

	var config *genai.GenerateContentConfig
	if systemPrompt != "" {
		config = &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: systemPrompt},
				},
			},
		}
	}

	result, err := client.Models.GenerateContent(ctx, model, genai.Text(inputText), config)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return result.Text(), nil
}
