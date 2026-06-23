package client

// Client holds the configuration needed to make API calls to any
// OpenAI-compatible endpoint (vLLM, Ollama, OpenAI, LiteLLM, etc.).
type Client struct {
	APIBase string
	APIKey  string
	Model   string
}
