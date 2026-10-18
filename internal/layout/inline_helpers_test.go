package layout

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
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

// condensedTestFontURL returns a data URL of the test font with every
// advance width halved, so that a family can have a condensed face of its own.
func condensedTestFontURL(t testing.TB) string {
	t.Helper()
	font, err := os.ReadFile(testutil.ResourcePath(t, "testfont.otf"))
	if err != nil {
		t.Fatalf("read the test font: %v", err)
	}
	tables := make(map[string][2]int)
	for i := range int(binary.BigEndian.Uint16(font[4:])) {
		record := font[12+16*i:]
		tables[string(record[:4])] = [2]int{int(binary.BigEndian.Uint32(record[8:])), int(binary.BigEndian.Uint32(record[12:]))}
	}
	hhea, hmtx := tables["hhea"], tables["hmtx"]
	if hhea[1] < 36 || hmtx[1] == 0 {
		t.Fatalf("the test font has no horizontal metrics: %v", tables)
	}
	metrics := int(binary.BigEndian.Uint16(font[hhea[0]+34:]))
	for i := range metrics {
		advance := font[hmtx[0]+4*i:]
		binary.BigEndian.PutUint16(advance, binary.BigEndian.Uint16(advance)/2)
	}
	return "data:font/otf;base64," + base64.StdEncoding.EncodeToString(font)
}

// newLayoutText returns the text layout context of the fonts that the style
// of a box uses.
func newLayoutText(box boxes.Box) *linebreak.Context {
	return linebreak.NewContext(box.Base().Style.Context().FontConfig)
}
