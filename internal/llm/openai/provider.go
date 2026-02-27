package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"text2sql/internal/llm"
)

const name = "openai"
const defaultHTTPTimeout = 60 * time.Second

// Config OpenAI 配置
type Config struct {
	APIKey string // 支持环境变量 OPENAI_API_KEY
	BaseURL string // 默认为 https://api.openai.com/v1
	Model  string  // 如 gpt-4o
}

// Provider OpenAI 实现（兼容 OpenRouter、Azure 等 OpenAI 兼容 API）
type Provider struct {
	client *http.Client
	config Config
}

// New 创建 OpenAI Provider
func New(cfg *Config) *Provider {
	p := &Provider{
		client: &http.Client{Timeout: defaultHTTPTimeout},
	}
	if cfg != nil {
		p.config = *cfg
	}
	if p.config.APIKey == "" {
		p.config.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if p.config.BaseURL == "" {
		p.config.BaseURL = "https://api.openai.com/v1"
	}
	if p.config.Model == "" {
		p.config.Model = "gpt-4o"
	}
	return p
}

// Name 返回 provider 名称
func (p *Provider) Name() string {
	return name
}

// openaiRequest OpenAI chat completions 请求
type openaiRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse OpenAI 响应
type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Complete 调用 OpenAI API
func (p *Provider) Complete(ctx context.Context, req *llm.CompleteRequest) (*llm.CompleteResponse, error) {
	msgs := make([]message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = message{Role: m.Role, Content: m.Content}
	}

	body := openaiRequest{
		Model:       p.config.Model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	if body.MaxTokens <= 0 {
		body.MaxTokens = 2048
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openai: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: unexpected status %d", resp.StatusCode)
	}

	var out openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}

	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}

	return &llm.CompleteResponse{
		Content: out.Choices[0].Message.Content,
		Usage: &llm.Usage{
			PromptTokens:     out.Usage.PromptTokens,
			CompletionTokens: out.Usage.CompletionTokens,
			TotalTokens:      out.Usage.TotalTokens,
		},
	}, nil
}
