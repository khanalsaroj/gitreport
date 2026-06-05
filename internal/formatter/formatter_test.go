package formatter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func sendAll(chunks ...string) <-chan string {
	ch := make(chan string, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch
}

func TestWriteJSONToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	if err := WriteJSON(sendAll("hello ", "world"), path); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]string
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
	}
	if env["report"] != "hello world" {
		t.Errorf("report = %q, want %q", env["report"], "hello world")
	}
}

func TestWriterStreamToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.md")
	w := NewWriter(path, "markdown")
	if err := w.Stream(sendAll("# Title\n", "body")); err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "# Title\nbody\n"; got != want {
		t.Errorf("file content = %q, want %q", got, want)
	}
}

func TestWriterStreamJSONRouting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	w := NewWriter(path, "json")
	if err := w.Stream(sendAll("data")); err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]string
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("json format did not produce a JSON envelope: %v", err)
	}
	if env["report"] != "data" {
		t.Errorf("report = %q, want %q", env["report"], "data")
	}
}
