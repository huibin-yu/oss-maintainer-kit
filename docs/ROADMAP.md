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
- 支持 GitHub REST/GraphQL API 分页导出，并按 state、since 和 limit 过滤 issues/PRs。
- 支持 Codex for OSS 申请证据包生成，聚合维护指标、仓库健康度、release/security readiness、Codex 使用计划和验证命令。
- 支持 SPDX 2.3 JSON SBOM 生成，为供应链审查和商用尽调提供可复验材料。
- 支持 tag 发布产物 workflow，生成多平台 CLI、SBOM、SHA256 checksums 和 provenance attestation。
- 支持 OSSF Scorecard 风格的本地治理内容检查，覆盖 CI 权限、测试构建命令、govulncheck、OpenSSF Scorecard、SARIF 上传、Dependabot 覆盖和 PR 模板提示。
- 支持仓库治理检查项的分项解释和修复建议。
- 支持仓库健康度趋势报告，使用 JSONL 快照记录评分变化。
- 支持可配置发布准备检查，结合 issues、PRs、仓库健康度和发布策略输出 READY/BLOCKED、阻塞项和发布前命令。
- 支持 release-check GitHub Actions workflow，在 PR 和 main 分支上导出当前仓库真实数据，并在发布被策略阻塞时失败。
- 支持安全专项报告和 GitHub Actions 门禁，聚合安全 issue、PR diff 风险发现和仓库治理缺口，并上传 Markdown/JSON 报告证据。
- 支持可解释 issue triage，在优先级结果中输出命中原因、证据和建议动作。

## v0.2

- 支持将可解释 triage 结果用于 GitHub 评论和 Checks 输出。

## v0.3

- 支持 OpenAI-compatible provider 配置文件和 HTTP 429/5xx 重试策略。
- 支持根据 PR diff 和本地风险发现生成测试建议和 Codex 测试计划 prompt。
- 支持根据已合并 PR 生成发布摘要、风险提示和 Codex 发布摘要 prompt，并可调用 OpenAI-compatible provider。
- 保留确定性规则作为兜底，避免 AI 输出不可审查。

## v0.4

- 支持 GitHub Checks / Markdown comment 输出。
- 支持维护趋势指标。
