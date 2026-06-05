package triagecomment

import (
	"fmt"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

const Marker = "<!-- oss-maintainer-kit:triage -->"

func Markdown(results []model.TriageResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", Marker)
	fmt.Fprintf(&b, "## oss-maintainer-kit Issue 分诊建议\n\n")
	if len(results) == 0 {
		fmt.Fprintf(&b, "当前没有需要分诊的 open issue。\n")
		return b.String()
	}

	fmt.Fprintf(&b, "发现 %d 个待处理 issue，其中 p0=%d，p1=%d。\n\n", len(results), count(results, "p0"), count(results, "p1"))
	fmt.Fprintf(&b, "| Issue | 优先级 | 建议标签 | 原因 | 证据 | 建议动作 |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | --- | --- |\n")
	for _, result := range results {
		fmt.Fprintf(
			&b,
			"| #%d %s | `%s` | `%s` | %s | %s | %s |\n",
			result.Number,
			tableCell(result.Title),
			tableCell(result.Priority),
			tableCell(strings.Join(result.Suggested, ",")),
			tableCell(strings.Join(result.Reasons, "<br>")),
			tableCell(strings.Join(result.Evidence, "<br>")),
			tableCell(result.Action),
		)
	}
	fmt.Fprintf(&b, "\n请维护者确认上述 issue 是否需要升级优先级、补充上下文、分配负责人或关闭。\n")
	return b.String()
}

func count(results []model.TriageResult, priority string) int {
	total := 0
	for _, result := range results {
		if result.Priority == priority {
			total++
		}
	}
	return total
}

func tableCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\n", "<br>")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
