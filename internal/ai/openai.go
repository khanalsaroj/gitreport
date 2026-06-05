package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpenAIProvider implements AIProvider using the OpenAI-compatible chat completions API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type settings struct {
	APIKey  string `json:"OPENAI_API_KEY"`
	BaseURL string `json:"OPENAI_BASE_URL"`
	Model   string `json:"OPENAI_MODEL"`
}

func loadSettings() (*settings, error) {
	home, err := homeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	settingsPath := filepath.Join(home, ".gitreport", "setting.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("read settings file %q: %w", settingsPath, err)
	}

	var cfg settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse settings file %q: %w", settingsPath, err)
	}

	return &cfg, nil
}

// NewOpenAIProvider creates a provider, sourcing credentials from environment
// variables first and falling back to ~/.gitreport/setting.json. Environment
// variables always take precedence so they can override file settings in CI.
func NewOpenAIProvider() (*OpenAIProvider, error) {
	var fileKey, fileURL, fileModel string
	if cfg, _ := loadSettings(); cfg != nil {
		fileKey, fileURL, fileModel = cfg.APIKey, cfg.BaseURL, cfg.Model
	}

	apiKey := firstNonEmpty(os.Getenv("OPENAI_API_KEY"), fileKey)
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set; run `gitreport init` then add your key, or set the environment variable")
	}

	baseURL := firstNonEmpty(os.Getenv("OPENAI_BASE_URL"), fileURL)
	if baseURL == "" {
		return nil, fmt.Errorf("OPENAI_BASE_URL is not set; set it in setting.json or the environment")
	}

	model := firstNonEmpty(os.Getenv("OPENAI_MODEL"), fileModel)
	if model == "" {
		return nil, fmt.Errorf("OPENAI_MODEL is not set; set it in setting.json or the environment")
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}, nil
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

// homeDir returns the current user's home directory on Windows, macOS, and Linux.
func homeDir() (string, error) {
	return os.UserHomeDir()
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

// sseEvent is a parsed server-sent event chunk.
type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// Stream implements AIProvider.Stream via server-sent events (SSE).
func (p *OpenAIProvider) Stream(ctx context.Context, system, user string) (<-chan string, <-chan error) {
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

func (p *OpenAIProvider) stream(ctx context.Context, system, user string, out chan<- string) error {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	// OpenRouter requires these headers
	req.Header.Set("HTTP-Referer", "https://github.com/khanalsaroj/gitreport")
	req.Header.Set("X-Title", "gitreport")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
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
