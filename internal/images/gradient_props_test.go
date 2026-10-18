package images

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func TestGradientFromPropsRejectsOtherImages(t *testing.T) {
	for _, img := range []props.Image{
		{Type: "none"},
		{Type: "url", URL: "https://example.org/image.png"},
		{Type: "linear-gradient"},
		{Type: "radial-gradient"},
	} {
		if g, ok := GradientFromProps(img, syntax.Color{}); ok {
			t.Errorf("GradientFromProps(%v) = %v, true", img, g)
		}
	}
}

func TestGradientsHaveNoIntrinsicSize(t *testing.T) {
	for _, css := range []string{"linear-gradient(red, blue)", "radial-gradient(red, blue)"} {
		w, h, ratio := gradientFor(t, css).IntrinsicSize(1, 16)
		if !w.IsAuto() || !h.IsAuto() || !ratio.IsAuto() {
			t.Errorf("%s: IntrinsicSize = %v, %v, %v, want all auto", css, w, h, ratio)
		}
	}
}

func TestLinearGradientBuiltFromStops(t *testing.T) {
	stop := func(c syntax.Color, px float64) props.ColorStop {
		p := props.Px(px)
		return props.ColorStop{Color: props.Color{Color: c}, Position: &p}
	}
	half := props.Length{Dimension: props.Dimension{Value: 50, Unit: "%"}}
	g := NewLinearGradient(&props.LinearGradient{
		Gradient:      props.Gradient{ColorStops: []props.ColorStop{stop(red, 0), stop(blue, 20)}, ColorHints: []props.Length{half}},
		DirectionType: "angle",
		Angle:         math.Pi / 2,
	}, syntax.Color{})
	got := g.layout(100, 50)
	if got.kind != gradientLinear || !approxSlice(got.points, []float64{0, 25, 20, 25}) {
		t.Errorf("layout = %v %v, want the first 20px of the box width", got.kind, got.points)
	}
}

func TestCurrentColorStopsTakeTheGivenColor(t *testing.T) {
	g := NewLinearGradient(&props.LinearGradient{
		Gradient:      props.Gradient{ColorStops: []props.ColorStop{{Color: props.Color{Color: syntax.Color{Space: syntax.CurrentColor, Alpha: 1}}}}},
		DirectionType: "angle",
	}, lime)
	got := g.layout(10, 10)
	if got.kind != gradientSolid || !colorsNear(got.colors[0], lime) {
		t.Errorf("layout = %v %v, want a solid lime", got.kind, got.colors)
	}
}

func TestRepeatingGradientTooFineToUnrollIsAveraged(t *testing.T) {
	got := gradientLayoutFor(t, "repeating-linear-gradient(to right, blue 0, lime 0.001px)")
	if got.kind != gradientSolid {
		t.Fatalf("kind = %v, want solid", got.kind)
	}
	if !colorsNear(got.colors[0], rgba(0, 0.5, 0.5, 1)) {
		t.Errorf("color = %v", got.colors[0].Coordinates)
	}
}
