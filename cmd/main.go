package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/parthdande/gitai/client"
	"github.com/spf13/viper"
)

const commitSystemPrompt = "You are an expert software engineer. Generate a structured conventional commit message based on the provided git diff. It must start with a short header line (conventional commit style) summarizing the overall change, followed by a blank line, then a brief paragraph describing the purpose of the changes, followed by another blank line, and a bulleted list detailing what the changes accomplished (focusing on logical and functional changes rather than listing file names). Do not include markdown formatting (like ```), just return the raw text."

func main() {
	commitMsgFlag := flag.Bool("commitmsg", false, "Generate a commit message from git diff and print it")
	commitFlag := flag.Bool("commit", false, "Generate a commit message and automatically commit all changes")
	updateFlag := flag.Bool("update", false, "Update gitai to the latest version")
	uninstallFlag := flag.Bool("uninstall", false, "Uninstall gitai from the system")

	flag.Usage = func() {
		fmt.Println("GitAI: AI-Powered Git Reviewer & Commit Generator")
		fmt.Println("\nUsage:")
		fmt.Println("  gitai [flags]")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Uninstall: remove the binary from /usr/local/bin
	if *uninstallFlag {
		fmt.Println("Uninstalling GitAI...")
		cmd := exec.Command("sudo", "rm", "-f", "/usr/local/bin/gitai")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Uninstall failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("GitAI uninstalled successfully!")
		return
	}

	// Self-update: just re-run the install script (cache-busted URL)
	if *updateFlag {
		fmt.Println("Updating GitAI to the latest version...")
		cmd := exec.Command("bash", "-c", "curl -sSL \"https://raw.githubusercontent.com/parthdande/gitai/main/install.sh?v=$(date +%s)\" | bash")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Load config from file if it exists, otherwise fall back to environment variable
	viper.SetConfigName("gitai")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")
	err := viper.ReadInConfig()

	apiKey := viper.GetString("api_key")
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	if err != nil && apiKey == "" {
		fmt.Println("ERROR: No API key found. Set the GEMINI_API_KEY environment variable or create ./config/gitai.json")
		os.Exit(1)
	}

	model := viper.GetString("model")
	if model == "" {
		model = "gemini-3.1-flash-lite"
	}

	gemini := client.Gemini{
		APIKey: apiKey,
		Model:  model,
	}

	if *commitMsgFlag || *commitFlag {
		fmt.Println("Fetching git diff...")
		diffBytes, err := exec.Command("git", "diff", "HEAD").Output()
		if err != nil {
			fmt.Printf("Error fetching git diff: %v\n", err)
			os.Exit(1)
		}

		diff := string(diffBytes)
		if diff == "" {
			fmt.Println("No changes detected. Your working tree is clean.")
			return
		}

		fmt.Println("Generating commit message via Gemini...")
		prompt := fmt.Sprintf("Analyze this git diff and write a commit message:\n\n%s", diff)
		commitMessage, err := gemini.GeminiAPI(prompt, commitSystemPrompt)
		if err != nil {
			fmt.Printf("Error generating commit message: %v\n", err)
			os.Exit(1)
		}

		if *commitMsgFlag {
			fmt.Println("\nSuggested Commit Message:")
			fmt.Println("-------------------------------------------")
			fmt.Println(commitMessage)
			fmt.Println("-------------------------------------------")
		}

		if *commitFlag {
			fmt.Println("Committing changes...")
			commitCmd := exec.Command("git", "commit", "-a", "-m", commitMessage)
			commitCmd.Stdout = os.Stdout
			commitCmd.Stderr = os.Stderr
			if err := commitCmd.Run(); err != nil {
				fmt.Printf("Failed to commit: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Successfully committed!")
		}
		return
	}

	flag.Usage()
}
