package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Summary struct {
	Score  int     `json:"score"`
	Checks []Check `json:"checks"`
}

func Repository(root string) Summary {
	checks := []Check{
		fileCheck(root, "README", "README.md", "说明项目用途、安装方式和使用方式"),
		fileCheck(root, "License", "LICENSE", "明确开源许可证"),
		fileCheck(root, "Security policy", "SECURITY.md", "提供安全问题报告方式"),
		fileCheck(root, "Contributing guide", "CONTRIBUTING.md", "提供贡献流程"),
		fileCheck(root, "Code of conduct", "CODE_OF_CONDUCT.md", "提供协作行为准则"),
		fileCheck(root, "CI workflow", ".github/workflows/ci.yml", "提供自动测试和构建"),
		fileCheck(root, "CodeQL workflow", ".github/workflows/codeql.yml", "提供 CodeQL 静态分析"),
		fileCheck(root, "Review diff workflow", ".github/workflows/review-diff.yml", "提供 PR diff SARIF 扫描"),
		fileCheck(root, "Dependabot config", ".github/dependabot.yml", "提供依赖更新自动化"),
		fileCheck(root, "Bug issue template", ".github/ISSUE_TEMPLATE/bug_report.md", "提供 Bug 报告模板"),
		fileCheck(root, "Feature issue template", ".github/ISSUE_TEMPLATE/feature_request.md", "提供功能请求模板"),
		fileCheck(root, "Pull request template", ".github/PULL_REQUEST_TEMPLATE.md", "提供 PR 说明模板"),
		fileCheck(root, "Roadmap", "docs/ROADMAP.md", "提供后续维护路线图"),
	}

	passed := 0
	for _, check := range checks {
		if check.Passed {
			passed++
		}
	}
	return Summary{
		Score:  int(float64(passed) / float64(len(checks)) * 100),
		Checks: checks,
	}
}

func Markdown(summary Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 开源仓库健康度报告\n\n")
	fmt.Fprintf(&b, "健康度评分：**%d/100**\n\n", summary.Score)
	fmt.Fprintf(&b, "| 检查项 | 状态 | 路径 | 说明 |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- |\n")
	for _, check := range summary.Checks {
		status := "FAIL"
		if check.Passed {
			status = "PASS"
		}
		fmt.Fprintf(&b, "| %s | %s | `%s` | %s |\n", check.Name, status, check.Path, check.Message)
	}
	return b.String()
}

func fileCheck(root, name, path, message string) Check {
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Stat(fullPath)
	if err != nil {
		return Check{Name: name, Passed: false, Path: path, Message: "缺失：" + message}
	}
	if info.IsDir() {
		return Check{Name: name, Passed: false, Path: path, Message: "路径是目录，不是文件"}
	}
	if info.Size() == 0 {
		return Check{Name: name, Passed: false, Path: path, Message: "文件为空"}
	}
	return Check{Name: name, Passed: true, Path: path, Message: message}
}
