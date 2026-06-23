package commands

import (
	"fmt"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/prompts"
)

// Review implements the code review workflow (security, quality, best practices).
type Review struct{}

func (r *Review) Name() string { return "review" }

func (r *Review) Run(cli *client.Client, diff string) (string, error) {
	prompt := fmt.Sprintf("Review this git diff for security vulnerabilities, code quality, and best practices:\n\n%s", diff)
	return cli.Generate(prompt, prompts.ReviewSystem())
}
