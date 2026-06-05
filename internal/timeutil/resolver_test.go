package timeutil

import (
	"testing"
	"time"
)

func TestResolve(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		week        int
		days        int
		month       int
		wantErr     bool
		wantApprox  time.Time
		toleranceOK bool
	}{
		{name: "one week", week: 1, wantApprox: now.AddDate(0, 0, -7), toleranceOK: true},
		{name: "three days", days: 3, wantApprox: now.AddDate(0, 0, -3), toleranceOK: true},
		{name: "two months", month: 2, wantApprox: now.AddDate(0, -2, 0), toleranceOK: true},
		{name: "none set", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.week, tt.days, tt.month)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Resolve calls time.Now() internally, so allow a small skew.
			if diff := got.Sub(tt.wantApprox); diff > time.Minute || diff < -time.Minute {
				t.Errorf("Resolve() = %v, want ~%v (diff %v)", got, tt.wantApprox, diff)
			}
		})
	}
}

func TestResolvePrecedence(t *testing.T) {
	// week is checked first; when set it wins over days/month.
	got, err := Resolve(1, 5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Now().AddDate(0, 0, -7)
	if diff := got.Sub(want); diff > time.Minute || diff < -time.Minute {
		t.Errorf("Resolve(1,5,5) = %v, want week semantics ~%v", got, want)
	}
}

func TestFormatGit(t *testing.T) {
	ts := time.Date(2025, 5, 11, 9, 30, 15, 0, time.UTC)
	if got, want := FormatGit(ts), "2025-05-11T09:30:15"; got != want {
		t.Errorf("FormatGit() = %q, want %q", got, want)
	}
}
