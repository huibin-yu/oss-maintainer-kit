# 路线图

## v0.1

- 提供 issue triage、release notes、maintainer report 三个基础命令。
- 支持本地 JSON 输入。
- 提供单元测试、CI 和示例数据。
- 提供仓库健康度检查和 Codex for OSS 使用计划生成。
- 提供 PR diff 风险扫描和 OpenAI-compatible review prompt 生成。
- 支持 SARIF 2.1.0 输出、CodeQL、Dependabot 和 PR diff 扫描 workflow。

## v0.2

- 完善 GitHub REST API 输入。
- 增加分页、时间范围过滤和 GraphQL 支持。
- 输出更细粒度的维护优先级解释。
- 支持把 `review-diff` 结果输出为 GitHub Markdown comment。
- 增加 OSSF Scorecard 风格的仓库治理检查项。

## v0.3

- 完善 OpenAI-compatible provider 的配置文件和重试策略。
- 支持使用 Codex 生成测试建议和发布说明摘要。
- 保留确定性规则作为兜底，避免 AI 输出不可审查。

## v0.4

- 支持 GitHub Checks / Markdown comment 输出。
- 增加安全问题专用报告。
- 支持维护趋势指标。
