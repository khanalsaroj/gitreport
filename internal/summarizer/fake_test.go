package summarizer

import (
	"context"
	"sync"
	"testing"

	"github.com/khanalsaroj/gitreport/internal/config"
	"gopkg.in/yaml.v3"
)

// fakeProvider is a test double for ai.AIProvider. It records the prompts it
// receives and replays a canned response. It is safe for the concurrent calls
// made by HardSummary's chunk fan-out.
type fakeProvider struct {
	mu sync.Mutex

	// respond, when set, produces the response for a given (system, user) call.
	respond func(system, user string) ([]string, error)

	// Defaults used when respond is nil.
	chunks []string
	err    error

	calls      int
	lastSystem string
	lastUser   string
}

func (f *fakeProvider) Stream(ctx context.Context, system, user string) (<-chan string, <-chan error) {
	f.mu.Lock()
	f.calls++
	f.lastSystem, f.lastUser = system, user
	chunks, rerr := f.chunks, f.err
	respond := f.respond
	f.mu.Unlock()

	if respond != nil {
		chunks, rerr = respond(system, user)
	}

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

// testConfig returns the real embedded default config, exercising the prompts
// the tool actually ships with.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	var cfg config.Config
	if err := yaml.Unmarshal(config.Default, &cfg); err != nil {
		t.Fatalf("unmarshal embedded default config: %v", err)
	}
	return &cfg
}

// collect drains a string channel into a single string.
func collect(ch <-chan string) string {
	var out string
	for s := range ch {
		out += s
	}
	return out
}
