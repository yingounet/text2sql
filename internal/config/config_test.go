package config

import (
	"testing"

	"text2sql/internal/llmfactory"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with api_key",
			config: Config{
				APIKey:       "test-key",
				Server:       ServerConfig{Port: 8080},
				LLM:          llmfactory.ProviderConfig{Provider: "ollama"},
				ContextStore: "memory",
			},
			wantErr: false,
		},
		{
			name: "valid config with api_keys",
			config: Config{
				APIKeys:      []string{"key1", "key2"},
				Server:       ServerConfig{Port: 8080},
				LLM:          llmfactory.ProviderConfig{Provider: "openai"},
				ContextStore: "sqlite",
			},
			wantErr: false,
		},
		{
			name: "missing api_key",
			config: Config{
				Server:       ServerConfig{Port: 8080},
				LLM:          llmfactory.ProviderConfig{Provider: "ollama"},
				ContextStore: "memory",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: Config{
				APIKey:       "test-key",
				Server:       ServerConfig{Port: 0},
				LLM:          llmfactory.ProviderConfig{Provider: "ollama"},
				ContextStore: "memory",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: Config{
				APIKey:       "test-key",
				Server:       ServerConfig{Port: 8080},
				LLM:          llmfactory.ProviderConfig{Provider: "invalid"},
				ContextStore: "memory",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
