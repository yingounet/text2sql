package llm

import (
	"fmt"
	"sync"
)

var (
	providers   = make(map[string]Provider)
	providersMu sync.RWMutex
)

// Register 注册 Provider
func Register(p Provider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[p.Name()] = p
}

// Get 获取已注册的 Provider
func Get(name string) (Provider, error) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("llm provider not found: %s", name)
	}
	return p, nil
}

// List 列出所有已注册的 Provider 名称
func List() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()
	names := make([]string, 0, len(providers))
	for n := range providers {
		names = append(names, n)
	}
	return names
}
