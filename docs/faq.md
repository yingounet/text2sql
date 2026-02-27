# 常见问题 (FAQ)

## 通用问题

### Q: Text2SQL API 是什么？

A: Text2SQL API 是一个将自然语言转换为 SQL 查询语句的服务。它支持多种 LLM 提供商，可以帮助开发者快速生成 SQL 查询。

### Q: 支持哪些数据库？

A: 目前支持以下数据库：
- MySQL
- PostgreSQL
- SQLite

SQL 校验会根据数据库类型进行相应的语法检查。

### Q: 支持哪些 LLM 提供商？

A: 目前支持：
- **Ollama**: 本地部署的开源 LLM
- **OpenAI**: GPT 系列模型
- **OpenRouter**: 聚合多个 LLM 提供商的平台
- **Kimi**: 月之暗面的 AI 模型

### Q: 如何切换 LLM 提供商？

A: 修改 `config.yaml` 中的 `llm.provider` 字段，并配置对应的提供商参数：

```yaml
llm:
  provider: openai  # 改为你想要的提供商
  openai:
    api_key: ${OPENAI_API_KEY}
    base_url: https://api.openai.com/v1
    model: gpt-4o
```

### Q: 生成的 SQL 安全吗？

A: 系统会自动拦截危险操作，只允许生成 SELECT 查询。被拦截的操作包括：
- DROP
- TRUNCATE
- DELETE
- UPDATE
- INSERT
- ALTER
- CREATE TABLE
- EXEC/EXECUTE

同时会对 SQL 进行语法校验，确保生成的 SQL 符合目标数据库的语法规范。

## 多轮对话

### Q: conversation_id 是什么？

A: `conversation_id` 是会话的唯一标识符，用于关联多轮对话的上下文。第一轮请求后，系统会返回一个 `conversation_id`，后续请求携带此 ID 即可在之前的 SQL 基础上进行修改。

### Q: conversation_id 会过期吗？

A: 是的，会话在 24 小时未使用后会自动清理。如果会话过期，可以：
1. 重新开始新会话（不提供 conversation_id）
2. 使用 `previous_sql` 方式直接提供上一轮的 SQL

### Q: 使用 conversation_id 时需要注意什么？

A: 需要注意以下几点：
1. **Schema 一致性**: 当前请求的 `schema` 必须与历史会话的 `schema` 一致
2. **Database 一致性**: 当前请求的 `database.type` 和 `database.version` 必须与历史会话一致
3. **会话过期**: 会话在 24 小时未使用后会自动清理

### Q: previous_sql 和 conversation_id 有什么区别？

A:
- **conversation_id**: 系统自动管理上下文，可以保留多轮对话历史
- **previous_sql**: 只提供上一轮的 SQL，不保留历史上下文

如果同时提供两者，系统会优先使用 `previous_sql`。

### Q: 如何开始新的对话？

A: 不提供 `conversation_id` 和 `previous_sql` 即可开始新对话。

## 配置和部署

### Q: 如何设置 API Key？

A: 有两种方式：
1. 在 `config.yaml` 中设置 `api_key`
2. 使用环境变量 `API_KEY`（会覆盖配置文件）

### Q: 如何修改服务端口？

A: 修改 `config.yaml` 中的 `server.port`：

```yaml
server:
  port: 9090  # 改为你想要的端口
```

### Q: Docker 部署时如何传递配置？

A: 可以通过环境变量或挂载配置文件：

```bash
# 方式1: 环境变量
docker run -e API_KEY=your-key -e CONFIG_PATH=/app/config.yaml text2sql

# 方式2: 挂载配置文件
docker run -v $(pwd)/config.yaml:/app/config.yaml:ro text2sql
```

### Q: 数据库配置和 context_store 有什么用？

A: 当配置 `context_store: sqlite` 时，会使用 `database.dsn` 指定的 SQLite 文件持久化存储多轮对话上下文，服务重启后会话可恢复。默认 `context_store: memory` 使用内存存储。

## 错误处理

### Q: 收到 `UNAUTHORIZED` 错误怎么办？

A: 检查：
1. API Key 是否正确
2. Header 格式是否正确：`Authorization: Bearer <key>` 或 `X-API-Key: <key>`

### Q: 收到 `SQL_VALIDATION_FAILED` 错误怎么办？

A: 这表示生成的 SQL 校验失败。可能的原因：
1. SQL 语法错误
2. 包含了危险操作（如 DROP、DELETE 等）
3. 不符合目标数据库的语法规范

可以尝试：
1. 更明确地描述查询需求
2. 检查表结构是否正确
3. 查看错误信息了解具体原因

### Q: 收到 `CONVERSATION_NOT_FOUND` 错误怎么办？

A: 这表示 `conversation_id` 不存在或已过期。可以：
1. 重新开始新会话（不提供 conversation_id）
2. 使用 `previous_sql` 方式直接提供上一轮的 SQL

### Q: 收到 `SCHEMA_MISMATCH` 或 `DATABASE_MISMATCH` 错误怎么办？

A: 这表示当前请求的 `schema` 或 `database` 与历史会话不一致。确保：
1. `schema` 中的表结构和列定义与历史会话完全一致
2. `database.type` 和 `database.version` 与历史会话一致

## 性能和使用

### Q: 响应时间一般是多少？

A: 响应时间主要取决于 LLM 提供商的响应速度：
- Ollama（本地）: 通常 1-5 秒
- OpenAI/OpenRouter: 通常 2-10 秒
- 网络延迟也会影响响应时间

### Q: 有使用限制吗？

A: 当前版本（一期）没有内置的使用限制。限制主要取决于：
1. LLM 提供商的 API 限制
2. 服务器资源

二期版本将支持限频和额度统计。

### Q: 支持并发请求吗？

A: 是的，服务支持并发请求。每个请求都会独立处理。

### Q: 上下文存储在哪里？

A: 当前版本使用内存存储，会话数据存储在服务进程的内存中。重启服务会丢失所有会话数据。

## 开发相关

### Q: 如何添加新的 LLM 提供商？

A: 参考 [开发指南](development.md) 中的"添加新的 LLM 提供商"章节。

### Q: 如何贡献代码？

A: 参考 [开发指南](development.md) 中的"贡献指南"章节。

### Q: 如何报告 Bug？

A: 请在 GitHub Issues 中提交，包含：
1. 问题描述
2. 复现步骤
3. 错误信息
4. 环境信息（Go 版本、操作系统等）

## 其他

### Q: 项目使用什么许可证？

A: MIT 许可证。详见 [LICENSE](../LICENSE) 文件。

### Q: 有 API 文档吗？

A: 是的，详见 [API 文档](api.md)。

### Q: 有示例代码吗？

A: 是的，API 文档中包含了 Python、JavaScript 和 Go 的示例代码。
