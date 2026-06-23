# gitai

AI-assisted git commits and messages. Analyzes your git diff and generates structured conventional commit messages using **any OpenAI-compatible API** (vLLM, Ollama, OpenAI, LiteLLM, etc.). Can also auto-commit directly.

---

## Installation

### Automated Global Installation (Linux & macOS)
Run the install script to compile and place the binary in `/usr/local/bin`:

```bash
bash <(curl -s https://raw.githubusercontent.com/parthdande/gitai/main/install.sh)
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

GitAI works with **any OpenAI-compatible API endpoint**. This includes vLLM, Ollama, OpenAI, LiteLLM, and more.

### Environment Variables (Recommended)

```bash
# Required: Base URL of your API server
export API_BASE="http://localhost:8000/v1"

# Optional: API key (not needed for local vLLM/Ollama)
export API_KEY="your-key-here"

# Optional: Model name (default: Qwen/Qwen3-32B)
export MODEL="Qwen/Qwen3-32B"
```

### Config File (Auto-Created)

The installation script automatically creates `~/.gitai/gitai.json` in your home directory:

**Path:** `~/.gitai/gitai.json`

```json
{
  "api_base": "http://localhost:8000/v1",
  "api_key": "",
  "model": "Qwen/Qwen3-32B"
}
```

### Example Configurations

**vLLM (self-hosted):**
```json
{
  "api_base": "http://localhost:8000/v1",
  "api_key": "",
  "model": "meta-llama/Llama-3.3-70B-Instruct"
}
```

**Ollama:**
```json
{
  "api_base": "http://localhost:11434/v1",
  "api_key": "",
  "model": "qwen3:32b"
}
```

**OpenAI:**
```json
{
  "api_base": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4o-mini"
}
```

---

## Usage

Once installed globally, navigate to any Git repository and run `gitai` with the following flags:

### 1. Preview Suggested Commit Message
Analyzes your current changes and prints a suggested commit message:
```bash
gitai -commitmsg
```

### 2. Auto-Stage and Commit Changes
Generates a commit message and automatically commits:
```bash
gitai -commit
```

### 3. Update to Latest Version
```bash
gitai -update
```

### 4. Uninstall
Removes the `gitai` binary from `/usr/local/bin`:
```bash
gitai -uninstall
```
