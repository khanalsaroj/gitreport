package ai

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// AIProvider is the minimal streaming contract consumed by the summarizer.
// All implementations MUST support streaming output via channels: the text
// channel carries response chunks and is closed when the response ends; the
// error channel carries at most one error, sent after all text, then closed.
type AIProvider interface {
	Stream(ctx context.Context, system, user string) (<-chan string, <-chan error)
}

// Provider is a named, detectable AI backend. The selection layer works with
// this richer interface; the summarizer only needs the embedded AIProvider.
type Provider interface {
	AIProvider

	// Name returns a stable identifier used in configuration and logs
	// (e.g. "claude-code", "openai", "openrouter").
	Name() string

	// Available reports whether the provider is usable without making a paid
	// request: credentials present, binary on PATH, URL well-formed, etc. It
	// returns nil when the provider can be attempted, or an error explaining
	// why it cannot. It is the cheap "health check" used during selection.
	Available(ctx context.Context) error
}

// APIError is returned by HTTP providers for non-2xx responses. It preserves
// the status code so the fallback chain can decide whether to retry the same
// provider or move on to the next one.
type APIError struct {
	Provider   string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	body := e.Body
	if len(body) > 500 {
		body = body[:500] + "…"
	}
	return fmt.Sprintf("%s: API error %d: %s", e.Provider, e.StatusCode, body)
}

// Temporary reports whether the status code indicates a transient condition
// (rate limiting or a server-side error) that is worth retrying.
func (e *APIError) Temporary() bool {
	switch e.StatusCode {
	case 408, 409, 425, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// isRetryable reports whether err is worth retrying against the same provider.
// Rate limits, transient server errors, and network failures are retryable;
// context cancellation and client errors (bad key, bad request) are not.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Temporary()
	}

	// Any network-level error (timeout, refused, reset, DNS) is transient.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return false
}
