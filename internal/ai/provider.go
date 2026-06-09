package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return ProviderConfig{}, fmt.Errorf("parse provider config %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return ProviderConfig{}, fmt.Errorf("parse provider config %s: unexpected trailing JSON value", path)
		}
		return ProviderConfig{}, fmt.Errorf("parse provider config %s: unexpected trailing JSON value", path)
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
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("base_url must not be empty")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("model must not be empty")
	}
	if strings.TrimSpace(c.APIKeyEnv) == "" {
		return fmt.Errorf("api_key_env must not be empty")
	}
	return nil
}
