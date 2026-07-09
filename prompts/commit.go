package prompts

// DefaultCommitSystem returns the built-in default system prompt for generating
// conventional commit messages. Used as a fallback when no custom prompt file exists.
func DefaultCommitSystem() string {
	return "You are an expert software engineer. Generate a structured conventional commit message based on the provided git diff. It must start with a short header line (conventional commit style) summarizing the overall change, followed by a blank line, then a brief paragraph describing the purpose of the changes, focusing on the impact and user-facing behavior, followed by another blank line, and a bulleted list detailing what the changes accomplished (focusing on logical and functional changes rather than listing file names). Do not include markdown formatting (like ```), just return the raw text also do not use ` or \"`\" marks in commit message ."
}

// DefaultSummarizeChunkPrompt returns the system prompt used during the first
// stage of hierarchical summarization (chunk-level). Each chunk is a portion
// of the full diff, and the model produces a concise summary of it.
func DefaultSummarizeChunkPrompt() string {
	return "You are an expert software engineer. Given the following git diff chunk, write a concise 1-2 sentence summary of what this chunk of changes does. Focus on the functional impact, not file names. Keep it short and specific."
}

// CommitSystem exists for backward compatibility — aliases DefaultCommitSystem.
func CommitSystem() string {
	return DefaultCommitSystem()
}
