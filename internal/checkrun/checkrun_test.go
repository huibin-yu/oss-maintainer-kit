package checkrun

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

func TestFromFindingsBuildsFailurePayloadWithAnnotations(t *testing.T) {
	payload := FromFindings([]diffreview.Finding{{
		File:     "main.go",
		Line:     3,
		Severity: "critical",
		Rule:     "possible-secret",
		Message:  "新增代码可能包含硬编码凭证或密钥",
		Snippet:  `const token = "sk_live_123456"`,
	}})

	if payload.Name != "oss-maintainer-kit review-diff" {
		t.Fatalf("name = %q", payload.Name)
	}
	if payload.Status != "completed" || payload.Conclusion != "failure" {
		t.Fatalf("status=%q conclusion=%q", payload.Status, payload.Conclusion)
	}
	if !strings.Contains(payload.Output.Summary, "critical=1") {
		t.Fatalf("summary = %q", payload.Output.Summary)
	}
	if len(payload.Output.Annotations) != 1 {
		t.Fatalf("annotations = %#v", payload.Output.Annotations)
	}
	annotation := payload.Output.Annotations[0]
	if annotation.Path != "main.go" || annotation.StartLine != 3 || annotation.AnnotationLevel != "failure" {
		t.Fatalf("annotation = %#v", annotation)
	}
	if !strings.Contains(annotation.RawDetails, "sk_live") {
		t.Fatalf("raw details = %q", annotation.RawDetails)
	}
}

func TestFromFindingsBuildsSuccessPayloadWhenClean(t *testing.T) {
	payload := FromFindings(nil)

	if payload.Conclusion != "success" {
		t.Fatalf("conclusion = %q", payload.Conclusion)
	}
	if len(payload.Output.Annotations) != 0 {
		t.Fatalf("annotations = %#v", payload.Output.Annotations)
	}
	if !strings.Contains(payload.Output.Summary, "未发现") {
		t.Fatalf("summary = %q", payload.Output.Summary)
	}
}
