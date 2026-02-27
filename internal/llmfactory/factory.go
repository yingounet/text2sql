package llmfactory

import (
	"fmt"
	"os"

	"text2sql/internal/llm"
	"text2sql/internal/llm/ollama"
	"text2sql/internal/llm/openai"
)

// ProviderConfig 各 provider 的配置（从 YAML 解析）
type ProviderConfig struct {
	Provider   string           `yaml:"provider"`
	Ollama     *ollama.Config   `yaml:"ollama,omitempty"`
	OpenAI     *openai.Config   `yaml:"openai,omitempty"`
	OpenRouter *openai.Config   `yaml:"openrouter,omitempty"` // OpenAI 兼容 API
	Kimi       *openai.Config   `yaml:"kimi,omitempty"`       // Kimi 月之暗面，OpenAI 兼容 API
}

// NewProviderFromConfig 根据配置创建 Provider
func NewProviderFromConfig(cfg *ProviderConfig) (llm.Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm config is nil")
	}
	name := cfg.Provider
	if name == "" {
		name = "ollama"
	}

	switch name {
	case "ollama":
		c := ollama.Config{BaseURL: "http://localhost:11434", Model: "llama3"}
		if cfg.Ollama != nil {
			if cfg.Ollama.BaseURL != "" {
				c.BaseURL = cfg.Ollama.BaseURL
			}
			if cfg.Ollama.Model != "" {
				c.Model = cfg.Ollama.Model
			}
		}
		return ollama.New(&c), nil
	case "openai":
		c := openai.Config{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"}
		if cfg.OpenAI != nil {
			if cfg.OpenAI.APIKey != "" {
				c.APIKey = cfg.OpenAI.APIKey
			}
			if cfg.OpenAI.BaseURL != "" {
				c.BaseURL = cfg.OpenAI.BaseURL
			}
			if cfg.OpenAI.Model != "" {
				c.Model = cfg.OpenAI.Model
			}
		}
		return openai.New(&c), nil
	case "openrouter":
		c := openai.Config{BaseURL: "https://openrouter.ai/api/v1", Model: "anthropic/claude-3-haiku"}
		if cfg.OpenRouter != nil {
			if cfg.OpenRouter.APIKey != "" {
				c.APIKey = cfg.OpenRouter.APIKey
			}
			if cfg.OpenRouter.BaseURL != "" {
				c.BaseURL = cfg.OpenRouter.BaseURL
			}
			if cfg.OpenRouter.Model != "" {
				c.Model = cfg.OpenRouter.Model
			}
		}
		return openai.New(&c), nil
	case "kimi":
		c := openai.Config{BaseURL: "https://api.moonshot.cn/v1", Model: "kimi-k2.5"}
		if cfg.Kimi != nil {
			if cfg.Kimi.APIKey != "" {
				c.APIKey = cfg.Kimi.APIKey
			}
			if cfg.Kimi.BaseURL != "" {
				c.BaseURL = cfg.Kimi.BaseURL
			}
			if cfg.Kimi.Model != "" {
				c.Model = cfg.Kimi.Model
			}
		}
		if c.APIKey == "" {
			c.APIKey = os.Getenv("MOONSHOT_API_KEY")
		}
		return openai.New(&c), nil
	default:
		return nil, fmt.Errorf("unknown llm provider: %s", name)
	}
}
