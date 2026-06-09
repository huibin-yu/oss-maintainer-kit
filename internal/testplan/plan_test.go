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

func TestBuildIgnoresDeletedFilesInAddedLineSummary(t *testing.T) {
	plan := Build(Input{
		Project: "demo",
		Diff: `diff --git a/old.go b/old.go
deleted file mode 100644
--- a/old.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package old
-func removed() {}
`,
	})

	if len(plan.Files) != 0 {
		t.Fatalf("deleted file should not be reported as added-line target: %#v", plan.Files)
	}
	if strings.Contains(plan.Prompt, "/dev/null") {
		t.Fatalf("prompt should not include /dev/null:\n%s", plan.Prompt)
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

func TestMarkdownEscapesFileTableCells(t *testing.T) {
	doc := Markdown(Plan{
		Project: "demo",
		Files:   []FileChange{{Path: "internal/a|b.go\nnext", AddedLines: 2}},
		Prompt:  "请生成测试计划",
	})
	if strings.Contains(doc, "internal/a|b.go") || strings.Contains(doc, "internal/a|b.go\nnext") {
		t.Fatalf("markdown did not escape file table cells:\n%s", doc)
	}
	if !strings.Contains(doc, "internal/a\\|b.go<br>next") {
		t.Fatalf("missing escaped file path:\n%s", doc)
	}
}

func TestMarkdownSanitizesInlineText(t *testing.T) {
	doc := Markdown(Plan{
		Project: "demo\n## injected",
		Suggestions: []Suggestion{{
			Area:      "基础回归\n- injected area",
			Priority:  "p1\n- injected priority",
			Rationale: "运行测试\n- injected rationale",
			Command:   "rtk go test ./...\nrm -rf /tmp/example",
		}},
		Prompt: "请生成测试计划",
	})

	for _, unwanted := range []string{
		"\n## injected",
		"\n- injected area",
		"\n- injected priority",
		"\n- injected rationale",
		"\nrm -rf /tmp/example",
	} {
		if strings.Contains(doc, unwanted) {
			t.Fatalf("test plan contains unsanitized text %q:\n%s", unwanted, doc)
		}
	}
	for _, want := range []string{
		"# demo ## injected 测试建议",
		"- `p1 - injected priority` **基础回归 - injected area**：运行测试 - injected rationale",
		"命令：`rtk go test ./... rm -rf /tmp/example`",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing sanitized text %q:\n%s", want, doc)
		}
	}
}

func TestBuildSanitizesPromptInlineText(t *testing.T) {
	plan := Build(Input{
		Project: "demo\n## injected",
		Diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+const token = "abc"
`,
		Findings: []diffreview.Finding{{
			Severity: "critical\n- injected severity",
			File:     "main.go\n- injected file",
			Line:     2,
			Rule:     "possible-secret\n- injected rule",
			Message:  "secret\n- injected message",
		}},
	})

	for _, unwanted := range []string{
		"\n## injected",
		"\n- injected severity",
		"\n- injected file",
		"\n- injected rule",
		"\n- injected message",
	} {
		if strings.Contains(plan.Prompt, unwanted) {
			t.Fatalf("prompt contains unsanitized text %q:\n%s", unwanted, plan.Prompt)
		}
	}
	for _, want := range []string{
		"请为开源项目 demo ## injected 生成可执行测试计划。",
		"- critical - injected severity main.go - injected file:2 possible-secret - injected rule：secret - injected message",
	} {
		if !strings.Contains(plan.Prompt, want) {
			t.Fatalf("missing sanitized prompt text %q:\n%s", want, plan.Prompt)
		}
	}
}
