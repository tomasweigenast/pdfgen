package props

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// Value is a property value, either specified or computed. The set of
// implementations is closed: every concrete type is declared in this package.
type Value interface{ isValue() }

// Keyword is an identifier such as "auto" or "border-box". Keywords are
// compared as lowercase strings.
type Keyword string

// String is an arbitrary string value, such as a hyphenation character.
type String string

// Strings is a list of strings, such as the families of font-family.
type Strings []string

// Keywords is a list of keywords. A property that accepts either a single
// keyword or a list of them holds a list in both cases, so "normal" is the
// one-element list {"normal"}.
type Keywords []Keyword

// Number is a unitless real number.
type Number float64

// Integer is a unitless integer.
type Integer int

// Pixels is a length already resolved to CSS pixels.
type Pixels float64

// Dimension is a number with a unit. The unit is "%" for percentages and
// empty for plain numbers; the lengths computed to pixels use "px".
type Dimension struct {
	Value float64
	Unit  string
}

// Calc is a math function (calc(), min(), ...) that could not be resolved at
// parse time because it depends on the style, such as one using em or %.
type Calc struct {
	Function *syntax.FunctionBlock
}

// Length is a length, a percentage, a plain number, a pending math function,
// or one of the keywords a property accepts in their place (auto, normal,
// content, ...). Exactly one of the three fields is meaningful: Keyword when
// it is not empty, then Calc when it is not nil, then Dimension.
type Length struct {
	Keyword   Keyword
	Dimension Dimension
	Calc      *Calc
}

// Px returns the length v CSS pixels long.
func Px(v float64) Length { return Length{Dimension: Dimension{v, "px"}} }

// LengthKeyword returns the length-like keyword k.
func LengthKeyword(k Keyword) Length { return Length{Keyword: k} }

// LengthNumber returns the plain number n as a Length.
func LengthNumber(n float64) Length { return Length{Dimension: Dimension{Value: n}} }

// IsKeyword reports whether l is a keyword.
func (l Length) IsKeyword() bool { return l.Keyword != "" }

// IsAuto reports whether l is the auto keyword.
func (l Length) IsAuto() bool { return l.Keyword == "auto" }

// IsInfinite reports whether l is the parsed value of max-width: none.
func (l Length) IsInfinite() bool {
	return l.Keyword == "" && l.Calc == nil && math.IsInf(l.Dimension.Value, 1)
}

// Color is a color value, including the currentcolor keyword, which is a
// syntax.Color whose space is syntax.CurrentColor.
type Color struct{ syntax.Color }

// IsCurrentColor reports whether c is the currentcolor keyword.
func (c Color) IsCurrentColor() bool { return c.Space == syntax.CurrentColor }

// Display is the list of keywords of display: outside and inside types and an
// optional list-item, or a single box-less or internal table keyword.
type Display []Keyword

// IntOrKeyword is an integer or a keyword such as auto or none, as in
// z-index, column-count, max-lines and bookmark-level.
type IntOrKeyword struct {
	Keyword Keyword
	N       int
}

// Image is the value of an <image>: none, an external URL or a gradient.
type Image struct {
	// Type is "none", "url", "linear-gradient" or "radial-gradient".
	Type   Keyword
	URL    string
	Linear *LinearGradient
	Radial *RadialGradient
}

// Images is a comma-separated list of images, one per background layer.
type Images []Image

// ColorStop is a color of a gradient with its optional position.
type ColorStop struct {
	Color    Color
	Position *Length
}

// Gradient holds what linear and radial gradients have in common.
type Gradient struct {
	ColorStops []ColorStop
	// ColorHints has one entry between each pair of color stops.
	ColorHints []Length
	Repeating  bool
}

// LinearGradient is a linear-gradient() or repeating-linear-gradient().
type LinearGradient struct {
	Gradient
	// DirectionType is "angle" or "corner".
	DirectionType Keyword
	// Angle is in radians, 0 pointing upwards, clockwise.
	Angle float64
	// Corner is a keyword such as "top_left", set for the "corner" type.
	Corner Keyword
}

// RadialGradient is a radial-gradient() or repeating-radial-gradient().
type RadialGradient struct {
	Gradient
	// Shape is "circle" or "ellipse".
	Shape Keyword
	// SizeType is "keyword" or "explicit".
	SizeType     Keyword
	SizeKeyword  Keyword
	SizeExplicit [2]Length
	Center       Position
}

// Position is a two-dimensional position with an origin edge per axis, as in
// background-position: {"left", x, "top", y}.
type Position struct {
	OriginX Keyword
	X       Length
	OriginY Keyword
	Y       Length
}

// Positions is a comma-separated list of positions.
type Positions []Position

// Repeat is the pair of keywords of a background-repeat layer.
type Repeat struct{ X, Y Keyword }

// Repeats is a comma-separated list of repeat pairs.
type Repeats []Repeat

// BackgroundSize is a background-size layer: contain, cover, or a width and
// a height that can be auto.
type BackgroundSize struct {
	Keyword       Keyword
	Width, Height Length
}

// BackgroundSizes is a comma-separated list of background sizes.
type BackgroundSizes []BackgroundSize

// LengthPair is two lengths: corner radii (horizontal, vertical) and
// transform-origin (x, y).
type LengthPair struct{ X, Y Length }

// PixelPair is two lengths in pixels: border-spacing (horizontal, vertical)
// and the page size (width, height).
type PixelPair struct{ X, Y Pixels }

// LengthQuad is four lengths, in top, right, bottom, left order.
type LengthQuad [4]Length

// Clip is the four values of rect(), each a length or auto. It is empty for
// clip: auto.
type Clip []Length

// BoxShadow is one shadow of box-shadow.
type BoxShadow struct {
	X, Y, Blur, Spread Length
	Color              Color
	Inset              bool
}

// BoxShadows is a comma-separated list of shadows. It is empty for none.
type BoxShadows []BoxShadow

// ImageSlice is border-image-slice: four numbers or percentages, where a
// number has an empty unit, and the optional fill keyword.
type ImageSlice struct {
	Edges [4]Dimension
	Fill  bool
}

// KeywordPair is two keywords, as in border-image-repeat.
type KeywordPair [2]Keyword

// IntTriple is three integers, as in hyphenate-limit-chars.
type IntTriple [3]int

// ListStyleKind tells what a ListStyleType holds.
type ListStyleKind uint8

const (
	// ListStyleKeyword is a counter style name such as decimal.
	ListStyleKeyword ListStyleKind = iota
	// ListStyleString is a literal marker string.
	ListStyleString
	// ListStyleSymbols is a symbols() function.
	ListStyleSymbols
)

// ListStyleType is a list-style-type or the counter style of counter().
type ListStyleType struct {
	Kind ListStyleKind
	// Name is the counter style name, the string, or the symbols() type
	// (cyclic, numeric, alphabetic, symbolic or fixed).
	Name    string
	Symbols []string
}

// ContentItem is one item of a content list.
type ContentItem struct {
	// Type is "string", "url", "quote", "content()", "attr()", "counter()",
	// "counters()", "leader()", "element()", "string()",
	// "target-counter()", "target-counters()" or "target-text()".
	Type string
	// Args are the string arguments of the item, in source order: the
	// text of a string, the kind and address of a url, the keyword of a
	// quote, and so on.
	Args []string
	// Style is the counter style of counter() and counters().
	Style *ListStyleType
	// LinkIsURL is set when the first argument of a target-*() function is
	// a url() instead of a string.
	LinkIsURL bool
}

// Content is the value of content: a keyword (normal, none, or after
// computing inhibit and contents) or a list of items.
type Content struct {
	Keyword Keyword
	Items   []ContentItem
}

// ContentList is a list of content items, as in bookmark-label.
type ContentList []ContentItem

// StringSetEntry assigns a content list to a named string.
type StringSetEntry struct {
	Name    string
	Content ContentList
}

// StringSet is the value of string-set. It is empty for none.
type StringSet []StringSetEntry

// Quotes is the value of quotes: auto, none, or the open and close quotes of
// each nesting level.
type Quotes struct {
	Keyword Keyword
	Open    []string
	Close   []string
}

// CounterOp is a counter name with an integer, as in counter-reset.
type CounterOp struct {
	Name  string
	Value int
}

// Counters is the value of counter-increment, counter-reset and counter-set.
// Auto is the initial value of counter-increment, which behaves as none but
// lets display: list-item increment the list-item counter.
type Counters struct {
	Auto  bool
	Items []CounterOp
}

// FontSetting is a four-letter OpenType tag with its value.
type FontSetting struct {
	Tag   string
	Value float64
}

// FontSettings is font-feature-settings or font-variation-settings.
type FontSettings struct {
	Normal bool
	Items  []FontSetting
}

// ImageOrientation is image-orientation: a keyword (none, from-image) or an
// angle with an optional flip.
type ImageOrientation struct {
	Keyword Keyword
	// Angle is in radians when specified and in degrees, a multiple of 90,
	// when computed.
	Angle float64
	Flip  bool
}

// TransformOp is one function of transform. Name is "rotate", "skew",
// "translate", "scale" or "matrix".
type TransformOp struct {
	Name Keyword
	// Numbers holds the angle of rotate, the two angles of skew, the two
	// factors of scale, or the six coefficients of matrix.
	Numbers []float64
	// Translate holds the arguments of translate.
	Translate LengthPair
}

// Transform is a list of transform functions. It is empty for none.
type Transform []TransformOp

// Positioning is the value of position: a keyword, or a running() or note()
// function with its identifier.
type Positioning struct {
	// Keyword is "static", "relative", "absolute", "fixed", "running()" or
	// "note()".
	Keyword Keyword
	Name    string
}

// KeywordOrString is a keyword or a string, as in block-ellipsis and
// font-language-override. Keyword is empty when the value is a string.
type KeywordOrString struct {
	Keyword Keyword
	String  string
}

// TrackSize is a grid track size: a single breadth (a length, a fraction
// with the fr unit, or auto, min-content, max-content), minmax() or
// fit-content().
type TrackSize struct {
	// Function is "", "minmax()" or "fit-content()".
	Function Keyword
	// Breadth is the whole size when Function is empty, or the argument of
	// fit-content().
	Breadth  Length
	Min, Max Length
}

// TrackList is a list of track sizes, as in grid-auto-columns.
type TrackList []TrackSize

// GridItemKind tells what a GridItem holds.
type GridItemKind uint8

const (
	// GridLineNames is a possibly empty list of line names.
	GridLineNames GridItemKind = iota
	// GridTrack is a track size.
	GridTrack
	// GridRepeat is a repeat() function.
	GridRepeat
)

// GridItem is an element of a track list with line names, as in
// grid-template-columns. Line names and tracks alternate, starting and
// ending with line names.
type GridItem struct {
	Kind  GridItemKind
	Names []string
	Track TrackSize
	// Count is the repeat count, or 0 when Auto is set.
	Count int
	// Auto is "auto-fill" or "auto-fit" for an auto repeat.
	Auto  Keyword
	Items []GridItem
}

// GridTemplate is grid-template-columns or grid-template-rows: none,
// subgrid with optional line names, or a track list.
type GridTemplate struct {
	None    bool
	Subgrid bool
	Items   []GridItem
}

// GridAreas is grid-template-areas. Each row has one name per column, empty
// for a cell that belongs to no area.
type GridAreas struct {
	None bool
	Rows [][]string
}

// GridLine is a grid-row-start, grid-row-end, grid-column-start or
// grid-column-end value.
type GridLine struct {
	Auto  bool
	Span  bool
	N     int
	Ident string
}

func (Keyword) isValue()          {}
func (String) isValue()           {}
func (Strings) isValue()          {}
func (Keywords) isValue()         {}
func (Number) isValue()           {}
func (Integer) isValue()          {}
func (Pixels) isValue()           {}
func (Dimension) isValue()        {}
func (Length) isValue()           {}
func (Color) isValue()            {}
func (Display) isValue()          {}
func (IntOrKeyword) isValue()     {}
func (Image) isValue()            {}
func (Images) isValue()           {}
func (Position) isValue()         {}
func (Positions) isValue()        {}
func (Repeat) isValue()           {}
func (Repeats) isValue()          {}
func (BackgroundSize) isValue()   {}
func (BackgroundSizes) isValue()  {}
func (LengthPair) isValue()       {}
func (PixelPair) isValue()        {}
func (LengthQuad) isValue()       {}
func (Clip) isValue()             {}
func (BoxShadow) isValue()        {}
func (BoxShadows) isValue()       {}
func (ImageSlice) isValue()       {}
func (KeywordPair) isValue()      {}
func (IntTriple) isValue()        {}
func (ListStyleType) isValue()    {}
func (ContentItem) isValue()      {}
func (Content) isValue()          {}
func (ContentList) isValue()      {}
func (StringSet) isValue()        {}
func (Quotes) isValue()           {}
func (Counters) isValue()         {}
func (FontSettings) isValue()     {}
func (ImageOrientation) isValue() {}
func (Transform) isValue()        {}
func (Positioning) isValue()      {}
func (KeywordOrString) isValue()  {}
func (TrackSize) isValue()        {}
func (TrackList) isValue()        {}
func (GridTemplate) isValue()     {}
func (GridAreas) isValue()        {}
func (GridLine) isValue()         {}
