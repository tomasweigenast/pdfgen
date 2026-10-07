package fonts

import (
	"os"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	textfonts "github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

func newTestFont(t *testing.T, full bool) *Font {
	t.Helper()
	data, err := os.ReadFile(testutil.ResourcePath(t, "testfont.otf"))
	if err != nil {
		t.Fatal(err)
	}
	face, err := textfonts.NewFace(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	return NewRegistry(full).Add(face)
}

func TestCIDsAreAssignedInOrderOfFirstUse(t *testing.T) {
	f := newTestFont(t, false)
	if got := f.CID(0); got != 0 {
		t.Errorf("CID(.notdef) = %d, want 0", got)
	}
	for _, tc := range []struct{ gid, want uint16 }{
		{33, 1}, {40, 2}, {33, 1}, {7, 3}, {40, 2}, {0, 0},
	} {
		if got := f.CID(tc.gid); got != tc.want {
			t.Errorf("CID(%d) = %d, want %d", tc.gid, got, tc.want)
		}
	}
	if want := []uint16{0, 33, 40, 7}; !equal(f.order, want) {
		t.Errorf("order = %v, want %v", f.order, want)
	}
}

func TestCIDsAreGlyphIndicesInFullFonts(t *testing.T) {
	f := newTestFont(t, true)
	if got := f.CID(40); got != 40 {
		t.Errorf("CID(40) = %d, want 40", got)
	}
}

func equal(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMissingGlyphsGetStandInCodes(t *testing.T) {
	f := newTestFont(t, false)
	f.CID(33)
	a := f.MissingCID('☃', 500)
	b := f.MissingCID('✈', 600)
	if a != 2 || b != 3 {
		t.Errorf("codes = %d, %d, want 2, 3", a, b)
	}
	if again := f.MissingCID('☃', 500); again != a {
		t.Errorf("the code of a character changed from %d to %d", a, again)
	}
	if w, _ := f.Width(0); w != 500 {
		t.Errorf("the width of .notdef = %v, want the first stand-in width 500", w)
	}
	if want := []uint16{0, 33, 0, 0}; !equal(f.order, want) {
		t.Errorf("order = %v, want %v", f.order, want)
	}
}

func TestMissingGlyphsOfWholeFontsAreBeyondItsGlyphs(t *testing.T) {
	f := newTestFont(t, true)
	a := f.MissingCID('☃', 500)
	b := f.MissingCID('✈', 500)
	if int(a) != f.glyphCount || int(b) != f.glyphCount+1 {
		t.Errorf("codes = %d, %d, want %d, %d", a, b, f.glyphCount, f.glyphCount+1)
	}
}

func TestLogicalWidthIsRoundedAndRecorded(t *testing.T) {
	f := newTestFont(t, false)
	cid := f.CID(33)
	// Every glyph of the test font is one em wide.
	if got := f.LogicalWidth(cid, 16); got != 1000 {
		t.Errorf("LogicalWidth = %v, want 1000", got)
	}
	f.SetWidth(cid, 999)
	if got := f.LogicalWidth(cid, 16); got != 999 {
		t.Errorf("LogicalWidth after SetWidth = %v, want 999", got)
	}
}

func TestSetTextKeepsTheFirstText(t *testing.T) {
	f := newTestFont(t, false)
	f.SetText(1, "a")
	f.SetText(1, "b")
	if f.text[1] != "a" {
		t.Errorf("text = %q, want a", f.text[1])
	}
}

func TestFontNaming(t *testing.T) {
	f := newTestFont(t, false)
	if len(f.Hash) != 6 || strings.Trim(f.Hash, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		t.Errorf("Hash = %q, want six capital letters", f.Hash)
	}
	if want := f.Hash + "+" + f.Family; f.BaseFont() != want {
		t.Errorf("BaseFont = %q, want %q", f.BaseFont(), want)
	}
	if f.Flags&4 == 0 {
		t.Errorf("Flags = %d, want the symbolic bit", f.Flags)
	}
	if again := newTestFont(t, false); again.Hash != f.Hash {
		t.Errorf("Hash changed between runs: %q, %q", f.Hash, again.Hash)
	}
}

func TestWidthsArrayGroupsConsecutiveCodes(t *testing.T) {
	got := string(writer.Append(nil, widthsArray(map[uint16]float64{
		0: 500, 1: 600, 2: 600, 5: 250, 7: 300, 8: 301,
	})))
	want := "[0 [500 600 600] 5 [250] 7 [300 301]]"
	if got != want {
		t.Errorf("W = %s, want %s", got, want)
	}
	if got := string(writer.Append(nil, widthsArray(nil))); got != "[]" {
		t.Errorf("empty W = %s, want []", got)
	}
}

func TestCIDSetMarksUsedCodes(t *testing.T) {
	got := cidSet(map[uint16]float64{0: 1, 1: 1, 9: 1})
	if want := "\xc0\x40"; string(got) != want {
		t.Errorf("CIDSet = %x, want %x", got, want)
	}
}

func TestToUnicodeMapsClustersToSeveralCodePoints(t *testing.T) {
	got := string(toUnicodeCMap(map[uint16]string{
		1: "a",
		2: "fi",
		3: "😀",
	}))
	for _, want := range []string{
		"/CIDSystemInfo\n<< /Registry (Adobe)\n/Ordering (UCS)\n/Supplement 0\n>> def",
		"/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <ffff>\nendcodespacerange",
		"3 beginbfchar\n<0001> <0061>\n<0002> <00660069>\n<0003> <d83dde00>\nendbfchar",
		"endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ToUnicode lacks %q:\n%s", want, got)
		}
	}
}

func TestToUnicodeSplitsEntriesInBatchesOfAHundred(t *testing.T) {
	text := map[uint16]string{}
	for i := 1; i <= 250; i++ {
		text[uint16(i)] = "x"
	}
	got := string(toUnicodeCMap(text))
	for _, want := range []string{"100 beginbfchar", "50 beginbfchar"} {
		if !strings.Contains(got, want) {
			t.Errorf("ToUnicode lacks %q", want)
		}
	}
	if n := strings.Count(got, "endbfchar"); n != 3 {
		t.Errorf("%d bfchar sections, want 3", n)
	}
}

func TestNotdefEncodingSendsStandInsToNotdef(t *testing.T) {
	f := newTestFont(t, true)
	cid := f.CID(33)
	f.SetText(cid, "A")
	stand := f.MissingCID('☃', 500)
	f.SetText(stand, "☃")
	got := string(f.notdefCMap())
	for _, want := range []string{"2 begincidchar", "<0021> 33", "<" + hex4(stand) + "> 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("encoding lacks %q:\n%s", want, got)
		}
	}
}

func hex4(v uint16) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>12], digits[v>>8&15], digits[v>>4&15], digits[v&15]})
}

func TestTJSerialization(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []TJElement
		want string
	}{
		{"empty", nil, ""},
		{"glyphs", []TJElement{{CID: 1}, {CID: 0xabcd}}, "<0001abcd>"},
		{"adjustment between", []TJElement{{CID: 1, Adjust: 12}, {CID: 2}}, "<0001>12<0002>"},
		{"adjustment after the last glyph", []TJElement{{CID: 1}, {CID: 2, Adjust: -3}}, "<00010002>-3"},
		{"truncated toward zero", []TJElement{{CID: 1, Adjust: 2.9}, {CID: 2, Adjust: -2.9}, {CID: 3}}, "<0001>2<0002>-2<0003>"},
		{"small adjustment keeps a zero", []TJElement{{CID: 1, Adjust: 0.5}, {CID: 2}}, "<0001>0<0002>"},
		{"offset before the glyph", []TJElement{{CID: 1}, {CID: 2, Before: -0.25}}, "<0001>-0.25<0002>"},
		{"offset first", []TJElement{{CID: 1, Before: 1.5}}, "<>1.5<0001>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(TJ(tc.in)); got != tc.want {
				t.Errorf("TJ = %q, want %q", got, tc.want)
			}
		})
	}
}
