package tokens

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func TestMathTokenHelpers(t *testing.T) {
	for _, tt := range []struct {
		value float64
		want  string
	}{
		{math.Inf(1), "inf"}, {math.Inf(-1), "-inf"}, {math.NaN(), "nan"}, {1.25, "1.25"},
	} {
		if got := formatNumber(tt.value); got != tt.want {
			t.Errorf("formatNumber(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
	for _, tt := range []struct {
		css, unit string
		value     float64
		ok        bool
	}{
		{"0", "", 0, true}, {"2px", "px", 2, true}, {"2in", "in", 192, true},
		{"2em", "em", 20, true}, {"2unknown", "unknown", 0, false},
	} {
		got, ok := toPixels(parseOne(t, tt.css), unitStyle{"em": 10}, "width")
		if ok != tt.ok || (ok && got != tt.value) {
			t.Errorf("toPixels(%q) = %v, %v; want %v, %v", tt.css, got, ok, tt.value, tt.ok)
		}
		if got := unitOf(parseOne(t, tt.css)); got != tt.unit {
			t.Errorf("unitOf(%q) = %q, want %q", tt.css, got, tt.unit)
		}
	}
	if !math.IsNaN(valueOf(parseOne(t, "keyword"))) {
		t.Error("valueOf(non-numeric) should be NaN")
	}
}

func TestResolveMathConstantsAndRejectedInputs(t *testing.T) {
	for _, tt := range []struct {
		css  string
		want float64
		ok   bool
	}{
		{"calc(e)", math.E, true}, {"calc(pi)", math.Pi, true},
		{"calc(infinity)", math.Inf(1), true}, {"calc(-infinity)", math.Inf(-1), true},
		{"calc(nan)", math.NaN(), true}, {"calc(unknown)", 0, false},
		{"sin(90deg)", 1, true}, {"cos(0)", 1, true}, {"atan2(1, 1)", math.Pi / 4, true},
		{"round(invalid, 2)", 0, false}, {"asin(2)", 0, false},
	} {
		got, err := ResolveMath(parseOne(t, tt.css), nil, "", nil)
		if err != nil {
			t.Errorf("ResolveMath(%q): %v", tt.css, err)
			continue
		}
		if !tt.ok {
			if got != nil {
				t.Errorf("ResolveMath(%q) = %s, want unresolved", tt.css, got.Serialize())
			}
			continue
		}
		n := valueOf(got)
		if !math.IsNaN(tt.want) && math.Abs(n-tt.want) > 1e-9 {
			t.Errorf("ResolveMath(%q) = %v, want %v", tt.css, n, tt.want)
		}
	}
	if resolveCalcValue(nil) != nil || resolveCalcValue([]syntax.Node{parseOne(t, "1"), parseOne(t, "2")}) != nil {
		t.Error("resolveCalcValue accepted an empty or multi-token expression")
	}
	for _, css := range []string{"nearest", "up", "down", "to-zero", "other"} {
		got := isRoundingStrategy(parseOne(t, css))
		if got != (css != "other") {
			t.Errorf("isRoundingStrategy(%q) = %v", css, got)
		}
	}
}

func TestResolvePercentageMathAndIncompatibleUnits(t *testing.T) {
	ref := 200.0
	for _, tt := range []struct {
		css  string
		ref  *float64
		want float64
		err  error
	}{
		{"min(10%, 30px)", &ref, 20, nil},
		{"clamp(10%, 30%, 20%)", &ref, 40, nil},
		{"min(10%, 30px)", nil, 0, ErrPercentageInMath},
		{"max(1em, 10px)", nil, 0, ErrRelativeLengthInMath},
		{"pow(2px, 3)", nil, 0, nil},
	} {
		got, err := ResolveMath(parseOne(t, tt.css), nil, "width", tt.ref)
		if err != tt.err {
			t.Errorf("ResolveMath(%q) error = %v, want %v", tt.css, err, tt.err)
			continue
		}
		if tt.err == nil && tt.css != "pow(2px, 3)" {
			if value := valueOf(got); math.Abs(value-tt.want) > 1e-9 {
				t.Errorf("ResolveMath(%q) = %v, want %v", tt.css, value, tt.want)
			}
		}
		if tt.css == "pow(2px, 3)" && got != nil {
			t.Errorf("ResolveMath(%q) = %v, want unresolved", tt.css, got.Serialize())
		}
	}
}

func TestInvalidValuesErrorText(t *testing.T) {
	for _, tt := range []struct {
		reason string
		want   string
	}{{"", "invalid value"}, {"bad unit", "bad unit"}} {
		if got := (&InvalidValues{Reason: tt.reason}).Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}
