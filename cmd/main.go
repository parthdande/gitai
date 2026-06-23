package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/commands"
	"github.com/spf13/viper"
)

const (
	maxCommitMsgLen = 7200 // Git's soft limit for commit message size
	gitTimeout      = 30 * time.Second
)

func main() {
	// ── Flags ──────────────────────────────────────────────
	commitMsgFlag := flag.Bool("commitmsg", false, "Generate a commit message from git diff and print it")
	commitFlag := flag.Bool("commit", false, "Generate a commit message and automatically commit all changes")
	reviewFlag := flag.Bool("review", false, "Review git diff for security, quality, and best practices")
	updateFlag := flag.Bool("update", false, "Update gitai to the latest version")
	uninstallFlag := flag.Bool("uninstall", false, "Uninstall gitai from the system")

	flag.Usage = func() {
		fmt.Println("gitai — AI-assisted git commits, messages, and code reviews")
		fmt.Println("\nUsage:")
		fmt.Println("  gitai [flags]")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	// ── Special commands (no config needed) ────────────────
	if *uninstallFlag {
		uninstall()
		return
	}
	if *updateFlag {
		// TODO: implement self-update via GitHub releases or git pull + rebuild
		fmt.Println("Updating GitAI...")
		fmt.Println("Run: git pull && go build -o gitai cmd/main.go && sudo mv gitai /usr/local/bin/")
		return
	}

	// ── Load config ────────────────────────────────────────
	v := viper.New() // isolated instance — no global side effects
	cli, err := loadClient(v)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	// ── Pick handler based on flag ─────────────────────────
	var handler commands.Handler

	switch {
	case *commitMsgFlag, *commitFlag:
		handler = &commands.Commit{}
	case *reviewFlag:
		handler = &commands.Review{}
	default:
		flag.Usage()
		return
	}

	// ── Run the handler ────────────────────────────────────
	result, err := runHandler(cli, handler)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// ── Output ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("───────────────────────────────────────")
	fmt.Println(result)
	fmt.Println("───────────────────────────────────────")

	// Auto-commit if -commit was used
	if *commitFlag {
		sanitized := sanitizeForGit(result)
		fmt.Println("\nCommitting changes...")
		ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "commit", "-a", "-m", sanitized)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to commit: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully committed!")
	}
}

// sanitizeForGit strips control characters and truncates AI output for safe use in git commit messages.
func sanitizeForGit(s string) string {
	// Remove control chars except newline and carriage return
	var b strings.Builder
	for _, r := range s {
		if (r >= 0x20 && r <= 0x7E) || r == '\n' || r == '\r' {
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	if len(result) > maxCommitMsgLen {
		result = result[:maxCommitMsgLen]
	}
	return result
}

// loadClient reads config from ~/.gitai/gitai.json and environment variables.
func loadClient(v *viper.Viper) (*client.Client, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("could not get current user: %w", err)
	}

	configDir := filepath.Join(currentUser.HomeDir, ".gitai")
	configFile := filepath.Join(configDir, "gitai.json")

	v.SetConfigName("gitai")
	v.SetConfigType("json")
	v.AddConfigPath(configDir)
	_ = v.ReadInConfig() // config file is optional

	// API Base
	apiBase := v.GetString("api_base")
	if envBase := os.Getenv("GEMINI_API_BASE"); envBase != "" {
		apiBase = envBase
	}
	if apiBase == "" {
		apiBase = os.Getenv("API_BASE")
	}

	// API Key
	apiKey := v.GetString("api_key")
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		apiKey = envKey
	}
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}

	// Model
	model := v.GetString("model")
	if envModel := os.Getenv("MODEL"); envModel != "" {
		model = envModel
	}

	if apiBase == "" {
		return nil, fmt.Errorf("no API base URL found. Set the API_BASE environment variable, or add api_base to %s", configFile)
	}
	if model == "" {
		return nil, fmt.Errorf("no model found. Set the MODEL environment variable, or add model to %s", configFile)
	}

	return &client.Client{
		APIBase: apiBase,
		APIKey:  apiKey,
		Model:   model,
	}, nil
}

// runHandler fetches the git diff and runs the selected handler.
func runHandler(cli *client.Client, h commands.Handler) (string, error) {
	fmt.Printf("Running '%s'...\n", h.Name())

	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "HEAD")
	diffBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not fetch git diff: %w", err)
	}

	diff := string(diffBytes)
	if diff == "" {
		return "", fmt.Errorf("no changes detected — working tree is clean")
	}

	return h.Run(cli, diff)
}

func uninstall() {
	// Resolve the actual binary path dynamically instead of hardcoding /usr/local/bin
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Could not determine binary path: %v\n", err)
		os.Exit(1)
	}

	// Check if running as root
	if os.Geteuid() != 0 {
		fmt.Printf("Cannot uninstall: not running as root.\nRun: sudo %s -uninstall\n", execPath)
		os.Exit(1)
	}

	fmt.Println("Uninstalling GitAI...")
	if err := os.Remove(execPath); err != nil {
		fmt.Printf("Uninstall failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("GitAI uninstalled successfully!")
}
