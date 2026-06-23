package prompts

// CommitSystem returns the system prompt for generating conventional commit messages.
func CommitSystem() string {
	return "You are an expert software engineer. Generate a structured conventional commit message based on the provided git diff. It must start with a short header line (conventional commit style) summarizing the overall change, followed by a blank line, then a brief paragraph describing the purpose of the changes, focusing on the impact and user-facing behavior, followed by another blank line, and a bulleted list detailing what the changes accomplished (focusing on logical and functional changes rather than listing file names). Do not include markdown formatting (like ```), just return the raw text."
}
