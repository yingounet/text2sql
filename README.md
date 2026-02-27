# Text2SQL API

自然语言转 SQL 的 API 服务，支持 Ollama、OpenAI、OpenRouter、Kimi 等 LLM 提供商。

## 功能特性

- ✅ 自然语言转 SQL（支持 MySQL、PostgreSQL、SQLite）
- ✅ 多 LLM 提供商支持（Ollama、OpenAI、OpenRouter、Kimi）
- ✅ 多轮对话上下文关联（支持 `conversation_id` 和 `previous_sql`）
- ✅ SQL 语法校验和安全检查（拦截危险操作）
- ✅ API Key 认证
- ✅ Docker 一键部署

## 一期功能（Docker 个人开源版）

- Text2SQL 核心（自然语言 + 表结构 → SQL）
- 配置文件 API Key 认证
- 多轮对话上下文管理（支持内存或 SQLite 存储，24小时自动过期）
- SQL 输出前校验（按数据库类型与版本）
- Docker 部署

## 快速开始

### 本地运行

1. 配置 `config.yaml`，设置 `api_key` 和 LLM 参数
2. 若使用 Ollama，确保 Ollama 已启动并加载模型
3. 运行：

```bash
go run ./cmd/server
```

### Docker

```bash
# 构建并运行
API_KEY=your-secret docker-compose up -d

# 调用健康检查
curl http://localhost:8080/api/v1/health

# 生成 SQL
curl -X POST http://localhost:8080/api/v1/sql/generate \
  -H "Authorization: Bearer your-secret" \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

## API

### 接口列表

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | /api/v1/health | 无 | 健康检查 |
| POST | /api/v1/sql/generate | API Key | 生成 SQL |

### POST /api/v1/sql/generate

生成 SQL 查询语句。

**请求体**：

```json
{
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
  "database": {
    "type": "mysql",
    "version": "8.0"
  },
  "conversation_id": "conv_xxx",  // 可选：会话ID，用于多轮对话
  "previous_sql": "SELECT * FROM users WHERE age > 30"  // 可选：上一轮SQL，用于追加修改
}
```

**请求字段说明**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | string | 是 | 用户自然语言查询意图 |
| `schema.tables` | array | 条件 | 表结构列表。新会话必填；续会话（提供有效的 conversation_id）时可省略 |
| `schema.tables[].name` | string | 是 | 表名 |
| `schema.tables[].columns` | array | 是 | 列定义 |
| `schema.tables[].columns[].name` | string | 是 | 列名 |
| `schema.tables[].columns[].type` | string | 否 | 列类型 |
| `schema.tables[].columns[].comment` | string | 否 | 列注释 |
| `database.type` | string | 条件 | 数据库类型：`mysql` / `postgresql` / `sqlite`。新会话必填；续会话时可省略 |
| `database.version` | string | 否 | 数据库版本，如 `8.0`、`14`、`3`，用于 SQL 校验与方言差异 |
| `conversation_id` | string | 否 | 会话ID，用于关联多轮对话上下文 |
| `previous_sql` | string | 否 | 上一轮的SQL语句，用于在现有SQL基础上修改 |

**响应**：

```json
{
  "sql": "SELECT * FROM users WHERE age > 30",
  "explanation": "筛选年龄大于30的用户",
  "conversation_id": "conv_a1b2c3d4e5f6g7h8i9j0k1l2"
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `sql` | string | 生成的 SQL 语句 |
| `explanation` | string | SQL 的简要说明 |
| `conversation_id` | string | 会话ID，供后续请求使用 |

### GET /api/v1/health

健康检查接口。

**响应**：

```json
{
  "status": "ok"
}
```

## 多轮对话

Text2SQL API 支持多轮对话，允许用户在第一轮生成的 SQL 基础上进行追加修改。

### 使用方式

#### 方式1：使用 conversation_id（推荐）

第一轮请求后，系统会返回 `conversation_id`，后续请求携带此 ID 即可关联上下文：

```bash
# 第一轮
curl -X POST http://localhost:8080/api/v1/sql/generate \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "查询所有年龄大于30的用户",
    "schema": {...},
    "database": {"type": "mysql", "version": "8.0"}
  }'

# 响应：{"sql": "...", "conversation_id": "conv_xxx", ...}

# 第二轮：追加修改（schema/database 可省略，从上下文复用）
curl -X POST http://localhost:8080/api/v1/sql/generate \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "还要筛选状态为active的用户",
    "conversation_id": "conv_xxx"
  }'
```

#### 方式2：直接提供 previous_sql

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

### 注意事项

1. **Schema/Database 可选**：续会话时，`schema` 和 `database` 可省略，系统会从上下文复用；若显式传入则需与历史一致，否则返回 `SCHEMA_MISMATCH` 或 `DATABASE_MISMATCH`。
3. **会话过期**：会话在 24 小时未使用后会自动清理。如果使用已过期的 `conversation_id`，会返回 `CONVERSATION_NOT_FOUND` 错误。
4. **优先级**：如果同时提供 `conversation_id` 和 `previous_sql`，系统会优先使用 `previous_sql`。

更多使用示例请参考 [多轮对话使用示例](docs/多轮对话使用示例.md)。

## 错误码

| 错误码 | HTTP状态码 | 说明 |
|--------|-----------|------|
| `UNAUTHORIZED` | 401 | API Key 缺失或无效 |
| `INVALID_REQUEST` | 400 | 请求参数错误 |
| `INVALID_SCHEMA` | 400 | Schema 格式错误或新会话未提供 schema |
| `SCHEMA_REQUIRED` | 400 | 新会话或 conversation_id 无效时需提供 schema |
| `DATABASE_REQUIRED` | 400 | 新会话或 conversation_id 无效时需提供 database |
| `SQL_VALIDATION_FAILED` | 400 | 生成的 SQL 校验失败 |
| `CONVERSATION_NOT_FOUND` | 404 | conversation_id 不存在或已过期 |
| `SCHEMA_MISMATCH` | 400 | schema 与历史会话不一致 |
| `DATABASE_MISMATCH` | 400 | database 与历史会话不一致 |
| `LLM_ERROR` | 500 | LLM 调用失败 |

## 配置

### 配置文件示例

创建 `config.yaml` 文件：

```yaml
server:
  port: 8080

# 调用 API 时需携带此 Key，支持环境变量 API_KEY 覆盖
api_key: "your-secret-api-key"

database:
  driver: sqlite
  dsn: "./data/text2sql.db"

# 上下文存储：memory（默认）| sqlite。sqlite 可持久化，支持服务重启后恢复
context_store: memory

llm:
  provider: ollama  # ollama | openai | openrouter | kimi
  
  # Ollama 配置
  ollama:
    base_url: http://localhost:11434
    model: qwen2.5:7b
  
  # OpenAI 配置
  # openai:
  #   api_key: ${OPENAI_API_KEY}
  #   base_url: https://api.openai.com/v1
  #   model: gpt-4o
  
  # OpenRouter 配置（OpenAI 兼容 API）
  # openrouter:
  #   api_key: ${OPENROUTER_API_KEY}
  #   base_url: https://openrouter.ai/api/v1
  #   model: anthropic/claude-3-haiku
  
  # Kimi 配置（月之暗面，OpenAI 兼容 API）
  # kimi:
  #   api_key: ${MOONSHOT_API_KEY}
  #   base_url: https://api.moonshot.cn/v1  # 中国站；国际站用 https://api.moonshot.ai/v1
  #   model: kimi-k2.5
```

### 环境变量

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `API_KEY` | API Key，覆盖配置文件中的 `api_key` | - |
| `CONFIG_PATH` | 配置文件路径 | `config.yaml` |

### 认证方式

调用 API 时需要在 Header 中携带 API Key：

- `Authorization: Bearer <api_key>`
- 或 `X-API-Key: <api_key>`

## 项目结构

```
text2sql/
├── cmd/server/          # 服务入口
├── internal/
│   ├── api/             # HTTP 处理器
│   ├── config/          # 配置管理
│   ├── llm/             # LLM 提供商抽象和实现
│   │   ├── ollama/      # Ollama 实现
│   │   └── openai/      # OpenAI/OpenRouter/Kimi 实现
│   └── text2sql/        # 核心服务逻辑
│       ├── service.go   # 服务主逻辑
│       ├── context.go   # 上下文存储
│       ├── validator.go # SQL 校验器
│       └── errors.go    # 错误定义
├── docs/                # 文档
│   └── 多轮对话使用示例.md
├── config.yaml          # 配置文件示例
├── Dockerfile           # Docker 构建文件
├── docker-compose.yaml  # Docker Compose 配置
└── README.md            # 项目说明
```

## 开发指南

### 本地开发

1. **环境要求**：
   - Go 1.22+
   - （可选）Ollama 本地运行

2. **安装依赖**：
   ```bash
   go mod download
   ```

3. **配置**：
   复制 `config.yaml` 并根据需要修改配置。

4. **运行**：
   ```bash
   go run ./cmd/server
   ```

5. **测试**：
   ```bash
   # 健康检查
   curl http://localhost:8080/api/v1/health
   
   # 生成 SQL
   curl -X POST http://localhost:8080/api/v1/sql/generate \
     -H "Authorization: Bearer your-api-key" \
     -H "Content-Type: application/json" \
     -d @test_request.json
   ```

### Docker 开发

```bash
# 构建镜像
docker build -t text2sql .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -e API_KEY=your-secret \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  text2sql
```

### 代码贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 常见问题

### Q: 支持哪些数据库？

A: 目前支持 MySQL、PostgreSQL 和 SQLite。SQL 校验会根据数据库类型进行相应的语法检查。

### Q: 如何切换 LLM 提供商？

A: 修改 `config.yaml` 中的 `llm.provider` 字段，并配置对应的提供商参数。

### Q: conversation_id 会过期吗？

A: 是的，会话在 24 小时未使用后会自动清理。如果会话过期，可以重新开始新会话或使用 `previous_sql` 方式。

### Q: 如何确保生成的 SQL 安全？

A: 系统会自动拦截危险操作（如 DROP、DELETE、UPDATE 等），只允许生成 SELECT 查询。同时会对 SQL 进行语法校验。

### Q: 支持哪些 LLM 提供商？

A: 目前支持：
- Ollama（本地部署）
- OpenAI
- OpenRouter
- Kimi（月之暗面）

更多问题请参考 [FAQ](docs/faq.md)。

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。
