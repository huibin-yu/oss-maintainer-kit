package testplan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

type Input struct {
	Project  string
	Diff     string
	Findings []diffreview.Finding
}

type Plan struct {
	Project     string       `json:"project"`
	Files       []FileChange `json:"files"`
	Suggestions []Suggestion `json:"suggestions"`
	Prompt      string       `json:"prompt"`
}

type FileChange struct {
	Path       string `json:"path"`
	AddedLines int    `json:"added_lines"`
}

type Suggestion struct {
	Area      string `json:"area"`
	Priority  string `json:"priority"`
	Rationale string `json:"rationale"`
	Command   string `json:"command,omitempty"`
}

func Build(input Input) Plan {
	files := parseFiles(input.Diff)
	suggestions := suggestions(input.Findings)
	plan := Plan{
		Project:     input.Project,
		Files:       files,
		Suggestions: suggestions,
	}
	plan.Prompt = prompt(plan, input.Findings)
	return plan
}

func Markdown(plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s 测试建议\n\n", markdownInline(plan.Project))
	fmt.Fprintf(&b, "## 变更文件\n\n")
	if len(plan.Files) == 0 {
		fmt.Fprintf(&b, "未识别到变更文件。\n\n")
	} else {
		fmt.Fprintf(&b, "| 文件 | 新增行数 |\n| --- | ---: |\n")
		for _, file := range plan.Files {
			fmt.Fprintf(&b, "| `%s` | %d |\n", markdownCell(file.Path), file.AddedLines)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## 测试建议\n\n")
	for _, suggestion := range plan.Suggestions {
		fmt.Fprintf(&b, "- `%s` **%s**：%s\n", markdownInline(suggestion.Priority), markdownInline(suggestion.Area), markdownInline(suggestion.Rationale))
		if suggestion.Command != "" {
			fmt.Fprintf(&b, "  - 命令：`%s`\n", markdownInline(suggestion.Command))
		}
	}

	fmt.Fprintf(&b, "\n## Codex Prompt\n\n")
	fmt.Fprintf(&b, "```text\n%s\n```\n", plan.Prompt)
	return b.String()
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", "<br>")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

func markdownInline(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func parseFiles(diff string) []FileChange {
	counts := map[string]int{}
	current := ""
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			current = strings.TrimPrefix(line, "+++ b/")
			if _, ok := counts[current]; !ok {
				counts[current] = 0
			}
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			if current != "" {
				counts[current]++
			}
		}
	}
	files := make([]FileChange, 0, len(counts))
	for path, added := range counts {
		files = append(files, FileChange{Path: path, AddedLines: added})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func suggestions(findings []diffreview.Finding) []Suggestion {
	values := []Suggestion{{
		Area:      "基础回归",
		Priority:  "p1",
		Rationale: "先运行全部现有测试，确认变更没有破坏既有 CLI 和内部包行为。",
		Command:   "rtk go test ./...",
	}}
	added := map[string]bool{"基础回归": true}
	for _, finding := range findings {
		item := suggestionFor(finding)
		if item.Area == "" || added[item.Area] {
			continue
		}
		values = append(values, item)
		added[item.Area] = true
	}
	sort.SliceStable(values, func(i, j int) bool {
		return rank(values[i].Priority) < rank(values[j].Priority)
	})
	return values
}

func suggestionFor(finding diffreview.Finding) Suggestion {
	switch finding.Rule {
	case "possible-secret":
		return Suggestion{Area: "凭证泄露防护", Priority: "p0", Rationale: "新增代码可能包含硬编码凭证，需要补充日志脱敏、配置读取和示例数据隔离测试。"}
	case "command-execution":
		return Suggestion{Area: "命令执行边界", Priority: "p0", Rationale: "新增代码涉及命令执行，需要覆盖参数转义、非法输入和上下文取消场景。"}
	case "tls-disabled":
		return Suggestion{Area: "TLS 安全配置", Priority: "p0", Rationale: "新增代码可能关闭证书校验，需要增加安全配置不可默认关闭的测试。"}
	case "plain-http":
		return Suggestion{Area: "网络安全", Priority: "p1", Rationale: "新增代码使用明文 HTTP，需要覆盖允许列表、生产配置拒绝和错误提示。"}
	case "missing-timeout":
		return Suggestion{Area: "网络超时", Priority: "p1", Rationale: "新增 HTTP 调用需要验证 timeout、context cancel 和失败重试行为。"}
	case "unfinished-work":
		return Suggestion{Area: "未完成标记", Priority: "p2", Rationale: "新增代码包含 TODO/FIXME，需要补充测试或在合并前明确跟踪 issue。"}
	default:
		return Suggestion{Area: finding.Rule, Priority: finding.Severity, Rationale: finding.Message}
	}
}

func prompt(plan Plan, findings []diffreview.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "请为开源项目 %s 生成可执行测试计划。\n\n", markdownInline(plan.Project))
	fmt.Fprintf(&b, "变更文件：\n")
	for _, file := range plan.Files {
		fmt.Fprintf(&b, "- %s（新增 %d 行）\n", markdownInline(file.Path), file.AddedLines)
	}
	if len(findings) > 0 {
		fmt.Fprintf(&b, "\n本地风险发现：\n")
		for _, finding := range findings {
			fmt.Fprintf(&b, "- %s %s:%d %s：%s\n", markdownInline(finding.Severity), markdownInline(finding.File), finding.Line, markdownInline(finding.Rule), markdownInline(finding.Message))
		}
	}
	fmt.Fprintf(&b, "\n请输出：\n")
	fmt.Fprintf(&b, "- 必须新增或更新的单元测试\n")
	fmt.Fprintf(&b, "- 必须执行的验证命令\n")
	fmt.Fprintf(&b, "- 高风险路径的边界条件\n")
	fmt.Fprintf(&b, "- 可以暂缓但需要记录的测试缺口\n")
	return b.String()
}

func rank(priority string) int {
	switch priority {
	case "p0", "critical", "high":
		return 0
	case "p1", "medium":
		return 1
	default:
		return 2
	}
}
