// Package diffprep filters, summarizes, and prepares git diffs before sending
// them to the LLM API. It handles:
//
//   - Filtering out noise files (lock files, auto-generated, binary)
//   - Computing per-file stats for context awareness
//   - Truncating extremely large diffs while preserving structure
//   - Preparing chunked output for hierarchical summarization
package diffprep

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// NoisePatterns are file patterns that should be excluded from diff analysis.
var NoisePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\.lock$`),                  // package-lock.json, Gemfile.lock
	regexp.MustCompile(`(?i)^node_modules/`),           // node_modules
	regexp.MustCompile(`(?i)\.min\.(js|css)$`),         // minified files
	regexp.MustCompile(`(?i)^vendor/`),                 // Go vendor dir
	regexp.MustCompile(`(?i)\.pb\.go$`),                // protobuf generated
	regexp.MustCompile(`(?i)\.generated\.`),             // auto-generated files
	regexp.MustCompile(`(?i)\.svg$`),                   // SVGs (diff is noise)
	regexp.MustCompile(`(?i)\.png$|\.jpg$|\.jpeg$`),    // images (binary)
	regexp.MustCompile(`(?i)\.git/`),                   // git internals
	regexp.MustCompile(`(?i)go\.sum$`),                 // Go dependency checksums
	regexp.MustCompile(`(?i)yarn\.lock$`),              // yarn lockfile
	regexp.MustCompile(`(?i)poetry\.lock$`),            // poetry lockfile
	regexp.MustCompile(`(?i)Pipfile\.lock$`),           // pip lockfile
	regexp.MustCompile(`(?i)\.bundle/`),                // bundler cache
	regexp.MustCompile(`(?i)dist/`),                    // build output
	regexp.MustCompile(`(?i)build/`),                   // build output
	regexp.MustCompile(`(?i)\.next/`),                  // next.js build cache
	regexp.MustCompile(`(?i)\.d\.ts$`),                 // TypeScript declaration files
	regexp.MustCompile(`(?i)\.map$`),                   // source maps
}

// FileStats tracks metadata about a single file in the diff.
type FileStats struct {
	Filename   string
	IsNewFile   bool
	IsDeleted   bool
	IsRenamed   bool
	Additions   int
	Deletions   int
	IsBinary    bool
	IsNoise     bool
	RawDiff     string
}

// PreparedDiff is the result of preprocessing a raw git diff.
type PreparedDiff struct {
	Files       []FileStats
	TotalFiles  int
	TotalAdded  int
	TotalDeleted int
	TotalNoiseFiltered int
	TotalBinarySkipped int
	Summary     string // human-readable summary of the diff
	Content     string // the actual diff content to send to the model
}

// Process takes a raw git diff string and returns a prepared, filtered diff.
func Process(rawDiff string) *PreparedDiff {
	files := parseDiff(rawDiff)

	result := &PreparedDiff{}
	var contentBuf bytes.Buffer

	for i := range files {
		f := &files[i]

		// Detect binary files
		if f.IsBinary = isBinary(f.RawDiff); f.IsBinary {
			result.TotalBinarySkipped++
			continue
		}

		// Detect noise files
		f.IsNoise = isNoiseFile(f.Filename)
		if f.IsNoise {
			result.TotalNoiseFiltered++
			continue
		}

		result.Files = append(result.Files, *f)
		result.TotalFiles++
		result.TotalAdded += f.Additions
		result.TotalDeleted += f.Deletions

		// Truncate individual file diffs if they're excessively large
		truncated := truncateFileDiff(f.RawDiff, f.Filename, 2000)
		contentBuf.WriteString(truncated)
		contentBuf.WriteString("\n\n")
	}

	// Build a summary header for model context
	result.Summary = buildSummary(result)
	result.Content = result.Summary + "\n" + contentBuf.String()

	return result
}

// parseDiff splits a unified git diff into per-file chunks.
func parseDiff(rawDiff string) []FileStats {
	var files []FileStats

	// Split on "diff --git" boundaries — use SubmatchIndex to get both
	// the overall match positions and the two captured groups (a/path and b/path).
	filePattern := regexp.MustCompile(`(?m)^diff --git a/(.*) b/(.*)$`)
	matches := filePattern.FindAllStringSubmatchIndex(rawDiff, -1)

	if len(matches) == 0 {
		return files
	}

	for i, m := range matches {
		// m[0:1] = overall match, m[2:3] = first capture (a/...), m[4:5] = second capture (b/...)
		start := m[0]
		end := len(rawDiff)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		chunk := rawDiff[start:end]

		var aPath, bPath string
		if m[2] >= 0 && m[3] > m[2] {
			aPath = rawDiff[m[2]:m[3]]
		}
		if m[4] >= 0 && m[5] > m[4] {
			bPath = rawDiff[m[4]:m[5]]
		}

		fs := parseFileChunk(chunk, aPath, bPath)
		files = append(files, fs)
	}

	return files
}

// parseFileChunk extracts stats from a single file's diff chunk.
func parseFileChunk(chunk, aPath, bPath string) FileStats {
	fs := FileStats{
		Filename:  normalizeFilename(aPath, bPath),
		RawDiff:   chunk,
	}

	// Detect file status from git headers
	if strings.Contains(chunk, "new file mode") || strings.Contains(chunk, "index 0000000..") {
		fs.IsNewFile = true
	}
	if strings.Contains(chunk, "deleted file mode") {
		fs.IsDeleted = true
	}
	if strings.Contains(chunk, "similarity index") {
		fs.IsRenamed = true
	}

	// Count additions and deletions
	scanDiff(chunk, &fs)

	return fs
}

// scanDiff counts + and - lines in a diff chunk.
func scanDiff(chunk string, fs *FileStats) {
	scanner := bufio.NewScanner(strings.NewReader(chunk))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && line[0] == '+' && !strings.HasPrefix(line, "+++") {
			fs.Additions++
		} else if len(line) > 0 && line[0] == '-' && !strings.HasPrefix(line, "---") {
			fs.Deletions++
		}
	}
}

// isBinary checks if the diff chunk represents a binary file.
func isBinary(rawDiff string) bool {
	return strings.Contains(rawDiff, "Binary files ") ||
		strings.Contains(rawDiff, "Binary file")
}

// isNoiseFile checks if the filename matches any noise pattern.
func isNoiseFile(filename string) bool {
	for _, pattern := range NoisePatterns {
		if pattern.MatchString(filename) {
			return true
		}
	}
	return false
}

// normalizeFilename returns the most useful filename from git diff paths.
func normalizeFilename(aPath, bPath string) string {
	// If renamed, use the new name
	if aPath != bPath && aPath != "dev/null" && bPath != "dev/null" {
		return bPath
	}
	if aPath == "dev/null" {
		return bPath // new file
	}
	if bPath == "dev/null" {
		return aPath // deleted file
	}
	return aPath
}

// truncateFileDiff truncates a single file's diff to maxLines lines,
// preserving the header and showing a summary of skipped content.
func truncateFileDiff(rawDiff string, filename string, maxLines int) string {
	lines := strings.Split(rawDiff, "\n")
	if len(lines) <= maxLines {
		return rawDiff
	}

	// Keep first 50 lines (file headers + context), truncate middle, keep last 50
	headerLines := 50
	footerLines := 50
	skipped := len(lines) - headerLines - footerLines

	var buf bytes.Buffer
	buf.WriteString(strings.Join(lines[:headerLines], "\n"))
	buf.WriteString(fmt.Sprintf("\n\\ No newline at end of file\n\\ ... %d lines of diff truncated ...\n\\ %s\n", skipped, filename))
	buf.WriteString(strings.Join(lines[len(lines)-footerLines:], "\n"))

	return buf.String()
}

// buildSummary creates a concise summary of the diff for model context.
func buildSummary(pd *PreparedDiff) string {
	if len(pd.Files) == 0 {
		return ""
	}

	// Sort files by change density (most changes first)
	sorted := make([]FileStats, len(pd.Files))
	copy(sorted, pd.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Additions+sorted[i].Deletions > sorted[j].Additions+sorted[j].Deletions
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Diff Summary ===\n"))
	sb.WriteString(fmt.Sprintf("Files changed: %d\n", pd.TotalFiles))
	sb.WriteString(fmt.Sprintf("Total insertions: %d\n", pd.TotalAdded))
	sb.WriteString(fmt.Sprintf("Total deletions: %d\n", pd.TotalDeleted))
	if pd.TotalNoiseFiltered > 0 {
		sb.WriteString(fmt.Sprintf("(Filtered out %d noise files, %d binary files)\n", pd.TotalNoiseFiltered, pd.TotalBinarySkipped))
	}
	sb.WriteString("\nFiles:\n")

	for _, f := range sorted {
		status := "modified"
		if f.IsNewFile {
			status = "new"
		} else if f.IsDeleted {
			status = "deleted"
		} else if f.IsRenamed {
			status = "renamed"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s (+%d -%d)\n", status, f.Filename, f.Additions, f.Deletions))
	}

	return sb.String()
}

// ShouldChunk returns true if the diff is large enough to benefit from
// chunked/hierarchical summarization instead of a single API call.
func ShouldChunk(rawDiff string) bool {
	lines := strings.Count(rawDiff, "\n")
	return lines > 500 // More than ~500 lines = worth chunking
}

// ChunkDiff splits a raw diff into manageable chunks for parallel processing.
// Each chunk contains one or more complete file diffs.
func ChunkDiff(rawDiff string, maxChunkLines int) [][]FileStats {
	files := parseDiff(rawDiff)
	if maxChunkLines <= 0 {
		maxChunkLines = 300
	}

	var chunks [][]FileStats
	var currentChunk []FileStats
	currentLines := 0

	for _, f := range files {
		fileLines := strings.Count(f.RawDiff, "\n")

		// If adding this file would exceed the chunk limit and we already have files,
		// flush the current chunk
		if currentLines+fileLines > maxChunkLines && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = nil
			currentLines = 0
		}

		currentChunk = append(currentChunk, f)
		currentLines += fileLines
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// ChunkToDiff converts a group of FileStats back into a diff string.
func ChunkToDiff(files []FileStats) string {
	var buf bytes.Buffer
	for _, f := range files {
		buf.WriteString(f.RawDiff)
		buf.WriteString("\n\n")
	}
	return buf.String()
}