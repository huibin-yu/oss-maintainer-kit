package releasecheck

import (
	"fmt"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
	"github.com/yuhuibin/oss-maintainer-kit/internal/report"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
)

type Input struct {
	Project string
	Version string
	Issues  []model.Issue
	Pulls   []model.PullRequest
	Health  health.Summary
}

type Result struct {
	Project  string
	Version  string
	Ready    bool
	Summary  model.MaintainerReport
	Health   health.Summary
	Policy   Policy
	Blockers []string
	Warnings []string
	Commands []string
}

func Build(input Input) Result {
	return BuildWithPolicy(input, DefaultPolicy())
}

func BuildWithPolicy(input Input, policy Policy) Result {
	summary := report.Maintainer(input.Issues, input.Pulls, triage.RuleSet{})
	result := Result{
		Project:  input.Project,
		Version:  input.Version,
		Summary:  summary,
		Health:   input.Health,
		Policy:   policy,
		Commands: commands(policy, input.Version),
	}
	if policy.BlockSecurityIssues && summary.SecurityIssues > 0 {
		result.Blockers = append(result.Blockers, fmt.Sprintf("安全相关 issue 未处理：%d 个", summary.SecurityIssues))
	}
	if input.Health.Score < policy.MinHealthScore {
		result.Blockers = append(result.Blockers, fmt.Sprintf("健康度未达到 %d/100：当前 %d/100", policy.MinHealthScore, input.Health.Score))
	}
	if policy.BlockStaleIssues && summary.StaleIssues > policy.MaxStaleIssues {
		result.Blockers = append(result.Blockers, fmt.Sprintf("长期未更新 issue 超过阈值：当前 %d 个，允许 %d 个", summary.StaleIssues, policy.MaxStaleIssues))
	}
	if summary.StaleIssues > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("存在长期未更新 issue：%d 个", summary.StaleIssues))
	}
	if summary.NeedsReview > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("存在需要维护者复查的 issue：%d 个", summary.NeedsReview))
	}
	result.Ready = len(result.Blockers) == 0
	return result
}

func Markdown(result Result) string {
	status := "READY"
	if !result.Ready {
		status = "BLOCKED"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s 发布准备检查\n\n", result.Project, result.Version)
	fmt.Fprintf(&b, "发布状态：**%s**\n\n", status)
	fmt.Fprintf(&b, "## 维护指标\n\n")
	fmt.Fprintf(&b, "- Open Issues：%d\n", result.Summary.OpenIssues)
	fmt.Fprintf(&b, "- 安全相关 Issues：%d\n", result.Summary.SecurityIssues)
	fmt.Fprintf(&b, "- 超 30 天未更新 Issues：%d\n", result.Summary.StaleIssues)
	fmt.Fprintf(&b, "- 需要维护者复查：%d\n", result.Summary.NeedsReview)
	fmt.Fprintf(&b, "- 已合并 PR：%d\n", result.Summary.MergedPulls)
	fmt.Fprintf(&b, "- 仓库健康度：%d/100\n", result.Health.Score)

	fmt.Fprintf(&b, "\n## 发布策略\n\n")
	fmt.Fprintf(&b, "- 最低健康度：%d/100\n", result.Policy.MinHealthScore)
	fmt.Fprintf(&b, "- 安全 issue 阻塞发布：%t\n", result.Policy.BlockSecurityIssues)
	fmt.Fprintf(&b, "- stale issue 阻塞发布：%t\n", result.Policy.BlockStaleIssues)
	if result.Policy.BlockStaleIssues {
		fmt.Fprintf(&b, "- stale issue 允许数量：%d\n", result.Policy.MaxStaleIssues)
	}

	fmt.Fprintf(&b, "\n## 阻塞项\n\n")
	if len(result.Blockers) == 0 {
		fmt.Fprintf(&b, "暂无阻塞项。\n")
	} else {
		for _, blocker := range result.Blockers {
			fmt.Fprintf(&b, "- %s\n", blocker)
		}
		for _, check := range result.Health.Checks {
			if !check.Passed {
				fmt.Fprintf(&b, "- %s（`%s`）：%s\n", check.Name, check.Path, check.Message)
			}
		}
	}

	fmt.Fprintf(&b, "\n## 风险提示\n\n")
	if len(result.Warnings) == 0 {
		fmt.Fprintf(&b, "暂无额外风险提示。\n")
	} else {
		for _, warning := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}

	fmt.Fprintf(&b, "\n## 发布前命令\n\n")
	fmt.Fprintf(&b, "```bash\n")
	for _, command := range result.Commands {
		fmt.Fprintf(&b, "%s\n", command)
	}
	fmt.Fprintf(&b, "```\n")
	return b.String()
}

func commands(policy Policy, version string) []string {
	values := append([]string{}, policy.RequiredCommands...)
	values = append(values, fmt.Sprintf("rtk go run ./cmd/oss-maintainer-kit release-notes --input examples/pulls.json --version %s", version))
	return values
}
