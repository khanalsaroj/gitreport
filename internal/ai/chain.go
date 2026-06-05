package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Chain streams from a list of providers in priority order. It commits to the
// first provider that produces output; if a provider fails before emitting any
// text (unavailable, rate-limited, auth failure), the chain retries transient
// errors and otherwise falls back to the next provider. Errors are surfaced on
// the error channel — the chain never injects error text into the stream — so
// the caller's existing error handling is preserved.
type Chain struct {
	providers  []Provider
	logger     *slog.Logger
	maxRetries int
	backoff    time.Duration
}

func newChain(providers []Provider, logger *slog.Logger) *Chain {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Chain{
		providers:  providers,
		logger:     logger,
		maxRetries: 2,
		backoff:    500 * time.Millisecond,
	}
}

func (c *Chain) Stream(ctx context.Context, system, user string) (<-chan string, <-chan error) {
	outText := make(chan string, 64)
	outErr := make(chan error, 1)

	go func() {
		defer close(outText)
		defer close(outErr)

		var lastErr error
		for _, p := range c.providers {
			committed, err := c.tryProvider(ctx, p, system, user, outText, outErr)
			if committed {
				return
			}
			if ctx.Err() != nil {
				outErr <- ctx.Err()
				return
			}
			lastErr = err
			c.logger.Warn("provider unavailable, falling back",
				"provider", p.Name(), "error", err)
		}

		if lastErr == nil {
			lastErr = errors.New("no provider produced a response")
		}
		outErr <- fmt.Errorf("all AI providers failed: %w", lastErr)
	}()

	return outText, outErr
}

// tryProvider attempts a single provider, with retries for transient errors. It
// returns committed=true once it has begun forwarding that provider's output
// (after which fallback is no longer possible); any later error is forwarded on
// outErr. committed=false means the provider produced no output and the caller
// should fall back.
func (c *Chain) tryProvider(
	ctx context.Context,
	p Provider,
	system, user string,
	outText chan<- string,
	outErr chan<- error,
) (committed bool, err error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if attempt > 0 {
			c.logger.Info("retrying provider", "provider", p.Name(), "attempt", attempt)
			if !sleep(ctx, c.backoff*time.Duration(attempt)) {
				return false, ctx.Err()
			}
		}

		textCh, errCh := p.Stream(ctx, system, user)

		first, ok := <-textCh
		if !ok {
			// No text produced; inspect the error to decide retry vs fallback.
			e := drainErr(errCh)
			if e == nil {
				return true, nil // succeeded with empty output
			}
			lastErr = e
			if isRetryable(e) && attempt < c.maxRetries {
				continue
			}
			return false, e
		}

		// Committed: forward the first chunk and everything after it.
		c.logger.Debug("streaming from provider", "provider", p.Name())
		outText <- first
		for chunk := range textCh {
			outText <- chunk
		}
		if e := drainErr(errCh); e != nil {
			outErr <- e
		}
		return true, nil
	}

	return false, lastErr
}

// drainErr reads the single error a provider may emit, returning nil if the
// channel closed without one.
func drainErr(errCh <-chan error) error {
	if e, ok := <-errCh; ok {
		return e
	}
	return nil
}

// sleep waits for d or until ctx is cancelled. It reports whether the full wait
// elapsed.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
