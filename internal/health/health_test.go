package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryScoresRequiredFiles(t *testing.T) {
	root := t.TempDir()
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
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
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
	if summary.Checks[0].Recommendation == "" {
		t.Fatalf("missing recommendation: %#v", summary.Checks[0])
	}
}

func TestRepositoryReportsContentQualityFailures(t *testing.T) {
	root := t.TempDir()
	writeHealthFile(t, root, ".github/workflows/ci.yml", "steps:\n  - run: go test ./...\n")
	writeHealthFile(t, root, ".github/workflows/govulncheck.yml", "permissions:\n  contents: read\n")
	writeHealthFile(t, root, ".github/workflows/scorecard.yml", "permissions:\n  contents: read\n")
	writeHealthFile(t, root, ".github/workflows/review-diff.yml", "permissions:\n  contents: read\n")
	writeHealthFile(t, root, ".github/workflows/security-report.yml", "permissions:\n  contents: read\n")
	writeHealthFile(t, root, ".github/workflows/release-check.yml", "permissions:\n  contents: read\n")
	writeHealthFile(t, root, ".github/workflows/release-artifacts.yml", "permissions:\n  contents: read\n")
	writeHealthFile(t, root, ".github/dependabot.yml", "package-ecosystem: \"gomod\"\n")
	writeHealthFile(t, root, ".github/PULL_REQUEST_TEMPLATE.md", "描述变更")

	summary := Repository(root)
	for _, name := range []string{
		"CI least privilege",
		"CI build command",
		"govulncheck package coverage",
		"Scorecard security permission",
		"Scorecard SARIF upload",
		"Review diff SARIF upload",
		"Security report least privilege",
		"Security report live data export",
		"Security report diff gate",
		"Security report artifact",
		"Release check least privilege",
		"Release check live data export",
		"Release check policy command",
		"Release check blocking exit",
		"Release artifacts tag trigger",
		"Release artifacts provenance",
		"Release artifacts SBOM checksums",
		"Dependabot GitHub Actions updates",
		"PR template test checklist",
	} {
		check, ok := findCheck(summary, name)
		if !ok {
			t.Fatalf("missing check %q in %#v", name, summary.Checks)
		}
		if check.Passed {
			t.Fatalf("%s passed unexpectedly: %#v", name, check)
		}
		if check.Recommendation == "" {
			t.Fatalf("%s missing recommendation: %#v", name, check)
		}
	}
}

func TestMarkdownIncludesFailedCheckRecommendations(t *testing.T) {
	doc := Markdown(Summary{
		Score: 50,
		Checks: []Check{
			{Name: "README", Passed: true, Path: "README.md", Message: "说明项目用途"},
			{Name: "CI build command", Passed: false, Path: ".github/workflows/ci.yml", Message: "缺少内容：go build ./cmd/oss-maintainer-kit", Recommendation: "在 CI workflow 中加入 go build ./cmd/oss-maintainer-kit。"},
		},
	})
	for _, want := range []string{
		"## 失败项修复建议",
		"CI build command",
		"在 CI workflow 中加入 go build ./cmd/oss-maintainer-kit。",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func writeHealthFile(t *testing.T, root, file, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func findCheck(summary Summary, name string) (Check, bool) {
	for _, check := range summary.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return Check{}, false
}
