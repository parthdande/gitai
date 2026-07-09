# gitai

AI-assisted git commits, code reviews, and pull request descriptions. Analyzes your staged changes (or branch diffs) and generates conventional commit messages, code reviews, or PR descriptions using **any OpenAI-compatible API** — vLLM, Ollama, OpenAI, LiteLLM, Groq, and more.

Built in Go. Zero runtime dependencies beyond Go itself during install. Streaming-first with automatic fallback to non-streaming responses.

---

## Quick Start

```bash
# Install
bash <(curl -s https://raw.githubusercontent.com/parth-nformis/gitai/main/install.sh)

# Configure
export API_BASE="http://localhost:8000/v1"
export MODEL="Qwen/Qwen3-32B"

# Generate a commit message from staged changes
gitai -commitmsg

# Generate and commit in one step
gitai -commit

# Review staged changes
gitai -review

# Generate a PR description for the current branch
gitai -pullreq
```

---

## Installation

### Automated (Linux & macOS)

```bash
bash <(curl -s https://raw.githubusercontent.com/parth-nformis/gitai/main/install.sh)
```

Clones the latest code, builds in a temp directory, and installs the binary to `/usr/local/bin`. Config at `~/.gitai/` is preserved across installs.

### Manual

```bash
git clone https://github.com/parth-nformis/gitai.git
cd gitai
go build -o gitai cmd/main.go
sudo mv gitai /usr/local/bin/
```

### Update

```bash
gitai -update
```

### Uninstall

```bash
sudo gitai -uninstall
```

---

## Configuration

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

Assign different models and thinking settings per command:

```json
{
  "api_base": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "commit": { "model": "gpt-4o-mini", "thinking": false },
  "review": { "model": "gpt-4o", "thinking": true },
  "pullreq": { "model": "gpt-4o", "thinking": false }
}
```

### Environment Variables

| Variable | Config Key | Description |
|----------|-----------|-------------|
| `API_BASE` | `api_base` | Base URL of your API server (required) |
| `API_KEY` | `api_key` | API key (optional for local servers) |
| `MODEL` | `model` | Global fallback model name |
| `GEMINI_API_BASE` | — | Alternative for `API_BASE` |
| `GEMINI_API_KEY` | — | Alternative for `API_KEY` |

Priority (highest to lowest):

1. Per-task config (`commit.model`, `review.model`, `pullreq.model`)
2. Global config (`model` key)
3. Environment variables

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

### Generate Commit Message

Analyzes staged changes and prints a conventional commit message:

```bash
gitai -commitmsg
```

### Auto-commit

Stages all changes, generates a commit message, and commits:

```bash
gitai -commit
```

### Code Review

Reviews staged changes for security, quality, and best practices:

```bash
gitai -review
```

### Pull Request Description

Generates a structured PR description from the diff between the current branch and `main`:

```bash
gitai -pullreq          # or: gitai -pr
gitai -pullreq -branch develop   # custom base branch
```

Output includes: summary, changes, testing steps, and breaking changes.

### Thinking Mode

Enable extended reasoning for models that support it (e.g. DeepSeek). Falls back automatically if the model doesn't support thinking:

```bash
gitai -commit -think
gitai -review -think
```

Or enable permanently in config per-task:

```json
{
  "commit": { "thinking": true },
  "review": { "thinking": false }
}
```

### Custom System Prompts

Override built-in prompts by placing `.md` files in `~/.gitai/system_prompts/`:

```
~/.gitai/system_prompts/
├── commit.md    # Commit message generation
├── review.md    # Code review
└── pullreq.md   # PR description
```

Changes take effect immediately — no rebuild needed. Missing files fall back to built-in defaults.

---

## Architecture

```
gitai/
├── cmd/main.go              # CLI entry point, flag parsing, config loading
├── client/                  # OpenAI-compatible HTTP client
│   ├── client.go            # Client struct (api_base, api_key, model)
│   └── api_call.go          # Streaming + non-streaming API calls, thinking mode, auto-fallback
├── commands/                # Feature handlers
│   ├── handler.go           # Handler interface (Name + Run)
│   ├── commit.go            # Commit message generation
│   ├── review.go            # Code review generation
│   └── pullreq.go           # PR description generation
├── config/                  # Typed configuration
│   └── config.go            # Config and TaskConfig types
├── diffprep/                # Diff preprocessing pipeline
│   └── preprocess.go        # Noise filtering, file stats, truncation, chunking
├── prompts/                 # System prompt management
│   ├── loader.go            # Custom prompt loader with built-in fallback
│   ├── commit.go            # Default commit prompt
│   ├── review.go            # Default review prompt
│   └── pullreq.go           # Default PR description prompt
└── install.sh               # Automated install script
```

### Key Features

- **Streaming-first**: Uses SSE streaming with automatic fallback to non-streaming for APIs that don't support it
- **Thinking mode**: Supports extended reasoning via `chat_template_kwargs`, with auto-fallback on failure
- **Diff preprocessing**: Filters noise files (lock files, binaries, generated code), truncates oversized diffs, computes per-file stats
- **Hierarchical chunking**: Splits large diffs into manageable chunks for models with context limits
- **Per-task configuration**: Different models and settings per command (commit, review, pullreq)
- **Hot-reload prompts**: Custom prompts picked up without rebuild

---

## License

MIT
