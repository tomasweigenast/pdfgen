package stream_test

import (
	"bytes"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// renderPDF builds a one-page document of width×height CSS pixels, scaled so
// that a CSS pixel is one raster pixel at testutil.PixelZoom, lets draw fill
// the page stream and returns the serialized file.
func renderPDF(t *testing.T, width, height float64, draw func(s *stream.Stream)) []byte {
	t.Helper()
	scale := testutil.PixelZoom * 0.75

	pdf := writer.New()
	colorSpaces := pdf.Add(writer.NewDict())
	resRef := pdf.Reserve()
	res := stream.NewResources(colorSpaces)

	s := stream.New(res, stream.Options{})
	s.Transform(1, 0, 0, -1, 0, height*scale)
	s.Transform(scale, 0, 0, scale, 0, 0)
	draw(s)

	contents := pdf.Add(s.Object())
	if err := res.Embed(pdf, resRef); err != nil {
		t.Fatal(err)
	}
	pdf.AddPage(writer.NewDict(
		"Type", writer.Name("Page"),
		"MediaBox", writer.Array{writer.Int(0), writer.Int(0), writer.Real(width * scale), writer.Real(height * scale)},
		"Contents", contents,
		"Resources", resRef,
	))

	var out bytes.Buffer
	if err := pdf.Write(&out, writer.WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func rgb(r, g, b float64) syntax.Color {
	return syntax.NewColor(syntax.SRGB, []float64{r, g, b}, 1)
}

func TestRenderShapesTransformsAndClipping(t *testing.T) {
	pdf := renderPDF(t, 12, 10, func(s *stream.Stream) {
		s.SetColor(rgb(1, 0, 0), false)
		s.Rectangle(0, 0, 4, 4)
		s.Fill(false)

		s.PushState()
		s.Transform(2, 0, 0, 2, 6, 0)
		s.SetColor(rgb(0, 0, 1), false)
		s.Rectangle(0, 0, 2, 2)
		s.Fill(false)
		s.PopState()

		s.PushState()
		s.Rectangle(0, 5, 6, 2)
		s.Clip(false)
		s.EndPath()
		s.SetColor(rgb(0, 1, 0), false)
		s.Rectangle(0, 4, 12, 6)
		s.Fill(false)
		s.PopState()

		s.SetColor(rgb(1, 1, 0), false)
		s.Rectangle(7, 5, 5, 5)
		s.Rectangle(8, 6, 3, 3)
		s.Fill(true)

		s.SetColor(rgb(0, 0, 0), true)
		s.SetLineWidth(1)
		s.MoveTo(0, 9.5)
		s.LineTo(6, 9.5)
		s.Stroke()
	})
	testutil.AssertPixelsPDF(t, `
		RRRR__BBBB__
		RRRR__BBBB__
		RRRR__BBBB__
		RRRR__BBBB__
		____________
		GGGGGG_YYYYY
		GGGGGG_Y___Y
		_______Y___Y
		_______Y___Y
		KKKKKK_YYYYY
	`, pdf)
}

func TestRenderTransparencyGroupBlendsAsOneObject(t *testing.T) {
	pdf := renderPDF(t, 8, 2, func(s *stream.Stream) {
		s.SetColor(rgb(0, 0, 1), false)
		s.Rectangle(4, 0, 4, 2)
		s.Fill(false)

		group := s.AddGroup(0, 0, 6, 2)
		group.SetColor(rgb(1, 0, 0), false)
		group.Rectangle(0, 0, 4, 2)
		group.Rectangle(2, 0, 4, 2)
		group.Fill(false)

		s.PushState()
		s.SetAlpha(0.5, false, nil)
		s.DrawXObject(group.ID())
		s.PopState()
	})
	testutil.AssertPixelsPDF(t, `
		ssssuuBB
		ssssuuBB
	`, pdf)
}

func TestRenderSoftMask(t *testing.T) {
	pdf := renderPDF(t, 8, 2, func(s *stream.Stream) {
		mask := s.SetAlphaState(0, 0, 8, 2, "luminosity")
		mask.SetColor(rgb(1, 1, 1), false)
		mask.Rectangle(0, 0, 4, 2)
		mask.Fill(false)

		s.SetColor(rgb(1, 0, 0), false)
		s.Rectangle(0, 0, 8, 2)
		s.Fill(false)
	})
	testutil.AssertPixelsPDF(t, `
		RRRR____
		RRRR____
	`, pdf)
}

func hardStop(from, to []float64) stream.Function {
	domain := []float64{0, 1}
	return stream.StitchingFunction(domain, []float64{0, 1, 0, 1}, []float64{0.5}, []stream.Function{
		stream.InterpolationFunction(domain, from, from, 1),
		stream.InterpolationFunction(domain, to, to, 1),
	})
}

func TestRenderAxialShading(t *testing.T) {
	pdf := renderPDF(t, 8, 2, func(s *stream.Stream) {
		red, blue := []float64{1, 0, 0}, []float64{0, 0, 1}
		name := s.AddShading(2, []float64{0, 1}, []float64{0, 0, 8, 0}, true, hardStop(red, blue), "")
		s.Rectangle(0, 0, 8, 2)
		s.Clip(false)
		s.EndPath()
		s.PaintShading(name)
	})
	testutil.AssertPixelsPDF(t, `
		RRRRBBBB
		RRRRBBBB
	`, pdf)
}

func TestRenderRadialShading(t *testing.T) {
	pdf := renderPDF(t, 8, 8, func(s *stream.Stream) {
		red, blue := []float64{1, 0, 0}, []float64{0, 0, 1}
		name := s.AddShading(3, []float64{0, 1}, []float64{4, 4, 0, 4, 4, 4}, true, hardStop(red, blue), "")
		s.PaintShading(name)
	})
	testutil.AssertPixelsPDF(t, `
		BBBBBBBB
		BBBBBBBB
		BBBRRBBB
		BBRRRRBB
		BBRRRRBB
		BBBRRBBB
		BBBBBBBB
		BBBBBBBB
		`, pdf)
}

func TestRenderTilingPattern(t *testing.T) {
	// A pattern matrix maps to the default coordinate system of the page, not
	// to the current one.
	scale := testutil.PixelZoom * 0.75
	pdf := renderPDF(t, 8, 4, func(s *stream.Stream) {
		pattern := s.AddPattern(0, 0, 2, 2, 4, 4, matrix.New(scale, 0, 0, -scale, 0, 4*scale))
		pattern.SetColor(rgb(1, 0, 0), false)
		pattern.Rectangle(0, 0, 2, 2)
		pattern.Fill(false)

		s.SetColorSpace("Pattern", false)
		s.SetColorSpecial(pattern.ID(), false)
		s.Rectangle(0, 0, 8, 4)
		s.Fill(false)
	})
	testutil.AssertPixelsPDF(t, `
		RR__RR__
		RR__RR__
		________
		________
	`, pdf)
}
