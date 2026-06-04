package codexplan

import (
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildIncludesRiskDrivenWorkflows(t *testing.T) {
	plan := Build("demo", "https://github.com/acme/demo", model.MaintainerReport{
		SecurityIssues: 1,
		StaleIssues:    2,
	})

	doc := Markdown(plan)
	if !strings.Contains(doc, "security workflow") {
		t.Fatalf("missing security workflow:\n%s", doc)
	}
	if !strings.Contains(doc, "maintenance workflow") {
		t.Fatalf("missing maintenance workflow:\n%s", doc)
	}
}
