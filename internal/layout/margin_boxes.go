package layout

import (
	"errors"
	"fmt"
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// orientedBox gives the box model of one axis of a box, horizontal or
// vertical, the same vocabulary: the inner size, the two margins and the
// padding and border together. It lets the rules of CSS Paged Media 3 §5
// and CSS 2.1 §10.3.3 be written once for widths and for heights.
type orientedBox struct {
	ctx               *Context
	box               boxes.Box
	vertical          bool
	inner             units.MaybeFloat
	marginA, marginB  units.MaybeFloat
	paddingPlusBorder float64

	minContent, maxContent         float64
	haveMinContent, haveMaxContent bool
}

func newOrientedBox(ctx *Context, box boxes.Box, vertical bool) *orientedBox {
	b := box.Base()
	if vertical {
		return &orientedBox{
			ctx: ctx, box: box, vertical: true,
			inner:             b.Height,
			marginA:           b.MarginTop,
			marginB:           b.MarginBottom,
			paddingPlusBorder: b.PaddingTop + b.PaddingBottom + b.BorderTopWidth + b.BorderBottomWidth,
		}
	}
	return &orientedBox{
		ctx: ctx, box: box,
		inner:             b.Width,
		marginA:           b.MarginLeft,
		marginB:           b.MarginRight,
		paddingPlusBorder: b.PaddingLeft + b.PaddingRight + b.BorderLeftWidth + b.BorderRightWidth,
	}
}

// restore copies the values of the axis back to the box.
func (o *orientedBox) restore() {
	b := o.box.Base()
	if o.vertical {
		b.Height, b.MarginTop, b.MarginBottom = o.inner, o.marginA, o.marginB
		return
	}
	b.Width, b.MarginLeft, b.MarginRight = o.inner, o.marginA, o.marginB
}

// sugar is the size of everything around the inner size.
func (o *orientedBox) sugar() float64 {
	return o.paddingPlusBorder + o.marginA.V + o.marginB.V
}

func (o *orientedBox) outer() float64 { return o.sugar() + o.inner.V }

// setOuter gives the box an outer size, bounded by its content sizes.
func (o *orientedBox) setOuter(outer float64) {
	o.inner = units.F(min(max(o.minContentSize(), outer-o.sugar()), o.maxContentSize()))
}

// The content sizes of a height are not defined for margin boxes: any height
// is valid between 0 and a limit far beyond the page.
func (o *orientedBox) minContentSize() float64 {
	if o.vertical {
		return 0
	}
	if !o.haveMinContent {
		o.minContent = minContentWidth(o.ctx, o.box, false)
		o.haveMinContent = true
	}
	return o.minContent
}

func (o *orientedBox) maxContentSize() float64 {
	if o.vertical {
		return 1e6
	}
	if !o.haveMaxContent {
		o.maxContent = maxContentWidth(o.ctx, o.box, false)
		o.haveMaxContent = true
	}
	return o.maxContent
}

func (o *orientedBox) outerMinContentSize() float64 {
	if o.inner.Auto {
		return o.sugar() + o.minContentSize()
	}
	return o.sugar() + o.inner.V
}

func (o *orientedBox) outerMaxContentSize() float64 {
	if o.inner.Auto {
		return o.sugar() + o.maxContentSize()
	}
	return o.sugar() + o.inner.V
}

// computeFixedDimension sets the dimension of a margin box that is fixed by
// the page margin, its height for a top or bottom margin box and its width
// for a left or right one, along with its margins in that axis (CSS Paged
// Media 3 §5.3, "margin constraints"). outer is the page margin, and
// topOrLeft tells which margin gives way when the values are
// over-constrained, the one facing the page content being the other.
func computeFixedDimension(ctx *Context, box boxes.Box, outer float64, vertical, topOrLeft bool) {
	o := newOrientedBox(ctx, box, vertical)

	// Rule 2.
	total := o.paddingPlusBorder
	for _, value := range []units.MaybeFloat{o.marginA, o.marginB, o.inner} {
		if !value.Auto {
			total += value.V
		}
	}
	if total > outer {
		if o.marginA.Auto {
			o.marginA = units.F(0)
		}
		if o.marginB.Auto {
			o.marginB = units.F(0)
		}
		if o.inner.Auto {
			// Without this the inner size would turn negative; with it,
			// rule 3 applies and gives the margin the missing space
			// (www-style, 2012-07-02).
			o.inner = units.F(0)
		}
	}
	// Rule 3.
	if !o.marginA.Auto && !o.marginB.Auto && !o.inner.Auto {
		if topOrLeft {
			o.marginA = units.Auto
		} else {
			o.marginB = units.Auto
		}
	}
	// Rule 4.
	if autoCount(o.marginA, o.marginB, o.inner) == 1 {
		switch {
		case o.inner.Auto:
			o.inner = units.F(outer - o.paddingPlusBorder - o.marginA.V - o.marginB.V)
		case o.marginA.Auto:
			o.marginA = units.F(outer - o.paddingPlusBorder - o.marginB.V - o.inner.V)
		case o.marginB.Auto:
			o.marginB = units.F(outer - o.paddingPlusBorder - o.marginA.V - o.inner.V)
		}
	}
	// Rule 5.
	if o.inner.Auto {
		if o.marginA.Auto {
			o.marginA = units.F(0)
		}
		if o.marginB.Auto {
			o.marginB = units.F(0)
		}
		o.inner = units.F(outer - o.paddingPlusBorder - o.marginA.V - o.marginB.V)
	}
	// Rule 6.
	if o.marginA.Auto && o.marginB.Auto {
		half := (outer - o.paddingPlusBorder - o.inner.V) / 2
		o.marginA, o.marginB = units.F(half), units.F(half)
	}
	o.restore()
}

func autoCount(values ...units.MaybeFloat) int {
	n := 0
	for _, v := range values {
		if v.Auto {
			n++
		}
	}
	return n
}

// flexFactorSum avoids dividing by zero when none of the boxes can grow.
func flexFactorSum(a, b float64) float64 {
	if sum := a + b; sum != 0 {
		return sum
	}
	return 1
}

// computeVariableDimension sets the width of the three margin boxes along a
// side of the page, or their height for the left and right sides, and their
// margins in that axis (CSS Paged Media 3 §5.3, "margin box dimensions").
// availableSize is the length of the side. The middle box, when it is not
// generated, takes no space and the outer ones share the side.
func computeVariableDimension(ctx *Context, sideBoxes [3]boxes.Box, vertical bool, availableSize float64) {
	var oriented [3]*orientedBox
	for i, box := range sideBoxes {
		oriented[i] = newOrientedBox(ctx, box, vertical)
	}
	a, b, c := oriented[0], oriented[1], oriented[2]

	for _, o := range oriented {
		if o.marginA.Auto {
			o.marginA = units.F(0)
		}
		if o.marginB.Auto {
			o.marginB = units.F(0)
		}
	}

	if !sideBoxes[1].(*boxes.MarginBox).IsGenerated {
		switch {
		case a.inner.Auto && c.inner.Auto:
			distributeBetweenOuterBoxes(a, c, availableSize)
		case a.inner.Auto:
			a.setOuter(availableSize - c.outer())
		case c.inner.Auto:
			c.setOuter(availableSize - a.outer())
		}
	} else {
		if b.inner.Auto {
			resolveMiddleBox(a, b, c, availableSize)
		}
		if a.inner.Auto {
			a.setOuter((availableSize - b.outer()) / 2)
		}
		if c.inner.Auto {
			c.setOuter((availableSize - b.outer()) / 2)
		}
	}

	for _, o := range oriented {
		o.restore()
	}
}

// distributeBetweenOuterBoxes shares a side between its two outer boxes
// when the middle one is not generated and both have an auto size: in
// proportion to their max-content sizes while those fit, then to the room
// above their min-content sizes, and below that to the min-content sizes.
func distributeBetweenOuterBoxes(a, c *orientedBox, available float64) {
	switch {
	case available > a.outerMaxContentSize()+c.outerMaxContentSize():
		flexSpace := available - a.outerMaxContentSize() - c.outerMaxContentSize()
		factorA, factorC := a.outerMaxContentSize(), c.outerMaxContentSize()
		sum := flexFactorSum(factorA, factorC)
		a.setOuter(a.maxContentSize() + flexSpace*factorA/sum)
		c.setOuter(c.maxContentSize() + flexSpace*factorC/sum)
	case available > a.outerMinContentSize()+c.outerMinContentSize():
		flexSpace := available - a.outerMinContentSize() - c.outerMinContentSize()
		factorA := a.maxContentSize() - a.minContentSize()
		factorC := c.maxContentSize() - c.minContentSize()
		sum := flexFactorSum(factorA, factorC)
		a.setOuter(a.minContentSize() + flexSpace*factorA/sum)
		c.setOuter(c.minContentSize() + flexSpace*factorC/sum)
	default:
		flexSpace := available - a.outerMinContentSize() - c.outerMinContentSize()
		factorA, factorC := a.minContentSize(), c.minContentSize()
		sum := flexFactorSum(factorA, factorC)
		a.setOuter(a.minContentSize() + flexSpace*factorA/sum)
		c.setOuter(c.minContentSize() + flexSpace*factorC/sum)
	}
}

// resolveMiddleBox gives an auto size to the middle box of a side. The outer
// boxes are kept as wide as twice the larger of them, so that the middle one
// stays centred.
func resolveMiddleBox(a, b, c *orientedBox, available float64) {
	acMax := 2 * max(a.outerMaxContentSize(), c.outerMaxContentSize())
	if available > b.outerMaxContentSize()+acMax {
		flexSpace := available - b.outerMaxContentSize() - acMax
		factorB, factorAC := b.outerMaxContentSize(), acMax
		sum := flexFactorSum(factorB, factorAC)
		b.setOuter(b.maxContentSize() + flexSpace*factorB/sum)
		return
	}
	acMin := 2 * max(a.outerMinContentSize(), c.outerMinContentSize())
	if available > b.outerMinContentSize()+acMin {
		flexSpace := available - b.outerMinContentSize() - acMin
		factorB := b.maxContentSize() - b.minContentSize()
		factorAC := acMax - acMin
		sum := flexFactorSum(factorB, factorAC)
		b.setOuter(b.minContentSize() + flexSpace*factorB/sum)
		return
	}
	flexSpace := available - b.outerMinContentSize() - acMin
	factorB, factorAC := b.minContentSize(), acMin
	sum := flexFactorSum(factorB, factorAC)
	b.setOuter(b.minContentSize() + flexSpace*factorB/sum)
}

// standardizePageBasedCounters drops the pages counter from the counter
// properties of a page or margin box style, since it is set by layout, and
// makes pages increment the page counter unless the style handles it
// (CSS Paged Media 3 §7.1). atKeyword is empty for the page itself.
func standardizePageBasedCounters(style *computed.Style, atKeyword string) {
	touched := false
	standardize := func(id props.PropID, counters props.Counters) {
		if counters.Auto {
			style.Set(id, props.Counters{})
			return
		}
		kept := make([]props.CounterOp, 0, len(counters.Items))
		for _, op := range counters.Items {
			if op.Name == "page" {
				touched = true
			}
			if op.Name != "pages" {
				kept = append(kept, op)
			}
		}
		style.Set(id, props.Counters{Items: kept})
	}
	standardize(props.PropCounterSet, style.CounterSet())
	standardize(props.PropCounterReset, style.CounterReset())
	standardize(props.PropCounterIncrement, style.CounterIncrement())

	if atKeyword == "" && !touched {
		items := append([]props.CounterOp{{Name: "page", Value: 1}}, style.CounterIncrement().Items...)
		style.Set(props.PropCounterIncrement, props.Counters{Items: items})
	}
}

// resetSpacing sets the margin, padding and border of every side of a box
// to zero.
func resetSpacing(box boxes.Box) {
	b := box.Base()
	b.RemoveDecorationSides |= boxes.SideTop | boxes.SideRight | boxes.SideBottom | boxes.SideLeft
	b.MarginTop, b.MarginRight, b.MarginBottom, b.MarginLeft = units.F(0), units.F(0), units.F(0), units.F(0)
	b.PaddingTop, b.PaddingRight, b.PaddingBottom, b.PaddingLeft = 0, 0, 0, 0
	b.BorderTopWidth, b.BorderRightWidth, b.BorderBottomWidth, b.BorderLeftWidth = 0, 0, 0, 0
}

// isGeneratedContent reports whether the content property makes a box:
// a list of items, as opposed to normal, none or inhibit.
func isGeneratedContent(content props.Content) bool { return content.Keyword == "" }

// makeMarginBox returns the box of a margin at-rule with its content built
// and its percentages resolved against the given containing block, or a
// box without size when it has no content: it takes no space but its
// neighbours still need it.
func makeMarginBox(ctx *Context, page *boxes.PageBox, state PageState, atKeyword string, cb containingBlock) *boxes.MarginBox {
	style := ctx.StyleFor.Page(page.PageType, atKeyword)
	if style == nil {
		// A margin box without declarations does not affect counters.
		style = page.Style.Anonymous()
	}
	standardizePageBasedCounters(style, atKeyword)
	box := boxes.NewMarginBox(atKeyword, style)
	box.IsGenerated = isGeneratedContent(style.Content())

	var laidOut boxes.Box = box
	if box.IsGenerated {
		// Margin boxes must not change the counters of the pages.
		marginState := state.Build.Clone()
		marginState.CounterScopes = append(marginState.CounterScopes, map[string]bool{})
		boxes.UpdateCounters(marginState, box.Style)
		box.Children = ctx.Builder.ContentToBoxes(box.Style, box, &marginState.QuoteDepth,
			marginState.CounterValues, page, ctx)
		boxes.ProcessWhitespace(box, false)
		boxes.ProcessTextTransform(box)
		laidOut = boxes.CreateAnonymousBoxes(box)
	}
	resolvePercentages(laidOut, cb)
	if !box.IsGenerated {
		b := laidOut.Base()
		b.Width, b.Height = units.F(0), units.F(0)
		resetSpacing(laidOut)
	}
	marginBox, ok := laidOut.(*boxes.MarginBox)
	if !ok {
		ctx.fail(fmt.Errorf("a margin box became a %s box", laidOut.Kind()))
		return box
	}
	return marginBox
}

// makeMarginBoxes returns the laid-out margin boxes of a page. state is the
// state the page maker holds after the page: the counters at its end.
func makeMarginBoxes(ctx *Context, page *boxes.PageBox, state PageState) []boxes.Box {
	marginTop, marginBottom := page.MarginTop.V, page.MarginBottom.V
	marginLeft, marginRight := page.MarginLeft.V, page.MarginRight.V
	maxBoxWidth, maxBoxHeight := page.BorderWidth(), page.BorderHeight()

	// The bottom right corner of the border box.
	pageEndX := marginLeft + maxBoxWidth
	pageEndY := marginTop + maxBoxHeight

	// CSS Paged Media 3 §5.3.
	var generated []*boxes.MarginBox

	sides := []struct {
		prefix                   string
		vertical                 bool
		cb                       containingBlock
		positionX, positionY     float64
		fixedOuter, variableSize float64
	}{
		{"top", false, containingBlock{Width: units.F(maxBoxWidth), Height: units.F(marginTop)}, marginLeft, 0, marginTop, maxBoxWidth},
		{"bottom", false, containingBlock{Width: units.F(maxBoxWidth), Height: units.F(marginBottom)}, marginLeft, pageEndY, marginBottom, maxBoxWidth},
		{"left", true, containingBlock{Width: units.F(marginLeft), Height: units.F(maxBoxHeight)}, 0, marginTop, marginLeft, maxBoxHeight},
		{"right", true, containingBlock{Width: units.F(marginRight), Height: units.F(maxBoxHeight)}, pageEndX, marginTop, marginRight, maxBoxHeight},
	}
	for _, side := range sides {
		suffixes := [3]string{"left", "center", "right"}
		if side.vertical {
			suffixes = [3]string{"top", "middle", "bottom"}
		}
		var sideBoxes [3]*boxes.MarginBox
		anyGenerated := false
		for i, suffix := range suffixes {
			sideBoxes[i] = makeMarginBox(ctx, page, state, "@"+side.prefix+"-"+suffix, side.cb)
			anyGenerated = anyGenerated || sideBoxes[i].IsGenerated
		}
		if !anyGenerated {
			continue
		}
		// The three boxes are needed together for the variable dimension.
		computeVariableDimension(ctx, [3]boxes.Box{sideBoxes[0], sideBoxes[1], sideBoxes[2]}, side.vertical, side.variableSize)
		for i, offset := range [3]float64{0, 0.5, 1} {
			box := sideBoxes[i]
			if !box.IsGenerated {
				continue
			}
			box.PositionX, box.PositionY = side.positionX, side.positionY
			if side.vertical {
				box.PositionY += offset * (side.variableSize - box.MarginHeight())
			} else {
				box.PositionX += offset * (side.variableSize - box.MarginWidth())
			}
			computeFixedDimension(ctx, box, side.fixedOuter, !side.vertical, side.prefix == "top" || side.prefix == "left")
			generated = append(generated, box)
		}
	}

	corners := []struct {
		atKeyword            string
		cbWidth, cbHeight    float64
		positionX, positionY float64
		top, left            bool
	}{
		{"@top-left-corner", marginLeft, marginTop, 0, 0, true, true},
		{"@top-right-corner", marginRight, marginTop, pageEndX, 0, true, false},
		{"@bottom-left-corner", marginLeft, marginBottom, 0, pageEndY, false, true},
		{"@bottom-right-corner", marginRight, marginBottom, pageEndX, pageEndY, false, false},
	}
	for _, corner := range corners {
		box := makeMarginBox(ctx, page, state, corner.atKeyword,
			containingBlock{Width: units.F(corner.cbWidth), Height: units.F(corner.cbHeight)})
		if !box.IsGenerated {
			continue
		}
		box.PositionX, box.PositionY = corner.positionX, corner.positionY
		computeFixedDimension(ctx, box, corner.cbHeight, true, corner.top)
		computeFixedDimension(ctx, box, corner.cbWidth, false, corner.left)
		generated = append(generated, box)
	}

	laidOut := make([]boxes.Box, 0, len(generated))
	for _, box := range generated {
		if laid := marginBoxContentLayout(ctx, box); laid != nil {
			laidOut = append(laidOut, laid)
		}
	}
	return laidOut
}

// marginBoxContentLayout lays out the content of a margin box, once the
// box has its dimensions, and aligns it vertically (CSS Paged Media 3 §5.3).
func marginBoxContentLayout(ctx *Context, box *boxes.MarginBox) boxes.Box {
	positioned := &positionedBoxes{}
	result := blockContainerLayout(ctx, box, box, blockOptions{
		BottomSpace: math.Inf(-1),
		PageIsEmpty: true,
		Positioned:  positioned,
	})
	if result.Box == nil {
		ctx.fail(errors.New("a margin box did not fit"))
		return nil
	}
	laid := result.Box
	for _, placeholder := range positioned.Absolute {
		absoluteLayout(ctx, placeholder, laid, &positioned.Fixed, 0, nil)
	}
	for i := 0; i < len(positioned.Fixed); i++ {
		absoluteLayout(ctx, positioned.Fixed[i], laid, &positioned.Fixed, 0, nil)
	}

	// Every other value is read as top, which leaves the content in place.
	alignment := laid.Base().Style.VerticalAlign().Keyword
	children := boxes.ChildrenOf(laid)
	if (alignment == "middle" || alignment == "bottom") && len(children) != 0 {
		first, last := children[0], children[len(children)-1]
		top := first.Base().PositionY
		bottom := last.Base().PositionY + last.MarginHeight()
		offset := laid.Base().Height.V - (bottom - top)
		if alignment == "middle" {
			offset /= 2
		}
		for _, child := range children {
			child.Translate(0, offset, false)
		}
	}
	return laid
}
