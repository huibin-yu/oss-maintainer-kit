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
	Project        string
	Version        string
	Ready          bool
	Summary        model.MaintainerReport
	Health         health.Summary
	Blockers       []string
	Warnings       []string
	ReleaseCommand string
}

func Build(input Input) Result {
	summary := report.Maintainer(input.Issues, input.Pulls, triage.RuleSet{})
	result := Result{
		Project:        input.Project,
		Version:        input.Version,
		Summary:        summary,
		Health:         input.Health,
		ReleaseCommand: fmt.Sprintf("release-notes --input examples/pulls.json --version %s", input.Version),
	}
	if summary.SecurityIssues > 0 {
		result.Blockers = append(result.Blockers, fmt.Sprintf("安全相关 issue 未处理：%d 个", summary.SecurityIssues))
	}
	if input.Health.Score < 100 {
		result.Blockers = append(result.Blockers, fmt.Sprintf("健康度未达到 100/100：当前 %d/100", input.Health.Score))
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
	fmt.Fprintf(&b, "rtk go test ./...\n")
	fmt.Fprintf(&b, "rtk go build ./cmd/oss-maintainer-kit\n")
	fmt.Fprintf(&b, "rtk go run ./cmd/oss-maintainer-kit %s\n", result.ReleaseCommand)
	fmt.Fprintf(&b, "```\n")
	return b.String()
}
