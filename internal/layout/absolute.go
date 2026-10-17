package layout

import (
	"fmt"
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// absoluteWidth sets the width and the horizontal margins of an absolutely
// positioned box, within min-width and max-width (CSS 2.1 §10.3.7). The
// containing block is the rectangle given. It returns the offset that moves the
// box to its place and whether the width of the box must be subtracted from it,
// for boxes that are placed by their right edge.
func absoluteWidth(ctx *Context, box boxes.Box, cbX, cbWidth float64) (translateBoxWidth bool, translateX float64) {
	b := box.Base()
	withMinMaxWidth(b, func() {
		translateBoxWidth, translateX = absoluteWidthWithoutMinMax(ctx, box, cbX, cbWidth)
	})
	return translateBoxWidth, translateX
}

func absoluteWidthWithoutMinMax(ctx *Context, box boxes.Box, cbX, cbWidth float64) (translateBoxWidth bool, translateX float64) {
	b := box.Base()
	parent := b.Style.Parent()
	ltr := parent == nil || parent.Direction() == "ltr"
	paddingsBorders := b.PaddingLeft + b.PaddingRight + b.BorderLeftWidth + b.BorderRightWidth
	defaultTranslateX := cbX - b.PositionX
	zeroAutoMargins := func(margins ...*units.MaybeFloat) {
		for _, margin := range margins {
			if margin.Auto {
				*margin = units.F(0)
			}
		}
	}

	switch {
	case b.Left.Auto && b.Right.Auto && b.Width.Auto:
		zeroAutoMargins(&b.MarginLeft, &b.MarginRight)
		available := cbWidth - (paddingsBorders + b.MarginLeft.V + b.MarginRight.V)
		b.Width = units.F(shrinkToFit(ctx, box, available))
		if b.IsOutsideMarker {
			translateBoxWidth = ltr
		} else if !ltr {
			translateBoxWidth = true
			translateX = defaultTranslateX + available
		}

	case !b.Left.Auto && !b.Right.Auto && !b.Width.Auto:
		widthForMargins := cbWidth - (b.Right.V + b.Left.V + b.Width.V + paddingsBorders)
		switch {
		case b.MarginLeft.Auto && b.MarginRight.Auto:
			if b.Width.V+paddingsBorders+b.Right.V+b.Left.V <= cbWidth {
				b.MarginLeft = units.F(widthForMargins / 2)
				b.MarginRight = units.F(widthForMargins / 2)
			} else if ltr {
				b.MarginLeft, b.MarginRight = units.F(0), units.F(widthForMargins)
			} else {
				b.MarginLeft, b.MarginRight = units.F(widthForMargins), units.F(0)
			}
		case b.MarginLeft.Auto:
			b.MarginLeft = units.F(widthForMargins)
		case b.MarginRight.Auto:
			b.MarginRight = units.F(widthForMargins)
		case ltr:
			b.MarginRight = units.F(widthForMargins)
		default:
			b.MarginLeft = units.F(widthForMargins)
		}
		translateX = b.Left.V + defaultTranslateX

	default:
		zeroAutoMargins(&b.MarginLeft, &b.MarginRight)
		spacing := paddingsBorders + b.MarginLeft.V + b.MarginRight.V
		switch {
		case b.Left.Auto && b.Width.Auto:
			b.Width = units.F(shrinkToFit(ctx, box, cbWidth-spacing-b.Right.V))
			translateX = cbWidth - b.Right.V - spacing + defaultTranslateX
			translateBoxWidth = true
		case b.Left.Auto && b.Right.Auto:
			if !ltr {
				available := cbWidth - (paddingsBorders + b.MarginLeft.V + b.MarginRight.V)
				translateBoxWidth = true
				translateX = defaultTranslateX + available
			}
		case b.Width.Auto && b.Right.Auto:
			b.Width = units.F(shrinkToFit(ctx, box, cbWidth-spacing-b.Left.V))
			translateX = b.Left.V + defaultTranslateX
		case b.Left.Auto:
			translateX = cbWidth + defaultTranslateX - (b.Right.V + spacing + b.Width.V)
		case b.Width.Auto:
			b.Width = units.F(cbWidth - b.Right.V - b.Left.V - spacing)
			translateX = b.Left.V + defaultTranslateX
		case b.Right.Auto:
			translateX = b.Left.V + defaultTranslateX
		}
	}
	return translateBoxWidth, translateX
}

// absoluteHeight sets the height and the vertical margins of an absolutely
// positioned box (CSS 2.1 §10.6.4). It returns the offset that moves the box
// to its place and whether its height must be subtracted from it.
func absoluteHeight(box boxes.Box, cbY, cbHeight float64) (translateBoxHeight bool, translateY float64) {
	b := box.Base()
	paddingsBorders := b.PaddingTop + b.PaddingBottom + b.BorderTopWidth + b.BorderBottomWidth
	defaultTranslateY := cbY - b.PositionY
	zeroAutoMargins := func() {
		if b.MarginTop.Auto {
			b.MarginTop = units.F(0)
		}
		if b.MarginBottom.Auto {
			b.MarginBottom = units.F(0)
		}
	}

	switch {
	case b.Top.Auto && b.Bottom.Auto && b.Height.Auto:
		// The box keeps its static position.
		zeroAutoMargins()

	case !b.Top.Auto && !b.Bottom.Auto && !b.Height.Auto:
		heightForMargins := cbHeight - (b.Top.V + b.Bottom.V + b.Height.V + paddingsBorders)
		switch {
		case b.MarginTop.Auto && b.MarginBottom.Auto:
			b.MarginTop = units.F(heightForMargins / 2)
			b.MarginBottom = units.F(heightForMargins / 2)
		case b.MarginTop.Auto:
			b.MarginTop = units.F(heightForMargins)
		default:
			b.MarginBottom = units.F(heightForMargins)
		}
		translateY = b.Top.V + defaultTranslateY

	default:
		zeroAutoMargins()
		spacing := paddingsBorders + b.MarginTop.V + b.MarginBottom.V
		switch {
		case b.Top.Auto && b.Height.Auto:
			translateY = cbHeight - b.Bottom.V - spacing + defaultTranslateY
			translateBoxHeight = true
		case b.Top.Auto && b.Bottom.Auto:
			// The box keeps its static position.
		case b.Height.Auto && b.Bottom.Auto:
			translateY = b.Top.V + defaultTranslateY
		case b.Top.Auto:
			translateY = cbHeight + defaultTranslateY - (b.Bottom.V + spacing + b.Height.V)
		case b.Height.Auto:
			b.Height = units.F(cbHeight - b.Bottom.V - b.Top.V - spacing)
			translateY = b.Top.V + defaultTranslateY
		case b.Bottom.Auto:
			translateY = b.Top.V + defaultTranslateY
		}
	}
	return translateBoxHeight, translateY
}

// absoluteBlock lays out an absolutely positioned box whose content is laid
// out as a block, and moves it to its place in the containing block.
func absoluteBlock(ctx *Context, box, containing boxes.Box, fixed *[]*boxes.AbsolutePlaceholder, bottomSpace float64, skip ResumeAt, cbX, cbY, cbWidth, cbHeight float64) (boxes.Box, ResumeAt) {
	b := box.Base()
	translateBoxWidth, translateX := absoluteWidth(ctx, box, cbX, cbWidth)
	var translateBoxHeight bool
	var translateY float64
	if len(skip) == 0 {
		translateBoxHeight, translateY = absoluteHeight(box, cbY, cbHeight)
	}

	if translateBoxHeight {
		bottomSpace += -b.PositionY
	} else {
		bottomSpace += translateY
	}

	// The box is the containing block of its absolute descendants.
	positioned := &positionedBoxes{Fixed: *fixed}
	defer func() { *fixed = positioned.Fixed }()

	if b.IsTableWrapper {
		ctx.unsupported("tables")
		return nil, nil
	}

	var newBox boxes.Box
	var resume ResumeAt
	switch {
	case isBlockBox(box):
		result := blockContainerLayout(ctx, box, containing, blockOptions{
			BottomSpace: bottomSpace,
			SkipStack:   skip,
			PageIsEmpty: true,
			Positioned:  positioned,
		})
		newBox, resume = result.Box, result.ResumeAt
	case box.Kind() == boxes.KindFlex || box.Kind() == boxes.KindInlineFlex:
		ctx.unsupported("flex containers")
		return nil, nil
	case box.Kind() == boxes.KindGrid || box.Kind() == boxes.KindInlineGrid:
		ctx.unsupported("grid containers")
		return nil, nil
	default:
		ctx.fail(fmt.Errorf("layout of absolutely positioned %s boxes is not handled", box.Kind()))
		return nil, nil
	}
	if newBox == nil {
		// The layout failed and recorded why.
		return nil, nil
	}

	for _, placeholder := range positioned.Absolute {
		absoluteLayout(ctx, placeholder, newBox, &positioned.Fixed, bottomSpace, nil)
	}

	if translateBoxWidth {
		translateX -= newBox.Base().Width.V
	}
	if translateBoxHeight {
		translateY -= newBox.Base().Height.V
	}
	newBox.Translate(translateX, translateY, false)
	return newBox, resume
}

// absoluteLayout lays out the box that a placeholder stands for against its
// containing block, and puts the result in its place. A box that did not fit
// on the page is recorded together with the point where it resumes.
func absoluteLayout(ctx *Context, placeholder *boxes.AbsolutePlaceholder, containing boxes.Box, fixed *[]*boxes.AbsolutePlaceholder, bottomSpace float64, skip ResumeAt) {
	if placeholder.LayoutDone {
		ctx.fail(fmt.Errorf("an absolutely positioned box was laid out twice"))
		return
	}
	box := placeholder.Box
	newBox, resume := absoluteBoxLayout(ctx, box, containing, fixed, bottomSpace, skip)
	if newBox == nil {
		return
	}
	placeholder.SetLaidOutBox(newBox)
	if len(resume) != 0 {
		ctx.addBrokenOutOfFlow(placeholder, box, containing, resume)
	}
}

// absoluteBoxLayout lays out an absolutely positioned box against its
// containing block. The containing block of a box is the content box of the
// page for the initial one, and the padding box of the nearest positioned
// ancestor otherwise (CSS 2.1 §10.1). It returns nil when the layout failed.
func absoluteBoxLayout(ctx *Context, box, containing boxes.Box, fixed *[]*boxes.AbsolutePlaceholder, bottomSpace float64, skip ResumeAt) (boxes.Box, ResumeAt) {
	var cbX, cbY, cbWidth, cbHeight float64
	if containing.Kind() == boxes.KindPage {
		cbX, cbY = containing.ContentBoxX(), containing.ContentBoxY()
		cbWidth, cbHeight = containing.Base().Width.V, containing.Base().Height.V
	} else {
		cbX, cbY = containing.PaddingBoxX(), containing.PaddingBoxY()
		cbWidth, cbHeight = containing.PaddingWidth(), containing.PaddingHeight()
	}

	cb := containingBlock{Width: units.F(cbWidth), Height: units.F(cbHeight)}
	resolvePercentages(box, cb)
	resolvePositionPercentages(box, cb)

	if replaced, ok := box.(*boxes.BlockReplacedBox); ok {
		return absoluteReplaced(replaced, cbX, cbY, cbWidth, cbHeight), nil
	}
	return absoluteBlock(ctx, box, containing, fixed, bottomSpace, skip, cbX, cbY, cbWidth, cbHeight)
}

// absoluteReplaced sets the size and the position of an absolutely
// positioned replaced box (CSS 2.1 §10.3.8, §10.6.5). The box has no
// children to move, and is not copied.
func absoluteReplaced(box *boxes.BlockReplacedBox, cbX, cbY, cbWidth, cbHeight float64) boxes.Box {
	b := &box.BoxFields
	// A box whose width and height follow its ratio takes the width of its
	// containing block from the first value of this rectangle, as the block
	// width solver reads the origin of the rectangle in its place.
	inlineReplacedBoxWidthHeight(&box.ReplacedFields, containingBlock{Width: units.F(cbX), Height: units.F(cbY)}, "ltr")
	parent := b.Style.Parent()
	ltr := parent == nil || parent.Direction() == "ltr"

	// CSS 2.1 §10.3.8.
	if b.Left.Auto && b.Right.Auto {
		// The static position.
		if ltr {
			b.Left = units.F(b.PositionX - cbX)
		} else {
			b.Right = units.F(cbX + cbWidth - b.PositionX)
		}
	}
	switch {
	case b.Left.Auto || b.Right.Auto:
		if b.MarginLeft.Auto {
			b.MarginLeft = units.F(0)
		}
		if b.MarginRight.Auto {
			b.MarginRight = units.F(0)
		}
		remaining := cbWidth - box.MarginWidth()
		if b.Left.Auto {
			b.Left = units.F(remaining - b.Right.V)
		}
		if b.Right.Auto {
			b.Right = units.F(remaining - b.Left.V)
		}
	case b.MarginLeft.Auto || b.MarginRight.Auto:
		remaining := cbWidth - (box.BorderWidth() + b.Left.V + b.Right.V)
		switch {
		case b.MarginLeft.Auto && b.MarginRight.Auto:
			if remaining >= 0 {
				half := math.Floor(remaining / 2)
				b.MarginLeft, b.MarginRight = units.F(half), units.F(half)
			} else if ltr {
				b.MarginLeft, b.MarginRight = units.F(0), units.F(remaining)
			} else {
				b.MarginLeft, b.MarginRight = units.F(remaining), units.F(0)
			}
		case b.MarginLeft.Auto:
			b.MarginLeft = units.F(remaining)
		default:
			b.MarginRight = units.F(remaining)
		}
	default:
		// Over-constrained.
		if ltr {
			b.Right = units.F(cbWidth - (box.MarginWidth() + b.Left.V))
		} else {
			b.Left = units.F(cbWidth - (box.MarginWidth() + b.Right.V))
		}
	}

	// CSS 2.1 §10.6.5.
	if b.Top.Auto && b.Bottom.Auto {
		b.Top = units.F(b.PositionY - cbY)
	}
	switch {
	case b.Top.Auto || b.Bottom.Auto:
		if b.MarginTop.Auto {
			b.MarginTop = units.F(0)
		}
		if b.MarginBottom.Auto {
			b.MarginBottom = units.F(0)
		}
		remaining := cbHeight - box.MarginHeight()
		if b.Top.Auto {
			b.Top = units.F(remaining - b.Bottom.V)
		}
		if b.Bottom.Auto {
			b.Bottom = units.F(remaining - b.Top.V)
		}
	case b.MarginTop.Auto || b.MarginBottom.Auto:
		remaining := cbHeight - (box.BorderHeight() + b.Top.V + b.Bottom.V)
		switch {
		case b.MarginTop.Auto && b.MarginBottom.Auto:
			half := math.Floor(remaining / 2)
			b.MarginTop, b.MarginBottom = units.F(half), units.F(half)
		case b.MarginTop.Auto:
			b.MarginTop = units.F(remaining)
		default:
			b.MarginBottom = units.F(remaining)
		}
	default:
		// Over-constrained.
		b.Bottom = units.F(cbHeight - (box.MarginHeight() + b.Top.V))
	}

	b.PositionX = cbX + b.Left.V
	b.PositionY = cbY + b.Top.V
	return box
}
