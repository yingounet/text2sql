package text2sql

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"text2sql/internal/llm"
)

// Service Text2SQL 核心服务
type Service struct {
	llm         llm.Provider
	validator   *SQLValidator
	maxRetries  int
	contextStore ContextStore
}

// NewService 创建 Text2SQL 服务
func NewService(llmProvider llm.Provider, validator *SQLValidator, maxRetries int) *Service {
	if validator == nil {
		validator = &SQLValidator{}
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	return &Service{
		llm:          llmProvider,
		validator:    validator,
		maxRetries:   maxRetries,
		contextStore: NewMemoryContextStore(),
	}
}

// NewServiceWithContextStore 创建带自定义上下文存储的 Text2SQL 服务
func NewServiceWithContextStore(llmProvider llm.Provider, validator *SQLValidator, maxRetries int, store ContextStore) *Service {
	if validator == nil {
		validator = &SQLValidator{}
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	if store == nil {
		store = NewMemoryContextStore()
	}
	return &Service{
		llm:          llmProvider,
		validator:    validator,
		maxRetries:   maxRetries,
		contextStore: store,
	}
}

// GenerateRequest 生成请求
// 多轮对话时，提供有效的 conversation_id 时 schema 和 database 可选，从上下文复用
type GenerateRequest struct {
	Query          string   `json:"query" validate:"required"`
	Schema         Schema   `json:"schema,omitempty"`           // 可选：续会话时可省略，从上下文读取
	Database       Database `json:"database,omitempty"`         // 可选：续会话时可省略，从上下文读取
	ConversationID string   `json:"conversation_id,omitempty"`  // 可选：会话ID，用于关联上下文
	PreviousSQL    string   `json:"previous_sql,omitempty"`     // 可选：上一轮SQL，用于追加修改
}

// Schema 表结构
type Schema struct {
	Tables []Table `json:"tables" validate:"omitempty,dive"`
}

// Table 表定义
type Table struct {
	Name    string   `json:"name" validate:"required"`
	Columns []Column `json:"columns" validate:"required,dive"`
}

// Column 列定义
type Column struct {
	Name    string `json:"name" validate:"required"`
	Type    string `json:"type"`
	Comment string `json:"comment"`
}

// Database 目标数据库信息
type Database struct {
	Type    string `json:"type" validate:"omitempty,oneof=mysql postgresql sqlite redis"`
	Version string `json:"version"`
}

// GenerateResponse 生成响应
type GenerateResponse struct {
	SQL            string `json:"sql"`
	Explanation    string `json:"explanation"`
	ConversationID string `json:"conversation_id"` // 会话ID，供后续请求使用
}

// Generate 根据自然语言和表结构生成 SQL
func (s *Service) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var conversationID string
	var previousSQL string
	var convCtx *ConversationContext
	var schema Schema
	var database Database

	// 1. 处理会话ID和上下文加载
	if req.ConversationID != "" {
		// 尝试加载历史上下文
		loadedCtx, err := s.contextStore.Get(req.ConversationID)
		if err == nil {
			convCtx = loadedCtx
			conversationID = req.ConversationID
			// 续会话：schema/database 可从上下文复用
			if len(req.Schema.Tables) > 0 {
				schema = req.Schema
				if !s.schemaEqual(convCtx.Schema, req.Schema) {
					return nil, fmt.Errorf("%w: schema 与历史会话不一致", ErrSchemaMismatch)
				}
			} else {
				schema = convCtx.Schema
			}
			if req.Database.Type != "" {
				database = req.Database
				if convCtx.Database.Type != req.Database.Type || convCtx.Database.Version != req.Database.Version {
					return nil, fmt.Errorf("%w: database 与历史会话不一致", ErrDatabaseMismatch)
				}
			} else {
				database = convCtx.Database
			}
			// 如果历史中有SQL，使用最新的SQL作为previous_sql
			if len(convCtx.History) > 0 && req.PreviousSQL == "" {
				previousSQL = convCtx.History[len(convCtx.History)-1].SQL
			}
		} else if err == ErrConversationNotFound {
			// 会话不存在，按新会话处理，需要 schema 和 database
			conversationID = generateConversationID()
			if len(req.Schema.Tables) == 0 {
				return nil, fmt.Errorf("%w: conversation_id 无效或已过期，请提供 schema", ErrSchemaRequired)
			}
			if req.Database.Type == "" {
				return nil, fmt.Errorf("%w: conversation_id 无效或已过期，请提供 database", ErrDatabaseRequired)
			}
			schema = req.Schema
			database = req.Database
		} else {
			return nil, fmt.Errorf("加载上下文失败: %w", err)
		}
	} else {
		// 创建新会话，schema 和 database 必填
		conversationID = generateConversationID()
		if len(req.Schema.Tables) == 0 {
			return nil, fmt.Errorf("%w: 新会话需提供 schema", ErrSchemaRequired)
		}
		if req.Database.Type == "" {
			return nil, fmt.Errorf("%w: 新会话需提供 database", ErrDatabaseRequired)
		}
		schema = req.Schema
		database = req.Database
	}

	// 2. 如果直接提供了 previous_sql，优先使用
	if req.PreviousSQL != "" {
		previousSQL = req.PreviousSQL
	}

	// 3. 构建 Prompt
	var systemPrompt string
	var userContent string
	if database.Type == "redis" {
		if previousSQL != "" {
			systemPrompt = buildSystemPromptForModifyRedis(database.Version)
			userContent = buildUserContentForModify(req.Query, schema, previousSQL)
		} else {
			systemPrompt = buildSystemPromptRedis(database.Version)
			userContent = buildUserContent(req.Query, schema)
		}
	} else {
		if previousSQL != "" {
			systemPrompt = buildSystemPromptForModify(database.Type, database.Version)
			userContent = buildUserContentForModify(req.Query, schema, previousSQL)
		} else {
			systemPrompt = buildSystemPrompt(database.Type, database.Version)
			userContent = buildUserContent(req.Query, schema)
		}
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	// 4. 如果有历史上下文，添加历史对话（可选，用于更好的上下文理解）
	if convCtx != nil && len(convCtx.History) > 0 && previousSQL == "" {
		// 添加最近几轮历史对话（最多3轮）
		startIdx := len(convCtx.History) - 3
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(convCtx.History); i++ {
			turn := convCtx.History[i]
			messages = append(messages,
				llm.Message{Role: "user", Content: turn.Query},
				llm.Message{Role: "assistant", Content: fmt.Sprintf("SQL: %s\n解释: %s", turn.SQL, turn.Explanation)},
			)
		}
		// 添加当前查询
		messages = append(messages, llm.Message{Role: "user", Content: req.Query})
	}

	// 5. 调用 LLM 生成 SQL
	var lastValidationErr error
	var sql, explanation string
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		resp, err := s.llm.Complete(ctx, &llm.CompleteRequest{
			Model:       "",
			Messages:    messages,
			MaxTokens:   2048,
			Temperature: 0.1,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: llm complete: %w", ErrLLMError, err)
		}

		if database.Type == "redis" {
			sql, explanation = parseLLMOutputRedis(resp.Content)
		} else {
			sql, explanation = parseLLMOutput(resp.Content)
		}

		// 校验（SQL 或 Redis 命令）
		if err := s.validator.Validate(sql, database.Type, database.Version); err != nil {
			lastValidationErr = err
			if attempt < s.maxRetries-1 {
				msg := "生成的 SQL 校验失败：%s\n请修正并重新生成。"
				if database.Type == "redis" {
					msg = "生成的 Redis 命令校验失败：%s\n请修正并重新生成。"
				}
				messages = append(messages,
					llm.Message{Role: "assistant", Content: resp.Content},
					llm.Message{Role: "user", Content: fmt.Sprintf(msg, err.Error())},
				)
				continue
			}
			return nil, fmt.Errorf("%w: %v", ErrSQLValidation, err)
		}

		break
	}

	if sql == "" {
		return nil, fmt.Errorf("%w: %v", ErrSQLValidation, lastValidationErr)
	}

	// 6. 保存上下文
	if convCtx == nil {
		convCtx = &ConversationContext{
			ConversationID: conversationID,
			Schema:         schema,
			Database:       database,
			History:        []ConversationTurn{},
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
	}
	convCtx.History = append(convCtx.History, ConversationTurn{
		Query:       req.Query,
		SQL:         sql,
		Explanation: explanation,
		Timestamp:   time.Now(),
	})
	if err := s.contextStore.Save(convCtx); err != nil {
		// 保存失败不影响返回结果，只记录错误
		// 注意：在生产环境中应使用结构化日志库
		log.Printf("保存上下文失败: conversation_id=%s, error=%v", conversationID, err)
	}

	return &GenerateResponse{
		SQL:            sql,
		Explanation:    explanation,
		ConversationID: conversationID,
	}, nil
}

// generateConversationID 生成会话ID
func generateConversationID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备方案
		// 注意：这不是最佳实践，但在错误情况下提供降级方案
		b = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
		if len(b) > 16 {
			b = b[:16]
		}
	}
	return "conv_" + base64.URLEncoding.EncodeToString(b)[:22] // conv_ + 22字符 = 27字符
}

// schemaEqual 比较两个 Schema 是否相等
func (s *Service) schemaEqual(s1, s2 Schema) bool {
	if len(s1.Tables) != len(s2.Tables) {
		return false
	}
	// 简单比较：表名和列数相同即可（实际可以更严格）
	for i, t1 := range s1.Tables {
		if i >= len(s2.Tables) {
			return false
		}
		t2 := s2.Tables[i]
		if t1.Name != t2.Name || len(t1.Columns) != len(t2.Columns) {
			return false
		}
	}
	return true
}

// buildSystemPrompt 构建 system prompt
func buildSystemPrompt(dbType, version string) string {
	v := ""
	if version != "" {
		v = fmt.Sprintf("（版本 %s）", version)
	}
	return fmt.Sprintf(`你是一个专业的 SQL 专家。根据用户提供的数据库表结构和自然语言问题，生成对应的 %s%s SQL 查询语句。

规则：
1. 只生成 SELECT 查询，不要生成 INSERT/UPDATE/DELETE/DROP 等修改语句
2. SQL 必须符合 %s 语法
3. 表名和列名使用 schema 中提供的名称
4. 输出格式：第一行是 SQL 语句，第二行以"解释："开头是简要说明（可选）`, dbType, v, dbType)
}

// buildSystemPromptForModify 构建追加修改模式的 system prompt
func buildSystemPromptForModify(dbType, version string) string {
	v := ""
	if version != "" {
		v = fmt.Sprintf("（版本 %s）", version)
	}
	return fmt.Sprintf(`你是一个专业的 SQL 专家。用户会提供现有的 SQL 语句和新的需求，你需要在现有 SQL 基础上进行修改。

规则：
1. 理解现有 SQL 的意图
2. 根据新需求，在现有 SQL 基础上追加或修改条件
3. 保持 SQL 的完整性和正确性
4. 只生成 SELECT 查询，不要生成 INSERT/UPDATE/DELETE/DROP 等修改语句
5. SQL 必须符合 %s%s 语法
6. 表名和列名使用 schema 中提供的名称
7. 输出格式：第一行是修改后的完整 SQL 语句，第二行以"解释："开头是简要说明（可选）`, dbType, v)
}

// buildSystemPromptRedis 构建 Redis 的 system prompt（只读命令）
func buildSystemPromptRedis(version string) string {
	v := ""
	if version != "" {
		v = fmt.Sprintf("（版本 %s）", version)
	}
	return fmt.Sprintf(`你是一个 Redis 专家。根据用户提供的结构描述（表名表示 key 模式或结构名，列表示 hash 的 field 等）和自然语言问题，生成对应的只读 Redis 命令。%s

规则：
1. 只生成只读命令，如 GET、HGET、HGETALL、LRANGE、SMEMBERS、ZRANGE、SCAN、HSCAN、KEYS、EXISTS、TYPE、TTL 等
2. 禁止 FLUSHALL、DEL、SET、HSET、LPUSH、SADD、ZADD 等任何写操作
3. 在大键空间场景下优先使用 SCAN/HSCAN 等迭代命令，慎用 KEYS
4. 表名/列名对应 schema 中的 key 模式或 hash field，请据此生成正确的 key 和 field 名
5. 输出格式：第一行开始是 Redis 命令（可多行、多条命令），之后空行可选，再以"解释："开头是简要说明（可选）`, v)
}

// buildSystemPromptForModifyRedis 构建 Redis 追加修改模式的 system prompt
func buildSystemPromptForModifyRedis(version string) string {
	v := ""
	if version != "" {
		v = fmt.Sprintf("（版本 %s）", version)
	}
	return fmt.Sprintf(`你是一个 Redis 专家。用户会提供现有的 Redis 命令和新的需求，你需要在现有命令基础上进行修改或补充。%s

规则：
1. 理解现有 Redis 命令的意图
2. 根据新需求，在现有命令基础上追加或修改，只输出只读命令（GET、HGET、LRANGE、SCAN 等）
3. 禁止任何写操作（SET、HSET、DEL、FLUSHALL 等）
4. 输出格式：第一行开始是修改后的完整 Redis 命令（可多行），之后以"解释："开头是简要说明（可选）`, v)
}

// buildUserContent 构建 user 消息内容
func buildUserContent(query string, schema Schema) string {
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	return fmt.Sprintf("表结构：\n%s\n\n用户问题：%s", string(schemaJSON), query)
}

// buildUserContentForModify 构建追加修改模式的 user 消息内容
func buildUserContentForModify(query string, schema Schema, previousSQL string) string {
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	return fmt.Sprintf(`现有 SQL：
%s

表结构：
%s

新的需求：%s

请基于现有 SQL，根据新需求进行修改。`, previousSQL, string(schemaJSON), query)
}

// parseLLMOutput 解析 LLM 输出，提取 SQL 和解释
func parseLLMOutput(content string) (sql, explanation string) {
	lines := splitLines(content)
	var sqlLines []string
	inSQL := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inSQL {
			if trimmed == "```" || (len(trimmed) >= 3 && trimmed[:3] == "```") {
				break
			}
			sqlLines = append(sqlLines, trimmed)
			continue
		}
		if len(trimmed) >= 6 && (strings.HasPrefix(trimmed, "```sql") || strings.HasPrefix(trimmed, "```SQL")) {
			inSQL = true
			rest := trimmed[6:]
			rest = strings.TrimPrefix(rest, "\n")
			rest = strings.TrimSpace(rest)
			if rest != "" && rest != "`" {
				sqlLines = append(sqlLines, rest)
			}
			continue
		}
		// 非代码块：第一行以 SELECT 开头，收集到空行或 解释/说明
		if len(sqlLines) == 0 && trimmed != "" && strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") {
			sqlLines = append(sqlLines, trimmed)
			continue
		}
		if len(sqlLines) > 0 && !inSQL {
			if trimmed == "" || startsWithIgnoreCase(trimmed, "解释") || startsWithIgnoreCase(trimmed, "说明") {
				break
			}
			sqlLines = append(sqlLines, trimmed)
		}
	}
	if len(sqlLines) > 0 {
		sql = strings.TrimSpace(joinLines(sqlLines))
		sql = strings.TrimSuffix(sql, ";")
		sql = strings.TrimSpace(sql)
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if startsWithIgnoreCase(trimmed, "解释：") || startsWithIgnoreCase(trimmed, "说明：") {
			explanation = trimPrefix(trimmed, "解释：", "说明：")
			break
		}
	}
	return sql, explanation
}

// redisCommandPrefixes Redis 只读命令前缀，用于识别 LLM 输出中的命令行
var redisCommandPrefixes = []string{
	"GET ", "MGET ", "HGET ", "HGETALL ", "HMGET ", "LRANGE ", "LINDEX ", "LLEN ",
	"SMEMBERS ", "SISMEMBER ", "SCARD ", "ZRANGE ", "ZREVRANGE ", "ZRANGEBYSCORE ", "ZREVRANGEBYSCORE ",
	"ZRANK ", "ZREVRANK ", "ZSCORE ", "ZCARD ", "KEYS ", "SCAN ", "HSCAN ", "SSCAN ", "ZSCAN ",
	"EXISTS ", "TYPE ", "TTL ", "PTTL ", "STRLEN ", "HLEN ",
}

func isRedisCommandLine(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	for _, p := range redisCommandPrefixes {
		if strings.HasPrefix(upper, p) || upper == strings.TrimSpace(p) {
			return true
		}
	}
	// GET key 可能没有空格（GET 后直接换行少见，但 GET key 常见）
	if strings.HasPrefix(upper, "GET ") || upper == "GET" {
		return true
	}
	return false
}

// parseLLMOutputRedis 解析 LLM 输出，提取 Redis 命令和解释
func parseLLMOutputRedis(content string) (commands, explanation string) {
	lines := splitLines(content)
	var cmdLines []string
	inBlock := false
	blockEndMarker := "```"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inBlock {
			if trimmed == blockEndMarker || (len(trimmed) >= 3 && trimmed[:3] == "```") {
				break
			}
			cmdLines = append(cmdLines, trimmed)
			continue
		}
		if len(trimmed) >= 7 && (strings.HasPrefix(trimmed, "```redis") || strings.HasPrefix(trimmed, "```Redis")) {
			inBlock = true
			rest := trimmed[7:]
			rest = strings.TrimPrefix(rest, "\n")
			rest = strings.TrimSpace(rest)
			if rest != "" && rest != "`" {
				cmdLines = append(cmdLines, rest)
			}
			continue
		}
		if len(trimmed) >= 3 && strings.HasPrefix(trimmed, "```") {
			inBlock = true
			rest := trimmed[3:]
			rest = strings.TrimPrefix(rest, "\n")
			rest = strings.TrimSpace(rest)
			if rest != "" && rest != "`" && isRedisCommandLine(rest) {
				cmdLines = append(cmdLines, rest)
			}
			continue
		}
		if len(cmdLines) == 0 && trimmed != "" && isRedisCommandLine(trimmed) {
			cmdLines = append(cmdLines, trimmed)
			continue
		}
		if len(cmdLines) > 0 && !inBlock {
			if trimmed == "" || startsWithIgnoreCase(trimmed, "解释") || startsWithIgnoreCase(trimmed, "说明") {
				break
			}
			if isRedisCommandLine(trimmed) {
				cmdLines = append(cmdLines, trimmed)
			}
		}
	}
	if len(cmdLines) > 0 {
		commands = strings.TrimSpace(joinLines(cmdLines))
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if startsWithIgnoreCase(trimmed, "解释：") || startsWithIgnoreCase(trimmed, "说明：") {
			explanation = trimPrefix(trimmed, "解释：", "说明：")
			break
		}
	}
	return commands, explanation
}

func splitLines(s string) []string {
	var lines []string
	var b []rune
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, string(b))
			b = nil
		} else {
			b = append(b, r)
		}
	}
	if len(b) > 0 {
		lines = append(lines, string(b))
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}

func startsWithIgnoreCase(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c := s[i]
		p := prefix[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if p >= 'A' && p <= 'Z' {
			p += 32
		}
		if c != p {
			return false
		}
	}
	return true
}

func trimPrefix(s, p1, p2 string) string {
	if startsWithIgnoreCase(s, p1) {
		return s[len(p1):]
	}
	if startsWithIgnoreCase(s, p2) {
		return s[len(p2):]
	}
	return s
}
