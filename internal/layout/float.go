package layout

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

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
