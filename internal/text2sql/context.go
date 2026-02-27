package text2sql

import (
	"sync"
	"time"
)

// ConversationContext 会话上下文
type ConversationContext struct {
	ConversationID string
	Schema         Schema
	Database       Database
	History        []ConversationTurn
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ConversationTurn 对话轮次
type ConversationTurn struct {
	Query       string
	SQL         string
	Explanation string
	Timestamp   time.Time
}

// ContextStore 上下文存储接口
type ContextStore interface {
	Get(conversationID string) (*ConversationContext, error)
	Save(ctx *ConversationContext) error
	Delete(conversationID string) error
	Cleanup(maxAge time.Duration) error
	Close() error // 关闭存储，释放资源
}

// MemoryContextStore 内存上下文存储
type MemoryContextStore struct {
	mu     sync.RWMutex
	store  map[string]*ConversationContext
	stopCh chan struct{}
}

// NewMemoryContextStore 创建内存上下文存储
func NewMemoryContextStore() *MemoryContextStore {
	store := &MemoryContextStore{
		store:  make(map[string]*ConversationContext),
		stopCh: make(chan struct{}),
	}
	// 启动后台清理任务
	go store.startCleanupTask()
	return store
}

// Get 获取上下文
func (m *MemoryContextStore) Get(conversationID string) (*ConversationContext, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ctx, ok := m.store[conversationID]
	if !ok {
		return nil, ErrConversationNotFound
	}
	return ctx, nil
}

// Save 保存上下文
func (m *MemoryContextStore) Save(ctx *ConversationContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctx.UpdatedAt = time.Now()
	m.store[ctx.ConversationID] = ctx
	return nil
}

// Delete 删除上下文
func (m *MemoryContextStore) Delete(conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, conversationID)
	return nil
}

// Cleanup 清理过期上下文
func (m *MemoryContextStore) Cleanup(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, ctx := range m.store {
		if now.Sub(ctx.UpdatedAt) > maxAge {
			delete(m.store, id)
		}
	}
	return nil
}

// startCleanupTask 启动清理任务
func (m *MemoryContextStore) startCleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			_ = m.Cleanup(24 * time.Hour) // 清理24小时未更新的会话
		}
	}
}

// Close 关闭存储，停止清理任务
func (m *MemoryContextStore) Close() error {
	close(m.stopCh)
	return nil
}
