package releasecheck

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Policy struct {
	MinHealthScore      int      `json:"min_health_score"`
	BlockSecurityIssues bool     `json:"block_security_issues"`
	BlockStaleIssues    bool     `json:"block_stale_issues"`
	MaxStaleIssues      int      `json:"max_stale_issues"`
	RequiredCommands    []string `json:"required_commands"`
}

func DefaultPolicy() Policy {
	return Policy{
		MinHealthScore:      100,
		BlockSecurityIssues: true,
		BlockStaleIssues:    false,
		MaxStaleIssues:      0,
		RequiredCommands: []string{
			"rtk go test ./...",
			"rtk go build ./cmd/oss-maintainer-kit",
		},
	}
}

func LoadPolicy(path string) (Policy, error) {
	if path == "" {
		return DefaultPolicy(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read release policy %s: %w", path, err)
	}
	policy := DefaultPolicy()
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("parse release policy %s: %w", path, err)
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, fmt.Errorf("invalid release policy %s: %w", path, err)
	}
	return policy, nil
}

func (p Policy) Validate() error {
	if p.MinHealthScore < 0 || p.MinHealthScore > 100 {
		return fmt.Errorf("min_health_score must be between 0 and 100")
	}
	if p.MaxStaleIssues < 0 {
		return fmt.Errorf("max_stale_issues must be greater than or equal to 0")
	}
	for i, command := range p.RequiredCommands {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("required_commands[%d] must not be empty", i)
		}
	}
	return nil
}
