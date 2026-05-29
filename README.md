# AI PR Review Demo

一个两天 Demo 范围的 AI PR Review 助手。用户输入 GitHub PR 链接后，后端获取 PR 基本信息和文件 patch，并在配置 AI 后生成结构化 Review 结果；前端展示 PR 信息、文件列表、diff、风险项和可复制的 Review Summary。

## 技术栈

- Frontend: React + Vite + Tailwind CSS
- Backend: Go + Gin
- Data source: GitHub REST API
- AI: OpenAI-compatible Chat Completions API

## 本地启动

启动后端：

```powershell
cd D:\Source\demo\backend
go run .
```

启动前端：

```powershell
cd D:\Source\demo\frontend
bun run dev
```

打开 Vite 输出的地址，默认通常是：

```text
http://localhost:5173
```

## 环境变量

后端支持以下环境变量：

```powershell
$env:PORT="8080"
$env:GITHUB_TOKEN="your_github_token"
$env:OPENAI_API_KEY="your_ai_api_key"
$env:OPENAI_MODEL="your_model"
$env:OPENAI_BASE_URL="https://api.openai.com/v1"
```

说明：

- `GITHUB_TOKEN` 可选；访问私有仓库或遇到 GitHub 限流时需要配置。
- `OPENAI_API_KEY` 和 `OPENAI_MODEL` 都配置后才会启用 AI 分析。
- `OPENAI_BASE_URL` 可选；兼容 OpenAI-compatible 服务。
- 未配置 AI 时，系统仍会返回 GitHub PR 数据和占位 Review 结果，方便演示基础链路。

## 当前能力

- 解析 GitHub PR URL。
- 获取 PR 标题、作者、变更文件数、增删行数。
- 获取 PR 文件列表和 patch。
- 调用 AI 生成 `summary`、`risks`、`review_comments`、`final_review`。
- 前端展示文件列表、patch diff、风险项和 Review Summary。
- 支持一键复制 Review Summary。

## 演示建议

可以优先选择公开仓库里变更较小的 PR，例如：

```text
https://github.com/gin-gonic/gin/pull/3950
```

后端接口也可以直接测试：

```powershell
Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/api/review `
  -ContentType application/json `
  -Body '{"pr_url":"https://github.com/gin-gonic/gin/pull/3950"}'
```

## Demo 限制

- 目前不自动把评论提交到 GitHub。
- 行级评论只展示模型输出，不做 GitHub 评论定位校验。
- 大型 PR 的 diff 会在后端 prompt 中截断。
- 没有历史记录、登录、多用户和持久化。
