package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// WriteJSON collects a full streamed response and wraps it in a JSON envelope.
// Used when --format=json is requested.
func WriteJSON(ch <-chan string, output string) error {
	var sb strings.Builder
	for chunk := range ch {
		sb.WriteString(chunk)
	}

	envelope := map[string]string{
		"report": sb.String(),
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	var dest io.Writer
	if output == "" {
		dest = os.Stdout
	} else {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		dest = f
	}

	_, err = fmt.Fprintln(dest, string(data))
	return err
}
