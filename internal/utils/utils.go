package utils

import (
	"log/slog"
	"os"
	"testing"
)

// Test utils
func EnforceManualTest(t *testing.T) {
	if os.Getenv("RUN_MANUAL_TEST") != "1" {
		t.Skip("set RUN_MANUAL_TEST=1 to run manual LLM tests")
	}
}

// GCP utils
func LoggerReplaceLevelWithSeverity(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		level, ok := a.Value.Any().(slog.Level)
		if !ok {
			return a
		}
		switch {
		case level >= slog.LevelError:
			a.Value = slog.StringValue("ERROR")
		case level >= slog.LevelWarn:
			a.Value = slog.StringValue("WARNING")
		case level >= slog.LevelInfo:
			a.Value = slog.StringValue("INFO")
		default:
			a.Value = slog.StringValue("DEBUG")
		}
	case slog.MessageKey:
		a.Key = "message"
	}
	return a
}

// Helper functions
func TruncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
