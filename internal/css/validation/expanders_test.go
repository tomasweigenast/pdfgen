package validation

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

type values = map[string]props.Value

func assertExpands(t *testing.T, css string, want values) {
	t.Helper()
	got := expandToMap(t, css)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s\n got  %#v\n want %#v", css, got, want)
	}
}

func assertExpansionInvalid(t *testing.T, css string, message ...string) {
	t.Helper()
	want := "invalid"
	if len(message) > 0 {
		want = message[0]
	}
	got := getValueExpand(t, css, want)
	if len(got) != 0 {
		t.Errorf("%s: got %#v, want it ignored", css, got)
	}
}

// getValueExpand preprocesses css expecting exactly one log containing want.
func getValueExpand(t *testing.T, css, want string) values {
	t.Helper()
	capture := startCapture()
	result := PreprocessDeclarations(testBaseURL, syntax.ParseBlocksContents(css, false, false))
	capture.Close()
	logs := capture.Lines()
	if len(logs) != 1 || !strings.Contains(logs[0], want) {
		t.Fatalf("%q: logs = %q, want one containing %q", css, logs, want)
	}
	m := values{}
	for _, d := range result {
		if d.Value != props.Keyword("initial") {
			m[d.Name] = d.Value
		}
	}
	return m
}

func inherit(names ...string) values {
	m := values{}
	for _, n := range names {
		m[n] = props.Keyword("inherit")
	}
	return m
}

func sides(format string, v props.Value) values {
	m := values{}
	for _, side := range []string{"top", "right", "bottom", "left"} {
		m[strings.Replace(format, "*", side, 1)] = v
	}
	return m
}

func TestEmptyExpanderValueIsIgnored(t *testing.T) {
	for name := range expanders {
		assertExpansionInvalid(t, name+":", "Ignored")
	}
}

func TestTextDecoration(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"none", values{"text-decoration-line": kws("none")}},
		{"overline", values{"text-decoration-line": kws("overline")}},
		{"overline blink line-through", values{"text-decoration-line": kws("overline", "blink", "line-through")}},
		{"red", values{"text-decoration-color": rgb(1, 0, 0, 1)}},
		{"blue 1px", values{"text-decoration-color": rgb(0, 0, 1, 1), "text-decoration-thickness": px(1)}},
		{"100% none", values{"text-decoration-line": kws("none"), "text-decoration-thickness": pct(100)}},
		{"inherit", inherit("text-decoration-color", "text-decoration-line", "text-decoration-style", "text-decoration-thickness")},
	}
	for _, tt := range tests {
		assertExpands(t, "text-decoration: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"solid solid", "red red", "underline none", "1px 100%", "none none"} {
		assertExpansionInvalid(t, "text-decoration: "+rule)
	}
}

func TestFourSides(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"margin: inherit", inherit("margin-top", "margin-right", "margin-bottom", "margin-left")},
		{"margin: 1em", sides("margin-*", dim(1, "em"))},
		{"margin: -1em auto 20%", values{"margin-top": dim(-1, "em"), "margin-right": kw("auto"), "margin-bottom": pct(20), "margin-left": kw("auto")}},
		{"padding: 1em 0", values{"padding-top": dim(1, "em"), "padding-right": num(0), "padding-bottom": dim(1, "em"), "padding-left": num(0)}},
		{"padding: 1em 0 2%", values{"padding-top": dim(1, "em"), "padding-right": num(0), "padding-bottom": pct(2), "padding-left": num(0)}},
		{"padding: 1em 0 2em 5px", values{"padding-top": dim(1, "em"), "padding-right": num(0), "padding-bottom": dim(2, "em"), "padding-left": px(5)}},
	}
	for _, tt := range tests {
		assertExpands(t, tt.rule, tt.want)
	}
}

func TestFourSidesWarning(t *testing.T) {
	assertExpansionInvalid(t, "padding: 1 2 3 4 5", "Expected 1 to 4 tokens, got 5")
}

func TestFourSidesInvalid(t *testing.T) {
	for _, rule := range []string{"margin: rgb(0, 0, 0)", "padding: auto", "padding: -12px", "border-width: -3em", "border-width: 12%"} {
		assertExpansionInvalid(t, rule)
	}
}

func TestBorders(t *testing.T) {
	red, lime := rgb(1, 0, 0, 1), rgb(0, 1, 0, 1)
	tests := []struct {
		rule string
		want values
	}{
		{"border-top: 3px dotted red", values{"border-top-width": px(3), "border-top-style": props.Keyword("dotted"), "border-top-color": red}},
		{"border-top: 3px dotted", values{"border-top-width": px(3), "border-top-style": props.Keyword("dotted")}},
		{"border-top: 3px red", values{"border-top-width": px(3), "border-top-color": red}},
		{"border-top: solid", values{"border-top-style": props.Keyword("solid")}},
	}
	all := values{}
	for _, side := range []string{"top", "left", "bottom", "right"} {
		all["border-"+side+"-width"] = px(6)
		all["border-"+side+"-style"] = props.Keyword("dashed")
		all["border-"+side+"-color"] = lime
	}
	tests = append(tests, struct {
		rule string
		want values
	}{"border: 6px dashed lime", all})
	for _, tt := range tests {
		assertExpands(t, tt.rule, tt.want)
	}
	assertExpansionInvalid(t, "border: 6px dashed left")
}

func TestListStyle(t *testing.T) {
	typ := func(name string) props.ListStyleType { return props.ListStyleType{Name: name} }
	none := props.Image{Type: "none"}
	tests := []struct {
		rule string
		want values
	}{
		{"inherit", inherit("list-style-position", "list-style-image", "list-style-type")},
		{"url(../bar/lipsum.png)", values{"list-style-image": props.Image{Type: "url", URL: "https://example.org/bar/lipsum.png"}}},
		{"square", values{"list-style-type": typ("square")}},
		{"circle inside", values{"list-style-position": props.Keyword("inside"), "list-style-type": typ("circle")}},
		{"none circle inside", values{"list-style-position": props.Keyword("inside"), "list-style-image": none, "list-style-type": typ("circle")}},
		{"none inside none", values{"list-style-position": props.Keyword("inside"), "list-style-image": none, "list-style-type": typ("none")}},
		{"inside special none", values{"list-style-position": props.Keyword("inside"), "list-style-image": none, "list-style-type": typ("special")}},
	}
	for _, tt := range tests {
		assertExpands(t, "list-style: "+tt.rule, tt.want)
	}
	assertExpansionInvalid(t, "list-style: circle disc", "got multiple type values in a list-style shorthand")
	for _, rule := range []string{"none inside none none", "1px"} {
		assertExpansionInvalid(t, "list-style: "+rule)
	}
}

func TestBackground(t *testing.T) {
	red := rgb(1, 0, 0, 1)
	pos := func(ox string, x props.Length, oy string, y props.Length) props.Positions {
		return props.Positions{{OriginX: props.Keyword(ox), X: x, OriginY: props.Keyword(oy), Y: y}}
	}
	repeat := func(x, y string) props.Repeats { return props.Repeats{{X: props.Keyword(x), Y: props.Keyword(y)}} }
	urlImage := func(name string) props.Images {
		return props.Images{{Type: "url", URL: "https://example.org/foo/" + name}}
	}
	center := pos("left", pct(50), "top", pct(50))
	tests := []struct {
		rule string
		want values
	}{
		{"red", values{"background-color": red}},
		{"url(lipsum.png)", values{"background-image": urlImage("lipsum.png")}},
		{"no-repeat", values{"background-repeat": repeat("no-repeat", "no-repeat")}},
		{"fixed", values{"background-attachment": kws("fixed")}},
		{"repeat no-repeat fixed", values{"background-repeat": repeat("repeat", "no-repeat"), "background-attachment": kws("fixed")}},
		{"top", values{"background-position": pos("left", pct(50), "top", pct(0))}},
		{"top right", values{"background-position": pos("left", pct(100), "top", pct(0))}},
		{"top right 20px", values{"background-position": pos("right", px(20), "top", pct(0))}},
		{"top 1% right 20px", values{"background-position": pos("right", px(20), "top", pct(1))}},
		{"top no-repeat", values{"background-repeat": repeat("no-repeat", "no-repeat"), "background-position": pos("left", pct(50), "top", pct(0))}},
		{"top right no-repeat", values{"background-repeat": repeat("no-repeat", "no-repeat"), "background-position": pos("left", pct(100), "top", pct(0))}},
		{"top right 20px no-repeat", values{"background-repeat": repeat("no-repeat", "no-repeat"), "background-position": pos("right", px(20), "top", pct(0))}},
		{"top 1% right 20px no-repeat", values{"background-repeat": repeat("no-repeat", "no-repeat"), "background-position": pos("right", px(20), "top", pct(1))}},
		{"url(bar) #f00 repeat-y center left fixed", values{
			"background-color": red, "background-image": urlImage("bar"),
			"background-repeat": repeat("no-repeat", "repeat"), "background-attachment": kws("fixed"),
			"background-position": pos("left", pct(0), "top", pct(50))}},
		{"#00f 10% 200px", values{"background-color": rgb(0, 0, 1, 1), "background-position": pos("left", pct(10), "top", px(200))}},
		{"right 78px fixed", values{"background-attachment": kws("fixed"), "background-position": pos("left", pct(100), "top", px(78))}},
		{"center / cover red", values{"background-size": props.BackgroundSizes{{Keyword: "cover"}}, "background-position": center, "background-color": red}},
		{"center / auto red", values{"background-size": props.BackgroundSizes{{Width: auto, Height: auto}}, "background-position": center, "background-color": red}},
		{"center / 42px", values{"background-size": props.BackgroundSizes{{Width: px(42), Height: auto}}, "background-position": center}},
		{"center / 7% 4em", values{"background-size": props.BackgroundSizes{{Width: pct(7), Height: dim(4, "em")}}, "background-position": center}},
		{"red content-box", values{"background-color": red, "background-origin": kws("content-box"), "background-clip": kws("content-box")}},
		{"red border-box content-box", values{"background-color": red, "background-origin": kws("border-box"), "background-clip": kws("content-box")}},
		{"border-box red", values{"background-color": red, "background-origin": kws("border-box"), "background-clip": kws("border-box")}},
		{"url(bar) center, no-repeat", values{
			"background-color": props.Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)},
			"background-image": props.Images{{Type: "url", URL: "https://example.org/foo/bar"}, {Type: "none"}},
			"background-position": props.Positions{
				{OriginX: "left", X: pct(50), OriginY: "top", Y: pct(50)},
				{OriginX: "left", X: pct(0), OriginY: "top", Y: pct(0)}},
			"background-repeat": props.Repeats{{X: "repeat", Y: "repeat"}, {X: "no-repeat", Y: "no-repeat"}}}},
	}
	for _, tt := range tests {
		got := expandToMap(t, "background: "+tt.rule)
		wantColor, ok := tt.want["background-color"]
		if !ok {
			wantColor = props.Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)}
		}
		if !reflect.DeepEqual(got["background-color"], wantColor) {
			t.Errorf("%s: background-color = %#v, want %#v", tt.rule, got["background-color"], wantColor)
		}
		delete(got, "background-color")
		delete(tt.want, "background-color")
		layers := 1
		if images, ok := got["background-image"].(props.Images); ok {
			layers = len(images)
		}
		for name, v := range got {
			want, ok := tt.want[name]
			if !ok {
				want = repeatLayer(initialLayer(name), layers)
			}
			if !reflect.DeepEqual(v, want) {
				t.Errorf("background: %s: %s = %#v, want %#v", tt.rule, name, v, want)
			}
		}
	}
	assertExpands(t, "background: inherit", inherit(backgroundLonghands...))
	for _, rule := range []string{"red, url(foo)", "10px lipsum", "content-box red content-box"} {
		assertExpansionInvalid(t, "background: "+rule)
	}
}

func repeatLayer(v props.Value, n int) props.Value {
	values := make([]props.Value, n)
	for i := range values {
		values[i] = v
	}
	return backgroundList("background-"+layerKind(v), values)
}

// layerKind maps the type of a layer value to the suffix of its longhand.
func layerKind(v props.Value) string {
	switch v := v.(type) {
	case props.Image:
		return "image"
	case props.Repeat:
		return "repeat"
	case props.Position:
		return "position"
	case props.BackgroundSize:
		return "size"
	case props.Keyword:
		switch v {
		case "scroll":
			return "attachment"
		case "border-box":
			return "clip"
		}
		return "origin"
	}
	return ""
}

func TestBorderRadius(t *testing.T) {
	pair := func(x, y props.Length) props.LengthPair { return props.LengthPair{X: x, Y: y} }
	radii := func(tl, tr, br, bl props.LengthPair) values {
		return values{"border-top-left-radius": tl, "border-top-right-radius": tr, "border-bottom-right-radius": br, "border-bottom-left-radius": bl}
	}
	same := func(l props.Length) props.LengthPair { return pair(l, l) }
	tests := []struct {
		rule string
		want values
	}{
		{"1px", radii(same(px(1)), same(px(1)), same(px(1)), same(px(1)))},
		{"1px 2em", radii(same(px(1)), same(dim(2, "em")), same(px(1)), same(dim(2, "em")))},
		{"1px / 2em", radii(pair(px(1), dim(2, "em")), pair(px(1), dim(2, "em")), pair(px(1), dim(2, "em")), pair(px(1), dim(2, "em")))},
		{"1px 3px / 2em 4%", radii(pair(px(1), dim(2, "em")), pair(px(3), pct(4)), pair(px(1), dim(2, "em")), pair(px(3), pct(4)))},
		{"1px 2em 3%", radii(same(px(1)), same(dim(2, "em")), same(pct(3)), same(dim(2, "em")))},
		{"1px 2em 3% 4rem", radii(same(px(1)), same(dim(2, "em")), same(pct(3)), same(dim(4, "rem")))},
		{"inherit", inherit("border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius")},
	}
	for _, tt := range tests {
		assertExpands(t, "border-radius: "+tt.rule, tt.want)
	}
	invalid := []struct{ rule, message string }{
		{"1px 1px 1px 1px 1px", "1 to 4 token"},
		{"1px 1px 1px 1px 1px / 1px", "1 to 4 token"},
		{"1px / 1px / 1px", `only one "/"`},
		{"12deg", "invalid"},
		{"1px 1px 1px 12deg", "invalid"},
		{"super", "invalid"},
		{"1px, 1px", "invalid"},
		{"1px /", `value after "/"`},
	}
	for _, tt := range invalid {
		assertExpansionInvalid(t, "border-radius: "+tt.rule, tt.message)
	}
}

func quad(l props.Length) props.LengthQuad { return props.LengthQuad{l, l, l, l} }

func slice(d props.Dimension) props.ImageSlice {
	return props.ImageSlice{Edges: [4]props.Dimension{d, d, d, d}}
}

func TestBorderImageAndMaskBorder(t *testing.T) {
	for _, prefix := range []string{"border-image", "mask-border"} {
		src := prefix + "-source"
		source := props.Image{Type: "url", URL: "https://example.org/foo/border.png"}
		tests := []struct {
			rule string
			want values
		}{
			{`url(border.png) 27`, values{src: source, prefix + "-slice": slice(props.Dimension{Value: 27})}},
			{`url(border.png) 10 / 4 / 2 round stretch`, values{
				src: source, prefix + "-slice": slice(props.Dimension{Value: 10}),
				prefix + "-width": quad(num(4)), prefix + "-outset": quad(num(2)),
				prefix + "-repeat": props.KeywordPair{"round", "stretch"}}},
			{`10 // 2`, values{prefix + "-slice": slice(props.Dimension{Value: 10}), prefix + "-outset": quad(num(2))}},
			{`5.5%`, values{prefix + "-slice": slice(props.Dimension{Value: 5.5, Unit: "%"})}},
			{`stretch 2 url("border.png")`, values{src: source, prefix + "-slice": slice(props.Dimension{Value: 2}), prefix + "-repeat": props.KeywordPair{"stretch", "stretch"}}},
			{`1/2 round`, values{prefix + "-slice": slice(props.Dimension{Value: 1}), prefix + "-width": quad(num(2)), prefix + "-repeat": props.KeywordPair{"round", "round"}}},
			{`none`, values{src: props.Image{Type: "none"}}},
		}
		if prefix == "mask-border" {
			tests = append(tests,
				struct {
					rule string
					want values
				}{`url(border.png) 27 alpha`, values{src: source, prefix + "-slice": slice(props.Dimension{Value: 27}), prefix + "-mode": props.Keyword("alpha")}},
				struct {
					rule string
					want values
				}{`url(border.png) 27 luminance`, values{src: source, prefix + "-slice": slice(props.Dimension{Value: 27}), prefix + "-mode": props.Keyword("luminance")}},
			)
		}
		for _, tt := range tests {
			assertExpands(t, prefix+": "+tt.rule, tt.want)
		}

		invalid := []struct{ rule, reason string }{
			{"url(border.png) url(border.png)", "multiple source"},
			{"10 10 10 10 10", "multiple slice"},
			{"1 / 2 / 3 / 4", "invalid"},
			{"/1", "invalid"},
			{"round round round", "invalid"},
			{"-1", "invalid"},
			{"1 repeat 2", "multiple slice"},
			{"1% // 1%", "invalid"},
			{"1 / repeat", "invalid"},
			{"", "no value"},
		}
		if prefix == "mask-border" {
			invalid = append(invalid, struct{ rule, reason string }{"alpha alpha", "multiple mode"}, struct{ rule, reason string }{"alpha luminance", "multiple mode"})
		}
		for _, tt := range invalid {
			assertExpansionInvalid(t, prefix+": "+tt.rule, tt.reason)
		}
	}
}

func TestFont(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"12px My Fancy Font, serif", values{"font-size": px(12), "font-family": props.Strings{"My Fancy Font", "serif"}}},
		{`small/1.2 "Some Font", serif`, values{"font-size": kw("small"), "line-height": num(1.2), "font-family": props.Strings{"Some Font", "serif"}}},
		{"small-caps italic 700 large serif", values{
			"font-style": props.Keyword("italic"), "font-variant-caps": props.Keyword("small-caps"),
			"font-weight": props.Integer(700), "font-size": kw("large"), "font-family": props.Strings{"serif"}}},
		{"small-caps condensed normal 700 large serif", values{
			"font-stretch": props.Keyword("condensed"), "font-variant-caps": props.Keyword("small-caps"),
			"font-weight": props.Integer(700), "font-size": kw("large"), "font-family": props.Strings{"serif"}}},
	}
	for _, tt := range tests {
		assertExpands(t, "font: "+tt.rule, tt.want)
	}
	invalid := []struct{ rule, message string }{
		{"menu", "System fonts are not supported"},
		{"12deg My Fancy Font, serif", "invalid"},
		{"12px", "invalid"},
		{"12px/foo serif", "invalid"},
		{`12px "Invalid" family`, "invalid"},
		{"normal normal normal normal normal large serif", "invalid"},
		{"normal small-caps italic 700 condensed large serif", "invalid"},
		{"small-caps italic 700 normal condensed large serif", "invalid"},
		{"small-caps italic 700 condensed normal large serif", "invalid"},
		{"normal normal normal normal", "invalid"},
		{"normal normal normal italic", "invalid"},
		{"caption", "System fonts"},
	}
	for _, tt := range invalid {
		assertExpansionInvalid(t, "font: "+tt.rule, tt.message)
	}
}

func TestFontVariant(t *testing.T) {
	normal := values{
		"font-variant-alternates": props.Keyword("normal"), "font-variant-caps": props.Keyword("normal"),
		"font-variant-east-asian": kws("normal"), "font-variant-ligatures": kws("normal"),
		"font-variant-numeric": kws("normal"), "font-variant-position": props.Keyword("normal"),
	}
	none := values{}
	for k, v := range normal {
		none[k] = v
	}
	none["font-variant-ligatures"] = kws("none")
	tests := []struct {
		rule string
		want values
	}{
		{"normal", normal},
		{"none", none},
		{"historical-forms petite-caps", values{"font-variant-alternates": props.Keyword("historical-forms"), "font-variant-caps": props.Keyword("petite-caps")}},
		{"lining-nums contextual small-caps common-ligatures", values{
			"font-variant-ligatures": kws("contextual", "common-ligatures"),
			"font-variant-numeric":   kws("lining-nums"), "font-variant-caps": props.Keyword("small-caps")}},
		{"jis78 ruby proportional-width", values{"font-variant-east-asian": kws("jis78", "ruby", "proportional-width")}},
		{"small-caps", values{"font-variant-caps": props.Keyword("small-caps")}},
	}
	for _, tt := range tests {
		assertExpands(t, "font-variant: "+tt.rule, tt.want)
	}
	for _, rule := range []string{
		"normal normal", "2", `""`, "extra", "jis78 jis04", "full-width lining-nums ordinal normal",
		"diagonal-fractions stacked-fractions", "common-ligatures contextual no-common-ligatures",
		"sub super", "slashed-zero slashed-zero",
	} {
		assertExpansionInvalid(t, "font-variant: "+rule)
	}
}

func TestWordWrap(t *testing.T) {
	assertExpands(t, "word-wrap: normal", values{"overflow-wrap": props.Keyword("normal")})
	assertExpands(t, "word-wrap: break-word", values{"overflow-wrap": props.Keyword("break-word")})
	assertExpands(t, "word-wrap: inherit", inherit("overflow-wrap"))
	assertExpansionInvalid(t, "word-wrap: none")
	assertExpansionInvalid(t, "word-wrap: normal, break-word")
}

func TestFlex(t *testing.T) {
	flex := func(grow, shrink float64, basis props.Length) values {
		return values{"flex-grow": props.Number(grow), "flex-shrink": props.Number(shrink), "flex-basis": basis}
	}
	tests := []struct {
		rule string
		want values
	}{
		{"auto", flex(1, 1, kw("auto"))},
		{"none", flex(0, 0, kw("auto"))},
		{"10", flex(10, 1, px(0))},
		{"2 2", flex(2, 2, px(0))},
		{"2 2 1px", flex(2, 2, px(1))},
		{"2 2 auto", flex(2, 2, kw("auto"))},
		{"2 auto", flex(2, 1, kw("auto"))},
		{"0 auto", flex(0, 1, kw("auto"))},
		{"inherit", inherit("flex-grow", "flex-shrink", "flex-basis")},
	}
	for _, tt := range tests {
		assertExpands(t, "flex: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"auto 0 0 0", "1px 2px", "auto auto", "auto 1 auto"} {
		assertExpansionInvalid(t, "flex: "+rule)
	}
}

func TestFlexFlow(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"column", values{"flex-direction": props.Keyword("column")}},
		{"wrap", values{"flex-wrap": props.Keyword("wrap")}},
		{"wrap column", values{"flex-direction": props.Keyword("column"), "flex-wrap": props.Keyword("wrap")}},
		{"row wrap", values{"flex-direction": props.Keyword("row"), "flex-wrap": props.Keyword("wrap")}},
		{"inherit", inherit("flex-direction", "flex-wrap")},
	}
	for _, tt := range tests {
		assertExpands(t, "flex-flow: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"1px", "wrap 1px", "row row", "wrap nowrap", "column wrap nowrap row"} {
		assertExpansionInvalid(t, "flex-flow: "+rule)
	}
}

func TestPageBreak(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"page-break-after: left", values{"break-after": props.Keyword("left")}},
		{"page-break-before: always", values{"break-before": props.Keyword("page")}},
		{"page-break-after: inherit", inherit("break-after")},
		{"page-break-before: inherit", inherit("break-before")},
		{"page-break-inside: avoid", values{"break-inside": props.Keyword("avoid")}},
		{"page-break-inside: inherit", inherit("break-inside")},
	}
	for _, tt := range tests {
		assertExpands(t, tt.rule, tt.want)
	}
	for _, rule := range []string{"page-break-after: top", "page-break-before: 1px", "page-break-inside: top"} {
		assertExpansionInvalid(t, rule)
	}
}

func TestColumns(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"1em", values{"column-width": dim(1, "em"), "column-count": props.IntOrKeyword{Keyword: "auto"}}},
		{"auto", values{"column-width": kw("auto"), "column-count": props.IntOrKeyword{Keyword: "auto"}}},
		{"auto auto", values{"column-width": kw("auto"), "column-count": props.IntOrKeyword{Keyword: "auto"}}},
	}
	for _, tt := range tests {
		assertExpands(t, "columns: "+tt.rule, tt.want)
	}
	assertExpansionInvalid(t, "columns: 1px 2px", "invalid")
	assertExpansionInvalid(t, "columns: auto auto auto", "multiple")
}

func TestLineClamp(t *testing.T) {
	tests := []struct {
		rule string
		want values
	}{
		{"none", values{"max-lines": props.IntOrKeyword{Keyword: "none"}, "continue": props.Keyword("auto"), "block-ellipsis": props.KeywordOrString{Keyword: "none"}}},
		{"2", values{"max-lines": props.IntOrKeyword{N: 2}, "continue": props.Keyword("discard"), "block-ellipsis": props.KeywordOrString{Keyword: "auto"}}},
		{`3 "…"`, values{"max-lines": props.IntOrKeyword{N: 3}, "continue": props.Keyword("discard"), "block-ellipsis": props.KeywordOrString{String: "…"}}},
		{"inherit", inherit("max-lines", "continue", "block-ellipsis")},
	}
	for _, tt := range tests {
		assertExpands(t, "line-clamp: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"none none none", "1px", `0 "…"`, "1px 2px"} {
		assertExpansionInvalid(t, "line-clamp: "+rule)
	}
}

func TestTextAlign(t *testing.T) {
	both := func(all, last string) values {
		return values{"text-align-all": props.Keyword(all), "text-align-last": props.Keyword(last)}
	}
	tests := []struct {
		rule string
		want values
	}{
		{"start", both("start", "start")},
		{"right", both("right", "right")},
		{"justify", both("justify", "start")},
		{"justify-all", both("justify", "justify")},
		{"inherit", inherit("text-align-all", "text-align-last")},
	}
	for _, tt := range tests {
		assertExpands(t, "text-align: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"none", "start end", "1", "left left", "top", `"right"`, "1px"} {
		assertExpansionInvalid(t, "text-align: "+rule)
	}
}

func TestGridColumnAndRow(t *testing.T) {
	auto := props.GridLine{Auto: true}
	n := func(v int) props.GridLine { return props.GridLine{N: v} }
	name := func(s string) props.GridLine { return props.GridLine{Ident: s} }
	tests := []struct {
		rule       string
		start, end props.GridLine
	}{
		{"auto", auto, auto},
		{"auto / auto", auto, auto},
		{"4", n(4), auto},
		{"c", name("c"), name("c")},
		{"4 / -4", n(4), n(-4)},
		{"c / d", name("c"), name("d")},
		{"ab / cd 4", name("ab"), props.GridLine{N: 4, Ident: "cd"}},
		{"ab 2 span", props.GridLine{Span: true, N: 2, Ident: "ab"}, auto},
	}
	for _, tt := range tests {
		for _, axis := range []string{"column", "row"} {
			assertExpands(t, "grid-"+axis+": "+tt.rule, values{"grid-" + axis + "-start": tt.start, "grid-" + axis + "-end": tt.end})
		}
	}
	for _, rule := range []string{"auto auto", "4 / 2 / c", "span", "4 / span", "c /", "/4", "col / 2.1"} {
		assertExpansionInvalid(t, "grid-column: "+rule)
		assertExpansionInvalid(t, "grid-row: "+rule)
	}
}

func TestGridArea(t *testing.T) {
	auto := props.GridLine{Auto: true}
	all := func(l props.GridLine) values {
		return values{"grid-row-start": l, "grid-row-end": l, "grid-column-start": l, "grid-column-end": l}
	}
	tests := []struct {
		rule string
		want values
	}{
		{"auto", all(auto)},
		{"auto / auto", all(auto)},
		{"auto / auto / auto", all(auto)},
		{"auto / auto / auto / auto", all(auto)},
		{"1/c/2 d/span 2 ab", values{
			"grid-row-start": props.GridLine{N: 1}, "grid-column-start": props.GridLine{Ident: "c"},
			"grid-row-end": props.GridLine{N: 2, Ident: "d"}, "grid-column-end": props.GridLine{Span: true, N: 2, Ident: "ab"}}},
		{"1  /  c", values{
			"grid-row-start": props.GridLine{N: 1}, "grid-column-start": props.GridLine{Ident: "c"},
			"grid-row-end": auto, "grid-column-end": props.GridLine{Ident: "c"}}},
		{"a / c 2", values{
			"grid-row-start": props.GridLine{Ident: "a"}, "grid-column-start": props.GridLine{N: 2, Ident: "c"},
			"grid-row-end": props.GridLine{Ident: "a"}, "grid-column-end": auto}},
		{"a", all(props.GridLine{Ident: "a"})},
		{"span 2", values{
			"grid-row-start": props.GridLine{Span: true, N: 2}, "grid-row-end": auto,
			"grid-column-start": auto, "grid-column-end": auto}},
	}
	for _, tt := range tests {
		assertExpands(t, "grid-area: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"auto auto", "auto / auto auto", "4 / 2 / c / d / e", "span", "4 / span", "c /", "/4", "c//4", "/", "1 / 2 / 4 / 0.5"} {
		assertExpansionInvalid(t, "grid-area: "+rule)
	}
}

func names(n ...string) props.GridItem {
	return props.GridItem{Kind: props.GridLineNames, Names: n}
}

func trackItem(l props.Length) props.GridItem {
	return props.GridItem{Kind: props.GridTrack, Track: props.TrackSize{Breadth: l}}
}

func fr(v float64) props.Length { return dim(v, "fr") }

func repeatItem(count int, auto string, items ...props.GridItem) props.GridItem {
	return props.GridItem{Kind: props.GridRepeat, Count: count, Auto: props.Keyword(auto), Items: items}
}

func template(items ...props.GridItem) props.GridTemplate { return props.GridTemplate{Items: items} }

func subgrid(items ...props.GridItem) props.GridTemplate {
	return props.GridTemplate{Subgrid: true, Items: items}
}

var (
	noneTemplate = props.GridTemplate{None: true}
	noneAreas    = props.GridAreas{None: true}
	autoTracks   = props.TrackList{{Breadth: kw("auto")}}
)

func TestGridTemplate(t *testing.T) {
	rows := template(names(), repeatItem(2, "", names("e"), trackItem(px(40)), names()), names(), repeatItem(5, "", names(), trackItem(kw("auto")), names()), names())
	columns := subgrid(names("a"), repeatItem(0, "auto-fill", names("b")))
	tests := []struct {
		rule string
		want values
	}{
		{"none", values{"grid-template-rows": noneTemplate, "grid-template-columns": noneTemplate, "grid-template-areas": noneAreas}},
		{"subgrid / [outer-edge] 20px [main-start]", values{
			"grid-template-rows": subgrid(), "grid-template-columns": template(names("outer-edge"), trackItem(px(20)), names("main-start")),
			"grid-template-areas": noneAreas}},
		{"repeat(2, [e] 40px) repeat(5, auto) / subgrid [a] repeat(auto-fill, [b])", values{
			"grid-template-rows": rows, "grid-template-columns": columns, "grid-template-areas": noneAreas}},
	}
	for _, tt := range tests {
		assertExpands(t, "grid-template: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"none none", "auto", "subgrid / subgrid / subgrid", "[a] 1px [b] / none /", "[a] 1px [b] // none", "[a] 1px [b] none"} {
		assertExpansionInvalid(t, "grid-template: "+rule)
	}
}

func TestGrid(t *testing.T) {
	autoFlowRow := kws("row")
	rows := template(names(), repeatItem(2, "", names("e"), trackItem(px(40)), names()), names(), repeatItem(5, "", names(), trackItem(kw("auto")), names()), names())
	columns := subgrid(names("a"), repeatItem(0, "auto-fill", names("b")))
	tests := []struct {
		rule string
		want values
	}{
		{"none", values{
			"grid-template-rows": noneTemplate, "grid-template-columns": noneTemplate, "grid-template-areas": noneAreas,
			"grid-auto-rows": autoTracks, "grid-auto-columns": autoTracks, "grid-auto-flow": autoFlowRow}},
		{"subgrid / [outer-edge] 20px [main-start]", values{
			"grid-template-rows": subgrid(), "grid-template-columns": template(names("outer-edge"), trackItem(px(20)), names("main-start")),
			"grid-template-areas": noneAreas, "grid-auto-rows": autoTracks, "grid-auto-columns": autoTracks, "grid-auto-flow": autoFlowRow}},
		{"repeat(2, [e] 40px) repeat(5, auto) / subgrid [a] repeat(auto-fill, [b])", values{
			"grid-template-rows": rows, "grid-template-columns": columns, "grid-template-areas": noneAreas,
			"grid-auto-rows": autoTracks, "grid-auto-columns": autoTracks, "grid-auto-flow": autoFlowRow}},
		{"auto-flow 1fr / 100px", values{
			"grid-template-rows": noneTemplate, "grid-template-columns": template(names(), trackItem(px(100)), names()),
			"grid-template-areas": noneAreas, "grid-auto-rows": props.TrackList{{Breadth: fr(1)}},
			"grid-auto-columns": autoTracks, "grid-auto-flow": autoFlowRow}},
		{"none / dense auto-flow 1fr", values{
			"grid-template-rows": noneTemplate, "grid-template-columns": noneTemplate, "grid-template-areas": noneAreas,
			"grid-auto-rows": autoTracks, "grid-auto-columns": props.TrackList{{Breadth: fr(1)}},
			"grid-auto-flow": kws("column", "dense")}},
	}
	for _, tt := range tests {
		assertExpands(t, "grid: "+tt.rule, tt.want)
	}
	for _, rule := range []string{
		"none none", "auto", "subgrid / subgrid / subgrid", "[a] 1px [b] / none /", "[a] 1px [b] // none",
		"[a] 1px [b] none", "none / auto-flow 1fr dense", "none / dense 1fr auto-flow",
		"100px auto-flow / none", "dense 100px / auto-flow 1fr",
	} {
		assertExpansionInvalid(t, "grid: "+rule)
	}
}
