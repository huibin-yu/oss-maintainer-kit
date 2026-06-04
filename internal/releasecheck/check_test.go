package releasecheck

import (
	"strings"
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildBlocksReleaseForSecurityAndHealthFailures(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	result := Build(Input{
		Project: "demo",
		Version: "v1.0.0",
		Issues: []model.Issue{
			{Number: 1, Title: "security token leak", State: "open", UpdatedAt: now},
		},
		Pulls: []model.PullRequest{
			{Number: 10, Title: "feat: add command", Author: "alice", Merged: true},
		},
		Health: health.Summary{
			Score: 95,
			Checks: []health.Check{
				{Name: "PR template test checklist", Passed: false, Path: ".github/PULL_REQUEST_TEMPLATE.md", Message: "缺少内容：测试"},
			},
		},
	})

	if result.Ready {
		t.Fatalf("release marked ready: %#v", result)
	}
	doc := Markdown(result)
	for _, want := range []string{
		"# demo v1.0.0 发布准备检查",
		"发布状态：**BLOCKED**",
		"安全相关 issue 未处理",
		"健康度未达到 100/100",
		"PR template test checklist",
		"release-notes --input examples/pulls.json --version v1.0.0",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func TestBuildMarksReleaseReadyWhenNoBlockers(t *testing.T) {
	result := Build(Input{
		Project: "demo",
		Version: "v1.0.1",
		Pulls: []model.PullRequest{
			{Number: 11, Title: "fix parser panic", Author: "bob", Merged: true},
		},
		Health: health.Summary{Score: 100},
	})

	if !result.Ready {
		t.Fatalf("release not ready: %#v", result)
	}
	doc := Markdown(result)
	for _, want := range []string{
		"发布状态：**READY**",
		"已合并 PR：1",
		"rtk go test ./...",
		"rtk go build ./cmd/oss-maintainer-kit",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}
