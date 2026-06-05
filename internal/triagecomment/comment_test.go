package triagecomment

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestMarkdownIncludesMarkerAndTriageDetails(t *testing.T) {
	doc := Markdown([]model.TriageResult{{
		Number:    7,
		Title:     "token leak in debug logs",
		Priority:  "p0",
		Suggested: []string{"security", "needs-review"},
		Reasons:   []string{"包含安全或凭证风险关键词"},
		Evidence:  []string{"命中关键词：token leak"},
		Action:    "优先安排安全复核，确认影响范围、修复方案和回归测试。",
	}})

	for _, want := range []string{
		"oss-maintainer-kit:triage",
		"## oss-maintainer-kit Issue 分诊建议",
		"发现 1 个待处理 issue，其中 p0=1，p1=0。",
		"#7",
		"token leak in debug logs",
		"`security,needs-review`",
		"包含安全或凭证风险关键词",
		"命中关键词：token leak",
		"优先安排安全复核",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func TestMarkdownHandlesEmptyTriage(t *testing.T) {
	doc := Markdown(nil)

	if !strings.Contains(doc, "oss-maintainer-kit:triage") {
		t.Fatalf("missing marker:\n%s", doc)
	}
	if !strings.Contains(doc, "当前没有需要分诊的 open issue。") {
		t.Fatalf("missing empty message:\n%s", doc)
	}
}
