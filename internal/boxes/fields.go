package boxes

import (
	"fmt"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/units"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Radius is the horizontal and vertical radius of a rounded corner, in CSS
// pixels.
type Radius struct{ X, Y float64 }

// Rect is a rectangle in CSS pixels.
type Rect struct{ X, Y, Width, Height float64 }

// Sides holds one value for each side of a box.
type Sides struct{ Top, Right, Bottom, Left float64 }

// DecorationSides is the set of sides whose margin, padding and border were
// removed when the box was split (CSS Fragmentation 3 §5.4).
type DecorationSides uint8

// The sides of a box.
const (
	SideTop DecorationSides = 1 << iota
	SideRight
	SideBottom
	SideLeft
)

// Has reports whether s contains every side of o.
func (s DecorationSides) Has(o DecorationSides) bool { return s&o == o }

// StringSetItem is a string assigned by string-set: its name and its text.
type StringSetItem struct {
	Name, Text string
}

// FlexItemFields holds the values computed for a flex item during the
// layout of its container (CSS Flexbox 1 §9.7).
type FlexItemFields struct {
	BaseSize             float64
	MainOuterExtra       float64
	HypotheticalMainSize float64
	TargetMainSize       float64
	Frozen               bool
	Factor               float64
	ScaledShrinkFactor   float64
	Adjustment           float64
	// Baseline is the baseline of the item before its container is laid
	// out in the cross axis.
	Baseline float64
}

// BoxFields holds what every box has: the element and style it comes from,
// the used values that layout sets, and the flags that building and layout
// attach to boxes. The geometry methods are defined on it.
//
// Fields documented as used values are meaningless until layout resolves
// them. Lengths that can be auto during layout are [units.MaybeFloat].
type BoxFields struct {
	kind Kind

	// ElementTag is the tag of the element, followed by the pseudo-element
	// for generated boxes, such as "p::before". It is empty for boxes that
	// have no element.
	ElementTag string
	Element    *html.Element
	Style      *computed.Style

	FirstLetterStyle *computed.Style
	FirstLineStyle   *computed.Style

	// RemoveDecorationSides are the sides whose decoration was removed by
	// RemoveDecoration.
	RemoveDecorationSides DecorationSides

	// Index is the position of the box among its siblings when it was
	// taken from the tree, which resume points refer to.
	Index int

	// Used values.
	PositionX, PositionY float64
	Width, Height        units.MaybeFloat
	MinWidth, MaxWidth   units.MaybeFloat
	MinHeight, MaxHeight units.MaybeFloat

	// Offsets of positioned boxes.
	Top, Right, Bottom, Left units.MaybeFloat

	MarginTop, MarginRight, MarginBottom, MarginLeft     units.MaybeFloat
	PaddingTop, PaddingRight, PaddingBottom, PaddingLeft float64
	BorderTopWidth, BorderRightWidth                     float64
	BorderBottomWidth, BorderLeftWidth                   float64
	// CollapsedBordersResolved distinguishes resolved zero widths from borders
	// whose used widths still need to be read from the style.
	CollapsedBordersResolved bool

	BorderTopLeftRadius, BorderTopRightRadius       Radius
	BorderBottomRightRadius, BorderBottomLeftRadius Radius

	// Baseline is the distance from the top of the margin box to the
	// baseline, nil when the box has none.
	Baseline *float64
	// Clearance is the space added above a block to clear floats, nil when
	// there is none.
	Clearance *float64

	// Background is the resolved background of the box, set by layout and
	// read by drawing; nil when there is nothing to paint.
	Background any
	// BorderImage and MaskBorderImage are the images of border-image-source
	// and mask-border-source.
	BorderImage, MaskBorderImage images.Image

	// Flags that building and layout set on instances.
	IsTableWrapper   bool
	IsFlexItem       bool
	IsGridItem       bool
	IsForRootElement bool
	IsColumn         bool
	IsLeader         bool
	IsOutsideMarker  bool
	// NotFloated makes IsFloated report false, for children of flex
	// containers, which ignore float.
	NotFloated bool
	// ForceFragmentation is set on boxes that must be split even when they
	// would fit.
	ForceFragmentation bool

	// Collapsible spaces removed at the start and end of the text of the
	// box, kept as wrap opportunities.
	LeadingCollapsibleSpace  bool
	TrailingCollapsibleSpace bool

	TransformationMatrix *matrix.Matrix
	// ViewportOverflow is the overflow of the root or body element, which
	// propagates to the viewport. It is set on the root box.
	ViewportOverflow string

	// Generated content and GCPM data.
	BookmarkLabel string
	StringSet     []StringSetItem
	// Footnote is the footnote box a footnote call points to.
	Footnote Box
	// Note, NoteCall and NoteCallback link a note() box with its call and
	// callback boxes.
	Note, NoteCall, NoteCallback Box
	// CounterValues are the counters in effect at the box, kept for
	// target-counter() and for content built during pagination.
	CounterValues targets.CounterValues
	// Link is the destination of the link the box is, or is inside of.
	Link *urls.Link
	// MissingLinkBox ties the boxes made from a box whose content has
	// counters known only during pagination to the original box.
	MissingLinkBox Box

	LinkAnnotation   *writer.Dict
	WidgetAnnotation *writer.Dict

	// Flex holds the values of a flex item.
	Flex FlexItemFields
}

// Base returns the fields of the box.
func (b *BoxFields) Base() *BoxFields { return b }

// Kind returns the concrete type of the box.
func (b *BoxFields) Kind() Kind { return b.kind }

// String describes the box for debugging, as <Block div>.
func (b *BoxFields) String() string {
	return fmt.Sprintf("<%s %s>", b.kind, b.ElementTag)
}

// Attr returns an attribute of the element of the box, for attr() links.
func (b *BoxFields) Attr(name string) (string, bool) {
	if b.Element == nil {
		return "", false
	}
	return b.Element.Lookup(name)
}

// BaseURL returns the base URL of the style of the box.
func (b *BoxFields) BaseURL() string { return b.Style.BaseURL() }

// CachedCounterValues returns the counters in effect at the box, or nil.
func (b *BoxFields) CachedCounterValues() targets.CounterValues { return b.CounterValues }

// SetCachedCounterValues sets the counters in effect at the box.
func (b *BoxFields) SetCachedCounterValues(v targets.CounterValues) { b.CounterValues = v }

// MissingLink returns the box that the boxes made from this one point to,
// or nil. It is an interface so that it satisfies targets.Linked.
func (b *BoxFields) MissingLink() any {
	if b.MissingLinkBox == nil {
		return nil
	}
	return b.MissingLinkBox
}

// SetMissingLink sets the box returned by MissingLink. Values that are not
// boxes clear it.
func (b *BoxFields) SetMissingLink(v any) {
	box, _ := v.(Box)
	b.MissingLinkBox = box
}

var (
	_ targets.Source = (*BoxFields)(nil)
	_ targets.Target = (*BoxFields)(nil)
	_ targets.Linked = (*BoxFields)(nil)
)
