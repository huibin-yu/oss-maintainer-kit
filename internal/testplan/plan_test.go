package testplan

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

func TestBuildCreatesSuggestionsFromDiffFindings(t *testing.T) {
	plan := Build(Input{
		Project: "demo",
		Diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,3 @@
 package main
+const token = "abc"
+http.Get("http://example.com")
`,
		Findings: []diffreview.Finding{
			{File: "main.go", Line: 2, Severity: "critical", Rule: "possible-secret", Message: "secret"},
			{File: "main.go", Line: 3, Severity: "medium", Rule: "plain-http", Message: "http"},
		},
	})

	if len(plan.Files) != 1 || plan.Files[0].Path != "main.go" || plan.Files[0].AddedLines != 2 {
		t.Fatalf("files = %#v", plan.Files)
	}
	if len(plan.Suggestions) < 3 {
		t.Fatalf("suggestions = %#v", plan.Suggestions)
	}
	if plan.Suggestions[0].Priority != "p0" {
		t.Fatalf("first suggestion = %#v", plan.Suggestions[0])
	}
	if !strings.Contains(plan.Prompt, "本地风险发现") || !strings.Contains(plan.Prompt, "必须新增或更新的单元测试") {
		t.Fatalf("prompt = %s", plan.Prompt)
	}
}

func TestMarkdownIncludesPromptAndCommands(t *testing.T) {
	doc := Markdown(Plan{
		Project: "demo",
		Files:   []FileChange{{Path: "main.go", AddedLines: 2}},
		Suggestions: []Suggestion{{
			Area:      "基础回归",
			Priority:  "p1",
			Rationale: "运行测试",
			Command:   "rtk go test ./...",
		}},
		Prompt: "请生成测试计划",
	})
	for _, want := range []string{"# demo 测试建议", "rtk go test ./...", "Codex Prompt", "请生成测试计划"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}
