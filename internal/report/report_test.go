package report

import (
	"strings"
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
)

func TestMaintainerReportCountsRiskItems(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	summary := Maintainer([]model.Issue{
		{Number: 1, Title: "security token leak", State: "open", UpdatedAt: now.AddDate(0, 0, -40)},
		{Number: 2, Title: "docs typo", State: "open", UpdatedAt: now},
	}, []model.PullRequest{
		{Number: 10, Title: "fix panic", Merged: true},
		{Number: 11, Title: "draft docs", Merged: false},
	}, triage.RuleSet{Now: now})

	if summary.OpenIssues != 2 || summary.StaleIssues != 1 || summary.SecurityIssues != 1 || summary.MergedPulls != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestMaintainerReportTreatsClosedStateCaseInsensitively(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	summary := Maintainer([]model.Issue{
		{Number: 1, Title: "closed upstream", State: "CLOSED", UpdatedAt: now},
		{Number: 2, Title: "active bug", State: "OPEN", UpdatedAt: now},
	}, nil, triage.RuleSet{Now: now})

	if summary.OpenIssues != 1 {
		t.Fatalf("open issues = %d, want 1", summary.OpenIssues)
	}
	if len(summary.TopSuggestedWork) != 1 || summary.TopSuggestedWork[0].Number != 2 {
		t.Fatalf("unexpected suggested work: %#v", summary.TopSuggestedWork)
	}
}

func TestMarkdownIncludesPriorityExplanation(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	summary := Maintainer([]model.Issue{
		{Number: 1, Title: "security token leak", State: "open", UpdatedAt: now.AddDate(0, 0, -40)},
	}, nil, triage.RuleSet{Now: now})

	doc := Markdown("demo", summary)
	for _, want := range []string{
		"原因：包含安全或凭证风险关键词",
		"证据：命中关键词：security",
		"建议动作：优先安排安全复核",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func TestMarkdownSanitizesInlineText(t *testing.T) {
	doc := Markdown("demo\n## injected", model.MaintainerReport{
		TopSuggestedWork: []model.TriageResult{{
			Number:    1,
			Priority:  "p0",
			Title:     "security token leak\n- injected item",
			Suggested: []string{"security\nneeds-review"},
			Reasons:   []string{"包含安全风险\n- injected reason"},
			Evidence:  []string{"命中关键词：token\n- injected evidence"},
			Action:    "优先处理\n- injected action",
		}},
	})

	if strings.Contains(doc, "\n## injected") ||
		strings.Contains(doc, "\n- injected item") ||
		strings.Contains(doc, "\n- injected reason") ||
		strings.Contains(doc, "\n- injected evidence") ||
		strings.Contains(doc, "\n- injected action") {
		t.Fatalf("maintainer report contains unsanitized inline text:\n%s", doc)
	}
	for _, want := range []string{
		"# demo ## injected 维护报告",
		"#1 `p0` security token leak - injected item：security needs-review",
		"原因：包含安全风险 - injected reason",
		"证据：命中关键词：token - injected evidence",
		"建议动作：优先处理 - injected action",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing sanitized text %q:\n%s", want, doc)
		}
	}
}

func TestReleaseNotesGroupsMergedPullRequests(t *testing.T) {
	doc := ReleaseNotes("v0.2.0", []model.PullRequest{
		{Number: 3, Title: "fix parser panic", Author: "alice", Merged: true},
		{Number: 4, Title: "docs update", Author: "bob", Merged: true},
		{Number: 5, Title: "unmerged change", Author: "eve", Merged: false},
	})

	if !strings.Contains(doc, "## 修复") || !strings.Contains(doc, "fix parser panic (#3)") {
		t.Fatalf("missing fix group:\n%s", doc)
	}
	if strings.Contains(doc, "unmerged change") {
		t.Fatalf("release notes included unmerged PR:\n%s", doc)
	}
}

func TestReleaseNotesExplainsWhenNoMergedPullRequests(t *testing.T) {
	doc := ReleaseNotes("v0.2.0", []model.PullRequest{
		{Number: 5, Title: "draft docs", Author: "alice", Merged: false},
	})

	if !strings.Contains(doc, "暂无已合并 PR。") {
		t.Fatalf("missing empty release note message:\n%s", doc)
	}
}

func TestReleaseNotesSanitizesInlineText(t *testing.T) {
	doc := ReleaseNotes("v0.2.0\n## injected", []model.PullRequest{
		{Number: 3, Title: "fix parser panic\n- injected item", Author: "alice\nbob", Merged: true},
	})

	if strings.Contains(doc, "\n- injected item") || strings.Contains(doc, "alice\nbob") || strings.Contains(doc, "\n## injected") {
		t.Fatalf("release notes contain unsanitized inline text:\n%s", doc)
	}
	for _, want := range []string{
		"# v0.2.0 ## injected 发布说明",
		"- fix parser panic - injected item (#3) by @alice bob",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing sanitized text %q:\n%s", want, doc)
		}
	}
}
