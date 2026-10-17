package layout

import (
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// marginList is the list of margins that adjoin at one point of the flow
// (CSS 2.1 §8.3.1). Layout passes it by reference: a box that appends its
// margins to the list of its parent makes them part of the parent's own
// collapsing, which is how a top margin collapses through the first child.
type marginList struct{ values []float64 }

func (m *marginList) add(margin float64) { m.values = append(m.values, margin) }

// collapse returns the collapsed margin of the list; a nil list has none.
func (m *marginList) collapse() float64 {
	if m == nil {
		return 0
	}
	return collapseMargin(m.values)
}

func (m *marginList) isEmpty() bool { return m == nil || len(m.values) == 0 }

// collapseMargin returns the margin that adjoining margins collapse to: the
// largest of the positive ones added to the smallest of the negative ones
// (CSS 2.1 §8.3.1).
func collapseMargin(margins []float64) float64 {
	var positive, negative float64
	for _, m := range margins {
		if m >= 0 && m > positive {
			positive = m
		}
		if m <= 0 && m < negative {
			negative = m
		}
	}
	return positive + negative
}

// isBlockParallel reports whether a box takes part in breaks between block
// siblings: block-level boxes, row groups and rows.
func isBlockParallel(box boxes.Box) bool {
	if _, ok := box.(boxes.BlockLevel); ok {
		return true
	}
	kind := box.Kind()
	return kind == boxes.KindTableRowGroup || kind == boxes.KindTableRow
}

// addBreakValues collects the break-before or break-after values that
// meet at the edge of a box: the box's own and those of its first or last
// descendants in the flow (CSS Fragmentation 3 §3.1).
func addBreakValues(box boxes.Box, side string) []props.Keyword {
	var values []props.Keyword
	for box != nil && isBlockParallel(box) && (box.IsInNormalFlow() || box.IsFloated()) {
		if side == "after" {
			values = append(values, box.Base().Style.BreakAfter())
		} else {
			values = append(values, box.Base().Style.BreakBefore())
		}
		children := boxes.ChildrenOf(box)
		if len(children) == 0 {
			break
		}
		if side == "after" {
			box = children[len(children)-1]
		} else {
			box = children[0]
		}
	}
	if side == "after" {
		slices.Reverse(values)
	}
	return values
}

// blockLevelPageBreak returns the break value that wins for two siblings:
// the break-after values of the first and the break-before values of the
// second compete, forced breaks beat avoid, which beats auto, and among
// left, right, recto and verso the later value wins (CSS Fragmentation 3
// §3.1). A nil sibling takes no part.
func blockLevelPageBreak(before, after boxes.Box) props.Keyword {
	result := props.Keyword("auto")
	values := append(addBreakValues(before, "after"), addBreakValues(after, "before")...)
	for _, value := range values {
		if breakWins(value, result) {
			result = value
		}
	}
	return result
}

func breakWins(value, current props.Keyword) bool {
	switch value {
	case "left", "right", "recto", "verso":
		return true
	case "page", "column":
		switch current {
		case "auto", "avoid", "avoid-page", "avoid-column":
			return true
		}
	case "avoid", "avoid-page", "avoid-column":
		return current == "auto"
	}
	return false
}

// blockLevelPageName returns the name of the page that starts between two
// siblings when it differs from the one that ends, or nil when they agree
// (CSS Paged Media 3 §7.3.2).
func blockLevelPageName(before, after boxes.Box) *string {
	_, beforePage := before.PageValues()
	afterPage, _ := after.PageValues()
	if beforePage == afterPage {
		return nil
	}
	return &afterPage
}

// avoidPageBreak reports whether a break value asks to keep content
// together. Inside columns, column breaks count too.
func avoidPageBreak(pageBreak props.Keyword, ctx *Context) bool {
	if ctx.InColumn {
		return pageBreak == "avoid" || pageBreak == "avoid-page" || pageBreak == "avoid-column"
	}
	return pageBreak == "avoid" || pageBreak == "avoid-page"
}

// forcePageBreak reports whether a break value forces a break.
func forcePageBreak(pageBreak props.Keyword, ctx *Context) bool {
	switch pageBreak {
	case "page", "left", "right", "recto", "verso":
		return true
	case "column":
		return ctx.InColumn
	}
	return false
}

// findLastInFlowChild returns the last child that is in normal flow.
func findLastInFlowChild(children []boxes.Box) boxes.Box {
	for i := len(children) - 1; i >= 0; i-- {
		if children[i].IsInNormalFlow() {
			return children[i]
		}
	}
	return nil
}

// findLastClassABreakPointChild returns the last child that can be one of
// the siblings around a break between blocks (CSS Fragmentation 4 §3.1).
func findLastClassABreakPointChild(children []boxes.Box) boxes.Box {
	for i := len(children) - 1; i >= 0; i-- {
		if children[i].IsInNormalFlow() || children[i].IsFloated() {
			return children[i]
		}
	}
	return nil
}

// removePlaceholders forgets what a rejected attempt to lay out boxes left
// behind: the placeholders of their positioned descendants, their floats,
// their broken continuations, their footnotes and their running elements.
func (ctx *Context) removePlaceholders(list []boxes.Box, positioned *positionedBoxes) {
	for _, box := range list {
		placeholder, isPlaceholder := box.(*boxes.AbsolutePlaceholder)
		inner := box
		if isPlaceholder {
			inner = placeholder.Box
		}
		if children := boxes.ChildrenOf(inner); len(children) != 0 {
			ctx.removePlaceholders(children, positioned)
		}
		if isPlaceholder && positioned != nil {
			switch box.Base().Style.Position().Keyword {
			case "absolute":
				positioned.Absolute = removePlaceholder(positioned.Absolute, placeholder)
			case "fixed":
				positioned.Fixed = removePlaceholder(positioned.Fixed, placeholder)
			}
		}
		b := box.Base()
		if b.Footnote != nil {
			ctx.unlayoutFootnote(b.Footnote)
		}
		ctx.removeBrokenOutOfFlow(box)
		ctx.removeExcludedShape(box)
		if b.Note != nil {
			ctx.removeRunningElement(b.Note)
		}
	}
}

func removePlaceholder(list []*boxes.AbsolutePlaceholder, placeholder *boxes.AbsolutePlaceholder) []*boxes.AbsolutePlaceholder {
	if i := slices.Index(list, placeholder); i >= 0 {
		return slices.Delete(list, i, i+1)
	}
	return list
}

// findEarlierPageBreak looks for the last place where the page can break in
// children, because a break value or a lack of room rules out the one that
// was found. It returns the children that stay on the page and where the
// rest resumes; ok is false when there is no such place.
//
// The placeholders of the boxes left for the next page are removed from the
// lists of positioned boxes.
func findEarlierPageBreak(ctx *Context, children []boxes.Box, positioned *positionedBoxes) (kept []boxes.Box, resume ResumeAt, ok bool) {
	if len(children) != 0 && children[0].Kind() == boxes.KindLine {
		// Orphans and widows apply to the block container, but its line
		// boxes inherit them.
		style := children[0].Base().Style
		orphans, widows := int(style.Orphans()), int(style.Widows())
		index := len(children) - widows
		if index < orphans {
			return nil, nil, false
		}
		kept = slices.Clone(children[:index])
		resume = ResumeAt{0: ctx.lineResumeAt(kept[len(kept)-1])}
		ctx.removePlaceholders(children[index:], positioned)
		return kept, resume, true
	}

	var previousInFlow boxes.Box
	for index := len(children) - 1; ; index-- {
		if index < 0 {
			return nil, nil, false
		}
		child := children[index]
		if group, isGroup := child.(*boxes.TableRowGroupBox); isGroup && (group.IsHeader || group.IsFooter) {
			// Pages do not break before table headers or footers.
			continue
		}
		if child.Base().IsColumn {
			continue
		}

		if child.IsInNormalFlow() {
			pageBreak := blockLevelPageBreak(child, previousInFlow)
			if previousInFlow != nil && !avoidPageBreak(pageBreak, ctx) {
				index++
				kept = slices.Clone(children[:index])
				resume = ResumeAt{children[index].Base().Index: nil}
				ctx.removePlaceholders(children[index:], positioned)
				return kept, resume, true
			}
			previousInFlow = child
		}

		if !child.IsInNormalFlow() || avoidPageBreak(child.Base().Style.BreakInside(), ctx) {
			continue
		}
		parent, isParent := child.(boxes.Parent)
		if !isParent || !canHoldPageBreak(child.Kind()) {
			continue
		}
		grandChildren, grandResume, found := findEarlierPageBreak(ctx, parent.ParentBase().Children, positioned)
		if !found {
			continue
		}
		newChild := parent.CopyWithChildren(grandChildren)
		kept = append(slices.Clone(children[:index]), newChild)
		if child.Kind() == boxes.KindTableRowGroup {
			// The footer is repeated at the end of the split table.
			for _, next := range children[index:] {
				if group, isGroup := next.(*boxes.TableRowGroupBox); isGroup && group.IsFooter {
					kept = append(kept, next)
				}
			}
		}
		resume = ResumeAt{newChild.Base().Index: grandResume}
		ctx.removePlaceholders(children[index+1:], positioned)
		return kept, resume, true
	}
}

// canHoldPageBreak reports whether a box of the kind can be split between
// its children: blocks, tables and row groups.
func canHoldPageBreak(kind boxes.Kind) bool {
	switch kind {
	case boxes.KindBlock, boxes.KindTableCaption, boxes.KindFootnoteArea,
		boxes.KindTable, boxes.KindInlineTable, boxes.KindTableRowGroup:
		return true
	}
	return false
}

// relativePositioning moves a box that is positioned relatively by its
// offsets, and the relatively positioned boxes that are inline descendants of
// a line or an inline box (CSS 2.1 §9.4.3). The size of the containing block
// resolves the percentages of the offsets.
func relativePositioning(box boxes.Box, cb containingBlock) {
	b := box.Base()
	if b.Style.Position().Keyword == "relative" {
		resolvePositionPercentages(box, cb)
		var translateX, translateY float64
		switch {
		case !b.Left.Auto && !b.Right.Auto:
			if b.Style.Direction() == "ltr" {
				translateX = b.Left.V
			} else {
				translateX = -b.Right.V
			}
		case !b.Left.Auto:
			translateX = b.Left.V
		case !b.Right.Auto:
			translateX = -b.Right.V
		}
		switch {
		case !b.Top.Auto:
			translateY = b.Top.V
		case !b.Bottom.Auto:
			translateY = -b.Bottom.V
		}
		box.Translate(translateX, translateY, false)
	}
	if kind := box.Kind(); kind == boxes.KindInline || kind == boxes.KindLine {
		for _, child := range boxes.ChildrenOf(box) {
			relativePositioning(child, cb)
		}
	}
}

// setLineResumeAt records where the content after a line box resumes.
func (ctx *Context) setLineResumeAt(line boxes.Box, resume ResumeAt) {
	if ctx.lineResume == nil {
		ctx.lineResume = make(map[boxes.Box]ResumeAt)
	}
	ctx.lineResume[line] = resume
}

// lineResumeAt returns where the content after a line box resumes.
func (ctx *Context) lineResumeAt(line boxes.Box) ResumeAt { return ctx.lineResume[line] }
