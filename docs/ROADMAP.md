# 路线图

## v0.1

- 提供 issue triage、release notes、maintainer report 三个基础命令。
- 支持本地 JSON 输入。
- 提供单元测试、CI 和示例数据。
- 提供仓库健康度检查和 Codex for OSS 使用计划生成。
- 提供 PR diff 风险扫描和 OpenAI-compatible review prompt 生成。
- 支持 SARIF 2.1.0 输出、CodeQL、govulncheck、OpenSSF Scorecard、Dependabot 和 PR diff 扫描 workflow。
- 支持 JSON 自定义 review 规则和 GitHub PR 评论格式输出。
- 支持 GitHub API 创建或更新 PR review comment，使用稳定 marker 避免重复评论。
- 支持 GitHub REST API 分页导出，并按 state、since 和 limit 过滤 issues/PRs。
- 支持 Codex for OSS 申请证据包生成，聚合维护指标、仓库健康度、Codex 使用计划和验证命令。
- 支持 SPDX 2.3 JSON SBOM 生成，为供应链审查和商用尽调提供可复验材料。
- 支持 tag 发布产物 workflow，生成多平台 CLI、SBOM、SHA256 checksums 和 provenance attestation。
- 支持 OSSF Scorecard 风格的本地治理内容检查，覆盖 CI 权限、测试构建命令、govulncheck、OpenSSF Scorecard、SARIF 上传、Dependabot 覆盖和 PR 模板提示。
- 支持仓库治理检查项的分项解释和修复建议。
- 支持仓库健康度趋势报告，使用 JSONL 快照记录评分变化。
- 支持可配置发布准备检查，结合 issues、PRs、仓库健康度和发布策略输出 READY/BLOCKED、阻塞项和发布前命令。
- 支持 release-check GitHub Actions workflow，在 PR 和 main 分支上导出当前仓库真实数据，并在发布被策略阻塞时失败。
- 支持安全专项报告，聚合安全 issue、PR diff 风险发现和仓库治理缺口。

## v0.2

- 增加 GraphQL 支持。
- 输出更细粒度的维护优先级解释。

## v0.3

- 完善 OpenAI-compatible provider 的配置文件和重试策略。
- 支持使用 Codex 生成测试建议和发布说明摘要。
- 保留确定性规则作为兜底，避免 AI 输出不可审查。

## v0.4

- 支持 GitHub Checks / Markdown comment 输出。
- 支持维护趋势指标。
