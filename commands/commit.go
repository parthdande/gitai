package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/diffprep"
	"github.com/parthdande/gitai/prompts"
)

// Commit implements the commit message generation workflow.
//
// For small diffs (<500 lines), the entire diff is sent in a single call.
// For larger diffs, a two-stage hierarchical summarization pipeline is used:
//   - Stage 1: Each chunk is summarized independently (1-2 sentences per chunk)
//   - Stage 2: All summaries are synthesized into the final commit message
type Commit struct{}

func (c *Commit) Name() string { return "commit" }

func (c *Commit) Run(ctx context.Context, cli *client.Client, diff string, model string, thinking bool, configDir string) (string, error) {
	// Load the system prompt from ~/.gitai/system_prompts/commit.md (or use default).
	systemPrompt := prompts.LoadSystemPrompt("commit", configDir)

	// Preprocess the diff — filter noise files, compute stats, prepare content.
	prepared := diffprep.Process(diff)

	// Decide: single call or hierarchical chunking?
	if diffprep.ShouldChunk(diff) {
		return c.runHierarchical(ctx, cli, diff, prepared, model, thinking, systemPrompt)
	}

	// Small diff — send everything in one shot.
	prompt := fmt.Sprintf("Analyze this git diff and write a commit message:\n\n%s", prepared.Content)
	return cli.Generate(ctx, prompt, systemPrompt, model, thinking)
}

// runHierarchical uses a two-stage pipeline for large diffs:
// Stage 1: Summarize each chunk independently
// Stage 2: Synthesize all summaries into the final commit message
func (c *Commit) runHierarchical(ctx context.Context, cli *client.Client, rawDiff string, prepared *diffprep.PreparedDiff, model string, thinking bool, systemPrompt string) (string, error) {
	chunks := diffprep.ChunkDiff(rawDiff, 300)

	stage1Prompt := prompts.DefaultSummarizeChunkPrompt()

	var summaries []string
	for i, chunk := range chunks {
		chunkDiff := diffprep.ChunkToDiff(chunk)
		chunkFileNames := chunkFileNames(chunk)

		prompt := fmt.Sprintf(
			"Chunk %d/%d (files: %s):\n\n%s",
			i+1, len(chunks), chunkFileNames, chunkDiff,
		)

		summary, err := cli.Generate(ctx, prompt, stage1Prompt, model, false)
		if err != nil {
			return "", fmt.Errorf("chunk summarization failed for chunk %d: %w", i+1, err)
		}
		summaries = append(summaries, strings.TrimSpace(summary))
	}

	// Stage 2: Synthesize all summaries into the final commit message.
	synthesisPrompt := fmt.Sprintf(
		"Here is a summary of all the changes in this commit:\n\n%s\n\nBased on these summaries, write a comprehensive conventional commit message.\n\nDiff metadata:\n%s",
		strings.Join(summaries, "\n\n"),
		prepared.Summary,
	)

	return cli.Generate(ctx, synthesisPrompt, systemPrompt, model, thinking)
}

func chunkFileNames(files []diffprep.FileStats) string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Filename
	}
	return strings.Join(names, ", ")
}
