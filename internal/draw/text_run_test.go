package draw

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
)

// testLine lays out text with the test font alone, so that no other font can
// provide a glyph.
func testLine(t *testing.T, text string) (*linebreak.Line, string) {
	t.Helper()
	fc := fonts.NewFontConfiguration(fonts.WithoutSystemFonts())
	rule := fonts.FontFaceRule{Family: "testfont", Src: []fonts.Source{{Value: "testfont.otf"}}}
	fetch := func(name string) ([]byte, error) { return os.ReadFile(testutil.ResourcePath(t, name)) }
	if err := fc.AddFontFace(rule, fetch); err != nil {
		t.Fatal(err)
	}
	layout := linebreak.NewLayout(linebreak.NewContext(fc), text, linebreak.NewStyle(2, "testfont"), nil, 0)
	return layout.FirstLine(), layout.Text
}

func TestShowRunLogsMissingGlyphs(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	line, text := testLine(t, "a'")
	s := stream.New(stream.NewResources(writer.Ref{Num: 1}), stream.Options{})
	s.BeginText()
	var shift float64
	for _, run := range line.Runs {
		showRun(s, run, text, &shift)
	}
	s.EndText()

	want := `.notdef glyph rendered for Unicode string unsupported by fonts: "'" (U+0027)`
	if !slices.Contains(logs.Warnings(), want) {
		t.Errorf("warnings = %q, want %q", logs.Warnings(), want)
	}
	if !strings.Contains(string(s.Bytes()), "TJ") {
		t.Errorf("no text shown: %q", s.Bytes())
	}
}

func TestShowRunMovesPastEmptyGlyphs(t *testing.T) {
	line, text := testLine(t, "a\tb")
	s := stream.New(stream.NewResources(writer.Ref{Num: 1}), stream.Options{})
	s.BeginText()
	var shift float64
	for _, run := range line.Runs {
		showRun(s, run, text, &shift)
	}
	s.EndText()
	// The tab is not drawn: the second glyph carries its advance.
	if got := string(s.Bytes()); strings.Count(got, "TJ") != 2 || !strings.Contains(got, ">-") {
		t.Errorf("content = %q, want two shown strings, the second one moved to the right", got)
	}
}
