package applicationpack

import (
	"strings"
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
	"github.com/yuhuibin/oss-maintainer-kit/internal/releasecheck"
	"github.com/yuhuibin/oss-maintainer-kit/internal/securityreport"
)

func TestMarkdownIncludesApplicationEvidence(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	doc := Markdown(Build(Input{
		Project:    "oss-maintainer-kit",
		Repository: "https://github.com/acme/oss-maintainer-kit",
		Issues: []model.Issue{
			{Number: 1, Title: "security token leak", State: "open", UpdatedAt: now.AddDate(0, 0, -40)},
			{Number: 2, Title: "docs typo", State: "open", UpdatedAt: now},
		},
		Pulls: []model.PullRequest{
			{Number: 10, Title: "feat: add review diff", Author: "alice", Merged: true},
		},
		Health: health.Summary{
			Score: 84,
			Checks: []health.Check{
				{Name: "README", Passed: true, Path: "README.md", Message: "说明项目用途"},
				{Name: "CI workflow", Passed: false, Path: ".github/workflows/ci.yml", Message: "缺失：提供自动测试和构建", Recommendation: "添加 CI workflow。"},
			},
		},
		Release: releasecheck.Result{
			Project:  "oss-maintainer-kit",
			Version:  "v0.1.0",
			Ready:    false,
			Blockers: []string{"安全相关 issue 未处理：1 个"},
		},
		Security: securityreport.Report{
			Project:            "oss-maintainer-kit",
			Blocked:            true,
			Blockers:           []string{"存在 1 个 open 安全 issue"},
			OpenSecurityIssues: 1,
		},
	}))

	for _, want := range []string{
		"# Codex for OSS 申请证据包",
		"https://github.com/acme/oss-maintainer-kit",
		"Project description",
		"PR review",
		"issue triage",
		"release workflow",
		"security workflow",
		"健康度评分：**84/100**",
		"需要补齐的申请前事项",
		"建议：添加 CI workflow。",
		"发布与安全门禁证据",
		"Release readiness | BLOCKED",
		"Security readiness | BLOCKED",
		"发布阻塞项",
		"安全阻塞项",
		"rtk go test ./...",
		"health-trend --history health-history.jsonl",
		"sbom --root . --project oss-maintainer-kit",
		"security-report --issues examples/issues.json",
		"release-check --issues examples/issues.json",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func TestMarkdownUsesFallbackRepositoryText(t *testing.T) {
	doc := Markdown(Build(Input{Project: "demo"}))
	if !strings.Contains(doc, "发布后填写公开 GitHub 仓库地址") {
		t.Fatalf("missing repository fallback:\n%s", doc)
	}
}
