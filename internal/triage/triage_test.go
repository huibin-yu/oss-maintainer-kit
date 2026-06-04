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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
