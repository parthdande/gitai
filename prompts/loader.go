// Package prompts provides default system prompts for each gitai task and a
// loader for custom user prompts. Users can place .md files in
// ~/.gitai/system_prompts/ to override the built-in defaults; the loader
// falls back to the built-in prompt if the file is missing.
package prompts

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadSystemPrompt loads a custom system prompt from disk.
//
// It looks for a file at: <configDir>/system_prompts/<taskName>.md
//
// If the file exists and is readable, its contents are returned.
// If the file is missing, the built-in default for that task is returned:
//
//   - "commit" → DefaultCommitSystem()
//   - "review" → DefaultReviewSystem()
//   - anything else → empty string
//
// This gives users hot-reload: edit the .md file, next run picks it up.
func LoadSystemPrompt(taskName, configDir string) string {
	promptPath := filepath.Join(configDir, "system_prompts", taskName+".md")

	data, err := os.ReadFile(promptPath)
	if err == nil {
		// File found and read successfully — use it.
		return string(data)
	}

	// File not found or unreadable — fall back to built-in defaults.
	switch taskName {
	case "commit":
		return DefaultCommitSystem()
	case "review":
		return DefaultReviewSystem()
	default:
		// Unknown task — return empty, the caller can handle it.
		fmt.Fprintf(os.Stderr, "WARNING: no system prompt found for task '%s', using default\n", taskName)
		return ""
	}
}
