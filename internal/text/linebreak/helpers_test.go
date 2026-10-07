package linebreak

import (
	"os"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// newTestContext returns a context whose only font is the test font,
// registered as "testfont". Its glyphs are one em wide, and its ascent and
// descent are 819 and 205 units of a 1024 units em.
func newTestContext(t testing.TB) *Context {
	t.Helper()
	fc := fonts.NewFontConfiguration(fonts.WithoutSystemFonts())
	rule := fonts.FontFaceRule{Family: "testfont", Src: []fonts.Source{{Value: "testfont.otf"}}}
	fetch := func(name string) ([]byte, error) {
		return os.ReadFile(testutil.ResourcePath(t, name))
	}
	if err := fc.AddFontFace(rule, fetch); err != nil {
		t.Fatalf("AddFontFace: %v", err)
	}
	return NewContext(fc)
}

// testStyle returns the initial style with the test font at 16 px.
func testStyle() *Style { return NewStyle(16, "testfont") }

func width(px float64) *float64 { return &px }
