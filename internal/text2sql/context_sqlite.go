package text2sql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	sqliteCleanupInterval = 1 * time.Hour
	sqliteMaxAge          = 24 * time.Hour
)

// SQLiteContextStore SQLite 持久化上下文存储
type SQLiteContextStore struct {
	db    *sql.DB
	mu    sync.RWMutex
	stopCh chan struct{}
}

// NewSQLiteContextStore 创建 SQLite 上下文存储
func NewSQLiteContextStore(dsn string) (*SQLiteContextStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// 配置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	store := &SQLiteContextStore{
		db:    db,
		stopCh: make(chan struct{}),
	}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	go store.startCleanupTask()
	return store, nil
}

// initSchema 初始化数据库表
func (s *SQLiteContextStore) initSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			schema_json TEXT NOT NULL,
			database_type TEXT NOT NULL,
			database_version TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS conversation_turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT NOT NULL,
			query TEXT NOT NULL,
			sql TEXT NOT NULL,
			explanation TEXT,
			turn_number INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_conversation_turns_conv ON conversation_turns(conversation_id);
		CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at);
	`)
	return err
}

// Get 获取上下文
func (s *SQLiteContextStore) Get(conversationID string) (*ConversationContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var schemaJSON, dbType, dbVersion string
	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(`
		SELECT schema_json, database_type, database_version, created_at, updated_at
		FROM conversations WHERE id = ?
	`, conversationID).Scan(&schemaJSON, &dbType, &dbVersion, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT query, sql, explanation, created_at
		FROM conversation_turns
		WHERE conversation_id = ?
		ORDER BY turn_number ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ConversationTurn
	for rows.Next() {
		var q, sqlStr, expl string
		var ts time.Time
		if err := rows.Scan(&q, &sqlStr, &expl, &ts); err != nil {
			return nil, err
		}
		history = append(history, ConversationTurn{
			Query:       q,
			SQL:         sqlStr,
			Explanation: expl,
			Timestamp:   ts,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ConversationContext{
		ConversationID: conversationID,
		Schema:         schema,
		Database:       Database{Type: dbType, Version: dbVersion},
		History:        history,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

// Save 保存上下文
func (s *SQLiteContextStore) Save(ctx *ConversationContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schemaJSON, err := json.Marshal(ctx.Schema)
	if err != nil {
		return err
	}

	ctx.UpdatedAt = time.Now()
	now := ctx.UpdatedAt

	// 使用事务确保数据一致性
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO conversations (id, schema_json, database_type, database_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			schema_json = excluded.schema_json,
			database_type = excluded.database_type,
			database_version = excluded.database_version,
			updated_at = excluded.updated_at
	`,
		ctx.ConversationID,
		string(schemaJSON),
		ctx.Database.Type,
		ctx.Database.Version,
		ctx.CreatedAt,
		now,
	)
	if err != nil {
		return err
	}

	turnNum := len(ctx.History)
	if turnNum > 0 {
		turn := ctx.History[turnNum-1]
		_, err = tx.Exec(`
			INSERT INTO conversation_turns (conversation_id, query, sql, explanation, turn_number, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, ctx.ConversationID, turn.Query, turn.SQL, turn.Explanation, turnNum-1, turn.Timestamp)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete 删除上下文
func (s *SQLiteContextStore) Delete(conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM conversations WHERE id = ?`, conversationID)
	return err
}

// Cleanup 清理过期上下文
func (s *SQLiteContextStore) Cleanup(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hours := int64(maxAge / time.Hour)
	if hours < 1 {
		hours = 1
	}
	modifier := fmt.Sprintf("-%d hours", hours)
	_, err := s.db.Exec(`
		DELETE FROM conversations
		WHERE datetime(updated_at) < datetime('now', ?)
	`, modifier)
	return err
}

// startCleanupTask 启动后台清理任务
func (s *SQLiteContextStore) startCleanupTask() {
	ticker := time.NewTicker(sqliteCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			_ = s.Cleanup(sqliteMaxAge)
		}
	}
}

// Close 关闭数据库连接
func (s *SQLiteContextStore) Close() error {
	close(s.stopCh)
	return s.db.Close()
}
