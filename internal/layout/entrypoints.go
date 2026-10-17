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
