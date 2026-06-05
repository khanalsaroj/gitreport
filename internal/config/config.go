package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the entire gitreport configuration file.
type Config struct {
	Version  string                `yaml:"version"`
	Defaults Defaults              `yaml:"defaults"`
	Formats  map[string]*FormatDef `yaml:"formats"`
	Prompts  map[string]*Prompt    `yaml:"prompts"`
}

// Defaults holds global fallback values applied to every prompt unless overridden.
type Defaults struct {
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	Format      string  `yaml:"format"`
	ByAuthor    bool    `yaml:"by_author"`
	Language    string  `yaml:"language"`
	Tone        string  `yaml:"tone"`
	DateRange   string  `yaml:"date_range"`
}

// FormatDef describes an output format (markdown, slack, html, json).
type FormatDef struct {
	Description  string `yaml:"description"`
	BulletStyle  string `yaml:"bullet_style"`
	HeadingLevel string `yaml:"heading_level"`
}

// Prompt represents a named AI prompt with metadata, templates, and output config.
type Prompt struct {
	Meta   PromptMeta   `yaml:"meta"`
	Input  PromptInput  `yaml:"input"`
	System string       `yaml:"system"`
	User   string       `yaml:"user"`
	Output PromptOutput `yaml:"output"`
}

// PromptMeta holds descriptive metadata for a prompt.
type PromptMeta struct {
	ID          string   `yaml:"id"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Audience    string   `yaml:"audience"`
}

// PromptInput declares the template variables a prompt accepts.
// All fields are template placeholders (e.g. "{{commits}}") until rendered.
type PromptInput struct {
	// Shared
	Format    string `yaml:"format"`
	ByAuthor  string `yaml:"by_author"`
	DateRange string `yaml:"date_range"`
	Repo      string `yaml:"repo"`
	Tone      string `yaml:"tone"`

	// summary_prompt / changelog_prompt / standup_prompt / release_notes_prompt
	Commits string `yaml:"commits"`

	// hard_summary_prompt
	Activity string `yaml:"activity"`

	// changelog_prompt
	FromTag         string `yaml:"from_tag"`
	ToTag           string `yaml:"to_tag"`
	BreakingChanges string `yaml:"breaking_changes"`

	// release_notes_prompt
	Version             string `yaml:"version"`
	ProductName         string `yaml:"product_name"`
	IncludeUpgradeNotes string `yaml:"include_upgrade_notes"`

	// standup_prompt
	Author   string `yaml:"author"`
	Date     string `yaml:"date"`
	Blockers string `yaml:"blockers"`

	// pr_description_prompt
	Diff     string `yaml:"diff"`
	Branch   string `yaml:"branch"`
	Ticket   string `yaml:"ticket"`
	Breaking string `yaml:"breaking"`

	// summary_prompt (author grouping)
	Team string `yaml:"team"`
}

// PromptOutput holds the structural rules for the model's response.
type PromptOutput struct {
	Strict            bool            `yaml:"strict"`
	OmitEmptySections bool            `yaml:"omit_empty_sections"`
	Title             string          `yaml:"title,omitempty"`
	Structure         []OutputSection `yaml:"structure"`
	Rules             []string        `yaml:"rules,omitempty"`
}

type OutputSection struct {
	Title       string              `yaml:"title"`
	Description string              `yaml:"description,omitempty"`
	Optional    *bool               `yaml:"optional,omitempty"`
	Conditional string              `yaml:"conditional,omitempty"`
	Style       string              `yaml:"style,omitempty"`
	Rules       *OutputSectionRules `yaml:"rules,omitempty"`
}

type OutputSectionRules struct {
	MinItems *int `yaml:"min_items,omitempty"`
	MaxItems *int `yaml:"max_items,omitempty"`
}

// Load reads and parses the configuration, searching in order:
//  1. GITREPORT_CONFIG env var
//  2. ./config/gitreport.yaml  (local project config)
//  3. ~/.gitreport/config/gitreport.yaml  (user-level config)
//  4. the embedded Default config (always available)
//
// The embedded default guarantees gitreport works out of the box even before
// `gitreport init` has been run.
func Load() (*Config, error) {
	data, source := readConfig()

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", source, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", source, err)
	}

	return &cfg, nil
}

// readConfig returns the raw config bytes and a human-readable source label for
// use in error messages. It falls back to the embedded Default when no config
// file is present on disk.
func readConfig() (data []byte, source string) {
	if p := resolvePath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			return b, fmt.Sprintf("%q", p)
		}
	}
	return Default, "(embedded default)"
}

// resolvePath returns the first config file path that exists, or "" if none do.
func resolvePath() string {
	candidates := configCandidates()
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// configCandidates lists the on-disk config locations in priority order.
func configCandidates() []string {
	candidates := []string{
		os.Getenv("GITREPORT_CONFIG"),
		filepath.Join("config", "gitreport.yaml"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".gitreport", "config", "gitreport.yaml"))
	}
	return candidates
}

// validate checks structural correctness after unmarshalling.
func (c *Config) validate() error {
	if c.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if len(c.Prompts) == 0 {
		return fmt.Errorf("config defines no prompts")
	}

	for name, p := range c.Prompts {
		if p == nil {
			return fmt.Errorf("prompt %q is null", name)
		}
		if p.System == "" {
			return fmt.Errorf("prompt %q: missing system template", name)
		}
		if p.User == "" {
			return fmt.Errorf("prompt %q: missing user template", name)
		}
		for i, s := range p.Output.Structure {
			if s.Title == "" {
				return fmt.Errorf("prompt %q: output.structure[%d] missing title", name, i)
			}
		}
	}

	return nil
}

// GetPrompt retrieves a named prompt, returning an error if not found.
func (c *Config) GetPrompt(name string) (*Prompt, error) {
	p, ok := c.Prompts[name]
	if !ok {
		return nil, fmt.Errorf("prompt %q not found; available: %s",
			name, strings.Join(c.PromptNames(), ", "))
	}
	return p, nil
}

// PromptNames returns all defined prompt names in sorted order.
func (c *Config) PromptNames() []string {
	names := make([]string, 0, len(c.Prompts))
	for name := range c.Prompts {
		names = append(names, name)
	}
	return names
}

// FormatSpec returns the human-readable formatting instructions for a format
// key (e.g. "markdown"), falling back to the default format and finally to a
// generic instruction when the key is unknown. The returned string is fed to
// the model as the OUTPUT FORMAT specification.
func (c *Config) FormatSpec(key string) string {
	if key == "" {
		key = c.Defaults.Format
	}
	if f, ok := c.Formats[key]; ok && f != nil && f.Description != "" {
		return f.Description
	}
	return "No specific formatting rules provided. Use clean, structured output."
}

// Vars maps template variable names to their rendered values. Keys correspond
// to {{variable}} placeholders in prompt templates.
type Vars map[string]string

// Render replaces every {{key}} placeholder in tmpl with its value from vars.
// Unknown placeholders are left untouched so callers can detect them. It is the
// single template-substitution primitive shared across the codebase.
func Render(tmpl string, vars Vars) string {
	for key, val := range vars {
		tmpl = strings.ReplaceAll(tmpl, "{{"+key+"}}", val)
	}
	return tmpl
}
