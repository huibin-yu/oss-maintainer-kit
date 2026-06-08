package releasedraft

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildCreatesReleaseDraftFromMergedPulls(t *testing.T) {
	draft := Build(Input{
		Project: "demo",
		Version: "v1.2.0",
		Pulls: []model.PullRequest{
			{Number: 3, Title: "fix panic on startup", Author: "bob", Labels: []string{"bug"}, Merged: true},
			{Number: 1, Title: "feat: add export", Author: "alice", Labels: []string{"feature"}, Merged: true},
			{Number: 2, Title: "docs: update readme", Author: "carol", Labels: []string{"documentation"}, Merged: true},
			{Number: 4, Title: "draft work", Author: "dave", Merged: false},
		},
	})

	if draft.TagName != "v1.2.0" {
		t.Fatalf("tag name = %q", draft.TagName)
	}
	if draft.Name != "demo v1.2.0" {
		t.Fatalf("name = %q", draft.Name)
	}
	if !draft.Draft || draft.Prerelease {
		t.Fatalf("unexpected release flags: %#v", draft)
	}
	for _, want := range []string{
		"## 功能",
		"- feat: add export (#1) by @alice",
		"## 修复",
		"- fix panic on startup (#3) by @bob",
		"## 文档",
		"- docs: update readme (#2) by @carol",
	} {
		if !strings.Contains(draft.Body, want) {
			t.Fatalf("missing %q:\n%s", want, draft.Body)
		}
	}
	if strings.Contains(draft.Body, "draft work") {
		t.Fatalf("unmerged PR leaked into body:\n%s", draft.Body)
	}
}

func TestBuildSupportsPreviousTagAndPrerelease(t *testing.T) {
	draft := Build(Input{
		Project:     "demo",
		Version:     "v2.0.0-rc.1",
		PreviousTag: "v1.9.0",
		Prerelease:  true,
	})

	if !draft.Prerelease {
		t.Fatalf("expected prerelease draft")
	}
	if !strings.Contains(draft.Body, "比较范围：`v1.9.0...v2.0.0-rc.1`") {
		t.Fatalf("missing compare range:\n%s", draft.Body)
	}
	if !strings.Contains(draft.Body, "暂无已合并 PR。") {
		t.Fatalf("missing empty state:\n%s", draft.Body)
	}
}
