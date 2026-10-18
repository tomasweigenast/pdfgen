package units

import (
	"math"
	"testing"
)

func TestMaybeFloat(t *testing.T) {
	if (MaybeFloat{}).IsAuto() {
		t.Error("zero value must be the number 0, not auto")
	}
	if !Auto.IsAuto() {
		t.Error("Auto.IsAuto() = false")
	}
	if got := F(3).Or(9); got != 3 {
		t.Errorf("F(3).Or(9) = %v, want 3", got)
	}
	if got := F(0).Or(9); got != 0 {
		t.Errorf("F(0).Or(9) = %v, want 0", got)
	}
	if got := Auto.Or(9); got != 9 {
		t.Errorf("Auto.Or(9) = %v, want 9", got)
	}
	if F(0) == Auto {
		t.Error("F(0) must differ from Auto")
	}
}

func TestAbsoluteToPixels(t *testing.T) {
	tests := []struct {
		value float64
		unit  string
		want  float64
	}{
		{1, "px", 1},
		{3, "pt", 4},
		{1, "pc", 16},
		{1, "in", 96},
		{2.54, "cm", 96},
		{25.4, "mm", 96},
		{4, "Q", 96 / 25.4},
		{1, "IN", 96},
	}
	for _, tt := range tests {
		got, ok := AbsoluteToPixels(tt.value, tt.unit)
		if !ok || math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("AbsoluteToPixels(%v, %q) = %v, %v; want %v", tt.value, tt.unit, got, ok, tt.want)
		}
	}
	if _, ok := AbsoluteToPixels(1, "em"); ok {
		t.Error("em must not convert as an absolute unit")
	}
}

func TestToRadians(t *testing.T) {
	tests := []struct {
		value float64
		unit  string
		want  float64
	}{
		{1, "rad", 1},
		{180, "deg", math.Pi},
		{200, "grad", math.Pi},
		{0.5, "turn", math.Pi},
	}
	for _, tt := range tests {
		got, ok := ToRadians(tt.value, tt.unit)
		if !ok || math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("ToRadians(%v, %q) = %v, %v; want %v", tt.value, tt.unit, got, ok, tt.want)
		}
	}
	if _, ok := ToRadians(1, "px"); ok {
		t.Error("px must not convert as an angle")
	}
}

func TestUnitClassification(t *testing.T) {
	tests := []struct {
		unit                                    string
		absolute, font, viewport, length, angle bool
	}{
		{"px", true, false, false, true, false},
		{"mm", true, false, false, true, false},
		{"em", false, true, false, true, false},
		{"rlh", false, true, false, true, false},
		{"vmin", false, false, true, true, false},
		{"PVH", false, false, true, true, false},
		{"deg", false, false, false, false, true},
		{"dppx", false, false, false, false, false},
		{"foo", false, false, false, false, false},
	}
	for _, tt := range tests {
		if got := IsAbsolute(tt.unit); got != tt.absolute {
			t.Errorf("IsAbsolute(%q) = %v", tt.unit, got)
		}
		if got := IsFontRelative(tt.unit); got != tt.font {
			t.Errorf("IsFontRelative(%q) = %v", tt.unit, got)
		}
		if got := IsViewportRelative(tt.unit); got != tt.viewport {
			t.Errorf("IsViewportRelative(%q) = %v", tt.unit, got)
		}
		if got := IsLength(tt.unit); got != tt.length {
			t.Errorf("IsLength(%q) = %v", tt.unit, got)
		}
		if got := IsAngle(tt.unit); got != tt.angle {
			t.Errorf("IsAngle(%q) = %v", tt.unit, got)
		}
	}
}

func TestResolutionToDppx(t *testing.T) {
	if got := ResolutionToDppx["dpi"]; math.Abs(got-1.0/96) > 1e-12 {
		t.Errorf("dpi = %v", got)
	}
	if got := ResolutionToDppx["dpcm"]; math.Abs(got-2.54/96) > 1e-12 {
		t.Errorf("dpcm = %v", got)
	}
}

func TestAbsoluteLengthUnitsRoundEveryDivision(t *testing.T) {
	// The operands are variables so that each division runs in float64
	// instead of being folded into one exact constant.
	inch, mm, cm, pt, four := 96.0, 25.4, 2.54, 0.75, 4.0
	for unit, want := range map[string]float64{
		"pt": 1 / pt,
		"cm": inch / cm,
		"mm": inch / mm,
		"q":  inch / mm / four,
	} {
		if got := LengthsToPixels[unit]; got != want {
			t.Errorf("1%s = %.17g px, want %.17g", unit, got, want)
		}
	}
}
