package diffreview

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Snippet  string `json:"snippet"`
}

func Review(r io.Reader) ([]Finding, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var findings []Finding
	var file string
	var newLine int
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			file = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@"):
			newLine = parseNewLine(line)
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			added := strings.TrimPrefix(line, "+")
			findings = append(findings, inspect(file, newLine, added)...)
			newLine++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			continue
		default:
			if !strings.HasPrefix(line, "\\") {
				newLine++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read diff: %w", err)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if severityRank(findings[i].Severity) == severityRank(findings[j].Severity) {
			return findings[i].Line < findings[j].Line
		}
		return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
	})
	return findings, nil
}

func Markdown(findings []Finding) string {
	if len(findings) == 0 {
		return "# PR Diff 风险检查\n\n未发现高风险新增代码。\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# PR Diff 风险检查\n\n")
	fmt.Fprintf(&b, "| 严重级别 | 文件 | 行号 | 规则 | 说明 |\n")
	fmt.Fprintf(&b, "| --- | --- | ---: | --- | --- |\n")
	for _, finding := range findings {
		fmt.Fprintf(&b, "| %s | `%s` | %d | `%s` | %s |\n", finding.Severity, finding.File, finding.Line, finding.Rule, finding.Message)
	}
	return b.String()
}

func inspect(file string, line int, code string) []Finding {
	text := strings.ToLower(code)
	var findings []Finding
	add := func(severity, rule, message string) {
		findings = append(findings, Finding{
			File:     file,
			Line:     line,
			Severity: severity,
			Rule:     rule,
			Message:  message,
			Snippet:  strings.TrimSpace(code),
		})
	}

	if strings.Contains(text, "password") || strings.Contains(text, "api_key") || strings.Contains(text, "secret") || strings.Contains(text, "token") {
		if strings.Contains(code, "=") || strings.Contains(code, ":") {
			add("critical", "possible-secret", "新增代码可能包含硬编码凭证或密钥")
		}
	}
	if strings.Contains(text, "exec.command") || strings.Contains(text, "os.system") || strings.Contains(text, "child_process") {
		add("high", "command-execution", "新增代码涉及命令执行，需要检查输入来源和转义")
	}
	if strings.Contains(text, "http://") {
		add("medium", "plain-http", "新增代码使用明文 HTTP，需确认是否允许")
	}
	if strings.Contains(text, "skip_verify") || strings.Contains(text, "insecureskipverify") {
		add("high", "tls-disabled", "新增代码可能关闭 TLS 证书校验")
	}
	if strings.Contains(text, "todo") || strings.Contains(text, "fixme") {
		add("low", "unfinished-work", "新增代码包含未完成标记")
	}
	return findings
}

func parseNewLine(hunk string) int {
	parts := strings.Split(hunk, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "+") {
			part = strings.TrimPrefix(part, "+")
			if idx := strings.Index(part, ","); idx >= 0 {
				part = part[:idx]
			}
			var line int
			_, _ = fmt.Sscanf(part, "%d", &line)
			return line
		}
	}
	return 0
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	default:
		return 3
	}
}
