package fonts

import (
	"os"
	"testing"

	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func readResource(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testutil.ResourcePath(t, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return data
}

// textWidth shapes text with the face at a size in px and returns the
// width in px. The features of the face come first, then style features.
func textWidth(t testing.TB, f *Face, size float64, text string, style ...Feature) float64 {
	t.Helper()
	runes := []rune(text)
	buf := harfbuzz.NewBuffer()
	buf.AddRunes(runes, 0, len(runes))
	buf.ClusterLevel = harfbuzz.MonotoneCharacters
	buf.Props.Script = language.Latin
	buf.Props.Language = language.NewLanguage("en")
	buf.Props.Direction = harfbuzz.LeftToRight
	features := append(append([]Feature(nil), f.Features...), style...)
	buf.Shape(f.HB(ToUnits(size)), HBFeatures(features))
	var width int32
	for _, p := range buf.Pos {
		width += p.XAdvance
	}
	return FromUnits(width)
}

// patchedFont returns a copy of the font data where the bytes at offset of
// the named table are replaced.
func patchedFont(t testing.TB, data []byte, table string, offset int, value []byte) []byte {
	t.Helper()
	out := append([]byte(nil), data...)
	n := int(out[4])<<8 | int(out[5])
	for i := 0; i < n; i++ {
		rec := out[12+16*i:]
		if string(rec[:4]) == table {
			start := int(rec[8])<<24 | int(rec[9])<<16 | int(rec[10])<<8 | int(rec[11])
			copy(out[start+offset:], value)
			return out
		}
	}
	t.Fatalf("table %q not found", table)
	return nil
}
