package ai

import (
	"encoding/json"
	"fmt"
	"os"
)

type ProviderConfig struct {
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	APIKeyEnv string `json:"api_key_env"`
	Retries   int    `json:"retries"`
}

func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		BaseURL:   "https://api.openai.com/v1",
		Model:     "gpt-4.1-mini",
		APIKeyEnv: "OPENAI_API_KEY",
		Retries:   0,
	}
}

func LoadProviderConfig(path string) (ProviderConfig, error) {
	cfg := DefaultProviderConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ProviderConfig{}, fmt.Errorf("read provider config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ProviderConfig{}, fmt.Errorf("parse provider config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return ProviderConfig{}, fmt.Errorf("invalid provider config %s: %w", path, err)
	}
	return cfg, nil
}

func (c ProviderConfig) Validate() error {
	if c.Retries < 0 {
		return fmt.Errorf("retries must be greater than or equal to 0")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("base_url must not be empty")
	}
	if c.Model == "" {
		return fmt.Errorf("model must not be empty")
	}
	if c.APIKeyEnv == "" {
		return fmt.Errorf("api_key_env must not be empty")
	}
	return nil
}
