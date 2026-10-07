package testutil

import (
	"strconv"
	"testing"
)

// HTMLToPixels renders an HTML document and returns its pixels. It is set by
// the rendering pipeline once that exists; until then pixel assertions on
// HTML skip the test.
var HTMLToPixels = func(t testing.TB, html string) (width, height int, pixels []Pixel) {
	t.Helper()
	t.Skip("rendering pipeline not implemented")
	return 0, 0, nil
}

// AssertPixels renders html and compares the result with a pixel-art string.
func AssertPixels(t testing.TB, expected, html string) {
	t.Helper()
	width, height, pixels := HTMLToPixels(t, html)
	assertPixelArt(t, expected, width, height, pixels)
}

// AssertPixelsPDF rasterizes a PDF that was written at PixelZoom and compares
// the result with a pixel-art string.
func AssertPixelsPDF(t testing.TB, expected string, pdf []byte) {
	t.Helper()
	width, height, pixels := ImagePixels(RasterizePDF(t, pdf, PixelZoom))
	assertPixelArt(t, expected, width, height, pixels)
}

func assertPixelArt(t testing.TB, expected string, width, height int, pixels []Pixel) {
	t.Helper()
	ew, eh, want, err := ParsePixels(expected)
	if err != nil {
		t.Fatalf("invalid expected pixels: %v", err)
	}
	if ew != width || eh != height {
		t.Fatalf("Images do not have the same sizes:\n- expected: %d × %d\n- result: %d × %d", ew, eh, width, height)
	}
	AssertPixelsEqual(t, ResultName(t), width, height, pixels, want, 0)
}

// AssertSameRenderings renders every document and checks that they all look
// like the first one, within tolerance per channel.
func AssertSameRenderings(t testing.TB, tolerance int, htmls ...string) {
	t.Helper()
	var width, height int
	var reference []Pixel
	for i, html := range htmls {
		w, h, pixels := HTMLToPixels(t, html)
		if i == 0 {
			width, height, reference = w, h, pixels
			continue
		}
		AssertPixelsEqual(t, ResultName(t)+"_"+strconv.Itoa(i), width, height, pixels, reference, tolerance)
	}
}

// AssertDifferentRenderings renders every document and checks that no two of
// them look the same.
func AssertDifferentRenderings(t testing.TB, htmls ...string) {
	t.Helper()
	type rendering struct {
		width, height int
		pixels        []Pixel
	}
	renderings := make([]rendering, len(htmls))
	for i, html := range htmls {
		w, h, p := HTMLToPixels(t, html)
		renderings[i] = rendering{w, h, p}
	}
	for i, a := range renderings {
		for j := i + 1; j < len(renderings); j++ {
			b := renderings[j]
			if a.width == b.width && a.height == b.height && equalPixels(a.pixels, b.pixels) {
				writePixelsPNG(t, ResultName(t)+"_"+strconv.Itoa(i+1), a.pixels, a.width, a.height)
				t.Fatalf("renderings %d and %d are the same", i+1, j+1)
			}
		}
	}
}

func equalPixels(a, b []Pixel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
