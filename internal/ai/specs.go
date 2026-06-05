package ai

import (
	"context"
	"net/url"
	"strings"
)

// Provider IDs. Adding a new HTTP provider is as simple as adding a spec below;
// no other code changes are required.
const (
	ProviderClaudeCode = "claude-code"
	ProviderOpenAI     = "openai"
	ProviderGemini     = "gemini"
	ProviderGrok       = "grok"
	ProviderOpenRouter = "openrouter"
)

// DefaultPriority is the built-in preference order: most capable first, with
// OpenRouter as the universal fallback.
var DefaultPriority = []string{
	ProviderClaudeCode,
	ProviderOpenAI,
	ProviderGemini,
	ProviderGrok,
	ProviderOpenRouter,
}

type providerKind int

const (
	kindHTTP providerKind = iota
	kindClaudeCode
)

// spec is the static description of a known provider.
type spec struct {
	id           string
	kind         providerKind
	displayName  string
	envKeys      []string // env vars holding the API key, in priority order
	defaultURL   string
	defaultModel string
}

var specs = map[string]spec{
	ProviderClaudeCode: {
		id:          ProviderClaudeCode,
		kind:        kindClaudeCode,
		displayName: "Claude Code",
	},
	ProviderOpenAI: {
		id:           ProviderOpenAI,
		kind:         kindHTTP,
		displayName:  "OpenAI",
		envKeys:      []string{"GITREPORT_OPENAI_API_KEY"},
		defaultURL:   "https://api.openai.com/v1/chat/completions",
		defaultModel: "gpt-4o-mini",
	},
	ProviderGemini: {
		id:           ProviderGemini,
		kind:         kindHTTP,
		displayName:  "Google Gemini",
		envKeys:      []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		defaultURL:   "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		defaultModel: "gemini-2.0-flash",
	},
	ProviderGrok: {
		id:           ProviderGrok,
		kind:         kindHTTP,
		displayName:  "xAI Grok",
		envKeys:      []string{"XAI_API_KEY", "GROK_API_KEY"},
		defaultURL:   "https://api.x.ai/v1/chat/completions",
		defaultModel: "grok-2-latest",
	},
	ProviderOpenRouter: {
		id:           ProviderOpenRouter,
		kind:         kindHTTP,
		displayName:  "OpenRouter",
		envKeys:      []string{"OPENROUTER_API_KEY"},
		defaultURL:   "https://openrouter.ai/api/v1/chat/completions",
		defaultModel: "nvidia/nemotron-3-super-120b-a12b:free",
	},
}

// legacyConfig is the flat OPENAI_* configuration (env first, then setting.json).
// It maps to whichever provider its base URL points at, preserving the behavior
// of pre-multi-provider installs.
type legacyConfig struct {
	apiKey  string
	baseURL string
	model   string
}

func loadLegacy(s *Settings) legacyConfig {
	return legacyConfig{
		apiKey:  firstNonEmpty(envValue("OPENAI_API_KEY"), s.APIKey),
		baseURL: firstNonEmpty(envValue("OPENAI_BASE_URL"), s.BaseURL),
		model:   firstNonEmpty(envValue("OPENAI_MODEL"), s.Model),
	}
}

// classifyURL maps a base URL to the provider ID that owns its host. Unknown or
// empty hosts fall to OpenRouter, the universal fallback tier.
func classifyURL(raw string) string {
	host := ""
	if u, err := url.Parse(raw); err == nil {
		host = strings.ToLower(u.Host)
	}
	switch {
	case strings.Contains(host, "openrouter.ai"):
		return ProviderOpenRouter
	case strings.Contains(host, "api.openai.com"):
		return ProviderOpenAI
	case strings.Contains(host, "x.ai"):
		return ProviderGrok
	case strings.Contains(host, "googleapis.com"):
		return ProviderGemini
	default:
		return ProviderOpenRouter
	}
}

// resolved is the merged configuration for one HTTP provider.
type resolved struct {
	apiKey  string
	baseURL string
	model   string
	enabled bool
}

// resolveHTTP merges configuration for an HTTP provider, in increasing order of
// precedence: spec defaults, the legacy OPENAI_* trio (if it targets this
// provider), provider-specific environment keys, and explicit setting.json.
func resolveHTTP(sp spec, pc ProviderConfig, legacy legacyConfig) resolved {
	r := resolved{baseURL: sp.defaultURL, model: sp.defaultModel, enabled: true}

	if legacy.apiKey != "" && classifyURL(legacy.baseURL) == sp.id {
		r.apiKey = legacy.apiKey
		if legacy.baseURL != "" {
			r.baseURL = legacy.baseURL
		}
		if legacy.model != "" {
			r.model = legacy.model
		}
	}

	for _, k := range sp.envKeys {
		if v := envValue(k); v != "" {
			r.apiKey = v
			break
		}
	}

	if pc.APIKey != "" {
		r.apiKey = pc.APIKey
	}
	if pc.BaseURL != "" {
		r.baseURL = pc.BaseURL
	}
	if pc.Model != "" {
		r.model = pc.Model
	}
	if pc.Enabled != nil {
		r.enabled = *pc.Enabled
	}

	return r
}

// candidate is a provider together with its availability assessment.
type candidate struct {
	id          string
	displayName string
	provider    Provider
	available   bool
	reason      string // why unavailable; "" when available
}

// candidatesInOrder builds the candidate list for the given priority order,
// running each provider's cheap Available() health check.
func candidatesInOrder(ctx context.Context, s *Settings, order []string) []candidate {
	legacy := loadLegacy(s)
	out := make([]candidate, 0, len(order))

	for _, id := range order {
		sp, ok := specs[id]
		if !ok {
			continue
		}
		pc := s.Providers[id]
		c := candidate{id: id, displayName: sp.displayName}

		switch sp.kind {
		case kindClaudeCode:
			enabled := pc.Enabled == nil || *pc.Enabled
			prov := newClaudeCodeProvider(pc.Model)
			c.provider = prov
			c.available, c.reason = assess(ctx, prov, enabled)
		case kindHTTP:
			r := resolveHTTP(sp, pc, legacy)
			prov := newHTTPProvider(id, r.apiKey, r.baseURL, r.model)
			c.provider = prov
			c.available, c.reason = assess(ctx, prov, r.enabled)
		}

		out = append(out, c)
	}
	return out
}

// assess runs a provider's health check unless it is disabled.
func assess(ctx context.Context, p Provider, enabled bool) (available bool, reason string) {
	if !enabled {
		return false, "disabled in settings"
	}
	if err := p.Available(ctx); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// effectiveOrder resolves the provider priority order from, in order: a forced
// single provider, the GITREPORT_PROVIDER / GITREPORT_PROVIDER_PRIORITY env
// vars, setting.json priority, and finally DefaultPriority. Partial orders are
// completed with the remaining known providers so nothing is silently dropped.
func effectiveOrder(force string, s *Settings) []string {
	if force != "" {
		return []string{force}
	}
	if v := envValue("GITREPORT_PROVIDER"); v != "" {
		return []string{v}
	}
	if v := envValue("GITREPORT_PROVIDER_PRIORITY"); v != "" {
		return completeOrder(splitCSV(v))
	}
	if len(s.Priority) > 0 {
		return completeOrder(s.Priority)
	}
	return DefaultPriority
}

// completeOrder appends any known providers missing from the preferred list, in
// their default order, so an explicit but partial priority still allows fallback.
func completeOrder(preferred []string) []string {
	seen := map[string]bool{}
	var order []string
	for _, id := range preferred {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		order = append(order, id)
	}
	for _, id := range DefaultPriority {
		if !seen[id] {
			order = append(order, id)
		}
	}
	return order
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
