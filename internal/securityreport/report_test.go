package securityreport

import (
	"strings"
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildAggregatesSecuritySignals(t *testing.T) {
	report := Build(Input{
		Project: "demo",
		Issues: []model.Issue{
			{
				Number:    7,
				Title:     "security token leak",
				State:     "open",
				UpdatedAt: time.Now(),
			},
			{
				Number:    8,
				Title:     "docs typo",
				State:     "open",
				UpdatedAt: time.Now(),
			},
		},
		Findings: []diffreview.Finding{
			{Severity: "critical", Rule: "possible-secret"},
			{Severity: "high", Rule: "command-execution"},
		},
		Health: health.Summary{
			Score: 88,
			Checks: []health.Check{
				{Name: "Security policy", Passed: false, Path: "SECURITY.md", Recommendation: "添加 SECURITY.md"},
				{Name: "README", Passed: false, Path: "README.md", Recommendation: "补充 README"},
			},
		},
	})

	if report.OpenSecurityIssues != 1 {
		t.Fatalf("open security issues = %d", report.OpenSecurityIssues)
	}
	if !report.Blocked {
		t.Fatal("expected report to be blocked")
	}
	if len(report.Blockers) != 4 {
		t.Fatalf("blockers = %#v", report.Blockers)
	}
	if report.CriticalFindings != 1 || report.HighFindings != 1 {
		t.Fatalf("finding counts critical=%d high=%d", report.CriticalFindings, report.HighFindings)
	}
	if len(report.FailedSecurityChecks) != 1 {
		t.Fatalf("failed security checks = %#v", report.FailedSecurityChecks)
	}
	if len(report.Recommendations) != 4 {
		t.Fatalf("recommendations = %#v", report.Recommendations)
	}
}

func TestBuildUsesEmptySlicesForMachineReadableJSON(t *testing.T) {
	report := Build(Input{Project: "demo"})
	if report.IssueFindings == nil {
		t.Fatal("issue findings should be an empty slice")
	}
	if report.Blockers == nil {
		t.Fatal("blockers should be an empty slice")
	}
	if report.DiffFindings == nil {
		t.Fatal("diff findings should be an empty slice")
	}
	if report.FailedSecurityChecks == nil {
		t.Fatal("failed security checks should be an empty slice")
	}
	if report.Recommendations == nil {
		t.Fatal("recommendations should be an empty slice")
	}
}

func TestMarkdownIncludesActionableSections(t *testing.T) {
	report := Report{
		Project:            "demo",
		OpenSecurityIssues: 1,
		CriticalFindings:   1,
		GovernanceScore:    90,
		IssueFindings: []model.TriageResult{
			{Number: 7, Priority: "p0", Title: "security token leak", Reasons: []string{"包含安全或凭证风险关键词"}},
		},
		DiffFindings: []diffreview.Finding{
			{Severity: "critical", File: "main.go", Line: 10, Rule: "possible-secret", Message: "新增代码可能包含硬编码凭证或密钥"},
		},
		FailedSecurityChecks: []health.Check{
			{Name: "CodeQL workflow", Path: ".github/workflows/codeql.yml", Recommendation: "添加 CodeQL workflow"},
		},
		Recommendations: []string{"阻塞合并：修复 critical diff finding。"},
	}

	md := Markdown(report)
	for _, want := range []string{
		"# demo 安全报告",
		"## 安全摘要",
		"## 阻塞项",
		"security token leak",
		"possible-secret",
		"CodeQL workflow",
		"阻塞合并",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in markdown:\n%s", want, md)
		}
	}
}

func TestMarkdownEscapesDiffFindingTableCells(t *testing.T) {
	report := Report{
		Project: "demo",
		DiffFindings: []diffreview.Finding{
			{
				Severity: "high",
				File:     "internal/a|b.go",
				Line:     10,
				Rule:     "custom|rule",
				Message:  "first line\nsecond | line",
			},
		},
	}

	md := Markdown(report)
	if strings.Contains(md, "internal/a|b.go") || strings.Contains(md, "custom|rule") || strings.Contains(md, "first line\nsecond") {
		t.Fatalf("markdown did not escape table cells:\n%s", md)
	}
	for _, want := range []string{"internal/a\\|b.go", "custom\\|rule", "first line<br>second \\| line"} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing escaped value %q:\n%s", want, md)
		}
	}
}

func TestMarkdownSanitizesInlineText(t *testing.T) {
	report := Report{
		Project:  "demo\n## injected",
		Blockers: []string{"安全阻塞\n- injected blocker"},
		IssueFindings: []model.TriageResult{{
			Number:   7,
			Priority: "p0\n- injected priority",
			Title:    "security token leak\n- injected issue",
			Reasons:  []string{"包含安全风险\n- injected reason"},
		}},
		FailedSecurityChecks: []health.Check{{
			Name:           "CodeQL workflow\n- injected check",
			Path:           ".github/workflows/codeql.yml\n- injected path",
			Recommendation: "添加 CodeQL\n- injected recommendation",
		}},
		Recommendations: []string{"阻塞合并\n- injected action"},
	}

	md := Markdown(report)
	for _, unwanted := range []string{
		"\n## injected",
		"\n- injected blocker",
		"\n- injected priority",
		"\n- injected issue",
		"\n- injected reason",
		"\n- injected check",
		"\n- injected path",
		"\n- injected recommendation",
		"\n- injected action",
	} {
		if strings.Contains(md, unwanted) {
			t.Fatalf("security report contains unsanitized text %q:\n%s", unwanted, md)
		}
	}
	for _, want := range []string{
		"# demo ## injected 安全报告",
		"- 安全阻塞 - injected blocker",
		"#7 `p0 - injected priority` security token leak - injected issue：包含安全风险 - injected reason",
		"- **CodeQL workflow - injected check**（`.github/workflows/codeql.yml - injected path`）：添加 CodeQL - injected recommendation",
		"- 阻塞合并 - injected action",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing sanitized text %q:\n%s", want, md)
		}
	}
}
