package sarif

import (
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

func TestFromFindingsMapsSARIFFields(t *testing.T) {
	log := FromFindings([]diffreview.Finding{{
		File:     "main.go",
		Line:     12,
		Severity: "critical",
		Rule:     "possible-secret",
		Message:  "secret found",
	}})

	if log.Version != "2.1.0" {
		t.Fatalf("version = %s", log.Version)
	}
	if got := log.Runs[0].Results[0].Level; got != "error" {
		t.Fatalf("level = %s", got)
	}
	if got := log.Runs[0].Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI; got != "main.go" {
		t.Fatalf("uri = %s", got)
	}
}
