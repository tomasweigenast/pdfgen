package pdf_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// renderAcid2Document lays out a test resource with the engine's own user agent
// stylesheet and a page size that keeps the images small and avoids rounding
// differences in the rasterizer.
func renderAcid2Document(t *testing.T, filename string) *pdf.Document {
	t.Helper()
	source, err := os.ReadFile(testutil.ResourcePath(t, filename))
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	doc := boxtest.FakeHTML(t, string(source),
		boxtest.WithDefaultUA(),
		boxtest.WithStylesheet("@page { size: 500px 800px }"))
	return pdf.NewDocument(doc.RenderPages(t), doc.HTML.Metadata())
}

func rasterizeAcid2(t *testing.T, doc *pdf.Document) (width, height int, pixels []testutil.Pixel) {
	t.Helper()
	var out bytes.Buffer
	opts := pdf.Options{Zoom: testutil.PixelZoom, Uncompressed: true}
	if err := doc.WritePDF(&out, opts); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	return testutil.ImagePixels(testutil.RasterizePDF(t, out.Bytes(), testutil.PixelZoom))
}

func TestAcid2MatchesItsReference(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.RequireGhostscript(t)
	testutil.CaptureLogs(t)

	document := renderAcid2Document(t, "acid2-test.html")
	if len(document.Pages) != 2 {
		t.Fatalf("test document has %d pages, want an introduction and the test page", len(document.Pages))
	}
	// The introduction page is not part of the reference.
	_, _, got := rasterizeAcid2(t, document.Copy(document.Pages[1:]))

	width, height, want := rasterizeAcid2(t, renderAcid2Document(t, "acid2-reference.html"))
	testutil.AssertPixelsEqual(t, "TestAcid2MatchesItsReference", width, height, got, want, 2)
}
