package fonts

import (
	"strconv"

	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// TJElement is one glyph of a text-showing array, with the adjustments that
// surround it. Both are in thousandths of the font size, positive values
// moving the next glyph to the left (ISO 32000-1 section 9.4.3).
type TJElement struct {
	// CID is the code of the glyph.
	CID uint16
	// Before is an adjustment written as is before the glyph. It carries a
	// glyph's horizontal offset.
	Before float64
	// Adjust is an adjustment after the glyph that makes the glyph advance
	// as the layout wants. It is truncated toward zero, as whole units are
	// precise enough and keep the stream short.
	Adjust float64
}

// AppendTJ appends the content of a TJ array for elements to dst: one hex
// string per run of glyphs, with the adjustments between the strings. It
// appends nothing when there are no elements.
func AppendTJ(dst []byte, elements []TJElement) []byte {
	if len(elements) == 0 {
		return dst
	}
	const hex = "0123456789abcdef"
	dst = append(dst, '<')
	for _, e := range elements {
		if e.Before != 0 {
			dst = append(dst, '>')
			dst = writer.AppendReal(dst, e.Before)
			dst = append(dst, '<')
		}
		dst = append(dst, hex[e.CID>>12], hex[e.CID>>8&15], hex[e.CID>>4&15], hex[e.CID&15])
		if e.Adjust != 0 {
			dst = append(dst, '>')
			dst = strconv.AppendInt(dst, int64(e.Adjust), 10)
			dst = append(dst, '<')
		}
	}
	// An adjustment last leaves an empty string open, which is dropped.
	if dst[len(dst)-1] == '<' {
		return dst[:len(dst)-1]
	}
	return append(dst, '>')
}

// TJ returns the content of a TJ array for elements, ready for
// [stream.Content.ShowText].
func TJ(elements []TJElement) []byte { return AppendTJ(nil, elements) }
