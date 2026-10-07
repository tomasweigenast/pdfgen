package testutil

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// AssertPixelsEqual fails the test when got and want differ by more than
// tolerance on any channel. Pixels with Any set in want are not compared.
// On failure, name.png and name.expected.png are written to testdata/results
// and the first differing pixel is reported.
func AssertPixelsEqual(t testing.TB, name string, width, height int, got, want []Pixel, tolerance int) {
	t.Helper()
	for i := 0; i < max(len(got), len(want)); i++ {
		g, w := Pixel{R: 255, G: 255, B: 255}, Pixel{R: 255, G: 255, B: 255}
		missingGot, missingWant := i >= len(got), i >= len(want)
		if !missingGot {
			g = got[i]
		}
		if !missingWant {
			w = want[i]
		}
		if w.Any && !missingWant {
			continue
		}
		if !missingGot && !missingWant && withinTolerance(g, w, tolerance) {
			continue
		}

		writePixelsPNG(t, name, got, width, len(got)/width)
		expected := make([]Pixel, len(want))
		for j, p := range want {
			if p.Any {
				p = Pixel{R: 255, G: 255, B: 255}
			}
			expected[j] = p
		}
		writePixelsPNG(t, name+".expected", expected, width, height)

		x, y := i%width, i/width
		switch {
		case missingGot:
			t.Fatalf("Pixel (%d, %d) in %s: expected rgb(%d, %d, %d), got no pixel", x, y, name, w.R, w.G, w.B)
		case missingWant:
			t.Fatalf("Pixel (%d, %d) in %s: expected no pixel, got rgb(%d, %d, %d)", x, y, name, g.R, g.G, g.B)
		}
		t.Fatalf("Pixel (%d, %d) in %s: expected rgb(%d, %d, %d), got rgb(%d, %d, %d)",
			x, y, name, w.R, w.G, w.B, g.R, g.G, g.B)
	}
}

func withinTolerance(a, b Pixel, tol int) bool {
	return absDiff(a.R, b.R) <= tol && absDiff(a.G, b.G) <= tol && absDiff(a.B, b.B) <= tol
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func writePixelsPNG(t testing.TB, name string, pixels []Pixel, width, height int) {
	t.Helper()
	if width <= 0 || height <= 0 {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, p := range pixels {
		if i >= width*height {
			break
		}
		img.SetRGBA(i%width, i/width, color.RGBA{R: p.R, G: p.G, B: p.B, A: 255})
	}
	path := filepath.Join(ResultsDir(t), name+".png")
	f, err := os.Create(path)
	if err != nil {
		t.Logf("write %s: %v", path, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Logf("encode %s: %v", path, err)
	}
}
