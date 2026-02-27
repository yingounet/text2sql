package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"text2sql/internal/llm"
)

const name = "ollama"
const defaultHTTPTimeout = 60 * time.Second

// Config Ollama 配置
type Config struct {
	BaseURL string // 如 http://localhost:11434
	Model   string // 如 qwen2.5:7b
}

// Provider Ollama 实现
type Provider struct {
	client *http.Client
	config Config
}

// New 创建 Ollama Provider
func New(cfg *Config) *Provider {
	p := &Provider{
		client: &http.Client{Timeout: defaultHTTPTimeout},
	}
	if cfg != nil {
		p.config = *cfg
	}
	if p.config.BaseURL == "" {
		p.config.BaseURL = "http://localhost:11434"
	}
	if p.config.Model == "" {
		p.config.Model = "llama3"
	}
	return p
}

// Name 返回 provider 名称
func (p *Provider) Name() string {
	return name
}

// ollamaRequest Ollama API 请求体
type ollamaRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Stream      bool      `json:"stream"`
	Options     options   `json:"options,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type options struct {
	NumPredict int `json:"num_predict,omitempty"`
}

// ollamaResponse Ollama API 响应体
type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	EvalCount int `json:"eval_count,omitempty"`
}

// Complete 调用 Ollama API
func (p *Provider) Complete(ctx context.Context, req *llm.CompleteRequest) (*llm.CompleteResponse, error) {
	msgs := make([]message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = message{Role: m.Role, Content: m.Content}
	}

	t := req.Temperature
	body := ollamaRequest{
		Model:    p.config.Model,
		Messages: msgs,
		Stream:   false,
		Options:  options{NumPredict: req.MaxTokens},
	}
	if t > 0 {
		body.Temperature = &t
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	var out ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama: decode response: %w", err)
	}

	content := strings.TrimSpace(out.Message.Content)
	usage := &llm.Usage{
		CompletionTokens: out.EvalCount,
		TotalTokens:      out.EvalCount, // Ollama 不返回 prompt tokens，简化处理
	}

	return &llm.CompleteResponse{
		Content: content,
		Usage:   usage,
	}, nil
}
