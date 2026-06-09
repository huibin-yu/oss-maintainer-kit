package diffreview

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/reviewconfig"
)

func TestReviewFindsRiskyAddedLines(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,4 @@
 package main
+const api_key = "abc123"
+client.Get("http://example.com")
`

	findings, err := Review(strings.NewReader(diff))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(findings), findings)
	}
	if findings[0].Severity != "critical" || findings[0].Rule != "possible-secret" {
		t.Fatalf("first finding = %#v", findings[0])
	}
}

func TestMarkdownHandlesNoFindings(t *testing.T) {
	doc := Markdown(nil)
	if !strings.Contains(doc, "未发现") {
		t.Fatalf("unexpected markdown:\n%s", doc)
	}
}

func TestMarkdownEscapesTableCells(t *testing.T) {
	doc := Markdown([]Finding{{
		File:     "internal/a|b.go",
		Line:     7,
		Severity: "medium",
		Rule:     "custom|rule",
		Message:  "first line\nsecond | line",
	}})

	if strings.Contains(doc, "internal/a|b.go") || strings.Contains(doc, "custom|rule") || strings.Contains(doc, "first line\nsecond") {
		t.Fatalf("markdown did not escape table cells:\n%s", doc)
	}
	for _, want := range []string{"internal/a\\|b.go", "custom\\|rule", "first line<br>second \\| line"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing escaped value %q:\n%s", want, doc)
		}
	}
}

func TestReviewWithConfigFindsCustomRule(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,2 @@
 package main
+dangerousCall()
`
	findings, err := ReviewWithConfig(strings.NewReader(diff), reviewconfig.Config{
		Rules: []reviewconfig.Rule{{
			ID:       "custom-danger",
			Severity: "high",
			Contains: []string{"dangerousCall"},
			Message:  "custom danger",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Rule != "custom-danger" {
		t.Fatalf("unexpected findings: %#v", findings)
	}
}
