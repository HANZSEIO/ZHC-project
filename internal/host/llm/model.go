package llm

import (
	"context"
)

type Message struct {
	Role    string
	Content string
}

type ToolCall struct {
	Name string
	Arguments map[string]interface{}
}

type Choice struct {
	Message Message
	ToolCall []ToolCall
}

type Response struct {
	Choices []Choice
}

type Provider interface {
	Chat(ctx context.Context, messages []Message) (*Response, error)
}