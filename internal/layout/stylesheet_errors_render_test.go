package layout_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func TestUnloadableLinkedStylesheetIsReportedWhenRendering(t *testing.T) {
	testutil.AssertNoLogs(t)
	logs := testutil.CaptureLogs(t)
	boxtest.RenderPages(t, `<link rel=stylesheet href=invalid-protocol://absolute>`)
	lines := logs.Lines()
	if len(lines) != 1 || !strings.Contains(lines[0], "ERROR: Failed to load stylesheet at") {
		t.Errorf("logs = %q, want one stylesheet error", lines)
	}
}

func TestInvalidDeclarationsDoNotStopTheCascadeWhenRendering(t *testing.T) {
	for _, source := range []string{
		`<style> html { color red; color: blue; color`,
		`<html style="color; color: blue; color red">`,
	} {
		t.Run(source, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			logs := testutil.CaptureLogs(t)
			html := child(t, onlyPage(t, boxtest.RenderPages(t, source)))
			c := html.Base().Style.Color()
			if got := [4]float64{c.Coordinates[0], c.Coordinates[1], c.Coordinates[2], c.Alpha}; got != [4]float64{0, 0, 1, 1} {
				t.Errorf("color = %v, want blue", got)
			}
			if lines := logs.Lines(); len(lines) != 2 {
				t.Errorf("logs = %q, want 2 messages", lines)
			}
		})
	}
}
