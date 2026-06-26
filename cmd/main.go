package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
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
	maxCommitMsgLen = 7200     // Git's soft limit for commit message size
	gitTimeout      = 30 * time.Second
)

func main() {
	// --- Read CLI flags ---
	commitMsgFlag := flag.Bool("commitmsg", false, "Generate a commit message from git diff and print it")
	commitFlag := flag.Bool("commit", false, "Generate a commit message and automatically commit all changes")
	reviewFlag := flag.Bool("review", false, "Review git diff for security, quality, and best practices")
	updateFlag := flag.Bool("update", false, "Update gitai to the latest version")
	uninstallFlag := flag.Bool("uninstall", false, "Uninstall gitai from the system")
	thinkFlag := flag.Bool("think", false, "Enable extended thinking mode (overrides config)")

	flag.Usage = func() {
		fmt.Println(`gitai - AI-assisted git commits, messages, and code reviews

Usage:
  gitai [flags]

Flags:`)
		flag.PrintDefaults()
		fmt.Println(`
Config (~/.gitai/gitai.json):
  {
    "api_base": "https://...",
    "api_key": "sk-...",
    "commit": { "model": "...", "thinking": true },
    "review": { "model": "...", "thinking": false }
  }

System prompts: ~/.gitai/system_prompts/<command>.md
  (e.g. commit.md, review.md) - edit for hot-reload`)
	}
	flag.Parse()

	// --- Special commands (no config needed) ---
	if *uninstallFlag {
		uninstall()
		return
	}
	if *updateFlag {
		doUpdate()
		return
	}

	// --- Load config from ~/.gitai/gitai.json ---
	v := viper.New()
	configDir, cli, err := loadClient(v)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	// --- Figure out which task to run (commit or review) ---
	taskName := ""
	var handler commands.Handler

	switch {
	case *commitMsgFlag, *commitFlag:
		taskName = "commit"
		handler = &commands.Commit{}
	case *reviewFlag:
		taskName = "review"
		handler = &commands.Review{}
	default:
		flag.Usage()
		return
	}

	// --- Pick the right model and thinking setting for this task ---
	// Priority (highest to lowest):
	//   1. --think flag (only for thinking, not model)
	//   2. gitai.json <task>.model and <task>.thinking
	//   3. gitai.json model (global fallback)
	//   4. ENV vars: MODEL, API_BASE, API_KEY

	model := v.GetString("model")
	thinking := *thinkFlag // --think flag is the lowest-level thinking default

	// Override with per-task config if set.
	if taskModel := v.GetString(taskName + ".model"); taskModel != "" {
		model = taskModel
	}
	if v.IsSet(taskName + ".thinking") {
		if tv, ok := v.Get(taskName + ".thinking").(bool); ok {
			thinking = tv
		}
	}

	// --- Run the handler ---
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	result, err := runHandler(ctx, cli, handler, model, thinking, configDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// --- Print the result ---
	fmt.Println()
	fmt.Println("───────────────────────────────────────")
	fmt.Println(result)
	fmt.Println("───────────────────────────────────────")

	// --- Auto-commit if --commit was used ---
	if *commitFlag {
		sanitized := sanitizeForGit(result)
		fmt.Println("\nCommitting changes...")
		cmd := exec.CommandContext(ctx, "git", "commit", "-m", sanitized)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to commit: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully committed!")
	}
}

// sanitizeForGit strips control characters and truncates AI output
// for safe use in git commit messages.
func sanitizeForGit(s string) string {
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
// Returns the config directory path (e.g. ~/.gitai) for use by prompts/loader.
func loadClient(v *viper.Viper) (string, *client.Client, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", nil, fmt.Errorf("could not get current user: %w", err)
	}

	configDir := filepath.Join(currentUser.HomeDir, ".gitai")
	configFile := filepath.Join(configDir, "gitai.json")

	v.SetConfigName("gitai")
	v.SetConfigType("json")
	v.AddConfigPath(configDir)
	_ = v.ReadInConfig() // config file is optional

	// API Base - env vars override config file.
	apiBase := v.GetString("api_base")
	if envBase := os.Getenv("GEMINI_API_BASE"); envBase != "" {
		apiBase = envBase
	}
	if apiBase == "" {
		apiBase = os.Getenv("API_BASE")
	}

	// API Key - env vars override config file.
	apiKey := v.GetString("api_key")
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		apiKey = envKey
	}
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}

	// Model - global fallback (env overrides config).
	model := v.GetString("model")
	if envModel := os.Getenv("MODEL"); envModel != "" {
		model = envModel
	}

	if apiBase == "" {
		return "", nil, fmt.Errorf("no API base URL found. Set the API_BASE environment variable, or add api_base to %s", configFile)
	}

	// A top-level "model" is optional if per-task models are set.
	// But at least one must exist.
	hasTaskModel := v.GetString("commit.model") != "" || v.GetString("review.model") != ""
	if model == "" && !hasTaskModel {
		return "", nil, fmt.Errorf("no model found. Set the MODEL environment variable, add \"model\" to %s, or add \"commit.model\" / \"review.model\" blocks", configFile)
	}

	return configDir, &client.Client{
		APIBase: apiBase,
		APIKey:  apiKey,
		Model:   model, // may be empty - per-task model will override at call time
	}, nil
}

// runHandler stages all changes, fetches the git diff, and runs the selected handler.
func runHandler(ctx context.Context, cli *client.Client, h commands.Handler, model string, thinking bool, configDir string) (string, error) {
	fmt.Printf("Running '%s' (model=%s, thinking=%v)...\n", h.Name(), model, thinking)

	// Stage all changes so git diff sees everything.
	if err := exec.CommandContext(ctx, "git", "add", "-A").Run(); err != nil {
		return "", fmt.Errorf("could not stage changes: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "diff", "--cached")
	diffBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not fetch git diff: %w", err)
	}

	diff := string(diffBytes)
	if diff == "" {
		return "", fmt.Errorf("no changes detected - working tree is clean")
	}

	return h.Run(ctx, cli, diff, model, thinking, configDir)
}

// doUpdate downloads and runs install.sh with GITAI_UPDATE=true,
// replacing the current binary while preserving ~/.gitai/ config.
func doUpdate() {
	// Check if Go is installed (required by install.sh).
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Println("Error: Go is not installed. Please install Go (https://go.dev/).")
		os.Exit(1)
	}

	// Check if curl is installed.
	if _, err := exec.LookPath("curl"); err != nil {
		fmt.Println("Error: curl is not installed. Please install curl.")
		os.Exit(1)
	}

	fmt.Println("Updating GitAI...")
	scriptURL := "https://raw.githubusercontent.com/parth-nformis/gitai/main/install.sh"

	resp, err := http.Get(scriptURL)
	if err != nil {
		fmt.Printf("Failed to download install script: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed to download install script (status: %d)\n", resp.StatusCode)
		os.Exit(1)
	}

	script, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read install script: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("bash", "-")
	cmd.Env = append(os.Environ(), "GITAI_UPDATE=true")
	cmd.Stdin = bytes.NewReader(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("\nUpdate failed. Run manually: curl -sL " + scriptURL + " | GITAI_UPDATE=true bash -")
		os.Exit(1)
	}
	fmt.Println("\nUpdate complete! Re-run gitai to use the new version.")
}

// uninstall deletes the gitai binary. Requires root.
func uninstall() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Could not determine binary path: %v\n", err)
		os.Exit(1)
	}

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
