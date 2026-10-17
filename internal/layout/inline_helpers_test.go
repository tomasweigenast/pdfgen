package layout

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
)

// kids returns the children of a box, failing the test unless there are n.
func kids(t testing.TB, box boxes.Box, n int) []boxes.Box {
	t.Helper()
	children := boxes.ChildrenOf(unwrapPlaceholder(box))
	if len(children) != n {
		t.Fatalf("%v has %d children, want %d: %v", box, len(children), n, children)
	}
	return children
}

// at returns the box reached from box by following child indexes.
func at(t testing.TB, box boxes.Box, path ...int) boxes.Box {
	t.Helper()
	for _, i := range path {
		children := boxes.ChildrenOf(unwrapPlaceholder(box))
		if i >= len(children) {
			t.Fatalf("%v has no child %d: %v", box, i, children)
		}
		box = children[i]
	}
	return box
}

// textAt returns the text of the text box reached by following child
// indexes.
func textAt(t testing.TB, box boxes.Box, path ...int) string {
	t.Helper()
	return textOf(t, at(t, box, path...))
}

// textOf returns the text of a text box, failing the test for another box.
func textOf(t testing.TB, box boxes.Box) string {
	t.Helper()
	text, ok := box.(*boxes.TextBox)
	if !ok {
		t.Fatalf("%v is not a text box", box)
	}
	return text.Text
}

// allText concatenates the text boxes under a box.
func allText(box boxes.Box) string {
	var out strings.Builder
	for d := range boxes.Descendants(box, false) {
		if text, ok := d.(*boxes.TextBox); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}

// sansFamilies are the fonts the tests that measure real text were written
// for.
const sansFamilies = "DejaVu Sans, sans"

// requireSansFonts skips the test when the font it was written for is not
// installed: the widths of its text, and so its line breaks, depend on it.
func requireSansFonts(t *testing.T) {
	t.Helper()
	fc := fonts.NewFontConfiguration()
	face := fc.Query(fonts.Description{Families: []string{"DejaVu Sans"}, Size: 16}, "", 'a')
	if face == nil || face.Family != "DejaVu Sans" {
		t.Skip("DejaVu Sans is not installed")
	}
}

// newLayoutText returns the text layout context of the fonts that the style
// of a box uses.
func newLayoutText(box boxes.Box) *linebreak.Context {
	return linebreak.NewContext(box.Base().Style.Context().FontConfig)
}
