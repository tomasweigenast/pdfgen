package tokens

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func parseOne(t *testing.T, css string) syntax.Node {
	t.Helper()
	return syntax.ParseOneComponentValue(css, true)
}

type unitStyle map[string]float64

func (s unitStyle) ToPixels(value float64, unit, _ string) float64 {
	return value * s[unit]
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

func TestTokenListHelpers(t *testing.T) {
	tokens := syntax.ParseComponentValueList("  alpha, calc(1px, 2px), , beta  ", true)
	if got := len(RemoveWhitespace(tokens)); got != 6 {
		t.Fatalf("RemoveWhitespace returned %d tokens, want 6", got)
	}
	parts := SplitOnComma(RemoveWhitespace(tokens))
	if len(parts) != 4 || len(parts[1]) != 1 || len(parts[1][0].(*syntax.FunctionBlock).Arguments) == 0 || len(parts[2]) != 0 {
		t.Fatalf("SplitOnComma returned unexpected groups: %v", parts)
	}
	if GetKeyword(parseOne(t, "MiXeD")) != "mixed" || GetCustomIdent(parseOne(t, "MiXeD")) != "MiXeD" {
		t.Fatal("identifier helpers did not preserve the expected casing")
	}
	if GetSingleKeyword([]syntax.Node{parseOne(t, "BLOCK")}) != "block" || GetSingleKeyword(tokens) != "" {
		t.Fatal("GetSingleKeyword did not enforce a single identifier")
	}
}

func TestParsePositionForms(t *testing.T) {
	tests := []struct {
		css string
		ok  bool
	}{
		{"left", true}, {"top right", true}, {"right 12px bottom 3%", true},
		{"center top 4px", true}, {"top 4px right", true}, {"left right", false},
		{"left 1px top 2px extra", false},
	}
	for _, tt := range tests {
		tokens := syntax.ParseComponentValueList(tt.css, true)
		_, got := ParsePosition(RemoveWhitespace(tokens))
		if got != tt.ok {
			t.Errorf("ParsePosition(%q) valid = %v, want %v", tt.css, got, tt.ok)
		}
	}
	for _, tt := range []struct {
		css  string
		want bool
	}{{"open-quote", true}, {"NO-CLOSE-QUOTE", true}, {"close", false}} {
		_, got := GetQuote(parseOne(t, tt.css))
		if got != tt.want {
			t.Errorf("GetQuote(%q) = %v, want %v", tt.css, got, tt.want)
		}
	}
}

func TestGetResolution(t *testing.T) {
	for _, tt := range []struct {
		css  string
		want float64
		ok   bool
	}{{"96dpi", 1, true}, {"2dppx", 2, true}, {"1dpcm", 2.54 / 96, true}, {"1px", 0, false}} {
		got, ok := GetResolution(parseOne(t, tt.css))
		if ok != tt.ok || (ok && math.Abs(got-tt.want) > 1e-9) {
			t.Errorf("GetResolution(%q) = %v, %v; want %v, %v", tt.css, got, ok, tt.want, tt.ok)
		}
	}
}

func TestTokenFunctionParsing(t *testing.T) {
	fn := NewFunction(parseOne(t, `fn(one, two)`))
	if !fn.IsFunction() || fn.Name != "fn" || len(fn.SplitSpace()) == 0 || len(fn.SplitComma()) != 2 {
		t.Fatalf("function was not split into its arguments: %#v", fn)
	}
	if got := fn.SplitCommaGroups(false); len(got) != 2 || len(got[0]) != 1 || len(got[1]) != 1 {
		t.Errorf("SplitCommaGroups() = %#v", got)
	}
	for _, css := range []string{"fn(one,,two)", "fn(one two)"} {
		if got := NewFunction(parseOne(t, css)).SplitComma(); got != nil {
			t.Errorf("SplitComma(%q) = %v, want nil", css, got)
		}
	}
	if NewFunction(parseOne(t, "word")).IsFunction() || NewFunction(parseOne(t, "word")).SplitSpace() != nil {
		t.Fatal("a non-function token was accepted as a function")
	}
	for _, tt := range []struct {
		css string
		ok  bool
	}{
		{`attr(title)`, true}, {`attr(title string, "fallback")`, true},
		{`counter(chapter)`, true}, {`counter(chapter, upper-roman)`, true},
		{`counters(chapter, ".", decimal)`, true}, {`counters(chapter, decimal)`, false},
		{`content()`, true}, {`content(marker)`, true}, {`content(unknown)`, false},
		{`string(chapter)`, true}, {`string(chapter, last)`, true}, {`string(chapter, middle)`, false},
	} {
		var got bool
		switch tt.css[:strings.IndexByte(tt.css, '(')] {
		case "attr":
			_, got = CheckAttr(parseOne(t, tt.css), "")
		case "counter", "counters":
			_, got = CheckCounter(parseOne(t, tt.css))
		case "content":
			_, got = CheckContent(parseOne(t, tt.css))
		case "string":
			_, got = CheckStringOrElement("string", parseOne(t, tt.css))
		}
		if got != tt.ok {
			t.Errorf("%s valid = %v, want %v", tt.css, got, tt.ok)
		}
	}
	if _, ok, _ := GetContentListToken(parseOne(t, "leader(dotted)"), ""); !ok {
		t.Error("leader() was rejected")
	}
	if _, ok, _ := GetContentListToken(parseOne(t, "element(chapter, last)"), ""); !ok {
		t.Error("element() was rejected")
	}
}

func TestPercentageAndRelativeMathResolution(t *testing.T) {
	for _, tt := range []struct {
		css      string
		negative bool
		want     float64
		ok       bool
	}{
		{"25%", true, 25, true}, {"-25%", false, 0, false},
		{"calc(-25%)", false, 0, true}, {"10px", true, 0, false},
	} {
		got, ok := GetPercentage(parseOne(t, tt.css), tt.negative)
		if ok != tt.ok || (ok && got.Value != tt.want) {
			t.Errorf("GetPercentage(%q) = %v, %v; want %v, %v", tt.css, got, ok, tt.want, tt.ok)
		}
	}
	style := unitStyle{"em": 10, "rem": 16, "vw": 2}
	for _, tt := range []struct {
		css  string
		want float64
	}{
		{"calc(2em + 5px)", 25}, {"calc(1rem + 2px)", 18},
		{"calc(10vw + 1px)", 21}, {"min(2em, 15px)", 15},
		{"max(2em, 25px)", 25}, {"clamp(10px, 2em, 20px)", 20},
		{"round(up, 1.2, 1)", 2}, {"round(down, 1.8, 1)", 1},
		{"round(to-zero, -1.8, 1)", -1}, {"round(nearest, 2.5, 1)", 3},
		{"round(1.25, .5)", 1.5}, {"tan(45deg)", 1},
		{"asin(1)", math.Pi / 2}, {"acos(1)", 0}, {"atan(1)", math.Pi / 4},
		{"exp(0)", 1}, {"log(8, 2)", 3}, {"hypot(3, 4)", 5},
		{"sign(0)", 0}, {"abs(-2)", 2},
	} {
		got, err := ResolveMath(parseOne(t, tt.css), style, "width", nil)
		if err != nil {
			t.Errorf("ResolveMath(%q): %v", tt.css, err)
			continue
		}
		n, ok := got.(*syntax.NumberToken)
		if !ok {
			d, dim := got.(*syntax.DimensionToken)
			if !dim || math.Abs(d.Value-tt.want) > 1e-9 {
				t.Errorf("ResolveMath(%q) = %s, want %v", tt.css, got.Serialize(), tt.want)
			}
			continue
		}
		if math.Abs(n.Value-tt.want) > 1e-9 {
			t.Errorf("ResolveMath(%q) = %v, want %v", tt.css, n.Value, tt.want)
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
