# Codex for OSS 申请资料参考

本文件用于填写 OpenAI Codex for OSS 试用申请。请在发布 GitHub 仓库后，把仓库真实 URL、维护者信息和使用计划填入申请表。

## 项目简介

`oss-maintainer-kit` 是一个 Go 编写的开源维护自动化 CLI。它读取 GitHub Issues 和 Pull Requests 的结构化导出数据，生成带命中证据和建议动作的 issue triage、发布说明草稿和维护报告，帮助维护者处理重复但重要的项目维护工作。

当前版本还支持 GitHub REST/GraphQL API 分页数据导出、PR diff 风险扫描、自定义 review 规则、PR 评论格式输出、GitHub PR 评论创建或更新、SARIF 输出、OpenAI-compatible review prompt 生成、provider 配置文件与重试策略、测试建议和 Codex 测试计划 prompt、安全专项报告与 GitHub Actions 安全门禁、SPDX SBOM 生成、可配置发布准备检查、release-check GitHub Actions 门禁、tag 发布产物与 provenance、开源仓库健康度检查、健康度趋势报告、CI/Dependabot/PR 模板内容质量检查、CodeQL、govulncheck、OpenSSF Scorecard、Dependabot，以及 Codex for OSS 使用计划生成。项目覆盖了申请页强调的 PR review、issue triage、release workflow、security/code quality 等真实开源维护场景。

新增的 `application-pack` 命令会把维护报告、仓库健康度、release/security readiness、Codex 使用计划、API credits 用途和可执行验证命令聚合成一份申请证据包，便于在公开仓库中展示项目与 Codex for OSS 试用目标的匹配度。

## 开源价值

- 降低维护者处理 issue 和 PR 的重复成本，并让优先级判断具备可审查的原因、证据和建议动作。
- 把安全风险、回归风险、长期未更新问题显式暴露出来。
- 对 PR diff 做确定性风险扫描，并把结果交给 Codex 复核。
- 在发布前按仓库策略检查安全 issue、stale issue、仓库健康度、风险提示和发布命令，减少遗漏。
- 在 GitHub Actions 中导出当前仓库 issues/PRs 后运行 release-check，并在 BLOCKED 时让 workflow 失败，把发布门禁纳入 PR 和 main 分支自动化。
- 支持通过 REST 或 GraphQL 按 state、更新时间窗口和 limit 分页导出真实 GitHub issues/PRs，便于对活跃维护窗口做可复验分析。
- 支持仓库自定义规则，让不同项目按自己的风险模型调整审查策略。
- 生成稳定 Markdown PR 评论，便于后续接入 GitHub Bot 自动更新。
- 通过稳定 marker 创建或更新 GitHub PR 评论，避免 Bot 重复刷评论。
- 输出 SARIF，便于把风险扫描接入 GitHub Code Scanning。
- 通过 govulncheck 在 PR、main 和定时任务中扫描 Go 漏洞，增强供应链安全证据。
- 通过 OpenSSF Scorecard 输出 SARIF 并发布安全治理结果，展示项目对开源安全成熟度的持续维护。
- 聚合安全 issue、PR diff 风险发现和仓库安全治理缺口，生成商用安全审查和发布前风险接受可读的安全报告，并在 GitHub Actions 中上传 Markdown/JSON 证据。
- 生成 SPDX 2.3 JSON SBOM，为供应链审查、商用尽调和发布归档提供机器可读证据。
- 在版本 tag 上构建多平台 CLI，生成 checksums，并通过 GitHub artifact attestation 生成发布产物 provenance。
- 提供可离线运行、可审查、可测试的规则引擎。
- 提供仓库治理检查和失败项修复建议，帮助维护者补齐 README、License、Security、CI、govulncheck、Scorecard、Issue/PR 模板、路线图和关键 workflow 内容质量。
- 记录健康度 JSONL 快照并生成趋势报告，展示开源治理质量随时间改善。
- 后续可作为 Codex 辅助开源维护的示例项目。

## 申请表可填写内容

### Repository URL

填写发布后的 GitHub 仓库地址，例如：

```text
https://github.com/<your-account>/oss-maintainer-kit
```

### Role

```text
Owner / Maintainer
```

### Project description

```text
oss-maintainer-kit is a Go CLI that helps open-source maintainers export paginated GitHub issues and pull requests through REST or GraphQL, triage issues, scan pull request diffs with repository-specific rules, emit SARIF for GitHub Code Scanning, generate security reports, test plans, and SPDX SBOMs, run security and release gates in GitHub Actions, build release artifacts with provenance attestations, run Go vulnerability checks and OpenSSF Scorecard, generate PR comments and Codex-ready review prompts, produce release notes, check repository health, track health trends, and build maintainer workflow plans.
```

### How you plan to use Codex

```text
I plan to use Codex to review pull requests, validate local diff-risk findings, expand explainable issue triage rules, improve GitHub export handling, generate repository health recommendations, improve test coverage, detect edge cases in release-note generation, and maintain security-related workflows. The project is intentionally built around common OSS maintainer tasks where Codex can provide measurable value.
```

### Why API credits help

```text
API credits will be used to prototype maintainer workflows that summarize issues, review PR diffs, suggest tests, and generate release notes for open-source repositories. The output will remain reviewable by maintainers and covered by deterministic tests where possible.
```

### Example commands

```text
go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff --config examples/review-rules.json
go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff --config examples/review-rules.json --format sarif
go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff --config examples/review-rules.json --format comment
GITHUB_TOKEN=ghp_xxx go run ./cmd/oss-maintainer-kit github-comment --repo owner/name --pr 123 --diff examples/pr.diff --config examples/review-rules.json
GITHUB_TOKEN=ghp_xxx go run ./cmd/oss-maintainer-kit github-export --repo owner/name --kind issues --api rest --state open --since 2026-06-01T00:00:00Z --limit 200
GITHUB_TOKEN=ghp_xxx go run ./cmd/oss-maintainer-kit github-export --repo owner/name --kind pulls --api graphql --state closed --limit 100
go run ./cmd/oss-maintainer-kit release-check --issues examples/issues.json --pulls examples/pulls.json --root . --version v0.1.0 --policy examples/release-policy.json
go run ./cmd/oss-maintainer-kit release-check --issues release-issues.json --pulls release-pulls.json --root . --version v0.1.0 --policy examples/release-policy.json --fail-on-blocked
go run ./cmd/oss-maintainer-kit security-report --issues examples/issues.json --diff examples/pr.diff --root . --project oss-maintainer-kit --fail-on-risk
go run ./cmd/oss-maintainer-kit health-snapshot --root . --history health-history.jsonl
go run ./cmd/oss-maintainer-kit health-trend --history health-history.jsonl
go run ./cmd/oss-maintainer-kit sbom --root . --project oss-maintainer-kit --output sbom.spdx.json
go run ./cmd/oss-maintainer-kit ai-review --diff examples/pr.diff --config examples/review-rules.json --provider-config examples/ai-provider.json --prompt-only
go run ./cmd/oss-maintainer-kit test-plan --diff examples/pr.diff --config examples/review-rules.json --project oss-maintainer-kit
go run ./cmd/oss-maintainer-kit codex-plan --issues examples/issues.json --pulls examples/pulls.json
go run ./cmd/oss-maintainer-kit application-pack --issues examples/issues.json --pulls examples/pulls.json --root . --repo-url https://github.com/<your-account>/oss-maintainer-kit --version v0.1.0 --policy examples/release-policy.json
```

### 生成申请证据包

```bash
go run ./cmd/oss-maintainer-kit application-pack \
  --issues examples/issues.json \
  --pulls examples/pulls.json \
  --root . \
  --project oss-maintainer-kit \
  --repo-url https://github.com/<your-account>/oss-maintainer-kit \
  --version v0.1.0 \
  --policy examples/release-policy.json \
  --output codex-oss-application.md
```

## 发布前检查清单

- [ ] 推送到公开 GitHub 仓库。
- [ ] 确认 README 能说明项目用途、运行方式和维护价值。
- [ ] 保留 MIT License 或替换为你需要的开源许可证。
- [ ] 至少创建 2-3 个真实 issue，展示路线图、bug 或 enhancement。
- [ ] 确认 GitHub Actions CI 通过。
- [ ] 在申请表中填写真实仓库 URL 和真实维护者身份。

## 注意事项

不要在申请中夸大项目使用量、star 数、下载量或维护者身份。申请更看重真实开源维护场景和清晰的 Codex 使用计划。
