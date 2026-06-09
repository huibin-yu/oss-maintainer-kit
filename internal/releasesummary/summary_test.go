package releasesummary

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildGroupsHighlightsAndRisks(t *testing.T) {
	summary := Build(Input{
		Project: "demo",
		Version: "v1.0.0",
		Pulls: []model.PullRequest{
			{Number: 1, Title: "feat: add export", Body: "Adds JSON export for maintainer workflows.", Author: "alice", Merged: true, Labels: []string{"feature"}},
			{Number: 2, Title: "fix panic on startup", Author: "bob", Merged: true, Labels: []string{"bug"}},
			{Number: 3, Title: "security token handling", Author: "carol", Merged: true, Labels: []string{"security"}},
			{Number: 4, Title: "draft change", Author: "dave", Merged: false},
		},
	})

	if len(summary.Highlights) == 0 {
		t.Fatalf("missing highlights: %#v", summary)
	}
	if len(summary.Risks) == 0 || !strings.Contains(summary.Risks[0], "安全相关变更") {
		t.Fatalf("missing risks: %#v", summary.Risks)
	}
	if !strings.Contains(summary.Prompt, "#1 feat: add export") || strings.Contains(summary.Prompt, "draft change") {
		t.Fatalf("unexpected prompt:\n%s", summary.Prompt)
	}
	if !strings.Contains(summary.Prompt, `summary="Adds JSON export for maintainer workflows."`) {
		t.Fatalf("prompt should include PR body evidence:\n%s", summary.Prompt)
	}
}

func TestMarkdownIncludesPrompt(t *testing.T) {
	doc := Markdown(Summary{
		Project:    "demo",
		Version:    "v1.0.0",
		Highlights: []string{"功能：feat (#1)"},
		Risks:      []string{"稳定性修复需要补充回归验证：#2 fix panic"},
		Prompt:     "请生成发布摘要",
	})
	for _, want := range []string{"# demo v1.0.0 发布摘要", "功能：feat", "发布风险", "Codex Prompt", "请生成发布摘要"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func TestBuildSanitizesReleaseSummaryInlineText(t *testing.T) {
	summary := Build(Input{
		Project: "demo",
		Version: "v1.0.0",
		Pulls: []model.PullRequest{
			{
				Number: 1,
				Title:  "feat: add export\n- injected item",
				Body:   "security fix",
				Author: "alice\nbob",
				Merged: true,
				Labels: []string{"feature\nsecurity"},
			},
		},
	})

	doc := Markdown(summary)
	if strings.Contains(doc, "\n- injected item") || strings.Contains(doc, "alice\nbob") {
		t.Fatalf("summary contains unsanitized inline text:\n%s", doc)
	}
	for _, want := range []string{
		"feat: add export - injected item (#1)",
		"安全相关变更需要复核影响范围：#1 feat: add export - injected item",
		"#1 feat: add export - injected item by @alice bob labels=feature security",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing sanitized text %q:\n%s", want, doc)
		}
	}
}
