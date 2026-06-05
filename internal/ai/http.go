package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// httpProvider talks to any OpenAI-compatible chat-completions endpoint. A
// single implementation backs OpenAI, Gemini, Grok, and OpenRouter; they differ
// only in name, base URL, model, and credentials.
type httpProvider struct {
	name    string
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// newHTTPProvider builds an OpenAI-compatible provider. The HTTP client carries
// a generous timeout because report generation streams for a while.
func newHTTPProvider(name, apiKey, baseURL, model string) *httpProvider {
	return &httpProvider{
		name:    name,
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *httpProvider) Name() string { return p.name }

// Available verifies the provider is configured without making a network call.
func (p *httpProvider) Available(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("%s: API key not configured", p.name)
	}
	if p.model == "" {
		return fmt.Errorf("%s: model not configured", p.name)
	}
	return validateBaseURL(p.baseURL)
}

// NewOpenAIProvider creates a provider from the legacy flat configuration
// (OPENAI_* environment variables, falling back to setting.json). It is kept
// for backward compatibility and direct single-provider use.
func NewOpenAIProvider() (*httpProvider, error) {
	cfg, _ := loadSettings()
	if cfg == nil {
		cfg = &Settings{}
	}

	apiKey := firstNonEmpty(envValue("OPENAI_API_KEY"), cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set; run `gitreport init` then add your key, or set the environment variable")
	}
	baseURL := firstNonEmpty(envValue("OPENAI_BASE_URL"), cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("OPENAI_BASE_URL is not set; set it in setting.json or the environment")
	}
	if err := validateBaseURL(baseURL); err != nil {
		return nil, err
	}
	model := firstNonEmpty(envValue("OPENAI_MODEL"), cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("OPENAI_MODEL is not set; set it in setting.json or the environment")
	}

	return newHTTPProvider("openai-compatible", apiKey, baseURL, model), nil
}

// validateBaseURL ensures the endpoint that receives the bearer token is a
// well-formed absolute http(s) URL. This avoids transmitting the API key to a
// malformed or unexpected (e.g. file://) destination.
func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base URL %q is not a valid URL: %w", raw, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("base URL %q must be an absolute http(s) URL", raw)
	}
	return nil
}

// firstNonEmpty returns the first argument that is not the empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// chatRequest is the JSON body sent to the chat completions endpoint.
type chatRequest struct {
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// sseChunk is a parsed server-sent event chunk from the chat completions stream.
type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// Stream implements AIProvider via server-sent events (SSE).
func (p *httpProvider) Stream(ctx context.Context, system, user string) (<-chan string, <-chan error) {
	textCh := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(textCh)
		defer close(errCh)
		if err := p.stream(ctx, system, user, textCh); err != nil {
			errCh <- err
		}
	}()

	return textCh, errCh
}

func (p *httpProvider) stream(ctx context.Context, system, user string, out chan<- string) error {
	body := chatRequest{
		Model:  p.model,
		Stream: true,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	// OpenRouter uses these for attribution; other providers ignore them.
	req.Header.Set("HTTP-Referer", "https://github.com/khanalsaroj/gitreport")
	req.Header.Set("X-Title", "gitreport")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{Provider: p.name, StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	reader := bufio.NewReaderSize(resp.Body, 4096)

	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk sseChunk
			if jsonErr := json.Unmarshal([]byte(data), &chunk); jsonErr == nil {
				for _, choice := range chunk.Choices {
					if content := choice.Delta.Content; content != "" {
						select {
						case out <- content:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
			}
		}

		if err != nil {
			break // io.EOF or real error — stream ended
		}
	}

	return nil
}
