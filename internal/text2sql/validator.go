package text2sql

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/xwb1989/sqlparser"
)

// SQLValidator SQL 校验器
type SQLValidator struct{}

// NewSQLValidator 创建 SQLValidator
func NewSQLValidator() *SQLValidator {
	return &SQLValidator{}
}

// Validate 按数据库类型和版本校验 SQL
func (v *SQLValidator) Validate(sql, dbType, version string) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return fmt.Errorf("SQL 不能为空")
	}

	switch dbType {
	case "mysql":
		return v.validateMySQL(sql, version)
	case "postgresql", "postgres":
		return v.validatePostgreSQL(sql, version)
	case "sqlite":
		return v.validateSQLite(sql, version)
	case "redis":
		return v.validateRedis(sql, version)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

var errNotReadOnly = errors.New("仅允许只读查询")

// validateMySQL 使用 MySQL 方言解析
func (v *SQLValidator) validateMySQL(sql, _ string) error {
	if err := v.ensureReadOnlySQL(sql); err != nil {
		if errors.Is(err, errNotReadOnly) {
			return err
		}
		return fmt.Errorf("MySQL 语法错误: %w", err)
	}
	return nil
}

// validatePostgreSQL 基础校验（纯 Go 的 PG 解析器较少，先做基本校验）
func (v *SQLValidator) validatePostgreSQL(sql, _ string) error {
	// 尝试用 MySQL parser 解析（SELECT 语法多数兼容）
	if err := v.ensureReadOnlySQL(sql); err != nil {
		if errors.Is(err, errNotReadOnly) {
			return err
		}
		// PG 特有语法可能失败，做基本结构校验
		if v.basicSelectCheck(sql) && !hasMultipleStatements(sql) {
			return nil
		}
		return fmt.Errorf("PostgreSQL 语法可能存在问题: %w", err)
	}
	return nil
}

// validateSQLite 基础校验
func (v *SQLValidator) validateSQLite(sql, _ string) error {
	if err := v.ensureReadOnlySQL(sql); err != nil {
		if errors.Is(err, errNotReadOnly) {
			return err
		}
		if v.basicSelectCheck(sql) && !hasMultipleStatements(sql) {
			return nil
		}
		return fmt.Errorf("SQLite 语法可能存在问题: %w", err)
	}
	return nil
}

// basicSelectCheck 基本 SELECT 结构校验
func (v *SQLValidator) basicSelectCheck(sql string) bool {
	upper := strings.TrimSpace(strings.ToUpper(sql))
	if !strings.HasPrefix(upper, "SELECT") {
		return false
	}
	// 简单正则：至少包含 SELECT ... FROM
	re := regexp.MustCompile(`(?i)\bSELECT\b.*\bFROM\b`)
	return re.MatchString(sql)
}

func (v *SQLValidator) ensureReadOnlySQL(sql string) error {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return err
	}
	if !isReadOnlyStatement(stmt) {
		return errNotReadOnly
	}
	return nil
}

func isReadOnlyStatement(stmt sqlparser.Statement) bool {
	switch stmt.(type) {
	case *sqlparser.Select, *sqlparser.Union, *sqlparser.ParenSelect, *sqlparser.With:
		return true
	default:
		return false
	}
}

func hasMultipleStatements(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	if !strings.Contains(trimmed, ";") {
		return false
	}
	if strings.HasSuffix(trimmed, ";") && strings.Count(trimmed, ";") == 1 {
		return false
	}
	return true
}

// Redis 只读命令白名单（仅允许查询类命令）
var redisReadOnlyCommands = map[string]bool{
	"GET": true, "MGET": true, "HGET": true, "HGETALL": true, "HMGET": true,
	"LRANGE": true, "LINDEX": true, "LLEN": true,
	"SMEMBERS": true, "SISMEMBER": true, "SCARD": true,
	"ZRANGE": true, "ZREVRANGE": true, "ZRANGEBYSCORE": true, "ZREVRANGEBYSCORE": true,
	"ZRANK": true, "ZREVRANK": true, "ZSCORE": true, "ZCARD": true,
	"KEYS": true, "SCAN": true, "HSCAN": true, "SSCAN": true, "ZSCAN": true,
	"EXISTS": true, "TYPE": true, "TTL": true, "PTTL": true,
	"STRLEN": true, "HLEN": true,
}

// Redis 危险/写命令黑名单
var redisDangerousCommands = map[string]bool{
	"FLUSHALL": true, "FLUSHDB": true, "DEL": true, "UNLINK": true,
	"SET": true, "SETEX": true, "SETNX": true, "MSET": true,
	"HSET": true, "HSETNX": true, "HMSET": true, "HDEL": true,
	"LPUSH": true, "RPUSH": true, "LPOP": true, "RPOP": true, "LREM": true, "LSET": true, "LTRIM": true,
	"SADD": true, "SREM": true, "SPOP": true,
	"ZADD": true, "ZREM": true, "ZINCRBY": true,
	"INCR": true, "INCRBY": true, "DECR": true, "DECRBY": true,
	"EXPIRE": true, "PEXPIRE": true, "PERSIST": true,
	"RENAME": true, "RENAMENX": true,
}

// validateRedis 校验 Redis 命令：仅允许只读命令
func (v *SQLValidator) validateRedis(commands, _ string) error {
	lines := strings.Split(commands, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 取第一个 token 作为命令（Redis 命令大小写不敏感，统一转大写比较）
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		cmd := strings.ToUpper(parts[0])
		if redisDangerousCommands[cmd] {
			return fmt.Errorf("不允许的 Redis 写操作: %s", cmd)
		}
		if !redisReadOnlyCommands[cmd] {
			return fmt.Errorf("不支持的 Redis 命令或非只读命令: %s（仅允许只读命令如 GET、HGET、LRANGE、SCAN 等）", cmd)
		}
	}
	return nil
}
