package layout

import (
	"fmt"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// Atomic inline-level boxes behave like ideographic characters when lines
// are broken (CSS Text 3 §5.1).
var atomicBoundary = boundary{Kind: characterBoundary, Rune: '\u2e80'}

// collapsibleSpace marks the end of a box whose last space was collapsed
// away: a line can break after it. The same value marks the start of a
// leader, which can always be preceded by a break.
var collapsibleSpace = boundary{Kind: leaderStartBoundary}

// inlineChild is a child of an inline box that was laid out: its position
// among its siblings, the laid-out box and the box it was made from.
type inlineChild struct {
	index    int
	box      boxes.Box
	original boxes.Box
}

// withDefaults gives a fresh list to every shared list that is missing, so
// that options built by hand work as those built by the line layout.
func (o inlineOptions) withDefaults() inlineOptions {
	if o.Positioned == nil {
		o.Positioned = &positionedBoxes{}
	}
	if o.LinePlaceholders == nil {
		o.LinePlaceholders = new([]*boxes.AbsolutePlaceholder)
	}
	if o.WaitingFloats == nil {
		o.WaitingFloats = new([]boxes.Box)
	}
	if o.LineChildren == nil {
		o.LineChildren = new([]boxes.Box)
	}
	return o
}

// splitInlineLevel fits as much of an inline-level box as possible between
// o.PositionX and o.MaxX. The box in the result is not empty, unless the
// box is, and is as large as possible while respecting o.MaxX; it overflows
// when no split is possible. ResumeAt is nil when all the content fits and
// otherwise resumes where the content was cut.
func splitInlineLevel(ctx *Context, box, containing boxes.Box, o inlineOptions) InlineResult {
	o = o.withDefaults()
	if hasDeclarations(o.FirstLineStyle) {
		box = box.Copy()
		base := box.Base()
		base.Style = base.Style.Copy()
		applyDeclared(base.Style, o.FirstLineStyle, true)
		boxes.ProcessTextTransform(box)
	}
	resolvePercentages(box, containingBlockOf(containing))

	switch b := box.(type) {
	case *boxes.TextBox:
		return splitTextLevel(ctx, b, o)
	case *boxes.InlineBox:
		for _, margin := range []*units.MaybeFloat{&b.MarginLeft, &b.MarginRight} {
			if margin.Auto {
				*margin = units.F(0)
			}
		}
		return splitInlineBox(ctx, b, containing, o)
	case boxes.AtomicInline:
		atomic := atomicBox(ctx, b, o.PositionX, o.SkipStack, containing, o.Positioned)
		atomic.Base().PositionX = o.PositionX
		return InlineResult{Box: atomic, FirstLetter: atomicBoundary, LastLetter: atomicBoundary}
	case *boxes.InlineFlexBox, *boxes.InlineGridBox:
		ctx.unsupported(fmt.Sprintf("%s boxes", box.Kind()))
		return InlineResult{Box: box, FirstLetter: atomicBoundary, LastLetter: atomicBoundary}
	}
	ctx.fail(fmt.Errorf("layout of %s boxes in a line is not supported", box.Kind()))
	return InlineResult{}
}

// splitTextLevel fits as much of a text box as possible on a line, and
// reports its first and last letters, which decide whether the line can
// break next to the boxes around it.
func splitTextLevel(ctx *Context, box *boxes.TextBox, o inlineOptions) InlineResult {
	box.PositionX = o.PositionX
	skip := 0
	if o.SkipStack != nil {
		skip, _ = firstResume(o.SkipStack)
	}
	isLineStart := len(*o.LineChildren) == 0
	split := splitTextBox(ctx, box, o.MaxX-o.PositionX, skip, isLineStart)

	result := InlineResult{PreservedLineBreak: split.PreservedLineBreak}
	if split.Box != nil {
		result.Box = split.Box
	}
	if split.ResumeIndex != nil {
		result.ResumeAt = ResumeAt{*split.ResumeIndex: nil}
	}
	if box.Text != "" {
		first, _ := utf8.DecodeRuneInString(box.Text)
		result.FirstLetter = boundary{Kind: characterBoundary, Rune: first}
		consumed := box.Text
		if split.ResumeIndex != nil {
			consumed = box.Text[:min(*split.ResumeIndex, len(box.Text))]
		}
		last, _ := utf8.DecodeLastRuneInString(consumed)
		result.LastLetter = boundary{Kind: characterBoundary, Rune: last}
	}
	return result
}

// atomicBox lays out an atomic inline-level box: a replaced element or an
// inline-block. Its baseline is relative to the top of its margin box.
func atomicBox(
	ctx *Context,
	box boxes.Box,
	positionX float64,
	skip ResumeAt,
	containing boxes.Box,
	positioned *positionedBoxes,
) boxes.Box {
	switch b := box.(type) {
	case *boxes.InlineReplacedBox:
		replaced := b.Copy().(*boxes.InlineReplacedBox)
		inlineReplacedBoxLayout(&replaced.ReplacedFields, containingBlockOf(containing), containing.Base().Style.Direction())
		baseline := replaced.MarginHeight()
		replaced.Baseline = &baseline
		return replaced
	case *boxes.InlineBlockBox:
		if b.IsTableWrapper {
			ctx.unsupported("inline tables")
			return box
		}
		return inlineBlockBoxLayout(ctx, b, containing, positionX, skip, positioned)
	}
	ctx.fail(fmt.Errorf("layout of %s as an atomic inline-level box is not supported", box.Kind()))
	return box
}

// inlineBlockBoxLayout lays out an inline-block as a block container with a
// width that shrinks to fit (CSS 2.1 §10.3.9). Its height is never limited
// by the page: it is placed whole on a line.
func inlineBlockBoxLayout(
	ctx *Context,
	box boxes.BlockContainer,
	containing boxes.Box,
	positionX float64,
	skip ResumeAt,
	positioned *positionedBoxes,
) boxes.Box {
	resolvePercentages(box, containingBlockOf(containing))

	// CSS 2.1 §10.3.9 and §10.6.6: auto margins of inline blocks are zero.
	b := box.Base()
	for _, margin := range []*units.MaybeFloat{&b.MarginLeft, &b.MarginRight, &b.MarginTop, &b.MarginBottom} {
		if margin.Auto {
			*margin = units.F(0)
		}
	}

	inlineBlockWidth(ctx, box, containing)

	b.PositionX = positionX
	b.PositionY = 0
	result := blockContainerLayout(ctx, box, containing, blockOptions{
		BottomSpace: math.Inf(-1),
		SkipStack:   skip,
		PageIsEmpty: true,
		Positioned:  positioned,
	})
	if result.Box == nil {
		// The layout failed and recorded why.
		return box
	}
	baseline := inlineBlockBaseline(result.Box)
	result.Box.Base().Baseline = &baseline
	return result.Box
}

// inlineBlockBaseline returns the position of the baseline of an
// inline-block from the top of its margin box: the baseline of its last
// line, unless it clips its content, and then the bottom margin edge
// (CSS 2.1 §10.8.1).
func inlineBlockBaseline(box boxes.Box) float64 {
	b := box.Base()
	if b.Style.Overflow() == "visible" {
		if baseline := findInFlowBaseline(box, true); baseline != nil && *baseline != 0 {
			return *baseline
		}
	}
	return b.PositionY + box.MarginHeight()
}

// findInFlowBaseline returns the absolute position of the baseline of the
// first or, with last, the last line of a box in normal flow, or nil when
// it has none.
func findInFlowBaseline(box boxes.Box, last bool) *float64 {
	switch box.Kind() {
	case boxes.KindLine:
		baseline := box.Base().PositionY + baselineOf(box)
		return &baseline
	case boxes.KindTableCaption:
		return nil
	}
	children := boxes.ChildrenOf(box)
	for i := range children {
		child := children[i]
		if last {
			child = children[len(children)-1-i]
		}
		if !child.IsInNormalFlow() {
			continue
		}
		if baseline := findInFlowBaseline(child, last); baseline != nil {
			return baseline
		}
	}
	return nil
}

// inlineBlockWidth resolves the width of an inline-block: when it is auto,
// the available width within its minimum and maximum content widths.
func inlineBlockWidth(ctx *Context, box boxes.Box, containing boxes.Box) {
	b := box.Base()
	withMinMaxWidth(b, func() {
		available := containing.Base().Width.V - (b.MarginLeft.V + b.MarginRight.V +
			b.BorderLeftWidth + b.BorderRightWidth + b.PaddingLeft + b.PaddingRight)
		if b.Width.Auto {
			b.Width = units.F(shrinkToFit(ctx, box, available))
		}
	})
}

// canBreakInside reports whether a line can break inside a box (CSS Text 3
// §3).
func canBreakInside(box boxes.Box) bool {
	if _, atomic := box.(boxes.AtomicInline); atomic || !wrapsIntrinsicText(box) {
		return false
	}
	switch box := box.(type) {
	case *boxes.TextBox:
		return linebreak.CanBreakText(box.Text, string(box.Style.Lang()))
	case boxes.Parent:
		return slices.ContainsFunc(box.ParentBase().Children, canBreakInside)
	}
	return false
}

// adjustLineHeight sets the margins of a line or inline box to its half
// leading, less its borders and padding, so that its margin box is as high
// as its line height.
func adjustLineHeight(box boxes.Box) {
	b := box.Base()
	lineHeight, baseline := b.Style.Strut()
	b.Baseline = &baseline
	height := float64(b.Style.FontSize())
	b.Height = units.F(height)
	halfLeading := (lineHeight - height) / 2
	b.MarginTop = units.F(halfLeading - b.BorderTopWidth - b.PaddingTop)
	b.MarginBottom = units.F(halfLeading - b.BorderBottomWidth - b.PaddingBottom)
}

// inlineSplit holds the state of the split of a line or inline box: the
// children placed so far, and the ones that can still move to the next line
// if a later child does not fit.
type inlineSplit struct {
	ctx         *Context
	box         boxes.Box
	containing  boxes.Box
	o           inlineOptions
	initialSkip ResumeAt
	// children can no longer move to the next line: a break opportunity
	// follows them. waiting children are not followed by one yet.
	children, waiting []inlineChild
	floats            floatWidths
}

// splitInlineBox fits as much of a line or inline box as possible between
// o.PositionX and o.MaxX, with the contract of splitInlineLevel. It loops
// over the children of the box; splitInlineLevel handles the leaves. The
// exclusions of floats are ignored here: the line takes them into account.
func splitInlineBox(ctx *Context, box boxes.Parent, containing boxes.Box, o inlineOptions) InlineResult {
	o = o.withDefaults()
	b := box.Base()
	// The maximum can come from the width of a text that was measured and
	// then added to a position, which may lose the last bits of precision
	// (PEP 485 relative tolerance of 1e-9): an exact fit must not wrap.
	o.MaxX *= 1 + 1e-9

	isStart := o.SkipStack == nil
	initialPositionX := o.PositionX
	leftSpacing := b.PaddingLeft + b.MarginLeft.V + b.BorderLeftWidth
	rightSpacing := b.PaddingRight + b.MarginRight.V + b.BorderRightWidth
	rtl := b.Style.Direction() == "rtl"

	s := &inlineSplit{ctx: ctx, box: box, containing: containing, o: o, initialSkip: o.SkipStack}
	outer := o.Positioned
	relative := b.Style.Position().Keyword == "relative"
	if relative {
		// The absolute boxes of a relative box are positioned against it.
		s.o.Positioned = &positionedBoxes{}
	}

	skip := 0
	var childSkip ResumeAt
	if !isStart {
		skip, childSkip = firstResume(o.SkipStack)
	}

	allChildren := box.ParentBase().Children
	skip = min(skip, len(allChildren))
	var resume ResumeAt
	preserved := false
	var firstLetter, lastLetter boundary
	floatResumeIndex := 0
	broke := false

	for i, child := range allChildren[skip:] {
		index := i + skip
		child.Base().PositionY = b.PositionY

		if !child.IsInNormalFlow() {
			s.outOfFlow(index, child)
			if child.IsFloated() {
				floatResumeIndex = index + 1
				if !slices.Contains(*s.o.WaitingFloats, child) {
					s.o.MaxX -= child.MarginWidth()
				}
			}
			continue
		}

		isLastChild := index == len(allChildren)-1
		availableWidth := s.o.MaxX
		var childWaitingFloats []boxes.Box
		childOptions := func() inlineOptions {
			opts := s.o
			opts.MaxX = availableWidth
			opts.SkipStack = childSkip
			opts.WaitingFloats = &childWaitingFloats
			return opts
		}
		result := splitInlineLevel(ctx, child, containing, childOptions())
		endSpacing := rightSpacing
		if rtl {
			endSpacing = leftSpacing
			s.o.MaxX -= result.FloatWidths.Left
		} else {
			s.o.MaxX -= result.FloatWidths.Right
		}
		if isLastChild && endSpacing != 0 && result.ResumeAt == nil {
			// The spacing after the last child must fit too.
			availableWidth -= endSpacing
			result = splitInlineLevel(ctx, child, containing, childOptions())
		}

		s.floats.Left = max(s.floats.Left, result.FloatWidths.Left)
		s.floats.Right = max(s.floats.Right, result.FloatWidths.Right)

		childSkip = nil
		if result.PreservedLineBreak {
			preserved = true
		}

		canBreak := s.canBreakBefore(child, lastLetter, result.FirstLetter)
		if canBreak {
			s.children = append(s.children, s.waiting...)
			s.waiting = nil
		}

		if firstLetter.Kind == absentBoundary {
			firstLetter = result.FirstLetter
		}
		if child.Base().TrailingCollapsibleSpace {
			lastLetter = collapsibleSpace
		} else {
			lastLetter = result.LastLetter
		}

		if result.Box != nil {
			// Lines keep what they placed, to know the previous break
			// opportunities when floats are added.
			if box.Kind() == boxes.KindLine {
				*s.o.LineChildren = append(*s.o.LineChildren, result.Box)
			}

			newPositionX := result.Box.Base().PositionX + result.Box.MarginWidth()
			if newPositionX > s.o.MaxX {
				// Trailing spaces hang: only what is not a space overflows.
				trailing := trailingWhitespaceSize(ctx, result.Box)
				if newPositionX-trailing > s.o.MaxX {
					if previous := s.breakWaiting(); previous != nil {
						resume = previous
						broke = true
						break
					}
				}
			}
			s.o.PositionX = newPositionX
			s.waiting = append(s.waiting, inlineChild{index, result.Box, child})
		}

		*s.o.WaitingFloats = append(*s.o.WaitingFloats, childWaitingFloats...)
		if result.ResumeAt != nil {
			s.children = append(s.children, s.waiting...)
			resume = ResumeAt{index: result.ResumeAt}
			broke = true
			break
		}
	}
	if !broke {
		s.children = append(s.children, s.waiting...)
		resume = nil
	}

	if rtl && len(s.children) > 1 {
		// Inline-level boxes are ordered from the right in right-to-left
		// text, whatever the order of the content.
		var inFlow []boxes.Box
		for _, c := range s.children {
			if c.box.IsInNormalFlow() {
				inFlow = append(inFlow, c.box)
			}
		}
		if len(inFlow) != 0 {
			x := inFlow[0].Base().PositionX
			for _, c := range slices.Backward(inFlow) {
				c.Translate(x-c.Base().PositionX, 0, true)
				x += c.MarginWidth()
			}
		}
	}

	isEnd := resume == nil
	placed := make([]boxes.Box, len(s.children))
	for i, c := range s.children {
		placed[i] = c.box
	}
	newBox := box.CopyWithChildren(placed)
	nb := newBox.Base()
	nb.RemoveDecoration(!isStart, !isEnd)

	if box.Kind() != boxes.KindLine {
		nb.PositionX = initialPositionX
		leftDecoration := leftSpacing != 0 && (b.Style.BoxDecorationBreak() == "clone" ||
			(isStart && !rtl) || (isEnd && rtl))
		if leftDecoration {
			for _, child := range boxes.ChildrenOf(newBox) {
				child.Translate(leftSpacing, 0, false)
			}
		}
		newBox.Translate(s.floats.Left, 0, true)
	}

	nb.Width = units.F(0)
	newChildren := boxes.ChildrenOf(newBox)
	for i := range newChildren {
		// The extent of the box is that of its last child in the direction
		// of the text.
		child := newChildren[len(newChildren)-1-i]
		if rtl {
			child = newChildren[i]
		}
		if child.IsInNormalFlow() {
			nb.Width = units.F(child.Base().PositionX + child.MarginWidth() - newBox.ContentBoxX())
			break
		}
	}

	adjustLineHeight(newBox)

	if relative {
		outer.Fixed = append(outer.Fixed, s.o.Positioned.Fixed...)
		for _, placeholder := range s.o.Positioned.Absolute {
			absoluteLayout(ctx, placeholder, newBox, &outer.Fixed, o.BottomSpace, nil)
		}
	}

	if resume != nil {
		if index, _ := firstResume(resume); index < floatResumeIndex {
			resume = ResumeAt{floatResumeIndex: nil}
		}
	}

	result := InlineResult{
		Box:                newBox,
		ResumeAt:           resume,
		PreservedLineBreak: preserved,
		FirstLetter:        firstLetter,
		LastLetter:         lastLetter,
		FloatWidths:        s.floats,
	}
	if b.IsLeader {
		result.FirstLetter = collapsibleSpace
		result.LastLetter = boundary{Kind: leaderEndBoundary}
	}
	return result
}

// canBreakBefore reports whether a line can break between what was placed
// before child, which ends with last, and child, which starts with first
// (CSS Text 3 §5.1).
func (s *inlineSplit) canBreakBefore(child boxes.Box, last, first boundary) bool {
	switch last.Kind {
	case leaderStartBoundary, leaderEndBoundary:
		last = boundary{Kind: characterBoundary, Rune: ' '}
	default:
		if ws := s.box.Base().Style.WhiteSpace(); ws == "pre" || ws == "nowrap" {
			return false
		}
	}
	switch {
	case last.Kind == absentBoundary || first.Kind == absentBoundary:
		return false
	case first.Kind == leaderStartBoundary:
		return true
	case first.Kind == leaderEndBoundary:
		return false
	}
	return linebreak.CanBreakText(string(last.Rune)+string(first.Rune), string(child.Base().Style.Lang()))
}

// outOfFlowLayout handles a child that is not in normal flow. Absolutely
// positioned boxes leave a placeholder that records their static position.
// A float is laid out at once if it fits on the line, otherwise it waits
// for the next one. Running elements are kept for the margin boxes.
func (s *inlineSplit) outOfFlow(index int, child boxes.Box) {
	switch {
	case child.IsAbsolutelyPositioned():
		child.Base().PositionX = s.o.PositionX
		placeholder := boxes.NewAbsolutePlaceholder(child)
		*s.o.LinePlaceholders = append(*s.o.LinePlaceholders, placeholder)
		s.waiting = append(s.waiting, inlineChild{index, placeholder, child})
		if child.Base().Style.Position().Keyword == "absolute" {
			s.o.Positioned.Absolute = append(s.o.Positioned.Absolute, placeholder)
		} else {
			s.o.Positioned.Fixed = append(s.o.Positioned.Fixed, placeholder)
		}

	case child.IsFloated():
		child.Base().PositionX = s.o.PositionX
		floatWidth := shrinkToFit(s.ctx, child, s.containing.Base().Width.V)

		// The space the float finds on the line does not include the
		// trailing spaces of what precedes it, which hang.
		var lastInFlow boxes.Box
		for _, c := range slices.Concat(s.children, s.waiting) {
			if !c.box.IsFloated() {
				lastInFlow = c.box
			}
		}
		if lastInFlow != nil {
			floatWidth -= trailingWhitespaceSize(s.ctx, lastInFlow)
		}

		if floatWidth > s.o.MaxX-s.o.PositionX || len(*s.o.WaitingFloats) != 0 {
			*s.o.WaitingFloats = append(*s.o.WaitingFloats, child)
			return
		}
		result := floatLayout(s.ctx, child, s.containing, s.o.Positioned, s.o.BottomSpace, nil)
		if result.Box == nil {
			return
		}
		if result.ResumeAt != nil {
			s.ctx.addBrokenOutOfFlow(child, child, s.containing, result.ResumeAt)
		}
		s.waiting = append(s.waiting, inlineChild{index, result.Box, child})

		float := result.Box
		dx := max(float.MarginWidth(), 0)
		rtl := s.box.Base().Style.Direction() == "rtl"
		side := floatSide(float.Base().Style.Float(), s.box.Base().Style.Direction())
		switch side {
		case "left":
			s.floats.Left += dx
		case "right":
			s.floats.Right += dx
		}
		// The boxes already on the line make room for a float on their
		// own side.
		if side == "left" && !rtl || side == "right" && rtl {
			for _, old := range *s.o.LineChildren {
				if old.IsInNormalFlow() {
					old.Translate(dx, 0, false)
				}
			}
		}

	case child.IsRunning() || child.IsNote():
		s.ctx.addRunningElement(child)
	}
}

// breakWaiting looks for a place to cut the line when a child overflows:
// inside one of the children that were waiting, starting from the last, or
// before the last child that cannot move. It returns where the next line
// resumes, nil when there is no better place than overflowing.
func (s *inlineSplit) breakWaiting() ResumeAt {
	remaining := slices.Clone(s.waiting)
	for len(remaining) != 0 {
		last := remaining[len(remaining)-1]
		remaining = remaining[:len(remaining)-1]
		child := last.box
		if !child.IsInNormalFlow() || !canBreakInside(child) {
			continue
		}

		var childSkip ResumeAt
		if skip, ok := s.initialSkip[last.index]; ok {
			childSkip = skip
		}

		// Cut the child at its last break opportunity by giving it one
		// pixel less each time, until it breaks.
		start := child.Base().PositionX
		var cut boxes.Box
		var cutResume ResumeAt
		for maxX := start + child.MarginWidth() - 1; maxX > start; maxX-- {
			opts := s.o
			opts.PositionX, opts.MaxX, opts.SkipStack = start, maxX, childSkip
			result := splitInlineLevel(s.ctx, last.original, s.containing, opts)
			if result.ResumeAt != nil {
				cut, cutResume = result.Box, result.ResumeAt
				break
			}
		}
		if cutResume == nil {
			continue
		}

		s.children = append(s.children, remaining...)
		if cut != nil {
			s.children = append(s.children, inlineChild{last.index, cut, last.original})
		}
		return ResumeAt{last.index: cutResume}
	}
	if len(s.children) != 0 {
		// The child does not fit and cannot be cut: it moves to the next
		// line as a whole.
		return ResumeAt{s.children[len(s.children)-1].index + 1: nil}
	}
	return nil
}
