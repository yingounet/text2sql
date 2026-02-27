package llm

import "context"

// Provider 统一 LLM provider 接口
type Provider interface {
	Name() string
	Complete(ctx context.Context, req *CompleteRequest) (*CompleteResponse, error)
}

// CompleteRequest 标准化请求
type CompleteRequest struct {
	Model       string    // 模型名
	Messages    []Message // 消息列表
	MaxTokens   int       // 最大生成 token 数
	Temperature float64   // 温度
}

// CompleteResponse 标准化响应
type CompleteResponse struct {
	Content string  // 模型返回文本
	Usage   *Usage  // token 用量（可选）
}

// Message 消息格式（OpenAI 风格）
type Message struct {
	Role    string `json:"role"`    // system/user/assistant
	Content string `json:"content"`
}

// Usage token 用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
