package cascade_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestStylesheetWarnings(t *testing.T) {
	tests := []struct {
		source   string
		messages []string
	}{
		{":lipsum { margin: 2cm", []string{"WARNING: Invalid or unsupported selector"}},
		{"::lipsum { margin: 2cm", []string{"WARNING: Invalid or unsupported selector"}},
		{"foo { margin-color: red", []string{"WARNING: Ignored", "unknown property"}},
		{"foo { margin-top: red", []string{"WARNING: Ignored", "invalid value"}},
		{`@import "relative-uri.css"`, []string{"ERROR: Relative URI reference"}},
		{`@import "invalid-protocol://absolute-URL"`, []string{"ERROR: Failed to load stylesheet"}},
		{"test", []string{"WARNING: Parse error"}},
		{"@test", []string{"WARNING: Unknown empty rule"}},
		{"@test {}", []string{"WARNING: Unknown rule"}},
		{`@import::before "test.css"`, []string{"WARNING: Invalid @import"}},
		{`@page { @top-left super { content: "a" } }`, []string{"WARNING: Invalid prelude"}},
		{`@page { @unknown { content: "a" } }`, []string{"WARNING: Unknown @page rule"}},
		{`@page { @footnote::before { content: "a" } }`, []string{"WARNING: Invalid prelude"}},
	}
	for _, tt := range tests {
		logs := logging.StartCapture()
		cascade.ParseStylesheet(tt.source, "", nil)
		logs.Close()
		lines := logs.Lines()
		if len(lines) != 1 {
			t.Errorf("%s: logs = %q, want one message", tt.source, lines)
			continue
		}
		for _, m := range tt.messages {
			if !strings.Contains(lines[0], m) {
				t.Errorf("%s: log %q does not contain %q", tt.source, lines[0], m)
			}
		}
	}
}

func TestLinkedStylesheetFailureIsReported(t *testing.T) {
	logs := logging.StartCapture()
	doc := csstest.Parse(t, "<link rel=stylesheet href=invalid-protocol://absolute>")
	csstest.StyleFor(t, doc, cascade.Options{})
	logs.Close()
	lines := logs.Lines()
	if len(lines) != 1 || !strings.Contains(lines[0], "ERROR: Failed to load stylesheet at") {
		t.Errorf("logs = %q", lines)
	}
}

func TestErrorRecovery(t *testing.T) {
	for _, source := range []string{
		"<style> html { color red; color: blue; color",
		`<html style="color; color: blue; color red">`,
	} {
		logs := logging.StartCapture()
		doc := csstest.Parse(t, source)
		sf := csstest.StyleFor(t, doc, cascade.Options{})
		color := rgba(sf.Get(doc.Root, "").Color())
		logs.Close()
		if color != [4]float64{0, 0, 1, 1} {
			t.Errorf("%s: color = %v, want blue", source, color)
		}
		if lines := logs.Lines(); len(lines) != 2 {
			t.Errorf("%s: logs = %q, want 2 messages", source, lines)
		}
	}
}
