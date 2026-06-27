// Package client provides the HTTP client for talking to OpenAI-compatible APIs.
//
// The Client struct holds connection settings (api_base, api_key, model) and a
// reused HTTP client with connection pooling. Generate sends a prompt to the
// chat completions endpoint and supports optional thinking mode with auto-fallback.
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
