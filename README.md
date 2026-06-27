# gitai

AI-assisted git commits and code reviews. Analyzes your staged changes and generates conventional commit messages or code reviews using **any OpenAI-compatible API** (vLLM, Ollama, OpenAI, LiteLLM, etc.).

---

## Installation

### Automated (Linux & macOS)

Run the install script to compile and place the binary in `/usr/local/bin`:

```bash
bash <(curl -s https://raw.githubusercontent.com/parth-nformis/gitai/main/install.sh)
```

The script clones the latest code, builds it in a temp directory, and moves the binary to `/usr/local/bin`. Your `~/.gitai/` config is preserved on subsequent runs.

### Manual

```bash
git clone https://github.com/parth-nformis/gitai.git
cd gitai
go build -o gitai cmd/main.go
sudo mv gitai /usr/local/bin/
```

---

## Configuration

GitAI works with any OpenAI-compatible API endpoint.

### Environment Variables

```bash
# Required: Base URL of your API server
export API_BASE="http://localhost:8000/v1"

# Optional: API key (not needed for local vLLM/Ollama)
export API_KEY="your-key-here"

# Optional: Model name (global fallback)
export MODEL="Qwen/Qwen3-32B"
```

### Config File

Created automatically at `~/.gitai/gitai.json` on first install:

```json
{
  "api_base": "http://localhost:8000/v1",
  "api_key": "",
  "model": "Qwen/Qwen3-32B"
}
```

### Per-task Model and Thinking Mode

Use different models for commit messages vs code reviews:

```json
{
  "api_base": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "commit": { "model": "gpt-4o-mini", "thinking": false },
  "review": { "model": "gpt-4o", "thinking": true }
}
```

Priority (highest to lowest):
1. `--think` CLI flag (thinking only)
2. Per-task config (`commit.model`, `commit.thinking`, `review.model`, `review.thinking`)
3. Global config (`model` key)
4. Environment variables (`MODEL`, `API_BASE`, `API_KEY`)

### Example Configurations

**vLLM (self-hosted):**
```json
{
  "api_base": "http://localhost:8000/v1",
  "model": "meta-llama/Llama-3.3-70B-Instruct"
}
```

**Ollama:**
```json
{
  "api_base": "http://localhost:11434/v1",
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

Navigate to any Git repository and run `gitai` with the following flags:

### Generate Commit Message

Prints a suggested commit message from your staged changes:

```bash
gitai -commitmsg
```

### Auto-commit

Generates a commit message and commits all staged changes:

```bash
gitai -commit
```

### Code Review

Reviews staged changes for security, quality, and best practices:

```bash
gitai -review
```

### Thinking Mode

Enable extended thinking for models that support it (e.g. DeepSeek). Falls back automatically if the model doesn't support it:

```bash
gitai -commit -think
gitai -review -think
```

Or enable permanently in `~/.gitai/gitai.json`:

```json
{
  "commit": { "thinking": true },
  "review": { "thinking": false }
}
```

### Custom System Prompts

Override the built-in prompts by placing `.md` files in `~/.gitai/system_prompts/`:

```
~/.gitai/system_prompts/
├── commit.md   # Overrides commit message generation prompt
└── review.md   # Overrides code review prompt
```

Edit the files and the next `gitai` run picks them up — no rebuild needed. If a file is missing, gitai falls back to its built-in default.

### Update to Latest Version

```bash
gitai -update
```

Downloads and runs the install script to replace the binary. Config at `~/.gitai/` is preserved.

### Uninstall

Removes the `gitai` binary (requires sudo):

```bash
sudo gitai -uninstall
```

---

## Architecture

```
gitai/
├── cmd/main.go          # CLI flags, config loading, update/uninstall handlers
├── client/              # HTTP client for OpenAI-compatible APIs
│   ├── client.go        # Client struct (api_base, api_key, model, http client)
│   └── api_call.go      # API call logic, thinking mode, auto-fallback
├── commands/            # Feature handlers (commit, review)
│   ├── handler.go       # Handler interface (Name + Run)
│   ├── commit.go        # Commit message generation
│   └── review.go        # Code review generation
├── config/              # Typed config structs
│   └── config.go        # Config and TaskConfig types
├── prompts/             # System prompts
│   ├── loader.go        # Loads custom prompts from disk, falls back to defaults
│   ├── commit.go        # Default commit system prompt
│   └── review.go        # Default review system prompt
└── install.sh           # Automated install: clone, build, install to /usr/local/bin
```

## License

MIT
