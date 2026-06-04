package prcomment

import (
	"fmt"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

func Markdown(findings []diffreview.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- oss-maintainer-kit:review-diff -->\n")
	fmt.Fprintf(&b, "## oss-maintainer-kit PR 风险检查\n\n")
	if len(findings) == 0 {
		fmt.Fprintf(&b, "未发现需要阻塞合并的新增代码风险。\n")
		return b.String()
	}

	critical := count(findings, "critical")
	high := count(findings, "high")
	fmt.Fprintf(&b, "发现 %d 个风险项，其中 critical=%d，high=%d。\n\n", len(findings), critical, high)
	fmt.Fprintf(&b, "| 严重级别 | 位置 | 规则 | 说明 |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- |\n")
	for _, finding := range findings {
		fmt.Fprintf(&b, "| %s | `%s:%d` | `%s` | %s |\n", finding.Severity, finding.File, finding.Line, finding.Rule, finding.Message)
	}
	fmt.Fprintf(&b, "\n请维护者确认上述项是否需要修复、豁免或补充测试。\n")
	return b.String()
}

func count(findings []diffreview.Finding, severity string) int {
	total := 0
	for _, finding := range findings {
		if finding.Severity == severity {
			total++
		}
	}
	return total
}
