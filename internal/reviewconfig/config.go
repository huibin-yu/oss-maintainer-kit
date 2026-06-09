package reviewconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	ID       string   `json:"id"`
	Enabled  *bool    `json:"enabled,omitempty"`
	Severity string   `json:"severity"`
	Contains []string `json:"contains"`
	Message  string   `json:"message"`
	Tags     []string `json:"tags"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	seen := map[string]bool{}
	for _, rule := range c.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("rule id is required")
		}
		if seen[rule.ID] {
			return fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = true
		if rule.Severity == "" {
			return fmt.Errorf("rule %q severity is required", rule.ID)
		}
		if !validSeverity(rule.Severity) {
			return fmt.Errorf("rule %q severity must be one of: critical, high, medium, low", rule.ID)
		}
		if rule.Message == "" {
			return fmt.Errorf("rule %q message is required", rule.ID)
		}
		if len(rule.Contains) == 0 {
			return fmt.Errorf("rule %q contains is required", rule.ID)
		}
	}
	return nil
}

func validSeverity(value string) bool {
	switch value {
	case "critical", "high", "medium", "low":
		return true
	default:
		return false
	}
}

func (r Rule) IsEnabled() bool {
	return r.Enabled == nil || *r.Enabled
}
