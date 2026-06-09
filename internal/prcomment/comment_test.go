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

func TestMarkdownEscapesTableCells(t *testing.T) {
	doc := Markdown([]diffreview.Finding{{
		File:     "internal/a|b.go",
		Line:     3,
		Severity: "high",
		Rule:     "custom|rule",
		Message:  "first line\nsecond | line",
	}})

	if strings.Contains(doc, "internal/a|b.go") || strings.Contains(doc, "custom|rule") || strings.Contains(doc, "first line\nsecond") {
		t.Fatalf("markdown did not escape table cells:\n%s", doc)
	}
	for _, want := range []string{"internal/a\\|b.go:3", "custom\\|rule", "first line<br>second \\| line"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing escaped value %q:\n%s", want, doc)
		}
	}
}
