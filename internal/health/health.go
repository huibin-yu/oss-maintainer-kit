package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Check struct {
	Name           string `json:"name"`
	Passed         bool   `json:"passed"`
	Path           string `json:"path"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
}

type Summary struct {
	Score  int     `json:"score"`
	Checks []Check `json:"checks"`
}

func Repository(root string) Summary {
	checks := []Check{
		fileCheck(root, "README", "README.md", "说明项目用途、安装方式和使用方式", "补充 README.md，至少包含项目简介、安装方式、使用方法、开发方式和测试方法。"),
		fileCheck(root, "License", "LICENSE", "明确开源许可证", "添加 LICENSE 文件，并选择 MIT、Apache-2.0 等明确开源许可证。"),
		fileCheck(root, "Security policy", "SECURITY.md", "提供安全问题报告方式", "添加 SECURITY.md，说明安全漏洞报告渠道、响应预期和不公开披露要求。"),
		fileCheck(root, "Contributing guide", "CONTRIBUTING.md", "提供贡献流程", "添加 CONTRIBUTING.md，说明开发环境、测试命令、PR 流程和 review 要求。"),
		fileCheck(root, "Code of conduct", "CODE_OF_CONDUCT.md", "提供协作行为准则", "添加 CODE_OF_CONDUCT.md，明确社区协作行为准则。"),
		fileCheck(root, "CI workflow", ".github/workflows/ci.yml", "提供自动测试和构建", "添加 .github/workflows/ci.yml，在 push 和 pull_request 上运行测试与构建。"),
		fileCheck(root, "CodeQL workflow", ".github/workflows/codeql.yml", "提供 CodeQL 静态分析", "添加 CodeQL workflow，并启用 Go 语言分析和 security-events 写权限。"),
		fileCheck(root, "govulncheck workflow", ".github/workflows/govulncheck.yml", "提供 Go 漏洞扫描", "添加 govulncheck workflow，定期扫描 ./... 的 Go 漏洞。"),
		fileCheck(root, "Scorecard workflow", ".github/workflows/scorecard.yml", "提供 OpenSSF Scorecard 安全治理评分", "添加 OpenSSF Scorecard workflow，输出 SARIF 并发布结果。"),
		fileCheck(root, "Review diff workflow", ".github/workflows/review-diff.yml", "提供 PR diff SARIF 扫描", "添加 review-diff workflow，把本项目 SARIF 输出上传到 Code Scanning。"),
		fileCheck(root, "Security report workflow", ".github/workflows/security-report.yml", "提供安全专项报告门禁", "添加 security-report workflow，导出真实 issues、扫描 PR diff，并在安全风险阻塞时失败。"),
		fileCheck(root, "Release check workflow", ".github/workflows/release-check.yml", "提供发布准备门禁检查", "添加 release-check workflow，使用真实 GitHub issues/PRs 并在 BLOCKED 时失败。"),
		fileCheck(root, "Release artifacts workflow", ".github/workflows/release-artifacts.yml", "提供 tag 发布产物、SBOM 和 provenance", "添加 release-artifacts workflow，在 v* tag 上构建产物、生成 SBOM/checksums 和 provenance。"),
		fileCheck(root, "Dependabot config", ".github/dependabot.yml", "提供依赖更新自动化", "添加 .github/dependabot.yml，覆盖 Go module 和 GitHub Actions。"),
		fileCheck(root, "Bug issue template", ".github/ISSUE_TEMPLATE/bug_report.md", "提供 Bug 报告模板", "添加 Bug issue template，收集复现步骤、期望行为、环境和日志。"),
		fileCheck(root, "Feature issue template", ".github/ISSUE_TEMPLATE/feature_request.md", "提供功能请求模板", "添加 Feature issue template，收集使用场景、方案和验收标准。"),
		fileCheck(root, "Pull request template", ".github/PULL_REQUEST_TEMPLATE.md", "提供 PR 说明模板", "添加 PR 模板，要求说明变更、测试、风险和回滚方式。"),
		fileCheck(root, "Roadmap", "docs/ROADMAP.md", "提供后续维护路线图", "添加 docs/ROADMAP.md，说明当前版本能力和后续维护计划。"),
		contentCheck(root, "CI least privilege", ".github/workflows/ci.yml", []string{"permissions:", "contents: read"}, "CI workflow 使用最小只读权限", "在 CI workflow 顶层添加 permissions: contents: read，避免默认过宽权限。"),
		contentCheck(root, "CI test command", ".github/workflows/ci.yml", []string{"go test ./..."}, "CI workflow 执行 Go 测试", "在 CI workflow 中加入 go test ./...，确保 PR 自动验证全部 Go package。"),
		contentCheck(root, "CI build command", ".github/workflows/ci.yml", []string{"go build ./cmd/oss-maintainer-kit"}, "CI workflow 执行 CLI 构建", "在 CI workflow 中加入 go build ./cmd/oss-maintainer-kit，确保 CLI 可构建。"),
		contentCheck(root, "govulncheck least privilege", ".github/workflows/govulncheck.yml", []string{"permissions:", "contents: read"}, "govulncheck workflow 使用最小只读权限", "在 govulncheck workflow 中配置 permissions: contents: read。"),
		contentCheck(root, "govulncheck package coverage", ".github/workflows/govulncheck.yml", []string{"golang/govulncheck-action", "go-version-input: \"1.21\"", "go-package: ./..."}, "govulncheck workflow 使用项目 Go 版本扫描全部 Go package", "使用 golang/govulncheck-action，设置 go-version-input: \"1.21\" 和 go-package: ./...。"),
		contentCheck(root, "Scorecard security permission", ".github/workflows/scorecard.yml", []string{"permissions:", "contents: read", "security-events: write", "id-token: write"}, "Scorecard workflow 使用 SARIF 上传和结果发布所需最小权限", "为 Scorecard workflow 配置 contents: read、security-events: write 和 id-token: write。"),
		contentCheck(root, "Scorecard SARIF upload", ".github/workflows/scorecard.yml", []string{"ossf/scorecard-action", "results_format: sarif", "publish_results: true", "github/codeql-action/upload-sarif"}, "Scorecard workflow 输出 SARIF 并上传 Code Scanning", "使用 ossf/scorecard-action 输出 SARIF，启用 publish_results，并用 upload-sarif 上传。"),
		contentCheck(root, "Review diff SARIF upload", ".github/workflows/review-diff.yml", []string{"github/codeql-action/upload-sarif"}, "PR diff workflow 上传 SARIF 到 code scanning", "在 review-diff workflow 中调用 github/codeql-action/upload-sarif 上传扫描结果。"),
		contentCheck(root, "Review diff security permission", ".github/workflows/review-diff.yml", []string{"security-events: write"}, "PR diff workflow 具备 SARIF 上传权限", "为 review-diff workflow 添加 security-events: write 权限。"),
		contentCheck(root, "Security report least privilege", ".github/workflows/security-report.yml", []string{"permissions:", "contents: read", "issues: read", "pull-requests: read"}, "security-report workflow 使用所需最小只读权限", "为 security-report workflow 配置 contents/issues/pull-requests 只读权限。"),
		contentCheck(root, "Security report live data export", ".github/workflows/security-report.yml", []string{"github-export", "--kind issues", "--state open", "github.repository"}, "security-report workflow 导出当前仓库 open issues", "在 security-report workflow 中用 github-export 导出当前仓库 open issues。"),
		contentCheck(root, "Security report diff gate", ".github/workflows/security-report.yml", []string{"git diff", "security-report", "--fail-on-risk"}, "security-report workflow 扫描 PR diff 并在风险阻塞时失败", "在 security-report workflow 中生成 PR diff，运行 security-report --fail-on-risk。"),
		contentCheck(root, "Security report artifact", ".github/workflows/security-report.yml", []string{"actions/upload-artifact", "security-report.md", "security-report.json"}, "security-report workflow 上传 Markdown 和 JSON 报告", "在 security-report workflow 中上传 Markdown/JSON 安全报告 artifact。"),
		contentCheck(root, "Release check least privilege", ".github/workflows/release-check.yml", []string{"permissions:", "contents: read", "issues: read", "pull-requests: read"}, "release-check workflow 使用所需最小只读权限", "为 release-check workflow 配置 contents/issues/pull-requests 只读权限。"),
		contentCheck(root, "Release check live data export", ".github/workflows/release-check.yml", []string{"github-export", "--kind issues", "--kind pulls", "github.repository"}, "release-check workflow 导出当前仓库真实 issue 和 PR 数据", "在 workflow 中用 github-export 导出当前仓库 issues 和 pulls，避免只检查示例数据。"),
		contentCheck(root, "Release check policy command", ".github/workflows/release-check.yml", []string{"release-check", "--policy examples/release-policy.json"}, "release-check workflow 使用发布策略执行门禁", "运行 release-check 时传入 examples/release-policy.json。"),
		contentCheck(root, "Release check blocking exit", ".github/workflows/release-check.yml", []string{"--fail-on-blocked"}, "release-check workflow 在发布被阻塞时返回失败状态", "在 release-check workflow 中启用 --fail-on-blocked，让 BLOCKED 使 workflow 失败。"),
		contentCheck(root, "Release artifacts tag trigger", ".github/workflows/release-artifacts.yml", []string{"tags:", "\"v*\""}, "release-artifacts workflow 仅在版本 tag 上构建发布产物", "将 release-artifacts workflow 触发条件限制为 push tags v*。"),
		contentCheck(root, "Release artifacts provenance", ".github/workflows/release-artifacts.yml", []string{"attestations: write", "id-token: write", "actions/attest-build-provenance", "subject-path: \"dist/*\""}, "release-artifacts workflow 为发布产物生成 provenance attestation", "配置 attestations/id-token 权限，并用 actions/attest-build-provenance 绑定 dist/*。"),
		contentCheck(root, "Release artifacts SBOM checksums", ".github/workflows/release-artifacts.yml", []string{"sbom", "--output", "sha256sum", "actions/upload-artifact"}, "release-artifacts workflow 生成 SBOM、校验和并上传产物", "在 release workflow 中生成 SBOM、sha256sum checksums，并通过 upload-artifact 上传。"),
		contentCheck(root, "Dependabot Go module updates", ".github/dependabot.yml", []string{"package-ecosystem: \"gomod\""}, "Dependabot 覆盖 Go module 更新", "在 dependabot.yml 中添加 package-ecosystem: \"gomod\"。"),
		contentCheck(root, "Dependabot GitHub Actions updates", ".github/dependabot.yml", []string{"package-ecosystem: \"github-actions\""}, "Dependabot 覆盖 GitHub Actions 更新", "在 dependabot.yml 中添加 package-ecosystem: \"github-actions\"。"),
		contentCheck(root, "PR template test checklist", ".github/PULL_REQUEST_TEMPLATE.md", []string{"测试"}, "PR 模板提醒贡献者说明测试结果", "在 PR 模板中加入测试结果说明项。"),
		contentCheck(root, "PR template risk checklist", ".github/PULL_REQUEST_TEMPLATE.md", []string{"风险"}, "PR 模板提醒贡献者说明风险和影响", "在 PR 模板中加入风险和影响说明项。"),
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
	var failed []Check
	for _, check := range summary.Checks {
		if !check.Passed {
			failed = append(failed, check)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(&b, "\n## 失败项修复建议\n\n")
		for _, check := range failed {
			fmt.Fprintf(&b, "- **%s**（`%s`）：%s\n", check.Name, check.Path, check.Recommendation)
		}
	}
	return b.String()
}

func fileCheck(root, name, path, message, recommendation string) Check {
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Stat(fullPath)
	if err != nil {
		return Check{Name: name, Passed: false, Path: path, Message: "缺失：" + message, Recommendation: recommendation}
	}
	if info.IsDir() {
		return Check{Name: name, Passed: false, Path: path, Message: "路径是目录，不是文件", Recommendation: recommendation}
	}
	if info.Size() == 0 {
		return Check{Name: name, Passed: false, Path: path, Message: "文件为空", Recommendation: recommendation}
	}
	return Check{Name: name, Passed: true, Path: path, Message: message}
}

func contentCheck(root, name, path string, required []string, message, recommendation string) Check {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return Check{Name: name, Passed: false, Path: path, Message: "无法读取：" + message, Recommendation: recommendation}
	}
	text := strings.ToLower(string(data))
	for _, item := range required {
		if !strings.Contains(text, strings.ToLower(item)) {
			return Check{Name: name, Passed: false, Path: path, Message: "缺少内容：" + item, Recommendation: recommendation}
		}
	}
	return Check{Name: name, Passed: true, Path: path, Message: message}
}
