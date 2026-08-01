package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type LLMConfig struct {
	Provider string `yaml:"provider"`
	Ollama   OllamaConfig `yaml:"ollama"`
	Cloud  CloudConfig `yaml:"cloud"`
}

type OllamaConfig struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type CloudConfig struct {
	APIKey string `yaml:"api_key"`
	Model string `yaml:"model"`
}

type QEMUConfig struct {
	BaseImage string `yaml:"base_image"`
	OverlayPath string `yaml:"overlay_path"`
	XMLConfig     string `yaml:"xml_config"`
}

type TransportConfig struct {
	SocketPath string `yaml:"socket_path"`
	SSH SSHConfig `yaml:"ssh"`
}

type SSHConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	KeyPath string `yaml:"key_path"`
}

type Config struct {
	LLM       LLMConfig       `yaml:"llm"`
	QEMU      QEMUConfig      `yaml:"qemu"`
	Transport TransportConfig `yaml:"transport"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}