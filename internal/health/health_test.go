package health

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryScoresRequiredFiles(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"README.md",
		"LICENSE",
		"SECURITY.md",
		"CONTRIBUTING.md",
		"CODE_OF_CONDUCT.md",
		".github/workflows/ci.yml",
		".github/workflows/codeql.yml",
		".github/workflows/review-diff.yml",
		".github/dependabot.yml",
		".github/ISSUE_TEMPLATE/bug_report.md",
		".github/ISSUE_TEMPLATE/feature_request.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"docs/ROADMAP.md",
	}
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("ok"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	summary := Repository(root)
	if summary.Score != 100 {
		t.Fatalf("score = %d, want 100", summary.Score)
	}
}

func TestRepositoryReportsMissingFiles(t *testing.T) {
	summary := Repository(t.TempDir())
	if summary.Score != 0 {
		t.Fatalf("score = %d, want 0", summary.Score)
	}
	if len(summary.Checks) == 0 {
		t.Fatal("expected checks")
	}
}
