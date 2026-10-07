package props

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// knownProperties lists the properties the engine supports, in table order.
var knownProperties = []string{
	"bottom", "caption-side", "clear", "clip", "color", "direction",
	"display", "empty-cells", "float", "left", "line-height", "margin-top",
	"margin-right", "margin-bottom", "margin-left", "padding-top", "padding-right", "padding-bottom",
	"padding-left", "position", "right", "table-layout", "top", "unicode-bidi",
	"vertical-align", "visibility", "z-index", "background-attachment", "background-clip", "background-color",
	"background-image", "background-origin", "background-position", "background-repeat", "background-size", "border-bottom-color",
	"border-bottom-left-radius", "border-bottom-right-radius", "border-bottom-style", "border-bottom-width", "border-collapse", "border-left-color",
	"border-left-style", "border-left-width", "border-right-color", "border-right-style", "border-right-width", "border-spacing",
	"border-top-color", "border-top-left-radius", "border-top-right-radius", "border-top-style", "border-top-width", "border-image-source",
	"border-image-slice", "border-image-width", "border-image-outset", "border-image-repeat", "box-shadow", "mask-border-source",
	"mask-border-slice", "mask-border-width", "mask-border-outset", "mask-border-repeat", "mask-border-mode", "color-scheme",
	"opacity", "column-width", "column-count", "column-rule-color", "column-rule-style", "column-rule-width",
	"column-fill", "column-span", "font-family", "font-feature-settings", "font-kerning", "font-language-override",
	"font-size", "font-stretch", "font-style", "font-variant", "font-variant-alternates", "font-variant-caps",
	"font-variant-east-asian", "font-variant-ligatures", "font-variant-numeric", "font-variant-position", "font-weight", "font-variation-settings",
	"box-decoration-break", "break-after", "break-before", "break-inside", "margin-break", "orphans",
	"widows", "bookmark-label", "bookmark-level", "bookmark-state", "content", "footnote-display",
	"footnote-policy", "quotes", "string-set", "image-resolution", "image-rendering", "image-orientation",
	"object-fit", "object-position", "size", "page", "bleed-left", "bleed-right",
	"bleed-top", "bleed-bottom", "marks", "hyphenate-character", "hyphenate-limit-chars", "hyphenate-limit-zone",
	"hyphens", "letter-spacing", "tab-size", "text-align-all", "text-align-last", "text-indent",
	"text-transform", "white-space", "word-break", "word-spacing", "transform-origin", "transform",
	"appearance", "outline-color", "outline-style", "outline-width", "outline-offset", "box-sizing",
	"height", "max-height", "max-width", "min-height", "min-width", "width",
	"block-size", "inline-size", "max-block-size", "max-inline-size", "min-block-size", "min-inline-size",
	"margin-block-start", "margin-inline-start", "margin-block-end", "margin-inline-end", "padding-block-start", "padding-inline-start",
	"padding-block-end", "padding-inline-end", "inset-block-start", "inset-inline-start", "inset-block-end", "inset-inline-end",
	"border-block-start-width", "border-inline-start-width", "border-block-end-width", "border-inline-end-width", "border-block-start-style", "border-inline-start-style",
	"border-block-end-style", "border-inline-end-style", "border-block-start-color", "border-inline-start-color", "border-block-end-color", "border-inline-end-color",
	"border-start-start-radius", "border-start-end-radius", "border-end-start-radius", "border-end-end-radius", "flex-basis", "flex-direction",
	"flex-grow", "flex-shrink", "flex-wrap", "grid-auto-columns", "grid-auto-flow", "grid-auto-rows",
	"grid-template-areas", "grid-template-columns", "grid-template-rows", "grid-row-start", "grid-column-start", "grid-row-end",
	"grid-column-end", "align-content", "align-items", "align-self", "justify-content", "justify-items",
	"justify-self", "order", "column-gap", "row-gap", "text-decoration-line", "text-decoration-color",
	"text-decoration-style", "text-decoration-thickness", "text-underline-offset", "block-ellipsis", "continue", "max-lines",
	"overflow", "overflow-wrap", "text-overflow", "counter-increment", "counter-reset", "counter-set",
	"list-style-image", "list-style-position", "list-style-type", "anchor", "lang",
}

var inheritedProperties = []string{
	"block-ellipsis",
	"border-collapse",
	"border-spacing",
	"caption-side",
	"color",
	"color-scheme",
	"direction",
	"empty-cells",
	"font-family",
	"font-feature-settings",
	"font-kerning",
	"font-language-override",
	"font-size",
	"font-stretch",
	"font-style",
	"font-variant",
	"font-variant-alternates",
	"font-variant-caps",
	"font-variant-east-asian",
	"font-variant-ligatures",
	"font-variant-numeric",
	"font-variant-position",
	"font-variation-settings",
	"font-weight",
	"hyphenate-character",
	"hyphenate-limit-chars",
	"hyphenate-limit-zone",
	"hyphens",
	"image-rendering",
	"image-resolution",
	"lang",
	"letter-spacing",
	"line-height",
	"list-style-image",
	"list-style-position",
	"list-style-type",
	"orphans",
	"overflow-wrap",
	"quotes",
	"tab-size",
	"text-align-all",
	"text-align-last",
	"text-indent",
	"text-transform",
	"text-underline-offset",
	"visibility",
	"white-space",
	"widows",
	"word-break",
	"word-spacing",
}

var initialNotComputedProperties = []string{
	"bleed-bottom",
	"bleed-left",
	"bleed-right",
	"bleed-top",
	"border-block-end-width",
	"border-block-start-width",
	"border-bottom-width",
	"border-inline-end-width",
	"border-inline-start-width",
	"border-left-width",
	"border-right-width",
	"border-top-width",
	"column-gap",
	"column-rule-width",
	"display",
	"outline-width",
}

var tableWrapperProperties = []string{
	"bottom",
	"break-after",
	"break-before",
	"clear",
	"counter-increment",
	"counter-reset",
	"counter-set",
	"float",
	"left",
	"margin-block-end",
	"margin-block-start",
	"margin-bottom",
	"margin-inline-end",
	"margin-inline-start",
	"margin-left",
	"margin-right",
	"margin-top",
	"opacity",
	"overflow",
	"position",
	"right",
	"top",
	"transform",
	"transform-origin",
	"vertical-align",
	"z-index",
}

func flagged(pick func(PropInfo) bool) []string {
	var names []string
	for _, p := range Table {
		if pick(p) {
			names = append(names, p.Name)
		}
	}
	slices.Sort(names)
	return names
}

func TestTableListsEveryKnownProperty(t *testing.T) {
	var names []string
	for _, p := range Table {
		names = append(names, p.Name)
	}
	if !slices.Equal(names, knownProperties) {
		t.Fatalf("table properties differ from the known list: got %d entries, want %d", len(names), len(knownProperties))
	}
}

func TestTablePropertyFlags(t *testing.T) {
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{"inherited", flagged(func(p PropInfo) bool { return p.Inherited }), inheritedProperties},
		{"initial not computed", flagged(func(p PropInfo) bool { return p.InitialNotComputed }), initialNotComputedProperties},
		{"table wrapper", flagged(func(p PropInfo) bool { return p.TableWrapper }), tableWrapperProperties},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got, tt.want) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestEveryPropertyHasAnInitialValue(t *testing.T) {
	for _, p := range Table {
		if p.Initial == nil {
			t.Errorf("%s: no initial value", p.Name)
		}
		if p.Validate != nil || p.Compute != nil {
			t.Errorf("%s: validator and computer are not expected yet", p.Name)
		}
	}
}

func TestIdentifiersMatchTable(t *testing.T) {
	if NumProps() != len(Table) {
		t.Fatalf("NumProps() = %d, want %d", NumProps(), len(Table))
	}
	for i, p := range Table {
		id, ok := ID(p.Name)
		if !ok || int(id) != i {
			t.Errorf("ID(%q) = %d, %v; want %d", p.Name, id, ok, i)
			continue
		}
		if id.Name() != p.Name || id.Info() != &Table[i] {
			t.Errorf("%s: name or info mismatch", p.Name)
		}
		if id.Inherited() != p.Inherited ||
			id.InitialNotComputed() != p.InitialNotComputed ||
			id.TableWrapper() != p.TableWrapper {
			t.Errorf("%s: generated flags differ from the table", p.Name)
		}
		if !reflect.DeepEqual(id.Initial(), p.Initial) {
			t.Errorf("%s: generated initial value %#v differs from the table's %#v", p.Name, id.Initial(), p.Initial)
		}
	}
}

func TestIDIsCaseInsensitive(t *testing.T) {
	want, _ := ID("margin-top")
	if got, ok := ID("Margin-TOP"); !ok || got != want || want != PropMarginTop {
		t.Errorf("ID(Margin-TOP) = %d, %v; want %d", got, ok, want)
	}
	if _, ok := ID("margin"); ok {
		t.Error("shorthand margin must not be a property")
	}
}

func TestInitialValues(t *testing.T) {
	tests := []struct {
		id   PropID
		want Value
	}{
		{PropColor, Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 1)}},
		{PropBackgroundColor, Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)}},
		{PropBorderTopColor, Color{Color: syntax.Color{Space: syntax.CurrentColor, Alpha: 1}}},
		{PropDisplay, Display{"inline", "flow"}},
		{PropFontSize, Pixels(16)},
		{PropFontWeight, Integer(400)},
		{PropBorderTopWidth, Pixels(3)},
		{PropWidth, Length{Keyword: "auto"}},
		{PropMarginLeft, Px(0)},
		{PropOpacity, Number(1)},
		{PropOrphans, Integer(2)},
		{PropZIndex, IntOrKeyword{Keyword: "auto"}},
		{PropTabSize, LengthNumber(8)},
		{PropFontFamily, Strings{"serif"}},
		{PropHyphenateLimitChars, IntTriple{5, 2, 2}},
		{PropCounterIncrement, Counters{Auto: true}},
		{PropBackgroundImage, Images{{Type: "none"}}},
		{PropBackgroundSize, BackgroundSizes{{Width: Length{Keyword: "auto"}, Height: Length{Keyword: "auto"}}}},
		{PropTransformOrigin, LengthPair{
			Length{Dimension: Dimension{50, "%"}},
			Length{Dimension: Dimension{50, "%"}},
		}},
		{PropGridTemplateColumns, GridTemplate{None: true}},
		{PropPosition, Positioning{Keyword: "static"}},
	}
	for _, tt := range tests {
		t.Run(tt.id.Name(), func(t *testing.T) {
			if got := tt.id.Initial(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestInitialPageSizeIsA4InPixels(t *testing.T) {
	got, ok := PropSize.Initial().(PixelPair)
	if !ok {
		t.Fatalf("size has initial value %#v", PropSize.Initial())
	}
	if math.Abs(float64(got.X)-793.7007874) > 1e-6 || math.Abs(float64(got.Y)-1122.5196850) > 1e-6 {
		t.Errorf("got %v", got)
	}
}

func TestMaxSizesAreUnbounded(t *testing.T) {
	for _, id := range []PropID{PropMaxWidth, PropMaxHeight, PropMaxBlockSize, PropMaxInlineSize} {
		if l, ok := id.Initial().(Length); !ok || !l.IsInfinite() {
			t.Errorf("%s: initial value %#v is not unbounded", id.Name(), id.Initial())
		}
	}
}
