package api

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*clientInfo
	limit    int
	window   time.Duration
}

type clientInfo struct {
	count     int
	expiresAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientInfo),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupExpiredClients()
	return rl
}

func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[clientID]

	if !exists || now.After(info.expiresAt) {
		rl.requests[clientID] = &clientInfo{
			count:     1,
			expiresAt: now.Add(rl.window),
		}
		return true
	}

	if info.count >= rl.limit {
		return false
	}

	info.count++
	return true
}

func (rl *RateLimiter) cleanupExpiredClients() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, info := range rl.requests {
			if now.After(info.expiresAt) {
				delete(rl.requests, id)
			}
		}
		rl.mu.Unlock()
	}
}

func (h *Handler) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.Header.Get("X-Forwarded-For")
		if clientID == "" {
			clientID = r.RemoteAddr
		}

		if !h.rateLimiter.Allow(clientID) {
			writeError(w, http.StatusTooManyRequests, "RATE_LIMIT", "请求过于频繁，请稍后再试")
			return
		}

		next.ServeHTTP(w, r)
	})
}
