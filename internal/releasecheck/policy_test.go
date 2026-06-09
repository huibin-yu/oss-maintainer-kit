package releasecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func TestBuildWithPolicyBlocksStaleIssues(t *testing.T) {
	now := time.Now().AddDate(0, 0, -40)
	result := BuildWithPolicy(Input{
		Project: "demo",
		Version: "v1.0.0",
		Health:  health.Summary{Score: 100},
		Issues: []model.Issue{{
			Number:    7,
			Title:     "needs follow up",
			State:     "open",
			UpdatedAt: now,
		}},
	}, Policy{
		BlockStaleIssues: true,
		MaxStaleIssues:   0,
	})

	if result.Ready {
		t.Fatalf("release unexpectedly ready: %#v", result)
	}
	if !contains(result.Blockers, "长期未更新 issue 超过阈值") {
		t.Fatalf("missing stale blocker: %#v", result.Blockers)
	}
}

func TestBuildWithPolicyCanAllowSecurityIssues(t *testing.T) {
	result := BuildWithPolicy(Input{
		Project: "demo",
		Version: "v1.0.0",
		Health:  health.Summary{Score: 100},
		Issues: []model.Issue{{
			Number: 1,
			Title:  "security token leak",
			State:  "open",
		}},
	}, Policy{
		BlockSecurityIssues: false,
		MinHealthScore:      90,
	})

	if !result.Ready {
		t.Fatalf("release should be ready when policy allows security issues: %#v", result)
	}
}

func TestLoadPolicyFromJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release-policy.json")
	if err := os.WriteFile(path, []byte(`{
		"min_health_score": 95,
		"block_security_issues": true,
		"block_stale_issues": true,
		"max_stale_issues": 2,
		"required_commands": ["rtk go test ./...", "rtk go build ./cmd/oss-maintainer-kit"]
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	policy, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if policy.MinHealthScore != 95 || !policy.BlockStaleIssues || policy.MaxStaleIssues != 2 {
		t.Fatalf("unexpected policy: %#v", policy)
	}
	if len(policy.RequiredCommands) != 2 {
		t.Fatalf("required commands = %#v", policy.RequiredCommands)
	}
}

func TestLoadPolicyRejectsInvalidValues(t *testing.T) {
	for name, body := range map[string]string{
		"bad health score": `{"min_health_score": 101}`,
		"negative stale":   `{"max_stale_issues": -1}`,
		"empty command":    `{"required_commands": ["rtk go test ./...", " "]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "release-policy.json")
			if err := os.WriteFile(path, []byte(body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadPolicy(path)
			if err == nil {
				t.Fatalf("expected error for %s", body)
			}
		})
	}
}

func TestLoadPolicyRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release-policy.json")
	if err := os.WriteFile(path, []byte(`{"min_health_score": 95, "minimum_health_score": 90}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPolicy(path)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "minimum_health_score") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
