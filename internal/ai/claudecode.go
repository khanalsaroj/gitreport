package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const claudeCodeBinary = "claude"

// claudeCodeProvider drives the locally installed Claude Code CLI in
// non-interactive print mode (`claude -p --output-format json`). It reuses the
// CLI's own authentication — the user's Claude subscription or API key — so no
// API key is configured inside gitreport.
//
// Trade-offs versus the HTTP providers, documented in docs/PROVIDERS.md:
//   - The response is returned as a single block, not token-streamed, because
//     the CLI buffers the answer in JSON output mode.
//   - Each call has higher latency and cost than a raw API request because the
//     CLI starts a full session. Passing --system-prompt replaces the heavy
//     default agent prompt, and running in a neutral directory stops the CLI
//     from scanning the current repository for extra context.
type claudeCodeProvider struct {
	bin   string // executable name or path; resolved via PATH
	model string // optional; empty uses the CLI default
}

func newClaudeCodeProvider(model string) *claudeCodeProvider {
	return &claudeCodeProvider{bin: claudeCodeBinary, model: model}
}

func (p *claudeCodeProvider) Name() string { return ProviderClaudeCode }

// Available reports whether the claude binary is on PATH. It deliberately does
// not invoke the CLI: a real call costs money and several seconds, so an
// unauthenticated or failing CLI is detected lazily at stream time and triggers
// fallback to the next provider.
func (p *claudeCodeProvider) Available(ctx context.Context) error {
	if _, err := exec.LookPath(p.bin); err != nil {
		return fmt.Errorf("claude-code: %q not found on PATH", p.bin)
	}
	return nil
}

// claudeResult is the JSON object emitted by `claude -p --output-format json`.
type claudeResult struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

// Stream implements AIProvider. Output is delivered as a single chunk.
func (p *claudeCodeProvider) Stream(ctx context.Context, system, user string) (<-chan string, <-chan error) {
	textCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(textCh)
		defer close(errCh)

		out, err := p.run(ctx, system, user)
		if err != nil {
			errCh <- err
			return
		}
		if out != "" {
			select {
			case textCh <- out:
			case <-ctx.Done():
				errCh <- ctx.Err()
			}
		}
	}()

	return textCh, errCh
}

// claudeArgs builds the CLI arguments for a print-mode request. The user
// message is supplied separately via stdin.
func claudeArgs(system, model string) []string {
	args := []string{"--print", "--output-format", "json"}
	if system != "" {
		// Replace (not append to) the default agent system prompt.
		args = append(args, "--system-prompt", system)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

func (p *claudeCodeProvider) run(ctx context.Context, system, user string) (string, error) {
	cmd := exec.CommandContext(ctx, p.bin, claudeArgs(system, p.model)...)
	cmd.Stdin = strings.NewReader(user)
	cmd.Dir = os.TempDir() // neutral working directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("claude-code: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return parseClaudeResult(stdout.Bytes())
}

// parseClaudeResult extracts the response text from the CLI's JSON output.
func parseClaudeResult(raw []byte) (string, error) {
	var res claudeResult
	if err := json.Unmarshal(bytes.TrimSpace(raw), &res); err != nil {
		return "", fmt.Errorf("claude-code: parsing CLI output: %w", err)
	}
	if res.IsError || (res.Subtype != "" && res.Subtype != "success") {
		msg := firstNonEmpty(res.Result, res.Subtype, "unknown error")
		return "", fmt.Errorf("claude-code: %s", msg)
	}
	return res.Result, nil
}
