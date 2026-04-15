package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/khanalsaroj/gitreport/internal/ai"
	"github.com/khanalsaroj/gitreport/internal/config"
)

// HardSummary generates engineering reports from code diffs using chunking.
type HardSummary struct {
	provider ai.AIProvider
	cfg      *config.Config
	chunker  *Chunker
}

// NewHardSummary creates a new HardSummary summarizer.
func NewHardSummary(provider ai.AIProvider, cfg *config.Config) *HardSummary {
	return &HardSummary{
		provider: provider,
		cfg:      cfg,
		chunker:  NewChunker(3000),
	}
}

// Stream generates a streaming leadership-level report from diff data.
func (s *HardSummary) Stream(ctx context.Context, activity, format string, byAuthor string) (<-chan string, error) {
	prompt, err := s.cfg.GetPrompt("hard_summary_prompt")
	if err != nil {
		return nil, fmt.Errorf("getting prompt: %w", err)
	}

	if byAuthor == "" {
		byAuthor = "ALL"
	}

	chunks := s.chunker.Split(activity) //TODO
	var finalActivity string

	if len(chunks) > 1 {
		intermediates, err := s.summarizeChunks(ctx, chunks)
		if err != nil {
			return nil, fmt.Errorf("summarizing chunks: %w", err)
		}
		finalActivity = strings.Join(intermediates, "\n\n---\n\n")
	} else {
		finalActivity = activity
	}

	// Final pass: produce the structured report
	system := buildPrompt(prompt.System, map[string]string{
		"author": byAuthor,
	})
	formatSpec := getFormatSpec(s.cfg, format)
	user := buildPrompt(prompt.User, map[string]string{
		"format":   formatSpec,
		"activity": finalActivity,
	})
	user += buildOutputSpec(&prompt.Output)

	textCh, errCh := s.provider.Stream(ctx, system, user)

	outCh := make(chan string, 64)
	go func() {
		defer close(outCh)
		for chunk := range textCh {
			outCh <- chunk
		}
		if err := <-errCh; err != nil {
			outCh <- fmt.Sprintf("\n[ERROR: %s]\n", err)
		}
	}()

	return outCh, nil
}

// summarizeChunks processes each diff chunk and returns intermediate summaries.
func (s *HardSummary) summarizeChunks(
	ctx context.Context,
	chunks []string,
) ([]string, error) {

	prompt, err := s.cfg.GetPrompt("changelog_prompt")
	if err != nil {
		return nil, fmt.Errorf("getting prompt: %w", err)
	}

	chunkSystem := buildPrompt(prompt.System, map[string]string{})

	type result struct {
		index int
		text  string
		err   error
	}

	resultCh := make(chan result, len(chunks))

	for i, chunk := range chunks {
		i, chunk := i, chunk

		go func() {
			select {
			case <-ctx.Done():
				resultCh <- result{index: i, err: ctx.Err()}
				return
			default:
			}

			chunkUser := fmt.Sprintf("Chunk %d/%d:\n\n%s", i+1, len(chunks), chunk)
			textCh, errCh := s.provider.Stream(ctx, chunkSystem, chunkUser)

			var sb strings.Builder
			for text := range textCh {
				sb.WriteString(text)
			}

			if err := <-errCh; err != nil {
				resultCh <- result{index: i, err: fmt.Errorf("chunk %d: %w", i+1, err)}
				return
			}

			resultCh <- result{index: i, text: sb.String()}
		}()
	}

	summaries := make([]string, len(chunks))
	for i := 0; i < len(chunks); i++ {
		res := <-resultCh
		if res.err != nil {
			return nil, res.err // fail fast
		}
		summaries[res.index] = res.text
	}

	return summaries, nil
}
