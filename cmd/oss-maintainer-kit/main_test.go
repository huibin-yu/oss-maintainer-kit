package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGitHubCommentCreatesPRComment(t *testing.T) {
	var createdBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/demo/issues/9/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/demo/issues/9/comments":
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			createdBody = string(data)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":301,"html_url":"https://github.com/acme/demo/pull/9#issuecomment-301"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	diffPath := filepath.Join(t.TempDir(), "pr.diff")
	if err := os.WriteFile(diffPath, []byte(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+const token = "sk_live_123456"
`), 0644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"github-comment",
		"--repo", "acme/demo",
		"--pr", "9",
		"--diff", diffPath,
		"--base-url", server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(createdBody, "oss-maintainer-kit:review-diff") {
		t.Fatalf("missing marker in created body: %q", createdBody)
	}
}

func TestRunReleaseCheckJSONReportsBlockedRelease(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pulls.json", `[
		{"number":2,"title":"feat: add release check","body":"","state":"closed","author":"bob","labels":[],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-02T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"release-check",
			"--issues", filepath.Join(root, "issues.json"),
			"--pulls", filepath.Join(root, "pulls.json"),
			"--root", root,
			"--project", "demo",
			"--version", "v1.0.0",
			"--format", "json",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	var result struct {
		Ready    bool     `json:"ready"`
		Blockers []string `json:"blockers"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Ready {
		t.Fatalf("release unexpectedly ready: %s", output)
	}
	if len(result.Blockers) == 0 {
		t.Fatalf("missing blockers: %s", output)
	}
}

func writeTestFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
