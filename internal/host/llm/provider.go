package llm

import (
	"fmt"
)

type Config struct {
	Provider string
	Ollama   OllamaConfig
	Cloud    CloudConfig
}

type OllamaConfig struct {
	BaseURL string
	Model   string
}

type CloudConfig struct {
	APIKey string
	Model  string
}

func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
		case "ollama":
			return NewOllamaProvider(cfg.Ollama)
		case "cloud":
			return NewCloudProvider(cfg.Cloud)
		default:
			return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}