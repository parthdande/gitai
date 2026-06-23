package commands

import (
	"fmt"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/prompts"
)

// Commit implements the commit message generation workflow.
type Commit struct{}

func (c *Commit) Name() string { return "commit" }

func (c *Commit) Run(cli *client.Client, diff string) (string, error) {
	prompt := fmt.Sprintf("Analyze this git diff and write a commit message:\n\n%s", diff)
	return cli.Generate(prompt, prompts.CommitSystem())
}
