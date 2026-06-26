package config

// TaskConfig holds per-task settings (model and thinking mode).
// Mapped from gitai.json keys like "commit" and "review".
type TaskConfig struct {
	Model    string `json:"model"`
	Thinking *bool  `json:"thinking,omitempty"` // pointer so we can detect "not set"
}

// Config holds the full gitai configuration loaded from ~/.gitai/gitai.json.
type Config struct {
	APIBase string       `json:"api_base"`
	APIKey  string       `json:"api_key"`
	Model   string       `json:"model"` // global fallback when per-task model is not set
	Commit  TaskConfig   `json:"commit"`
	Review  TaskConfig   `json:"review"`
}
