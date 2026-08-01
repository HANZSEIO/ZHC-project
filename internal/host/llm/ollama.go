package llm

import (
	"context"
	"encoding/json"
	"bytes"
	"io"
	"net/http"
	"time"
)

type ollamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}
type ollamaRequest struct {
	Model    string        `json:"model"`
	Messages []Message     `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ollamaResponse struct {
	Message struct {
	Content string
	}
}

func NewOllamaProvider(cfg OllamaConfig) (Provider, error) {
	return &ollamaProvider{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (o *ollamaProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	reqBody := ollamaRequest{
		Model:    o.model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, err
	}

	return &Response{
		Choices: []Choice{{
			Message: Message{
				Role:    "assistant",
				Content: ollamaResp.Message.Content,
			},
		}},
	}, nil
}