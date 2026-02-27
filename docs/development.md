# 开发指南

## 环境要求

- Go 1.22 或更高版本
- （可选）Ollama 本地运行（用于本地测试）

## 项目结构

```
text2sql/
├── cmd/
│   └── server/              # 服务入口
│       └── main.go
├── internal/
│   ├── api/                 # HTTP 处理器
│   │   └── handler.go
│   ├── config/              # 配置管理
│   │   └── config.go
│   ├── llm/                 # LLM 提供商抽象和实现
│   │   ├── provider.go      # Provider 接口定义
│   │   ├── registry.go      # 注册与获取
│   │   ├── factory.go       # 工厂方法
│   │   ├── ollama/          # Ollama 实现
│   │   │   └── provider.go
│   │   └── openai/          # OpenAI/OpenRouter/Kimi 实现
│   │       └── provider.go
│   └── text2sql/            # 核心服务逻辑
│       ├── service.go       # 服务主逻辑
│       ├── context.go       # 上下文存储
│       ├── validator.go     # SQL 校验器
│       └── errors.go        # 错误定义
├── docs/                    # 文档
├── config.yaml              # 配置文件示例
├── Dockerfile               # Docker 构建文件
├── docker-compose.yaml      # Docker Compose 配置
├── go.mod                   # Go 模块定义
└── README.md                # 项目说明
```

## 本地开发

### 1. 克隆项目

```bash
git clone <repository-url>
cd text2sql
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置

复制并修改配置文件：

```bash
cp config.yaml config.local.yaml
# 编辑 config.local.yaml，设置 api_key 和 LLM 参数
```

或使用环境变量：

```bash
export API_KEY=your-secret-api-key
export CONFIG_PATH=config.yaml
```

### 4. 运行服务

```bash
go run ./cmd/server
```

服务将在 `http://localhost:8080` 启动。

### 5. 测试

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 生成 SQL
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

## 代码结构说明

### 核心模块

#### 1. Service (`internal/text2sql/service.go`)

核心服务逻辑，负责：
- 处理生成 SQL 的请求
- 管理多轮对话上下文
- 调用 LLM 生成 SQL
- SQL 校验

#### 2. Context Store (`internal/text2sql/context.go`)

上下文存储接口和实现：
- `ContextStore`: 存储接口
- `MemoryContextStore`: 内存实现（当前使用）
- 支持会话的增删改查和过期清理

#### 3. SQL Validator (`internal/text2sql/validator.go`)

SQL 校验器：
- 语法校验（基于 sqlparser）
- 危险操作拦截（DROP、DELETE 等）
- 支持 MySQL、PostgreSQL、SQLite

#### 4. LLM Provider (`internal/llm/`)

LLM 提供商抽象：
- `Provider`: 统一接口
- `ollama`: Ollama 实现
- `openai`: OpenAI/OpenRouter/Kimi 实现（OpenAI 兼容 API）

### 添加新的 LLM 提供商

1. 在 `internal/llm/` 下创建新的目录，如 `internal/llm/custom/`
2. 实现 `Provider` 接口：

```go
package custom

import "text2sql/internal/llm"

type Provider struct {
    // 配置字段
}

func (p *Provider) Name() string {
    return "custom"
}

func (p *Provider) Complete(ctx context.Context, req *llm.CompleteRequest) (*llm.CompleteResponse, error) {
    // 实现逻辑
}
```

3. 在 `internal/config/config.go` 中添加配置结构
4. 在 `internal/llm/factory.go` 中注册新提供商

## 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/text2sql

# 运行测试并显示覆盖率
go test -cover ./...
```

### 编写测试

测试文件命名：`*_test.go`

示例：

```go
package text2sql

import (
    "testing"
)

func TestGenerateSQL(t *testing.T) {
    // 测试逻辑
}
```

## 构建

### 本地构建

```bash
go build -o text2sql ./cmd/server
```

### Docker 构建

```bash
docker build -t text2sql .
```

### 交叉编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o text2sql-linux ./cmd/server

# macOS
GOOS=darwin GOARCH=amd64 go build -o text2sql-macos ./cmd/server

# Windows
GOOS=windows GOARCH=amd64 go build -o text2sql.exe ./cmd/server
```

## 代码规范

### 格式化

```bash
# 格式化代码
go fmt ./...

# 使用 goimports（需要安装）
goimports -w .
```

### 代码检查

建议使用 `golangci-lint`：

```bash
# 安装
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行
golangci-lint run
```

### 命名规范

- 包名：小写，简短
- 函数/变量：驼峰命名
- 常量：全大写，下划线分隔
- 接口：以 `er` 结尾（如 `Provider`）

## 调试

### 使用日志

项目使用标准库 `log`，可以通过环境变量控制日志级别：

```go
log.Printf("debug: %v", value)
```

### 使用调试器

使用 Delve：

```bash
# 安装
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试
dlv debug ./cmd/server
```

## 常见问题

### 1. 依赖问题

```bash
# 清理并重新下载
go clean -modcache
go mod download
```

### 2. 端口被占用

修改 `config.yaml` 中的 `server.port` 或使用环境变量。

### 3. LLM 连接失败

检查：
- LLM 服务是否正常运行
- API Key 是否正确
- 网络连接是否正常

## 贡献指南

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 提交规范

提交信息格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

示例：

```
feat(api): add conversation_id support

Add conversation_id field to request and response for multi-turn conversation support.

Closes #123
```

## 发布流程

1. 更新版本号（在代码或配置中）
2. 更新 CHANGELOG.md
3. 创建 Git Tag
4. 构建并测试
5. 发布到 Docker Hub（如适用）
