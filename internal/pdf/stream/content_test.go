package stream

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"io"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

func TestContentOperators(t *testing.T) {
	tests := []struct {
		name string
		draw func(c *Content)
		want string
	}{
		{"state", func(c *Content) { c.PushState(); c.PopState() }, "q\nQ"},
		{"matrix", func(c *Content) { c.SetMatrix(1, 0, 0.5, -1, 12.345678901, 1e-7) }, "1 0 0.5 -1 12.345679 0. cm"},
		{"integral floats", func(c *Content) { c.SetMatrix(2.0, 0, 0.0, 1.0, 100000.0, 3.25) }, "2 0 0 1 100000 3.25 cm"},
		{"path", func(c *Content) {
			c.MoveTo(1, 2)
			c.LineTo(3.5, 4)
			c.CurveTo(1, 2, 3, 4, 5, 6)
			c.CurveStartTo(1, 2, 3, 4)
			c.CurveEndTo(1, 2, 3, 4)
			c.ClosePath()
			c.Rectangle(0.1, 0.2, 0.30000000000000004, 1e-3)
		}, "1 2 m\n3.5 4 l\n1 2 3 4 5 6 c\n1 2 3 4 v\n1 2 3 4 y\nh\n0.1 0.2 0.3 0.001 re"},
		{"painting", func(c *Content) {
			c.Stroke()
			c.StrokeAndClose()
			c.Fill(false)
			c.Fill(true)
			c.FillAndStroke(false)
			c.FillAndStroke(true)
			c.FillStrokeAndClose(false)
			c.FillStrokeAndClose(true)
			c.EndPath()
		}, "S\ns\nf\nf*\nB\nB*\nb\nb*\nn"},
		{"clip", func(c *Content) { c.Clip(false); c.Clip(true) }, "W\nW*"},
		{"line style", func(c *Content) {
			c.SetLineWidth(1.5)
			c.SetLineCap(1)
			c.SetLineJoin(2)
			c.SetMiterLimit(10)
			c.SetDash([]float64{3, 2.5}, 0)
			c.SetDash(nil, 0)
		}, "1.5 w\n1 J\n2 j\n10 M\n[3 2.5] 0 d\n[] 0 d"},
		{"graphics state", func(c *Content) { c.SetGraphicsState("a0.5") }, "/a0.5 gs"},
		{"colors", func(c *Content) {
			c.SetColorRGB(1, 0.5, 0, false)
			c.SetColorRGB(0, 0.25, 1, true)
			c.SetColorSpace("DeviceCMYK", false)
			c.SetColorSpace("lab-d50", true)
			c.SetColorSpecial("", false, 0.1, 0.2, 0.3, 0.4)
			c.SetColorSpecial("", true, 50.5, -20, 30)
			c.SetColorSpecial("p0", false)
			c.SetColorSpecial("p1", true, 1, 0, 0)
		}, "1 0.5 0 rg\n0 0.25 1 RG\n/DeviceCMYK cs\n/lab-d50 CS\n0.1 0.2 0.3 0.4 scn\n50.5 -20 30 SCN\n/p0 scn\n1 0 0 /p1 SCN"},
		{"text", func(c *Content) {
			c.BeginText()
			c.SetFontSize("F1", 12.5)
			c.SetTextRendering(1)
			c.SetTextRise(2)
			c.MoveTextTo(10, -20.5)
			c.SetTextMatrix(1, 0, 0, -1, 5, 6)
			c.ShowText([]byte("<0001>-20.5<0002>"))
			c.ShowTextString(`a(b)c\d`)
			c.ShowTextString("café ✓")
			c.EndText()
		}, "BT\n/F1 12.5 Tf\n1 Tr\n2 Ts\n10 -20.5 Td\n1 0 0 -1 5 6 Tm\n[<0001>-20.5<0002>] TJ\n(a\\(b\\)c\\\\d) Tj\n<feff00630061006600e900202713> Tj\nET"},
		{"external objects", func(c *Content) { c.DrawXObject("x0"); c.PaintShading("s1") }, "/x0 Do\n/s1 sh"},
		{"marked content", func(c *Content) {
			c.BeginMarkedContent("P", nil)
			c.EndMarkedContent()
			c.BeginMarkedContent("Span", writer.NewDict("MCID", 3))
			c.EndMarkedContent()
			c.BeginMarkedContent("Artifact", nil)
			c.EndMarkedContent()
		}, "/P\nBMC\nEMC\n/Span\n<</MCID 3>>\nBDC\nEMC\n/Artifact\nBMC\nEMC"},
		{"inline image", func(c *Content) {
			c.InlineImage(3, 2, "RGB", 8, rangeBytes(18))
			c.InlineImage(4, 1, "Gray", 8, make([]byte, 4))
		}, "BI /W 3 /H 2 /BPC 8 /CS /DeviceRGB /F /A85 /L 25 ID !!*-'\"9eu7#RLhG$k3[W&.n~> EI\n" +
			"BI /W 4 /H 1 /BPC 8 /CS /DeviceGray /F /A85 /L 3 ID z~> EI"},
		{"text state", func(c *Content) {
			c.SetCharSpacing(0.5)
			c.SetWordSpacing(-1)
			c.SetHorizontalScaling(80)
			c.SetLeading(14)
			c.MoveTextSetLeading(0, -14)
			c.NextLine()
			c.ShowTextNextLine("a")
			c.ShowTextSpaced(1, 0.25, "b")
		}, "0.5 Tc\n-1 Tw\n80 Tz\n14 TL\n0 -14 TD\nT*\n(a) '\n1 0.25 (b) \""},
		{"device colors and hints", func(c *Content) {
			c.SetColorGray(0.5, false)
			c.SetColorGray(1, true)
			c.SetColorCMYK(0, 0.5, 1, 0, false)
			c.SetColorCMYK(1, 0, 0, 0, true)
			c.SetColorComponents(false, 0.2)
			c.SetColorComponents(true, 0.2, 0.4)
			c.SetRenderingIntent("Perceptual")
			c.SetFlatness(2)
		}, "0.5 g\n1 G\n0 0.5 1 0 k\n1 0 0 0 K\n0.2 sc\n0.2 0.4 SC\n/Perceptual ri\n2 i"},
		{"marked points", func(c *Content) {
			c.MarkedPoint("Pt", nil)
			c.MarkedPoint("Pt", writer.NewDict("A", 1))
		}, "/Pt MP\n/Pt <</A 1>> DP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Content
			tt.draw(&c)
			if got := string(c.Bytes()); got != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func rangeBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func TestInlineImageFlate(t *testing.T) {
	c := Content{Compress: true}
	raw := rangeBytes(18)
	c.InlineImage(3, 2, "RGB", 8, raw)
	got := string(c.Bytes())
	if !strings.HasPrefix(got, "BI /W 3 /H 2 /BPC 8 /CS /DeviceRGB /F [/A85 /Fl] /L ") || !strings.HasSuffix(got, "~> EI") {
		t.Fatalf("unexpected inline image: %q", got)
	}

	// The compressed bytes depend on the deflate implementation, so check
	// that the data decodes back to the input.
	_, data, _ := strings.Cut(got, " ID ")
	data = strings.TrimSuffix(data, "~> EI")
	dec := decodeASCII85(t, data)
	zr, err := zlib.NewReader(bytes.NewReader(dec))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, raw) {
		t.Errorf("round trip = %v, want %v", out, raw)
	}
}

func decodeASCII85(t *testing.T, s string) []byte {
	t.Helper()
	dst := make([]byte, len(s))
	n, _, err := ascii85.Decode(dst, []byte(s), true)
	if err != nil {
		t.Fatal(err)
	}
	return dst[:n]
}
