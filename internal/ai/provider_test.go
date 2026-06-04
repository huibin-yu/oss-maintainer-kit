package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProviderConfigAppliesDefaults(t *testing.T) {
	cfg, err := LoadProviderConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL == "" || cfg.Model == "" || cfg.APIKeyEnv == "" {
		t.Fatalf("missing defaults: %#v", cfg)
	}
}

func TestLoadProviderConfigReadsJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provider.json")
	if err := os.WriteFile(path, []byte(`{
		"base_url": "https://example.com/v1",
		"model": "custom-model",
		"api_key_env": "CUSTOM_API_KEY",
		"retries": 2
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://example.com/v1" || cfg.Model != "custom-model" || cfg.APIKeyEnv != "CUSTOM_API_KEY" || cfg.Retries != 2 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoadProviderConfigRejectsInvalidRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provider.json")
	if err := os.WriteFile(path, []byte(`{"retries": -1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProviderConfig(path); err == nil {
		t.Fatal("expected invalid retries error")
	}
}
