package triage

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

type RuleSet struct {
	Now time.Time
}

func (r RuleSet) Issues(issues []model.Issue) []model.TriageResult {
	results := make([]model.TriageResult, 0, len(issues))
	for _, issue := range issues {
		if strings.EqualFold(issue.State, "closed") {
			continue
		}
		results = append(results, r.Issue(issue))
	}
	sort.SliceStable(results, func(i, j int) bool {
		return priorityRank(results[i].Priority) < priorityRank(results[j].Priority)
	})
	return results
}

func (r RuleSet) Issue(issue model.Issue) model.TriageResult {
	now := r.Now
	if now.IsZero() {
		now = time.Now()
	}

	text := strings.ToLower(issue.Title + "\n" + issue.Body + "\n" + strings.Join(issue.Labels, " "))
	result := model.TriageResult{
		Number:    issue.Number,
		Title:     issue.Title,
		Priority:  "p2",
		StaleDays: int(now.Sub(issue.UpdatedAt).Hours() / 24),
	}

	add := func(label, reason string) {
		if !slices.Contains(result.Suggested, label) {
			result.Suggested = append(result.Suggested, label)
		}
		result.Reasons = append(result.Reasons, reason)
	}

	if hasAny(text, "cve", "exploit", "vulnerability", "security", "token leak", "secret") {
		result.Priority = "p0"
		result.NeedsSecurity = true
		add("security", "包含安全或凭证风险关键词")
	}
	if hasAny(text, "panic", "crash", "data loss", "regression", "cannot start", "deadlock") {
		if result.Priority != "p0" {
			result.Priority = "p1"
		}
		add("bug", "影响稳定性或存在回归风险")
	}
	if hasAny(text, "docs", "readme", "typo", "example", "tutorial") {
		add("documentation", "与文档、示例或教程相关")
	}
	if hasAny(text, "feature", "proposal", "support", "enhancement", "add option") {
		add("enhancement", "包含功能请求或能力扩展诉求")
	}
	if hasAny(text, "test", "coverage", "flaky", "ci", "workflow") {
		add("ci-test", "涉及测试、覆盖率或 CI 稳定性")
	}
	if hasAny(text, "dependency", "deps", "upgrade", "bump", "module") {
		add("dependencies", "涉及依赖升级或模块兼容")
	}
	if result.StaleDays >= 30 {
		result.NeedsReview = true
		add("needs-review", "超过 30 天未更新")
	}
	if len(result.Suggested) == 0 {
		add("needs-triage", "未命中明确分类规则")
	}

	return result
}

func hasAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func priorityRank(priority string) int {
	switch priority {
	case "p0":
		return 0
	case "p1":
		return 1
	default:
		return 2
	}
}
