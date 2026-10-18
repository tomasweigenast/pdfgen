package images

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func lengthPtr(l props.Length) *props.Length { return &l }

func percent(v float64) props.Length {
	return props.Length{Dimension: props.Dimension{Value: v, Unit: "%"}}
}

func floatsNear(got, want []float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-9*math.Max(1, math.Abs(want[i])) {
			return false
		}
	}
	return true
}

func TestProcessColorStops(t *testing.T) {
	half := percent(50)
	none := (*props.Length)(nil)
	px := func(v float64) *props.Length { return lengthPtr(props.Px(v)) }
	tests := []struct {
		name      string
		length    float64
		positions []*props.Length
		hints     []props.Length
		wantPos   []float64
		wantHints []float64
	}{
		{"defaults span the vector", 100, []*props.Length{none, none}, []props.Length{half}, []float64{0, 100}, []float64{1}},
		{"missing positions are spread evenly", 90, []*props.Length{none, none, none, none}, []props.Length{half, half, half},
			[]float64{0, 30, 60, 90}, []float64{1, 1, 1}},
		{"percentages resolve against the vector", 200, []*props.Length{lengthPtr(percent(10)), lengthPtr(percent(60))}, []props.Length{half},
			[]float64{20, 120}, []float64{math.Log(0.5) / math.Log(0.8)}},
		{"decreasing positions are raised", 100, []*props.Length{px(50), px(20), none, px(10)}, []props.Length{half, half, half},
			[]float64{50, 50, 50, 50}, []float64{1, 1, 1}},
		{"missing position between two set ones", 100, []*props.Length{px(10), none, px(30)}, []props.Length{half, half},
			[]float64{10, 20, 30}, []float64{1, 1}},
		{"hint at the quarter", 100, []*props.Length{none, none}, []props.Length{percent(25)}, []float64{0, 100}, []float64{0.5}},
		{"hint before the segment is a step", 100, []*props.Length{none, none}, []props.Length{props.Px(-5)}, []float64{0, 100}, []float64{0}},
		{"hint after the segment is a step", 100, []*props.Length{none, none}, []props.Length{percent(100)}, []float64{0, 100}, []float64{1 << 32}},
		{"hints inside and before narrow segments are linear", 100, []*props.Length{px(0), px(10), px(100)}, []props.Length{percent(5), percent(5)},
			[]float64{0, 10, 100}, []float64{1, 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos, hints := processColorStops(tc.length, tc.positions, tc.hints)
			if !floatsNear(pos, tc.wantPos) {
				t.Errorf("positions = %v, want %v", pos, tc.wantPos)
			}
			if !floatsNear(hints, tc.wantHints) {
				t.Errorf("hints = %v, want %v", hints, tc.wantHints)
			}
		})
	}
}

func TestNormalizeStopPositions(t *testing.T) {
	tests := []struct {
		name        string
		in          []float64
		first, last float64
		want        []float64
	}{
		{"offset range", []float64{10, 20, 30}, 10, 30, []float64{0, 0.5, 1}},
		{"already normalized", []float64{0, 0.25, 1}, 0, 1, []float64{0, 0.25, 1}},
		{"empty range collapses to zero", []float64{5, 5, 5}, 5, 5, []float64{0, 0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			first, last, got := normalizeStopPositions(tc.in)
			if first != tc.first || last != tc.last || !floatsNear(got, tc.want) {
				t.Errorf("got (%v, %v, %v), want (%v, %v, %v)", first, last, got, tc.first, tc.last, tc.want)
			}
		})
	}
}

func rgba(r, g, b, a float64) syntax.Color {
	return syntax.NewColor(syntax.SRGB, []float64{r, g, b}, a)
}

func TestGradientAverageColor(t *testing.T) {
	red, lime, blue := rgba(1, 0, 0, 1), rgba(0, 1, 0, 1), rgba(0, 0, 1, 1)
	tests := []struct {
		name      string
		colors    []syntax.Color
		positions []float64
		want      syntax.Color
	}{
		{"two opaque colors", []syntax.Color{red, lime}, []float64{0, 1}, rgba(0.5, 0.5, 0, 1)},
		{"weighted by segment length", []syntax.Color{red, red, blue, blue}, []float64{0, 3, 3, 4}, rgba(0.75, 0, 0.25, 1)},
		{"empty range counts stops evenly", []syntax.Color{red, lime}, []float64{2, 2}, rgba(0.5, 0.5, 0, 1)},
		{"transparent colors do not tint", []syntax.Color{red, rgba(0, 0, 1, 0)}, []float64{0, 1}, rgba(1, 0, 0, 0.5)},
		{"fully transparent", []syntax.Color{rgba(0, 0, 0, 0), rgba(0, 0, 0, 0)}, []float64{0, 1}, rgba(0, 0, 0, 0)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := gradientAverageColor(tc.colors, tc.positions)
			if got.Space != syntax.SRGB || !floatsNear(got.Coordinates, tc.want.Coordinates) || math.Abs(got.Alpha-tc.want.Alpha) > 1e-9 {
				t.Errorf("got %v alpha %v, want %v alpha %v", got.Coordinates, got.Alpha, tc.want.Coordinates, tc.want.Alpha)
			}
		})
	}
}
