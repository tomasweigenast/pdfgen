package layout

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// textAlign returns the horizontal offset of a line inside the available
// width, according to text-align (CSS Text 3 §7.1). A justified line is
// stretched instead and keeps its offset. last tells that the line is the
// last one of its block or ends with a forced break, which text-align-last
// governs.
func textAlign(ctx *Context, line *boxes.LineBox, availableWidth float64, last bool) float64 {
	// When the content is as wide as the line there is nothing to
	// distribute.
	if line.Width.V >= availableWidth {
		return 0
	}

	style := line.Style
	align := style.TextAlignAll()
	if last {
		if alignLast := style.TextAlignLast(); alignLast != "auto" {
			align = alignLast
		}
	}
	rtl := style.Direction() == "rtl"
	if align == "left" || align == "right" {
		if (align == "left") != rtl {
			align = "start"
		} else {
			align = "end"
		}
	}

	offset := availableWidth - line.Width.V
	switch align {
	case "start":
		return 0
	case "justify":
		// Justifying text whose white space does not collapse is forbidden
		// by CSS 2 and not required by CSS Text 3.
		if collapsesSpaces(line) {
			justifyLine(ctx, line, offset)
		}
		return 0
	case "center":
		return offset / 2
	}
	return offset
}

// justifyLine spreads extraWidth over the expandable spaces of the line.
func justifyLine(ctx *Context, line boxes.Box, extraWidth float64) {
	if n := countExpandableSpaces(line); n != 0 {
		addWordSpacing(ctx, line, extraWidth/float64(n), 0)
	}
}

// countExpandableSpaces counts the spaces and no-break spaces that
// justification can widen.
func countExpandableSpaces(box boxes.Box) int {
	switch box := box.(type) {
	case *boxes.TextBox:
		return strings.Count(box.Text, " ") + strings.Count(box.Text, " ")
	case *boxes.LineBox, *boxes.InlineBox:
		n := 0
		for _, child := range boxes.ChildrenOf(box) {
			n += countExpandableSpaces(child)
		}
		return n
	}
	return 0
}

// addWordSpacing widens every expandable space under box by
// justificationSpacing and moves what follows a widened space by the
// accumulated advance, which it returns. The text is shaped again with the
// extra space, so that glyph positions agree with the new widths.
func addWordSpacing(ctx *Context, box boxes.Box, justificationSpacing, xAdvance float64) float64 {
	switch box := box.(type) {
	case *boxes.TextBox:
		box.JustificationSpacing = justificationSpacing
		box.PositionX += xAdvance
		if n := countExpandableSpaces(box); n > 0 {
			box.Layout = linebreak.NewLayout(
				ctx.Text, box.Text, box.Style.LineBreakStyle(), nil, justificationSpacing)
			extra := justificationSpacing * float64(n)
			xAdvance += extra
			box.Width = units.F(box.Width.V + extra)
		}
	case *boxes.LineBox, *boxes.InlineBox:
		base := box.Base()
		base.PositionX += xAdvance
		previous := xAdvance
		children := boxes.ChildrenOf(box)
		for i := range children {
			child := children[i]
			if base.Style.Direction() == "rtl" {
				child = children[len(children)-1-i]
			}
			if child.IsInNormalFlow() {
				xAdvance = addWordSpacing(ctx, child, justificationSpacing, xAdvance)
			}
		}
		base.Width = units.F(base.Width.V + xAdvance - previous)
	default:
		box.Translate(xAdvance, 0, false)
	}
	return xAdvance
}
