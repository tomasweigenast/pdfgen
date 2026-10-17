package layout

import (
	"fmt"
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// floatWidth sets the width of a float whose width is auto to the width that
// fits its content, within min-width and max-width (CSS 2.1 §10.3.5).
func floatWidth(ctx *Context, box, containing boxes.Box) {
	b := box.Base()
	withMinMaxWidth(b, func() {
		// The limits can replace the width by a value, so auto is checked
		// again every time.
		if b.Width.Auto {
			b.Width = units.F(shrinkToFit(ctx, box, containing.Base().Width.V))
		}
	})
}

// floatLayout sets the size of a floated box, lays out its content and puts
// it at the highest place where it fits next to the floats before it
// (CSS 2.1 §9.5.1). It returns the fragment and where the box resumes if it
// did not fit on the page.
func floatLayout(ctx *Context, box, containing boxes.Box, positioned *positionedBoxes, bottomSpace float64, skip ResumeAt) fragmentResult {
	b := box.Base()
	cb := containingBlockOf(containing)
	direction := containing.Base().Style.Direction()
	resolvePercentages(box, cb)

	// The height of the containing block is only known once it is laid
	// out; until then the content laid out so far stands for it.
	if cb.Height.Auto {
		cb.Height = units.F(containing.Base().PositionY - containing.ContentBoxY())
	}
	resolvePositionPercentages(box, cb)

	for _, margin := range []*units.MaybeFloat{&b.MarginLeft, &b.MarginRight, &b.MarginTop, &b.MarginBottom} {
		if margin.Auto {
			*margin = units.F(0)
		}
	}

	if clearance := getClearance(ctx, box, direction, 0); clearance != nil {
		b.PositionY += *clearance
	}

	if replaced, ok := box.(*boxes.BlockReplacedBox); ok {
		inlineReplacedBoxWidthHeight(&replaced.ReplacedFields, containingBlockOf(containing), direction)
	} else if b.Width.Auto {
		floatWidth(ctx, box, containing)
	}

	if b.IsTableWrapper {
		ctx.unsupported("tables")
		return fragmentResult{}
	}

	var resume ResumeAt
	switch box.(type) {
	case boxes.BlockContainer:
		result := blockContainerLayout(ctx, box, containing, blockOptions{
			BottomSpace: bottomSpace,
			SkipStack:   skip,
			PageIsEmpty: true,
			Positioned:  positioned,
		})
		if result.Box == nil {
			ctx.fail(fmt.Errorf("the layout of a float produced no box"))
			return fragmentResult{}
		}
		box, resume = result.Box, result.ResumeAt
	case boxes.FlexContainer:
		ctx.unsupported("flex containers")
		return fragmentResult{}
	case boxes.GridContainer:
		ctx.unsupported("grid containers")
		return fragmentResult{}
	}

	findFloatPosition(ctx, box, containing)
	ctx.addExcludedShape(box)
	return fragmentResult{Box: box, ResumeAt: resume}
}

// findFloatPosition moves a float to its place: below the floats before it
// unless it fits beside them, and against the side it floats to (CSS 2.1
// §9.5.1, rules 4 to 9). The vertical position of the box already accounts
// for the margins collapsed by its containing box (rule 4).
func findFloatPosition(ctx *Context, box, containing boxes.Box) {
	b := box.Base()
	// Rules 5 and 6: a float is never higher than the floats before it.
	if ctx.ExcludedShapes != nil && len(*ctx.ExcludedShapes) != 0 {
		shapes := *ctx.ExcludedShapes
		if highest := shapes[len(shapes)-1].Base().PositionY; b.PositionY < highest {
			box.Translate(0, highest-b.PositionY, false)
		}
	}

	// Rules 1 and 2.
	x, y, available := avoidCollisions(ctx, box, containing, true)

	// Rule 9: the vertical position is set, and the horizontal one is
	// already the left edge for floats to the left.
	direction := b.Style.Direction()
	side := b.Style.Float()
	if side == "right" || (direction == "ltr" && side == "inline-end") || (direction == "rtl" && side == "inline-start") {
		x += available - box.MarginWidth()
	}
	box.Translate(x-b.PositionX, y-b.PositionY, false)
}

func floatSide(value, direction props.Keyword) props.Keyword {
	switch value {
	case "inline-start":
		if direction == "rtl" {
			return "right"
		}
		return "left"
	case "inline-end":
		if direction == "rtl" {
			return "left"
		}
		return "right"
	}
	return value
}

func getClearance(ctx *Context, box boxes.Box, direction props.Keyword, collapsedMargin float64) *float64 {
	clear := floatSide(box.Base().Style.Clear(), direction)
	matches := func(shape boxes.Box) bool {
		return clear == "both" || clear == floatSide(shape.Base().Style.Float(), direction)
	}
	for _, broken := range ctx.brokenOutOfFlow {
		if broken.Fragment.IsFloated() && matches(broken.Fragment) {
			clearance := float64(math.MaxInt64)
			return &clearance
		}
	}
	if ctx.ExcludedShapes == nil {
		return nil
	}
	var clearance *float64
	hypothetical := box.Base().PositionY + collapsedMargin
	for _, shape := range *ctx.ExcludedShapes {
		if !matches(shape) {
			continue
		}
		bottom := shape.Base().PositionY + shape.MarginHeight()
		if hypothetical < bottom && (clearance == nil || *clearance < bottom-hypothetical) {
			value := bottom - hypothetical
			clearance = &value
		}
	}
	return clearance
}

func avoidCollisions(ctx *Context, box, containing boxes.Box, outer bool) (x, y, width float64) {
	b := box.Base()
	y = b.PositionY
	boxWidth, boxHeight := box.MarginWidth(), box.MarginHeight()
	if !outer {
		y = box.BorderBoxY()
		boxWidth, boxHeight = box.BorderWidth(), box.BorderHeight()
	}
	if box.BorderHeight() == 0 && box.IsFloated() {
		return 0, y, containing.Base().Width.V
	}
	direction := containing.Base().Style.Direction()
	var left, right float64
	for {
		left = containing.ContentBoxX()
		right = left + containing.Base().Width.V
		if !outer {
			left += b.MarginLeft.V
			right -= b.MarginRight.V
		}
		nextY := math.Inf(1)
		hasBounds := false
		if ctx.ExcludedShapes != nil {
			for _, shape := range *ctx.ExcludedShapes {
				top := shape.Base().PositionY
				bottom := top + shape.MarginHeight()
				startsInside := top < y && y < bottom
				endsInside := top < y+boxHeight && y+boxHeight < bottom
				containsShape := top >= y && bottom <= y+boxHeight
				if !startsInside && !endsInside && !containsShape {
					continue
				}
				nextY = min(nextY, bottom)
				switch floatSide(shape.Base().Style.Float(), direction) {
				case "left":
					left = max(left, shape.Base().PositionX+shape.MarginWidth())
					hasBounds = true
				case "right":
					right = min(right, shape.Base().PositionX)
					hasBounds = true
				}
			}
		}
		if hasBounds && boxWidth > right-left && nextY > y {
			y = nextY
			continue
		}
		break
	}
	x = left
	if b.Style.Float() == "none" && direction == "rtl" {
		x = right - boxWidth
		if box.Kind() == boxes.KindLine {
			x = right
		}
	}
	width = right - left
	if !outer {
		x -= b.MarginLeft.V
		y -= b.MarginTop.V
	}
	return x, y, width
}
