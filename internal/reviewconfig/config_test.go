package reviewconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(path, []byte(`{
		"rules": [
			{"id":"custom","severity":"high","contains":["danger"],"message":"custom risk","tags":["security"]}
		]
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].ID != "custom" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestValidateRejectsDuplicateRule(t *testing.T) {
	err := Config{Rules: []Rule{
		{ID: "x", Severity: "low", Contains: []string{"a"}, Message: "a"},
		{ID: "x", Severity: "low", Contains: []string{"b"}, Message: "b"},
	}}.Validate()
	if err == nil {
		t.Fatal("expected duplicate rule error")
	}
}
