package images

import (
	"bytes"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// drawGradient paints the gradient of css over a w×h page.
func drawGradient(t testing.TB, w, h float64, css string) []byte {
	t.Helper()
	g := gradientFor(t, css)
	return renderPDF(t, w, h, func(s *stream.Stream) { drawAt(s, g, 0, 0, w, h, "auto") })
}

type gradientPixelCase struct {
	name     string
	w, h     float64
	css      string
	expected string
}

func runGradientPixelCases(t *testing.T, cases []gradientPixelCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.RequireGhostscript(t)
			testutil.AssertNoLogs(t)
			testutil.AssertPixelsPDF(t, tc.expected, drawGradient(t, tc.w, tc.h, tc.css))
		})
	}
}

func TestLinearGradientPixels(t *testing.T) {
	runGradientPixelCases(t, []gradientPixelCase{
		{"hard stops with zero-length steps", 5, 9, "linear-gradient(white, white 3px, blue 0, blue 5px, red 0, red)", `
			_____
			_____
			_____
			BBBBB
			BBBBB
			RRRRR
			RRRRR
			RRRRR
			RRRRR
		`},
		{"first stop with a position", 5, 9, "linear-gradient(white 3px, blue 0, blue 5px, red 0)", `
			_____
			_____
			_____
			BBBBB
			BBBBB
			RRRRR
			RRRRR
			RRRRR
			RRRRR
		`},
		{"to right", 9, 5, "linear-gradient(to right, white 3px, blue 0, blue 5px, red 0)", `
			___BBrrrr
			___BBrrrr
			___BBrrrr
			___BBrrrr
			___BBrrrr
		`},
		{"hard stops between solid colors", 10, 5, "linear-gradient(to right, blue 5px, blue 6px, red 6px, red 9px)", `
			BBBBBBrrrr
			BBBBBBrrrr
			BBBBBBrrrr
			BBBBBBrrrr
			BBBBBBrrrr
		`},
		{"repeating", 10, 5, "repeating-linear-gradient(to right, blue 50%, blue 60%, red 60%, red 90%)", `
			rBrrzBrrrB
			rBrrrBrrrB
			rBrrrBrrrB
			rBrrrBrrrB
			rBrrzBrrrB
		`},
		{"stops at one position", 9, 5, "linear-gradient(to right, blue 3px, blue 3px, red 3px, red 3px)", `
			BBBrrrrrr
			BBBrrrrrr
			BBBrrrrrr
			BBBrrrrrr
			BBBrrrrrr
		`},
		{"repeating stops at one position average", 9, 5, "repeating-linear-gradient(to right, black 3px, black 3px, #800080 3px, #800080 3px)", `
			hhhhhhhhh
			hhhhhhhhh
			hhhhhhhhh
			hhhhhhhhh
			hhhhhhhhh
		`},
		{"repeating with one stop", 9, 5, "repeating-linear-gradient(to right, blue 3px)", `
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
		`},
		{"repeating with one stop at an angle", 9, 5, "repeating-linear-gradient(45deg, blue 3px)", `
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
		`},
		{"stops crossing back at an angle", 9, 5, "linear-gradient(45deg, blue 3px, red 3px, red 3px, blue 3px)", `
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
		`},
		{"thin stripe", 9, 5, "linear-gradient(to right, blue 3px, red 3px, red 4px, blue 4px)", `
			BBBrBBBBB
			BBBrBBBBB
			BBBrBBBBB
			BBBrBBBBB
			BBBrBBBBB
		`},
		{"repeating thin stripe", 9, 5, "repeating-linear-gradient(to right, red 3px, blue 3px, blue 4px, red 4px)", `
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
			BBBBBBBBB
		`},
		{"translucent stop", 5, 9, "linear-gradient(white, white 3px, rgba(255, 0, 0, 0.751) 0, rgba(255, 0, 0, 0.751) 5px, red 0, red)", `
			_____
			_____
			_____
			SSSSS
			SSSSS
			RRRRR
			RRRRR
			RRRRR
			RRRRR
		`},
		{"currentcolor", 5, 5, "linear-gradient(currentcolor)", `
			KKKKK
			KKKKK
			KKKKK
			KKKKK
			KKKKK
		`},
		{"hint near the end", 5, 9, "linear-gradient(white, 8.9px, red)", `
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			zzzzz
		`},
		{"percentage hint near the end", 5, 9, "linear-gradient(white, 99%, red)", `
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			_____
			zzzzz
		`},
	})
}

func TestRadialGradientPixels(t *testing.T) {
	runGradientPixelCases(t, []gradientPixelCase{
		{"all stops inside out", 6, 6, "radial-gradient(red -30%, blue -10%)", `
			BBBBBB
			BBBBBB
			BBBBBB
			BBBBBB
			BBBBBB
			BBBBBB
		`},
		{"all stops outside", 6, 6, "radial-gradient(red 110%, blue 130%)", `
			RRRRRR
			RRRRRR
			RRRRRR
			RRRRRR
			RRRRRR
			RRRRRR
		`},
		{"ellipse", 10, 16, "radial-gradient(red 20%, blue 80%)", `
			BzzzzzzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzRRzzzz
			zzzzRRzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzzzzzzB
		`},
		{"stops at one position", 10, 16, "radial-gradient(red 50%, blue 50%)", `
			BzzzzzzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzRRzzzz
			zzzzRRzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzzzzzzB
		`},
		{"translucent stop", 10, 16, "radial-gradient(red 50%, rgba(255, 0, 0, 0.751) 50%)", `
			SzzzzzzzzS
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzRRzzzz
			zzzzRRzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			SzzzzzzzzS
		`},
		{"negative first stop", 10, 16, "radial-gradient(circle, red -1px, red 3px, blue 3px, blue 6px)", `
			BzzzzzzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzRRzzzB
			BzzzRRzzzB
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			zzzzBBzzzz
			zzzzzzzzzz
			zzzzzzzzzz
			BzzzzzzzzB
		`},
		{"repeating", 10, 16, "repeating-radial-gradient(circle, red 0, red 3px, blue 3px, blue 6px)", repeatingRadial},
		{"repeating from outside", 10, 16, "repeating-radial-gradient(circle, red 6px, red 9px, blue 9px, blue 12px)", repeatingRadial},
		{"repeating from a partial ring", 10, 16, "repeating-radial-gradient(circle, blue 3px, blue 6px, red 6px, red 9px)", repeatingRadial},
		{"repeating with negative first stop", 10, 16, "repeating-radial-gradient(circle, blue -3px, blue 0, red 0, red 3px)", repeatingRadial},
		{"hint in px", 6, 6, "radial-gradient(blue, 4px, white)", radialHint},
		{"hint in percent", 6, 6, "radial-gradient(blue, 99%, white)", radialHint},
	})
}

const repeatingRadial = `
	RzzzzzzzzR
	zzzzzzzzzz
	zzzzzzzzzz
	zzzzzzzzzz
	zzzzBBzzzz
	zzzzzzzzzz
	zzzzzzzzzz
	zBzzRRzzBz
	zBzzRRzzBz
	zzzzzzzzzz
	zzzzzzzzzz
	zzzzBBzzzz
	zzzzzzzzzz
	zzzzzzzzzz
	zzzzzzzzzz
	RzzzzzzzzR
`

const radialHint = `
	zzzzzz
	zzBBzz
	zBBBBz
	zBBBBz
	zzBBzz
	zzzzzz
`

func TestGradientSoftMaskOnlyForTranslucentStops(t *testing.T) {
	tests := []struct {
		name, css string
		want      bool
	}{
		{"opaque", "linear-gradient(red, blue)", false},
		{"translucent", "linear-gradient(red, rgba(0, 0, 255, 0.5))", true},
		{"transparent keyword", "linear-gradient(red, transparent)", true},
		{"radial translucent", "radial-gradient(red, rgba(0, 0, 255, 0.5))", true},
		{"radial opaque", "radial-gradient(red, blue)", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pdf := drawGradient(t, 5, 5, tc.css)
			if got := bytes.Contains(pdf, []byte("/SMask <<")) || bytes.Contains(pdf, []byte("/Luminosity")); got != tc.want {
				t.Errorf("soft mask present = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestGradientsInEmptyBoxesDrawNothing(t *testing.T) {
	for _, css := range []string{"linear-gradient(red, blue)", "radial-gradient(red, blue)", "repeating-linear-gradient(red, blue 1px)"} {
		t.Run(css, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			pdf := renderPDF(t, 5, 5, func(s *stream.Stream) { drawAt(s, gradientFor(t, css), 0, 0, 0, 0, "auto") })
			if bytes.Contains(pdf, []byte(" sh")) {
				t.Error("an empty box painted a shading")
			}
		})
	}
}
