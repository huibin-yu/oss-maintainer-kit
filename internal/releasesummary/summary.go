package releasesummary

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

type Input struct {
	Project string
	Version string
	Pulls   []model.PullRequest
}

type Summary struct {
	Project    string   `json:"project"`
	Version    string   `json:"version"`
	Highlights []string `json:"highlights"`
	Risks      []string `json:"risks"`
	Prompt     string   `json:"prompt"`
}

func Build(input Input) Summary {
	merged := mergedPulls(input.Pulls)
	summary := Summary{
		Project:    input.Project,
		Version:    input.Version,
		Highlights: highlights(merged),
		Risks:      risks(merged),
	}
	summary.Prompt = prompt(summary, merged)
	return summary
}

func Markdown(summary Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s 发布摘要\n\n", summary.Project, summary.Version)
	fmt.Fprintf(&b, "## 重点变化\n\n")
	if len(summary.Highlights) == 0 {
		fmt.Fprintf(&b, "暂无已合并 PR 可用于生成摘要。\n\n")
	} else {
		for _, item := range summary.Highlights {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## 发布风险\n\n")
	if len(summary.Risks) == 0 {
		fmt.Fprintf(&b, "未识别到明确发布风险。\n\n")
	} else {
		for _, item := range summary.Risks {
			fmt.Fprintf(&b, "- %s\n", item)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## Codex Prompt\n\n")
	fmt.Fprintf(&b, "```text\n%s\n```\n", summary.Prompt)
	return b.String()
}

func mergedPulls(pulls []model.PullRequest) []model.PullRequest {
	var merged []model.PullRequest
	for _, pull := range pulls {
		if pull.Merged {
			merged = append(merged, pull)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].Number < merged[j].Number })
	return merged
}

func highlights(pulls []model.PullRequest) []string {
	if len(pulls) == 0 {
		return []string{}
	}
	groups := map[string][]model.PullRequest{}
	for _, pull := range pulls {
		groups[groupFor(pull)] = append(groups[groupFor(pull)], pull)
	}
	var values []string
	for _, name := range []string{"功能", "修复", "文档", "维护"} {
		items := groups[name]
		if len(items) == 0 {
			continue
		}
		titles := make([]string, 0, len(items))
		for _, item := range items {
			titles = append(titles, fmt.Sprintf("%s (#%d)", item.Title, item.Number))
		}
		values = append(values, fmt.Sprintf("%s：%s", name, strings.Join(titles, "；")))
	}
	return values
}

func risks(pulls []model.PullRequest) []string {
	var security []string
	var compatibility []string
	var stability []string
	for _, pull := range pulls {
		text := strings.ToLower(pull.Title + " " + pull.Body + " " + strings.Join(pull.Labels, " "))
		switch {
		case hasAny(text, "security", "token", "secret", "cve"):
			security = append(security, fmt.Sprintf("安全相关变更需要复核影响范围：#%d %s", pull.Number, pull.Title))
		case hasAny(text, "breaking", "migration", "config", "api"):
			compatibility = append(compatibility, fmt.Sprintf("兼容性或配置变更需要发布说明中特别标注：#%d %s", pull.Number, pull.Title))
		case hasAny(text, "panic", "crash", "regression"):
			stability = append(stability, fmt.Sprintf("稳定性修复需要补充回归验证：#%d %s", pull.Number, pull.Title))
		}
	}
	values := append([]string{}, security...)
	values = append(values, compatibility...)
	values = append(values, stability...)
	return values
}

func prompt(summary Summary, pulls []model.PullRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "请为开源项目 %s 的 %s 版本生成中文发布摘要。\n\n", summary.Project, summary.Version)
	fmt.Fprintf(&b, "要求：\n")
	fmt.Fprintf(&b, "- 面向用户和维护者，语言简洁。\n")
	fmt.Fprintf(&b, "- 按功能、修复、文档、维护分组。\n")
	fmt.Fprintf(&b, "- 标注安全、兼容性或稳定性风险。\n")
	fmt.Fprintf(&b, "- 不要编造未在 PR 中出现的变化。\n\n")
	fmt.Fprintf(&b, "已合并 PR：\n")
	if len(pulls) == 0 {
		fmt.Fprintf(&b, "- 暂无\n")
	} else {
		for _, pull := range pulls {
			fmt.Fprintf(&b, "- #%d %s by @%s labels=%s", pull.Number, pull.Title, pull.Author, strings.Join(pull.Labels, ","))
			if pull.MergedAt != nil {
				fmt.Fprintf(&b, " merged_at=%s", pull.MergedAt.Format("2006-01-02"))
			}
			if body := excerpt(pull.Body, 180); body != "" {
				fmt.Fprintf(&b, " summary=%q", body)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func groupFor(pull model.PullRequest) string {
	text := strings.ToLower(pull.Title + " " + strings.Join(pull.Labels, " "))
	switch {
	case hasAny(text, "fix", "bug", "panic", "crash"):
		return "修复"
	case hasAny(text, "doc", "readme", "example"):
		return "文档"
	case hasAny(text, "feat", "feature", "add", "support"):
		return "功能"
	default:
		return "维护"
	}
}

func hasAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func excerpt(text string, limit int) string {
	words := strings.Fields(text)
	if len(words) == 0 || limit <= 0 {
		return ""
	}
	value := strings.Join(words, " ")
	if len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit]) + "..."
}
