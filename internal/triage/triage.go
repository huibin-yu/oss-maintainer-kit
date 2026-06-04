package triage

import (
	"slices"
	"sort"
	"strconv"
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

	add := func(label, reason, evidence string) {
		if !slices.Contains(result.Suggested, label) {
			result.Suggested = append(result.Suggested, label)
		}
		result.Reasons = append(result.Reasons, reason)
		if evidence != "" && !slices.Contains(result.Evidence, evidence) {
			result.Evidence = append(result.Evidence, evidence)
		}
	}

	if matched := firstMatch(text, "cve", "exploit", "vulnerability", "security", "token leak", "secret"); matched != "" {
		result.Priority = "p0"
		result.NeedsSecurity = true
		add("security", "包含安全或凭证风险关键词", "命中关键词："+matched)
	}
	if matched := firstMatch(text, "panic", "crash", "data loss", "regression", "cannot start", "deadlock"); matched != "" {
		if result.Priority != "p0" {
			result.Priority = "p1"
		}
		add("bug", "影响稳定性或存在回归风险", "命中关键词："+matched)
	}
	if matched := firstMatch(text, "docs", "readme", "typo", "example", "tutorial"); matched != "" {
		add("documentation", "与文档、示例或教程相关", "命中关键词："+matched)
	}
	if matched := firstMatch(text, "feature", "proposal", "support", "enhancement", "add option"); matched != "" {
		add("enhancement", "包含功能请求或能力扩展诉求", "命中关键词："+matched)
	}
	if matched := firstMatch(text, "test", "coverage", "flaky", "ci", "workflow"); matched != "" {
		add("ci-test", "涉及测试、覆盖率或 CI 稳定性", "命中关键词："+matched)
	}
	if matched := firstMatch(text, "dependency", "deps", "upgrade", "bump", "module"); matched != "" {
		add("dependencies", "涉及依赖升级或模块兼容", "命中关键词："+matched)
	}
	if result.StaleDays >= 30 {
		result.NeedsReview = true
		add("needs-review", "超过 30 天未更新", "已停滞 "+formatDays(result.StaleDays))
	}
	if len(result.Suggested) == 0 {
		add("needs-triage", "未命中明确分类规则", "未命中内置关键词")
	}
	result.Action = actionFor(result)

	return result
}

func firstMatch(text string, words ...string) string {
	for _, word := range words {
		if strings.Contains(text, word) {
			return word
		}
	}
	return ""
}

func actionFor(result model.TriageResult) string {
	switch {
	case result.NeedsSecurity:
		return "优先安排安全复核，确认影响范围、修复方案和回归测试。"
	case result.Priority == "p1":
		return "优先复现并补充失败用例，确认是否阻塞发布。"
	case result.NeedsReview:
		return "维护者复查当前状态，决定关闭、降级或升级优先级。"
	case slices.Contains(result.Suggested, "dependencies"):
		return "确认依赖升级影响，补充兼容性验证。"
	case slices.Contains(result.Suggested, "ci-test"):
		return "复查 CI 日志或测试覆盖，补充稳定性验证。"
	case slices.Contains(result.Suggested, "documentation"):
		return "更新文档或示例，并确认 README/教程与当前行为一致。"
	case slices.Contains(result.Suggested, "enhancement"):
		return "补充需求边界、验收标准和可行实现路径。"
	default:
		return "补充分类信息和复现上下文后再安排处理。"
	}
}

func formatDays(days int) string {
	return strconv.Itoa(days) + " 天"
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
