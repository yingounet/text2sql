# 贡献指南

感谢您对 Text2SQL API 项目的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 报告问题

如果您发现了 Bug 或有功能建议，请：

1. 检查 [Issues](https://github.com/your-repo/text2sql/issues) 中是否已有相关问题
2. 如果没有，创建新的 Issue，包含：
   - 问题描述
   - 复现步骤
   - 期望行为
   - 实际行为
   - 环境信息（Go 版本、操作系统等）

### 提交代码

1. **Fork 项目**
   ```bash
   # 在 GitHub 上 Fork 项目
   ```

2. **克隆你的 Fork**
   ```bash
   git clone https://github.com/your-username/text2sql.git
   cd text2sql
   ```

3. **创建特性分支**
   ```bash
   git checkout -b feature/AmazingFeature
   ```

4. **进行修改**
   - 编写代码
   - 添加测试（如适用）
   - 更新文档

5. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add AmazingFeature"
   ```

6. **推送分支**
   ```bash
   git push origin feature/AmazingFeature
   ```

7. **创建 Pull Request**
   - 在 GitHub 上创建 Pull Request
   - 描述你的更改和原因
   - 等待代码审查

## 代码规范

### 提交信息格式

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型 (type)**:
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整（不影响代码运行）
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关
- `perf`: 性能优化

**示例**:
```
feat(api): add conversation_id support

Add conversation_id field to request and response for multi-turn conversation support.

Closes #123
```

### 代码风格

1. **格式化**: 使用 `go fmt` 格式化代码
2. **命名**: 遵循 Go 命名规范
3. **注释**: 公共函数和类型需要添加注释
4. **测试**: 新功能需要添加测试

### 运行检查

```bash
# 格式化
go fmt ./...

# 代码检查（如果配置了 golangci-lint）
golangci-lint run

# 运行测试
go test ./...
```

## 开发流程

### 1. 设置开发环境

参考 [开发指南](docs/development.md) 设置本地开发环境。

### 2. 编写代码

- 遵循项目代码风格
- 添加必要的注释
- 编写测试

### 3. 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/text2sql

# 运行测试并显示覆盖率
go test -cover ./...
```

### 4. 更新文档

如果添加了新功能或修改了 API，请更新：
- README.md
- docs/api.md（如适用）
- docs/development.md（如适用）

## Pull Request 指南

### PR 标题

使用与提交信息相同的格式：

```
feat(api): add conversation_id support
```

### PR 描述

包含：
1. **变更说明**: 简要描述做了什么
2. **变更原因**: 为什么需要这个变更
3. **测试**: 如何测试这个变更
4. **相关 Issue**: 关联的 Issue 编号

### PR 检查清单

- [ ] 代码已格式化 (`go fmt`)
- [ ] 代码已通过检查（如 `golangci-lint`）
- [ ] 测试已通过
- [ ] 文档已更新
- [ ] 提交信息符合规范

## 代码审查

所有 PR 都需要经过代码审查。审查者会检查：
- 代码质量和风格
- 测试覆盖
- 文档完整性
- 性能影响

请耐心等待审查，并根据反馈进行修改。

## 行为准则

### 我们的承诺

为了营造开放和友好的环境，我们承诺：

- 尊重所有贡献者
- 欢迎不同观点和经验
- 优雅地接受建设性批评
- 关注对社区最有利的事情

### 不可接受的行为

- 使用性化语言或图像
- 人身攻击、侮辱性/贬损性评论
- 公开或私下骚扰
- 未经明确许可发布他人私人信息
- 其他不道德或不专业的行为

## 许可证

通过贡献，您同意您的贡献将在与项目相同的许可证下授权（MIT 许可证）。

## 联系方式

如有问题，可以通过以下方式联系：
- 创建 Issue
- 发送邮件（如有）

感谢您的贡献！
