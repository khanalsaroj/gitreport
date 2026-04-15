package summarizer

import "strings"

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

// splitOnDiffBoundary splits a diff string at "diff --git" markers.
func splitOnDiffBoundary(text string) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") && current.Len() > 0 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}
	return sections
}

// hardSplit splits a string into chunks of maxChars, breaking at newlines where possible.
func hardSplit(text string, maxChars int) []string {
	var chunks []string
	for len(text) > maxChars {
		// Find the last newline within the limit
		cut := maxChars
		if idx := strings.LastIndex(text[:cut], "\n"); idx > 0 {
			cut = idx + 1
		}
		chunks = append(chunks, text[:cut])
		text = text[cut:]
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}
