package commands

import (
	"context"
	"fmt"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/prompts"
)

// Commit implements the commit message generation workflow.
type Commit struct{}

func (c *Commit) Name() string { return "commit" }

func (c *Commit) Run(ctx context.Context, cli *client.Client, diff string, model string, thinking bool, configDir string) (string, error) {
	// Load the system prompt from ~/.gitai/system_prompts/commit.md (or use default).
	systemPrompt := prompts.LoadSystemPrompt("commit", configDir)

	prompt := fmt.Sprintf("Analyze this git diff and write a commit message:\n\n%s", diff)
	return cli.Generate(ctx, prompt, systemPrompt, model, thinking)
}
