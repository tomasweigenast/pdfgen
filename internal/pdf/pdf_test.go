package pdf_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// writePDF lays out source and writes it as an uncompressed PDF file.
func writePDF(t *testing.T, source string, opts pdf.Options) []byte {
	t.Helper()
	doc := boxtest.FakeHTML(t, source)
	pages := doc.RenderPages(t)
	opts.Uncompressed = true
	var out bytes.Buffer
	if err := pdf.NewDocument(pages, doc.HTML.Metadata()).WritePDF(&out, opts); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	return out.Bytes()
}

func TestPageSizeFollowsZoom(t *testing.T) {
	for _, zoom := range []float64{1, 1.5, 0.5} {
		t.Run(fmt.Sprint(zoom), func(t *testing.T) {
			testutil.AssertNoLogs(t)
			out := writePDF(t, `<style>@page{size:3in 4in`, pdf.Options{Zoom: zoom})
			want := fmt.Sprintf("/MediaBox [0 0 %d %d]", int(216*zoom), int(288*zoom))
			if !bytes.Contains(out, []byte(want)) {
				t.Errorf("PDF has no %q", want)
			}
		})
	}
}

func TestPageBoxesWithBleed(t *testing.T) {
	tests := []struct {
		name, style, media, bleed, trim string
	}{
		{"uniform", "bleed: 30pt; size: 10pt", "[-30 -30 40 40]", "[-10 -10 20 20]", "[0 0 10 10]"},
		{"per side", "bleed: 15pt 3pt 6pt 18pt; size: 12pt 15pt", "[-18 -15 15 21]", "[-10 -10 15 21]", "[0 0 12 15]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			out := writePDF(t, fmt.Sprintf(`
				<title>Test document</title>
				<style>@page { %s }</style>
				<body>test`, tt.style), pdf.Options{})
			for key, want := range map[string]string{"MediaBox": tt.media, "BleedBox": tt.bleed, "TrimBox": tt.trim} {
				if entry := "/" + key + " " + want; !bytes.Contains(out, []byte(entry)) {
					t.Errorf("PDF has no %q", entry)
				}
			}
		})
	}
}
