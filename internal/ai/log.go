package ai

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger returns a logger for provider selection and fallback diagnostics.
// It writes to stderr so report output on stdout stays clean. The level is WARN
// by default, DEBUG when verbose is set, and can be overridden with the
// GITREPORT_LOG environment variable (debug|info|warn|error).
func NewLogger(verbose bool) *slog.Logger {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	if v := envValue("GITREPORT_LOG"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
