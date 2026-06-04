package securityreport

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
)

type Input struct {
	Project  string
	Issues   []model.Issue
	Findings []diffreview.Finding
	Health   health.Summary
}

type Report struct {
	Project              string               `json:"project"`
	OpenSecurityIssues   int                  `json:"open_security_issues"`
	CriticalFindings     int                  `json:"critical_findings"`
	HighFindings         int                  `json:"high_findings"`
	GovernanceScore      int                  `json:"governance_score"`
	FailedSecurityChecks []health.Check       `json:"failed_security_checks"`
	IssueFindings        []model.TriageResult `json:"issue_findings"`
	DiffFindings         []diffreview.Finding `json:"diff_findings"`
	Recommendations      []string             `json:"recommendations"`
}

func Build(input Input) Report {
	issueFindings := securityIssues(input.Issues)
	diffFindings := input.Findings
	if issueFindings == nil {
		issueFindings = []model.TriageResult{}
	}
	if diffFindings == nil {
		diffFindings = []diffreview.Finding{}
	}
	report := Report{
		Project:              input.Project,
		OpenSecurityIssues:   len(issueFindings),
		CriticalFindings:     countSeverity(diffFindings, "critical"),
		HighFindings:         countSeverity(diffFindings, "high"),
		GovernanceScore:      input.Health.Score,
		FailedSecurityChecks: failedSecurityChecks(input.Health.Checks),
		IssueFindings:        issueFindings,
		DiffFindings:         diffFindings,
	}
	report.Recommendations = recommendations(report)
	if report.FailedSecurityChecks == nil {
		report.FailedSecurityChecks = []health.Check{}
	}
	if report.Recommendations == nil {
		report.Recommendations = []string{}
	}
	return report
}

func Markdown(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s 安全报告\n\n", report.Project)
	fmt.Fprintf(&b, "## 安全摘要\n\n")
	fmt.Fprintf(&b, "| 指标 | 数量 |\n| --- | ---: |\n")
	fmt.Fprintf(&b, "| Open 安全 Issues | %d |\n", report.OpenSecurityIssues)
	fmt.Fprintf(&b, "| Critical Diff Findings | %d |\n", report.CriticalFindings)
	fmt.Fprintf(&b, "| High Diff Findings | %d |\n", report.HighFindings)
	fmt.Fprintf(&b, "| 仓库治理评分 | %d/100 |\n\n", report.GovernanceScore)

	fmt.Fprintf(&b, "## 安全 Issues\n\n")
	if len(report.IssueFindings) == 0 {
		fmt.Fprintf(&b, "未发现 open 安全 issue。\n\n")
	} else {
		for _, item := range report.IssueFindings {
			fmt.Fprintf(&b, "- #%d `%s` %s：%s\n", item.Number, item.Priority, item.Title, strings.Join(item.Reasons, "；"))
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## Diff 风险发现\n\n")
	if len(report.DiffFindings) == 0 {
		fmt.Fprintf(&b, "未发现新增代码安全风险。\n\n")
	} else {
		fmt.Fprintf(&b, "| 严重级别 | 文件 | 行号 | 规则 | 说明 |\n")
		fmt.Fprintf(&b, "| --- | --- | ---: | --- | --- |\n")
		for _, finding := range report.DiffFindings {
			fmt.Fprintf(&b, "| %s | `%s` | %d | `%s` | %s |\n", finding.Severity, finding.File, finding.Line, finding.Rule, finding.Message)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## 治理缺口\n\n")
	if len(report.FailedSecurityChecks) == 0 {
		fmt.Fprintf(&b, "未发现安全治理检查失败项。\n\n")
	} else {
		for _, check := range report.FailedSecurityChecks {
			fmt.Fprintf(&b, "- **%s**（`%s`）：%s\n", check.Name, check.Path, check.Recommendation)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## 建议动作\n\n")
	if len(report.Recommendations) == 0 {
		fmt.Fprintf(&b, "- 当前安全风险处于可发布范围，继续保留 CI、漏洞扫描和 review-diff 门禁。\n")
		return b.String()
	}
	for _, recommendation := range report.Recommendations {
		fmt.Fprintf(&b, "- %s\n", recommendation)
	}
	return b.String()
}

func securityIssues(issues []model.Issue) []model.TriageResult {
	results := triage.RuleSet{}.Issues(issues)
	var filtered []model.TriageResult
	for _, result := range results {
		if result.NeedsSecurity {
			filtered = append(filtered, result)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Number < filtered[j].Number
	})
	return filtered
}

func countSeverity(findings []diffreview.Finding, severity string) int {
	var count int
	for _, finding := range findings {
		if finding.Severity == severity {
			count++
		}
	}
	return count
}

func failedSecurityChecks(checks []health.Check) []health.Check {
	keywords := []string{
		"security",
		"codeql",
		"govulncheck",
		"scorecard",
		"sarif",
		"dependabot",
		"release artifacts",
		"release check",
	}
	var failed []health.Check
	for _, check := range checks {
		if check.Passed {
			continue
		}
		text := strings.ToLower(check.Name + " " + check.Path + " " + check.Message)
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				failed = append(failed, check)
				break
			}
		}
	}
	return failed
}

func recommendations(report Report) []string {
	var items []string
	if report.OpenSecurityIssues > 0 {
		items = append(items, fmt.Sprintf("优先处理 %d 个 open 安全 issue，发布前关闭或给出可审计的风险接受说明。", report.OpenSecurityIssues))
	}
	if report.CriticalFindings > 0 {
		items = append(items, fmt.Sprintf("阻塞合并：修复 %d 个 critical diff finding，重点检查硬编码凭证和密钥泄露。", report.CriticalFindings))
	}
	if report.HighFindings > 0 {
		items = append(items, fmt.Sprintf("发布前复查 %d 个 high diff finding，确认命令执行、TLS 配置等高风险变更已隔离。", report.HighFindings))
	}
	if len(report.FailedSecurityChecks) > 0 {
		items = append(items, fmt.Sprintf("补齐 %d 个安全治理检查失败项，优先恢复 Security policy、CodeQL、govulncheck、Scorecard、Dependabot 和 SARIF 上传。", len(report.FailedSecurityChecks)))
	}
	return items
}
