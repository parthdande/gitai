package commands

import "github.com/parthdande/gitai/client"

// Handler defines the contract for all gitai features.
// Each feature (commit, review, etc.) implements this interface.
type Handler interface {
	// Name returns the command name (e.g. "commit", "review").
	Name() string

	// Run executes the feature's workflow using the provided client and git diff.
	// It returns the AI's response text and any error.
	Run(cli *client.Client, diff string) (string, error)
}
