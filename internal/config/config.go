package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"text2sql/internal/llmfactory"
)

// Config 应用配置
type Config struct {
	Server       ServerConfig            `yaml:"server"`
	APIKey       string                  `yaml:"api_key"`
	Database     DatabaseConfig          `yaml:"database"`
	ContextStore string                  `yaml:"context_store"` // memory | sqlite，默认 memory
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
