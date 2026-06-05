package formatter

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Writer streams a generated report to stdout or a file in the requested format.
type Writer struct {
	output string
	format string
}

// NewWriter creates a writer that sends output to the given file path, or to
// stdout when output is empty.
func NewWriter(output, format string) *Writer {
	return &Writer{output: output, format: format}
}

// Stream consumes ch and writes it progressively. JSON is buffered and emitted
// as a single envelope; every other format is streamed chunk-by-chunk at the
// model's natural cadence (no artificial per-character delay).
func (w *Writer) Stream(ch <-chan string) error {
	if w.format == "json" {
		return WriteJSON(ch, w.output)
	}

	dest, closeFn, err := w.dest()
	if err != nil {
		return err
	}
	defer closeFn()

	toStdout := w.output == ""
	bw := bufio.NewWriter(dest)
	for chunk := range ch {
		if _, err := io.WriteString(bw, chunk); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		// Flush each chunk to stdout so streamed output appears as it arrives.
		// File output is buffered and flushed once at the end for throughput.
		if toStdout {
			if err := bw.Flush(); err != nil {
				return fmt.Errorf("writing output: %w", err)
			}
		}
	}

	fmt.Fprintln(bw)
	return bw.Flush()
}

// dest returns the output destination and a cleanup function to run when done.
func (w *Writer) dest() (io.Writer, func() error, error) {
	if w.output == "" {
		return os.Stdout, func() error { return nil }, nil
	}
	f, err := os.Create(w.output)
	if err != nil {
		return nil, nil, fmt.Errorf("creating output file %q: %w", w.output, err)
	}
	return f, f.Close, nil
}
