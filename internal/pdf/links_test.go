package pdf_test

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// Top and right positions of an A4 page in points, rounded to the six digits
// of the writer.
var (
	top   = round6(297 * 72 / 25.4)
	right = round6(210 * 72 / 25.4)
)

func round6(v float64) float64 { return math.Round(v*1e6) / 1e6 }

var (
	uriRE     = regexp.MustCompile(`/URI \((.*)\)`)
	actionRE  = regexp.MustCompile(`/S (/\w*)`)
	subtypeRE = regexp.MustCompile(`/Subtype (/\w*)`)
	rectRE    = regexp.MustCompile(`/Rect \[([\d.]+ [\d.]+ [\d.]+ [\d.]+)\]`)
)

func floats(t *testing.T, s string) []float64 {
	t.Helper()
	var out []float64
	for _, f := range strings.Fields(s) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			t.Fatalf("parse %q: %v", f, err)
		}
		out = append(out, v)
	}
	return out
}

func rects(t *testing.T, out []byte) [][]float64 {
	t.Helper()
	var got [][]float64
	for _, m := range matches(rectRE, out) {
		got = append(got, floats(t, m))
	}
	return got
}

// destination returns the position of the named destination, which is
// written as (name) [page /XYZ x y 0].
func destination(t *testing.T, out []byte, name string) []float64 {
	t.Helper()
	re := regexp.MustCompile(`\(` + regexp.QuoteMeta(name) + `\) \[\d+ 0 R /XYZ ([\d.]+ [\d.]+ [\d.]+)]`)
	m := re.FindSubmatch(out)
	if m == nil {
		t.Fatalf("PDF has no destination %q", name)
	}
	return floats(t, string(m[1]))
}

func assertFloats(t *testing.T, what string, got, want []float64) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestNoAnnotationsWithoutLinks(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<body>`, pdf.Options{})
	if bytes.Contains(out, []byte("Annots")) {
		t.Error("PDF has /Annots")
	}
}

func TestLinkAnnotations(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `
		<style>
			body { margin: 0; font-size: 10pt; line-height: 2 }
			p { display: block; height: 90pt; margin: 0 0 10pt 0 }
			img { width: 30pt; vertical-align: top }
		</style>
		<p><a href=" https://exam
ple.org	"><img src=pattern.png></a></p>
		<p style="padding: 0 10pt"><a
			href="#lipsum"><img style="border: solid 1pt"
				src=pattern.png></a></p>
		<p id=hello>Hello, World</p>
		<p id=lipsum>
			<a style="display: block; page-break-before: always; height: 30pt"
				href="#hel%6Co"></a>a
		</p>`, pdf.Options{})

	uris := matches(uriRE, out)
	actions := matches(actionRE, out)
	subtypes := matches(subtypeRE, out)
	gotRects := rects(t, out)
	if len(gotRects) != 5 {
		t.Fatalf("got %d link rectangles, want 5", len(gotRects))
	}

	// 30pt wide (like the image), 20pt high (like the line height).
	if !slices.Equal(uris, []string{"https://example.org", "https://example.org"}) {
		t.Errorf("URIs = %q", uris)
	}
	if !slices.Equal(actions, []string{"/URI", "/URI"}) {
		t.Errorf("actions = %q, want two /URI", actions)
	}
	if want := slices.Repeat([]string{"/Link"}, 5); len(subtypes) < 5 || !slices.Equal(subtypes[:5], want) {
		t.Errorf("subtypes = %q, want five /Link first", subtypes)
	}
	assertFloats(t, "line box rectangle", gotRects[0], []float64{0, top, 30, top - 20})

	// The image itself: 30*30pt.
	assertFloats(t, "image rectangle", gotRects[1], []float64{0, top, 30, top - 30})

	// 32pt wide (image and 2 * 1pt of border), 20pt high.
	if !bytes.Contains(out, []byte("/Dest (lipsum)")) {
		t.Error("PDF has no /Dest (lipsum)")
	}
	assertFloats(t, "lipsum destination", destination(t, out, "lipsum"), []float64{0, top, 0})
	assertFloats(t, "bordered line rectangle", gotRects[2], []float64{10, top - 100, 10 + 32, top - 100 - 20})

	// The image itself: 32*32pt.
	assertFloats(t, "bordered image rectangle", gotRects[3], []float64{10, top - 100, 10 + 32, top - 100 - 32})

	// 100% wide (block), 30pt high.
	if !bytes.Contains(out, []byte("/Dest (hello)")) {
		t.Error("PDF has no /Dest (hello)")
	}
	assertFloats(t, "hello destination", destination(t, out, "hello"), []float64{0, top - 200, 0})
	assertFloats(t, "block rectangle", gotRects[4], []float64{0, top, right, top - 30})
}

func TestLinkDestinationsAreSortedByName(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `
		<p id="zzz">zzz</p>
		<p id="aaa">aaa</p>
		<a href="#zzz">z</a>
		<a href="#aaa">a</a>`, pdf.Options{})
	_, afterAaa, _ := bytes.Cut(out, []byte("(aaa) ["))
	if !bytes.Contains(afterAaa, []byte("(zzz) [")) {
		t.Error("destination zzz does not come after aaa")
	}
}

func TestRelativeLinks(t *testing.T) {
	tests := []struct {
		name, html string
		opts       []boxtest.Option
		uri        string
	}{
		{
			name: "no height",
			html: `<a href="../lipsum" style="display: block"></a>a`,
			opts: []boxtest.Option{boxtest.WithBaseURL("https://example.org/foo/bar/")},
			uri:  "https://example.org/foo/lipsum",
		},
		{
			name: "missing base",
			html: `<a href="../lipsum" style="display: block"></a>a`,
			opts: []boxtest.Option{boxtest.WithoutBaseURL()},
			uri:  "../lipsum",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			out := writePDF(t, tt.html, pdf.Options{}, tt.opts...)
			if got := matches(uriRE, out); !slices.Equal(got, []string{tt.uri}) {
				t.Errorf("URIs = %q, want %q", got, tt.uri)
			}
			// A block without height still has a rectangle, as wide as the page.
			want := fmt.Sprintf("/Rect [0 %s %s %s]", fmtReal(top), fmtReal(right), fmtReal(top))
			if !bytes.Contains(out, []byte(want)) {
				t.Errorf("PDF has no %q", want)
			}
		})
	}
}

func fmtReal(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func TestInternalLinkToItself(t *testing.T) {
	// A reference without a base URI is fine when it points inside the
	// document.
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<a href="#lipsum" id="lipsum" style="display: block"></a>a`,
		pdf.Options{}, boxtest.WithoutBaseURL())
	if !bytes.Contains(out, []byte("/Dest (lipsum)")) {
		t.Error("PDF has no /Dest (lipsum)")
	}
	assertFloats(t, "destination", destination(t, out, "lipsum"), []float64{0, top, 0})
	got := rects(t, out)
	if len(got) != 1 {
		t.Fatalf("got %d rectangles, want 1", len(got))
	}
	assertFloats(t, "rectangle", got[0], []float64{0, top, right, top})
}

func TestLinkToDifferentPathOfTheBase(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<a href="/test/lipsum"></a>a`, pdf.Options{},
		boxtest.WithBaseURL("https://example.org/foo/bar/"))
	if !bytes.Contains(out, []byte("https://example.org/test/lipsum")) {
		t.Error("PDF has no link to https://example.org/test/lipsum")
	}
}

func TestLinkToFragmentOfTheSameDocument(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `<a id="test" href="/foo/bar/#test"></a>a`, pdf.Options{},
		boxtest.WithBaseURL("https://example.org/foo/bar/"))
	if !bytes.Contains(out, []byte("/Dest (test)")) {
		t.Error("PDF has no /Dest (test)")
	}
}

func TestLinksToMissingAnchorsAreDropped(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	out := writePDF(t, `
		<style> a { display: block; height: 15pt } </style>
		<a href="#lipsum"></a>
		<a href="#missing" id="lipsum"></a>
		<a href=""></a>a`, pdf.Options{}, boxtest.WithoutBaseURL())
	if !bytes.Contains(out, []byte("/Dest (lipsum)")) {
		t.Error("PDF has no /Dest (lipsum)")
	}
	lines := logs.Lines()
	if len(lines) != 1 || lines[0] != "ERROR: No anchor #missing for internal URI reference" {
		t.Errorf("logs = %q, want the error about #missing", lines)
	}
	assertFloats(t, "destination", destination(t, out, "lipsum"), []float64{0, top - 15, 0})
	got := rects(t, out)
	if len(got) != 1 {
		t.Fatalf("got %d rectangles, want 1", len(got))
	}
	assertFloats(t, "rectangle", got[0], []float64{0, top, right, top - 15})
}

func TestAnchorOnFirstPageWins(t *testing.T) {
	testutil.AssertNoLogs(t)
	out := writePDF(t, `
		<style> a { display: block; break-after: page } </style>
		<div id="lipsum">
			<a href="#lipsum"></a>
			<a href="#lipsum"></a>
			<a href="#lipsum"></a>
		</div>`, pdf.Options{}, boxtest.WithoutBaseURL())
	kids := regexp.MustCompile(`/Kids \[(\d+) 0 R`).FindSubmatch(out)
	if kids == nil {
		t.Fatal("PDF has no page tree")
	}
	if want := "/Names [(lipsum) [" + string(kids[1]); !bytes.Contains(out, []byte(want)) {
		t.Errorf("PDF has no %q", want)
	}
}
