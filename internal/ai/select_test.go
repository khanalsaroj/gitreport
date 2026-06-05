package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeSettings writes a setting.json into an isolated home directory and points
// the home-dir lookup at it. claude-code is disabled so the result does not
// depend on whether the CLI is installed on the test machine.
func writeSettings(t *testing.T, json string) {
	t.Helper()
	clearProviderEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".gitreport")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "setting.json"), []byte(json), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStatusesReflectsConfiguration(t *testing.T) {
	writeSettings(t, `{
		"OPENAI_API_KEY": "or-key",
		"OPENAI_BASE_URL": "https://openrouter.ai/api/v1/chat/completions",
		"OPENAI_MODEL": "m",
		"providers": { "claude-code": { "enabled": false } }
	}`)

	statuses, err := Statuses(context.Background())
	if err != nil {
		t.Fatalf("Statuses error: %v", err)
	}

	byID := map[string]Status{}
	for _, s := range statuses {
		byID[s.ID] = s
	}

	if byID[ProviderClaudeCode].Available {
		t.Error("claude-code disabled in settings should be unavailable")
	}
	if !byID[ProviderOpenRouter].Available {
		t.Errorf("openrouter should be available from legacy key: %s", byID[ProviderOpenRouter].Detail)
	}
	if !byID[ProviderOpenRouter].Primary {
		t.Error("openrouter should be the primary provider when claude-code is disabled")
	}
}

func TestSelectReturnsChain(t *testing.T) {
	writeSettings(t, `{
		"OPENAI_API_KEY": "or-key",
		"OPENAI_BASE_URL": "https://openrouter.ai/api/v1/chat/completions",
		"OPENAI_MODEL": "m",
		"providers": { "claude-code": { "enabled": false } }
	}`)

	prov, err := Select(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}
	if _, ok := prov.(*Chain); !ok {
		t.Errorf("Select returned %T, want *Chain", prov)
	}
}

func TestSelectNoProviderError(t *testing.T) {
	// No legacy key, no provider keys, claude-code disabled -> nothing available.
	writeSettings(t, `{ "providers": { "claude-code": { "enabled": false } } }`)

	_, err := Select(context.Background(), Options{})
	if err == nil {
		t.Fatal("expected an error when no provider is available")
	}
}

func TestSelectForcedUnknown(t *testing.T) {
	writeSettings(t, `{}`)
	if _, err := Select(context.Background(), Options{Force: "nonexistent"}); err == nil {
		t.Fatal("expected an error for an unknown forced provider")
	}
}
