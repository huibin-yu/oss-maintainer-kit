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
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":[],"created_at":"2026-04-01T00:00:00Z","updated_at":"2026-04-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pulls.json", `[
		{"number":2,"title":"feat: add release check","body":"","state":"closed","author":"bob","labels":[],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-02T00:00:00Z"}
	]`)
	writeTestFile(t, root, "policy.json", `{
		"min_health_score": 90,
		"block_security_issues": true,
		"block_stale_issues": true,
		"max_stale_issues": 0
	}`)

	output := captureStdout(t, func() {
		err := run([]string{
			"release-check",
			"--issues", filepath.Join(root, "issues.json"),
			"--pulls", filepath.Join(root, "pulls.json"),
			"--root", root,
			"--project", "demo",
			"--version", "v1.0.0",
			"--policy", filepath.Join(root, "policy.json"),
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
	if !strings.Contains(strings.Join(result.Blockers, "\n"), "长期未更新 issue 超过阈值") {
		t.Fatalf("missing policy stale blocker: %#v", result.Blockers)
	}
}

func TestRunReleaseCheckRejectsInvalidPolicy(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[]`)
	writeTestFile(t, root, "pulls.json", `[]`)
	writeTestFile(t, root, "policy.json", `{"min_health_score": 120}`)

	err := run([]string{
		"release-check",
		"--issues", filepath.Join(root, "issues.json"),
		"--pulls", filepath.Join(root, "pulls.json"),
		"--root", root,
		"--policy", filepath.Join(root, "policy.json"),
	})
	if err == nil {
		t.Fatal("expected invalid policy error")
	}
	if !strings.Contains(err.Error(), "invalid release policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReleaseCheckFailOnBlockedReturnsError(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":["security"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pulls.json", `[]`)
	writeTestFile(t, root, "policy.json", `{
		"min_health_score": 0,
		"block_security_issues": true,
		"block_stale_issues": false
	}`)

	output := captureStdout(t, func() {
		err := run([]string{
			"release-check",
			"--issues", filepath.Join(root, "issues.json"),
			"--pulls", filepath.Join(root, "pulls.json"),
			"--root", root,
			"--policy", filepath.Join(root, "policy.json"),
			"--fail-on-blocked",
		})
		if err == nil {
			t.Fatal("expected blocked release error")
		}
		if !strings.Contains(err.Error(), "release blocked by policy") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "发布状态：**BLOCKED**") {
		t.Fatalf("missing blocked report: %s", output)
	}
}

func TestRunSBOMWritesSPDXJSON(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", `module github.com/acme/demo

go 1.21
`)

	outputPath := filepath.Join(root, "sbom.spdx.json")
	err := run([]string{
		"sbom",
		"--root", root,
		"--project", "demo",
		"--namespace", "https://example.com/sbom/demo",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"spdxVersion": "SPDX-2.3"`) {
		t.Fatalf("missing SPDX version: %s", data)
	}
	if !strings.Contains(string(data), `"name": "github.com/acme/demo"`) {
		t.Fatalf("missing module package: %s", data)
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
