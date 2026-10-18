package pdf_test

import (
	"bytes"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

var dctFilter = []byte("/Filter /DCTDecode")

func TestGIFIsNotEmbeddedAsJPEG(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<img src="pattern.gif">`, pdf.Options{})
	if bytes.Contains(out, dctFilter) {
		t.Error("GIF image uses DCTDecode")
	}
}

func TestJPEGIsEmbeddedAsIs(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<img src="blue.jpg">`, pdf.Options{})
	if !bytes.Contains(out, dctFilter) {
		t.Error("JPEG image does not use DCTDecode")
	}
}

func TestRepeatedImageIsEmbeddedOnce(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `
		<img src="blue.jpg">
		<div style="background: url(blue.jpg)"></div>
		<img src="blue.jpg">
		<div style="background: url(blue.jpg) no-repeat"></div>`, pdf.Options{})
	if n := bytes.Count(out, dctFilter); n != 1 {
		t.Errorf("JPEG embedded %d times, want 1", n)
	}
}

func TestImagesFromSeparateRenderingsAreEmbedded(t *testing.T) {
	testutil.AssertNoLogs(t)
	first := boxtest.FakeHTML(t, `<img src="blue.jpg">`)
	second := boxtest.FakeHTML(t, `<img src="not-optimized.jpg">`)
	pages := append(first.RenderPages(t), second.RenderPages(t)...)

	var out bytes.Buffer
	if err := pdf.NewDocument(pages, first.HTML.Metadata()).WritePDF(&out, pdf.Options{Uncompressed: true}); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	if n := bytes.Count(out.Bytes(), dctFilter); n != 2 {
		t.Errorf("JPEG embedded %d times, want 2", n)
	}
}
