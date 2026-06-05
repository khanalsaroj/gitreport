package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseServer returns a test server that streams the given SSE data lines.
func sseServer(t *testing.T, lines ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("missing/incorrect Authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, l := range lines {
			fmt.Fprint(w, l)
		}
	}))
}

func TestHTTPProviderStreamSuccess(t *testing.T) {
	srv := sseServer(t,
		"data: {\"choices\":[{\"delta\":{\"content\":\"Hello \"}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n",
		"data: [DONE]\n\n",
	)
	defer srv.Close()

	p := newHTTPProvider("test", "test-key", srv.URL, "model-x")
	textCh, errCh := p.Stream(context.Background(), "sys", "usr")

	var sb strings.Builder
	for c := range textCh {
		sb.WriteString(c)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.String() != "Hello world" {
		t.Errorf("output = %q, want %q", sb.String(), "Hello world")
	}
}

func TestHTTPProviderStreamAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"slow down"}`)
	}))
	defer srv.Close()

	p := newHTTPProvider("test", "test-key", srv.URL, "model-x")
	textCh, errCh := p.Stream(context.Background(), "sys", "usr")

	for range textCh { // drain (should be empty)
	}
	err := <-errCh
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests || !apiErr.Temporary() {
		t.Errorf("got %+v, want a temporary 429", apiErr)
	}
	if !isRetryable(err) {
		t.Error("a 429 APIError should be retryable")
	}
}
