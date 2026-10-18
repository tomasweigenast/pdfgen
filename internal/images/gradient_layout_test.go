package images

import (
	"math"
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

func TestRadialGradientLayout(t *testing.T) {
	sqrt2 := math.Sqrt2
	radial := func(cx, cy, r0, r1, scaleY float64) layoutWant {
		return layoutWant{kind: gradientRadial, scaleY: scaleY, points: []float64{cx, cy / scaleY, r0, cx, cy / scaleY, r1}}
	}
	circle := func(cx, cy, r0, r1 float64) layoutWant { return radial(cx, cy, r0, r1, 1) }
	solid := func(c syntax.Color) layoutWant {
		return layoutWant{kind: gradientSolid, positions: []float64{}, colors: []syntax.Color{c}}
	}
	with := func(w layoutWant, positions []float64, colors ...syntax.Color) layoutWant {
		if positions != nil {
			w.positions = positions
		}
		w.colors = colors
		return w
	}
	tests := []struct {
		css  string
		want layoutWant
	}{
		{"radial-gradient(blue)", solid(blue)},
		{"repeating-radial-gradient(blue)", solid(blue)},
		{"radial-gradient(100px, blue, lime)", circle(200, 150, 0, 100)},
		{"radial-gradient(100px at right 20px bottom 30px, lime, red)", with(circle(380, 270, 0, 100), nil, lime, red)},
		{"radial-gradient(0 0, blue, lime)", circle(200, 150, 0, 1e-7)},
		{"radial-gradient(1px 0, blue, lime)", radial(200, 150, 0, 1e7, 1e-14)},
		{"radial-gradient(0 1px, blue, lime)", radial(200, 150, 0, 1e-7, 1e14)},
		{"repeating-radial-gradient(100px 200px, blue, lime)", with(radial(200, 150, 0, 300, 2), []float64{0, 1, 1, 2, 2, 3}, blue, lime)},
		{"repeating-radial-gradient(42px, blue -100px, lime 100px)",
			with(circle(200, 150, 0, 300), []float64{-0.5, 0, 0, 1}, rgba(0, 0.5, 0.5, 1), lime, blue, lime)},
		{"radial-gradient(42px, blue -20px, lime -1px)", solid(lime)},
		{"radial-gradient(42px, blue -20px, lime 0)", solid(lime)},
		{"radial-gradient(42px, blue -20px, lime 20px)", with(circle(200, 150, 0, 20), nil, rgba(0, 0.5, 0.5, 1), lime)},

		{"radial-gradient(100px 120px, blue, lime)", radial(200, 150, 0, 100, 1.2)},
		{"radial-gradient(25% 40%, blue, lime)", radial(200, 150, 0, 100, 1.2)},

		{"radial-gradient(circle closest-side, blue, lime)", circle(200, 150, 0, 150)},
		{"radial-gradient(circle closest-side at 150px 50px, blue, lime)", circle(150, 50, 0, 50)},
		{"radial-gradient(circle closest-side at 45px 50px, blue, lime)", circle(45, 50, 0, 45)},
		{"radial-gradient(circle closest-side at 420px 50px, blue, lime)", circle(420, 50, 0, 20)},
		{"radial-gradient(circle closest-side at 420px 281px, blue, lime)", circle(420, 281, 0, 19)},

		{"radial-gradient(closest-side, blue 20%, lime)", with(radial(200, 150, 40, 200, 150.0/200), nil, blue, lime)},
		{"radial-gradient(closest-side at 300px 20%, blue, lime)", radial(300, 60, 0, 100, 60.0/100)},
		{"radial-gradient(closest-side at 10% 230px, blue, lime)", radial(40, 230, 0, 40, 70.0/40)},

		{"radial-gradient(circle farthest-side, blue, lime)", circle(200, 150, 0, 200)},
		{"radial-gradient(circle farthest-side at 150px 50px, blue, lime)", circle(150, 50, 0, 250)},
		{"radial-gradient(circle farthest-side at 45px 50px, blue, lime)", circle(45, 50, 0, 355)},
		{"radial-gradient(circle farthest-side at 420px 50px, blue, lime)", circle(420, 50, 0, 420)},
		{"radial-gradient(circle farthest-side at 220px 310px, blue, lime)", circle(220, 310, 0, 310)},

		{"radial-gradient(farthest-side, blue, lime)", radial(200, 150, 0, 200, 150.0/200)},
		{"radial-gradient(farthest-side at 300px 20%, blue, lime)", radial(300, 60, 0, 300, 240.0/300)},
		{"radial-gradient(farthest-side at 10% 230px, blue, lime)", radial(40, 230, 0, 360, 230.0/360)},

		{"radial-gradient(circle closest-corner, blue, lime)", circle(200, 150, 0, 250)},
		{"radial-gradient(circle closest-corner at 340px 80px, blue, lime)", circle(340, 80, 0, 100)},
		{"radial-gradient(circle closest-corner at 0 342px, blue, lime)", circle(0, 342, 0, 42)},

		{"radial-gradient(closest-corner, blue, lime)", radial(200, 150, 0, 200*sqrt2, 150.0/200)},
		{"radial-gradient(closest-corner at 450px 100px, blue, lime)", radial(450, 100, 0, 50*sqrt2, 100.0/50)},
		{"radial-gradient(closest-corner at 40px 210px, blue, lime)", radial(40, 210, 0, 40*sqrt2, 90.0/40)},

		{"radial-gradient(circle farthest-corner, blue, lime)", circle(200, 150, 0, 250)},
		{"radial-gradient(circle farthest-corner at 300px -100px, blue, lime)", circle(300, -100, 0, 500)},
		{"radial-gradient(circle farthest-corner at 400px 0, blue, lime)", circle(400, 0, 0, 500)},

		{"radial-gradient(farthest-corner, blue, lime)", radial(200, 150, 0, 200*sqrt2, 150.0/200)},
		{"radial-gradient(farthest-corner at 450px 100px, blue, lime)", radial(450, 100, 0, 450*sqrt2, 200.0/450)},
		{"radial-gradient(farthest-corner at 40px 210px, blue, lime)", radial(40, 210, 0, 360*sqrt2, 210.0/360)},
	}
	for _, tc := range tests {
		t.Run(tc.css, func(t *testing.T) { checkLayout(t, tc.css, tc.want) })
	}
}
