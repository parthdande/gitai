package client

import "net/http"

// Client holds the configuration needed to make API calls to any
// OpenAI-compatible endpoint (vLLM, Ollama, OpenAI, LiteLLM, etc.).
//
// Model is the global fallback — individual calls may override it.
// HTTPClient is reused across calls for connection pooling.
type Client struct {
	APIBase    string
	APIKey     string
	Model      string
	HTTPClient *http.Client // nil = uses default with 120s timeout
}
