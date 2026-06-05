package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// envValue reads an environment variable and trims surrounding whitespace, so a
// stray newline in an exported API key does not corrupt the Authorization header.
func envValue(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// Settings is the parsed ~/.gitreport/setting.json. It is backward compatible:
// older files containing only the flat OPENAI_* fields continue to work, and
// are interpreted as the configuration for a single OpenAI-compatible endpoint
// (historically OpenRouter). New files may additionally declare a provider
// priority order and per-provider credentials.
type Settings struct {
	// Legacy flat fields — a fully-specified OpenAI-compatible endpoint.
	APIKey  string `json:"OPENAI_API_KEY"`
	BaseURL string `json:"OPENAI_BASE_URL"`
	Model   string `json:"OPENAI_MODEL"`

	// Priority overrides the default provider preference order. Entries are
	// provider IDs (see specs.go); unknown IDs are ignored.
	Priority []string `json:"priority,omitempty"`

	// Providers holds per-provider configuration keyed by provider ID.
	Providers map[string]ProviderConfig `json:"providers,omitempty"`
}

// ProviderConfig is the per-provider section of setting.json.
type ProviderConfig struct {
	// Enabled, when non-nil and false, removes the provider from selection even
	// if it would otherwise be available.
	Enabled *bool `json:"enabled,omitempty"`

	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model,omitempty"`
}

// settingsPath returns the path to the user's setting.json.
func settingsPath() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".gitreport", "setting.json"), nil
}

// loadSettings reads and parses setting.json. A missing file is not an error:
// it returns an empty Settings so configuration can still come entirely from
// environment variables.
func loadSettings() (*Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return nil, fmt.Errorf("read settings file %q: %w", path, err)
	}

	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse settings file %q: %w", path, err)
	}
	return &cfg, nil
}

// homeDir returns the current user's home directory on Windows, macOS, and Linux.
func homeDir() (string, error) {
	return os.UserHomeDir()
}
