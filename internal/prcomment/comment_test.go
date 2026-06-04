package prcomment

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

func TestMarkdownIncludesMarkerAndFindings(t *testing.T) {
	doc := Markdown([]diffreview.Finding{{
		File:     "main.go",
		Line:     3,
		Severity: "critical",
		Rule:     "possible-secret",
		Message:  "secret",
	}})

	if !strings.Contains(doc, "oss-maintainer-kit:review-diff") {
		t.Fatalf("missing marker:\n%s", doc)
	}
	if !strings.Contains(doc, "`main.go:3`") {
		t.Fatalf("missing location:\n%s", doc)
	}
}
