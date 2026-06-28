package commands

import (
	"context"
	"fmt"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/prompts"
)

// PullReq implements the PR description generation workflow.
type PullReq struct{}

func (p *PullReq) Name() string { return "pullreq" }

func (p *PullReq) Run(ctx context.Context, cli *client.Client, diff string, model string, thinking bool, configDir string) (string, error) {
	systemPrompt := prompts.LoadSystemPrompt("pullreq", configDir)

	prompt := fmt.Sprintf("Generate a PR description for these changes:\n\n%s", diff)
	return cli.Generate(ctx, prompt, systemPrompt, model, thinking)
}
