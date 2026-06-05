package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Options configures provider selection.
type Options struct {
	// Force selects a single provider by ID, bypassing auto-detection and
	// fallback. Empty means automatic selection.
	Force string

	// Logger receives selection, fallback, and retry diagnostics. May be nil.
	Logger *slog.Logger
}

// Select detects the available AI providers and returns a Chain that streams
// from them in priority order with retries and graceful fallback. It returns an
// error only when no provider is available at all.
func Select(ctx context.Context, opts Options) (AIProvider, error) {
	s, err := loadSettings()
	if err != nil {
		return nil, err
	}

	order := effectiveOrder(opts.Force, s)
	cands := candidatesInOrder(ctx, s, order)

	var available []Provider
	for _, c := range cands {
		if c.available {
			available = append(available, c.provider)
		}
	}
	if len(available) == 0 {
		return nil, noProviderError(opts.Force, cands)
	}

	if opts.Logger != nil {
		names := make([]string, len(available))
		for i, p := range available {
			names[i] = p.Name()
		}
		opts.Logger.Info("AI providers selected", "primary", names[0], "fallback_order", names)
	}

	return newChain(available, opts.Logger), nil
}

// Status describes a provider's configuration state for the `providers` command.
type Status struct {
	ID          string
	DisplayName string
	Available   bool
	Detail      string // "ready" or the reason it is unavailable
	Primary     bool   // the provider that would be used first
}

// Statuses reports every known provider in effective priority order, annotated
// with availability. It powers `gitreport providers`.
func Statuses(ctx context.Context) ([]Status, error) {
	s, err := loadSettings()
	if err != nil {
		return nil, err
	}

	cands := candidatesInOrder(ctx, s, effectiveOrder("", s))
	out := make([]Status, len(cands))
	firstAvailable := true
	for i, c := range cands {
		detail := "ready"
		primary := false
		if !c.available {
			detail = c.reason
		} else if firstAvailable {
			primary = true
			firstAvailable = false
		}
		out[i] = Status{
			ID:          c.id,
			DisplayName: c.displayName,
			Available:   c.available,
			Detail:      detail,
			Primary:     primary,
		}
	}
	return out, nil
}

// noProviderError builds an actionable error listing why each provider was
// rejected.
func noProviderError(force string, cands []candidate) error {
	if force != "" {
		if len(cands) == 1 && cands[0].reason != "" {
			return fmt.Errorf("provider %q is not usable: %s", force, cands[0].reason)
		}
		return fmt.Errorf("provider %q is not a known provider", force)
	}

	var reasons []string
	for _, c := range cands {
		reasons = append(reasons, fmt.Sprintf("  - %s: %s", c.id, c.reason))
	}
	return fmt.Errorf(
		"no AI provider is available.\n%s\n\nRun `gitreport init`, then either install Claude Code or add an API key.",
		strings.Join(reasons, "\n"),
	)
}
