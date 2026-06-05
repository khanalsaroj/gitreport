package summarizer

import (
	"strings"
	"unicode/utf8"
)

// Chunker splits large text inputs into token-approximate chunks.
type Chunker struct {
	// maxTokens is the approximate token limit per chunk.
	// We use a 4-char-per-token heuristic for estimation.
	maxTokens int
}

// NewChunker creates a Chunker with the given token limit per chunk.
func NewChunker(maxTokens int) *Chunker {
	return &Chunker{maxTokens: maxTokens}
}

// Split divides text into chunks that respect approximate token limits.
// It tries to split on diff boundaries (lines starting with "diff --git")
// to keep related changes together.
func (c *Chunker) Split(text string) []string {
	maxChars := c.maxTokens * 4 // ~4 chars per token

	if len(text) <= maxChars {
		return []string{text}
	}

	// Try to split on diff file boundaries
	sections := splitOnDiffBoundary(text)

	var chunks []string
	var current strings.Builder

	for _, section := range sections {
		// If adding this section exceeds the limit, flush current chunk
		if current.Len() > 0 && current.Len()+len(section) > maxChars {
			chunks = append(chunks, current.String())
			current.Reset()
		}

		// If a single section is too large, hard-split it
		if len(section) > maxChars {
			// Flush anything accumulated
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
			hardChunks := hardSplit(section, maxChars)
			chunks = append(chunks, hardChunks...)
			continue
		}

		current.WriteString(section)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// splitOnDiffBoundary splits a diff string at "diff --git" markers. The
// returned sections concatenate back to the exact input (no bytes are added or
// dropped); each section after the first begins with a "diff --git" line.
func splitOnDiffBoundary(text string) []string {
	if text == "" {
		return nil
	}

	const marker = "diff --git"
	var sections []string
	start := 0

	// i is always positioned at the start of a line (0, or just past a '\n').
	for i := 0; i < len(text); {
		if i > start && strings.HasPrefix(text[i:], marker) {
			sections = append(sections, text[start:i])
			start = i
		}
		nl := strings.IndexByte(text[i:], '\n')
		if nl < 0 {
			break
		}
		i += nl + 1
	}

	return append(sections, text[start:])
}

// hardSplit splits a string into chunks of at most maxChars bytes, preferring
// to break at a newline. When no newline is available within the limit it falls
// back to the nearest UTF-8 rune boundary so a multi-byte character is never
// split across two chunks.
func hardSplit(text string, maxChars int) []string {
	var chunks []string
	for len(text) > maxChars {
		cut := maxChars
		if idx := strings.LastIndex(text[:cut], "\n"); idx > 0 {
			cut = idx + 1
		} else {
			for cut > 1 && !utf8.RuneStart(text[cut]) {
				cut--
			}
		}
		chunks = append(chunks, text[:cut])
		text = text[cut:]
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}
