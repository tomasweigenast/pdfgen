package tokens

import (
	"errors"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func parseOne(t *testing.T, css string) syntax.Node {
	t.Helper()
	return syntax.ParseOneComponentValue(css, true)
}

func TestGetLength(t *testing.T) {
	tests := []struct {
		css        string
		negative   bool
		percentage bool
		want       props.Length
		ok         bool
	}{
		{"3EM", true, false, props.Length{Dimension: props.Dimension{Value: 3, Unit: "em"}}, true},
		{"0", true, false, props.Length{Dimension: props.Dimension{}}, true},
		{"1", true, false, props.Length{}, false},
		{"-1px", false, false, props.Length{}, false},
		{"-1px", true, false, props.Px(-1), true},
		{"20%", true, false, props.Length{}, false},
		{"20%", true, true, props.Length{Dimension: props.Dimension{Value: 20, Unit: "%"}}, true},
		{"calc(1px + 2px)", true, false, props.Px(3), true},
		{"calc(1in - 48px)", true, false, props.Px(48), true},
		{"calc(-1px)", false, false, props.Px(0), true},
		{"min(10px, 5px)", true, false, props.Px(5), true},
		{"max(2px, 1in)", true, false, props.Px(96), true},
		{"clamp(1px, 50px, 10px)", true, false, props.Px(10), true},
		{"calc(2 * 3px)", true, false, props.Px(6), true},
		{"calc(6px / 2)", true, false, props.Px(3), true},
		{"calc(10% + 1px)", true, false, props.Length{}, false},
		{"scale(2)", true, false, props.Length{}, false},
	}
	for _, tt := range tests {
		got, ok := GetLength(parseOne(t, tt.css), tt.negative, tt.percentage)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("GetLength(%q) = %v, %v; want %v, %v", tt.css, got, ok, tt.want, tt.ok)
		}
	}
}

func TestGetLengthKeepsRelativeMathUnresolved(t *testing.T) {
	for _, css := range []string{"calc(1em + 2px)", "min(1em, 2px)", "calc(10% + 1em)"} {
		got, ok := GetLength(parseOne(t, css), true, true)
		if !ok || got.Calc == nil {
			t.Errorf("GetLength(%q) = %v, %v; want an unresolved calc", css, got, ok)
		}
	}
	got, ok := GetLength(parseOne(t, "calc(10% + 1px)"), true, true)
	if !ok || got.Calc == nil {
		t.Errorf("a percentage mixed with pixels should stay unresolved when allowed, got %v, %v", got, ok)
	}
}

func TestGetNumber(t *testing.T) {
	tests := []struct {
		css      string
		negative bool
		integer  bool
		want     float64
		ok       bool
	}{
		{"3", true, false, 3, true},
		{"-3", false, false, 0, false},
		{"2.5", true, true, 0, false},
		{"2", true, true, 2, true},
		{"calc(1 + 2)", true, true, 3, true},
		{"calc(1 - 2)", false, false, 0, true},
		{"calc(2.5)", true, true, 2, true},
		{"calc(3.5)", true, true, 4, true},
		{"calc(1px)", true, false, 0, false},
		{"pi", true, false, 0, false},
		{"calc(pi * 2)", true, false, 2 * math.Pi, true},
		{"round(up, 1.2, 1)", true, false, 2, true},
		{"mod(7, 4)", true, false, 3, true},
		{"mod(-7, 4)", true, false, 1, true},
		{"rem(-7, 4)", true, false, 5, true},
		{"abs(-2)", true, false, 2, true},
		{"sign(-2)", true, false, -1, true},
		{"sqrt(9)", true, false, 3, true},
		{"pow(2, 3)", true, false, 8, true},
		{"hypot(3, 4)", true, false, 5, true},
		{"log(8, 2)", true, false, 3, true},
	}
	for _, tt := range tests {
		got, ok := GetNumber(parseOne(t, tt.css), tt.negative, tt.integer)
		if ok != tt.ok || (ok && math.Abs(got.Value-tt.want) > 1e-9) {
			t.Errorf("GetNumber(%q) = %v, %v; want %v, %v", tt.css, got.Value, ok, tt.want, tt.ok)
		}
	}
}

func TestGetAngle(t *testing.T) {
	tests := []struct {
		css  string
		want float64
		ok   bool
	}{
		{"90deg", math.Pi / 2, true},
		{"0", 0, true},
		{"1turn", 2 * math.Pi, true},
		{"200grad", math.Pi, true},
		{"calc(45deg * 2)", math.Pi / 2, true},
		{"1", 0, false},
		{"1px", 0, false},
	}
	for _, tt := range tests {
		got, ok := GetAngle(parseOne(t, tt.css))
		if ok != tt.ok || (ok && math.Abs(got-tt.want) > 1e-9) {
			t.Errorf("GetAngle(%q) = %v, %v; want %v, %v", tt.css, got, ok, tt.want, tt.ok)
		}
	}
}

func TestResolveMathErrors(t *testing.T) {
	tests := []struct {
		css  string
		want error
	}{
		{"calc(1em + 1px)", ErrRelativeLengthInMath},
		{"calc(10% + 1px)", ErrPercentageInMath},
		{"calc(10% + 1em)", ErrPercentageInMath},
		{"calc(1em + 10%)", ErrPercentageInMath},
		{"min(1px, 10%)", ErrPercentageInMath},
		{"calc(1px)", nil},
	}
	for _, tt := range tests {
		_, err := ResolveMath(parseOne(t, tt.css), nil, "", nil)
		if !errors.Is(err, tt.want) {
			t.Errorf("ResolveMath(%q) error = %v, want %v", tt.css, err, tt.want)
		}
	}
}

func TestResolveMathWithReferenceLength(t *testing.T) {
	ref := 200.0
	got, err := ResolveMath(parseOne(t, "calc(10% + 5px)"), nil, "", &ref)
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := got.(*syntax.DimensionToken); !ok || d.LowerUnit != "px" || d.Value != 25 {
		t.Errorf("got %v, want 25px", got.Serialize())
	}
}

func TestCheckVar(t *testing.T) {
	tests := []struct {
		css  string
		want bool
	}{
		{"var(--a)", true},
		{"calc(var(--a) + 1px)", true},
		{"(var(--a))", true},
		{"var(a)", false},
		{"calc(1px)", false},
		{"red", false},
	}
	for _, tt := range tests {
		if got := CheckVar(parseOne(t, tt.css)); got != tt.want {
			t.Errorf("CheckVar(%q) = %v, want %v", tt.css, got, tt.want)
		}
	}
}

func TestParse2DPosition(t *testing.T) {
	tests := []struct {
		css  string
		x, y string
		ok   bool
	}{
		{"left top", "0%", "0%", true},
		{"top left", "0%", "0%", true},
		{"center", "50%", "50%", true},
		{"10px", "10px", "50%", true},
		{"right 3px", "100%", "3px", true},
		{"left right", "", "", false},
	}
	for _, tt := range tests {
		tokens := RemoveWhitespace(syntax.ParseComponentValueList(tt.css, true))
		x, y, ok := Parse2DPosition(tokens)
		if ok != tt.ok {
			t.Errorf("Parse2DPosition(%q) ok = %v, want %v", tt.css, ok, tt.ok)
			continue
		}
		if ok && (lengthString(x) != tt.x || lengthString(y) != tt.y) {
			t.Errorf("Parse2DPosition(%q) = %s %s, want %s %s", tt.css, lengthString(x), lengthString(y), tt.x, tt.y)
		}
	}
}

func lengthString(l props.Length) string {
	return formatNumber(l.Dimension.Value) + l.Dimension.Unit
}
