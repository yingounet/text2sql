# 更新日志

所有重要的变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [未发布]

### 新增
- 多轮对话上下文关联功能（conversation_id 和 previous_sql）
- 内存上下文存储（自动过期清理）
- 支持 Kimi（月之暗面）LLM 提供商
- 完整的 API 文档和使用示例
- 开发指南和贡献指南

### 改进
- 完善 README 文档
- 增强错误处理和错误码
- 优化 SQL 校验器

### 文档
- 添加 API 文档 (docs/api.md)
- 添加开发指南 (docs/development.md)
- 添加常见问题 (docs/faq.md)
- 添加贡献指南 (CONTRIBUTING.md)
- 添加更新日志 (CHANGELOG.md)

## [0.1.0] - 2024-XX-XX

### 新增
- Text2SQL 核心功能（自然语言转 SQL）
- 支持 MySQL、PostgreSQL、SQLite
- SQL 语法校验和安全检查
- 支持 Ollama、OpenAI、OpenRouter LLM 提供商
- API Key 认证
- Docker 部署支持
- 健康检查接口

### 技术栈
- Go 1.22+
- Chi Router
- SQL Parser
- YAML 配置

---

## 版本说明

- **主版本号**: 不兼容的 API 修改
- **次版本号**: 向下兼容的功能性新增
- **修订号**: 向下兼容的问题修正

## 链接

- [GitHub Releases](https://github.com/yingounet/text2sql/releases)
- [完整变更历史](https://github.com/yingounet/text2sql/commits/main)
