package triage

import (
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestIssueDetectsSecurityAndPriority(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	result := RuleSet{Now: now}.Issue(model.Issue{
		Number:    7,
		Title:     "token leak in debug logs",
		Body:      "A secret is printed when debug mode is enabled.",
		State:     "open",
		UpdatedAt: now.AddDate(0, 0, -2),
	})

	if result.Priority != "p0" {
		t.Fatalf("priority = %s, want p0", result.Priority)
	}
	if !result.NeedsSecurity {
		t.Fatal("expected security review flag")
	}
	if !contains(result.Suggested, "security") {
		t.Fatalf("labels = %#v, want security", result.Suggested)
	}
	if !contains(result.Evidence, "命中关键词：token leak") {
		t.Fatalf("evidence = %#v", result.Evidence)
	}
	if result.Action == "" || result.Action != "优先安排安全复核，确认影响范围、修复方案和回归测试。" {
		t.Fatalf("action = %q", result.Action)
	}
}

func TestIssuesSkipClosedAndSortByPriority(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	results := RuleSet{Now: now}.Issues([]model.Issue{
		{Number: 1, Title: "docs typo", State: "open", UpdatedAt: now},
		{Number: 2, Title: "panic on startup", State: "open", UpdatedAt: now},
		{Number: 3, Title: "closed security issue", State: "closed", UpdatedAt: now},
	})

	if len(results) != 2 {
		t.Fatalf("len = %d, want 2", len(results))
	}
	if results[0].Number != 2 || results[0].Priority != "p1" {
		t.Fatalf("first result = %#v, want panic issue first", results[0])
	}
}

func TestIssueExplainsStaleMaintenanceWork(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	result := RuleSet{Now: now}.Issue(model.Issue{
		Number:    9,
		Title:     "support new config option",
		State:     "open",
		UpdatedAt: now.AddDate(0, 0, -45),
	})

	if !result.NeedsReview {
		t.Fatal("expected needs review")
	}
	if !contains(result.Evidence, "已停滞 45 天") {
		t.Fatalf("evidence = %#v", result.Evidence)
	}
	if result.Action == "" {
		t.Fatal("missing action")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
