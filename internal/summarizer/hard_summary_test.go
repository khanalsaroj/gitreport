package summarizer

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestHardSummarySingleChunk(t *testing.T) {
	fake := &fakeProvider{chunks: []string{"Shipped ", "value."}}
	hs := NewHardSummary(fake, testConfig(t))

	activity := "=== Commit: abc\nAuthor: Ann <a@x>\ndiff --git a/f b/f\n+ line\n"
	ch, err := hs.Stream(context.Background(), activity, "markdown", "")
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	if got := collect(ch); got != "Shipped value." {
		t.Errorf("output = %q", got)
	}
	// Small input must not trigger the chunk fan-out: exactly one model call.
	if fake.callCount() != 1 {
		t.Errorf("call count = %d, want 1 for single-chunk input", fake.callCount())
	}
	// The raw activity should reach the model unchanged for single-chunk input.
	if !strings.Contains(fake.lastUser, "diff --git a/f b/f") {
		t.Errorf("activity did not reach the model: %q", fake.lastUser)
	}
}

func TestHardSummaryMultiChunkMapReduce(t *testing.T) {
	// Build activity larger than the chunker limit (3000 tokens => 12000 chars)
	// across several diff boundaries to force a map-reduce.
	var b strings.Builder
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&b, "diff --git a/file%d b/file%d\n", i, i)
		b.WriteString(strings.Repeat("+x\n", 4000))
	}
	activity := b.String()
	if len(activity) <= 12000 {
		t.Fatalf("test setup: activity must exceed the chunk limit, got %d", len(activity))
	}

	fake := &fakeProvider{
		respond: func(system, user string) ([]string, error) {
			// Distinguish the per-chunk review pass from the final report pass.
			if strings.Contains(system, "code review") {
				return []string{"chunk insight"}, nil
			}
			return []string{"FINAL REPORT"}, nil
		},
	}
	hs := NewHardSummary(fake, testConfig(t))

	ch, err := hs.Stream(context.Background(), activity, "markdown", "")
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	out := collect(ch)
	if out != "FINAL REPORT" {
		t.Errorf("output = %q, want the final-pass result", out)
	}
	// At least one chunk pass plus the final pass.
	if fake.callCount() < 2 {
		t.Errorf("call count = %d, want >= 2 (chunks + final)", fake.callCount())
	}
}

func TestHardSummaryChunkErrorPropagates(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&b, "diff --git a/file%d b/file%d\n", i, i)
		b.WriteString(strings.Repeat("+x\n", 4000))
	}

	fake := &fakeProvider{
		respond: func(system, user string) ([]string, error) {
			if strings.Contains(system, "code review") {
				return nil, fmt.Errorf("chunk boom")
			}
			return []string{"FINAL"}, nil
		},
	}
	hs := NewHardSummary(fake, testConfig(t))

	_, err := hs.Stream(context.Background(), b.String(), "markdown", "")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected chunk error to propagate, got %v", err)
	}
}
