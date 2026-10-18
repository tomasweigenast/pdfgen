package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

var noEllipsis = props.KeywordOrString{Keyword: "none"}

// DrawPage paints a laid-out page on s, whose origin is the top-left corner
// of the page and whose unit is the CSS pixel.
func DrawPage(s *stream.Stream, page *boxes.PageBox) {
	ctx := FromPage(page)
	bleed := page.Bleed()

	drawBackground(s, ctx.Box.Base().Background, false, &bleed, page.Style.Marks())
	setMaskBorder(s, page)
	drawBackground(s, page.CanvasBackground, false, nil, nil)
	drawBorder(s, page)
	drawStackingContext(s, ctx)
}

// drawStackingContext paints a stacking context in the order of CSS 2.1
// Appendix E. Step 1 is done by DrawPage.
func drawStackingContext(s *stream.Stream, ctx *StackingContext) {
	s.PushState()
	defer s.PopState()

	box := ctx.Box
	base := box.Base()
	style := base.Style

	// The overflow of the root element applies to the viewport.
	if base.IsForRootElement && ctx.Page.Style.Overflow() != "visible" {
		clipRoundedBox(s, ctx.Page.RoundedPaddingBox())
	}
	if box.IsAbsolutelyPositioned() {
		clipToClipProperty(s, box)
	}

	out := s
	opacity := float64(style.Opacity())
	if opacity < 1 {
		r := s.PageRectangle
		out = s.AddGroup(r[0], r[1], r[2], r[3])
	}

	if m := base.TransformationMatrix; m != nil {
		if m.Determinant() == 0 {
			return
		}
		out.Transform(m.Values())
	}

	// Step 2.
	switch box.Kind() {
	case boxes.KindBlock, boxes.KindTableCaption, boxes.KindFootnoteArea,
		boxes.KindMargin, boxes.KindInlineBlock, boxes.KindTableCell,
		boxes.KindFlex, boxes.KindInlineFlex, boxes.KindGrid, boxes.KindInlineGrid,
		boxes.KindBlockReplaced, boxes.KindInlineReplaced:
		setMaskBorder(out, box)
		drawBackground(out, base.Background, true, nil, nil)
		drawBorder(out, box)
	}

	out.PushState()
	// The page box is not clipped, its overflow applies to the viewport. The
	// border and the background are painted outside of the clip.
	if style.Overflow() != "visible" && box.Kind() != boxes.KindPage {
		clipRoundedBox(out, box.RoundedPaddingBox())
	}

	for _, c := range ctx.NegativeZ { // Step 3.
		drawStackingContext(out, c)
	}
	for _, block := range ctx.BlockLevelBoxes { // Step 4.
		setMaskBorder(out, block)
		if _, ok := block.(boxes.TableLike); ok {
			drawTable(out, block)
		} else {
			drawBackground(out, block.Base().Background, true, nil, nil)
			drawBorder(out, block)
		}
	}
	for _, c := range ctx.FloatContexts { // Step 5.
		drawStackingContext(out, c)
	}
	if box.Kind() == boxes.KindInline { // Step 6.
		drawInlineLevel(out, box, 0, "clip", noEllipsis)
	}
	// Step 7.
	drawBlockLevel(out, []BlockEntry{{Box: box, Children: ctx.BlocksAndCells}})
	for _, c := range ctx.ZeroZ { // Step 8.
		drawStackingContext(out, c)
	}
	for _, c := range ctx.PositiveZ { // Step 9.
		drawStackingContext(out, c)
	}
	out.PopState()

	drawOutline(out, box) // Step 10.

	if opacity < 1 {
		s.PushState()
		both := true
		s.SetAlpha(opacity, true, &both)
		s.DrawXObject(out.ID())
		s.PopState()
	}
}

// clipToClipProperty restricts the painting of an absolutely positioned box
// to the rectangle of its clip property. The second and fourth values are the
// left and right edges, and auto stands for the border box edge.
func clipToClipProperty(s *stream.Stream, box boxes.Box) {
	clip := box.Base().Style.Clip()
	if len(clip) != 4 {
		return
	}
	edge := func(i int, auto float64) float64 {
		if clip[i].Keyword == "auto" {
			return auto
		}
		return clip[i].Dimension.Value
	}
	top := edge(0, 0)
	left := edge(1, 0)
	bottom := edge(2, box.BorderHeight())
	right := edge(3, box.BorderWidth())
	s.Rectangle(box.BorderBoxX()+left, box.BorderBoxY()+top, right-left, bottom-top)
	s.Clip(false)
	s.EndPath()
}

// drawInlineLevel paints an inline-level box and its descendants. offsetX
// moves the text of the line, when it is aligned by the line box.
func drawInlineLevel(s *stream.Stream, box boxes.Box, offsetX float64,
	textOverflow props.Keyword, blockEllipsis props.KeywordOrString) {
	if cb, ok := box.(*contextBox); ok {
		drawStackingContext(s, cb.context)
		return
	}

	base := box.Base()
	setMaskBorder(s, box)
	drawBackground(s, base.Background, true, nil, nil)
	drawBorder(s, box)

	switch box.Kind() {
	case boxes.KindInline, boxes.KindLine:
		if line, ok := box.(*boxes.LineBox); ok {
			textOverflow = line.TextOverflow
			blockEllipsis = line.BlockEllipsis
			if blockEllipsis == (props.KeywordOrString{}) {
				blockEllipsis = noEllipsis
			}
		}
		children := boxes.ChildrenOf(box)
		for i, child := range children {
			ellipsis := noEllipsis
			if i == len(children)-1 {
				ellipsis = blockEllipsis
			}
			childOffset := offsetX
			if _, ok := child.(*contextBox); !ok {
				childOffset += child.Base().PositionX - base.PositionX
			}
			if text, ok := child.(*boxes.TextBox); ok {
				drawText(s, text, childOffset, textOverflow, ellipsis)
			} else {
				drawInlineLevel(s, child, childOffset, textOverflow, ellipsis)
			}
		}
	case boxes.KindInlineReplaced:
		drawReplaced(s, box.(boxes.Replaced))
	case boxes.KindText:
		// List markers are text boxes outside of a line.
		drawText(s, box.(*boxes.TextBox), offsetX, textOverflow, noEllipsis)
	}
}

// drawBlockLevel paints the inline content and the replaced content of the
// blocks and cells of a stacking context, in tree order.
func drawBlockLevel(s *stream.Stream, entries []BlockEntry) {
	for _, e := range entries {
		if r, ok := e.Box.(boxes.Replaced); ok {
			drawReplaced(s, r)
		} else if children := boxes.ChildrenOf(e.Box); len(children) > 0 {
			if _, ok := children[len(children)-1].(*boxes.LineBox); ok {
				for _, child := range children {
					drawInlineLevel(s, child, 0, "clip", noEllipsis)
				}
			}
		}
		drawBlockLevel(s, e.Children)
	}
}
