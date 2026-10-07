package logging

import (
	"slices"
	"testing"
)

func TestCaptureRecordsWarningsAndRestores(t *testing.T) {
	c := StartCapture()
	Warnf("ignored %s", "declaration")
	Errorf("failed: %d", 3)
	Logger.Info("note")
	Logger.Debug("dropped")
	Progressf("step %d", 1)
	c.Close()

	if got, want := c.Warnings(), []string{"ignored declaration", "failed: 3"}; !slices.Equal(got, want) {
		t.Errorf("Warnings() = %q, want %q", got, want)
	}
	if got, want := c.Messages(), []string{"ignored declaration", "failed: 3", "note"}; !slices.Equal(got, want) {
		t.Errorf("Messages() = %q, want %q", got, want)
	}

	wantLines := []string{"WARNING: ignored declaration", "ERROR: failed: 3", "INFO: note"}
	if got := c.Lines(); !slices.Equal(got, wantLines) {
		t.Errorf("Lines() = %q, want %q", got, wantLines)
	}

	// Entries logged after Close must not reach the capture.
	Warnf("after close")
	if n := len(c.Records()); n != 3 {
		t.Errorf("records after close = %d, want 3", n)
	}
}

func TestCaptureSeesDerivedLoggers(t *testing.T) {
	c := StartCapture()
	defer c.Close()
	Logger.With("key", "value").WithGroup("g").Warn("derived")
	if got := c.Warnings(); !slices.Equal(got, []string{"derived"}) {
		t.Errorf("Warnings() = %q", got)
	}
}
