package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"text2sql/internal/llmfactory"
)

// Config 应用配置
type Config struct {
	Server       ServerConfig              `yaml:"server"`
	APIKey       string                    `yaml:"api_key"`
	APIKeys      []string                  `yaml:"api_keys"` // 支持多个 API Key
	Database     DatabaseConfig            `yaml:"database"`
	ContextStore string                    `yaml:"context_store"` // memory | sqlite，默认 memory
	LLM          llmfactory.ProviderConfig `yaml:"llm"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 展开环境变量
	cfg.APIKey = os.ExpandEnv(cfg.APIKey)
	for i := range cfg.APIKeys {
		cfg.APIKeys[i] = os.ExpandEnv(cfg.APIKeys[i])
	}
	if cfg.LLM.OpenAI != nil {
		cfg.LLM.OpenAI.APIKey = os.ExpandEnv(cfg.LLM.OpenAI.APIKey)
	}
	if cfg.LLM.OpenRouter != nil {
		cfg.LLM.OpenRouter.APIKey = os.ExpandEnv(cfg.LLM.OpenRouter.APIKey)
	}
	if cfg.LLM.Kimi != nil {
		cfg.LLM.Kimi.APIKey = os.ExpandEnv(cfg.LLM.Kimi.APIKey)
	}

	// 环境变量覆盖
	if k := os.Getenv("API_KEY"); k != "" {
		cfg.APIKey = k
	}

	// 如果只有一个 api_key，添加到 api_keys 列表
	if cfg.APIKey != "" && len(cfg.APIKeys) == 0 {
		cfg.APIKeys = []string{cfg.APIKey}
	}

	// 默认值
	if cfg.Server.Port <= 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" && cfg.Database.Driver == "sqlite" {
		cfg.Database.DSN = "./data/text2sql.db"
	}
	if cfg.ContextStore == "" {
		cfg.ContextStore = "memory"
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv 从环境变量指定的路径或默认路径加载
func LoadFromEnv() (*Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yaml"
	}
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			path = filepath.Join(cwd, path)
		}
	}
	return Load(path)
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.APIKeys) == 0 && c.APIKey == "" {
		return errors.New("api_key or api_keys is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("invalid server port (must be 1-65535)")
	}
	if c.LLM.Provider == "" {
		return errors.New("llm.provider is required")
	}
	validProviders := map[string]bool{
		"ollama": true, "openai": true, "openrouter": true, "kimi": true,
	}
	if !validProviders[c.LLM.Provider] {
		return fmt.Errorf("invalid llm.provider: %s (supported: ollama, openai, openrouter, kimi)", c.LLM.Provider)
	}
	if c.ContextStore != "memory" && c.ContextStore != "sqlite" {
		return fmt.Errorf("invalid context_store: %s (must be memory or sqlite)", c.ContextStore)
	}
	return nil
}
