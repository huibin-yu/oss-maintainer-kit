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
		".github/workflows/review-diff.yml":         "permissions:\n  contents: read\n  security-events: write\nsteps:\n  - uses: github/codeql-action/upload-sarif@v3\n",
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
