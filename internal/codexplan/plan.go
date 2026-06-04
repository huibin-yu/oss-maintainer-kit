package codexplan

import (
	"fmt"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

type Plan struct {
	Project     string   `json:"project"`
	Repository  string   `json:"repository"`
	Workflows   []string `json:"workflows"`
	Guardrails  []string `json:"guardrails"`
	APIUseCases []string `json:"api_use_cases"`
}

func Build(project, repository string, summary model.MaintainerReport) Plan {
	workflows := []string{
		"PR review：审查规则变更、CLI 输出变更、边界条件和测试覆盖",
		"issue triage：总结 issue 内容、建议标签、识别重复问题和优先级",
		"release workflow：根据合并 PR 生成发布说明草稿并检查遗漏",
	}
	if summary.SecurityIssues > 0 {
		workflows = append(workflows, "security workflow：优先处理安全相关 issue，并生成修复验证清单")
	}
	if summary.StaleIssues > 0 {
		workflows = append(workflows, "maintenance workflow：梳理长期未更新 issue，建议关闭、复现或升级优先级")
	}

	return Plan{
		Project:    project,
		Repository: repository,
		Workflows:  workflows,
		Guardrails: []string{
			"所有 AI 输出必须由维护者确认后才能合并",
			"安全问题不公开粘贴 token、密钥或私有仓库信息",
			"规则引擎保持确定性测试，AI 建议作为辅助输入",
			"发布说明和 PR review 结果必须可追溯到具体 issue 或 PR",
		},
		APIUseCases: []string{
			"对 PR diff 生成 review 摘要和测试建议",
			"对 issue 生成分类、优先级和维护者回复草稿",
			"对合并 PR 生成 release notes 和 changelog 草稿",
			"对安全相关报告生成修复步骤和回归测试建议",
		},
	}
}

func Markdown(plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Codex for OSS 使用计划\n\n")
	fmt.Fprintf(&b, "项目：%s\n\n", plan.Project)
	if plan.Repository != "" {
		fmt.Fprintf(&b, "仓库：%s\n\n", plan.Repository)
	}
	fmt.Fprintf(&b, "## 计划使用场景\n\n")
	for _, item := range plan.Workflows {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	fmt.Fprintf(&b, "\n## API Credits 用途\n\n")
	for _, item := range plan.APIUseCases {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	fmt.Fprintf(&b, "\n## 维护者约束\n\n")
	for _, item := range plan.Guardrails {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	return b.String()
}
