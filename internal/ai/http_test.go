package ai

import "testing"

func TestValidateBaseURL(t *testing.T) {
	valid := []string{
		"https://openrouter.ai/api/v1/chat/completions",
		"http://localhost:11434/v1/chat/completions",
		"https://example.com",
	}
	for _, u := range valid {
		if err := validateBaseURL(u); err != nil {
			t.Errorf("validateBaseURL(%q) = %v, want nil", u, err)
		}
	}

	invalid := []string{
		"",                   // empty
		"openrouter.ai/api",  // no scheme/host
		"file:///etc/passwd", // non-http scheme
		"ftp://example.com",  // non-http scheme
		"://missing-scheme",  // malformed
	}
	for _, u := range invalid {
		if err := validateBaseURL(u); err == nil {
			t.Errorf("validateBaseURL(%q) = nil, want error", u)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "x", "y"); got != "x" {
		t.Errorf("firstNonEmpty = %q, want x", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("firstNonEmpty(all empty) = %q, want empty", got)
	}
	if got := firstNonEmpty("a", "b"); got != "a" {
		t.Errorf("firstNonEmpty = %q, want a", got)
	}
}
