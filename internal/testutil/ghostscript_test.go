package testutil

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// rect is a filled rectangle in CSS pixels, with y growing downwards.
type rect struct {
	x, y, w, h float64
	r, g, b    float64
}

// handmadePDF writes an uncompressed PDF with one page per rectangle list,
// each page sized in CSS pixels by sizes. Pages are
// scaled the way the page writer will do it: 0.75 pt per pixel times zoom,
// with the y axis flipped through the initial cm operator.
func handmadePDF(zoom float64, sizes [][2]float64, pages ...[]rect) []byte {
	scale := 0.75 * zoom
	var objs []string
	kids := make([]string, len(pages))
	for i, rects := range pages {
		w, h := sizes[i][0], sizes[i][1]
		var content strings.Builder
		fmt.Fprintf(&content, "%g 0 0 %g 0 %g cm\n", scale, -scale, h*scale)
		for _, r := range rects {
			fmt.Fprintf(&content, "%g %g %g rg\n%g %g %g %g re f\n", r.r, r.g, r.b, r.x, r.y, r.w, r.h)
		}
		pageObj, contentObj := 3+2*i, 4+2*i
		kids[i] = fmt.Sprintf("%d 0 R", pageObj)
		objs = append(objs,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %g %g] /Contents %d 0 R /Resources << >> >>",
				w*scale, h*scale, contentObj),
			fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", content.Len(), content.String()))
	}
	all := append([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)),
	}, objs...)

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(all))
	for i, o := range all {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(all)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(all)+1, xref)
	return buf.Bytes()
}

func TestAssertPixelsPDFOnHandmadePDF(t *testing.T) {
	pdf := handmadePDF(PixelZoom, [][2]float64{{5, 3}}, []rect{
		{x: 0, y: 0, w: 2, h: 2, r: 1},
		{x: 2, y: 1, w: 3, h: 2, b: 1},
	})
	AssertPixelsPDF(t, `
		RR___
		RRBBB
		__BBB
	`, pdf)
}

func TestRasterizePDFStacksPagesCentered(t *testing.T) {
	pdf := handmadePDF(PixelZoom, [][2]float64{{4, 2}, {2, 2}},
		[]rect{{x: 0, y: 0, w: 4, h: 2, r: 1}},
		[]rect{{x: 0, y: 0, w: 2, h: 2, b: 1}},
	)
	pages := RasterizePDFPages(t, pdf, PixelZoom)
	if len(pages) != 2 {
		t.Fatalf("pages = %d, want 2", len(pages))
	}
	w, h, pixels := ImagePixels(StackPages(pages))
	if w != 4 || h != 4 {
		t.Fatalf("stacked size = %d×%d, want 4×4", w, h)
	}
	// The padding around the narrower page is transparent, which compares
	// as black.
	AssertPixelsEqual(t, ResultName(t), w, h, pixels, mustPixels(t, `
		RRRR
		RRRR
		KBBK
		KBBK
	`), 0)
}

func TestRasterizePDFReportsGhostscriptFailure(t *testing.T) {
	if _, ok := ghostscriptPath(); !ok {
		t.Skip("Ghostscript (gs) not found in PATH")
	}

	msg := runRecorded(t, func(tb testing.TB) {
		RasterizePDF(tb, []byte("not a pdf"), PixelZoom)
	})
	if !strings.HasPrefix(msg, "Ghostscript error:") {
		t.Errorf("message = %q", msg)
	}
}

func TestPngLengthRejectsBadData(t *testing.T) {
	if _, err := pngLength([]byte("\x89PNG\r\n\x1a\nshort")); err == nil {
		t.Error("truncated stream accepted")
	}
	if _, err := pngLength([]byte("GIF89a")); err == nil {
		t.Error("non-PNG data accepted")
	}
}

func mustPixels(t testing.TB, art string) []Pixel {
	t.Helper()
	_, _, p, err := ParsePixels(art)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
