package llm

import (
	"context"
	"fmt"
)

type cloudProvider struct {
	config CloudConfig
}

func NewCloudProvider(cfg CloudConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for cloud provider")
	}
	return &cloudProvider{config: cfg}, nil
}

func (p *cloudProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	return nil, fmt.Errorf("cloud provider is not implemented yet")
}