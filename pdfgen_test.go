package pdfgen_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/tomasweigenast/pdfgen"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// parsePDF checks that data is accepted by a PDF parser, in relaxed mode as
// the other tests do, and returns what it read.
func parsePDF(t *testing.T, data []byte) *model.Context {
	t.Helper()
	api.DisableConfigDir()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	ctx, err := api.ReadValidateAndOptimize(bytes.NewReader(data), conf)
	if err != nil {
		t.Fatalf("PDF parser rejects the output: %v", err)
	}
	return ctx
}

// render writes a document as an uncompressed PDF file unless opts say
// otherwise, and checks that it can be parsed.
func render(t *testing.T, h *pdfgen.HTML, opts pdfgen.Options) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := h.WritePDF(&out, opts); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}
	parsePDF(t, out.Bytes())
	return out.Bytes()
}

func fromString(t *testing.T, source string, o ...pdfgen.SourceOption) *pdfgen.HTML {
	t.Helper()
	h, err := pdfgen.HTMLFromString(source, o...)
	if err != nil {
		t.Fatalf("HTMLFromString: %v", err)
	}
	return h
}

func mediaBox(width, height string) string {
	return fmt.Sprintf("/MediaBox [0 0 %s %s]", width, height)
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			img.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRenderPagesAndPageSize(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		pages    int
		mediaBox string
	}{
		{"default size is A4", `<p>`, 1, mediaBox("595.275591", "841.889764")},
		{"page size from @page", `<style>@page{size:100px 50px;margin:0}</style>`, 1, mediaBox("75", "37.5")},
		{
			"page breaks",
			`<style>@page{size:100px;margin:0} div{height:100px;break-after:page}</style><div></div><div></div><div></div><p>`,
			4, mediaBox("75", "75"),
		},
		{
			"content overflows to the next page",
			`<style>@page{size:100px;margin:0} div{height:40px}</style><div></div><div></div><div></div>`,
			2, mediaBox("75", "75"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			h := fromString(t, tt.html)
			doc, err := h.Render(pdfgen.Options{})
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if len(doc.Pages) != tt.pages {
				t.Errorf("len(Pages) = %d, want %d", len(doc.Pages), tt.pages)
			}
			var out bytes.Buffer
			if err := doc.WritePDF(&out, pdfgen.Options{UncompressedPDF: true}); err != nil {
				t.Fatalf("WritePDF: %v", err)
			}
			if got := parsePDF(t, out.Bytes()).PageCount; got != tt.pages {
				t.Errorf("PDF page count = %d, want %d", got, tt.pages)
			}
			if !bytes.Contains(out.Bytes(), []byte(tt.mediaBox)) {
				t.Errorf("PDF has no %q", tt.mediaBox)
			}
		})
	}
}

func TestDocumentPageSizeIsInCSSPixels(t *testing.T) {
	doc, err := fromString(t, `<style>@page{size:100px 50px;margin:0}</style>`).Render(pdfgen.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p := doc.Pages[0]; p.Width != 100 || p.Height != 50 {
		t.Errorf("page size = %gx%g, want 100x50", p.Width, p.Height)
	}
}

func TestMetadata(t *testing.T) {
	testutil.AssertNoLogs(t)
	h := fromString(t, `<!doctype html><html lang="es"><head>
		<title>Quarterly report</title>
		<meta name="author" content="A. Writer">
		<meta name="keywords" content="sales, q3">
		<meta name="description" content="Numbers">
		</head><body>`)
	doc, err := h.Render(pdfgen.Options{})
	if err != nil {
		t.Fatal(err)
	}
	m := doc.Metadata
	if m.Title != "Quarterly report" || m.Lang != "es" || m.Description != "Numbers" {
		t.Errorf("Metadata = %+v", m)
	}
	if strings.Join(m.Authors, ",") != "A. Writer" || strings.Join(m.Keywords, ",") != "sales,q3" {
		t.Errorf("Authors = %q, Keywords = %q", m.Authors, m.Keywords)
	}

	var out bytes.Buffer
	if err := doc.WritePDF(&out, pdfgen.Options{}); err != nil {
		t.Fatal(err)
	}
	ctx := parsePDF(t, out.Bytes())
	if ctx.Title != "Quarterly report" || ctx.Author != "A. Writer" {
		t.Errorf("Info title = %q, author = %q", ctx.Title, ctx.Author)
	}
}

func TestSourcesGiveTheSameDocument(t *testing.T) {
	testutil.AssertNoLogs(t)
	const page = `<title>Same</title><style>@page{size:80px 40px;margin:0}</style><p>`
	dir := writeFiles(t, map[string]string{"doc.html": page})
	path := filepath.Join(dir, "doc.html")
	server := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer server.Close()

	tests := []struct {
		name string
		load func() (*pdfgen.HTML, error)
	}{
		{"string", func() (*pdfgen.HTML, error) { return pdfgen.HTMLFromString(page) }},
		{"file", func() (*pdfgen.HTML, error) { return pdfgen.HTMLFromFile(path) }},
		{"url", func() (*pdfgen.HTML, error) { return pdfgen.HTMLFromURL(server.URL + "/doc.html") }},
		{"reader", func() (*pdfgen.HTML, error) { return pdfgen.HTMLFromReader(strings.NewReader(page)) }},
		{"guess path", func() (*pdfgen.HTML, error) { return pdfgen.HTMLGuess(path) }},
		{"guess url", func() (*pdfgen.HTML, error) { return pdfgen.HTMLGuess(server.URL + "/doc.html") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := tt.load()
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			out := render(t, h, pdfgen.Options{UncompressedPDF: true})
			if !bytes.Contains(out, []byte(mediaBox("60", "30"))) {
				t.Errorf("PDF has no %q", mediaBox("60", "30"))
			}
			if got := parsePDF(t, out).Title; got != "Same" {
				t.Errorf("title = %q, want Same", got)
			}
		})
	}
}

func TestMissingSourcesAreErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.html")
	if _, err := pdfgen.HTMLFromFile(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("HTMLFromFile error = %v, want one wrapping fs.ErrNotExist", err)
	}
	if _, err := pdfgen.HTMLGuess(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("HTMLGuess error = %v, want one wrapping fs.ErrNotExist", err)
	}
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	if _, err := pdfgen.HTMLFromURL(server.URL + "/missing.html"); err == nil {
		t.Error("HTMLFromURL of a 404 page succeeded")
	}
	if _, err := pdfgen.CSSFromFile(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("CSSFromFile error = %v, want one wrapping fs.ErrNotExist", err)
	}
}

func TestEncoding(t *testing.T) {
	latin1 := []byte("<title>caf\xe9</title>")
	tests := []struct {
		name  string
		input []byte
		opts  []pdfgen.SourceOption
		want  string
	}{
		{"forced", latin1, []pdfgen.SourceOption{pdfgen.WithEncoding("iso-8859-1")}, "café"},
		{"declared", append([]byte(`<meta charset="windows-1252">`), latin1...), nil, "café"},
		{"utf-8", []byte("<title>café</title>"), nil, "café"},
		{"unknown label is skipped", append([]byte(`<meta charset="windows-1252">`), latin1...),
			[]pdfgen.SourceOption{pdfgen.WithEncoding("no-such-encoding")}, "café"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := pdfgen.HTMLFromReader(bytes.NewReader(tt.input), tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := h.Render(pdfgen.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if doc.Metadata.Title != tt.want {
				t.Errorf("title = %q, want %q", doc.Metadata.Title, tt.want)
			}
		})
	}
}

func TestBaseURLResolvesResources(t *testing.T) {
	const style = "@page { size: 120px 60px; margin: 0 }"
	const doc = `<link rel="stylesheet" href="css/style.css"><img src="img/pic.png">`
	dir := writeFiles(t, map[string]string{
		"css/style.css": style,
		"img/pic.png":   string(pngBytes(t)),
		"page.html":     doc,
	})
	server := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer server.Close()

	tests := []struct {
		name string
		load func(t *testing.T) *pdfgen.HTML
	}{
		{"base URL option with a path", func(t *testing.T) *pdfgen.HTML {
			return fromString(t, doc, pdfgen.WithBaseURL(dir+"/"))
		}},
		{"base URL option with a URL", func(t *testing.T) *pdfgen.HTML {
			return fromString(t, doc, pdfgen.WithBaseURL(server.URL+"/"))
		}},
		{"base element", func(t *testing.T) *pdfgen.HTML {
			return fromString(t, `<base href="`+server.URL+`/">`+doc)
		}},
		{"file", func(t *testing.T) *pdfgen.HTML {
			h, err := pdfgen.HTMLFromFile(filepath.Join(dir, "page.html"))
			if err != nil {
				t.Fatal(err)
			}
			return h
		}},
		{"file with another base URL", func(t *testing.T) *pdfgen.HTML {
			h, err := pdfgen.HTMLFromFile(filepath.Join(dir, "page.html"), pdfgen.WithBaseURL(server.URL+"/"))
			if err != nil {
				t.Fatal(err)
			}
			return h
		}},
		{"url", func(t *testing.T) *pdfgen.HTML {
			h, err := pdfgen.HTMLFromURL(server.URL + "/page.html")
			if err != nil {
				t.Fatal(err)
			}
			return h
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			out := render(t, tt.load(t), pdfgen.Options{UncompressedPDF: true})
			if !bytes.Contains(out, []byte(mediaBox("90", "45"))) {
				t.Errorf("the stylesheet was not applied: PDF has no %q", mediaBox("90", "45"))
			}
			if !bytes.Contains(out, []byte("/Subtype /Image")) {
				t.Error("the image was not embedded")
			}
		})
	}
}

func TestUnresolvableResourcesAreReportedAndSkipped(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	h := fromString(t, `<link rel="stylesheet" href="style.css"><img src="pic.png"><title>Still here</title>`)
	out := render(t, h, pdfgen.Options{UncompressedPDF: true})
	if got := parsePDF(t, out).Title; got != "Still here" {
		t.Errorf("title = %q", got)
	}
	if bytes.Contains(out, []byte("/Subtype /Image")) {
		t.Error("an image without a base URL was embedded")
	}
	if len(logs.Warnings()) == 0 {
		t.Error("nothing was logged about the missing resources")
	}
}

type fakeFetcher map[string]struct{ mime, body string }

func (f fakeFetcher) Fetch(_ context.Context, url string, _ http.Header) (*pdfgen.Response, error) {
	r, ok := f[url]
	if !ok {
		return nil, fmt.Errorf("no resource %s", url)
	}
	return &pdfgen.Response{
		URL:      url,
		Status:   http.StatusOK,
		MIMEType: r.mime,
		Body:     io.NopCloser(strings.NewReader(r.body)),
	}, nil
}

func TestFetcherServesTheResources(t *testing.T) {
	testutil.AssertNoLogs(t)
	fetcher := fakeFetcher{
		"https://example.test/doc.html":  {"text/html", `<link rel=stylesheet href=a.css><title>Fetched</title>`},
		"https://example.test/a.css":     {"text/css", `@import "b.css"`},
		"https://example.test/b.css":     {"text/css", `@page { size: 40px 20px; margin: 0 }`},
		"https://example.test/user.css":  {"text/css", `@page { margin: 0 }`},
		"https://example.test/local.css": {"text/css", `@page { size: 40px 20px; margin: 0 }`},
	}
	h, err := pdfgen.HTMLFromURL("https://example.test/doc.html", pdfgen.WithFetcher(fetcher))
	if err != nil {
		t.Fatal(err)
	}
	out := render(t, h, pdfgen.Options{UncompressedPDF: true})
	if !bytes.Contains(out, []byte(mediaBox("30", "15"))) {
		t.Errorf("PDF has no %q", mediaBox("30", "15"))
	}

	css, err := pdfgen.CSSFromURL("https://example.test/user.css", pdfgen.WithFetcher(fetcher))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Render(pdfgen.Options{Stylesheets: []*pdfgen.CSS{css}}); err != nil {
		t.Errorf("Render with a stylesheet from the fetcher: %v", err)
	}
}

func TestUserStylesheets(t *testing.T) {
	testutil.AssertNoLogs(t)
	dir := writeFiles(t, map[string]string{
		"user.css":     `@import "imported.css"; body { margin: 0 }`,
		"imported.css": `@page { size: 60px 60px; margin: 0 }`,
		"latin1.css":   "@charset \"iso-8859-1\"; p::before { content: \"\xe9\" }",
	})
	server := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer server.Close()

	fromFile, err := pdfgen.CSSFromFile(filepath.Join(dir, "user.css"))
	if err != nil {
		t.Fatal(err)
	}
	fromURL, err := pdfgen.CSSFromURL(server.URL + "/user.css")
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := pdfgen.CSSFromReader(strings.NewReader(`@page { size: 60px 60px; margin: 0 }`))
	if err != nil {
		t.Fatal(err)
	}
	guessed, err := pdfgen.CSSGuess(filepath.Join(dir, "user.css"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		css  *pdfgen.CSS
	}{
		{"string", pdfgen.CSSFromString(`@page { size: 60px 60px; margin: 0 }`)},
		{"file with an import", fromFile},
		{"url with an import", fromURL},
		{"reader", fromReader},
		{"guess", guessed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := render(t, fromString(t, `<p>`), pdfgen.Options{UncompressedPDF: true, Stylesheets: []*pdfgen.CSS{tt.css}})
			if !bytes.Contains(out, []byte(mediaBox("45", "45"))) {
				t.Errorf("PDF has no %q", mediaBox("45", "45"))
			}
		})
	}

	t.Run("author styles take precedence over user styles", func(t *testing.T) {
		h := fromString(t, `<style>@page{size:100px 100px;margin:0}</style>`)
		css := pdfgen.CSSFromString(`@page { size: 60px 60px; margin: 0 }`)
		out := render(t, h, pdfgen.Options{UncompressedPDF: true, Stylesheets: []*pdfgen.CSS{css}})
		if !bytes.Contains(out, []byte(mediaBox("75", "75"))) {
			t.Errorf("PDF has no %q", mediaBox("75", "75"))
		}
	})

	t.Run("important user styles take precedence over author styles", func(t *testing.T) {
		h := fromString(t, `<style>@page{size:100px 100px;margin:0}</style>`)
		css := pdfgen.CSSFromString(`@page { size: 60px 60px !important; margin: 0 }`)
		out := render(t, h, pdfgen.Options{UncompressedPDF: true, Stylesheets: []*pdfgen.CSS{css}})
		if !bytes.Contains(out, []byte(mediaBox("45", "45"))) {
			t.Errorf("PDF has no %q", mediaBox("45", "45"))
		}
	})

	t.Run("nil stylesheet", func(t *testing.T) {
		_, err := fromString(t, `<p>`).Render(pdfgen.Options{Stylesheets: []*pdfgen.CSS{nil}})
		if err == nil {
			t.Error("a nil stylesheet was accepted")
		}
	})
}

func TestMediaType(t *testing.T) {
	const page = `<style media="screen">@page{size:200px 200px;margin:0}</style>
		<style media="print">@page{size:100px 100px;margin:0}</style>`
	tests := []struct {
		name string
		opts []pdfgen.SourceOption
		want string
	}{
		{"print by default", nil, mediaBox("75", "75")},
		{"print", []pdfgen.SourceOption{pdfgen.WithMediaType("print")}, mediaBox("75", "75")},
		{"screen", []pdfgen.SourceOption{pdfgen.WithMediaType("screen")}, mediaBox("150", "150")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := render(t, fromString(t, page, tt.opts...), pdfgen.Options{UncompressedPDF: true})
			if !bytes.Contains(out, []byte(tt.want)) {
				t.Errorf("PDF has no %q", tt.want)
			}
		})
	}

	t.Run("user stylesheet with its own media type", func(t *testing.T) {
		css := pdfgen.CSSFromString(`@media screen { @page { size: 200px 200px; margin: 0 } } @media print { @page { size: 100px 100px; margin: 0 } }`,
			pdfgen.WithMediaType("screen"))
		out := render(t, fromString(t, `<p>`), pdfgen.Options{UncompressedPDF: true, Stylesheets: []*pdfgen.CSS{css}})
		if !bytes.Contains(out, []byte(mediaBox("150", "150"))) {
			t.Errorf("PDF has no %q", mediaBox("150", "150"))
		}
	})
}

func TestWritePDFOptions(t *testing.T) {
	const page = `<style>@page{size:100px 100px;margin:0}</style><div style="height:10px;background:red"></div>`
	h := fromString(t, page)

	t.Run("zoom", func(t *testing.T) {
		out := render(t, h, pdfgen.Options{UncompressedPDF: true, Zoom: 2})
		if !bytes.Contains(out, []byte(mediaBox("150", "150"))) {
			t.Errorf("PDF has no %q", mediaBox("150", "150"))
		}
	})

	t.Run("compression", func(t *testing.T) {
		compressed := render(t, h, pdfgen.Options{})
		plain := render(t, h, pdfgen.Options{UncompressedPDF: true})
		if !bytes.Contains(compressed, []byte("/FlateDecode")) {
			t.Error("the default PDF is not compressed")
		}
		if bytes.Contains(plain, []byte("/FlateDecode")) {
			t.Error("the uncompressed PDF has compressed streams")
		}
	})

	t.Run("version", func(t *testing.T) {
		out := render(t, h, pdfgen.Options{PDFVersion: "1.4"})
		if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
			t.Errorf("header = %q, want version 1.4", out[:8])
		}
	})

	t.Run("identifier", func(t *testing.T) {
		out := render(t, h, pdfgen.Options{UncompressedPDF: true, PDFIdentifier: []byte("fixed-id")})
		if !bytes.Contains(out, []byte("/ID")) {
			t.Fatal("PDF has no /ID")
		}
		again := render(t, h, pdfgen.Options{UncompressedPDF: true, PDFIdentifier: []byte("fixed-id")})
		if !bytes.Equal(out, again) {
			t.Error("two writes with the same identifier differ")
		}
	})
}

func TestInvalidOptionsAreErrors(t *testing.T) {
	h := fromString(t, `<p>`)
	tests := []struct {
		name string
		opts pdfgen.Options
	}{
		{"negative zoom", pdfgen.Options{Zoom: -1}},
		{"malformed version", pdfgen.Options{PDFVersion: "seven"}},
		{"JPEG quality above range", pdfgen.Options{JPEGQuality: 96}},
		{"negative JPEG quality", pdfgen.Options{JPEGQuality: -1}},
		{"negative DPI", pdfgen.Options{DPI: -72}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := h.WritePDF(&out, tt.opts); err == nil {
				t.Error("WritePDF accepted the options")
			}
			if out.Len() != 0 {
				t.Errorf("WritePDF wrote %d bytes before failing", out.Len())
			}
		})
	}
}

func TestImageOptions(t *testing.T) {
	testutil.AssertNoLogs(t)
	dir := writeFiles(t, map[string]string{"pic.png": string(pngBytes(t))})
	h := fromString(t, `<img src="pic.png" style="width:40px">`, pdfgen.WithBaseURL(dir+"/"))
	for _, opts := range []pdfgen.Options{
		{OptimizeImages: true},
		{DPI: 10},
		{JPEGQuality: 50},
	} {
		out := render(t, h, pdfgen.Options{
			UncompressedPDF: true, OptimizeImages: opts.OptimizeImages, DPI: opts.DPI, JPEGQuality: opts.JPEGQuality,
		})
		if !bytes.Contains(out, []byte("/Subtype /Image")) {
			t.Errorf("options %+v: the image was not embedded", opts)
		}
	}
}

func TestPresentationalHints(t *testing.T) {
	h := fromString(t, `<body bgcolor="#ff0000">`)
	with := render(t, h, pdfgen.Options{UncompressedPDF: true, PresentationalHints: true})
	without := render(t, h, pdfgen.Options{UncompressedPDF: true})
	const red = "1 0 0 rg"
	if !bytes.Contains(with, []byte(red)) {
		t.Errorf("with hints, the page is not painted red (no %q)", red)
	}
	if bytes.Contains(without, []byte(red)) {
		t.Errorf("without hints, the page is painted red")
	}
}

func TestFontConfigurationIsSharedBetweenRenders(t *testing.T) {
	testutil.AssertNoLogs(t)
	fonts := pdfgen.NewFontConfiguration(pdfgen.WithoutSystemFonts())
	h := fromString(t, `<style>
		@font-face { font-family: sample; src: url(testfont.otf) }
		body { font: 16px sample }
	</style><p>abc`, pdfgen.WithBaseURL(testutil.ResourceURL(t, "x")))
	for i := 0; i < 2; i++ {
		out := render(t, h, pdfgen.Options{UncompressedPDF: true, FontConfig: fonts})
		if !bytes.Contains(out, []byte("/FontFile")) {
			t.Errorf("render %d: the font was not embedded", i+1)
		}
	}
}

func TestWriteFile(t *testing.T) {
	testutil.AssertNoLogs(t)
	h := fromString(t, `<title>On disk</title><style>@page{size:50px 50px;margin:0}</style>`)
	path := filepath.Join(t.TempDir(), "out.pdf")
	if err := h.WriteFile(path, pdfgen.Options{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsePDF(t, data).Title; got != "On disk" {
		t.Errorf("title = %q", got)
	}

	if err := h.WriteFile(filepath.Join(t.TempDir(), "no", "such", "dir.pdf"), pdfgen.Options{}); err == nil {
		t.Error("WriteFile into a missing directory succeeded")
	}

	bad := filepath.Join(t.TempDir(), "bad.pdf")
	if err := h.WriteFile(bad, pdfgen.Options{Zoom: -1}); err == nil {
		t.Error("WriteFile with an invalid zoom succeeded")
	}
	if _, err := os.Stat(bad); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a failed WriteFile left %s behind", bad)
	}
}

func TestDocumentIsWrittenMoreThanOnce(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc, err := fromString(t, `<style>@page{size:50px 50px;margin:0}</style><p>`).Render(pdfgen.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var first, second bytes.Buffer
	for _, buf := range []*bytes.Buffer{&first, &second} {
		if err := doc.WritePDF(buf, pdfgen.Options{UncompressedPDF: true, PDFIdentifier: []byte("id")}); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("writing a document twice gives different files")
	}
	parsePDF(t, first.Bytes())
}
