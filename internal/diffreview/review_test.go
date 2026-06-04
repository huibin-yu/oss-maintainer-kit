package diffreview

import (
	"strings"
	"testing"
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
