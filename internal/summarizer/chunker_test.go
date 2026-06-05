package summarizer

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitSmallReturnsSingleChunk(t *testing.T) {
	c := NewChunker(1000)
	text := "a small diff"
	chunks := c.Split(text)
	if len(chunks) != 1 || chunks[0] != text {
		t.Fatalf("Split() = %v, want single chunk equal to input", chunks)
	}
}

func TestSplitPreservesContent(t *testing.T) {
	c := NewChunker(50) // small limit to force multiple chunks (50*4=200 chars)
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("diff --git a/file")
		b.WriteByte(byte('0' + i%10))
		b.WriteString(" b/file\n")
		b.WriteString(strings.Repeat("+ added line\n", 5))
	}
	text := b.String()

	chunks := c.Split(text)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if got := strings.Join(chunks, ""); got != text {
		t.Errorf("rejoined chunks != original input")
	}
}

func TestSplitOnDiffBoundary(t *testing.T) {
	// Two files, each comfortably under the limit but together over it.
	maxTokens := 30 // 120 chars
	c := NewChunker(maxTokens)
	fileA := "diff --git a/a b/a\n" + strings.Repeat("+a\n", 30)
	fileB := "diff --git a/b b/b\n" + strings.Repeat("+b\n", 30)
	text := fileA + fileB

	chunks := c.Split(text)
	if strings.Join(chunks, "") != text {
		t.Fatalf("rejoined chunks != original")
	}
	// Each diff file should start a chunk boundary.
	for _, ch := range chunks {
		if !strings.HasPrefix(ch, "diff --git") {
			t.Errorf("chunk does not start at a diff boundary: %q", ch[:min(20, len(ch))])
		}
	}
}

func TestHardSplitLongLineUTF8Safe(t *testing.T) {
	// One enormous line of multibyte runes, no newline, exceeding the limit.
	maxChars := 100
	line := strings.Repeat("é", 500) // 2 bytes each => 1000 bytes
	chunks := hardSplit(line, maxChars)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for i, ch := range chunks {
		if !utf8.ValidString(ch) {
			t.Errorf("chunk %d is not valid UTF-8 (a rune was split)", i)
		}
	}
	if strings.Join(chunks, "") != line {
		t.Errorf("rejoined chunks != original")
	}
}

func TestHardSplitBreaksAtNewline(t *testing.T) {
	maxChars := 20
	text := strings.Repeat("abcdefghij\n", 5) // 11-char lines
	chunks := hardSplit(text, maxChars)
	if strings.Join(chunks, "") != text {
		t.Fatalf("rejoined chunks != original")
	}
	// Every chunk except possibly the last should end on a newline boundary.
	for i, ch := range chunks[:len(chunks)-1] {
		if !strings.HasSuffix(ch, "\n") {
			t.Errorf("chunk %d does not end at a newline: %q", i, ch)
		}
	}
}
