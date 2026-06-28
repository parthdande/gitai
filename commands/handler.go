// Package commands defines the Handler interface and its implementations for each
// gitai feature. Each handler receives a git diff, loads the appropriate system
// prompt, and calls the AI client to generate its output (commit message, review, etc.).
package commands

import (
	"context"

	"github.com/parthdande/gitai/client"
)

// Handler defines the contract for all gitai features.
// Each feature (commit, review, etc.) implements this interface.
type Handler interface {
	// Name returns the command name (e.g. "commit", "review").
	Name() string

	// Run executes the feature's workflow.
	//
	//   - ctx:       context for cancellation (Ctrl+C) and timeouts
	//   - cli:       the API client (holds api_base, api_key, global model fallback)
	//   - diff:      the staged git diff to analyze
	//   - model:     which model to use for this call (may override client.Model)
	//   - thinking:  whether to enable extended thinking mode
	//   - configDir: path to ~/.gitai (used to find system prompt files)
	//
	// Returns the AI's response text and any error.
	Run(ctx context.Context, cli *client.Client, diff string, model string, thinking bool, configDir string) (string, error)
}
