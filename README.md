# GitAI: AI-Powered Git Reviewer & Commit Generator

GitAI is an AI-powered command-line tool designed to review code modifications in your Git repository and automatically generate structured, logical git commit messages using the Gemini API.

---

## Installation

### Automated Global Installation (Linux & macOS)
You can install GitAI globally to `/usr/local/bin` using a single command:

```bash
curl -sSL https://raw.githubusercontent.com/parthdande/gitai/main/install.sh | bash
```

*Note: The script compiles the tool from source and moves the binary to `/usr/local/bin` (this step will prompt for sudo privileges to write to the global directory).*

### Manual Installation
If you prefer to compile it manually:
```bash
git clone https://github.com/parthdande/gitai.git
cd gitai
go build -o gitai cmd/main.go
mv gitai /usr/local/bin/
```

---

## Configuration

GitAI supports authentication and configuration in two ways:

### 1. Environment Variable (Recommended)
Set the `GEMINI_API_KEY` variable in your terminal environment (e.g., in your `~/.bashrc` or `~/.zshrc`):
```bash
export GEMINI_API_KEY="your_gemini_api_key_here"
```

### 2. Config File
Alternatively, you can create a local configuration file in `./config/gitai.json`:
```json
{
  "api_key": "your_gemini_api_key_here",
  "model": "gemini-3.1-flash-lite"
}
```

---

## Usage

Once installed globally, navigate to any Git repository and run `gitai` with the following flags:

### 1. Preview Suggested Commit Message
Analyzes your current changes (both staged and unstaged) and prints a suggested commit message structure to the console:
```bash
gitai -commitmsg
```

### 2. Auto-Stage and Commit Changes
Generates a commit message and automatically executes `git commit -a -m` to stage and commit your changes in one step:
```bash
gitai -commit
```
