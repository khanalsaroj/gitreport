package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/khanalsaroj/gitreport/internal/ai"
	"github.com/khanalsaroj/gitreport/internal/config"
)

// Summary generates engineering reports from commit messages.
type Summary struct {
	provider ai.AIProvider
	cfg      *config.Config
}

// NewSummary creates a new Summary summarizer.
func NewSummary(provider ai.AIProvider, cfg *config.Config) *Summary {
	return &Summary{provider: provider, cfg: cfg}
}

// Stream generates a streaming report from commit data.
func (s *Summary) Stream(ctx context.Context, commits, format string, byAuthor string) (<-chan string, error) {
	prompt, err := s.cfg.GetPrompt("summary_prompt")
	if err != nil {
		return nil, fmt.Errorf("getting prompt: %w", err)
	}

	if byAuthor == "" {
		byAuthor = "ALL"
	}

	formatSpec := s.cfg.FormatSpec(format)

	system := config.Render(prompt.System, config.Vars{
		"author": byAuthor,
	})

	user := config.Render(prompt.User, config.Vars{
		"format":  formatSpec,
		"commits": commits,
	})

	user += buildOutputSpec(&prompt.Output)

	textCh, errCh := s.provider.Stream(ctx, system, user)

	// Merge error into a single output channel
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

func buildOutputSpec(cfg *config.PromptOutput) string {
	var b strings.Builder

	b.WriteString("You MUST follow the exact structure below.\n")
	b.WriteString("Do not add or rename sections.\n")
	b.WriteString("Sections must appear in the exact order.\n\n")

	if cfg.Title != "" {
		b.WriteString(cfg.Title + "\n\n")
	}

	b.WriteString("OUTPUT STRUCTURE:\n\n")

	for _, s := range cfg.Structure {

		b.WriteString("Section: " + s.Title + "\n")

		if s.Description != "" {
			b.WriteString("- Description: " + s.Description + "\n")
		}

		if s.Style != "" {
			b.WriteString("- Style: " + s.Style + "\n")
		}

		if s.Rules != nil {
			b.WriteString("- Constraints:\n")

			if s.Rules.MinItems != nil {
				b.WriteString(fmt.Sprintf("  - Minimum items: %d\n", *s.Rules.MinItems))
			}
			if s.Rules.MaxItems != nil {
				b.WriteString(fmt.Sprintf("  - Maximum items: %d\n", *s.Rules.MaxItems))
			}
		}

		b.WriteString("\n")
	}

	// Global rules
	b.WriteString("GLOBAL RULES:\n")

	if cfg.Strict {
		b.WriteString("- Follow structure strictly\n")
	}
	if cfg.OmitEmptySections {
		b.WriteString("- Omit empty sections\n")
	}

	for _, r := range cfg.Rules {
		b.WriteString("- " + r + "\n")
	}

	return b.String()
}
