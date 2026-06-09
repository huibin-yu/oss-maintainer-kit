package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
)

func Maintainer(issues []model.Issue, pulls []model.PullRequest, rules triage.RuleSet) model.MaintainerReport {
	results := rules.Issues(issues)
	report := model.MaintainerReport{TopSuggestedWork: top(results, 5)}

	for _, issue := range issues {
		if !strings.EqualFold(issue.State, "closed") {
			report.OpenIssues++
		}
	}
	for _, result := range results {
		if result.StaleDays >= 30 {
			report.StaleIssues++
		}
		if result.NeedsSecurity {
			report.SecurityIssues++
		}
		if result.NeedsReview {
			report.NeedsReview++
		}
	}
	for _, pull := range pulls {
		if pull.Merged {
			report.MergedPulls++
		}
	}
	return report
}

func Markdown(project string, summary model.MaintainerReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s 维护报告\n\n", markdownInline(project))
	fmt.Fprintf(&b, "## 指标\n\n")
	fmt.Fprintf(&b, "| 指标 | 数量 |\n| --- | ---: |\n")
	fmt.Fprintf(&b, "| Open Issues | %d |\n", summary.OpenIssues)
	fmt.Fprintf(&b, "| 超 30 天未更新 Issues | %d |\n", summary.StaleIssues)
	fmt.Fprintf(&b, "| 安全相关 Issues | %d |\n", summary.SecurityIssues)
	fmt.Fprintf(&b, "| 需要维护者复查 | %d |\n", summary.NeedsReview)
	fmt.Fprintf(&b, "| 已合并 PR | %d |\n\n", summary.MergedPulls)

	fmt.Fprintf(&b, "## 优先处理项\n\n")
	if len(summary.TopSuggestedWork) == 0 {
		fmt.Fprintf(&b, "暂无待处理项。\n")
		return b.String()
	}
	for _, item := range summary.TopSuggestedWork {
		fmt.Fprintf(&b, "- #%d `%s` %s：%s\n", item.Number, markdownInline(item.Priority), markdownInline(item.Title), inlineList(item.Suggested, ", "))
		if len(item.Reasons) > 0 {
			fmt.Fprintf(&b, "  - 原因：%s\n", inlineList(item.Reasons, "；"))
		}
		if len(item.Evidence) > 0 {
			fmt.Fprintf(&b, "  - 证据：%s\n", inlineList(item.Evidence, "；"))
		}
		if item.Action != "" {
			fmt.Fprintf(&b, "  - 建议动作：%s\n", markdownInline(item.Action))
		}
	}
	return b.String()
}

func ReleaseNotes(version string, pulls []model.PullRequest) string {
	groups := map[string][]model.PullRequest{
		"功能": {},
		"修复": {},
		"文档": {},
		"维护": {},
	}
	for _, pull := range pulls {
		if !pull.Merged {
			continue
		}
		groups[groupFor(pull)] = append(groups[groupFor(pull)], pull)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s 发布说明\n\n", markdownInline(version))
	wrote := false
	for _, name := range []string{"功能", "修复", "文档", "维护"} {
		items := groups[name]
		if len(items) == 0 {
			continue
		}
		wrote = true
		sort.SliceStable(items, func(i, j int) bool { return items[i].Number < items[j].Number })
		fmt.Fprintf(&b, "## %s\n\n", name)
		for _, pull := range items {
			fmt.Fprintf(&b, "- %s (#%d) by @%s\n", markdownInline(pull.Title), pull.Number, markdownInline(pull.Author))
		}
		fmt.Fprintln(&b)
	}
	if !wrote {
		fmt.Fprintln(&b, "暂无已合并 PR。")
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func top(results []model.TriageResult, limit int) []model.TriageResult {
	if len(results) <= limit {
		return results
	}
	return results[:limit]
}

func groupFor(pull model.PullRequest) string {
	text := strings.ToLower(pull.Title + " " + strings.Join(pull.Labels, " "))
	switch {
	case strings.Contains(text, "fix") || strings.Contains(text, "bug"):
		return "修复"
	case strings.Contains(text, "doc") || strings.Contains(text, "readme"):
		return "文档"
	case strings.Contains(text, "feature") || strings.Contains(text, "feat") || strings.Contains(text, "add"):
		return "功能"
	default:
		return "维护"
	}
}

func markdownInline(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func inlineList(values []string, sep string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		if item := markdownInline(value); item != "" {
			items = append(items, item)
		}
	}
	return strings.Join(items, sep)
}
