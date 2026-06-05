package ai

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestClaudeArgs(t *testing.T) {
	got := claudeArgs("be terse", "claude-opus-4-8")
	want := []string{"--print", "--output-format", "json", "--system-prompt", "be terse", "--model", "claude-opus-4-8"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("claudeArgs = %v, want %v", got, want)
	}

	// Empty system and model are omitted.
	got = claudeArgs("", "")
	want = []string{"--print", "--output-format", "json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("claudeArgs(empty) = %v, want %v", got, want)
	}
}

func TestParseClaudeResult(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "success",
			input: `{"type":"result","subtype":"success","is_error":false,"result":"the report"}`,
			want:  "the report",
		},
		{
			name:    "is_error true",
			input:   `{"type":"result","subtype":"success","is_error":true,"result":"rate limited"}`,
			wantErr: true,
		},
		{
			name:    "non-success subtype",
			input:   `{"type":"result","subtype":"error_max_turns","is_error":false}`,
			wantErr: true,
		},
		{
			name:    "malformed json",
			input:   `not json`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseClaudeResult([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got result %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClaudeCodeProviderName(t *testing.T) {
	if got := newClaudeCodeProvider("").Name(); got != ProviderClaudeCode {
		t.Errorf("Name() = %q, want %q", got, ProviderClaudeCode)
	}
}

func TestClaudeCodeUnavailableWhenBinaryMissing(t *testing.T) {
	p := &claudeCodeProvider{bin: "gitreport-no-such-binary-xyz"}
	err := p.Available(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Available() = %v, want a not-found error", err)
	}
}
