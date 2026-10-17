package layout

import (
	"errors"
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// Context owns the mutable state and shared resources of document layout.
type Context struct {
	StyleFor          *cascade.StyleFor
	GetImage          images.Getter
	FontConfig        *fonts.FontConfiguration
	CounterStyle      *counters.Registry
	Targets           *targets.Collector
	ExcludedShapes    *[]boxes.Box
	StringSet         map[string]map[int][]string
	RunningElements   map[string]map[int][]boxes.Box
	CurrentPage       int
	Footnotes         []boxes.Box
	ReportedFootnotes []boxes.Box
	// CurrentPageFootnotes are the footnotes laid out on the current page.
	CurrentPageFootnotes []boxes.Box
	// RenderedNotes are the notes already laid out on previous pages.
	RenderedNotes []boxes.Box
	FootnoteArea  *boxes.FootnoteAreaBox
	PageBottom    float64
	Builder       *boxes.Builder
	Text          *linebreak.Context
	PageMaker     []PageMaker
	ForcedBreak   bool
	InColumn      bool

	lineResume      map[boxes.Box]ResumeAt
	bfcStack        []*bfcFrame
	bfcFrames       map[boxes.Box]*bfcFrame
	brokenOutOfFlow []brokenOutOfFlow
	err             error
}

type bfcFrame struct {
	Root           boxes.Box
	Shapes         []boxes.Box
	SuppressFloats bool
}

type brokenOutOfFlow struct {
	Fragment, Original, ContainingBlock, BFCRoot boxes.Box
	ResumeAt                                     ResumeAt
}

// NewContext shares the resources used to build the document's box tree.
func NewContext(builder *boxes.Builder) *Context {
	ctx := &Context{
		Builder:         builder,
		StringSet:       make(map[string]map[int][]string),
		RunningElements: make(map[string]map[int][]boxes.Box),
		bfcFrames:       make(map[boxes.Box]*bfcFrame),
	}
	if builder == nil || builder.StyleFor == nil {
		ctx.err = errors.New("layout requires a builder with computed styles")
		return ctx
	}
	ctx.StyleFor = builder.StyleFor
	ctx.GetImage = builder.GetImage
	ctx.FontConfig = builder.StyleFor.Context().FontConfig
	ctx.CounterStyle = &builder.CounterStyle
	ctx.Targets = builder.Targets
	ctx.Text = linebreak.NewContext(ctx.FontConfig)
	ctx.Footnotes = slices.Clone(builder.Footnotes)
	return ctx
}

// Err returns the first fatal layout or resource error.
func (ctx *Context) Err() error {
	if ctx.err != nil {
		return ctx.err
	}
	if ctx.Builder != nil {
		return ctx.Builder.Err()
	}
	return nil
}

// CreateBFC starts a formatting context, retaining exclusions restored from
// a continuation of the same root on a previous page.
func (ctx *Context) CreateBFC(root boxes.Box, shapes []boxes.Box) {
	if ctx.bfcFrames == nil {
		ctx.bfcFrames = make(map[boxes.Box]*bfcFrame)
	}
	frame := ctx.bfcFrames[root]
	if frame == nil {
		frame = &bfcFrame{Root: root, Shapes: slices.Clone(shapes)}
		ctx.bfcFrames[root] = frame
	}
	ctx.bfcStack = append(ctx.bfcStack, frame)
	ctx.ExcludedShapes = &frame.Shapes
}

// FinishBFC includes the context's floats in an auto-height root and restores
// the enclosing context's exclusions.
func (ctx *Context) FinishBFC(root boxes.Box) {
	if len(ctx.bfcStack) == 0 {
		return
	}
	frame := ctx.bfcStack[len(ctx.bfcStack)-1]
	if root != nil && root.Base().Style.Height().Keyword == "auto" && len(frame.Shapes) != 0 {
		boxBottom := root.ContentBoxY() + root.Base().Height.V
		bottom := boxBottom
		for _, shape := range frame.Shapes {
			bottom = max(bottom, shape.Base().PositionY+shape.MarginHeight())
		}
		root.Base().Height = units.F(root.Base().Height.V + (bottom - boxBottom))
	}
	delete(ctx.bfcFrames, frame.Root)
	ctx.bfcStack = ctx.bfcStack[:len(ctx.bfcStack)-1]
	ctx.ExcludedShapes = nil
	if len(ctx.bfcStack) > 0 {
		ctx.ExcludedShapes = &ctx.bfcStack[len(ctx.bfcStack)-1].Shapes
	}
}

// OverflowsPage compares a position to the available page bottom, allowing
// the relative rounding error accumulated by sums of CSS lengths.
func (ctx *Context) OverflowsPage(bottomSpace, positionY float64) bool {
	return overflows(ctx.PageBottom-bottomSpace, positionY)
}

// overflows reports whether a position is below a bottom, allowing for the
// relative error that sums of CSS lengths accumulate.
func overflows(bottom, positionY float64) bool {
	return positionY > bottom*(1+1e-9)
}

// StringSetFor selects the named strings used by a page margin box.
func (ctx *Context) StringSetFor(page *boxes.PageBox, name, keyword string) []string {
	return namedValuesFor(ctx.StringSet[name], page, ctx.CurrentPage, name, keyword)
}

// RunningElementFor selects the running elements used by a page margin box.
func (ctx *Context) RunningElementFor(page *boxes.PageBox, name, keyword string) []boxes.Box {
	return namedValuesFor(ctx.RunningElements[name], page, ctx.CurrentPage, name, keyword)
}

func namedValuesFor[T any](
	store map[int][]T,
	page *boxes.PageBox,
	currentPage int,
	name string,
	keyword string,
) []T {
	if current := store[currentPage]; len(current) != 0 {
		switch keyword {
		case "first":
			return []T{current[0]}
		case "last":
			return []T{current[len(current)-1]}
		case "first-except":
			return nil
		case "all-once":
			return slices.Clone(current)
		case "start":
			for element := boxes.Box(page); element != nil; {
				for _, assignment := range element.Base().Style.StringSet() {
					if assignment.Name == name {
						return []T{current[0]}
					}
				}
				children := boxes.ChildrenOf(element)
				if len(children) == 0 {
					break
				}
				element = children[0]
			}
		}
	}
	if keyword != "all-once" {
		for previous := currentPage - 1; previous > 0; previous-- {
			if values := store[previous]; len(values) != 0 {
				return []T{values[len(values)-1]}
			}
		}
	}
	return nil
}

var _ boxes.PageContext = (*Context)(nil)
