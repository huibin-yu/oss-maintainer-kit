package applicationpack

import (
	"fmt"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/codexplan"
	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
	"github.com/yuhuibin/oss-maintainer-kit/internal/report"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
)

type Input struct {
	Project    string
	Repository string
	Issues     []model.Issue
	Pulls      []model.PullRequest
	Health     health.Summary
}

type Pack struct {
	Project    string
	Repository string
	Summary    model.MaintainerReport
	Plan       codexplan.Plan
	Health     health.Summary
}

func Build(input Input) Pack {
	summary := report.Maintainer(input.Issues, input.Pulls, triage.RuleSet{})
	return Pack{
		Project:    input.Project,
		Repository: input.Repository,
		Summary:    summary,
		Plan:       codexplan.Build(input.Project, input.Repository, summary),
		Health:     input.Health,
	}
}

func Markdown(pack Pack) string {
	var b strings.Builder
	repository := pack.Repository
	if repository == "" {
		repository = "发布后填写公开 GitHub 仓库地址"
	}

	fmt.Fprintf(&b, "# Codex for OSS 申请证据包\n\n")
	fmt.Fprintf(&b, "项目：%s\n\n", pack.Project)
	fmt.Fprintf(&b, "仓库：%s\n\n", repository)

	fmt.Fprintf(&b, "## 申请表可填内容\n\n")
	fmt.Fprintf(&b, "### Project description\n\n")
	fmt.Fprintf(&b, "oss-maintainer-kit is a Go CLI for open-source maintainers. It supports PR review, issue triage, release workflow, security workflow, repository health checks, SARIF output, GitHub export, and Codex-ready review prompts.\n\n")
	fmt.Fprintf(&b, "### How you plan to use Codex\n\n")
	fmt.Fprintf(&b, "Use Codex to review pull requests, validate deterministic diff findings, improve issue triage rules, generate test suggestions, draft release notes, and keep security/code quality workflows reviewable by maintainers.\n\n")
	fmt.Fprintf(&b, "### Why API credits help\n\n")
	fmt.Fprintf(&b, "API credits will be used to prototype OSS maintainer workflows that summarize issues, review PR diffs, suggest tests, and generate release notes while keeping deterministic checks and maintainer approval as guardrails.\n\n")

	fmt.Fprintf(&b, "## 当前维护证据\n\n")
	fmt.Fprintf(&b, "| 指标 | 数量 |\n| --- | ---: |\n")
	fmt.Fprintf(&b, "| Open Issues | %d |\n", pack.Summary.OpenIssues)
	fmt.Fprintf(&b, "| 超 30 天未更新 Issues | %d |\n", pack.Summary.StaleIssues)
	fmt.Fprintf(&b, "| 安全相关 Issues | %d |\n", pack.Summary.SecurityIssues)
	fmt.Fprintf(&b, "| 需要维护者复查 | %d |\n", pack.Summary.NeedsReview)
	fmt.Fprintf(&b, "| 已合并 PR | %d |\n\n", pack.Summary.MergedPulls)

	fmt.Fprintf(&b, "## Codex 使用场景\n\n")
	for _, item := range pack.Plan.Workflows {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	fmt.Fprintf(&b, "\n## API Credits 用途\n\n")
	for _, item := range pack.Plan.APIUseCases {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	fmt.Fprintf(&b, "\n## 维护者约束\n\n")
	for _, item := range pack.Plan.Guardrails {
		fmt.Fprintf(&b, "- %s\n", item)
	}

	fmt.Fprintf(&b, "\n## 开源仓库健康度\n\n")
	fmt.Fprintf(&b, "健康度评分：**%d/100**\n\n", pack.Health.Score)
	missing := missingChecks(pack.Health)
	if len(missing) == 0 {
		fmt.Fprintf(&b, "申请前基础治理项已通过。\n")
	} else {
		fmt.Fprintf(&b, "### 需要补齐的申请前事项\n\n")
		for _, check := range missing {
			fmt.Fprintf(&b, "- `%s`：%s\n", check.Path, check.Message)
			if check.Recommendation != "" {
				fmt.Fprintf(&b, "  建议：%s\n", check.Recommendation)
			}
		}
	}

	fmt.Fprintf(&b, "\n## 可执行验证命令\n\n")
	fmt.Fprintf(&b, "```bash\n")
	fmt.Fprintf(&b, "rtk go test ./...\n")
	fmt.Fprintf(&b, "rtk go build ./cmd/oss-maintainer-kit\n")
	fmt.Fprintf(&b, "rtk go run ./cmd/oss-maintainer-kit health --root .\n")
	fmt.Fprintf(&b, "rtk go run ./cmd/oss-maintainer-kit sbom --root . --project oss-maintainer-kit --output sbom.spdx.json\n")
	fmt.Fprintf(&b, "rtk go run ./cmd/oss-maintainer-kit review-diff --diff examples/pr.diff --config examples/review-rules.json --format sarif\n")
	fmt.Fprintf(&b, "```\n")
	return b.String()
}

func missingChecks(summary health.Summary) []health.Check {
	var checks []health.Check
	for _, check := range summary.Checks {
		if !check.Passed {
			checks = append(checks, check)
		}
	}
	return checks
}
