package main 

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/parthdande/gitai/client"
	"github.com/parthdande/gitai/config"
	"github.com/spf13/viper"
)


func main() {  
	commitMsgFlag := flag.Bool("commitmsg", false, "Generate a commit message from git diff and print it")
	commitFlag := flag.Bool("commit", false, "Generate a commit message and automatically commit all changes")
	
	flag.Usage = func() {
		fmt.Println("GitAI: AI-Powered Git Reviewer & Commit Generator")
		fmt.Println("\nUsage:")
		fmt.Println("  gitai [flags]")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	viper.SetConfigName("gitai")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")
	err := viper.ReadInConfig()
	apiKey := viper.GetString("api_key")

	// Allow environment variable override
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	if err != nil && apiKey == "" { 
		fmt.Println("ERROR: Configuration file not found, and GEMINI_API_KEY is not set.")
		fmt.Println("Please set GEMINI_API_KEY environment variable or create ./config/gitai.json")
		os.Exit(1)
	}

	model := viper.GetString("model")
	if model == "" {
		model = "gemini-3.1-flash-lite"
	}

	cfg := config.Config{
		APIKey: apiKey,
		APIUrl: viper.GetString("api_url"),
		Model:  model,
	}

	gemini := client.Gemini{ 
		APIKey: cfg.APIKey, 
		APIUrl: cfg.APIUrl, 
		Model:  cfg.Model,
	}

	if *commitMsgFlag || *commitFlag {
		fmt.Println("Fetching git diff...")
		// Use 'git diff HEAD' to capture both staged and unstaged modifications
		cmd := exec.Command("git", "diff", "HEAD")
		diffBytes, err := cmd.Output()
		if err != nil {
			fmt.Printf("Error fetching git diff: %v\n", err)
			os.Exit(1)
		}

		diff := string(diffBytes)
		if diff == "" {
			fmt.Println("No git diff detected. Your working tree is clean.")
			return
		}

		systemPrompt := "You are an expert software engineer. Generate a structured conventional commit message based on the provided git diff. It must start with a short header line (conventional commit style) summarizing the overall change, followed by a blank line, then a brief paragraph describing the purpose of the changes, followed by another blank line, and a bulleted list detailing what the changes accomplished (focusing on logical and functional changes rather than listing file names). Do not include markdown formatting (like ```), just return the raw text."
		prompt := fmt.Sprintf("Analyze this git diff and write a commit message:\n\n%s", diff)

		fmt.Println("Generating commit message via Gemini...")
		commitMessage, err := gemini.GeminiAPI(prompt, systemPrompt)
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
			// Runs 'git commit -a -m "message"' to stage and commit modified/deleted files
			commitCmd := exec.Command("git", "commit", "-a", "-m", commitMessage)
			commitCmd.Stdout = os.Stdout
			commitCmd.Stderr = os.Stderr
			if err := commitCmd.Run(); err != nil {
				fmt.Printf("Failed to execute git commit: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Successfully committed changes!")
		}
		return
	}

	// If no valid flag is passed, show usage instructions
	fmt.Println("GitAI: AI-Powered Git Reviewer & Commit Generator")
	fmt.Println("Usage:")
	flag.PrintDefaults()
}
