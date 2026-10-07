package testutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// PixelZoom is the zoom at which one CSS pixel rasterizes to exactly one
// device pixel when the PDF is rendered at 96 dpi: a CSS pixel is 0.75 pt, so
// zoom 4/30 shrinks the page and the rasterizer resolution compensates.
const PixelZoom = 4.0 / 30

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func ghostscriptPath() (string, bool) {
	names := []string{"gs"}
	if runtime.GOOS == "windows" {
		names = []string{"gswin64c", "gswin32c", "gs"}
	}
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p, true
		}
	}
	return "", false
}

// RequireGhostscript skips the test when Ghostscript is not installed. When
// the CI environment variable is set, a missing Ghostscript fails the test
// instead.
func RequireGhostscript(t testing.TB) string {
	t.Helper()
	p, ok := ghostscriptPath()
	if !ok {
		if os.Getenv("CI") != "" {
			t.Fatal("Ghostscript (gs) is required in CI but was not found in PATH")
		}
		t.Skip("Ghostscript (gs) not found in PATH")
	}
	return p
}

// RasterizePDFPages renders each page of a PDF to its own image at 96 dpi
// divided by zoom. The PDF is expected to have been written with the same
// zoom, so that one CSS pixel is one image pixel when zoom is PixelZoom.
func RasterizePDFPages(t testing.TB, pdf []byte, zoom float64) []*image.NRGBA {
	t.Helper()
	gs := RequireGhostscript(t)

	// Ghostscript on Windows cannot read binary data from stdin.
	file := filepath.Join(t.TempDir(), "document.pdf")
	if err := os.WriteFile(file, pdf, 0o600); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	cmd := exec.Command(gs,
		"-q", "-sDEVICE=png16m", "-dTextAlphaBits=1", "-dGraphicsAlphaBits=1",
		"-dBATCH", "-dNOPAUSE", "-dPDFSTOPONERROR",
		"-r"+strconv.FormatFloat(96/zoom, 'g', -1, 64),
		"-dUsePDFX3Profile", "-sOutputFile=-", file)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// A non-zero exit still leaves a diagnostic worth reporting below.
	runErr := cmd.Run()

	out := stdout.Bytes()
	if !bytes.HasPrefix(out, pngSignature) {
		msg := string(bytes.TrimSpace(append(append([]byte(nil), firstSegment(out)...), stderr.Bytes()...)))
		if msg == "" {
			msg = "no output"
		}
		if runErr != nil {
			msg = fmt.Sprintf("%s (%v)", msg, runErr)
		}
		t.Fatalf("Ghostscript error: %s", msg)
	}

	var pages []*image.NRGBA
	for len(out) > 0 {
		n, err := pngLength(out)
		if err != nil {
			t.Fatalf("Ghostscript output: %v", err)
		}
		img, err := png.Decode(bytes.NewReader(out[:n]))
		if err != nil {
			t.Fatalf("decode Ghostscript PNG: %v", err)
		}
		pages = append(pages, toNRGBA(img))
		out = out[n:]
	}
	return pages
}

func firstSegment(b []byte) []byte {
	if i := bytes.Index(b, pngSignature); i >= 0 {
		return b[:i]
	}
	return b
}

// pngLength returns the size of the PNG file at the start of b by walking its
// chunks up to IEND (PNG specification §5.3).
func pngLength(b []byte) (int, error) {
	if !bytes.HasPrefix(b, pngSignature) {
		return 0, fmt.Errorf("data does not start with a PNG signature")
	}
	pos := len(pngSignature)
	for {
		if pos+8 > len(b) {
			return 0, fmt.Errorf("truncated PNG stream")
		}
		size := int(binary.BigEndian.Uint32(b[pos:]))
		kind := string(b[pos+4 : pos+8])
		pos += 8 + size + 4 // length, type, data, CRC
		if pos > len(b) {
			return 0, fmt.Errorf("truncated PNG stream")
		}
		if kind == "IEND" {
			return pos, nil
		}
	}
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	n := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(n, n.Bounds(), img, b.Min, draw.Src)
	return n
}

// StackPages stacks page images vertically, centered horizontally, on a
// transparent canvas as wide as the widest page.
func StackPages(pages []*image.NRGBA) *image.NRGBA {
	if len(pages) == 1 {
		return pages[0]
	}
	var width, height int
	for _, p := range pages {
		width = max(width, p.Bounds().Dx())
		height += p.Bounds().Dy()
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	top := 0
	for _, p := range pages {
		w, h := p.Bounds().Dx(), p.Bounds().Dy()
		dst := image.Rect((width-w)/2, top, (width-w)/2+w, top+h)
		draw.Draw(out, dst, p, p.Bounds().Min, draw.Src)
		top += h
	}
	return out
}

// RasterizePDF renders a PDF to a single image: the pages of a multi-page
// document are stacked with StackPages.
func RasterizePDF(t testing.TB, pdf []byte, zoom float64) *image.NRGBA {
	t.Helper()
	return StackPages(RasterizePDFPages(t, pdf, zoom))
}

// WritePNG renders a PDF with RasterizePDF and returns the result encoded as
// a PNG file.
func WritePNG(t testing.TB, pdf []byte, zoom float64) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, RasterizePDF(t, pdf, zoom)); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return buf.Bytes()
}

// ImagePixels returns the RGB values of img in row-major order. Alpha is
// ignored, as in the comparison of expected pixel art.
func ImagePixels(img *image.NRGBA) (width, height int, pixels []Pixel) {
	b := img.Bounds()
	pixels = make([]Pixel, 0, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			pixels = append(pixels, Pixel{R: c.R, G: c.G, B: c.B})
		}
	}
	return b.Dx(), b.Dy(), pixels
}
