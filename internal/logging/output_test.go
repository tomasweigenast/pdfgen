package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSetOutputFiltersByLevel(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  string
	}{
		{"warnings", slog.LevelWarn, "WARNING: careful\nERROR: broken\n"},
		{"info", slog.LevelInfo, "INFO: note\nWARNING: careful\nERROR: broken\n"},
		{"silent", Silent, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			restore := SetOutput(&buf, tt.level)
			Logger.Info("note")
			Warnf("careful")
			Errorf("broken")
			restore()
			Warnf("after restore")
			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetOutputKeepsAttributes(t *testing.T) {
	var buf bytes.Buffer
	defer SetOutput(&buf, slog.LevelWarn)()
	Logger.With("file", "a.css").Warn("ignored", "line", 3)
	if got, want := buf.String(), "WARNING: ignored file=a.css line=3\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestSetProgressOutput(t *testing.T) {
	var buf bytes.Buffer
	restore := SetProgressOutput(&buf)
	Progressf("Step %d - Laying out", 4)
	restore()
	Progressf("hidden")
	if got := buf.String(); !strings.Contains(got, "Step 4 - Laying out") || strings.Contains(got, "hidden") {
		t.Errorf("progress output = %q", got)
	}
}
