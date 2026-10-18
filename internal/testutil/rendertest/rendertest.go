// Package rendertest renders HTML documents through the whole pipeline for
// tests: layout, drawing and PDF generation.
//
// Importing the package, usually with a blank import, makes
// [testutil.AssertPixels] and the other pixel assertions render HTML. It
// lives apart from testutil because it imports the packages that testutil is
// used to test.
package rendertest

import (
	"bytes"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func init() {
	testutil.HTMLToPixels = htmlToPixels
}

// PDF renders an HTML document as an uncompressed PDF file at
// [testutil.PixelZoom], where a CSS pixel is a pixel of the rasterized page.
func PDF(t testing.TB, html string, opts ...boxtest.Option) []byte {
	t.Helper()
	doc := boxtest.FakeHTML(t, html, opts...)
	pages := doc.RenderPages(t)

	var out bytes.Buffer
	err := pdf.Generate(pdf.NewDocument(pages, doc.HTML.Metadata()), &out, pdf.Options{
		Zoom:         testutil.PixelZoom,
		Uncompressed: true,
	})
	if err != nil {
		t.Fatalf("generate PDF: %v", err)
	}
	return out.Bytes()
}

// Pages lays out an HTML document and returns its page boxes.
func Pages(t testing.TB, html string) []*boxes.PageBox {
	t.Helper()
	return boxtest.RenderPages(t, html)
}

func htmlToPixels(t testing.TB, html string) (width, height int, pixels []testutil.Pixel) {
	t.Helper()
	testutil.RequireGhostscript(t)
	return testutil.ImagePixels(testutil.RasterizePDF(t, PDF(t, html), testutil.PixelZoom))
}
