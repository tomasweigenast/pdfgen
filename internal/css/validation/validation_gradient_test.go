package validation

import (
	"math"
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

var (
	blue  = rgb(0, 0, 1, 1)
	lime  = rgb(0, 1, 0, 1)
	redC  = rgb(1, 0, 0, 1)
	three = []props.Color{blue, lime, redC}
)

func lp(l props.Length) *props.Length { return &l }

func gradientOf(t *testing.T, css string) props.Image {
	t.Helper()
	list, ok := getValue(t, css, "").(props.Images)
	if !ok || len(list) != 1 {
		t.Fatalf("%s: got %#v, want one image", css, list)
	}
	return list[0]
}

func checkStops(t *testing.T, css string, g props.Gradient, colors []props.Color, positions []*props.Length) {
	t.Helper()
	if len(g.ColorStops) != len(colors) {
		t.Errorf("%s: %d color stops, want %d", css, len(g.ColorStops), len(colors))
		return
	}
	for i, stop := range g.ColorStops {
		if !reflect.DeepEqual(stop.Color, colors[i]) {
			t.Errorf("%s: color %d = %#v, want %#v", css, i, stop.Color, colors[i])
		}
		if !reflect.DeepEqual(stop.Position, positions[i]) {
			t.Errorf("%s: stop position %d = %#v, want %#v", css, i, stop.Position, positions[i])
		}
	}
}

func TestLinearGradient(t *testing.T) {
	stops := "blue 1%, lime,red 2em "
	positions := []*props.Length{lp(pct(1)), nil, lp(dim(2, "em"))}
	angle := func(a float64) props.LinearGradient {
		return props.LinearGradient{DirectionType: "angle", Angle: a}
	}
	corner := func(c string) props.LinearGradient {
		return props.LinearGradient{DirectionType: "corner", Corner: props.Keyword(c)}
	}
	tests := []struct {
		rule      string
		direction props.LinearGradient
		colors    []props.Color
		positions []*props.Length
	}{
		{"blue", angle(math.Pi), []props.Color{blue}, []*props.Length{nil}},
		{"red", angle(math.Pi), []props.Color{redC}, []*props.Length{nil}},
		{stops, angle(math.Pi), three, positions},
		{"18deg, blue", angle(math.Pi / 10), []props.Color{blue}, []*props.Length{nil}},
		{"4rad, blue", angle(4), []props.Color{blue}, []*props.Length{nil}},
		{".25turn, blue", angle(math.Pi / 2), []props.Color{blue}, []*props.Length{nil}},
		{"100grad, blue", angle(math.Pi / 2), []props.Color{blue}, []*props.Length{nil}},
		{"12rad, " + stops, angle(12), three, positions},
		{"to top, blue", angle(0), []props.Color{blue}, []*props.Length{nil}},
		{"to right, blue", angle(math.Pi / 2), []props.Color{blue}, []*props.Length{nil}},
		{"to bottom, blue", angle(math.Pi), []props.Color{blue}, []*props.Length{nil}},
		{"to left, blue", angle(math.Pi * 3 / 2), []props.Color{blue}, []*props.Length{nil}},
		{"to right, " + stops, angle(math.Pi / 2), three, positions},
		{"to top left, blue", corner("top_left"), []props.Color{blue}, []*props.Length{nil}},
		{"to left top, blue", corner("top_left"), []props.Color{blue}, []*props.Length{nil}},
		{"to top right, blue", corner("top_right"), []props.Color{blue}, []*props.Length{nil}},
		{"to right top, blue", corner("top_right"), []props.Color{blue}, []*props.Length{nil}},
		{"to bottom left, blue", corner("bottom_left"), []props.Color{blue}, []*props.Length{nil}},
		{"to left bottom, blue", corner("bottom_left"), []props.Color{blue}, []*props.Length{nil}},
		{"to bottom right, blue", corner("bottom_right"), []props.Color{blue}, []*props.Length{nil}},
		{"to right bottom, blue", corner("bottom_right"), []props.Color{blue}, []*props.Length{nil}},
	}
	for _, tt := range tests {
		for _, repeating := range []bool{false, true} {
			prefix := ""
			if repeating {
				prefix = "repeating-"
			}
			css := "background-image: " + prefix + "linear-gradient(" + tt.rule + ")"
			img := gradientOf(t, css)
			g := img.Linear
			if img.Type != "linear-gradient" || g == nil {
				t.Errorf("%s: got %#v", css, img)
				continue
			}
			if g.Repeating != repeating || g.DirectionType != tt.direction.DirectionType ||
				g.Corner != tt.direction.Corner || !approx(g.Angle, tt.direction.Angle) {
				t.Errorf("%s: direction = %v %v %v, want %v %v %v", css, g.DirectionType, g.Angle, g.Corner,
					tt.direction.DirectionType, tt.direction.Angle, tt.direction.Corner)
			}
			checkStops(t, css, g.Gradient, tt.colors, tt.positions)
		}
	}
}

func TestLinearGradientInvalid(t *testing.T) {
	for _, rule := range []string{
		" ", "1% blue", "blue 10deg", "blue 4", "soylent-green 4px", "red 4px 2px", "18deg",
		"10arc-minutes, blue", "10px, blue", "to 90deg, blue", "to the top, blue", "to up, blue",
		"into top, blue", "top, blue", "to bottom up, blue", "bottom left, blue",
	} {
		assertInvalid(t, "background-image: linear-gradient("+rule+")")
		assertInvalid(t, "background-image: repeating-linear-gradient("+rule+")")
	}
}

func TestRadialGradient(t *testing.T) {
	stops := "blue 1%, lime,red 2em "
	positions := []*props.Length{lp(pct(1)), nil, lp(dim(2, "em"))}
	center := props.Position{OriginX: "left", X: pct(50), OriginY: "top", Y: pct(50)}
	one, nilPos := []props.Color{blue}, []*props.Length{nil}
	type radial struct {
		shape    string
		sizeType string
		sizeKw   string
		size     [2]props.Length
		center   props.Position
		colors   []props.Color
		stopsAt  []*props.Length
	}
	def := radial{shape: "ellipse", sizeType: "keyword", sizeKw: "farthest-corner", center: center, colors: one, stopsAt: nilPos}
	with := func(f func(r *radial)) radial { r := def; f(&r); return r }
	explicit := func(r *radial, a, b props.Length) {
		r.sizeType, r.sizeKw, r.size = "explicit", "", [2]props.Length{a, b}
	}
	tests := []struct {
		rule string
		want radial
	}{
		{"blue", def},
		{"red", with(func(r *radial) { r.colors = []props.Color{redC} })},
		{stops, with(func(r *radial) { r.colors, r.stopsAt = three, positions })},
		{"circle, blue", with(func(r *radial) { r.shape = "circle" })},
		{"ellipse, blue", def},
		{"ellipse closest-corner, blue", with(func(r *radial) { r.sizeKw = "closest-corner" })},
		{"circle closest-side, blue", with(func(r *radial) { r.shape, r.sizeKw = "circle", "closest-side" })},
		{"farthest-corner circle, blue", with(func(r *radial) { r.shape = "circle" })},
		{"farthest-side, blue", with(func(r *radial) { r.sizeKw = "farthest-side" })},
		{"5ch, blue", with(func(r *radial) { r.shape = "circle"; explicit(r, dim(5, "ch"), dim(5, "ch")) })},
		{"5ch circle, blue", with(func(r *radial) { r.shape = "circle"; explicit(r, dim(5, "ch"), dim(5, "ch")) })},
		{"circle 5ch, blue", with(func(r *radial) { r.shape = "circle"; explicit(r, dim(5, "ch"), dim(5, "ch")) })},
		{"10px 50px, blue", with(func(r *radial) { explicit(r, px(10), px(50)) })},
		{"10px 50px ellipse, blue", with(func(r *radial) { explicit(r, px(10), px(50)) })},
		{"ellipse 10px 50px, blue", with(func(r *radial) { explicit(r, px(10), px(50)) })},
		{"at top 10% right, blue", with(func(r *radial) {
			r.center = props.Position{OriginX: "right", X: pct(0), OriginY: "top", Y: pct(10)}
		})},
		{"circle at bottom, blue", with(func(r *radial) {
			r.shape = "circle"
			r.center = props.Position{OriginX: "left", X: pct(50), OriginY: "top", Y: pct(100)}
		})},
		{"circle at 10px, blue", with(func(r *radial) {
			r.shape = "circle"
			r.center = props.Position{OriginX: "left", X: px(10), OriginY: "top", Y: pct(50)}
		})},
		{"closest-side circle at right 5em, blue", with(func(r *radial) {
			r.shape, r.sizeKw = "circle", "closest-side"
			r.center = props.Position{OriginX: "left", X: pct(100), OriginY: "top", Y: dim(5, "em")}
		})},
	}
	for _, tt := range tests {
		for _, repeating := range []bool{false, true} {
			prefix := ""
			if repeating {
				prefix = "repeating-"
			}
			css := "background-image: " + prefix + "radial-gradient(" + tt.rule + ")"
			img := gradientOf(t, css)
			g := img.Radial
			if img.Type != "radial-gradient" || g == nil {
				t.Errorf("%s: got %#v", css, img)
				continue
			}
			if g.Repeating != repeating || string(g.Shape) != tt.want.shape || string(g.SizeType) != tt.want.sizeType ||
				string(g.SizeKeyword) != tt.want.sizeKw || g.SizeExplicit != tt.want.size || g.Center != tt.want.center {
				t.Errorf("%s: got shape %v size %v %v %v center %v", css, g.Shape, g.SizeType, g.SizeKeyword, g.SizeExplicit, g.Center)
			}
			checkStops(t, css, g.Gradient, tt.want.colors, tt.want.stopsAt)
		}
	}
}

func TestRadialGradientInvalid(t *testing.T) {
	for _, rule := range []string{
		" ", "1% blue", "blue 10deg", "blue 4", "soylent-green 4px", "red 4px 2px", "circle",
		"square, blue", "closest-triangle, blue", "center, blue", "ellipse 5ch", "5ch ellipse",
		"circle 10px 50px, blue", "10px 50px circle, blue", "10%, blue", "10% circle, blue",
		"circle 10%, blue", "at appex, blue",
	} {
		assertInvalid(t, "background-image: radial-gradient("+rule+")")
		assertInvalid(t, "background-image: repeating-radial-gradient("+rule+")")
	}
}
