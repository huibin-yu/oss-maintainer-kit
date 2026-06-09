package releasedraft

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

type Input struct {
	Project     string
	Version     string
	PreviousTag string
	Pulls       []model.PullRequest
	Prerelease  bool
}

type Draft struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func Build(input Input) Draft {
	return Draft{
		TagName:    input.Version,
		Name:       fmt.Sprintf("%s %s", input.Project, input.Version),
		Body:       body(input),
		Draft:      true,
		Prerelease: input.Prerelease,
	}
}

func Markdown(draft Draft) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", draft.Name)
	fmt.Fprint(&b, draft.Body)
	if !strings.HasSuffix(draft.Body, "\n") {
		fmt.Fprintln(&b)
	}
	return b.String()
}

func body(input Input) string {
	groups := map[string][]model.PullRequest{
		"功能": {},
		"修复": {},
		"文档": {},
		"维护": {},
	}
	for _, pull := range input.Pulls {
		if !pull.Merged {
			continue
		}
		groups[groupFor(pull)] = append(groups[groupFor(pull)], pull)
	}
	for name := range groups {
		sort.SliceStable(groups[name], func(i, j int) bool {
			return groups[name][i].Number < groups[name][j].Number
		})
	}

	var b strings.Builder
	if input.PreviousTag != "" {
		fmt.Fprintf(&b, "比较范围：`%s...%s`\n\n", markdownInline(input.PreviousTag), markdownInline(input.Version))
	}
	wrote := false
	for _, name := range []string{"功能", "修复", "文档", "维护"} {
		items := groups[name]
		if len(items) == 0 {
			continue
		}
		wrote = true
		fmt.Fprintf(&b, "## %s\n\n", name)
		for _, pull := range items {
			fmt.Fprintf(&b, "- %s (#%d) by @%s\n", markdownInline(pull.Title), pull.Number, markdownInline(pull.Author))
		}
		fmt.Fprintln(&b)
	}
	if !wrote {
		fmt.Fprintln(&b, "暂无已合并 PR。")
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func groupFor(pull model.PullRequest) string {
	text := strings.ToLower(pull.Title + " " + strings.Join(pull.Labels, " "))
	switch {
	case strings.Contains(text, "fix") || strings.Contains(text, "bug"):
		return "修复"
	case strings.Contains(text, "doc") || strings.Contains(text, "readme"):
		return "文档"
	case strings.Contains(text, "feature") || strings.Contains(text, "feat") || strings.Contains(text, "add"):
		return "功能"
	default:
		return "维护"
	}
}

func markdownInline(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
