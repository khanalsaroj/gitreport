package ai

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// fakeProvider is a controllable Provider for chain tests.
type fakeProvider struct {
	name      string
	available error

	mu        sync.Mutex
	calls     int
	responses [][]string // per-call text chunks
	errs      []error    // per-call terminal error (nil = success)
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Available(ctx context.Context) error { return f.available }

func (f *fakeProvider) Stream(ctx context.Context, system, user string) (<-chan string, <-chan error) {
	f.mu.Lock()
	call := f.calls
	f.calls++
	var chunks []string
	if call < len(f.responses) {
		chunks = f.responses[call]
	}
	var rerr error
	if call < len(f.errs) {
		rerr = f.errs[call]
	}
	f.mu.Unlock()

	textCh := make(chan string, len(chunks)+1)
	errCh := make(chan error, 1)
	go func() {
		defer close(textCh)
		defer close(errCh)
		for _, c := range chunks {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case textCh <- c:
			}
		}
		if rerr != nil {
			errCh <- rerr
		}
	}()
	return textCh, errCh
}

func (f *fakeProvider) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// drain collects the chain's output and final error.
func drain(textCh <-chan string, errCh <-chan error) (string, error) {
	var sb strings.Builder
	for c := range textCh {
		sb.WriteString(c)
	}
	return sb.String(), <-errCh
}

func TestChainFirstProviderWins(t *testing.T) {
	a := &fakeProvider{name: "a", responses: [][]string{{"hello ", "world"}}, errs: []error{nil}}
	b := &fakeProvider{name: "b", responses: [][]string{{"unused"}}}
	c := newChain([]Provider{a, b}, nil)

	out, err := drain(c.Stream(context.Background(), "s", "u"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello world" {
		t.Errorf("output = %q, want %q", out, "hello world")
	}
	if b.callCount() != 0 {
		t.Errorf("fallback provider should not be called when primary succeeds")
	}
}

func TestChainFallsBackOnPreTokenError(t *testing.T) {
	// a fails before any token with a non-retryable error; b succeeds.
	a := &fakeProvider{name: "a", responses: [][]string{nil}, errs: []error{errors.New("auth failed")}}
	b := &fakeProvider{name: "b", responses: [][]string{{"from b"}}, errs: []error{nil}}
	c := newChain([]Provider{a, b}, nil)

	out, err := drain(c.Stream(context.Background(), "s", "u"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "from b" {
		t.Errorf("output = %q, want fallback output", out)
	}
	if a.callCount() != 1 {
		t.Errorf("non-retryable error should not be retried, got %d calls", a.callCount())
	}
}

func TestChainRetriesTransientThenFallsBack(t *testing.T) {
	// a returns a retryable APIError on every attempt (maxRetries+1 calls), then b wins.
	rateLimited := &APIError{Provider: "a", StatusCode: 429}
	a := &fakeProvider{
		name:      "a",
		responses: [][]string{nil, nil, nil},
		errs:      []error{rateLimited, rateLimited, rateLimited},
	}
	b := &fakeProvider{name: "b", responses: [][]string{{"ok"}}, errs: []error{nil}}
	c := newChain([]Provider{a, b}, nil)
	c.backoff = 0 // no delay in tests

	out, err := drain(c.Stream(context.Background(), "s", "u"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Errorf("output = %q, want b's output", out)
	}
	if a.callCount() != c.maxRetries+1 {
		t.Errorf("a called %d times, want %d (initial + retries)", a.callCount(), c.maxRetries+1)
	}
}

func TestChainAllProvidersFail(t *testing.T) {
	a := &fakeProvider{name: "a", responses: [][]string{nil}, errs: []error{errors.New("boom a")}}
	b := &fakeProvider{name: "b", responses: [][]string{nil}, errs: []error{errors.New("boom b")}}
	c := newChain([]Provider{a, b}, nil)

	out, err := drain(c.Stream(context.Background(), "s", "u"))
	if out != "" {
		t.Errorf("expected no output, got %q", out)
	}
	if err == nil || !strings.Contains(err.Error(), "all AI providers failed") {
		t.Fatalf("expected aggregate failure, got %v", err)
	}
}

func TestChainPostTokenErrorIsForwarded(t *testing.T) {
	// a emits a token then fails: the chain has committed, so it forwards the
	// partial text and surfaces the error rather than falling back.
	a := &fakeProvider{name: "a", responses: [][]string{{"partial"}}, errs: []error{errors.New("mid-stream")}}
	b := &fakeProvider{name: "b", responses: [][]string{{"unused"}}}
	c := newChain([]Provider{a, b}, nil)

	out, err := drain(c.Stream(context.Background(), "s", "u"))
	if out != "partial" {
		t.Errorf("output = %q, want the committed partial output", out)
	}
	if err == nil || !strings.Contains(err.Error(), "mid-stream") {
		t.Errorf("post-token error should be surfaced, got %v", err)
	}
	if b.callCount() != 0 {
		t.Errorf("must not fall back after committing to a provider")
	}
}
