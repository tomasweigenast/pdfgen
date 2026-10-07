package validation

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func TestBorderImageSlice(t *testing.T) {
	d := func(v float64, unit string) props.Dimension { return props.Dimension{Value: v, Unit: unit} }
	edges := func(fill bool, a, b, c, e props.Dimension) props.ImageSlice {
		return props.ImageSlice{Edges: [4]props.Dimension{a, b, c, e}, Fill: fill}
	}
	tests := []struct {
		rule string
		want props.ImageSlice
	}{
		{"1", edges(false, d(1, ""), d(1, ""), d(1, ""), d(1, ""))},
		{"1 2    3 4", edges(false, d(1, ""), d(2, ""), d(3, ""), d(4, ""))},
		{"50% 1000.1 0", edges(false, d(50, "%"), d(1000.1, ""), d(0, ""), d(1000.1, ""))},
		{"1% 2% 3% 4%", edges(false, d(1, "%"), d(2, "%"), d(3, "%"), d(4, "%"))},
		{"fill 10% 20", edges(true, d(10, "%"), d(20, ""), d(10, "%"), d(20, ""))},
		{"0 1 0.5 fill", edges(true, d(0, ""), d(1, ""), d(0.5, ""), d(1, ""))},
		{"calc(10%)", edges(false, d(10, "%"), d(10, "%"), d(10, "%"), d(10, "%"))},
		{"min(10%, 20%)", edges(false, d(10, "%"), d(10, "%"), d(10, "%"), d(10, "%"))},
		{"clamp(0%, 50%, 100%) fill", edges(true, d(50, "%"), d(50, "%"), d(50, "%"), d(50, "%"))},
	}
	for _, prop := range []string{"border-image-slice", "mask-border-slice"} {
		for _, tt := range tests {
			assertValue(t, prop+": "+tt.rule, tt.want)
		}
		for _, rule := range []string{
			"none", "1, 2", "-10", "-10%", "1 2 3 -10%", "-0.3", "1 fill 2", "fill 1 2 3 fill",
			"scale(calc(0))", "min(0px, 0)", "rgb(calc(0%) 0 0)",
		} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func TestBorderImageWidth(t *testing.T) {
	tests := []struct {
		rule string
		want props.LengthQuad
	}{
		{"1", quad(num(1))},
		{"1 2    3 4", props.LengthQuad{num(1), num(2), num(3), num(4)}},
		{"50% 1000.1 0", props.LengthQuad{pct(50), num(1000.1), num(0), num(1000.1)}},
		{"1% 2px 3em 4", props.LengthQuad{pct(1), px(2), dim(3, "em"), num(4)}},
		{"auto", quad(kw("auto"))},
		{"1 auto", props.LengthQuad{num(1), kw("auto"), num(1), kw("auto")}},
		{"auto auto", quad(kw("auto"))},
		{"auto auto auto 2", props.LengthQuad{kw("auto"), kw("auto"), kw("auto"), num(2)}},
	}
	for _, prop := range []string{"border-image-width", "mask-border-width"} {
		for _, tt := range tests {
			assertValue(t, prop+": "+tt.rule, tt.want)
		}
		for _, rule := range []string{
			"none", "1, 2", "1 -2", "-10", "-10%", "1px 2px 3px -10%", "-3px",
			"auto auto auto auto auto", "1 2 3 4 5",
		} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func TestBorderImageOutset(t *testing.T) {
	tests := []struct {
		rule string
		want props.LengthQuad
	}{
		{"1", quad(num(1))},
		{"1 2    3 4", props.LengthQuad{num(1), num(2), num(3), num(4)}},
		{"50px 1000.1 0", props.LengthQuad{px(50), num(1000.1), num(0), num(1000.1)}},
		{"1in 2px 3em 4", props.LengthQuad{dim(1, "in"), px(2), dim(3, "em"), num(4)}},
	}
	for _, prop := range []string{"border-image-outset", "mask-border-outset"} {
		for _, tt := range tests {
			assertValue(t, prop+": "+tt.rule, tt.want)
		}
		for _, rule := range []string{
			"none", "auto", "1, 2", "-10", "1 -2", "10%", "1px 2px 3px -10px", "-3px", "1 2 3 4 5",
		} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func TestBorderImageRepeat(t *testing.T) {
	for _, prop := range []string{"border-image-repeat", "mask-border-repeat"} {
		assertValue(t, prop+": stretch", props.KeywordPair{"stretch", "stretch"})
		assertValue(t, prop+": repeat repeat", props.KeywordPair{"repeat", "repeat"})
		assertValue(t, prop+": round     space", props.KeywordPair{"round", "space"})
		for _, rule := range []string{"none", "test", "round round round", "stretch space round", "repeat test"} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func TestMaskBorderMode(t *testing.T) {
	assertValue(t, "mask-border-mode: alpha", props.Keyword("alpha"))
	assertValue(t, "mask-border-mode: luminance", props.Keyword("luminance"))
	assertValue(t, "mask-border-mode: alpha     ", props.Keyword("alpha"))
	for _, rule := range []string{"none", "test", "alpha alpha", "alpha luminance"} {
		assertInvalid(t, "mask-border-mode: "+rule)
	}
}

func TestBorderImageSliceWithUnresolvableMathWarns(t *testing.T) {
	for _, tt := range []struct{ prop, value string }{
		{"border-image-slice", "scale(calc(0))"},
		{"border-image-slice", "min(0px, 0)"},
		{"mask-border-slice", "scale(calc(0))"},
		{"mask-border-slice", "rgb(calc(0%) 0 0)"},
	} {
		assertInvalid(t, tt.prop+": "+tt.value, "invalid value")
	}
}

func TestStringSet(t *testing.T) {
	str := func(s string) props.ContentItem { return props.ContentItem{Type: "string", Args: []string{s}} }
	attr := func(a ...string) props.ContentItem { return props.ContentItem{Type: "attr()", Args: a} }
	counter := func(name, style string) props.ContentItem {
		return props.ContentItem{Type: "counter()", Args: []string{name}, Style: &props.ListStyleType{Name: style}}
	}
	content := func(k string) props.ContentItem { return props.ContentItem{Type: "content()", Args: []string{k}} }
	entry := func(name string, items ...props.ContentItem) props.StringSetEntry {
		return props.StringSetEntry{Name: name, Content: items}
	}
	tests := []struct {
		rule string
		want props.StringSet
	}{
		{"test content(text)", props.StringSet{entry("test", content("text"))}},
		{"test content(before)", props.StringSet{entry("test", content("before"))}},
		{`test "string"`, props.StringSet{entry("test", str("string"))}},
		{`test1 "string", test2 "string"`, props.StringSet{entry("test1", str("string")), entry("test2", str("string"))}},
		{"test attr(class)", props.StringSet{entry("test", attr("class", "string", ""))}},
		{"test attr(class url)", props.StringSet{entry("test", attr("class", "url", ""))}},
		{`test attr(class, "test")`, props.StringSet{entry("test", attr("class", "string", "test"))}},
		{`test attr(class string, "test")`, props.StringSet{entry("test", attr("class", "string", "test"))}},
		{"test counter(count)", props.StringSet{entry("test", counter("count", "decimal"))}},
		{"test counter(count, upper-roman)", props.StringSet{entry("test", counter("count", "upper-roman"))}},
		{`test counters(count, ".")`, props.StringSet{entry("test", props.ContentItem{
			Type: "counters()", Args: []string{"count", "."}, Style: &props.ListStyleType{Name: "decimal"}})}},
		{`test counters(count, ".", upper-roman)`, props.StringSet{entry("test", props.ContentItem{
			Type: "counters()", Args: []string{"count", "."}, Style: &props.ListStyleType{Name: "upper-roman"}})}},
		{`test content(text) "string" attr(title) attr(title) counter(count)`, props.StringSet{entry("test",
			content("text"), str("string"), attr("title", "string", ""), attr("title", "string", ""), counter("count", "decimal"))}},
		{"none", props.StringSet{}},
	}
	for _, tt := range tests {
		assertValue(t, "string-set: "+tt.rule, tt.want)
	}
	for _, rule := range []string{"test", "test test1", "test content(test)", "test unknown()"} {
		assertInvalid(t, "string-set: "+rule)
	}
}

func TestOverflowWrap(t *testing.T) {
	assertValue(t, "overflow-wrap: normal", props.Keyword("normal"))
	assertValue(t, "overflow-wrap: break-word", props.Keyword("break-word"))
	assertValue(t, "overflow-wrap: inherit", props.Keyword("inherit"))
	assertInvalid(t, "overflow-wrap: none")
	assertInvalid(t, "overflow-wrap: normal, break-word")
}

func TestGridAutoColumnsAndRows(t *testing.T) {
	tests := []struct {
		rule string
		want props.TrackList
	}{
		{"40px", props.TrackList{{Breadth: px(40)}}},
		{"2fr", props.TrackList{{Breadth: fr(2)}}},
		{"18%", props.TrackList{{Breadth: pct(18)}}},
		{"auto", props.TrackList{{Breadth: kw("auto")}}},
		{"min-content", props.TrackList{{Breadth: kw("min-content")}}},
		{"max-content", props.TrackList{{Breadth: kw("max-content")}}},
		{"fit-content(20%)", props.TrackList{{Function: "fit-content()", Breadth: pct(20)}}},
		{"minmax(20px, 25px)", props.TrackList{{Function: "minmax()", Min: px(20), Max: px(25)}}},
		{"minmax(min-content, max-content)", props.TrackList{{Function: "minmax()", Min: kw("min-content"), Max: kw("max-content")}}},
		{"min-content max-content", props.TrackList{{Breadth: kw("min-content")}, {Breadth: kw("max-content")}}},
	}
	for _, prop := range []string{"grid-auto-columns", "grid-auto-rows"} {
		for _, tt := range tests {
			assertValue(t, prop+": "+tt.rule, tt.want)
		}
		for _, rule := range []string{
			"40", "coucou", "fit-content", "fit-content(min-content)", "minmax(40px)", "minmax(2fr, 1fr)",
			"1fr 1fr coucou", "fit-content()", "fit-content(2%, 18%)",
		} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func TestGridAutoFlow(t *testing.T) {
	for rule, want := range map[string]props.Keywords{
		"row": kws("row"), "column": kws("column"), "row dense": kws("row", "dense"),
		"column dense": kws("column", "dense"), "dense row": kws("dense", "row"),
		"dense column": kws("dense", "column"), "dense": kws("dense", "row"),
	} {
		assertValue(t, "grid-auto-flow: "+rule, want)
	}
	for _, rule := range []string{
		"row row", "column column", "dense dense", "coucou", "row column", "column row", "row coucou",
		"column coucou", "coucou row", "coucou column", "row column dense",
	} {
		assertInvalid(t, "grid-auto-flow: "+rule)
	}
}

func TestGridTemplateColumnsAndRows(t *testing.T) {
	tests := []struct {
		rule string
		want props.GridTemplate
	}{
		{"none", noneTemplate},
		{"subgrid", subgrid()},
		{"subgrid [a] repeat(auto-fill, [b]) [c]", subgrid(names("a"), repeatItem(0, "auto-fill", names("b")), names("c"))},
		{"subgrid [a] [a] [a] [a] repeat(auto-fill, [b]) [c] [c]", subgrid(
			names("a"), names("a"), names("a"), names("a"), repeatItem(0, "auto-fill", names("b")), names("c"), names("c"))},
		{"subgrid [] [a]", subgrid(names(), names("a"))},
		{"subgrid [a] [b] [c] [d] [e] [f]", subgrid(names("a"), names("b"), names("c"), names("d"), names("e"), names("f"))},
		{"[outer-edge] 20px [main-start] 1fr [center] 1fr max-content [main-end]", template(
			names("outer-edge"), trackItem(px(20)), names("main-start"), trackItem(fr(1)), names("center"),
			trackItem(fr(1)), names(), trackItem(kw("max-content")), names("main-end"))},
		{"repeat(auto-fill, minmax(25ch, 1fr))", template(names(), repeatItem(0, "auto-fill", names(),
			props.GridItem{Kind: props.GridTrack, Track: props.TrackSize{Function: "minmax()", Min: dim(25, "ch"), Max: fr(1)}}, names()), names())},
		{"[a] auto [b] minmax(min-content, 1fr) [b c d] repeat(2, [e] 40px) repeat(5, auto)", template(
			names("a"), trackItem(kw("auto")), names("b"),
			props.GridItem{Kind: props.GridTrack, Track: props.TrackSize{Function: "minmax()", Min: kw("min-content"), Max: fr(1)}},
			names("b", "c", "d"), repeatItem(2, "", names("e"), trackItem(px(40)), names()),
			names(), repeatItem(5, "", names(), trackItem(kw("auto")), names()), names())},
	}
	for _, prop := range []string{"grid-template-columns", "grid-template-rows"} {
		for _, tt := range tests {
			assertValue(t, prop+": "+tt.rule, tt.want)
		}
		for _, rule := range []string{
			"coucou", "subgrid subgrid", "subgrid coucou", "subgrid [coucou] repeat(0, [wow])",
			"subgrid [coucou] repeat(auto-fit [wow])", "fit-content(18%) repeat(auto-fill, 15em)", "[coucou] [wow]",
		} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func TestGridTemplateAreas(t *testing.T) {
	tests := []struct {
		rule string
		want props.GridAreas
	}{
		{"none", noneAreas},
		{`"head head" "nav main" "foot ...."`, props.GridAreas{Rows: [][]string{{"head", "head"}, {"nav", "main"}, {"foot", ""}}}},
		{`"title board" "stats board"`, props.GridAreas{Rows: [][]string{{"title", "board"}, {"stats", "board"}}}},
		{`". a" "b a" ".a"`, props.GridAreas{Rows: [][]string{{"", "a"}, {"b", "a"}, {"", "a"}}}},
		{`"a b b" "c b b" "d e f"`, props.GridAreas{Rows: [][]string{{"a", "b", "b"}, {"c", "b", "b"}, {"d", "e", "f"}}}},
	}
	for _, tt := range tests {
		assertValue(t, "grid-template-areas: "+tt.rule, tt.want)
	}
	for _, rule := range []string{
		`"head head coucou" "nav main" "foot ...."`, `". a" "b c" ". a"`, `". a" "b a" "a a"`,
		`"a a a a" "a b b a" "a a a a"`, `" "`,
	} {
		assertInvalid(t, "grid-template-areas: "+rule)
	}
}

func TestGridLine(t *testing.T) {
	tests := []struct {
		rule string
		want props.GridLine
	}{
		{"auto", props.GridLine{Auto: true}},
		{"4", props.GridLine{N: 4}},
		{"C", props.GridLine{Ident: "C"}},
		{"4 c", props.GridLine{N: 4, Ident: "c"}},
		{"col -4", props.GridLine{N: -4, Ident: "col"}},
		{"span c 4", props.GridLine{Span: true, N: 4, Ident: "c"}},
		{"span 4 c", props.GridLine{Span: true, N: 4, Ident: "c"}},
		{"4 span C", props.GridLine{Span: true, N: 4, Ident: "C"}},
		{"super 4 span", props.GridLine{Span: true, N: 4, Ident: "super"}},
	}
	for _, prop := range []string{"grid-row-start", "grid-row-end", "grid-column-start", "grid-column-end"} {
		for _, tt := range tests {
			assertValue(t, prop+": "+tt.rule, tt.want)
		}
		for _, rule := range []string{
			"span", "0", "1.1", "span 0", "span -1", "span 2.1", "span auto", "auto auto", "-4 cOL span", "span 1.1 col",
		} {
			assertInvalid(t, prop+": "+rule)
		}
	}
}

func alignmentCases(t *testing.T, prop string, valid map[string]props.Keywords, invalid []string) {
	t.Helper()
	for rule, want := range valid {
		assertValue(t, prop+": "+rule, want)
	}
	for _, rule := range invalid {
		assertInvalid(t, prop+": "+rule)
	}
}

func TestAlignContent(t *testing.T) {
	valid := map[string]props.Keywords{
		"normal": kws("normal"), "baseline": kws("first", "baseline"), "first baseline": kws("first", "baseline"),
		"last baseline": kws("last", "baseline"), "baseline last": kws("baseline", "last"),
		"space-between": kws("space-between"), "space-around": kws("space-around"), "space-evenly": kws("space-evenly"),
		"stretch": kws("stretch"), "center": kws("center"), "start": kws("start"), "end": kws("end"),
		"flex-start": kws("flex-start"), "flex-end": kws("flex-end"), "safe center": kws("safe", "center"),
		"unsafe start": kws("unsafe", "start"), "safe end": kws("safe", "end"),
		"safe flex-start": kws("safe", "flex-start"), "unsafe flex-start": kws("unsafe", "flex-start"),
	}
	alignmentCases(t, "align-content", valid, []string{
		"auto", "none", "auto auto", "first last", "baseline baseline", "start safe", "start end", "safe unsafe", "left", "right",
	})
}

func itemsValid(extra map[string]props.Keywords) map[string]props.Keywords {
	valid := map[string]props.Keywords{
		"normal": kws("normal"), "stretch": kws("stretch"), "baseline": kws("first", "baseline"),
		"first baseline": kws("first", "baseline"), "last baseline": kws("last", "baseline"),
		"baseline last": kws("baseline", "last"), "center": kws("center"), "self-start": kws("self-start"),
		"self-end": kws("self-end"), "start": kws("start"), "end": kws("end"), "flex-start": kws("flex-start"),
		"flex-end": kws("flex-end"), "safe center": kws("safe", "center"), "unsafe start": kws("unsafe", "start"),
		"safe end": kws("safe", "end"), "unsafe self-start": kws("unsafe", "self-start"),
		"safe self-end": kws("safe", "self-end"), "safe flex-start": kws("safe", "flex-start"),
		"unsafe flex-start": kws("unsafe", "flex-start"),
	}
	for k, v := range extra {
		valid[k] = v
	}
	return valid
}

func TestAlignItems(t *testing.T) {
	alignmentCases(t, "align-items", itemsValid(nil), []string{
		"auto", "none", "auto auto", "first last", "baseline baseline", "start safe", "start end",
		"safe unsafe", "left", "right", "space-between",
	})
}

func TestAlignSelf(t *testing.T) {
	alignmentCases(t, "align-self", itemsValid(map[string]props.Keywords{"auto": kws("auto")}), []string{
		"none", "auto auto", "first last", "baseline baseline", "start safe", "start end", "safe unsafe",
		"left", "right", "space-between",
	})
}

func TestJustifyContent(t *testing.T) {
	valid := map[string]props.Keywords{
		"normal": kws("normal"), "space-between": kws("space-between"), "space-around": kws("space-around"),
		"space-evenly": kws("space-evenly"), "stretch": kws("stretch"), "center": kws("center"),
		"left": kws("left"), "right": kws("right"), "start": kws("start"), "end": kws("end"),
		"flex-start": kws("flex-start"), "flex-end": kws("flex-end"), "safe center": kws("safe", "center"),
		"unsafe start": kws("unsafe", "start"), "safe end": kws("safe", "end"), "unsafe left": kws("unsafe", "left"),
		"safe right": kws("safe", "right"), "safe flex-start": kws("safe", "flex-start"),
		"unsafe flex-start": kws("unsafe", "flex-start"),
	}
	alignmentCases(t, "justify-content", valid, []string{
		"auto", "none", "baseline", "auto auto", "first last", "baseline baseline", "start safe", "start end", "safe unsafe",
	})
}

func TestJustifyItems(t *testing.T) {
	valid := itemsValid(map[string]props.Keywords{
		"left": kws("left"), "right": kws("right"), "legacy": kws("legacy"), "legacy left": kws("legacy", "left"),
		"left legacy": kws("left", "legacy"), "legacy center": kws("legacy", "center"),
	})
	alignmentCases(t, "justify-items", valid, []string{
		"auto", "none", "auto auto", "first last", "baseline baseline", "start safe", "start end",
		"safe unsafe", "space-between",
	})
}

func TestJustifySelf(t *testing.T) {
	valid := itemsValid(map[string]props.Keywords{
		"auto": kws("auto"), "left": kws("left"), "right": kws("right"),
		"unsafe left": kws("unsafe", "left"), "safe right": kws("safe", "right"),
	})
	alignmentCases(t, "justify-self", valid, []string{
		"none", "auto auto", "first last", "baseline baseline", "start safe", "start end", "safe unsafe", "space-between",
	})
}

func TestInvalidColors(t *testing.T) {
	for _, prop := range []string{"color", "background", "background-color", "border-color", "border-left-color", "outline-color"} {
		for _, value := range []string{"rgb()", "device-cmyk(0%, 0%, 0%, 30%)", "#abcde"} {
			assertInvalid(t, prop+": "+value)
		}
	}
}

func TestColorScheme(t *testing.T) {
	tests := map[string]props.Keywords{
		"normal": kws("normal"), "light": kws("light"), "dark": kws("dark"), "light dark": kws("light", "dark"),
		"dark light": kws("dark", "light"), "light only": kws("light", "only"), "only light": kws("light", "only"),
		"dark dark": kws("dark", "dark"), "light something": kws("light", "something"),
		"only dark light": kws("dark", "light", "only"), "only something light": kws("something", "light", "only"),
	}
	for rule, want := range tests {
		assertValue(t, "color-scheme: "+rule, want)
	}
	for _, rule := range []string{"normal only", "normal something", "only", "only only", "light only dark"} {
		assertInvalid(t, "color-scheme: "+rule)
	}
}

func TestBoxShadow(t *testing.T) {
	black := func(name string) props.Color {
		c, _ := syntax.ParseColorString(name)
		return props.Color{Color: c}
	}
	shadow := func(x, y, blur, spread props.Length, color props.Color, inset bool) props.BoxShadow {
		return props.BoxShadow{X: x, Y: y, Blur: blur, Spread: spread, Color: color, Inset: inset}
	}
	z := px(0)
	tests := []struct {
		rule string
		want props.BoxShadows
	}{
		{"none", props.BoxShadows{}},
		{"1px 10px", props.BoxShadows{shadow(px(1), px(10), z, z, currentColor, false)}},
		{"1px 10px 0 3cm", props.BoxShadows{shadow(px(1), px(10), num(0), dim(3, "cm"), currentColor, false)}},
		{"1px 10px inset", props.BoxShadows{shadow(px(1), px(10), z, z, currentColor, true)}},
		{"1px 10px 2px 5cm inset", props.BoxShadows{shadow(px(1), px(10), px(2), dim(5, "cm"), currentColor, true)}},
		{"black 1px 10px 2px 5cm", props.BoxShadows{shadow(px(1), px(10), px(2), dim(5, "cm"), black("black"), false)}},
		{"red 1px 10px inset, blue 1px 10px inset", props.BoxShadows{
			shadow(px(1), px(10), z, z, black("red"), true), shadow(px(1), px(10), z, z, black("blue"), true)}},
		{"red 1px 10px 2px 5cm", props.BoxShadows{shadow(px(1), px(10), px(2), dim(5, "cm"), black("red"), false)}},
		{"inset red 1px 10px 2px 5cm", props.BoxShadows{shadow(px(1), px(10), px(2), dim(5, "cm"), black("red"), true)}},
		{"1px 10px 2px 5cm red inset", props.BoxShadows{shadow(px(1), px(10), px(2), dim(5, "cm"), black("red"), true)}},
		{"1px 10px, 2px 4px #ff0000", props.BoxShadows{
			shadow(px(1), px(10), z, z, currentColor, false), shadow(px(2), px(4), z, z, black("red"), false)}},
	}
	for _, tt := range tests {
		assertValue(t, "box-shadow: "+tt.rule, tt.want)
	}
	for _, rule := range []string{
		"1px 10px,", "1px 10px,,1px", "1px red inset", "1px 1px red blue inset", "blue 1px 1px red",
		"inset 1px 1px 1px blue inset", "1px 10px red 3px", "1px 10px inset 3px", "1px 1px 1px inset inset",
		"1px 0 0 3%", "1px 0, none", "none, none",
	} {
		assertInvalid(t, "box-shadow: "+rule)
	}
}

func TestLightDarkColorKeepsItsTokens(t *testing.T) {
	v := getValue(t, "color: light-dark(red, blue)", "")
	c, ok := v.(props.Color)
	if !ok || c.Token == nil || !approx(c.Coordinates[0], 1) {
		t.Errorf("got %#v, want the light color with its token", v)
	}
	if plain, _ := getValue(t, "color: red", "").(props.Color); plain.Token != nil {
		t.Errorf("a plain color should not keep its token")
	}
}
