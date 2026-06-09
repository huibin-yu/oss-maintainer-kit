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
