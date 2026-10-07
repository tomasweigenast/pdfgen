package fonts_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	textfonts "github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

func loadFace(t testing.TB, path string) *textfonts.Face {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	face, err := textfonts.NewFace(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	return face
}

func testFace(t testing.TB) *textfonts.Face {
	t.Helper()
	return loadFace(t, testutil.ResourcePath(t, "testfont.otf"))
}

// systemTrueType returns a TrueType face from the system, or skips.
func systemTrueType(t testing.TB) *textfonts.Face {
	t.Helper()
	for _, p := range []string{
		"/System/Library/Fonts/Supplemental/Andale Mono.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"C:\\Windows\\Fonts\\arial.ttf",
	} {
		if _, err := os.Stat(p); err == nil {
			return loadFace(t, p)
		}
	}
	t.Skip("no TrueType font found on this system")
	return nil
}

// drawText shows each rune of text with face at size px, advancing by the
// advance of the glyph.
func drawText(s *stream.Stream, face *textfonts.Face, size, x, y float64, text string) {
	font := s.AddFont(face)
	s.BeginText()
	s.SetTextMatrix(1, 0, 0, -1, x, y)
	s.SetFont(font, size)
	var els []fonts.TJElement
	for _, r := range text {
		gid, _ := face.GoText.NominalGlyph(r)
		cid := font.CID(uint16(gid))
		font.SetText(cid, string(r))
		font.LogicalWidth(cid, size)
		els = append(els, fonts.TJElement{CID: cid})
	}
	s.ShowGlyphs(els)
	s.EndText()
}

// buildPDF writes a one-page document of width×height CSS pixels, at
// testutil.PixelZoom, with the fonts that draw registered.
func buildPDF(t testing.TB, width, height float64, opts fonts.BuildOptions, registry *fonts.Registry, draw func(s *stream.Stream)) []byte {
	t.Helper()
	scale := testutil.PixelZoom * 0.75

	pdf := writer.New()
	colorSpaces := pdf.Add(writer.NewDict())
	resRef := pdf.Reserve()
	res := stream.NewResources(colorSpaces)
	res.Fonts = registry

	s := stream.New(res, stream.Options{})
	s.Transform(1, 0, 0, -1, 0, height*scale)
	s.Transform(scale, 0, 0, scale, 0, 0)
	draw(s)

	dict, err := res.Fonts.Build(pdf, opts)
	if err != nil {
		t.Fatal(err)
	}
	res.Font = pdf.Add(dict)
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
