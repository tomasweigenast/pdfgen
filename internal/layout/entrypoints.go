package layout

import (
	"errors"
	"iter"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

var errUnimplemented = errors.New("layout entry point not implemented")

func (ctx *Context) unimplemented() {
	if ctx.err == nil {
		ctx.err = errUnimplemented
	}
}

// iterLineBoxes lays out the lines of a block container's inline content.
func iterLineBoxes(ctx *Context, box, containing boxes.Box, opts blockOptions, positionY float64) iter.Seq[lineResult] {
	ctx.unimplemented()
	return func(func(lineResult) bool) {}
}

// absoluteLayout resolves an absolute placeholder against its containing block.
func absoluteLayout(ctx *Context, placeholder *boxes.AbsolutePlaceholder, containing boxes.Box, fixed *[]*boxes.AbsolutePlaceholder, bottomSpace float64, skip ResumeAt) {
	ctx.unimplemented()
}

// absoluteBoxLayout lays out an absolutely positioned box and returns the
// fragment, or nil when it did not fit.
func absoluteBoxLayout(ctx *Context, box, containing boxes.Box, fixed *[]*boxes.AbsolutePlaceholder, bottomSpace float64, skip ResumeAt) (boxes.Box, ResumeAt) {
	ctx.unimplemented()
	return nil, nil
}
