# API 文档

## 概述

Text2SQL API 提供 RESTful 接口，用于将自然语言转换为 SQL 查询语句或 Redis 只读命令（当 `database.type` 为 `redis` 时）。

**Base URL**: `http://localhost:8080/api/v1`

**认证方式**: API Key（通过 Header 传递）

## 认证

所有需要认证的接口都需要在请求头中携带 API Key：

```
Authorization: Bearer <api_key>
```

或

```
X-API-Key: <api_key>
```

## 接口列表

### 1. 健康检查

检查服务是否正常运行。

**接口**: `GET /api/v1/health`

**认证**: 不需要

**请求示例**:

```bash
curl http://localhost:8080/api/v1/health
```

**响应示例**:

```json
{
  "status": "ok"
}
```

**状态码**:
- `200 OK`: 服务正常

---

### 2. 生成 SQL

根据自然语言查询和表结构生成 SQL 语句。

**接口**: `POST /api/v1/sql/generate`

**认证**: 需要

**请求体**:

```json
{
  "query": "查询所有年龄大于30的用户",
  "schema": {
    "tables": [
      {
        "name": "users",
        "columns": [
          {
            "name": "id",
            "type": "int",
            "comment": "用户ID"
          },
          {
            "name": "name",
            "type": "varchar(100)",
            "comment": "用户名"
          },
          {
            "name": "age",
            "type": "int",
            "comment": "年龄"
          }
        ]
      }
    ]
  },
  "database": {
    "type": "mysql",
    "version": "8.0"
  },
  "conversation_id": "conv_xxx",
  "previous_sql": "SELECT * FROM users WHERE age > 30"
}
```

**请求字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | string | 是 | 用户自然语言查询意图 |
| `schema` | object | 条件 | 数据库表结构。新会话必填；续会话（提供有效的 conversation_id）时可省略，从上下文复用 |
| `schema.tables` | array | 条件 | 表结构列表。同上 |
| `schema.tables[].name` | string | 是 | 表名 |
| `schema.tables[].columns` | array | 是 | 列定义列表 |
| `schema.tables[].columns[].name` | string | 是 | 列名 |
| `schema.tables[].columns[].type` | string | 否 | 列类型（如 `int`、`varchar(100)`） |
| `schema.tables[].columns[].comment` | string | 否 | 列注释 |
| `database` | object | 条件 | 目标数据库信息。新会话必填；续会话时可省略，从上下文复用 |
| `database.type` | string | 条件 | 数据库类型：`mysql` / `postgresql` / `sqlite` / `redis`。同上 |
| `database.version` | string | 否 | 数据库版本，如 `8.0`、`14`、`3` |
| `conversation_id` | string | 否 | 会话ID，用于关联多轮对话上下文 |
| `previous_sql` | string | 否 | 上一轮的SQL语句，用于在现有SQL基础上修改 |

**响应示例**:

```json
{
  "sql": "SELECT * FROM users WHERE age > 30",
  "explanation": "筛选年龄大于30的用户",
  "conversation_id": "conv_a1b2c3d4e5f6g7h8i9j0k1l2"
}
```

当 `database.type` 为 `redis` 时，请求可用 schema 描述 key 结构（表名表示 key 模式或结构名，列表示 hash 的 field），响应中 `sql` 为 Redis 只读命令，例如：

```json
{
  "query": "获取 user:1001 的 name 字段",
  "schema": {
    "tables": [
      {
        "name": "user:*",
        "columns": [
          {"name": "name", "type": "string", "comment": "用户名"},
          {"name": "age", "type": "string", "comment": "年龄"}
        ]
      }
    ]
  },
  "database": { "type": "redis", "version": "7.0" }
}
```

响应示例：

```json
{
  "sql": "HGET user:1001 name",
  "explanation": "获取 key user:1001 的 hash 字段 name",
  "conversation_id": "conv_xxx"
}
```

**响应字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `sql` | string | 生成的语句：当 `database.type` 为 `mysql`/`postgresql`/`sqlite` 时为 SQL；为 `redis` 时为 Redis 只读命令（可多行） |
| `explanation` | string | 语句的简要说明 |
| `conversation_id` | string | 会话ID，供后续请求使用 |

**状态码**:

- `200 OK`: 成功生成 SQL
- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: API Key 无效
- `404 Not Found`: conversation_id 不存在或已过期
- `500 Internal Server Error`: 服务器内部错误

**错误响应示例**:

```json
{
  "code": "INVALID_REQUEST",
  "message": "query 不能为空"
}
```

## 多轮对话

### 使用 conversation_id

第一轮请求后，系统会返回 `conversation_id`，后续请求携带此 ID 即可关联上下文：

**第一轮请求**:

```bash
curl -X POST http://localhost:8080/api/v1/sql/generate \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "查询所有年龄大于30的用户",
    "schema": {
      "tables": [
        {
          "name": "users",
          "columns": [
            {"name": "id", "type": "int"},
            {"name": "name", "type": "varchar(100)"},
            {"name": "age", "type": "int"}
          ]
        }
      ]
    },
    "database": {"type": "mysql", "version": "8.0"}
  }'
```

**响应**:

```json
{
  "sql": "SELECT * FROM users WHERE age > 30",
  "explanation": "筛选年龄大于30的用户",
  "conversation_id": "conv_a1b2c3d4e5f6g7h8i9j0k1l2"
}
```

**第二轮请求（追加修改）**:

```bash
curl -X POST http://localhost:8080/api/v1/sql/generate \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "还要筛选状态为active的用户",
    "schema": {
      "tables": [
        {
          "name": "users",
          "columns": [
            {"name": "id", "type": "int"},
            {"name": "name", "type": "varchar(100)"},
            {"name": "age", "type": "int"},
            {"name": "status", "type": "varchar(20)"}
          ]
        }
      ]
    },
    "database": {"type": "mysql", "version": "8.0"},
    "conversation_id": "conv_a1b2c3d4e5f6g7h8i9j0k1l2"
  }'
```

**响应**:

```json
{
  "sql": "SELECT * FROM users WHERE age > 30 AND status = '\''active'\''",
  "explanation": "在原有查询基础上增加了状态筛选条件",
  "conversation_id": "conv_a1b2c3d4e5f6g7h8i9j0k1l2"
}
```

### 使用 previous_sql

如果不想使用会话ID，可以直接提供上一轮的 SQL：

```bash
curl -X POST http://localhost:8080/api/v1/sql/generate \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "还要筛选状态为active的用户",
    "schema": {...},
    "database": {"type": "mysql", "version": "8.0"},
    "previous_sql": "SELECT * FROM users WHERE age > 30"
  }'
```

## 错误码

| 错误码 | HTTP状态码 | 说明 |
|--------|-----------|------|
| `UNAUTHORIZED` | 401 | API Key 缺失或无效 |
| `INVALID_REQUEST` | 400 | 请求参数错误（如 query 为空、schema 格式错误等） |
| `INVALID_SCHEMA` | 400 | Schema 格式错误（如新会话未提供 tables） |
| `INVALID_DATABASE` | 400 | Database 格式错误（如新会话未提供 type） |
| `SCHEMA_REQUIRED` | 400 | 新会话或 conversation_id 无效时需提供 schema |
| `DATABASE_REQUIRED` | 400 | 新会话或 conversation_id 无效时需提供 database |
| `SQL_VALIDATION_FAILED` | 400 | 生成的 SQL 校验失败 |
| `CONVERSATION_NOT_FOUND` | 404 | conversation_id 不存在或已过期 |
| `SCHEMA_MISMATCH` | 400 | schema 与历史会话不一致 |
| `DATABASE_MISMATCH` | 400 | database 与历史会话不一致 |
| `LLM_ERROR` | 500 | LLM 调用失败 |

## 注意事项

1. **Content-Type**: 请求头必须设置为 `application/json`
2. **Schema/Database 可选**: 续会话时，`schema` 和 `database` 可省略，从上下文复用；若显式传入则需与历史一致
4. **会话过期**: 会话在 24 小时未使用后会自动清理
5. **优先级**: 如果同时提供 `conversation_id` 和 `previous_sql`，系统会优先使用 `previous_sql`
6. **SQL 安全**: 系统只允许生成 SELECT 查询，会自动拦截 DROP、DELETE、UPDATE 等危险操作

## 示例代码

### Python

```python
import requests

url = "http://localhost:8080/api/v1/sql/generate"
headers = {
    "Authorization": "Bearer your-api-key",
    "Content-Type": "application/json"
}
data = {
    "query": "查询所有年龄大于30的用户",
    "schema": {
        "tables": [
            {
                "name": "users",
                "columns": [
                    {"name": "id", "type": "int", "comment": "用户ID"},
                    {"name": "name", "type": "varchar(100)", "comment": "用户名"},
                    {"name": "age", "type": "int", "comment": "年龄"}
                ]
            }
        ]
    },
    "database": {"type": "mysql", "version": "8.0"}
}

response = requests.post(url, json=data, headers=headers)
result = response.json()
print(result["sql"])
```

### JavaScript/Node.js

```javascript
const fetch = require('node-fetch');

const url = 'http://localhost:8080/api/v1/sql/generate';
const headers = {
  'Authorization': 'Bearer your-api-key',
  'Content-Type': 'application/json'
};
const data = {
  query: '查询所有年龄大于30的用户',
  schema: {
    tables: [
      {
        name: 'users',
        columns: [
          { name: 'id', type: 'int', comment: '用户ID' },
          { name: 'name', type: 'varchar(100)', comment: '用户名' },
          { name: 'age', type: 'int', comment: '年龄' }
        ]
      }
    ]
  },
  database: { type: 'mysql', version: '8.0' }
};

fetch(url, {
  method: 'POST',
  headers: headers,
  body: JSON.stringify(data)
})
  .then(res => res.json())
  .then(result => console.log(result.sql));
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    url := "http://localhost:8080/api/v1/sql/generate"
    
    data := map[string]interface{}{
        "query": "查询所有年龄大于30的用户",
        "schema": map[string]interface{}{
            "tables": []map[string]interface{}{
                {
                    "name": "users",
                    "columns": []map[string]interface{}{
                        {"name": "id", "type": "int", "comment": "用户ID"},
                        {"name": "name", "type": "varchar(100)", "comment": "用户名"},
                        {"name": "age", "type": "int", "comment": "年龄"},
                    },
                },
            },
        },
        "database": map[string]string{
            "type":    "mysql",
            "version": "8.0",
        },
    }
    
    jsonData, _ := json.Marshal(data)
    
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer your-api-key")
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, _ := client.Do(req)
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    
    fmt.Println(result["sql"])
}
```
