package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name string
		tmpl string
		vars Vars
		want string
	}{
		{"simple", "hello {{name}}", Vars{"name": "world"}, "hello world"},
		{"repeated", "{{x}}-{{x}}", Vars{"x": "a"}, "a-a"},
		{"multiple keys", "{{a}} and {{b}}", Vars{"a": "1", "b": "2"}, "1 and 2"},
		{"unknown left intact", "{{a}} {{b}}", Vars{"a": "1"}, "1 {{b}}"},
		{"nil vars", "{{a}}", nil, "{{a}}"},
		{"no placeholders", "plain text", Vars{"a": "1"}, "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Render(tt.tmpl, tt.vars); got != tt.want {
				t.Errorf("Render(%q) = %q, want %q", tt.tmpl, got, tt.want)
			}
		})
	}
}

func TestFormatSpec(t *testing.T) {
	cfg := &Config{
		Defaults: Defaults{Format: "markdown"},
		Formats: map[string]*FormatDef{
			"markdown": {Description: "md spec"},
			"empty":    {Description: ""},
		},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"markdown", "md spec"},
		{"", "md spec"}, // empty -> defaults.Format
		{"unknown", "No specific formatting rules provided. Use clean, structured output."},
		{"empty", "No specific formatting rules provided. Use clean, structured output."},
	}
	for _, tt := range tests {
		if got := cfg.FormatSpec(tt.key); got != tt.want {
			t.Errorf("FormatSpec(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func validPrompt() *Prompt {
	return &Prompt{System: "sys", User: "usr"}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing version",
			cfg:     Config{Prompts: map[string]*Prompt{"p": validPrompt()}},
			wantErr: "version",
		},
		{
			name:    "no prompts",
			cfg:     Config{Version: "1"},
			wantErr: "no prompts",
		},
		{
			name:    "nil prompt",
			cfg:     Config{Version: "1", Prompts: map[string]*Prompt{"p": nil}},
			wantErr: "is null",
		},
		{
			name:    "missing system",
			cfg:     Config{Version: "1", Prompts: map[string]*Prompt{"p": {User: "u"}}},
			wantErr: "missing system",
		},
		{
			name:    "missing user",
			cfg:     Config{Version: "1", Prompts: map[string]*Prompt{"p": {System: "s"}}},
			wantErr: "missing user",
		},
		{
			name: "structure section missing title",
			cfg: Config{Version: "1", Prompts: map[string]*Prompt{"p": {
				System: "s", User: "u",
				Output: PromptOutput{Structure: []OutputSection{{Title: ""}}},
			}}},
			wantErr: "missing title",
		},
		{
			name: "valid",
			cfg:  Config{Version: "1", Prompts: map[string]*Prompt{"p": validPrompt()}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetPrompt(t *testing.T) {
	cfg := &Config{Prompts: map[string]*Prompt{"summary_prompt": validPrompt()}}

	if _, err := cfg.GetPrompt("summary_prompt"); err != nil {
		t.Errorf("GetPrompt(existing) unexpected error: %v", err)
	}

	_, err := cfg.GetPrompt("missing")
	if err == nil || !strings.Contains(err.Error(), "summary_prompt") {
		t.Errorf("GetPrompt(missing) error = %v, want it to list available prompts", err)
	}
}

func TestLoadFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	content := "version: \"1.0\"\nprompts:\n  p:\n    system: \"s\"\n    user: \"u\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITREPORT_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Version != "1.0" {
		t.Errorf("Version = %q, want 1.0", cfg.Version)
	}
	if _, err := cfg.GetPrompt("p"); err != nil {
		t.Errorf("expected prompt p: %v", err)
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// Parses as YAML but fails validation (no version).
	if err := os.WriteFile(path, []byte("prompts: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITREPORT_CONFIG", path)

	if _, err := Load(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

// TestEmbeddedDefaultValid guards against shipping a broken default config: the
// embedded bytes must parse and pass the same validation as a user config.
func TestEmbeddedDefaultValid(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal(Default, &cfg); err != nil {
		t.Fatalf("embedded default does not parse: %v", err)
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("embedded default fails validation: %v", err)
	}
	for _, want := range []string{"summary_prompt", "hard_summary_prompt"} {
		if _, err := cfg.GetPrompt(want); err != nil {
			t.Errorf("embedded default missing %q: %v", want, err)
		}
	}
}
