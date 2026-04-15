package formatter

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// Writer handles streaming output to stdout or a file.
type Writer struct {
	output     string
	format     string
	typewriter bool
	speed      string // "slow" | "normal" | "fast"
}

// NewWriter creates a new output writer.
func NewWriter(output, format string) *Writer {
	return &Writer{output: output, format: format, typewriter: true, speed: "fast"}
}

// delays returns per-character sleep durations based on speed setting.
func (w *Writer) delays() (char, space, newline time.Duration) {
	return 12 * time.Millisecond, 8 * time.Millisecond, 30 * time.Millisecond
}

// Stream consumes a text channel and writes output progressively.
func (w *Writer) Stream(ch <-chan string) error {
	if w.format == "json" {
		return WriteJSON(ch, w.output)
	}

	// File output — fast write, no effect
	if w.output != "" {
		f, err := os.Create(w.output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()

		bw := bufio.NewWriter(f)
		for chunk := range ch {
			if _, err := fmt.Fprint(bw, chunk); err != nil {
				return fmt.Errorf("writing output: %w", err)
			}
		}
		fmt.Fprintln(bw)
		return bw.Flush()
	}

	// Stdout — typewriter or plain streaming
	bw := bufio.NewWriter(os.Stdout)

	if w.typewriter {
		charDelay, spaceDelay, newlineDelay := w.delays()
		for chunk := range ch {
			for _, c := range chunk {
				fmt.Fprintf(bw, "%c", c)
				bw.Flush()
				switch c {
				case '\n':
					time.Sleep(newlineDelay)
				case ' ':
					time.Sleep(spaceDelay)
				default:
					time.Sleep(charDelay)
				}
			}
		}
	} else {
		// Plain streaming: flush every chunk so it appears immediately
		for chunk := range ch {
			fmt.Fprint(bw, chunk)
			bw.Flush()
		}
	}

	fmt.Fprintln(bw)
	return bw.Flush()
}
