package layout

import (
	"fmt"
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// fail records the first fatal error of the layout.
func (ctx *Context) fail(err error) {
	if ctx.err == nil {
		ctx.err = err
	}
}

// unsupported records that the layout met a box whose layout mode is not
// implemented. It is a fatal error rather than an empty result, which would
// read as "nothing fits on this page".
func (ctx *Context) unsupported(feature string) {
	ctx.fail(fmt.Errorf("layout of %s is not supported yet", feature))
}

// addExcludedShape adds a float to the exclusions of the current block
// formatting context, unless the context ignores floats.
func (ctx *Context) addExcludedShape(shape boxes.Box) {
	if len(ctx.bfcStack) == 0 {
		ctx.fail(fmt.Errorf("float laid out outside of a block formatting context"))
		return
	}
	frame := ctx.bfcStack[len(ctx.bfcStack)-1]
	if frame.SuppressFloats {
		return
	}
	frame.Shapes = append(frame.Shapes, shape)
}

// removeExcludedShape takes back a float that was laid out on a rejected
// attempt.
func (ctx *Context) removeExcludedShape(shape boxes.Box) {
	if ctx.ExcludedShapes == nil {
		return
	}
	if i := slices.Index(*ctx.ExcludedShapes, shape); i >= 0 {
		*ctx.ExcludedShapes = slices.Delete(*ctx.ExcludedShapes, i, i+1)
	}
}

// addBrokenOutOfFlow records an out-of-flow box that did not fit on the
// page, so that the next page lays out its remainder first. Records keep
// the order in which their fragments were first added.
func (ctx *Context) addBrokenOutOfFlow(fragment, original, containing boxes.Box, resume ResumeAt) {
	record := brokenOutOfFlow{
		Fragment:        fragment,
		Original:        original,
		ContainingBlock: containing,
		ResumeAt:        resume.Clone(),
	}
	if len(ctx.bfcStack) != 0 {
		record.BFCRoot = ctx.bfcStack[len(ctx.bfcStack)-1].Root
	}
	for i, existing := range ctx.brokenOutOfFlow {
		if existing.Fragment == fragment {
			ctx.brokenOutOfFlow[i] = record
			return
		}
	}
	ctx.brokenOutOfFlow = append(ctx.brokenOutOfFlow, record)
}

// removeBrokenOutOfFlow forgets the record of a fragment.
func (ctx *Context) removeBrokenOutOfFlow(fragment boxes.Box) {
	ctx.brokenOutOfFlow = slices.DeleteFunc(ctx.brokenOutOfFlow, func(r brokenOutOfFlow) bool {
		return r.Fragment == fragment
	})
}

// addRunningElement stores a running element or a note for the margin
// boxes of the current page. The first time a note is met it receives the
// anchors that link its call and its callback.
func (ctx *Context) addRunningElement(box boxes.Box) {
	b := box.Base()
	name := b.Style.Position().Name
	if box.IsNote() && b.Style.Anchor() == "" {
		index := 0
		for _, pages := range ctx.RunningElements {
			for _, list := range pages {
				index += len(list)
			}
		}
		b.Style.Set(props.PropAnchor, props.String(fmt.Sprintf("note-%d", index)))
		if b.NoteCall != nil {
			call := b.NoteCall.Base()
			call.Style.Set(props.PropAnchor, props.String(fmt.Sprintf("call-%d", index)))
			call.Link = urls.GetLink(fmt.Sprintf("#note-%d", index), b.Style.BaseURL())
		}
		if b.NoteCallback != nil {
			b.NoteCallback.Base().Link = urls.GetLink(fmt.Sprintf("#call-%d", index), b.Style.BaseURL())
		}
	}
	if slices.Contains(ctx.RenderedNotes, box) {
		return
	}
	if ctx.RunningElements == nil {
		ctx.RunningElements = make(map[string]map[int][]boxes.Box)
	}
	pages := ctx.RunningElements[name]
	if pages == nil {
		pages = make(map[int][]boxes.Box)
		ctx.RunningElements[name] = pages
	}
	pages[ctx.CurrentPage] = append(pages[ctx.CurrentPage], box)
}

// removeRunningElement takes back a running element stored for the current
// page by a rejected attempt.
func (ctx *Context) removeRunningElement(box boxes.Box) {
	name := box.Base().Style.Position().Name
	list := ctx.RunningElements[name][ctx.CurrentPage]
	if i := slices.Index(list, box); i >= 0 {
		ctx.RunningElements[name][ctx.CurrentPage] = slices.Delete(list, i, i+1)
	}
}
