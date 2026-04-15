package formatter

// Markdown formatting helpers.
// The AI is instructed to produce markdown when format=markdown,
// so this file exists for any post-processing or wrapping needed.

// WrapMarkdown wraps content in a markdown code fence if needed.
// Currently a no-op since the AI produces markdown directly.
func WrapMarkdown(content string) string {
	return content
}
