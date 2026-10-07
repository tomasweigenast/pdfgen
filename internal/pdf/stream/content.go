package stream

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"strconv"

	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// Content accumulates the operators of a content stream. Items are the
// operator lines; they are joined by a newline when the stream is written.
// Each method appends exactly one operator, with operands formatted like
// [writer.Real] values (ISO 32000-1 §7.3.3).
type Content struct {
	buf    []byte
	starts []int
	// Compress selects Flate for inline image data.
	Compress bool
}

// Bytes returns the stream data.
func (c *Content) Bytes() []byte { return c.buf }

// Len returns the number of items in the stream.
func (c *Content) Len() int { return len(c.starts) }

// lastIs reports whether the most recent item is exactly op.
func (c *Content) lastIs(op string) bool {
	n := len(c.starts)
	return n > 0 && string(c.buf[c.starts[n-1]:]) == op
}

// dropLast removes the most recent item.
func (c *Content) dropLast() {
	n := len(c.starts)
	if n == 0 {
		return
	}
	start := c.starts[n-1]
	if n > 1 {
		start-- // the separating newline
	}
	c.buf = c.buf[:start]
	c.starts = c.starts[:n-1]
}

// begin starts a new item.
func (c *Content) begin() {
	if len(c.starts) > 0 {
		c.buf = append(c.buf, '\n')
	}
	c.starts = append(c.starts, len(c.buf))
}

func (c *Content) num(f float64) { c.buf = writer.AppendReal(c.buf, f); c.buf = append(c.buf, ' ') }

func (c *Content) name(n string) {
	c.buf = writer.Append(c.buf, writer.Name(n))
	c.buf = append(c.buf, ' ')
}

// op appends "operands... op".
func (c *Content) op(op string, operands ...float64) {
	c.begin()
	for _, f := range operands {
		c.num(f)
	}
	c.buf = append(c.buf, op...)
}

// PushState saves the graphics state (q).
func (c *Content) PushState() { c.op("q") }

// PopState restores the graphics state (Q).
func (c *Content) PopState() { c.op("Q") }

// SetMatrix concatenates a matrix to the current transformation matrix (cm).
func (c *Content) SetMatrix(a, b, cc, d, e, f float64) { c.op("cm", a, b, cc, d, e, f) }

// SetLineWidth sets the line width (w).
func (c *Content) SetLineWidth(w float64) { c.op("w", w) }

// SetLineCap sets the line cap style (J).
func (c *Content) SetLineCap(style int) { c.op("J", float64(style)) }

// SetLineJoin sets the line join style (j).
func (c *Content) SetLineJoin(style int) { c.op("j", float64(style)) }

// SetMiterLimit sets the miter limit (M).
func (c *Content) SetMiterLimit(limit float64) { c.op("M", limit) }

// SetDash sets the dash pattern (d).
func (c *Content) SetDash(array []float64, phase float64) {
	c.begin()
	c.buf = append(c.buf, '[')
	for i, f := range array {
		if i > 0 {
			c.buf = append(c.buf, ' ')
		}
		c.buf = writer.AppendReal(c.buf, f)
	}
	c.buf = append(c.buf, "] "...)
	c.num(phase)
	c.buf = append(c.buf, 'd')
}

// SetRenderingIntent sets the color rendering intent (ri).
func (c *Content) SetRenderingIntent(intent string) {
	c.begin()
	c.name(intent)
	c.buf = append(c.buf, "ri"...)
}

// SetFlatness sets the flatness tolerance (i).
func (c *Content) SetFlatness(f float64) { c.op("i", f) }

// SetGraphicsState applies the named ExtGState resource (gs).
func (c *Content) SetGraphicsState(name string) {
	c.begin()
	c.name(name)
	c.buf = append(c.buf, "gs"...)
}

// MoveTo begins a new subpath (m).
func (c *Content) MoveTo(x, y float64) { c.op("m", x, y) }

// LineTo appends a straight segment (l).
func (c *Content) LineTo(x, y float64) { c.op("l", x, y) }

// CurveTo appends a cubic Bézier curve with two control points (c).
func (c *Content) CurveTo(x1, y1, x2, y2, x3, y3 float64) { c.op("c", x1, y1, x2, y2, x3, y3) }

// CurveStartTo appends a cubic Bézier curve whose first control point is the
// current point (v).
func (c *Content) CurveStartTo(x2, y2, x3, y3 float64) { c.op("v", x2, y2, x3, y3) }

// CurveEndTo appends a cubic Bézier curve whose second control point is its
// end point (y).
func (c *Content) CurveEndTo(x1, y1, x3, y3 float64) { c.op("y", x1, y1, x3, y3) }

// ClosePath closes the current subpath (h).
func (c *Content) ClosePath() { c.op("h") }

// Rectangle appends a rectangle as a complete subpath (re).
func (c *Content) Rectangle(x, y, width, height float64) { c.op("re", x, y, width, height) }

func (c *Content) evenOddOp(op string, evenOdd bool) {
	if evenOdd {
		op += "*"
	}
	c.op(op)
}

// Stroke strokes the path (S).
func (c *Content) Stroke() { c.op("S") }

// StrokeAndClose closes and strokes the path (s).
func (c *Content) StrokeAndClose() { c.op("s") }

// Fill fills the path with the nonzero winding rule, or the even-odd rule
// when evenOdd is set (f, f*).
func (c *Content) Fill(evenOdd bool) { c.evenOddOp("f", evenOdd) }

// FillAndStroke fills and strokes the path (B, B*).
func (c *Content) FillAndStroke(evenOdd bool) { c.evenOddOp("B", evenOdd) }

// FillStrokeAndClose closes, fills and strokes the path (b, b*).
func (c *Content) FillStrokeAndClose(evenOdd bool) { c.evenOddOp("b", evenOdd) }

// EndPath ends the path without painting it (n).
func (c *Content) EndPath() { c.op("n") }

// Clip intersects the clipping path with the current path (W, W*).
func (c *Content) Clip(evenOdd bool) { c.evenOddOp("W", evenOdd) }

// SetColorSpace selects the named color space for non-stroking operations,
// or for stroking ones when stroke is set (cs, CS).
func (c *Content) SetColorSpace(space string, stroke bool) {
	c.begin()
	c.name(space)
	c.buf = append(c.buf, strokeOp("cs", "CS", stroke)...)
}

func strokeOp(fill, stroke string, isStroke bool) string {
	if isStroke {
		return stroke
	}
	return fill
}

// SetColorRGB sets a DeviceRGB color (rg, RG).
func (c *Content) SetColorRGB(r, g, b float64, stroke bool) {
	c.op(strokeOp("rg", "RG", stroke), r, g, b)
}

// SetColorGray sets a DeviceGray color (g, G).
func (c *Content) SetColorGray(gray float64, stroke bool) {
	c.op(strokeOp("g", "G", stroke), gray)
}

// SetColorCMYK sets a DeviceCMYK color (k, K).
func (c *Content) SetColorCMYK(cyan, magenta, yellow, black float64, stroke bool) {
	c.op(strokeOp("k", "K", stroke), cyan, magenta, yellow, black)
}

// SetColorComponents sets a color in the current color space (sc, SC).
func (c *Content) SetColorComponents(stroke bool, operands ...float64) {
	c.op(strokeOp("sc", "SC", stroke), operands...)
}

// SetColorSpecial sets a color in the current color space, optionally
// followed by a pattern name (scn, SCN).
func (c *Content) SetColorSpecial(name string, stroke bool, operands ...float64) {
	c.begin()
	for _, f := range operands {
		c.num(f)
	}
	if name != "" {
		c.name(name)
	}
	c.buf = append(c.buf, strokeOp("scn", "SCN", stroke)...)
}

// PaintShading paints the named shading over the clipping region (sh).
func (c *Content) PaintShading(name string) {
	c.begin()
	c.name(name)
	c.buf = append(c.buf, "sh"...)
}

// DrawXObject paints the named XObject (Do).
func (c *Content) DrawXObject(name string) {
	c.begin()
	c.name(name)
	c.buf = append(c.buf, "Do"...)
}

// BeginText begins a text object (BT).
func (c *Content) BeginText() { c.op("BT") }

// EndText ends a text object (ET).
func (c *Content) EndText() { c.op("ET") }

// SetCharSpacing sets the character spacing (Tc).
func (c *Content) SetCharSpacing(v float64) { c.op("Tc", v) }

// SetWordSpacing sets the word spacing (Tw).
func (c *Content) SetWordSpacing(v float64) { c.op("Tw", v) }

// SetHorizontalScaling sets the horizontal scaling in percent (Tz).
func (c *Content) SetHorizontalScaling(v float64) { c.op("Tz", v) }

// SetLeading sets the text leading (TL).
func (c *Content) SetLeading(v float64) { c.op("TL", v) }

// SetFontSize selects the named font resource and size (Tf).
func (c *Content) SetFontSize(font string, size float64) {
	c.begin()
	c.name(font)
	c.num(size)
	c.buf = append(c.buf, "Tf"...)
}

// SetTextRendering sets the text rendering mode (Tr).
func (c *Content) SetTextRendering(mode int) { c.op("Tr", float64(mode)) }

// SetTextRise sets the text rise (Ts).
func (c *Content) SetTextRise(v float64) { c.op("Ts", v) }

// MoveTextTo moves to the start of the next line, offset from the start of
// the current one (Td).
func (c *Content) MoveTextTo(x, y float64) { c.op("Td", x, y) }

// MoveTextSetLeading moves like [Content.MoveTextTo] and sets the leading to
// -y (TD).
func (c *Content) MoveTextSetLeading(x, y float64) { c.op("TD", x, y) }

// SetTextMatrix sets the text and text line matrices (Tm).
func (c *Content) SetTextMatrix(a, b, cc, d, e, f float64) { c.op("Tm", a, b, cc, d, e, f) }

// NextLine moves to the start of the next text line (T*).
func (c *Content) NextLine() { c.op("T*") }

// ShowText shows glyphs with individual positioning (TJ). body is the
// content of the array, already serialized.
func (c *Content) ShowText(body []byte) {
	c.begin()
	c.buf = append(c.buf, '[')
	c.buf = append(c.buf, body...)
	c.buf = append(c.buf, "] TJ"...)
}

// ShowTextString shows a single string (Tj).
func (c *Content) ShowTextString(text string) {
	c.begin()
	c.buf = writer.Append(c.buf, writer.Text(text))
	c.buf = append(c.buf, " Tj"...)
}

// ShowTextNextLine moves to the next line and shows a string (').
func (c *Content) ShowTextNextLine(text string) {
	c.begin()
	c.buf = writer.Append(c.buf, writer.Text(text))
	c.buf = append(c.buf, " '"...)
}

// ShowTextSpaced sets the word and character spacing, moves to the next line
// and shows a string (").
func (c *Content) ShowTextSpaced(wordSpacing, charSpacing float64, text string) {
	c.begin()
	c.num(wordSpacing)
	c.num(charSpacing)
	c.buf = writer.Append(c.buf, writer.Text(text))
	c.buf = append(c.buf, " \""...)
}

// BeginMarkedContent opens a marked-content sequence (BMC, or BDC when a
// property list is given).
func (c *Content) BeginMarkedContent(tag string, properties *writer.Dict) {
	c.begin()
	c.buf = append(c.buf, '/')
	c.buf = append(c.buf, tag...)
	if properties == nil {
		c.begin()
		c.buf = append(c.buf, "BMC"...)
		return
	}
	c.begin()
	c.buf = writer.Append(c.buf, properties)
	c.begin()
	c.buf = append(c.buf, "BDC"...)
}

// EndMarkedContent closes a marked-content sequence (EMC).
func (c *Content) EndMarkedContent() { c.op("EMC") }

// MarkedPoint designates a marked-content point (MP, or DP with a property
// list).
func (c *Content) MarkedPoint(tag string, properties *writer.Dict) {
	c.begin()
	c.name(tag)
	if properties == nil {
		c.buf = append(c.buf, "MP"...)
		return
	}
	c.buf = writer.Append(c.buf, properties)
	c.buf = append(c.buf, " DP"...)
}

// InlineImage embeds an image in the stream (BI, ID, EI). colorSpace is the
// suffix of a device color space: "RGB", "Gray" or "CMYK". The data is
// ASCII85-encoded, after Flate compression when [Content.Compress] is set.
func (c *Content) InlineImage(width, height int, colorSpace string, bpc int, raw []byte) {
	data := raw
	if c.Compress {
		var z bytes.Buffer
		w := zlib.NewWriter(&z)
		_, _ = w.Write(raw)
		_ = w.Close()
		data = z.Bytes()
	}
	enc := make([]byte, ascii85.MaxEncodedLen(len(data)))
	enc = append(enc[:ascii85.Encode(enc, data)], "~>"...)

	filter := "/A85"
	if c.Compress {
		filter = "[/A85 /Fl]"
	}
	c.begin()
	c.buf = append(c.buf, "BI /W "...)
	c.buf = strconv.AppendInt(c.buf, int64(width), 10)
	c.buf = append(c.buf, " /H "...)
	c.buf = strconv.AppendInt(c.buf, int64(height), 10)
	c.buf = append(c.buf, " /BPC "...)
	c.buf = strconv.AppendInt(c.buf, int64(bpc), 10)
	c.buf = append(c.buf, " /CS /Device"...)
	c.buf = append(c.buf, colorSpace...)
	c.buf = append(c.buf, " /F "...)
	c.buf = append(c.buf, filter...)
	c.buf = append(c.buf, " /L "...)
	c.buf = strconv.AppendInt(c.buf, int64(len(enc)), 10)
	c.buf = append(c.buf, " ID "...)
	c.buf = append(c.buf, enc...)
	c.buf = append(c.buf, " EI"...)
}
