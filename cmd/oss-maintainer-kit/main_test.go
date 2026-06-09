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
	t.Setenv("GITHUB_TOKEN", "gh_test")

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

func TestRunGitHubCommentRequiresToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing token should fail before calling GitHub API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	diffPath := filepath.Join(t.TempDir(), "pr.diff")
	if err := os.WriteFile(diffPath, []byte(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
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
		"--token-env", "MISSING_GITHUB_TOKEN_FOR_TEST",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "MISSING_GITHUB_TOKEN_FOR_TEST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGitHubCheckRunCreatesCheckRun(t *testing.T) {
	var payload struct {
		Name       string `json:"name"`
		HeadSHA    string `json:"head_sha"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		Output     struct {
			Annotations []struct {
				AnnotationLevel string `json:"annotation_level"`
			} `json:"annotations"`
		} `json:"output"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repos/acme/demo/check-runs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":601,"html_url":"https://github.com/acme/demo/runs/601"}`))
	}))
	defer server.Close()
	t.Setenv("GITHUB_TOKEN", "gh_test")

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

	output := captureStdout(t, func() {
		err := run([]string{
			"github-check-run",
			"--repo", "acme/demo",
			"--sha", "abc123",
			"--diff", diffPath,
			"--base-url", server.URL,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if payload.HeadSHA != "abc123" || payload.Conclusion != "failure" || len(payload.Output.Annotations) != 1 {
		t.Fatalf("payload = %#v", payload)
	}
	if !strings.Contains(output, "https://github.com/acme/demo/runs/601") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRunGitHubCheckRunRequiresToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing token should fail before calling GitHub API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	diffPath := filepath.Join(t.TempDir(), "pr.diff")
	if err := os.WriteFile(diffPath, []byte(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+const token = "sk_live_123456"
`), 0644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"github-check-run",
		"--repo", "acme/demo",
		"--sha", "abc123",
		"--diff", diffPath,
		"--base-url", server.URL,
		"--token-env", "MISSING_GITHUB_TOKEN_FOR_TEST",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "MISSING_GITHUB_TOKEN_FOR_TEST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGitHubReleaseCreatesDraftRelease(t *testing.T) {
	var payload struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		Body       string `json:"body"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/repos/acme/demo/releases" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":701,"html_url":"https://github.com/acme/demo/releases/tag/v1.0.0","tag_name":"v1.0.0"}`))
	}))
	defer server.Close()
	t.Setenv("GITHUB_TOKEN", "gh_test")

	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"},
		{"number":2,"title":"draft change","body":"","state":"open","author":"bob","labels":[],"merged":false,"created_at":"2026-06-02T00:00:00Z","updated_at":"2026-06-02T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"github-release",
			"--repo", "acme/demo",
			"--input", filepath.Join(root, "pulls.json"),
			"--project", "demo",
			"--version", "v1.0.0",
			"--previous-tag", "v0.9.0",
			"--prerelease",
			"--base-url", server.URL,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if payload.TagName != "v1.0.0" || payload.Name != "demo v1.0.0" || !payload.Draft || !payload.Prerelease {
		t.Fatalf("payload = %#v", payload)
	}
	if !strings.Contains(payload.Body, "比较范围：`v0.9.0...v1.0.0`") || !strings.Contains(payload.Body, "feat: add export (#1)") {
		t.Fatalf("unexpected release body: %s", payload.Body)
	}
	if strings.Contains(payload.Body, "draft change") {
		t.Fatalf("unmerged PR leaked into release body: %s", payload.Body)
	}
	if !strings.Contains(output, "https://github.com/acme/demo/releases/tag/v1.0.0") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRunGitHubReleaseDryRunPrintsPayloadWithoutRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry run should not call GitHub API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"github-release",
			"--repo", "acme/demo",
			"--input", filepath.Join(root, "pulls.json"),
			"--project", "demo",
			"--version", "v1.0.0",
			"--previous-tag", "v0.9.0",
			"--base-url", server.URL,
			"--dry-run",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	var payload struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
		Draft   bool   `json:"draft"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TagName != "v1.0.0" || payload.Name != "demo v1.0.0" || !payload.Draft {
		t.Fatalf("payload = %#v", payload)
	}
	if !strings.Contains(payload.Body, "比较范围：`v0.9.0...v1.0.0`") || !strings.Contains(payload.Body, "feat: add export (#1)") {
		t.Fatalf("unexpected release body: %s", payload.Body)
	}
}

func TestRunGitHubReleaseRequiresTokenWhenCreating(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing token should fail before calling GitHub API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"}
	]`)

	err := run([]string{
		"github-release",
		"--repo", "acme/demo",
		"--input", filepath.Join(root, "pulls.json"),
		"--project", "demo",
		"--version", "v1.0.0",
		"--base-url", server.URL,
		"--token-env", "MISSING_GITHUB_TOKEN_FOR_TEST",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "MISSING_GITHUB_TOKEN_FOR_TEST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGitHubTriageCommentCreatesIssueComment(t *testing.T) {
	var createdBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/demo/issues/42/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/demo/issues/42/comments":
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			createdBody = string(data)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":401,"html_url":"https://github.com/acme/demo/issues/42#issuecomment-401"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("GITHUB_TOKEN", "gh_test")

	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":7,"title":"token leak in debug logs","body":"A secret is printed.","state":"open","author":"alice","labels":["bug"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"github-triage-comment",
			"--repo", "acme/demo",
			"--issue", "42",
			"--input", filepath.Join(root, "issues.json"),
			"--base-url", server.URL,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(createdBody, "oss-maintainer-kit:triage") || !strings.Contains(createdBody, "token leak in debug logs") {
		t.Fatalf("unexpected created body: %q", createdBody)
	}
	if !strings.Contains(output, "https://github.com/acme/demo/issues/42#issuecomment-401") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRunGitHubTriageCommentRequiresToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing token should fail before calling GitHub API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":7,"title":"token leak in debug logs","body":"A secret is printed.","state":"open","author":"alice","labels":["bug"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)

	err := run([]string{
		"github-triage-comment",
		"--repo", "acme/demo",
		"--issue", "42",
		"--input", filepath.Join(root, "issues.json"),
		"--base-url", server.URL,
		"--token-env", "MISSING_GITHUB_TOKEN_FOR_TEST",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "MISSING_GITHUB_TOKEN_FOR_TEST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGitHubWriteCommandsValidateArgumentsBeforeReadingInput(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.json")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "github-comment-pr",
			args: []string{"github-comment", "--repo", "acme/demo", "--pr", "0", "--diff", missingPath},
			want: "pull request number is required",
		},
		{
			name: "github-check-run-sha",
			args: []string{"github-check-run", "--repo", "acme/demo", "--sha", "", "--diff", missingPath},
			want: "head sha is required",
		},
		{
			name: "github-triage-comment-issue",
			args: []string{"github-triage-comment", "--repo", "acme/demo", "--issue", "0", "--input", missingPath},
			want: "issue or pull request number is required",
		},
		{
			name: "github-release-repo",
			args: []string{"github-release", "--repo", "invalid", "--input", missingPath},
			want: "expected owner/name",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.args)
			if err == nil {
				t.Fatal("expected argument validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(err.Error(), "missing.json") {
				t.Fatalf("validated too late after reading input: %v", err)
			}
		})
	}
}

func TestRunGitHubWriteCommandsValidateTokenBeforeReadingInput(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.json")
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "github-comment",
			args: []string{"github-comment", "--repo", "acme/demo", "--pr", "9", "--diff", missingPath},
		},
		{
			name: "github-check-run",
			args: []string{"github-check-run", "--repo", "acme/demo", "--sha", "abc123", "--diff", missingPath},
		},
		{
			name: "github-triage-comment",
			args: []string{"github-triage-comment", "--repo", "acme/demo", "--issue", "42", "--input", missingPath},
		},
		{
			name: "github-release",
			args: []string{"github-release", "--repo", "acme/demo", "--input", missingPath},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append(tc.args, "--token-env", "MISSING_GITHUB_TOKEN_BEFORE_INPUT_TEST")
			err := run(args)
			if err == nil {
				t.Fatal("expected missing token error")
			}
			if !strings.Contains(err.Error(), "MISSING_GITHUB_TOKEN_BEFORE_INPUT_TEST") {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(err.Error(), "missing.json") {
				t.Fatalf("validated token too late after reading input: %v", err)
			}
		})
	}
}

func TestRunTriageCommentOutputsMarkdown(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":7,"title":"token leak in debug logs","body":"A secret is printed.","state":"open","author":"alice","labels":["bug"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"triage-comment",
			"--input", filepath.Join(root, "issues.json"),
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, want := range []string{
		"oss-maintainer-kit:triage",
		"Issue 分诊建议",
		"#7 token leak in debug logs",
		"命中关键词：token leak",
		"优先安排安全复核",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q:\n%s", want, output)
		}
	}
}

func TestRunCommandOutputCreatesParentDirectory(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":7,"title":"token leak in debug logs","body":"A secret is printed.","state":"open","author":"alice","labels":["bug"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)
	outputPath := filepath.Join(root, "nested", "triage-comment.md")

	err := run([]string{
		"triage-comment",
		"--input", filepath.Join(root, "issues.json"),
		"--output", outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "oss-maintainer-kit:triage") {
		t.Fatalf("unexpected output: %s", data)
	}
}

func TestRunReviewDiffCheckRunFormat(t *testing.T) {
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

	output := captureStdout(t, func() {
		err := run([]string{
			"review-diff",
			"--diff", diffPath,
			"--format", "check-run",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, want := range []string{
		`"name": "oss-maintainer-kit review-diff"`,
		`"status": "completed"`,
		`"conclusion": "failure"`,
		`"annotations"`,
		`"annotation_level": "failure"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q:\n%s", want, output)
		}
	}
}

func TestRunGitHubExportUsesPaginationFilters(t *testing.T) {
	var state, since, perPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/demo/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		state = r.URL.Query().Get("state")
		since = r.URL.Query().Get("since")
		perPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"bug","body":"","state":"open","user":{"login":"alice"},"labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
		]`))
	}))
	defer server.Close()

	output := captureStdout(t, func() {
		err := run([]string{
			"github-export",
			"--repo", "acme/demo",
			"--kind", "issues",
			"--state", "open",
			"--since", "2026-06-01T00:00:00Z",
			"--limit", "1",
			"--per-page", "25",
			"--base-url", server.URL,
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	if state != "open" || since != "2026-06-01T00:00:00Z" || perPage != "25" {
		t.Fatalf("query state=%s since=%s per_page=%s", state, since, perPage)
	}
	if !strings.Contains(output, `"number": 1`) {
		t.Fatalf("missing exported issue: %s", output)
	}
}

func TestRunGitHubExportRejectsInvalidOptionsBeforeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid options should fail before request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "kind",
			args: []string{"--kind", "milestones"},
			want: `unknown kind "milestones"`,
		},
		{
			name: "api",
			args: []string{"--api", "soap"},
			want: `unknown api "soap"`,
		},
		{
			name: "graphql-kind-before-token",
			args: []string{"--api", "graphql", "--kind", "milestones", "--token-env", "MISSING_GITHUB_TOKEN_FOR_KIND_ORDER"},
			want: `unknown kind "milestones"`,
		},
		{
			name: "state",
			args: []string{"--state", "merged"},
			want: `unknown state "merged"`,
		},
		{
			name: "limit",
			args: []string{"--limit", "0"},
			want: "limit must be greater than 0",
		},
		{
			name: "per-page-low",
			args: []string{"--per-page", "0"},
			want: "per-page must be between 1 and 100",
		},
		{
			name: "per-page-high",
			args: []string{"--per-page", "101"},
			want: "per-page must be between 1 and 100",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{
				"github-export",
				"--repo", "acme/demo",
				"--kind", "issues",
				"--base-url", server.URL,
			}
			args = append(args, tc.args...)
			err := run(args)
			if err == nil {
				t.Fatal("expected invalid option error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunGitHubExportUsesGraphQL(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":{"repository":{"issues":{
				"nodes":[{"number":1,"title":"bug","body":"","state":"OPEN","author":{"login":"alice"},"labels":{"nodes":[]},"createdAt":"2026-06-01T00:00:00Z","updatedAt":"2026-06-01T00:00:00Z","closedAt":null}],
				"pageInfo":{"hasNextPage":false,"endCursor":""}
			}}}
		}`))
	}))
	defer server.Close()
	t.Setenv("GITHUB_TOKEN", "gh_test")

	output := captureStdout(t, func() {
		err := run([]string{
			"github-export",
			"--repo", "acme/demo",
			"--kind", "issues",
			"--api", "graphql",
			"--graphql-url", server.URL + "/graphql",
			"--limit", "1",
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	if path != "/graphql" {
		t.Fatalf("path = %s", path)
	}
	if !strings.Contains(output, `"author": "alice"`) {
		t.Fatalf("missing graphql issue: %s", output)
	}
}

func TestRunGitHubExportGraphQLRequiresToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing token should fail before calling GitHub GraphQL API: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := run([]string{
		"github-export",
		"--repo", "acme/demo",
		"--kind", "issues",
		"--api", "graphql",
		"--graphql-url", server.URL + "/graphql",
		"--token-env", "MISSING_GITHUB_TOKEN_FOR_GRAPHQL_TEST",
		"--limit", "1",
	})
	if err == nil {
		t.Fatal("expected missing token error")
	}
	if !strings.Contains(err.Error(), "MISSING_GITHUB_TOKEN_FOR_GRAPHQL_TEST") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "GitHub API calls") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReleaseSummaryPromptOnly(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"release-summary",
			"--input", filepath.Join(root, "pulls.json"),
			"--project", "demo",
			"--version", "v1.0.0",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "# demo v1.0.0 发布摘要") || !strings.Contains(output, "Codex Prompt") {
		t.Fatalf("unexpected release summary:\n%s", output)
	}
}

func TestRunReleaseDraftOutputsGitHubReleaseJSON(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"},
		{"number":2,"title":"draft change","body":"","state":"open","author":"bob","labels":[],"merged":false,"created_at":"2026-06-02T00:00:00Z","updated_at":"2026-06-02T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"release-draft",
			"--input", filepath.Join(root, "pulls.json"),
			"--project", "demo",
			"--version", "v1.0.0",
			"--previous-tag", "v0.9.0",
			"--prerelease",
			"--format", "json",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	var payload struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		Body       string `json:"body"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TagName != "v1.0.0" || payload.Name != "demo v1.0.0" || !payload.Draft || !payload.Prerelease {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if !strings.Contains(payload.Body, "比较范围：`v0.9.0...v1.0.0`") || !strings.Contains(payload.Body, "feat: add export (#1)") {
		t.Fatalf("unexpected release body: %s", payload.Body)
	}
	if strings.Contains(payload.Body, "draft change") {
		t.Fatalf("unmerged PR leaked into release body: %s", payload.Body)
	}
}

func TestRunReleaseDraftRejectsUnknownFormat(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"}
	]`)

	err := run([]string{
		"release-draft",
		"--input", filepath.Join(root, "pulls.json"),
		"--project", "demo",
		"--version", "v1.0.0",
		"--format", "xml",
	})
	if err == nil {
		t.Fatal("expected unknown format error")
	}
	if !strings.Contains(err.Error(), `unknown format "xml"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCommandsRejectUnknownFormat(t *testing.T) {
	root := t.TempDir()
	writeMinimalHealthyRepo(t, root)
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":["security"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pulls.json", `[
		{"number":2,"title":"feat: add export","body":"","state":"closed","author":"bob","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-02T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pr.diff", `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+fmt.Println("hello")
`)
	historyPath := filepath.Join(root, "health-history.jsonl")
	if err := run([]string{
		"health-snapshot",
		"--root", root,
		"--project", "demo",
		"--ref", "abc123",
		"--history", historyPath,
	}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
	}{
		{name: "triage", args: []string{"triage", "--input", filepath.Join(root, "issues.json"), "--format", "xml"}},
		{name: "review-diff", args: []string{"review-diff", "--diff", filepath.Join(root, "pr.diff"), "--format", "xml"}},
		{name: "test-plan", args: []string{"test-plan", "--diff", filepath.Join(root, "pr.diff"), "--format", "xml"}},
		{name: "release-check", args: []string{"release-check", "--issues", filepath.Join(root, "issues.json"), "--pulls", filepath.Join(root, "pulls.json"), "--root", root, "--format", "xml"}},
		{name: "security-report", args: []string{"security-report", "--issues", filepath.Join(root, "issues.json"), "--root", root, "--format", "xml"}},
		{name: "health", args: []string{"health", "--root", root, "--format", "xml"}},
		{name: "health-trend", args: []string{"health-trend", "--history", historyPath, "--format", "xml"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.args)
			if err == nil {
				t.Fatal("expected unknown format error")
			}
			if !strings.Contains(err.Error(), `unknown format "xml"`) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunReleaseSummaryCallsProvider(t *testing.T) {
	var seenUserPrompt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		seenUserPrompt = req.Messages[len(req.Messages)-1].Content
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"summary ok"}}]}`))
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "provider.json", `{
		"base_url": "`+server.URL+`",
		"model": "summary-model",
		"api_key_env": "SUMMARY_API_KEY"
	}`)
	t.Setenv("SUMMARY_API_KEY", "test-key")

	output := captureStdout(t, func() {
		err := run([]string{
			"release-summary",
			"--input", filepath.Join(root, "pulls.json"),
			"--project", "demo",
			"--version", "v1.0.0",
			"--provider-config", filepath.Join(root, "provider.json"),
			"--prompt-only=false",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "summary ok") {
		t.Fatalf("unexpected provider output: %s", output)
	}
	if !strings.Contains(seenUserPrompt, "请为开源项目 demo 的 v1.0.0 版本生成中文发布摘要") {
		t.Fatalf("unexpected prompt: %s", seenUserPrompt)
	}
}

func TestRunReleaseSummaryRequiresAPIKeyWhenCallingProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing AI key should fail before calling provider: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "pulls.json", `[
		{"number":1,"title":"feat: add export","body":"","state":"closed","author":"alice","labels":["feature"],"merged":true,"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "provider.json", `{
		"base_url": "`+server.URL+`",
		"model": "summary-model",
		"api_key_env": "MISSING_SUMMARY_API_KEY"
	}`)

	err := run([]string{
		"release-summary",
		"--input", filepath.Join(root, "pulls.json"),
		"--provider-config", filepath.Join(root, "provider.json"),
		"--prompt-only=false",
	})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "MISSING_SUMMARY_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAIReviewUsesProviderConfigAndFlagOverrides(t *testing.T) {
	var calls int
	var seenModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		seenModel = req.Model
		if calls == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"review ok"}}]}`))
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "provider.json", `{
		"base_url": "`+server.URL+`",
		"model": "config-model",
		"api_key_env": "CUSTOM_AI_KEY",
		"retries": 1
	}`)
	writeTestFile(t, root, "pr.diff", `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+fmt.Println("hello")
`)
	t.Setenv("CUSTOM_AI_KEY", "test-key")

	output := captureStdout(t, func() {
		err := run([]string{
			"ai-review",
			"--diff", filepath.Join(root, "pr.diff"),
			"--provider-config", filepath.Join(root, "provider.json"),
			"--model", "flag-model",
			"--prompt-only=false",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if calls != 2 {
		t.Fatalf("calls=%d, want retry", calls)
	}
	if seenModel != "flag-model" {
		t.Fatalf("model=%s", seenModel)
	}
	if !strings.Contains(output, "review ok") {
		t.Fatalf("missing review output: %s", output)
	}
}

func TestRunAIReviewPromptOnlyDoesNotRequireProviderConfig(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "provider.json", `{`)
	writeTestFile(t, root, "pr.diff", `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+fmt.Println("hello")
`)

	output := captureStdout(t, func() {
		err := run([]string{
			"ai-review",
			"--diff", filepath.Join(root, "pr.diff"),
			"--provider-config", filepath.Join(root, "provider.json"),
			"--prompt-only",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "你是开源项目维护者的 PR review 助手") || !strings.Contains(output, "+fmt.Println") {
		t.Fatalf("unexpected prompt output: %s", output)
	}
}

func TestRunAIReviewRequiresAPIKeyWhenCallingProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("missing AI key should fail before calling provider: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	root := t.TempDir()
	writeTestFile(t, root, "provider.json", `{
		"base_url": "`+server.URL+`",
		"model": "review-model",
		"api_key_env": "MISSING_REVIEW_API_KEY"
	}`)
	writeTestFile(t, root, "pr.diff", `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+fmt.Println("hello")
`)

	err := run([]string{
		"ai-review",
		"--diff", filepath.Join(root, "pr.diff"),
		"--provider-config", filepath.Join(root, "provider.json"),
		"--prompt-only=false",
	})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	if !strings.Contains(err.Error(), "MISSING_REVIEW_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTestPlanOutputsJSON(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pr.diff", `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,2 @@
 package main
+const token = "abc"
`)

	output := captureStdout(t, func() {
		err := run([]string{
			"test-plan",
			"--diff", filepath.Join(root, "pr.diff"),
			"--project", "demo",
			"--format", "json",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	var result struct {
		Project     string `json:"project"`
		Suggestions []struct {
			Area string `json:"area"`
		} `json:"suggestions"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Project != "demo" || len(result.Suggestions) == 0 || !strings.Contains(result.Prompt, "测试计划") {
		t.Fatalf("unexpected test plan: %s", output)
	}
}

func TestRunTriageTableIncludesAction(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":["security"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"triage",
			"--input", filepath.Join(root, "issues.json"),
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(output, "ACTION") || !strings.Contains(output, "优先安排安全复核") {
		t.Fatalf("missing action column:\n%s", output)
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

func TestRunSecurityReportJSONAggregatesIssuesAndDiff(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":["security"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pr.diff", `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+const token = "sk_live_123456"
`)
	writeTestFile(t, root, "SECURITY.md", "security policy")
	writeTestFile(t, root, "README.md", "readme")

	output := captureStdout(t, func() {
		err := run([]string{
			"security-report",
			"--issues", filepath.Join(root, "issues.json"),
			"--diff", filepath.Join(root, "pr.diff"),
			"--root", root,
			"--project", "demo",
			"--format", "json",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	var result struct {
		Project            string `json:"project"`
		OpenSecurityIssues int    `json:"open_security_issues"`
		CriticalFindings   int    `json:"critical_findings"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Project != "demo" || result.OpenSecurityIssues != 1 || result.CriticalFindings != 1 {
		t.Fatalf("unexpected security report: %s", output)
	}
}

func TestRunSecurityReportWritesMarkdown(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[]`)

	outputPath := filepath.Join(root, "security.md")
	err := run([]string{
		"security-report",
		"--issues", filepath.Join(root, "issues.json"),
		"--root", root,
		"--project", "demo",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# demo 安全报告") {
		t.Fatalf("missing security report title: %s", data)
	}
}

func TestRunSecurityReportFailOnRiskReturnsError(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":["security"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)

	output := captureStdout(t, func() {
		err := run([]string{
			"security-report",
			"--issues", filepath.Join(root, "issues.json"),
			"--root", root,
			"--project", "demo",
			"--fail-on-risk",
		})
		if err == nil {
			t.Fatal("expected security report risk error")
		}
		if !strings.Contains(err.Error(), "security report blocked by risk") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "安全门禁阻塞") {
		t.Fatalf("missing security gate report: %s", output)
	}
}

func TestRunApplicationPackIncludesReadinessEvidence(t *testing.T) {
	root := t.TempDir()
	writeMinimalHealthyRepo(t, root)
	writeTestFile(t, root, "issues.json", `[
		{"number":1,"title":"security token leak","body":"","state":"open","author":"alice","labels":["security"],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
	]`)
	writeTestFile(t, root, "pulls.json", `[]`)
	writeTestFile(t, root, "policy.json", `{
		"min_health_score": 100,
		"block_security_issues": true,
		"block_stale_issues": false
	}`)

	output := captureStdout(t, func() {
		err := run([]string{
			"application-pack",
			"--issues", filepath.Join(root, "issues.json"),
			"--pulls", filepath.Join(root, "pulls.json"),
			"--root", root,
			"--project", "demo",
			"--policy", filepath.Join(root, "policy.json"),
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, want := range []string{
		"发布与安全门禁证据",
		"Release readiness | BLOCKED",
		"Security readiness | BLOCKED",
		"安全相关 issue 未处理",
		"存在 1 个 open 安全 issue",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q:\n%s", want, output)
		}
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

func TestRunHealthSnapshotAndTrend(t *testing.T) {
	root := t.TempDir()
	writeMinimalHealthyRepo(t, root)
	historyPath := filepath.Join(root, "health-history.jsonl")

	output := captureStdout(t, func() {
		err := run([]string{
			"health-snapshot",
			"--root", root,
			"--project", "demo",
			"--ref", "abc123",
			"--history", historyPath,
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(output, `"score": 100`) {
		t.Fatalf("missing snapshot score: %s", output)
	}

	trend := captureStdout(t, func() {
		err := run([]string{
			"health-trend",
			"--history", historyPath,
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{"仓库健康度趋势报告", "项目：demo", "`abc123`"} {
		if !strings.Contains(trend, want) {
			t.Fatalf("missing %q:\n%s", want, trend)
		}
	}
}

func TestRunHealthSnapshotAndTrendRejectInvalidArguments(t *testing.T) {
	root := t.TempDir()
	writeMinimalHealthyRepo(t, root)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "snapshot-project",
			args: []string{"health-snapshot", "--root", root, "--project", "", "--history", filepath.Join(root, "history.jsonl")},
			want: "project is required",
		},
		{
			name: "snapshot-history",
			args: []string{"health-snapshot", "--root", root, "--project", "demo", "--history", ""},
			want: "history path is required",
		},
		{
			name: "trend-history",
			args: []string{"health-trend", "--history", ""},
			want: "history path is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.args)
			if err == nil {
				t.Fatal("expected invalid argument error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func writeMinimalHealthyRepo(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"README.md":                                 "说明项目用途",
		"LICENSE":                                   "MIT",
		"SECURITY.md":                               "security@example.com",
		"CONTRIBUTING.md":                           "go test ./...",
		"CODE_OF_CONDUCT.md":                        "行为准则",
		".github/workflows/ci.yml":                  "permissions:\n  contents: read\nsteps:\n  - run: go test ./...\n  - run: go build ./cmd/oss-maintainer-kit\n",
		".github/workflows/codeql.yml":              "github/codeql-action/init",
		".github/workflows/govulncheck.yml":         "permissions:\n  contents: read\nsteps:\n  - uses: golang/govulncheck-action@v1\n    with:\n      go-version-input: \"1.21\"\n      go-package: ./...\n",
		".github/workflows/scorecard.yml":           "permissions:\n  contents: read\n  security-events: write\n  id-token: write\nsteps:\n  - uses: ossf/scorecard-action@v2\n    with:\n      results_format: sarif\n      publish_results: true\n  - uses: github/codeql-action/upload-sarif@v3\n",
		".github/workflows/review-diff.yml":         "permissions:\n  contents: read\n  security-events: write\n  checks: write\nsteps:\n  - run: go run ./cmd/oss-maintainer-kit github-check-run --repo ${{ github.repository }} --sha ${{ github.event.pull_request.head.sha }} --diff pr.diff\n  - uses: github/codeql-action/upload-sarif@v3\n",
		".github/workflows/security-report.yml":     "permissions:\n  contents: read\n  issues: read\n  pull-requests: read\nsteps:\n  - run: go run ./cmd/oss-maintainer-kit github-export --repo ${{ github.repository }} --kind issues --state open --output security-issues.json\n  - run: git diff origin/main HEAD > security-pr.diff\n  - run: go run ./cmd/oss-maintainer-kit security-report --issues security-issues.json --diff security-pr.diff --root . --fail-on-risk\n  - uses: actions/upload-artifact@v4\n    with:\n      path: |\n        security-report.md\n        security-report.json\n",
		".github/workflows/release-check.yml":       "permissions:\n  contents: read\n  issues: read\n  pull-requests: read\nsteps:\n  - run: go run ./cmd/oss-maintainer-kit github-export --repo ${{ github.repository }} --kind issues --output release-issues.json\n  - run: go run ./cmd/oss-maintainer-kit github-export --repo ${{ github.repository }} --kind pulls --output release-pulls.json\n  - run: go run ./cmd/oss-maintainer-kit release-check --issues release-issues.json --pulls release-pulls.json --root . --policy examples/release-policy.json --fail-on-blocked\n",
		".github/workflows/release-artifacts.yml":   "on:\n  push:\n    tags:\n      - \"v*\"\npermissions:\n  contents: read\n  attestations: write\n  id-token: write\nsteps:\n  - run: go run ./cmd/oss-maintainer-kit sbom --output dist/sbom.spdx.json\n  - run: sha256sum * > checksums.sha256\n  - uses: actions/attest-build-provenance@v2\n    with:\n      subject-path: \"dist/*\"\n  - uses: actions/upload-artifact@v4\n",
		".github/dependabot.yml":                    "package-ecosystem: \"gomod\"\npackage-ecosystem: \"github-actions\"\n",
		".github/ISSUE_TEMPLATE/bug_report.md":      "bug",
		".github/ISSUE_TEMPLATE/feature_request.md": "feature",
		".github/PULL_REQUEST_TEMPLATE.md":          "测试\n风险",
		"docs/ROADMAP.md":                           "路线图",
	}
	for file, body := range files {
		writeTestFile(t, root, file, body)
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
