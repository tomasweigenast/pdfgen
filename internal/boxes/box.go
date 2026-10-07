package boxes

// Box is a box of the formatting structure. Its geometry and positioning
// methods come from [BoxFields]; use [Descendants], [ChildrenOf] and the
// other functions of the package for what depends on the children.
type Box interface {
	// Base returns the fields shared by all boxes.
	Base() *BoxFields
	// Kind returns the concrete type of the box.
	Kind() Kind
	// Copy returns a shallow copy: the copy shares the style, the children
	// and every other reference with the original.
	Copy() Box
	// DeepCopy returns a copy whose descendants are copies too.
	DeepCopy() Box
	// Translate moves the box and its descendants. Floats and their
	// descendants are left in place when ignoreFloats is set.
	Translate(dx, dy float64, ignoreFloats bool)

	PaddingWidth() float64
	PaddingHeight() float64
	BorderWidth() float64
	BorderHeight() float64
	MarginWidth() float64
	MarginHeight() float64
	ContentBoxX() float64
	ContentBoxY() float64
	PaddingBoxX() float64
	PaddingBoxY() float64
	BorderBoxX() float64
	BorderBoxY() float64
	HitArea() Rect
	RoundedBox(bt, br, bb, bl float64) RoundedBox
	RoundedBoxRatio(ratio float64) RoundedBox
	RoundedPaddingBox() RoundedBox
	RoundedBorderBox() RoundedBox
	RoundedContentBox() RoundedBox

	IsFloated() bool
	IsFootnote() bool
	IsNote() bool
	IsAbsolutelyPositioned() bool
	IsRunning() bool
	IsInNormalFlow() bool
	IsMonolithic() bool
	EstablishesFormattingContext() bool
	TopMarginCollapses() bool
	BottomMarginCollapses() bool
	PageValues() (start, end string)
	RemoveDecoration(start, end bool)
	IsAttachment() bool
	IsInput() bool
	IsForm() bool
}
