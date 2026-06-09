package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIssuesLoadsKnownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	data := `[
		{
			"number": 1,
			"title": "bug",
			"body": "details",
			"state": "open",
			"author": "alice",
			"labels": ["bug"],
			"created_at": "2026-06-01T00:00:00Z",
			"updated_at": "2026-06-02T00:00:00Z"
		}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	issues, err := Issues(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Number != 1 || issues[0].Labels[0] != "bug" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if want := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC); !issues[0].UpdatedAt.Equal(want) {
		t.Fatalf("updated_at = %s", issues[0].UpdatedAt)
	}
}

func TestIssuesRejectUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	data := `[
		{
			"number": 1,
			"title": "bug",
			"body": "",
			"state": "open",
			"author": "alice",
			"labels": [],
			"created_at": "2026-06-01T00:00:00Z",
			"updated_at": "2026-06-02T00:00:00Z",
			"updated": "2026-06-03T00:00:00Z"
		}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Issues(path)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "updated") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullRequestsRejectUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pulls.json")
	data := `[
		{
			"number": 1,
			"title": "feat: export",
			"body": "",
			"state": "closed",
			"author": "alice",
			"labels": [],
			"merged": true,
			"created_at": "2026-06-01T00:00:00Z",
			"updated_at": "2026-06-02T00:00:00Z",
			"merged_at": "2026-06-02T00:00:00Z",
			"merge_at": "2026-06-02T00:00:00Z"
		}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := PullRequests(path)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "merge_at") {
		t.Fatalf("unexpected error: %v", err)
	}
}
