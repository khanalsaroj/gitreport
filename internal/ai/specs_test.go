package ai

import (
	"context"
	"reflect"
	"testing"
)

// clearProviderEnv unsets every environment variable that influences provider
// resolution, so tests are deterministic regardless of the host machine.
func clearProviderEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL",
		"GITREPORT_OPENAI_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY",
		"XAI_API_KEY", "GROK_API_KEY", "OPENROUTER_API_KEY",
		"GITREPORT_PROVIDER", "GITREPORT_PROVIDER_PRIORITY",
	} {
		t.Setenv(k, "")
	}
}

func TestClassifyURL(t *testing.T) {
	tests := map[string]string{
		"https://openrouter.ai/api/v1/chat/completions":               ProviderOpenRouter,
		"https://api.openai.com/v1/chat/completions":                  ProviderOpenAI,
		"https://api.x.ai/v1/chat/completions":                        ProviderGrok,
		"https://generativelanguage.googleapis.com/v1beta/openai/...": ProviderGemini,
		"https://my-proxy.internal/v1":                                ProviderOpenRouter,
		"":                                                            ProviderOpenRouter,
	}
	for url, want := range tests {
		if got := classifyURL(url); got != want {
			t.Errorf("classifyURL(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestResolveHTTPLegacyMapping(t *testing.T) {
	clearProviderEnv(t)
	legacy := legacyConfig{apiKey: "or-key", baseURL: "https://openrouter.ai/api/v1/chat/completions", model: "m1"}

	// The legacy OpenRouter trio configures the openrouter provider...
	or := resolveHTTP(specs[ProviderOpenRouter], ProviderConfig{}, legacy)
	if or.apiKey != "or-key" || or.model != "m1" {
		t.Errorf("openrouter resolved = %+v, want legacy values applied", or)
	}

	// ...but not the openai provider (avoids the OPENAI_API_KEY collision).
	oa := resolveHTTP(specs[ProviderOpenAI], ProviderConfig{}, legacy)
	if oa.apiKey != "" {
		t.Errorf("openai should not inherit the legacy OpenRouter key, got %q", oa.apiKey)
	}
}

func TestResolveHTTPPrecedence(t *testing.T) {
	clearProviderEnv(t)
	legacy := legacyConfig{apiKey: "L", baseURL: "https://openrouter.ai/x"}

	// legacy only.
	if r := resolveHTTP(specs[ProviderOpenRouter], ProviderConfig{}, legacy); r.apiKey != "L" {
		t.Errorf("apiKey = %q, want legacy L", r.apiKey)
	}

	// env overrides legacy.
	t.Setenv("OPENROUTER_API_KEY", "E")
	if r := resolveHTTP(specs[ProviderOpenRouter], ProviderConfig{}, legacy); r.apiKey != "E" {
		t.Errorf("apiKey = %q, want env E", r.apiKey)
	}

	// explicit settings override env.
	pc := ProviderConfig{APIKey: "P", Model: "pm"}
	r := resolveHTTP(specs[ProviderOpenRouter], pc, legacy)
	if r.apiKey != "P" || r.model != "pm" {
		t.Errorf("resolved = %+v, want explicit settings P/pm", r)
	}
}

func TestResolveHTTPDisabled(t *testing.T) {
	clearProviderEnv(t)
	no := false
	r := resolveHTTP(specs[ProviderGemini], ProviderConfig{APIKey: "g", Enabled: &no}, legacyConfig{})
	if r.enabled {
		t.Error("provider should be disabled when Enabled=false")
	}
}

func TestEffectiveOrder(t *testing.T) {
	clearProviderEnv(t)

	if got := effectiveOrder("openai", &Settings{}); !reflect.DeepEqual(got, []string{"openai"}) {
		t.Errorf("force = %v, want [openai]", got)
	}

	if got := effectiveOrder("", &Settings{}); !reflect.DeepEqual(got, DefaultPriority) {
		t.Errorf("default = %v, want DefaultPriority", got)
	}

	// Partial settings priority is completed with the remaining providers.
	got := effectiveOrder("", &Settings{Priority: []string{"openrouter"}})
	want := []string{"openrouter", "claude-code", "openai", "gemini", "grok"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("partial priority = %v, want %v", got, want)
	}
}

func TestEffectiveOrderEnvOverride(t *testing.T) {
	clearProviderEnv(t)
	t.Setenv("GITREPORT_PROVIDER", "grok")
	if got := effectiveOrder("", &Settings{Priority: []string{"openai"}}); !reflect.DeepEqual(got, []string{"grok"}) {
		t.Errorf("GITREPORT_PROVIDER should win, got %v", got)
	}
}

func TestCompleteOrderDedup(t *testing.T) {
	got := completeOrder([]string{"openai", "openai", "", "  gemini  "})
	want := []string{"openai", "gemini", "claude-code", "grok", "openrouter"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("completeOrder = %v, want %v", got, want)
	}
}

func TestCandidatesInOrderHTTPAvailability(t *testing.T) {
	clearProviderEnv(t)
	s := &Settings{
		Providers: map[string]ProviderConfig{
			ProviderOpenAI: {APIKey: "k", Model: "gpt"},
			// gemini left unconfigured -> unavailable
			ProviderOpenRouter: {APIKey: "or", Model: "m"},
		},
	}
	order := []string{ProviderOpenAI, ProviderGemini, ProviderOpenRouter}
	cands := candidatesInOrder(context.Background(), s, order)

	if len(cands) != 3 {
		t.Fatalf("got %d candidates, want 3", len(cands))
	}
	avail := map[string]bool{}
	for _, c := range cands {
		avail[c.id] = c.available
	}
	if !avail[ProviderOpenAI] || !avail[ProviderOpenRouter] {
		t.Errorf("configured providers should be available: %+v", avail)
	}
	if avail[ProviderGemini] {
		t.Error("unconfigured gemini should be unavailable")
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a, b ,,c ")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitCSV = %v, want %v", got, want)
	}
}
