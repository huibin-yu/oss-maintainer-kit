package codexplan

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildIncludesRiskDrivenWorkflows(t *testing.T) {
	plan := Build("demo", "https://github.com/acme/demo", model.MaintainerReport{
		SecurityIssues: 1,
		StaleIssues:    2,
	})

	doc := Markdown(plan)
	if !strings.Contains(doc, "security workflow") {
		t.Fatalf("missing security workflow:\n%s", doc)
	}
	if !strings.Contains(doc, "maintenance workflow") {
		t.Fatalf("missing maintenance workflow:\n%s", doc)
	}
}

func TestMarkdownSanitizesInlineText(t *testing.T) {
	doc := Markdown(Plan{
		Project:     "demo\n## injected",
		Repository:  "https://github.com/acme/demo\n- injected repo",
		Workflows:   []string{"PR review\n- injected workflow"},
		APIUseCases: []string{"生成摘要\n- injected api"},
		Guardrails:  []string{"维护者确认\n- injected guardrail"},
	})

	for _, unwanted := range []string{
		"\n## injected",
		"\n- injected repo",
		"\n- injected workflow",
		"\n- injected api",
		"\n- injected guardrail",
	} {
		if strings.Contains(doc, unwanted) {
			t.Fatalf("codex plan contains unsanitized text %q:\n%s", unwanted, doc)
		}
	}
	for _, want := range []string{
		"项目：demo ## injected",
		"仓库：https://github.com/acme/demo - injected repo",
		"- PR review - injected workflow",
		"- 生成摘要 - injected api",
		"- 维护者确认 - injected guardrail",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing sanitized text %q:\n%s", want, doc)
		}
	}
}
