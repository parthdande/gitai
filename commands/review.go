package commands

import (
	"context"
	"fmt"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/prompts"
)

// Review implements the code review workflow (security, quality, best practices).
type Review struct{}

func (r *Review) Name() string { return "review" }

func (r *Review) Run(ctx context.Context, cli *client.Client, diff string, model string, thinking bool, configDir string) (string, error) {
	// Load the system prompt from ~/.gitai/system_prompts/review.md (or use default).
	systemPrompt := prompts.LoadSystemPrompt("review", configDir)

	prompt := fmt.Sprintf("Review this git diff for security vulnerabilities, code quality, and best practices:\n\n%s", diff)
	return cli.Generate(ctx, prompt, systemPrompt, model, thinking)
}
