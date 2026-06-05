package summarizer

import (
	"context"
	"strings"
	"testing"

	"github.com/khanalsaroj/gitreport/internal/config"
)

func TestSummaryStreamSuccess(t *testing.T) {
	fake := &fakeProvider{chunks: []string{"Hello ", "world"}}
	sum := NewSummary(fake, testConfig(t))

	ch, err := sum.Stream(context.Background(), "alice: fix bug", "markdown", "")
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	if got := collect(ch); got != "Hello world" {
		t.Errorf("output = %q, want %q", got, "Hello world")
	}

	// The commit data and the resolved format spec must reach the model.
	if !strings.Contains(fake.lastUser, "alice: fix bug") {
		t.Errorf("user prompt missing commit data: %q", fake.lastUser)
	}
	if !strings.Contains(fake.lastUser, "markdown") {
		t.Errorf("user prompt missing markdown format spec")
	}
	// Empty author defaults to ALL in the system prompt.
	if !strings.Contains(fake.lastSystem, "ALL") {
		t.Errorf("system prompt should mention ALL when no author filter: %q", fake.lastSystem)
	}
}

func TestSummaryStreamAuthorFilter(t *testing.T) {
	fake := &fakeProvider{chunks: []string{"ok"}}
	sum := NewSummary(fake, testConfig(t))

	ch, err := sum.Stream(context.Background(), "data", "text", "Bob")
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	collect(ch)
	if !strings.Contains(fake.lastSystem, "Bob") {
		t.Errorf("system prompt should mention the author filter Bob: %q", fake.lastSystem)
	}
}

func TestSummaryStreamSurfacesError(t *testing.T) {
	fake := &fakeProvider{chunks: []string{"partial"}, err: context.DeadlineExceeded}
	sum := NewSummary(fake, testConfig(t))

	ch, err := sum.Stream(context.Background(), "data", "text", "")
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	out := collect(ch)
	if !strings.Contains(out, "partial") || !strings.Contains(out, "ERROR") {
		t.Errorf("output should contain partial text and an error marker, got %q", out)
	}
}

func TestSummaryStreamMissingPrompt(t *testing.T) {
	cfg := &config.Config{Version: "1", Prompts: map[string]*config.Prompt{}}
	sum := NewSummary(&fakeProvider{}, cfg)

	if _, err := sum.Stream(context.Background(), "data", "text", ""); err == nil {
		t.Fatal("expected error when summary_prompt is absent")
	}
}

func TestBuildOutputSpec(t *testing.T) {
	minItems := 2
	maxItems := 5
	out := config.PromptOutput{
		Strict:            true,
		OmitEmptySections: true,
		Title:             "My Report",
		Structure: []config.OutputSection{
			{
				Title:       "Summary",
				Description: "the gist",
				Style:       "concise",
				Rules:       &config.OutputSectionRules{MinItems: &minItems, MaxItems: &maxItems},
			},
		},
		Rules: []string{"no fluff"},
	}

	spec := buildOutputSpec(&out)

	for _, want := range []string{
		"My Report",
		"Section: Summary",
		"the gist",
		"concise",
		"Minimum items: 2",
		"Maximum items: 5",
		"Follow structure strictly",
		"Omit empty sections",
		"no fluff",
	} {
		if !strings.Contains(spec, want) {
			t.Errorf("buildOutputSpec output missing %q\n---\n%s", want, spec)
		}
	}
}
