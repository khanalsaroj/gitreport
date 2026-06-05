package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestValidateTimeFlags(t *testing.T) {
	tests := []struct {
		name            string
		week, days, mon int
		wantErr         bool
	}{
		{name: "week only", week: 1},
		{name: "days only", days: 3},
		{name: "month only", mon: 2},
		{name: "none", wantErr: true},
		{name: "two set", week: 1, days: 1, wantErr: true},
		{name: "all set", week: 1, days: 1, mon: 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagWeek, flagDays, flagMonth = tt.week, tt.days, tt.mon
			t.Cleanup(func() { flagWeek, flagDays, flagMonth = 0, 0, 0 })

			err := validateTimeFlags()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTimeFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteIfAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	var buf bytes.Buffer
	if err := writeIfAbsent(&buf, path, []byte("payload"), 0o600); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if !strings.Contains(buf.String(), "Created") {
		t.Errorf("expected 'Created' message, got %q", buf.String())
	}
	if data, _ := os.ReadFile(path); string(data) != "payload" {
		t.Errorf("file content = %q, want payload", data)
	}

	// Second call must not overwrite and must report skipping.
	buf.Reset()
	if err := writeIfAbsent(&buf, path, []byte("different"), 0o600); err != nil {
		t.Fatalf("second write: %v", err)
	}
	if !strings.Contains(buf.String(), "skipping") {
		t.Errorf("expected 'skipping' message, got %q", buf.String())
	}
	if data, _ := os.ReadFile(path); string(data) != "payload" {
		t.Errorf("file was overwritten: %q", data)
	}
}

func TestRunInitWritesEmbeddedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows home

	c := &cobra.Command{}
	var buf bytes.Buffer
	c.SetOut(&buf)

	if err := runInit(c, nil); err != nil {
		t.Fatalf("runInit error: %v", err)
	}

	settingPath := filepath.Join(home, ".gitreport", "setting.json")
	configPath := filepath.Join(home, ".gitreport", "config", "gitreport.yaml")
	for _, p := range []string{settingPath, configPath} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
	if data, _ := os.ReadFile(configPath); len(data) == 0 {
		t.Error("config file is empty; embedded default not written")
	}
}
