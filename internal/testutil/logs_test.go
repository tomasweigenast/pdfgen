package testutil

import (
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/logging"
)

func TestCaptureLogsCollectsLoggerOutput(t *testing.T) {
	logs := CaptureLogs(t)
	logging.Warnf("Ignored `%s`", "color: nope")
	if got, want := logs.Lines(), []string{"WARNING: Ignored `color: nope`"}; !slices.Equal(got, want) {
		t.Errorf("Lines() = %q, want %q", got, want)
	}
}

func TestAssertNoLogsPassesWhenQuiet(t *testing.T) {
	AssertNoLogs(t)
	logging.Progressf("progress is not captured")
}

func TestAssertNoLogsFailsOnWarning(t *testing.T) {
	var failed bool
	t.Run("inner", func(inner *testing.T) {
		// The inner test failing is the behaviour under test; report it
		// through a recorder rather than failing the outer run.
		AssertNoLogs(&failRecorder{TB: inner, failed: &failed})
		logging.Warnf("boom")
	})
	if !failed {
		t.Error("AssertNoLogs did not fail after a warning")
	}
}

type failRecorder struct {
	testing.TB
	failed *bool
}

func (r *failRecorder) Errorf(string, ...any) { *r.failed = true }
