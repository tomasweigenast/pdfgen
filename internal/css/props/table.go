package props

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// PropInfo describes a CSS property.
type PropInfo struct {
	// Name is the property name as written in CSS, such as "margin-top".
	Name string
	// Initial is the initial value. For properties flagged
	// InitialNotComputed it is a specified value that still has to go
	// through Compute; for the others it is already the computed value.
	Initial Value
	// Inherited reports whether the computed value is inherited by default.
	Inherited bool
	// InitialNotComputed marks properties whose initial value computes to
	// something that depends on the style, so it cannot be used as is.
	InitialNotComputed bool
	// TableWrapper marks the non-inherited properties that apply to the
	// wrapper box of a table instead of the table box (CSS 2.1 §17.4).
	TableWrapper bool
	// Validate parses the tokens of a declaration into a specified value.
	Validate func(tokens []syntax.Node, baseURL string) (Value, error)
	// Compute turns a specified value into a computed value.
	Compute func(ctx ComputeContext, v Value) Value
}

// ComputeContext is the style a value is computed for, as seen by the
// property computers. It is implemented by the computed package, which
// depends on this one.
type ComputeContext interface {
	// Get returns the computed value of a property of the same element.
	Get(id PropID) Value
	// Specified returns the specified value of position, float or display,
	// which the computed values of float and display depend on.
	Specified(id PropID) Value
	// ParentValue returns the computed value of a property of the parent
	// element, or its initial value for the root element.
	ParentValue(id PropID) Value
	// IsRootElement reports whether the style is the one of the root
	// element.
	IsRootElement() bool
	// PseudoType is the pseudo-element the style is for, or "".
	PseudoType() string
	// ToPixels converts a length to CSS pixels in the context of the named
	// property, resolving font and viewport relative units.
	ToPixels(value float64, unit, property string) float64
	// CharacterRatio returns the size of the ex, ch, cap or ic unit as a
	// fraction of the font size.
	CharacterRatio(unit string) float64
	// Strut returns the used line height and the baseline position of the
	// first available font.
	Strut() (lineHeight, baseline float64)
}

// mmToPx is the size of a millimetre in CSS pixels (CSS Values 3 §5.2).
const mmToPx = 96 / 25.4

var (
	black        = Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 1)}
	transparent  = Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)}
	currentColor = Color{Color: syntax.Color{Space: syntax.CurrentColor, Alpha: 1}}

	// unbounded is the parsed value of none for the max-* properties.
	unbounded = Px(math.Inf(1))
)

// Table lists every supported property. The position of a property in the
// table is its PropID; props_gen.go and the typed getters of the computed
// package are generated from it.
var Table = []PropInfo{
	{Name: "bottom", Initial: LengthKeyword("auto"), TableWrapper: true},
	{Name: "caption-side", Initial: Keyword("top"), Inherited: true},
	{Name: "clear", Initial: Keyword("none"), TableWrapper: true},
	{Name: "clip", Initial: Clip{}},
	{Name: "color", Initial: black, Inherited: true},
	{Name: "direction", Initial: Keyword("ltr"), Inherited: true},
	{Name: "display", Initial: Display{"inline", "flow"}, InitialNotComputed: true},
	{Name: "empty-cells", Initial: Keyword("show"), Inherited: true},
	{Name: "float", Initial: Keyword("none"), TableWrapper: true},
	{Name: "left", Initial: LengthKeyword("auto"), TableWrapper: true},
	{Name: "line-height", Initial: LengthKeyword("normal"), Inherited: true},
	{Name: "margin-top", Initial: Px(0), TableWrapper: true},
	{Name: "margin-right", Initial: Px(0), TableWrapper: true},
	{Name: "margin-bottom", Initial: Px(0), TableWrapper: true},
	{Name: "margin-left", Initial: Px(0), TableWrapper: true},
	{Name: "padding-top", Initial: Px(0)},
	{Name: "padding-right", Initial: Px(0)},
	{Name: "padding-bottom", Initial: Px(0)},
	{Name: "padding-left", Initial: Px(0)},
	{Name: "position", Initial: Positioning{Keyword: "static"}, TableWrapper: true},
	{Name: "right", Initial: LengthKeyword("auto"), TableWrapper: true},
	{Name: "table-layout", Initial: Keyword("auto")},
	{Name: "top", Initial: LengthKeyword("auto"), TableWrapper: true},
	{Name: "unicode-bidi", Initial: Keyword("normal")},
	{Name: "vertical-align", Initial: LengthKeyword("baseline"), TableWrapper: true},
	{Name: "visibility", Initial: Keyword("visible"), Inherited: true},
	{Name: "z-index", Initial: IntOrKeyword{Keyword: "auto"}, TableWrapper: true},
	{Name: "background-attachment", Initial: Keywords{"scroll"}},
	{Name: "background-clip", Initial: Keywords{"border-box"}},
	{Name: "background-color", Initial: transparent},
	{Name: "background-image", Initial: Images{{Type: "none"}}},
	{Name: "background-origin", Initial: Keywords{"padding-box"}},
	{Name: "background-position", Initial: Positions{{OriginX: "left", X: Length{Dimension: Dimension{Value: 0, Unit: "%"}}, OriginY: "top", Y: Length{Dimension: Dimension{Value: 0, Unit: "%"}}}}},
	{Name: "background-repeat", Initial: Repeats{{"repeat", "repeat"}}},
	{Name: "background-size", Initial: BackgroundSizes{{Width: LengthKeyword("auto"), Height: LengthKeyword("auto")}}},
	{Name: "border-bottom-color", Initial: currentColor},
	{Name: "border-bottom-left-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-bottom-right-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-bottom-style", Initial: Keyword("none")},
	{Name: "border-bottom-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-collapse", Initial: Keyword("separate"), Inherited: true},
	{Name: "border-left-color", Initial: currentColor},
	{Name: "border-left-style", Initial: Keyword("none")},
	{Name: "border-left-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-right-color", Initial: currentColor},
	{Name: "border-right-style", Initial: Keyword("none")},
	{Name: "border-right-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-spacing", Initial: PixelPair{0, 0}, Inherited: true},
	{Name: "border-top-color", Initial: currentColor},
	{Name: "border-top-left-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-top-right-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-top-style", Initial: Keyword("none")},
	{Name: "border-top-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-image-source", Initial: Image{Type: "none"}},
	{Name: "border-image-slice", Initial: ImageSlice{Edges: [4]Dimension{{100, "%"}, {100, "%"}, {100, "%"}, {100, "%"}}}},
	{Name: "border-image-width", Initial: LengthQuad{LengthNumber(1), LengthNumber(1), LengthNumber(1), LengthNumber(1)}},
	{Name: "border-image-outset", Initial: LengthQuad{LengthNumber(0), LengthNumber(0), LengthNumber(0), LengthNumber(0)}},
	{Name: "border-image-repeat", Initial: KeywordPair{"stretch", "stretch"}},
	{Name: "box-shadow", Initial: BoxShadows{}},
	{Name: "mask-border-source", Initial: Image{Type: "none"}},
	{Name: "mask-border-slice", Initial: ImageSlice{Edges: [4]Dimension{{100, "%"}, {100, "%"}, {100, "%"}, {100, "%"}}}},
	{Name: "mask-border-width", Initial: LengthQuad{LengthKeyword("auto"), LengthKeyword("auto"), LengthKeyword("auto"), LengthKeyword("auto")}},
	{Name: "mask-border-outset", Initial: LengthQuad{LengthNumber(0), LengthNumber(0), LengthNumber(0), LengthNumber(0)}},
	{Name: "mask-border-repeat", Initial: KeywordPair{"stretch", "stretch"}},
	{Name: "mask-border-mode", Initial: Keyword("alpha")},
	{Name: "color-scheme", Initial: Keywords{"normal"}, Inherited: true},
	{Name: "opacity", Initial: Number(1), TableWrapper: true},
	{Name: "column-width", Initial: LengthKeyword("auto")},
	{Name: "column-count", Initial: IntOrKeyword{Keyword: "auto"}},
	{Name: "column-rule-color", Initial: currentColor},
	{Name: "column-rule-style", Initial: Keyword("none")},
	{Name: "column-rule-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "column-fill", Initial: Keyword("balance")},
	{Name: "column-span", Initial: Keyword("none")},
	{Name: "font-family", Initial: Strings{"serif"}, Inherited: true},
	{Name: "font-feature-settings", Initial: FontSettings{Normal: true}, Inherited: true},
	{Name: "font-kerning", Initial: Keyword("auto"), Inherited: true},
	{Name: "font-language-override", Initial: KeywordOrString{Keyword: "normal"}, Inherited: true},
	{Name: "font-size", Initial: Pixels(16), Inherited: true},
	{Name: "font-stretch", Initial: Keyword("normal"), Inherited: true},
	{Name: "font-style", Initial: Keyword("normal"), Inherited: true},
	{Name: "font-variant", Initial: Keyword("normal"), Inherited: true},
	{Name: "font-variant-alternates", Initial: Keyword("normal"), Inherited: true},
	{Name: "font-variant-caps", Initial: Keyword("normal"), Inherited: true},
	{Name: "font-variant-east-asian", Initial: Keywords{"normal"}, Inherited: true},
	{Name: "font-variant-ligatures", Initial: Keywords{"normal"}, Inherited: true},
	{Name: "font-variant-numeric", Initial: Keywords{"normal"}, Inherited: true},
	{Name: "font-variant-position", Initial: Keyword("normal"), Inherited: true},
	{Name: "font-weight", Initial: Integer(400), Inherited: true},
	{Name: "font-variation-settings", Initial: FontSettings{Normal: true}, Inherited: true},
	{Name: "box-decoration-break", Initial: Keyword("slice")},
	{Name: "break-after", Initial: Keyword("auto"), TableWrapper: true},
	{Name: "break-before", Initial: Keyword("auto"), TableWrapper: true},
	{Name: "break-inside", Initial: Keyword("auto")},
	{Name: "margin-break", Initial: Keyword("auto")},
	{Name: "orphans", Initial: Integer(2), Inherited: true},
	{Name: "widows", Initial: Integer(2), Inherited: true},
	{Name: "bookmark-label", Initial: ContentList{{Type: "content()", Args: []string{"text"}}}},
	{Name: "bookmark-level", Initial: IntOrKeyword{Keyword: "none"}},
	{Name: "bookmark-state", Initial: Keyword("open")},
	{Name: "content", Initial: Content{Keyword: "normal"}},
	{Name: "footnote-display", Initial: Keyword("block")},
	{Name: "footnote-policy", Initial: Keyword("auto")},
	{Name: "quotes", Initial: Quotes{Keyword: "auto"}, Inherited: true},
	{Name: "string-set", Initial: StringSet{}},
	{Name: "image-resolution", Initial: Number(1), Inherited: true},
	{Name: "image-rendering", Initial: Keyword("auto"), Inherited: true},
	{Name: "image-orientation", Initial: ImageOrientation{Keyword: "from-image"}},
	{Name: "object-fit", Initial: Keyword("fill")},
	{Name: "object-position", Initial: Positions{{OriginX: "left", X: Length{Dimension: Dimension{Value: 50, Unit: "%"}}, OriginY: "top", Y: Length{Dimension: Dimension{Value: 50, Unit: "%"}}}}},
	{Name: "size", Initial: PixelPair{Pixels(210 * mmToPx), Pixels(297 * mmToPx)}},
	{Name: "page", Initial: Keyword("auto")},
	{Name: "bleed-left", Initial: LengthKeyword("auto"), InitialNotComputed: true},
	{Name: "bleed-right", Initial: LengthKeyword("auto"), InitialNotComputed: true},
	{Name: "bleed-top", Initial: LengthKeyword("auto"), InitialNotComputed: true},
	{Name: "bleed-bottom", Initial: LengthKeyword("auto"), InitialNotComputed: true},
	{Name: "marks", Initial: Keywords{}},
	{Name: "hyphenate-character", Initial: String("‐"), Inherited: true},
	{Name: "hyphenate-limit-chars", Initial: IntTriple{5, 2, 2}, Inherited: true},
	{Name: "hyphenate-limit-zone", Initial: Px(0), Inherited: true},
	{Name: "hyphens", Initial: Keyword("manual"), Inherited: true},
	{Name: "letter-spacing", Initial: LengthKeyword("normal"), Inherited: true},
	{Name: "tab-size", Initial: LengthNumber(8), Inherited: true},
	{Name: "text-align-all", Initial: Keyword("start"), Inherited: true},
	{Name: "text-align-last", Initial: Keyword("auto"), Inherited: true},
	{Name: "text-indent", Initial: Px(0), Inherited: true},
	{Name: "text-transform", Initial: Keyword("none"), Inherited: true},
	{Name: "white-space", Initial: Keyword("normal"), Inherited: true},
	{Name: "word-break", Initial: Keyword("normal"), Inherited: true},
	{Name: "word-spacing", Initial: Pixels(0), Inherited: true},
	{Name: "transform-origin", Initial: LengthPair{Length{Dimension: Dimension{Value: 50, Unit: "%"}}, Length{Dimension: Dimension{Value: 50, Unit: "%"}}}, TableWrapper: true},
	{Name: "transform", Initial: Transform{}, TableWrapper: true},
	{Name: "appearance", Initial: Keyword("none")},
	{Name: "outline-color", Initial: currentColor},
	{Name: "outline-style", Initial: Keyword("none")},
	{Name: "outline-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "outline-offset", Initial: Pixels(0)},
	{Name: "box-sizing", Initial: Keyword("content-box")},
	{Name: "height", Initial: LengthKeyword("auto")},
	{Name: "max-height", Initial: unbounded},
	{Name: "max-width", Initial: unbounded},
	{Name: "min-height", Initial: LengthKeyword("auto")},
	{Name: "min-width", Initial: LengthKeyword("auto")},
	{Name: "width", Initial: LengthKeyword("auto")},
	{Name: "block-size", Initial: LengthKeyword("auto")},
	{Name: "inline-size", Initial: LengthKeyword("auto")},
	{Name: "max-block-size", Initial: unbounded},
	{Name: "max-inline-size", Initial: unbounded},
	{Name: "min-block-size", Initial: LengthKeyword("auto")},
	{Name: "min-inline-size", Initial: LengthKeyword("auto")},
	{Name: "margin-block-start", Initial: Px(0), TableWrapper: true},
	{Name: "margin-inline-start", Initial: Px(0), TableWrapper: true},
	{Name: "margin-block-end", Initial: Px(0), TableWrapper: true},
	{Name: "margin-inline-end", Initial: Px(0), TableWrapper: true},
	{Name: "padding-block-start", Initial: Px(0)},
	{Name: "padding-inline-start", Initial: Px(0)},
	{Name: "padding-block-end", Initial: Px(0)},
	{Name: "padding-inline-end", Initial: Px(0)},
	{Name: "inset-block-start", Initial: LengthKeyword("auto")},
	{Name: "inset-inline-start", Initial: LengthKeyword("auto")},
	{Name: "inset-block-end", Initial: LengthKeyword("auto")},
	{Name: "inset-inline-end", Initial: LengthKeyword("auto")},
	{Name: "border-block-start-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-inline-start-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-block-end-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-inline-end-width", Initial: Pixels(3), InitialNotComputed: true},
	{Name: "border-block-start-style", Initial: Keyword("none")},
	{Name: "border-inline-start-style", Initial: Keyword("none")},
	{Name: "border-block-end-style", Initial: Keyword("none")},
	{Name: "border-inline-end-style", Initial: Keyword("none")},
	{Name: "border-block-start-color", Initial: currentColor},
	{Name: "border-inline-start-color", Initial: currentColor},
	{Name: "border-block-end-color", Initial: currentColor},
	{Name: "border-inline-end-color", Initial: currentColor},
	{Name: "border-start-start-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-start-end-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-end-start-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "border-end-end-radius", Initial: LengthPair{Px(0), Px(0)}},
	{Name: "flex-basis", Initial: LengthKeyword("auto")},
	{Name: "flex-direction", Initial: Keyword("row")},
	{Name: "flex-grow", Initial: Number(0)},
	{Name: "flex-shrink", Initial: Number(1)},
	{Name: "flex-wrap", Initial: Keyword("nowrap")},
	{Name: "grid-auto-columns", Initial: TrackList{{Breadth: LengthKeyword("auto")}}},
	{Name: "grid-auto-flow", Initial: Keywords{"row"}},
	{Name: "grid-auto-rows", Initial: TrackList{{Breadth: LengthKeyword("auto")}}},
	{Name: "grid-template-areas", Initial: GridAreas{None: true}},
	{Name: "grid-template-columns", Initial: GridTemplate{None: true}},
	{Name: "grid-template-rows", Initial: GridTemplate{None: true}},
	{Name: "grid-row-start", Initial: GridLine{Auto: true}},
	{Name: "grid-column-start", Initial: GridLine{Auto: true}},
	{Name: "grid-row-end", Initial: GridLine{Auto: true}},
	{Name: "grid-column-end", Initial: GridLine{Auto: true}},
	{Name: "align-content", Initial: Keywords{"normal"}},
	{Name: "align-items", Initial: Keywords{"normal"}},
	{Name: "align-self", Initial: Keywords{"auto"}},
	{Name: "justify-content", Initial: Keywords{"normal"}},
	{Name: "justify-items", Initial: Keywords{"normal"}},
	{Name: "justify-self", Initial: Keywords{"auto"}},
	{Name: "order", Initial: Integer(0)},
	{Name: "column-gap", Initial: LengthKeyword("normal"), InitialNotComputed: true},
	{Name: "row-gap", Initial: LengthKeyword("normal")},
	{Name: "text-decoration-line", Initial: Keywords{"none"}},
	{Name: "text-decoration-color", Initial: currentColor},
	{Name: "text-decoration-style", Initial: Keyword("solid")},
	{Name: "text-decoration-thickness", Initial: LengthKeyword("auto")},
	{Name: "text-underline-offset", Initial: LengthKeyword("auto"), Inherited: true},
	{Name: "block-ellipsis", Initial: KeywordOrString{Keyword: "none"}, Inherited: true},
	{Name: "continue", Initial: Keyword("auto")},
	{Name: "max-lines", Initial: IntOrKeyword{Keyword: "none"}},
	{Name: "overflow", Initial: Keyword("visible"), TableWrapper: true},
	{Name: "overflow-wrap", Initial: Keyword("normal"), Inherited: true},
	{Name: "text-overflow", Initial: Keyword("clip")},
	{Name: "counter-increment", Initial: Counters{Auto: true}, TableWrapper: true},
	{Name: "counter-reset", Initial: Counters{}, TableWrapper: true},
	{Name: "counter-set", Initial: Counters{}, TableWrapper: true},
	{Name: "list-style-image", Initial: Image{Type: "none"}, Inherited: true},
	{Name: "list-style-position", Initial: Keyword("outside"), Inherited: true},
	{Name: "list-style-type", Initial: ListStyleType{Name: "disc"}, Inherited: true},
	{Name: "anchor", Initial: String("")},
	{Name: "lang", Initial: String(""), Inherited: true},
}
