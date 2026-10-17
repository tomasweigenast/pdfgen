package layout

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// withMinMaxWidth runs setWidth, which sets the used width of b, and then
// applies min-width and max-width (CSS 2.1 §10.4): when the width is out of
// bounds it is replaced by the bound and setWidth runs again, so that what
// depends on the width, such as auto margins, follows it. The minimum is
// checked last and wins over the maximum.
//
// A function F that honours the limits is written as a function that does
// not, FWithoutMinMax, and a wrapper that passes it to withMinMaxWidth;
// callers that must ignore the limits call the former.
func withMinMaxWidth(b *boxes.BoxFields, setWidth func()) {
	setWidth()
	if greater(b.Width, b.MaxWidth) {
		b.Width = b.MaxWidth
		setWidth()
	}
	if greater(b.MinWidth, b.Width) {
		b.Width = b.MinWidth
		setWidth()
	}
}

// withMinMaxHeight is withMinMaxWidth for min-height and max-height (CSS 2.1
// §10.7).
func withMinMaxHeight(b *boxes.BoxFields, setHeight func()) {
	setHeight()
	if greater(b.Height, b.MaxHeight) {
		b.Height = b.MaxHeight
		setHeight()
	}
	if greater(b.MinHeight, b.Height) {
		b.Height = b.MinHeight
		setHeight()
	}
}

// greater reports whether a is larger than b. An auto dimension has no
// size to compare, so it is never larger and nothing is larger than it.
func greater(a, b units.MaybeFloat) bool {
	return !a.Auto && !b.Auto && a.V > b.V
}
