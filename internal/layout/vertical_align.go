package layout

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// extent is the vertical range covered by margin boxes. The zero value is
// the empty range.
type extent struct {
	min, max float64
	set      bool
}

// include widens the range to cover [top, bottom].
func (e *extent) include(top, bottom float64) {
	if !e.set || top < e.min {
		e.min = top
	}
	if !e.set || bottom > e.max {
		e.max = bottom
	}
	e.set = true
}

// merge widens the range to cover another.
func (e *extent) merge(o extent) {
	if o.set {
		e.include(o.min, o.max)
	}
}

// baselineOf returns the distance from the top of the margin box to the
// baseline, zero for a box that has none.
func baselineOf(b boxes.Box) float64 {
	if base := b.Base().Baseline; base != nil {
		return *base
	}
	return 0
}

// lineBoxVerticality places the boxes of a line according to their
// vertical-align, with the baseline of the line at y = 0, and returns the
// bottom and the top of the line (CSS 2.1 §10.8.1). Boxes aligned to the
// top or the bottom of the line, and floats, depend on the height of the
// line, so they are placed last.
func lineBoxVerticality(line boxes.Box) (bottom, top float64) {
	var subtrees []boxes.Box
	all := alignedSubtreeVerticality(line, &subtrees, 0)

	// Aligning a subtree can find more of them, so the list may grow
	// while it is walked.
	type placed struct {
		box    boxes.Box
		extent extent
	}
	var placedSubtrees []placed
	for i := 0; i < len(subtrees); i++ {
		subtree := subtrees[i]
		var e extent
		if !subtree.IsFloated() {
			e = alignedSubtreeVerticality(subtree, &subtrees, 0)
		}
		placedSubtrees = append(placedSubtrees, placed{subtree, e})
	}

	tallest, found := 0.0, false
	for _, p := range placedSubtrees {
		if p.box.IsFloated() {
			continue
		}
		if height := p.extent.max - p.extent.min; !found || height > tallest {
			tallest, found = height, true
		}
	}
	if found {
		all.max = max(all.max, all.min+tallest)
	}

	for _, p := range placedSubtrees {
		var dy float64
		switch {
		case p.box.IsFloated():
			dy = all.min - p.box.Base().PositionY
		case p.box.Base().Style.VerticalAlign().Keyword == "top":
			dy = all.min - p.extent.min
		default:
			dy = all.max - p.extent.max
		}
		translateSubtree(p.box, dy)
	}
	return all.max, all.min
}

// translateSubtree moves a box vertically. An inline box aligned to the top
// or the bottom of the line takes its children with it; the children of
// the others were placed relative to their parent.
func translateSubtree(box boxes.Box, dy float64) {
	if inline, ok := box.(*boxes.InlineBox); ok {
		inline.PositionY += dy
		switch inline.Style.VerticalAlign().Keyword {
		case "top", "bottom":
			for _, child := range inline.Children {
				translateSubtree(child, dy)
			}
		}
		return
	}
	box.Translate(0, dy, false)
}

// alignedSubtreeVerticality places the descendants of box, whose baseline is
// at baselineY, and returns the range covered by them and by the box itself.
func alignedSubtreeVerticality(box boxes.Box, topBottomSubtrees *[]boxes.Box, baselineY float64) extent {
	e := inlineBoxVerticality(box, topBottomSubtrees, baselineY)
	top := baselineY - baselineOf(box)
	e.include(top, top+box.MarginHeight())
	return e
}

// inlineBoxVerticality places the children of a line or inline box, whose
// baseline is at baselineY, and returns the range covered by their margin
// boxes. Children aligned to the top or the bottom are appended to
// topBottomSubtrees instead.
func inlineBoxVerticality(box boxes.Box, topBottomSubtrees *[]boxes.Box, baselineY float64) extent {
	var e extent
	if k := box.Kind(); k != boxes.KindLine && k != boxes.KindInline {
		return e
	}
	parent := box.Base()

	for _, child := range boxes.ChildrenOf(box) {
		if !child.IsInNormalFlow() {
			if child.IsFloated() {
				*topBottomSubtrees = append(*topBottomSubtrees, child)
			}
			continue
		}
		childBase := child.Base()
		align := childBase.Style.VerticalAlign()
		var childBaselineY float64
		switch align.Keyword {
		case "baseline":
			childBaselineY = baselineY
		case "middle":
			oneEx := float64(parent.Style.FontSize()) * parent.Style.CharacterRatio("ex")
			top := baselineY - (oneEx+child.MarginHeight())/2
			childBaselineY = top + baselineOf(child)
		case "text-top":
			top := baselineY - baselineOf(box) + parent.MarginTop.V +
				parent.BorderTopWidth + parent.PaddingTop
			childBaselineY = top + baselineOf(child)
		case "text-bottom":
			bottom := baselineY - baselineOf(box) + parent.MarginTop.V +
				parent.BorderTopWidth + parent.PaddingTop + parent.Height.V
			childBaselineY = bottom - child.MarginHeight() + baselineOf(child)
		case "top", "bottom":
			// The subtree is placed as if its baseline were at y = 0, and
			// moved once the height of the line is known.
			childBaselineY = 0
		default:
			if align.Calc != nil {
				lineHeight, _ := parent.Style.Strut()
				childBaselineY = baselineY - percentage(align, parent.Style, units.F(lineHeight)).Or(0)
			} else {
				// A length: the baseline of the child is that much above
				// the baseline of the parent.
				childBaselineY = baselineY - align.Dimension.Value
			}
		}

		top := childBaselineY - baselineOf(child)
		switch child.(type) {
		case *boxes.InlineBlockBox, *boxes.InlineFlexBox, *boxes.InlineGridBox:
			child.Translate(0, top-childBase.PositionY, false)
		default:
			childBase.PositionY = top
		}

		if align.Keyword == "top" || align.Keyword == "bottom" {
			*topBottomSubtrees = append(*topBottomSubtrees, child)
			continue
		}

		e.include(top, top+child.MarginHeight())
		if child.Kind() == boxes.KindInline {
			e.merge(inlineBoxVerticality(child, topBottomSubtrees, childBaselineY))
		}
	}
	return e
}
