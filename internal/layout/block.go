package layout

import (
	"fmt"
	"iter"
	"maps"
	"math"
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
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
// descendants in the flow (CSS Fragmentation 3, breaking between boxes).
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
// controlling breaks). A nil sibling takes no part.
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
// (CSS Paged Media 3, named pages).
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
// the siblings around a break between blocks (CSS Fragmentation 4, class A
// break points).
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

func isBlockBox(box boxes.Box) bool {
	switch box.Kind() {
	case boxes.KindBlock, boxes.KindTableCaption, boxes.KindFootnoteArea:
		return true
	}
	return false
}

func isTable(box boxes.Box) bool {
	kind := box.Kind()
	return kind == boxes.KindTable || kind == boxes.KindInlineTable
}

// withDefaults gives lists to the options that callers may leave empty.
func (o blockOptions) withDefaults() blockOptions {
	if o.Positioned == nil {
		o.Positioned = &positionedBoxes{}
	}
	if o.AdjoiningMargins == nil {
		o.AdjoiningMargins = &marginList{}
	}
	return o
}

// blockLevelLayout lays out a block-level box and returns the fragment that
// fits in the space left on the page. It sets the used values of the box
// that depend on its containing block, collapses its top margin with the
// ones adjoining it and applies clearance (CSS 2.1 §8.3.1, §9.5.2).
func blockLevelLayout(ctx *Context, box, containing boxes.Box, opts blockOptions) BlockResult {
	opts = opts.withDefaults()
	if !isTable(box) {
		resolvePercentages(box, containingBlockOf(containing))
		b := box.Base()
		if b.MarginTop.Auto {
			b.MarginTop = units.F(0)
		}
		if b.MarginBottom.Auto {
			b.MarginBottom = units.F(0)
		}

		if ctx.CurrentPage > 1 && opts.PageIsEmpty {
			// An unforced break before or after a block-level box truncates
			// the margins adjoining it (CSS Fragmentation 3, adjoining margins
			// at breaks). This only finds the margins when the parent separates
			// them from its ancestors, and not when an ancestor does.
			collapseWithPage := containing.Base().IsForRootElement || !opts.AdjoiningMargins.isEmpty()
			if collapseWithPage {
				switch b.Style.MarginBreak() {
				case "discard":
					b.MarginTop = units.F(0)
				case "auto":
					if !ctx.ForcedBreak {
						b.MarginTop = units.F(0)
					}
				}
			}
		}

		values := opts.AdjoiningMargins.values
		collapsed := collapseMargin(append(values[:len(values):len(values)], b.MarginTop.V))
		b.Clearance = getClearance(ctx, box, containing.Base().Style.Direction(), collapsed)
		if b.Clearance != nil {
			topBorderEdge := b.PositionY + collapsed + *b.Clearance
			b.PositionY = topBorderEdge - b.MarginTop.V
			opts.AdjoiningMargins = &marginList{}
		}
	}
	return blockLevelLayoutSwitch(ctx, box, containing, opts)
}

// blockLevelLayoutSwitch lays out a block-level box according to its type.
func blockLevelLayoutSwitch(ctx *Context, box, containing boxes.Box, opts blockOptions) BlockResult {
	switch box.Kind() {
	case boxes.KindBlock, boxes.KindTableCaption, boxes.KindFootnoteArea:
		return blockBoxLayout(ctx, box, containing, opts)
	case boxes.KindBlockReplaced:
		return blockReplacedBoxLayout(ctx, box.(*boxes.BlockReplacedBox), containing)
	case boxes.KindTable, boxes.KindInlineTable:
		ctx.unsupported("tables")
	case boxes.KindFlex:
		ctx.unsupported("flex containers")
	case boxes.KindGrid:
		ctx.unsupported("grid containers")
	default:
		ctx.fail(fmt.Errorf("layout of %s boxes is not handled", box.Kind()))
	}
	return BlockResult{}
}

// blockBoxLayout lays out a block box: it resolves its width, lays out its
// content and keeps a box that establishes a formatting context away from the
// floats beside it (CSS 2.1 §9.4.1).
func blockBoxLayout(ctx *Context, box, containing boxes.Box, opts blockOptions) BlockResult {
	b := box.Base()
	if !b.Style.ColumnWidth().IsAuto() || b.Style.ColumnCount().Keyword != "auto" {
		ctx.unsupported("multi-column layout")
		return BlockResult{}
	}
	if b.IsTableWrapper {
		ctx.unsupported("tables")
		return BlockResult{}
	}
	blockLevelWidth(b, containingBlockOf(containing), containing.Base().Style.Direction())

	result := blockContainerLayout(ctx, box, containing, opts)
	if result.Box == nil {
		return result
	}
	nb := result.Box.Base()
	if nb.IsColumn || nb.IsFlexItem {
		return result
	}
	if nb.IsTableWrapper || result.Box.EstablishesFormattingContext() {
		x, y, _ := avoidCollisions(ctx, result.Box, containing, false)
		result.Box.Translate(x-nb.PositionX, y-nb.PositionY, false)
	}
	return result
}

// blockReplacedBoxLayout lays out a replaced box that is block-level: its
// size follows CSS 2.1 §10.3.4 and §10.6.2, and it does not overlap floats
// (§9.5).
func blockReplacedBoxLayout(ctx *Context, box *boxes.BlockReplacedBox, containing boxes.Box) BlockResult {
	fragment := box.Copy().(*boxes.BlockReplacedBox)
	replaced := &fragment.ReplacedFields
	cb := containingBlockOf(containing)
	direction := containing.Base().Style.Direction()

	style := fragment.Style
	if style.Width().IsAuto() && style.Height().IsAuto() {
		marginLeft, marginRight := replaced.MarginLeft, replaced.MarginRight
		blockReplacedWidthWithoutMinMax(replaced, cb, direction)
		replacedBoxHeightWithoutMinMax(replaced)
		minMaxAutoReplaced(replaced)
		replaced.MarginLeft, replaced.MarginRight = marginLeft, marginRight
		blockLevelWidthWithoutMinMax(&replaced.BoxFields, cb, direction)
	} else {
		blockReplacedWidth(replaced, cb, direction)
		replacedBoxHeight(replaced)
	}

	if !fragment.IsFlexItem {
		replaced.PositionX, replaced.PositionY, _ = avoidCollisions(ctx, fragment, containing, false)
	}
	return BlockResult{
		Box:              fragment,
		NextPage:         PageBreak{Break: "any"},
		AdjoiningMargins: &marginList{},
	}
}

// overlayStyle returns the declarations of inner on top of those of outer,
// for boxes whose ::first-line or ::first-letter style adds to the one of an
// ancestor. Values that only outer declares are computed again against the
// parent of inner.
func overlayStyle(outer, inner *computed.Style) *computed.Style {
	if outer == nil {
		return inner
	}
	cascaded := make(computed.Cascaded, len(outer.Cascaded())+len(inner.Cascaded()))
	maps.Copy(cascaded, outer.Cascaded())
	maps.Copy(cascaded, inner.Cascaded())
	return computed.NewStyle(inner.Context(), inner.Parent(), inner.Root(), cascaded, inner.PseudoType(), inner.BaseURL())
}

// onlyResume returns the child index and the continuation inside the child
// of a continuation that points at a single child.
func onlyResume(resume ResumeAt) (index int, inner ResumeAt, ok bool) {
	if len(resume) != 1 {
		return 0, nil, false
	}
	for index, inner = range resume {
		return index, inner, true
	}
	return 0, nil, false
}

// containerFlow holds what the layout of the children of a block container
// shares between its steps.
type containerFlow struct {
	ctx         *Context
	box         boxes.Box
	pageIsEmpty bool
	positioned  *positionedBoxes
	bottomSpace float64
	discard     bool
}

// blockContainerLayout lays out the children of a block container and sets
// its height. It returns the fragment that fits in the space left on the
// page, and where the rest of the box resumes on the next one.
//
// A top margin that collapses with the ones of its first descendants moves
// the box once all of them are known, which is why the margins adjoining it
// are kept in a list that its descendants add theirs to.
func blockContainerLayout(ctx *Context, box, _ boxes.Box, opts blockOptions) BlockResult {
	container, isContainer := box.(boxes.BlockContainer)
	if !isContainer {
		ctx.fail(fmt.Errorf("layout of %s boxes as block containers is not handled", box.Kind()))
		return BlockResult{}
	}
	opts = opts.withDefaults()
	b := box.Base()
	style := b.Style

	establishesContext := box.EstablishesFormattingContext()
	if establishesContext {
		ctx.CreateBFC(box, nil)
	}
	finishContext := func(root boxes.Box) {
		if establishesContext {
			ctx.FinishBFC(root)
		}
	}

	skipStack := opts.SkipStack
	isStart := skipStack == nil
	box.RemoveDecoration(!isStart, false)

	discard := opts.Discard || style.Continue() == "discard"
	drawBottomDecoration := discard || style.BoxDecorationBreak() == "clone"

	bottomSpace := opts.BottomSpace
	if drawBottomDecoration {
		bottomSpace += b.PaddingBottom + b.BorderBottomWidth + b.MarginBottom.V
	}

	adjoining := opts.AdjoiningMargins
	adjoining.add(b.MarginTop.V)
	thisBoxAdjoining := adjoining

	var positionY float64
	if box.TopMarginCollapses() {
		// The margins adjoining the box are not counted: the box has
		// neither padding nor border.
		positionY = b.PositionY
	} else {
		b.PositionY += adjoining.collapse() - b.MarginTop.V
		adjoining = &marginList{}
		positionY = box.ContentBoxY()
	}
	positionX := box.ContentBoxX()

	positioned := opts.Positioned
	if style.Position().Keyword == "relative" {
		// The box is a new containing block for its absolute descendants.
		outer := positioned
		positioned = &positionedBoxes{Fixed: outer.Fixed}
		defer func() { outer.Fixed = positioned.Fixed }()
	}

	var newChildren []boxes.Box
	nextPage := PageBreak{Break: "any"}

	maxLines := opts.MaxLines
	if limit := style.MaxLines(); limit.Keyword != "none" {
		if maxLines == nil || *maxLines == 0 || limit.N < *maxLines {
			n := limit.N
			maxLines = &n
		}
	}

	firstLine, firstLetter := opts.FirstLineStyle, opts.FirstLetterStyle
	if b.FirstLineStyle != nil {
		firstLine = overlayStyle(firstLine, b.FirstLineStyle)
	}
	if b.FirstLetterStyle != nil {
		firstLetter = overlayStyle(firstLetter, b.FirstLetterStyle)
	}

	skip := 0
	if !isStart {
		var ok bool
		if skip, skipStack, ok = onlyResume(skipStack); !ok {
			ctx.fail(fmt.Errorf("invalid continuation of a %s box", box.Kind()))
			finishContext(nil)
			return BlockResult{}
		}
	}

	flow := &containerFlow{
		ctx:         ctx,
		box:         box,
		pageIsEmpty: opts.PageIsEmpty,
		positioned:  positioned,
		bottomSpace: bottomSpace,
		discard:     discard,
	}
	children := container.ParentBase().Children
	var resumeAt ResumeAt
	stopped := false
	for index := skip; index < len(children); index++ {
		child := children[index]
		child.Base().PositionX = positionX
		// The position does not count the adjoining margins.
		child.Base().PositionY = positionY
		var abort, stop bool
		var newFootnotes []boxes.Box

		if child.IsInNormalFlow() || child.IsFloated() {
			// A break between siblings may be forced, or move to a page of
			// another name.
			if last := findLastClassABreakPointChild(newChildren); last != nil {
				pageBreak := blockLevelPageBreak(last, child)
				pageName := blockLevelPageName(last, child)
				if (pageName != nil && *pageName != "") || forcePageBreak(pageBreak, ctx) {
					nextPage = PageBreak{Break: string(pageBreak), Page: pageName}
					resumeAt = ResumeAt{index: nil}
					stop = true
				}
			}
		}

		switch {
		case stop:
		case !child.IsInNormalFlow():
			var newChild boxes.Box
			var outOfFlowResume ResumeAt
			stop, resumeAt, newChild, outOfFlowResume = flow.outOfFlowLayout(index, child, &newChildren, adjoining)
			if len(outOfFlowResume) != 0 {
				ctx.addBrokenOutOfFlow(newChild, child, box, outOfFlowResume)
			}
			if child.Base().IsOutsideMarker && newChild != nil {
				marker := newChild.Base()
				marker.PositionX = box.BorderBoxX()
				if child.Base().Style.Direction() == "rtl" {
					marker.PositionX += b.Width.V + b.PaddingRight
				}
			}

		case child.Kind() == boxes.KindLine:
			line := flow.lineboxLayout(index, child, &newChildren, adjoining, positionY, skipStack,
				firstLetter, firstLine, drawBottomDecoration, maxLines)
			abort, stop, resumeAt, positionY = line.abort, line.stop, line.resumeAt, line.positionY
			newFootnotes, maxLines = line.newFootnotes, line.maxLines
			drawBottomDecoration = drawBottomDecoration || resumeAt == nil
			adjoining = &marginList{}
			firstLetter, firstLine = nil, nil

		default:
			result := flow.inFlowLayout(index, child, &newChildren, adjoining, positionY, skipStack,
				firstLetter, firstLine, maxLines)
			abort, stop, resumeAt, positionY = result.abort, result.stop, result.resumeAt, result.positionY
			adjoining, nextPage = result.adjoining, result.nextPage
			skipStack = nil
			firstLetter, firstLine = nil, nil

			if result.maxLines != nil && maxLines != nil {
				maxLines = result.maxLines
				if *maxLines <= 0 {
					// The box has all the lines it can show.
					stop = true
					if child != children[len(children)-1] {
						// The last line shown gets the ellipsis.
						setBlockEllipsis(newChildren, style.BlockEllipsis())
					}
				}
			}
		}

		if abort {
			pageStart, _ := child.PageValues()
			ctx.removePlaceholders(children[skip:], positioned)
			for _, footnote := range newFootnotes {
				ctx.unlayoutFootnote(footnote)
			}
			finishContext(nil)
			return BlockResult{
				NextPage:         PageBreak{Break: "any", Page: &pageStart},
				AdjoiningMargins: &marginList{},
				MaxLines:         maxLines,
			}
		}
		if stop {
			if !b.Height.Auto {
				boxBottom := b.PositionY + box.BorderHeight()
				bottomMargin := adjoining.collapse()
				if overflows(boxBottom, positionY-bottomMargin) {
					// The height is fixed and the box overflows the page:
					// what overflows is forgotten.
					resumeAt = nil
				}
			}
			adjoining = &marginList{}
			stopped = true
			break
		}
	}
	if !stopped {
		resumeAt = nil
	}

	fragmented := resumeAt != nil || b.ForceFragmentation
	if style.Continue() == "discard" {
		resumeAt = nil
	}

	if fragmented && avoidPageBreak(style.BreakInside(), ctx) && !opts.PageIsEmpty {
		ctx.removePlaceholders(append(slices.Clone(newChildren), children[skip:]...), positioned)
		finishContext(nil)
		return BlockResult{
			NextPage:         PageBreak{Break: "any"},
			AdjoiningMargins: &marginList{},
			MaxLines:         maxLines,
		}
	}

	if box.TopMarginCollapses() {
		b.PositionY += thisBoxAdjoining.collapse() - b.MarginTop.V
	}

	// Detect a box that margins collapse through (CSS 2.1 §8.3.1).
	collapsingThrough := false
	if findLastInFlowChild(newChildren) == nil {
		// There is no in-flow child: the top and the bottom margins of the
		// box collapse together.
		collapsed := adjoining.collapse()
		if (b.Height.Auto || b.Height.V == 0) &&
			getClearance(ctx, box, style.Direction(), collapsed) == nil &&
			b.MinHeight.V == 0 && b.BorderTopWidth == 0 && b.PaddingTop == 0 &&
			b.BorderBottomWidth == 0 && b.PaddingBottom == 0 {
			collapsingThrough = true
		} else {
			positionY += collapsed
			adjoining = &marginList{}
		}
	} else if !b.Height.Auto {
		// The bottom margin of the last child does not adjoin the one of the
		// box when the height is fixed.
		adjoining = &marginList{}
	}

	if !box.BottomMarginCollapses() {
		positionY += adjoining.collapse()
		adjoining = &marginList{}
	}

	if fragmented && len(newChildren) != 0 {
		setBlockEllipsis(newChildren[len(newChildren)-1:], style.BlockEllipsis())
	}

	newBox := container.CopyWithChildren(newChildren)
	nb := newBox.Base()
	newBox.RemoveDecoration(!isStart, fragmented && !discard)

	if nb.Height.Auto {
		nb.Height = units.F(firstMax(0, positionY-newBox.ContentBoxY()))
	}

	if nb.Style.Position().Keyword == "relative" {
		// The box is a new containing block: its absolute descendants can
		// now be placed.
		for _, placeholder := range positioned.Absolute {
			absoluteLayout(ctx, placeholder, newBox, &positioned.Fixed, bottomSpace, nil)
		}
	}

	cb := containingBlock{Width: nb.Width, Height: nb.Height}
	for _, child := range boxes.ChildrenOf(newBox) {
		relativePositioning(child, cb)
	}

	finishContext(newBox)

	switch {
	case discard || !fragmented:
		// After the formatting context, which may make the box higher.
		nb.Height = units.F(firstMax(firstMin(nb.Height.V, nb.MaxHeight.V), nb.MinHeight.V))
	case !math.IsInf(bottomSpace, -1) && !nb.IsColumn:
		// The box fills the blank space at the bottom of the page
		// (CSS Fragmentation 3, splitting boxes).
		height := ctx.PageBottom - bottomSpace - nb.PositionY - (newBox.MarginHeight() - nb.Height.V)
		if height > nb.Height.V {
			nb.Height = units.F(height)
			if drawBottomDecoration {
				nb.Height = units.F(nb.Height.V + b.PaddingBottom + b.BorderBottomWidth + b.MarginBottom.V)
			}
		}
	}

	if nextPage.Page == nil {
		_, end := newBox.PageValues()
		nextPage.Page = &end
	}

	return BlockResult{
		Box:               newBox,
		ResumeAt:          resumeAt,
		NextPage:          nextPage,
		AdjoiningMargins:  adjoining,
		CollapsingThrough: collapsingThrough,
		MaxLines:          maxLines,
	}
}

// setBlockEllipsis sets the ellipsis of block-ellipsis on the last line box
// found by following the last children of the boxes.
func setBlockEllipsis(children []boxes.Box, ellipsis props.KeywordOrString) {
	for len(children) != 0 {
		last := children[len(children)-1]
		if line, ok := last.(*boxes.LineBox); ok {
			line.BlockEllipsis = ellipsis
			return
		}
		parent, ok := last.(boxes.Parent)
		if !ok {
			return
		}
		children = parent.ParentBase().Children
	}
}

// outOfFlowLayout lays out a child that is not in the normal flow: it leaves
// a placeholder for an absolute box, lays out a float, and keeps a running
// element for the margin boxes. It reports whether the parent must stop after
// it, where the in-flow layout resumes, the fragment laid out, and where the
// fragment of an out-of-flow box that did not fit resumes.
func (f *containerFlow) outOfFlowLayout(index int, child boxes.Box, newChildren *[]boxes.Box, adjoining *marginList) (stop bool, resume ResumeAt, newChild boxes.Box, outOfFlowResume ResumeAt) {
	ctx := f.ctx

	// The margins of the parent shift the child. The top margin of the child
	// does not collapse with its parent, so it does not take part.
	child.Base().PositionY += adjoining.collapse()

	switch {
	case child.IsAbsolutelyPositioned():
		placeholder := boxes.NewAbsolutePlaceholder(child)
		placeholder.Base().Index = index
		*newChildren = append(*newChildren, placeholder)
		if child.Base().Style.Position().Keyword == "absolute" {
			f.positioned.Absolute = append(f.positioned.Absolute, placeholder)
		} else {
			f.positioned.Fixed = append(f.positioned.Fixed, placeholder)
		}
		newChild = placeholder

	case child.IsFloated():
		result := floatLayout(ctx, child, f.box, f.positioned, f.bottomSpace, nil)
		newChild, outOfFlowResume = result.Box, result.ResumeAt
		if newChild == nil {
			return false, nil, nil, nil
		}
		nc := newChild.Base()
		pageOverflow := ctx.OverflowsPage(f.bottomSpace, nc.PositionY+nc.Height.V)
		if (f.pageIsEmpty && len(*newChildren) == 0) || !pageOverflow || f.box.IsMonolithic() {
			// The float fits, or has to be placed anyway.
			nc.Index = index
			*newChildren = append(*newChildren, newChild)
			break
		}
		// The float does not fit and the page can break: find where, and
		// stop the parent.
		pageBreak := blockLevelPageBreak(findLastClassABreakPointChild(*newChildren), child)
		resume = ResumeAt{index: nil}
		outOfFlowResume = nil
		stop = true
		ctx.removePlaceholders([]boxes.Box{newChild}, f.positioned)
		if len(*newChildren) != 0 && avoidPageBreak(pageBreak, ctx) {
			// A float cannot be split: look for an earlier break.
			if kept, earlier, ok := findEarlierPageBreak(ctx, *newChildren, f.positioned); ok {
				*newChildren, resume = kept, earlier
				newChild = nil
			}
		}

	case child.IsRunning() || child.IsNote():
		ctx.addRunningElement(child)
	}
	return stop, resume, newChild, outOfFlowResume
}

// breakLine decides how the lines before an overflowing one end the page,
// given the orphans and widows of the box. It reports whether the whole box
// moves to the next page, and whether the page ends here, and where the
// content resumes.
func (f *containerFlow) breakLine(line boxes.Box, newChildren *[]boxes.Box, needed, index int, skipStack, resumeAt ResumeAt) (abort, stop bool, resume ResumeAt) {
	ctx := f.ctx
	overOrphans := len(*newChildren) - int(f.box.Base().Style.Orphans())
	if overOrphans < 0 && !f.pageIsEmpty {
		// The page filled up before the lines needed for orphans: the whole
		// box moves.
		ctx.removePlaceholders(boxes.ChildrenOf(line), f.positioned)
		return true, false, resumeAt
	}
	// The lines that the next page needs for widows, not counting this one.
	if needed > overOrphans && !f.pageIsEmpty {
		// There are fewer lines than orphans and widows together.
		ctx.removePlaceholders(boxes.ChildrenOf(line), f.positioned)
		return true, false, resumeAt
	}
	if needed != 0 && needed <= overOrphans {
		// The page gives lines to the next one to satisfy widows.
		kept := len(*newChildren) - needed
		for _, moved := range (*newChildren)[kept:] {
			ctx.removePlaceholders(boxes.ChildrenOf(moved), f.positioned)
		}
		*newChildren = (*newChildren)[:kept]
	}
	ctx.removePlaceholders(boxes.ChildrenOf(line), f.positioned)
	return false, true, ResumeAt{index: skipStack}
}

type lineboxResult struct {
	abort, stop  bool
	resumeAt     ResumeAt
	positionY    float64
	newFootnotes []boxes.Box
	maxLines     *int
}

// lineboxLayout lays out the lines of a block container, from the line box
// that holds all of its inline content, until the lines stop fitting on the
// page.
func (f *containerFlow) lineboxLayout(index int, child boxes.Box, newChildren *[]boxes.Box, adjoining *marginList, positionY float64, skipStack ResumeAt, firstLetter, firstLine *computed.Style, drawBottomDecoration bool, maxLines *int) lineboxResult {
	ctx, box := f.ctx, f.box
	b := box.Base()
	result := lineboxResult{maxLines: maxLines, positionY: positionY}

	if len(boxes.ChildrenOf(box)) != 1 {
		ctx.fail(fmt.Errorf("line box with siblings before layout"))
		return result
	}
	if !adjoining.isEmpty() {
		positionY += adjoining.collapse()
	}
	lines := iterLineBoxes(ctx, child, box, blockOptions{
		BottomSpace:      f.bottomSpace,
		SkipStack:        skipStack,
		Positioned:       f.positioned,
		FirstLetterStyle: firstLetter,
		FirstLineStyle:   firstLine,
	}, positionY)
	nextLine, stopLines := iter.Pull(lines)
	defer stopLines()

	var resumeAt ResumeAt
	for {
		next, ok := nextLine()
		if !ok {
			break
		}
		line, resume := next.Line, next.ResumeAt
		resumeAt = resume

		// The box is cut when it reaches max-lines.
		if result.maxLines != nil {
			if *result.maxLines == 0 {
				setBlockEllipsis(*newChildren, b.Style.BlockEllipsis())
				break
			}
			left := *result.maxLines - 1
			result.maxLines = &left
		}

		ctx.setLineResumeAt(line, resume)
		newPositionY := line.PositionY + line.Height.V

		// The bottom padding and border end the box when the line is its last.
		drawBottomDecoration = drawBottomDecoration || resume == nil
		var offsetY float64
		if drawBottomDecoration {
			offsetY = b.BorderBottomWidth + b.PaddingBottom
		}

		// The first line of a page may be higher than the page itself: it is
		// kept anyway, so that the content advances.
		overflow := (len(*newChildren) != 0 || !f.pageIsEmpty) && ctx.OverflowsPage(f.bottomSpace, newPositionY+offsetY)
		if overflow {
			couldBreakBefore, canBreakNow := true, true
			needed := int(b.Style.Widows()) - 1
			for {
				following, ok := nextLine()
				if !ok {
					break
				}
				needed--
				ctx.removePlaceholders(boxes.ChildrenOf(following.Line), f.positioned)
				// Not all the lines are laid out, as there can be many.
				if needed == -1 {
					break
				}
			}
			orphans := int(b.Style.Orphans())
			if len(*newChildren)+1 < orphans {
				canBreakNow = false
			} else if needed >= 0 {
				canBreakNow = false
			}
			if len(*newChildren) < orphans {
				couldBreakBefore = false
			} else if needed > 0 {
				couldBreakBefore = false
			}
			needed = max(0, needed)

			// If the line could not end the page before but can now, report
			// footnotes first and see whether the line then fits.
			report := !ctx.InColumn && canBreakNow && !couldBreakBefore
			reported := 0
			fitsAfterReport := false
			for report && len(ctx.CurrentPageFootnotes) != 0 {
				ctx.reportFootnote(ctx.CurrentPageFootnotes[len(ctx.CurrentPageFootnotes)-1])
				reported++
				if !ctx.OverflowsPage(f.bottomSpace, newPositionY+offsetY) {
					*newChildren = append(*newChildren, line)
					result.stop = true
					fitsAfterReport = true
					break
				}
			}
			if !fitsAfterReport {
				result.abort, result.stop, resumeAt = f.breakLine(line, newChildren, needed, index, skipStack, resumeAt)
			}

			// Footnotes were reported last first: restore their order.
			if reported >= 2 {
				slices.Reverse(ctx.ReportedFootnotes[len(ctx.ReportedFootnotes)-reported:])
			}
			break
		} else if f.pageIsEmpty && ctx.OverflowsPage(f.bottomSpace, newPositionY) {
			// The page is empty and the box too high for it: the top margin
			// goes, so that as much as possible of the box is on the page.
			newPositionY -= b.MarginTop.V
			line.Translate(0, -b.MarginTop.V, false)
			b.MarginTop = units.F(0)
		}

		if len(ctx.Footnotes) != 0 {
			breakLine := false
			for descendant := range boxes.Descendants(line, false) {
				footnote := descendant.Base().Footnote
				if footnote == nil || !slices.Contains(ctx.Footnotes, footnote) {
					continue
				}
				overflow := ctx.layoutFootnote(footnote)
				result.newFootnotes = append(result.newFootnotes, footnote)
				overflow = overflow || len(ctx.ReportedFootnotes) != 0 ||
					ctx.OverflowsPage(f.bottomSpace, newPositionY+offsetY)
				if !overflow {
					continue
				}
				ctx.reportFootnote(footnote)
				// With other content on the page the line or the block can move
				// to the next one. Without, it cannot, and trying would loop
				// forever.
				if len(*newChildren) == 0 && f.pageIsEmpty {
					continue
				}
				switch footnote.Base().Style.FootnotePolicy() {
				case "line":
					needed := int(b.Style.Widows()) - 1
					for needed != 0 {
						if _, ok := nextLine(); !ok {
							break
						}
						needed--
						// Not all the lines are laid out, as there can be many.
					}
					result.abort, result.stop, resumeAt = f.breakLine(line, newChildren, needed, index, skipStack, resumeAt)
					breakLine = true
				case "block":
					result.abort = true
					breakLine = true
				}
				if breakLine {
					break
				}
			}
			if breakLine {
				break
			}
		}

		*newChildren = append(*newChildren, line)
		positionY = newPositionY
		skipStack = resume
	}

	if len(*newChildren) != 0 {
		last := (*newChildren)[len(*newChildren)-1]
		resumeAt = ResumeAt{index: ctx.lineResumeAt(last)}
	}
	result.resumeAt = resumeAt
	result.positionY = positionY
	return result
}

type inFlowResult struct {
	abort, stop bool
	resumeAt    ResumeAt
	positionY   float64
	adjoining   *marginList
	nextPage    PageBreak
	maxLines    *int
}

// inFlowLayout lays out a block-level child in normal flow. It decides
// where the page breaks when the child does not fit, searching for an
// earlier opportunity when the break values forbid the one at hand.
func (f *containerFlow) inFlowLayout(index int, child boxes.Box, newChildren *[]boxes.Box, adjoining *marginList, positionY float64, skipStack ResumeAt, firstLetter, firstLine *computed.Style, maxLines *int) inFlowResult {
	ctx, box := f.ctx, f.box
	b := box.Base()
	bottomSpace := f.bottomSpace
	result := inFlowResult{}

	if !b.IsTableWrapper {
		resolvePercentages(child, containingBlockOf(box))
		if findLastInFlowChild(*newChildren) == nil && box.TopMarginCollapses() {
			// The top margin of the first child collapses with the margins
			// adjoining the box, which moves the children already laid out.
			oldCollapsed := adjoining.collapse()
			childMarginTop := child.Base().MarginTop
			var marginTop float64
			if !childMarginTop.Auto {
				marginTop = childMarginTop.V
				if ctx.CurrentPage > 1 && f.pageIsEmpty {
					switch b.Style.MarginBreak() {
					case "discard":
						marginTop = 0
					case "auto":
						if !ctx.ForcedBreak {
							marginTop = 0
						}
					}
				}
			}
			values := adjoining.values
			newCollapsed := collapseMargin(append(values[:len(values):len(values)], marginTop))
			difference := newCollapsed - oldCollapsed
			for _, previous := range *newChildren {
				previous.Translate(0, difference, false)
			}
			clearance := getClearance(ctx, child, b.Style.Direction(), newCollapsed)
			if clearance != nil {
				for _, previous := range *newChildren {
					previous.Translate(0, -difference, false)
				}
				collapsed := adjoining.collapse()
				b.PositionY += collapsed - b.MarginTop.V
				// The top margin of the box counts, as the adjoining margins
				// are emptied.
				adjoining = &marginList{}
				positionY = box.ContentBoxY()
			}
		}
	}

	if !adjoining.isEmpty() {
		if b.IsTableWrapper {
			child.Base().PositionY += adjoining.collapse()
			adjoining = &marginList{}
		} else if !isBlockBox(child) {
			// Blocks collapse their margins themselves.
			var marginTop float64
			if length := child.Base().Style.MarginTop(); !length.IsAuto() {
				marginTop = Percentage(length, child.Base().Style, b.Width.V).Or(0)
			}
			adjoining.add(marginTop)
			child.Base().PositionY += adjoining.collapse() - marginTop
			adjoining = &marginList{}
		}
	}

	pageIsEmptyWithNoChildren := f.pageIsEmpty && !slices.ContainsFunc(*newChildren, func(c boxes.Box) bool {
		_, isPlaceholder := c.(*boxes.AbsolutePlaceholder)
		return !isPlaceholder
	})

	childX, childY := child.Base().PositionX, child.Base().PositionY
	layout := func(bottomSpace float64, maxLines *int) BlockResult {
		return blockLevelLayout(ctx, child, box, blockOptions{
			BottomSpace:      bottomSpace,
			SkipStack:        skipStack,
			PageIsEmpty:      pageIsEmptyWithNoChildren,
			Positioned:       f.positioned,
			AdjoiningMargins: adjoining,
			FirstLetterStyle: firstLetter,
			FirstLineStyle:   firstLine,
			Discard:          f.discard,
			MaxLines:         maxLines,
		})
	}
	laid := layout(bottomSpace, maxLines)
	newChild, resumeAt := laid.Box, laid.ResumeAt
	maxLines = laid.MaxLines

	if newChild != nil {
		if !laid.CollapsingThrough {
			nc := newChild.Base()
			newContentY := newChild.ContentBoxY() + nc.Height.V
			contentOverflow := ctx.OverflowsPage(bottomSpace, newContentY)

			// The padding, the border and the margin of the child take room
			// at the bottom of the page too.
			bottomSpace += nc.PaddingBottom + nc.BorderBottomWidth
			if !box.BottomMarginCollapses() {
				bottomSpace += nc.MarginBottom.V
			}
			newPositionY := newChild.BorderBoxY() + newChild.BorderHeight()
			borderOverflow := ctx.OverflowsPage(bottomSpace, newPositionY)

			canBreak := !(pageIsEmptyWithNoChildren || box.IsMonolithic())
			switch {
			case canBreak && contentOverflow:
				// The content of the child overflows the page: it goes to the
				// next one.
				ctx.removePlaceholders([]boxes.Box{newChild}, f.positioned)
				newChild = nil
			case canBreak && borderOverflow:
				// Only the border, the padding or the margin overflow: lay
				// the child out again with room for them.
				ctx.removePlaceholders([]boxes.Box{newChild}, f.positioned)
				child.Base().PositionX, child.Base().PositionY = childX, childY
				laid = layout(bottomSpace, maxLines)
				newChild, resumeAt, maxLines = laid.Box, laid.ResumeAt, laid.MaxLines
				if newChild != nil {
					positionY = newChild.BorderBoxY() + newChild.BorderHeight()
				}
			default:
				positionY = newPositionY
			}
		}

		adjoining = laid.AdjoiningMargins
		if newChild != nil {
			adjoining.add(newChild.Base().MarginBottom.V)
		}
		if newChild != nil {
			if clearance := newChild.Base().Clearance; clearance != nil && *clearance != 0 {
				positionY = newChild.BorderBoxY() + newChild.BorderHeight()
			}
		}
	}
	result.nextPage = laid.NextPage
	result.maxLines = maxLines
	result.adjoining = adjoining
	result.positionY = positionY

	if newChild == nil {
		// Nothing fits in the space left on the page: break.
		pageBreak := props.Keyword("auto")
		if last := findLastClassABreakPointChild(*newChildren); last != nil {
			pageBreak = blockLevelPageBreak(last, child)
		}
		if avoidPageBreak(pageBreak, ctx) {
			if kept, earlier, ok := findEarlierPageBreak(ctx, *newChildren, f.positioned); ok {
				*newChildren = kept
				result.resumeAt = earlier
				result.stop = true
				return result
			}
			if !f.pageIsEmpty {
				// The page has content before this block: cancel the block
				// and look for a break in the parent.
				result.abort = true
				result.resumeAt = resumeAt
				return result
			}
			// Otherwise ignore the avoid and break anyway.
		}

		onlyAbsolute := true
		for _, c := range *newChildren {
			if !c.IsAbsolutelyPositioned() {
				onlyAbsolute = false
				break
			}
		}
		if onlyAbsolute {
			// The box has only absolute children so far, which stay for the
			// next page, as list markers do.
			ctx.removePlaceholders(*newChildren, f.positioned)
			*newChildren = nil
		}
		result.resumeAt = resumeAt
		if len(*newChildren) != 0 {
			// The children laid out stay, and the box ends here.
			result.resumeAt = ResumeAt{index: nil}
			result.stop = true
		} else {
			// The first child did not fit: cancel the box.
			result.abort = true
		}
		return result
	}

	// The index in the parent before layout, which findEarlierPageBreak uses.
	newChild.Base().Index = index
	*newChildren = append(*newChildren, newChild)
	result.resumeAt = resumeAt
	if resumeAt != nil {
		result.resumeAt = ResumeAt{index: resumeAt}
		result.stop = true
	}
	return result
}
