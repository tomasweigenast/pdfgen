package images

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// layoutWant is the expected layout of a gradient. A nil colors slice only
// checks the colors it lists, the way the layout may repeat the others.
type layoutWant struct {
	kind      gradientKind
	scaleY    float64
	points    []float64
	positions []float64
	colors    []syntax.Color
}

func checkLayout(t *testing.T, css string, want layoutWant) {
	t.Helper()
	got := gradientLayoutFor(t, css)
	if want.scaleY == 0 {
		want.scaleY = 1
	}
	if want.positions == nil && want.kind != gradientSolid {
		want.positions = []float64{0, 1}
	}
	if want.colors == nil {
		want.colors = []syntax.Color{blue, lime}
	}
	if !approx(got.scaleY, want.scaleY) {
		t.Errorf("scaleY = %v, want %v", got.scaleY, want.scaleY)
	}
	if got.kind != want.kind {
		t.Errorf("kind = %v, want %v", got.kind, want.kind)
	}
	if !approxSlice(got.points, want.points) {
		t.Errorf("points = %v, want %v", got.points, want.points)
	}
	if !approxSlice(got.positions, want.positions) {
		t.Errorf("positions = %v, want %v", got.positions, want.positions)
	}
	for i, c := range want.colors {
		if i >= len(got.colors) {
			t.Errorf("only %d colors, want at least %d", len(got.colors), len(want.colors))
			break
		}
		if !colorsNear(got.colors[i], c) {
			t.Errorf("colors[%d] = %v alpha %v, want %v", i, got.colors[i].Coordinates, got.colors[i].Alpha, c.Coordinates)
		}
	}
}

func TestLinearGradientLayout(t *testing.T) {
	line := func(x0, y0, x1, y1 float64) layoutWant {
		return layoutWant{kind: gradientLinear, points: []float64{x0, y0, x1, y1}}
	}
	solid := func(c syntax.Color) layoutWant {
		return layoutWant{kind: gradientSolid, positions: []float64{}, colors: []syntax.Color{c}}
	}
	tests := []struct {
		css  string
		want layoutWant
	}{
		{"linear-gradient(blue)", solid(blue)},
		{"repeating-linear-gradient(blue)", solid(blue)},
		{"linear-gradient(blue, lime)", line(200, 0, 200, 300)},
		{"repeating-linear-gradient(blue, lime)", line(200, 0, 200, 300)},
		{"repeating-linear-gradient(blue, lime 100px)", layoutWant{
			kind: gradientLinear, points: []float64{200, 0, 200, 300}, positions: []float64{0, 1, 1, 2, 2, 3},
		}},
		{"linear-gradient(to bottom, blue, lime)", line(200, 0, 200, 300)},
		{"linear-gradient(to top, blue, lime)", line(200, 300, 200, 0)},
		{"linear-gradient(to right, blue, lime)", line(0, 150, 400, 150)},
		{"linear-gradient(to left, blue, lime)", line(400, 150, 0, 150)},
		{"linear-gradient(to top left, blue, lime)", line(344, 342, 56, -42)},
		{"linear-gradient(to top right, blue, lime)", line(56, 342, 344, -42)},
		{"linear-gradient(to bottom left, blue, lime)", line(344, -42, 56, 342)},
		{"linear-gradient(to bottom right, blue, lime)", line(56, -42, 344, 342)},
		{"linear-gradient(270deg, blue, lime)", line(400, 150, 0, 150)},
		{"linear-gradient(.75turn, blue, lime)", line(400, 150, 0, 150)},
		{"linear-gradient(45deg, blue, lime)", line(25, 325, 375, -25)},
		{".125", line(25, 325, 375, -25)},
		{".375", line(25, -25, 375, 325)},
		{".625", line(375, -25, 25, 325)},
		{".875", line(375, 325, 25, -25)},
		{"linear-gradient(blue 2em, lime 20%)", line(200, 32, 200, 60)},
		{"linear-gradient(blue 100px, red, blue, red 160px, lime)", layoutWant{
			kind: gradientLinear, points: []float64{200, 100, 200, 300},
			colors: []syntax.Color{blue, red, blue, red, lime}, positions: []float64{0, .1, .2, .3, 1},
		}},
		{"linear-gradient(blue -100px, blue 0, red -12px, lime 50%)", layoutWant{
			kind: gradientLinear, points: []float64{200, -100, 200, 150},
			colors: []syntax.Color{blue, blue, red, lime}, positions: []float64{0, .4, .4, 1},
		}},
		{"linear-gradient(blue, blue, red, lime -7px)", layoutWant{
			kind: gradientLinear, points: []float64{200, -1, 200, 1},
			colors: []syntax.Color{blue, blue, blue, red, lime, lime}, positions: []float64{0, .5, .5, .5, .5, 1},
		}},
		{"repeating-linear-gradient(blue, blue, lime, lime -7px)", solid(rgba(0, .5, .5, 1))},
	}
	for _, tc := range tests {
		css := tc.css
		if css[0] == '.' {
			css = "linear-gradient(" + css + "turn, blue, lime)"
		}
		t.Run(css, func(t *testing.T) { checkLayout(t, css, tc.want) })
	}
}
