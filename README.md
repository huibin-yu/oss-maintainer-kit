# oss-maintainer-kit

`oss-maintainer-kit` 是一个面向开源维护者的 Go CLI，用于把 GitHub Issues 和 Pull Requests 的导出数据转成可执行的维护建议。

项目目标是帮助维护者降低重复维护成本：快速分类 issue、识别安全风险、生成发布说明草稿，并输出维护报告。它可以离线运行，不依赖第三方服务；后续可接入 OpenAI 兼容 API，用 Codex 辅助 PR review、issue triage 和 release workflow。

## 功能

- `triage`：根据标题、正文、标签和更新时间生成优先级与建议标签。
- `release-notes`：从已合并 PR 生成发布说明草稿。
- `report`：汇总 open issues、长期未更新问题、安全风险和优先处理项。
- `health`：检查开源仓库是否具备 README、License、Security、CI、Issue/PR 模板和路线图。
- `codex-plan`：根据维护报告生成 Codex for OSS 使用计划。
- `application-pack`：聚合维护报告、健康度检查和 Codex 使用计划，生成 Codex for OSS 申请证据包。
- `github-export`：从 GitHub REST API 导出 issues 或 PRs，便于接入真实仓库数据。
- `review-diff`：离线扫描 PR diff 中的硬编码密钥、命令执行、明文 HTTP、禁用 TLS 等风险。
- `review-diff --format sarif`：输出 SARIF 2.1.0，便于接入 GitHub Code Scanning。
- `review-diff --config`：加载仓库自定义 JSON 规则。
- `review-diff --format comment`：生成可直接用于 GitHub PR 的 Markdown 评论。
- `github-comment`：基于稳定 marker 在 GitHub PR 中创建或更新风险检查评论，避免重复刷屏。
- `ai-review`：生成 Codex/AI PR review prompt，或调用 OpenAI-compatible `/v1/chat/completions`。
- GitHub Actions：内置 CI、CodeQL、PR diff SARIF 扫描和 Dependabot 配置。
- 纯标准库实现，便于审查、测试和二次开发。
- 内置示例数据、单元测试和 GitHub Actions CI。

## 安装

```bash
go install github.com/yuhuibin/oss-maintainer-kit/cmd/oss-maintainer-kit@latest
```

本地开发时也可以直接构建：

```bash
go build ./cmd/oss-maintainer-kit
```

## 使用方法

运行 issue 分类：

```bash
go run ./cmd/oss-maintainer-kit triage --input examples/issues.json
```

输出 JSON：

```bash
go run ./cmd/oss-maintainer-kit triage --input examples/issues.json --format json
```

生成发布说明：

```bash
go run ./cmd/oss-maintainer-kit release-notes --input examples/pulls.json --version v0.1.0
```

生成维护报告：

```bash
go run ./cmd/oss-maintainer-kit report \
  --issues examples/issues.json \
  --pulls examples/pulls.json \
  --project oss-maintainer-kit \
  --output maintainer-report.md
```

检查开源仓库健康度：

```bash
go run ./cmd/oss-maintainer-kit health --root .
```

生成 Codex for OSS 使用计划：

```bash
go run ./cmd/oss-maintainer-kit codex-plan \
  --issues examples/issues.json \
  --pulls examples/pulls.json \
  --project oss-maintainer-kit \
  --repo-url https://github.com/<your-account>/oss-maintainer-kit \
  --output codex-plan.md
```

生成 Codex for OSS 申请证据包：

```bash
go run ./cmd/oss-maintainer-kit application-pack \
  --issues examples/issues.json \
  --pulls examples/pulls.json \
  --root . \
  --project oss-maintainer-kit \
  --repo-url https://github.com/<your-account>/oss-maintainer-kit \
  --output codex-oss-application.md
```

从 GitHub 导出数据：

```bash
GITHUB_TOKEN=ghp_xxx go run ./cmd/oss-maintainer-kit github-export \
  --repo owner/name \
  --kind issues \
  --output examples/issues.json
```

离线检查 PR diff：

```bash
go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff
```

使用自定义规则：

```bash
go run ./cmd/oss-maintainer-kit review-diff \
  --diff examples/pr.diff \
  --config examples/review-rules.json
```

输出 SARIF：

```bash
go run ./cmd/oss-maintainer-kit review-diff \
  --diff examples/pr.diff \
  --format sarif
```

生成 PR 评论：

```bash
go run ./cmd/oss-maintainer-kit review-diff \
  --diff examples/pr.diff \
  --config examples/review-rules.json \
  --format comment
```

创建或更新 GitHub PR 评论：

```bash
GITHUB_TOKEN=ghp_xxx go run ./cmd/oss-maintainer-kit github-comment \
  --repo owner/name \
  --pr 123 \
  --diff examples/pr.diff \
  --config examples/review-rules.json
```

生成 Codex review prompt：

```bash
go run ./cmd/oss-maintainer-kit ai-review \
  --diff examples/pr.diff \
  --project oss-maintainer-kit \
  --prompt-only
```

调用 OpenAI-compatible API：

```bash
OPENAI_API_KEY=sk_xxx go run ./cmd/oss-maintainer-kit ai-review \
  --diff examples/pr.diff \
  --project oss-maintainer-kit \
  --prompt-only=false \
  --base-url https://api.openai.com/v1 \
  --model gpt-4.1-mini
```

## 数据格式

`examples/issues.json`：

```json
[
  {
    "number": 12,
    "title": "token leak in debug output",
    "body": "When debug mode is enabled the CLI prints a secret token in logs.",
    "state": "open",
    "author": "alice",
    "labels": ["bug"],
    "created_at": "2026-04-20T08:00:00Z",
    "updated_at": "2026-04-22T08:00:00Z"
  }
]
```

`examples/pulls.json`：

```json
[
  {
    "number": 31,
    "title": "feat: add issue triage command",
    "state": "closed",
    "author": "alice",
    "labels": ["feature"],
    "merged": true,
    "created_at": "2026-05-20T08:00:00Z",
    "updated_at": "2026-05-21T08:00:00Z",
    "merged_at": "2026-05-21T08:00:00Z"
  }
]
```

`examples/review-rules.json`：

```json
{
  "rules": [
    {
      "id": "missing-timeout",
      "severity": "medium",
      "contains": ["http.Get(", "http.Post("],
      "message": "新增 HTTP 调用需要确认 timeout、错误处理和上下文取消",
      "tags": ["reliability", "network"]
    }
  ]
}
```

## 适合 Codex 辅助的维护场景

这个项目刻意选择了开源维护者高频、可验证的工作流：

- PR review：对规则变更、输出格式、边界条件和测试覆盖进行审查。
- AI-assisted review：把本地 diff 风险扫描结果和 PR diff 组合成可审查的 Codex prompt。
- Repository-specific rules：通过 JSON 配置给不同仓库定制 review 规则。
- PR comments：生成带稳定标记的 Markdown 评论，便于后续 GitHub Bot 更新评论。
- GitHub comment upsert：使用 GitHub issue comments API 更新已有 Bot 评论或创建新评论，形成 PR review 自动化闭环。
- Code scanning：通过 SARIF 输出接入 GitHub Code Scanning。
- issue triage：把非结构化 issue 内容转换为优先级、标签和处理建议。
- release workflow：根据合并 PR 自动生成发布说明草稿。
- security workflow：识别安全关键词、凭证泄露和高风险问题。
- repository health：检查开源治理材料是否完整，便于维护者持续改进仓库质量。
- application evidence：把维护指标、健康度、Codex 使用场景、API credits 用途和验证命令聚合成申请证据包。
- dependency maintenance：使用 Dependabot 管理 Go module 和 GitHub Actions 更新。
- code quality：维护规则引擎、CLI 体验、测试覆盖和 CI。

## 开发方式

```bash
go test ./...
go build ./cmd/oss-maintainer-kit
```

建议提交前运行：

```bash
gofmt -w .
go test ./...
```

## 目录结构

```text
cmd/oss-maintainer-kit   CLI 入口
internal/input           JSON 输入加载
internal/model           核心数据结构
internal/triage          issue 分类规则
internal/report          报告与发布说明生成
internal/github          GitHub REST API 数据导出
internal/health          开源仓库健康度检查
internal/codexplan       Codex 使用计划生成
internal/applicationpack  Codex for OSS 申请证据包生成
internal/diffreview      PR diff 风险扫描
internal/sarif           SARIF 2.1.0 输出
internal/reviewconfig    自定义 review 规则配置
internal/prcomment       PR 评论格式生成
internal/ai              OpenAI-compatible review client
examples                 示例数据
docs                     申请与路线图材料
```

## 路线图

- 增加 OSSF Scorecard 风格的治理检查项。
- 增加仓库健康度趋势报告。

## 许可证

MIT License，详见 [LICENSE](LICENSE)。
