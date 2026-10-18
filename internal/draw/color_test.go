package draw

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func TestGetColorResolvesCurrentColor(t *testing.T) {
	got := getColor(computed.New(nil), props.PropBorderTopColor)
	if got.IsCurrentColor() || got.Alpha != 1 || got.Coordinates[0] != 0 {
		t.Errorf("border-top-color = %#v, want the opaque black of color", got)
	}
}

func TestDarkenAndLightenKeepHueAndAlpha(t *testing.T) {
	c := syntax.NewColor(syntax.SRGB, []float64{0.5, 0, 0}, 0.5)
	dark, light := darken(c), lighten(c)
	if dark.Alpha != 0.5 || light.Alpha != 0.5 {
		t.Errorf("alpha changed: darken %v, lighten %v", dark.Alpha, light.Alpha)
	}
	if got := dark.Coordinates[0]; math.Abs(got-0.5/1.5) > 1e-9 || dark.Coordinates[1] <= 0 {
		t.Errorf("darken = %v", dark.Coordinates)
	}
	if got := light.Coordinates[0]; math.Abs(got-(1-0.5/1.5)) > 1e-9 || light.Coordinates[1] != 0 {
		t.Errorf("lighten = %v", light.Coordinates)
	}
}

func TestStyledColorByBorderSide(t *testing.T) {
	c := syntax.NewColor(syntax.SRGB, []float64{0.5, 0, 0}, 1)
	tests := []struct {
		style, side              string
		outerLighter, innerLight bool
	}{
		{"inset", "top", false, false},
		{"inset", "bottom", true, true},
		{"outset", "left", true, true},
		{"outset", "right", false, false},
		{"ridge", "top", false, true},
		{"ridge", "right", true, false},
		{"groove", "left", true, false},
		{"groove", "bottom", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.style+" "+tt.side, func(t *testing.T) {
			outer, inner := styledColor(props.Keyword(tt.style), c, tt.side)
			if got := outer.Coordinates[0] > 0.5; got != tt.outerLighter {
				t.Errorf("outer lighter = %v, want %v (%v)", got, tt.outerLighter, outer.Coordinates)
			}
			if got := inner.Coordinates[0] > 0.5; got != tt.innerLight {
				t.Errorf("inner lighter = %v, want %v (%v)", got, tt.innerLight, inner.Coordinates)
			}
		})
	}

	outer, inner := styledColor("solid", c, "top")
	if outer.Coordinates[0] != 0.5 || inner.Coordinates[0] != 0.5 {
		t.Errorf("solid border changed the color: %v, %v", outer.Coordinates, inner.Coordinates)
	}
}

func TestShadedColorChannelsAreNeverNegative(t *testing.T) {
	for _, c := range []syntax.Color{
		syntax.NewColor(syntax.SRGB, []float64{0, 0, 1}, 1),
		syntax.NewColor(syntax.SRGB, []float64{1, 0, 0}, 1),
		syntax.NewColor(syntax.SRGB, []float64{0, 1, 0}, 1),
	} {
		for name, shade := range map[string]func(syntax.Color) syntax.Color{"darken": darken, "lighten": lighten} {
			for _, v := range shade(c).Coordinates {
				if v < 0 {
					t.Errorf("%s(%v) has a negative channel: %v", name, c.Coordinates, shade(c).Coordinates)
					break
				}
			}
		}
	}
}
