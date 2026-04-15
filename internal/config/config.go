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

// Load reads and parses the config file, searching in order:
//  1. GITREPORT_CONFIG env var
//  2. ./config/gitreport.yaml  (local project config)
//  3. ~/.gitreport/config/gitreport.yaml  (user-level config)
func Load() (*Config, error) {
	path, err := resolvePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}

	return &cfg, nil
}

// resolvePath returns the first config file path that exists.
func resolvePath() (string, error) {
	if p := os.Getenv("GITREPORT_CONFIG"); p != "" {
		return p, nil
	}

	candidates := []string{
		"config/gitreport.yaml",
		func() string {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, ".gitreport", "config", "gitreport.yaml")
		}(),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf(
		"no config file found; set GITREPORT_CONFIG or create one of: %s",
		strings.Join(candidates, ", "),
	)
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

// Format returns the FormatDef for a given format key (e.g. "markdown"),
// falling back to the defaults format, then nil if unknown.
func (c *Config) Format(key string) *FormatDef {
	if key == "" {
		key = c.Defaults.Format
	}
	return c.Formats[key]
}

// Vars is a map of template variable names to their rendered values.
// Keys correspond to {{variable}} placeholders in system/user templates.
type Vars map[string]string

// Render substitutes all {{key}} placeholders in the prompt's system and user
// templates using the provided vars, then returns the rendered pair.
// Missing keys are left as-is so callers can detect unresolved placeholders.
func (p *Prompt) Render(vars Vars) (system, user string) {
	system = renderTemplate(p.System, vars)
	user = renderTemplate(p.User, vars)
	return system, user
}

// MissingVars returns placeholder names present in the templates but absent
// from vars. Useful for pre-flight checks before calling the AI API.
func (p *Prompt) MissingVars(vars Vars) []string {
	all := extractPlaceholders(p.System + p.User)
	missing := make([]string, 0)
	for _, key := range all {
		if _, ok := vars[key]; !ok {
			missing = append(missing, key)
		}
	}
	return missing
}

// renderTemplate replaces every {{key}} occurrence with vars[key].
func renderTemplate(tmpl string, vars Vars) string {
	result := tmpl
	for key, val := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", val)
	}
	return result
}

// extractPlaceholders returns all unique {{key}} names found in a template string.
func extractPlaceholders(tmpl string) []string {
	seen := map[string]bool{}
	var keys []string

	for {
		start := strings.Index(tmpl, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(tmpl[start:], "}}")
		if end == -1 {
			break
		}
		key := strings.TrimSpace(tmpl[start+2 : start+end])
		if !strings.HasPrefix(key, "#") && !strings.HasPrefix(key, "/") && key != "else" {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
		tmpl = tmpl[start+end+2:]
	}

	return keys
}
