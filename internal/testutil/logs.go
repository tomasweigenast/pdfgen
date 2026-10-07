package testutil

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// CaptureLogs redirects the warning logger for the rest of the test and
// returns what it collected. Lines have the form "LEVEL: message".
func CaptureLogs(t testing.TB) *logging.Capture {
	t.Helper()
	c := logging.StartCapture()
	t.Cleanup(c.Close)
	return c
}

// AssertNoLogs fails the test at its end if anything was logged at info
// level or above while it ran.
func AssertNoLogs(t testing.TB) {
	t.Helper()
	c := logging.StartCapture()
	t.Cleanup(func() {
		c.Close()
		if lines := c.Lines(); len(lines) > 0 {
			t.Errorf("%d messages logged:\n%s", len(lines), strings.Join(lines, "\n"))
		}
	})
}
