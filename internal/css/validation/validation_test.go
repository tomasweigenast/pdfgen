package validation

import (
	"log/slog"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func TestNotPrintPropertiesAreIgnored(t *testing.T) {
	if v := getValueLevel(t, "volume: 42", "does not apply for the print media", slog.LevelDebug); v != nil {
		t.Errorf("got %v", v)
	}
}

func TestUnstablePropertyWithPrefixWarns(t *testing.T) {
	old := VendorPrefix
	VendorPrefix = "-x-"
	defer func() { VendorPrefix = old }()
	got := getValue(t, "-x-max-lines: 3", "prefixes on unstable attributes are deprecated")
	if got != (props.IntOrKeyword{N: 3}) {
		t.Errorf("got %#v", got)
	}
}

func TestStablePropertyWithPrefixIsIgnored(t *testing.T) {
	old := VendorPrefix
	VendorPrefix = "-x-"
	defer func() { VendorPrefix = old }()
	assertInvalid(t, "-x-display: block", "prefix on this attribute is not supported")
}

func TestUnknownPrefixIsIgnored(t *testing.T) {
	if v := getValueLevel(t, "-unknown-display: block", "prefixed selectors are ignored", slog.LevelDebug); v != nil {
		t.Errorf("got %v", v)
	}
}

func TestEmptyPropertyValueIsIgnored(t *testing.T) {
	for _, p := range props.Table {
		if p.Validate == nil {
			continue
		}
		assertInvalid(t, p.Name+":", "Ignored")
	}
}

func TestClip(t *testing.T) {
	tests := []struct {
		rule string
		want props.Clip
	}{
		{"1px, 3EM, auto, auto", props.Clip{px(1), dim(3, "em"), kw("auto"), kw("auto")}},
		{"1px,3em,auto,1px", props.Clip{px(1), dim(3, "em"), kw("auto"), px(1)}},
	}
	for _, tt := range tests {
		assertValue(t, "clip: rect("+tt.rule+")", tt.want)
	}
}

func TestClipInvalid(t *testing.T) {
	for _, rule := range []string{
		"clip: square(1px, 3em, auto, auto)",
		"clip: rect(1px 3em auto auto)",
		"clip: rect(1px, 3em, auto)",
		"clip: rect(1px, 3em / auto)",
	} {
		assertInvalid(t, rule)
	}
}

func TestCounters(t *testing.T) {
	ops := func(kv ...any) props.Counters {
		var c props.Counters
		for i := 0; i < len(kv); i += 2 {
			c.Items = append(c.Items, props.CounterOp{Name: kv[i].(string), Value: kv[i+1].(int)})
		}
		return c
	}
	tests := []struct {
		rule string
		want props.Counters
	}{
		{"counter-reset: foo bar 2 baz", ops("foo", 0, "bar", 2, "baz", 0)},
		{"counter-increment: foo bar 2 baz", ops("foo", 1, "bar", 2, "baz", 1)},
		{"counter-reset: foo", ops("foo", 0)},
		{"counter-set: FoO", ops("FoO", 0)},
		{"counter-increment: foo bAr 2 Bar", ops("foo", 1, "bAr", 2, "Bar", 1)},
		{"counter-reset: none", props.Counters{}},
	}
	for _, tt := range tests {
		assertValue(t, tt.rule, tt.want)
	}
}

func TestCountersWarnOnReservedNames(t *testing.T) {
	assertInvalid(t, "counter-reset: foo initial", "Invalid counter name: initial.")
	assertInvalid(t, "counter-reset: foo none", "Invalid counter name: none.")
}

func TestCountersInvalid(t *testing.T) {
	assertInvalid(t, "counter-reset: foo 3pX")
	assertInvalid(t, "counter-reset: 3")
}

func TestSpacing(t *testing.T) {
	for _, prop := range []string{"letter-spacing", "word-spacing"} {
		assertValue(t, prop+": normal", kw("normal"))
		assertValue(t, prop+": 3px", px(3))
		assertInvalid(t, prop+": 3")
	}
}

func TestTextDecorationLine(t *testing.T) {
	assertValue(t, "text-decoration-line: none", kws("none"))
	assertValue(t, "text-decoration-line: overline", kws("overline"))
	assertValue(t, "text-decoration-line: overline blink line-through", kws("overline", "blink", "line-through"))
}

func TestTextDecorationStyle(t *testing.T) {
	for _, s := range []string{"solid", "double", "dotted", "dashed"} {
		assertValue(t, "text-decoration-style: "+s, props.Keyword(s))
	}
}

func TestSize(t *testing.T) {
	mm := func(w, h float64) props.LengthPair { return props.LengthPair{X: dim(w, "mm"), Y: dim(h, "mm")} }
	tests := []struct {
		rule string
		want props.LengthPair
	}{
		{"200px", props.LengthPair{X: px(200), Y: px(200)}},
		{"200px 300pt", props.LengthPair{X: px(200), Y: dim(300, "pt")}},
		{"auto", mm(210, 297)},
		{"portrait", mm(210, 297)},
		{"landscape", mm(297, 210)},
		{"A3 portrait", mm(297, 420)},
		{"A3 landscape", mm(420, 297)},
		{"portrait A3", mm(297, 420)},
		{"landscape A3", mm(420, 297)},
	}
	for _, tt := range tests {
		assertValue(t, "size: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"A3 landscape A3", "A12", "foo", "foo bar", "20%"} {
		assertInvalid(t, "size: "+rule)
	}
}

func TestTransform(t *testing.T) {
	tests := []struct {
		rule string
		want props.Transform
	}{
		{"none", props.Transform{}},
		{"translate(6px) rotate(90deg)", props.Transform{
			{Name: "translate", Translate: props.LengthPair{X: px(6), Y: px(0)}},
			{Name: "rotate", Numbers: []float64{math.Pi / 2}},
		}},
		{"rotate(0)", props.Transform{{Name: "rotate", Numbers: []float64{0}}}},
		{"translate(-4px, 0)", props.Transform{{Name: "translate", Translate: props.LengthPair{X: px(-4), Y: num(0)}}}},
		{"translate(6px, 20%)", props.Transform{{Name: "translate", Translate: props.LengthPair{X: px(6), Y: pct(20)}}}},
		{"scale(2)", props.Transform{{Name: "scale", Numbers: []float64{2, 2}}}},
	}
	for _, tt := range tests {
		assertValue(t, "transform: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"lipsumize(6px)", "foo", "scale(2) foo", "6px", "translate(6px 20%)"} {
		assertInvalid(t, "transform: "+rule)
	}
}

func TestBackgroundImageInvalid(t *testing.T) {
	assertInvalid(t, "background-image: inexistent-gradient(blue, green)")
}

func TestBackgroundPosition(t *testing.T) {
	pos := func(ox string, x props.Length, oy string, y props.Length) props.Positions {
		return props.Positions{{OriginX: props.Keyword(ox), X: x, OriginY: props.Keyword(oy), Y: y}}
	}
	tests := []struct {
		rule string
		want props.Positions
	}{
		{"top", pos("left", pct(50), "top", pct(0))},
		{"bottom", pos("left", pct(50), "top", pct(100))},
		{"center top 10%", pos("left", pct(50), "top", pct(10))},
		{"top 10% center", pos("left", pct(50), "top", pct(10))},
		{"center bottom 10%", pos("left", pct(50), "bottom", pct(10))},
		{"bottom 10% center", pos("left", pct(50), "bottom", pct(10))},
		{"right top 10%", pos("right", pct(0), "top", pct(10))},
		{"top 10% right", pos("right", pct(0), "top", pct(10))},
		{"right bottom 10%", pos("right", pct(0), "bottom", pct(10))},
		{"bottom 10% right", pos("right", pct(0), "bottom", pct(10))},
		{"center left 10%", pos("left", pct(10), "top", pct(50))},
		{"left 10% center", pos("left", pct(10), "top", pct(50))},
		{"center right 10%", pos("right", pct(10), "top", pct(50))},
		{"right 10% center", pos("right", pct(10), "top", pct(50))},
		{"bottom left 10%", pos("left", pct(10), "bottom", pct(0))},
		{"left 10% bottom", pos("left", pct(10), "bottom", pct(0))},
		{"bottom right 10%", pos("right", pct(10), "bottom", pct(0))},
		{"right 10% bottom", pos("right", pct(10), "bottom", pct(0))},
		{"left 10% bottom 3PX", pos("left", pct(10), "bottom", px(3))},
		{"bottom 3px left 10%", pos("left", pct(10), "bottom", px(3))},
		{"right 10% top 3px", pos("right", pct(10), "top", px(3))},
		{"top 3px right 10%", pos("right", pct(10), "top", px(3))},
	}
	xs := []struct {
		css string
		val props.Length
	}{{"left", pct(0)}, {"center", pct(50)}, {"right", pct(100)}, {"4.5%", pct(4.5)}, {"12px", px(12)}}
	ys := []struct {
		css string
		val props.Length
	}{{"top", pct(0)}, {"center", pct(50)}, {"bottom", pct(100)}, {"7%", pct(7)}, {"1.5px", px(1.5)}}
	for _, x := range xs {
		tests = append(tests, struct {
			rule string
			want props.Positions
		}{x.css, pos("left", x.val, "top", pct(50))})
		for _, y := range ys {
			tests = append(tests, struct {
				rule string
				want props.Positions
			}{x.css + " " + y.css, pos("left", x.val, "top", y.val)})
		}
	}
	for _, tt := range tests {
		assertValue(t, "background-position: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"10px lipsum", "left center 3px", "3px left", "bottom 4%", "bottom top"} {
		assertInvalid(t, "background-position: "+rule)
	}
}

func TestFontFamilyInvalid(t *testing.T) {
	for _, rule := range []string{`"My" Font, serif`, `"My" "Font", serif`, `"My", 12pt, serif`} {
		assertInvalid(t, "font-family: "+rule)
	}
}

func TestLineHeight(t *testing.T) {
	tests := []struct {
		rule string
		want props.Length
	}{
		{"1px", px(1)}, {"1.1%", pct(1.1)}, {"1Em", dim(1, "em")},
		{"1", num(1)}, {"1.3", num(1.3)}, {"-0", num(0)}, {"0px", px(0)},
	}
	for _, tt := range tests {
		assertValue(t, "line-height: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"1deg", "-1px", "-1", "-0.5%", "1px 1px"} {
		assertInvalid(t, "line-height: "+rule)
	}
}

func TestListStyleTypeInvalid(t *testing.T) {
	for _, rule := range []string{
		"symbols()", "symbols(cyclic)", "symbols(symbolic)", "symbols(fixed)",
		`symbols(alphabetic "a")`, `symbols(numeric "1")`, `symbols(test "a" "b")`,
		`symbols(fixed symbolic "a" "b")`,
	} {
		assertInvalid(t, "list-style-type: "+rule)
	}
}

func TestImageOrientation(t *testing.T) {
	tests := []struct {
		rule string
		want props.ImageOrientation
	}{
		{"none", props.ImageOrientation{Keyword: "none"}},
		{"from-image", props.ImageOrientation{Keyword: "from-image"}},
		{"90deg", props.ImageOrientation{Angle: math.Pi / 2}},
		{"30deg", props.ImageOrientation{Angle: math.Pi / 6}},
		{"180deg flip", props.ImageOrientation{Angle: math.Pi, Flip: true}},
		{"0deg flip", props.ImageOrientation{Flip: true}},
		{"flip 90deg", props.ImageOrientation{Angle: math.Pi / 2, Flip: true}},
		{"flip", props.ImageOrientation{Flip: true}},
	}
	for _, tt := range tests {
		got := getValue(t, "image-orientation: "+tt.rule, "")
		o, ok := got.(props.ImageOrientation)
		if !ok || o.Keyword != tt.want.Keyword || o.Flip != tt.want.Flip || !approx(o.Angle, tt.want.Angle) {
			t.Errorf("image-orientation: %s = %#v, want %#v", tt.rule, got, tt.want)
		}
	}
	for _, rule := range []string{
		"none none", "unknown", "none flip", "from-image flip", "10", "10 flip",
		"flip 10", "flip flip", "90deg flop", "90deg 180deg",
	} {
		assertInvalid(t, "image-orientation: "+rule)
	}
}
