# Codex for OSS 申请资料参考

本文件用于填写 OpenAI Codex for OSS 试用申请。请在发布 GitHub 仓库后，把仓库真实 URL、维护者信息和使用计划填入申请表。

## 项目简介

`oss-maintainer-kit` 是一个 Go 编写的开源维护自动化 CLI。它读取 GitHub Issues 和 Pull Requests 的结构化导出数据，生成 issue triage、发布说明草稿和维护报告，帮助维护者处理重复但重要的项目维护工作。

当前版本还支持 GitHub REST API 数据导出、PR diff 风险扫描、SARIF 输出、OpenAI-compatible review prompt 生成、开源仓库健康度检查、CodeQL、Dependabot，以及 Codex for OSS 使用计划生成。项目覆盖了申请页强调的 PR review、issue triage、release workflow、security/code quality 等真实开源维护场景。

## 开源价值

- 降低维护者处理 issue 和 PR 的重复成本。
- 把安全风险、回归风险、长期未更新问题显式暴露出来。
- 对 PR diff 做确定性风险扫描，并把结果交给 Codex 复核。
- 输出 SARIF，便于把风险扫描接入 GitHub Code Scanning。
- 提供可离线运行、可审查、可测试的规则引擎。
- 提供仓库治理检查，帮助维护者补齐 README、License、Security、CI、Issue/PR 模板和路线图。
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
oss-maintainer-kit is a Go CLI that helps open-source maintainers export GitHub issues and pull requests, triage issues, scan pull request diffs for security risks, emit SARIF for GitHub Code Scanning, generate Codex-ready PR review prompts, produce release notes, check repository health, and build maintainer workflow plans.
```

### How you plan to use Codex

```text
I plan to use Codex to review pull requests, validate local diff-risk findings, expand issue triage rules, improve GitHub export handling, generate repository health recommendations, improve test coverage, detect edge cases in release-note generation, and maintain security-related workflows. The project is intentionally built around common OSS maintainer tasks where Codex can provide measurable value.
```

### Why API credits help

```text
API credits will be used to prototype maintainer workflows that summarize issues, review PR diffs, suggest tests, and generate release notes for open-source repositories. The output will remain reviewable by maintainers and covered by deterministic tests where possible.
```

### Example commands

```text
go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff
go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff --format sarif
go run ./cmd/oss-maintainer-kit ai-review --diff examples/pr.diff --prompt-only
go run ./cmd/oss-maintainer-kit codex-plan --issues examples/issues.json --pulls examples/pulls.json
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
