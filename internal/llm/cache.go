package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type CachedProvider struct {
	provider Provider
	cache    map[string]*cacheEntry
	mu       sync.RWMutex
	ttl      time.Duration
}

type cacheEntry struct {
	response  *CompleteResponse
	expiresAt time.Time
}

func NewCachedProvider(provider Provider, ttl time.Duration) *CachedProvider {
	cp := &CachedProvider{
		provider: provider,
		cache:    make(map[string]*cacheEntry),
		ttl:      ttl,
	}
	go cp.cleanupExpired()
	return cp
}

func (cp *CachedProvider) Name() string {
	return cp.provider.Name()
}

func (cp *CachedProvider) Complete(ctx context.Context, req *CompleteRequest) (*CompleteResponse, error) {
	cacheKey := cp.generateCacheKey(req)

	cp.mu.RLock()
	if entry, exists := cp.cache[cacheKey]; exists && time.Now().Before(entry.expiresAt) {
		cp.mu.RUnlock()
		return entry.response, nil
	}
	cp.mu.RUnlock()

	resp, err := cp.provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	cp.mu.Lock()
	cp.cache[cacheKey] = &cacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(cp.ttl),
	}
	cp.mu.Unlock()

	return resp, nil
}

func (cp *CachedProvider) generateCacheKey(req *CompleteRequest) string {
	hash := sha256.New()
	for _, msg := range req.Messages {
		hash.Write([]byte(msg.Role))
		hash.Write([]byte(msg.Content))
	}
	hash.Write([]byte(req.Model))
	return hex.EncodeToString(hash.Sum(nil))
}

func (cp *CachedProvider) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cp.mu.Lock()
		now := time.Now()
		for key, entry := range cp.cache {
			if now.After(entry.expiresAt) {
				delete(cp.cache, key)
			}
		}
		cp.mu.Unlock()
	}
}

func (cp *CachedProvider) Clear() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.cache = make(map[string]*cacheEntry)
}
