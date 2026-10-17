package layout

import (
	"errors"
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// containingBlock is the size that the percentages of a box refer to: the
// content box of its containing block. A height that is auto depends on the
// content of the box being laid out, so percentages of it cannot be
// resolved (CSS 2.1 §10.5).
type containingBlock struct{ Width, Height units.MaybeFloat }

// containingBlockOf returns the size of the content box of b.
func containingBlockOf(b boxes.Box) containingBlock {
	f := b.Base()
	return containingBlock{Width: f.Width, Height: f.Height}
}

// firstMax and firstMin pick the larger and the smaller of two numbers with
// plain comparisons: the first argument wins unless the second is strictly
// better, so a NaN never replaces a number. That matters for percentages of
// an unbounded maximum (inf * 0), where math.Max would propagate the NaN.
func firstMax(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func firstMin(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}

// Percentage returns the length l in pixels, where percentages are fractions
// of referTo. It returns auto for the keyword auto and for any other
// keyword a length property can hold.
func Percentage(l props.Length, style *computed.Style, referTo float64) units.MaybeFloat {
	return percentage(l, style, units.F(referTo))
}

// percentage is Percentage with a reference that can be auto, in which case
// a percentage is auto too.
func percentage(l props.Length, style *computed.Style, referTo units.MaybeFloat) units.MaybeFloat {
	switch {
	case l.IsKeyword():
		return units.Auto
	case l.Calc != nil:
		return resolveCalc(l.Calc, style, referTo)
	case l.Dimension.Unit == "%":
		if referTo.Auto {
			return units.Auto
		}
		return units.F(referTo.V * l.Dimension.Value / 100)
	}
	return units.F(l.Dimension.Value)
}

// resolveCalc resolves a math function that mixes percentages with other
// units, now that the length that 100% stands for is known.
func resolveCalc(c *props.Calc, style *computed.Style, referTo units.MaybeFloat) units.MaybeFloat {
	var ref *float64
	if !referTo.Auto {
		ref = &referTo.V
	}
	resolved, err := tk.ResolveMath(c.Function, style, "", ref)
	if errors.Is(err, tk.ErrPercentageInMath) {
		return units.Auto
	}
	if err != nil || resolved == nil {
		logging.Warnf("Invalid math function: %s", c.Function.Serialize())
		return units.F(0)
	}
	switch t := resolved.(type) {
	case *syntax.PercentageToken:
		if referTo.Auto {
			return units.Auto
		}
		return units.F(referTo.V * t.Value / 100)
	case *syntax.DimensionToken:
		return units.F(style.ToPixels(t.Value, t.LowerUnit, ""))
	case *syntax.NumberToken:
		return units.F(t.Value)
	}
	logging.Warnf("Invalid math function: %s", c.Function.Serialize())
	return units.F(0)
}

// lengthOf returns the computed value of a length property.
func lengthOf(style *computed.Style, id props.PropID) props.Length {
	switch v := style.Get(id).(type) {
	case props.Length:
		return v
	case props.Pixels:
		return props.Px(float64(v))
	}
	logging.Warnf("Property %s is not a length", id.Name())
	return props.LengthKeyword("auto")
}

// resolveOnePercentage returns the used value of a length property of a
// style: percentages are fractions of referTo, and an auto minimum is 0.
func resolveOnePercentage(style *computed.Style, id props.PropID, referTo units.MaybeFloat) units.MaybeFloat {
	used := percentage(lengthOf(style, id), style, referTo)
	if used.Auto && (id == props.PropMinWidth || id == props.PropMinHeight) {
		return units.F(0)
	}
	return used
}

// resolvePositionPercentages sets the used offsets of a positioned box.
// Left and right refer to the width of the containing block, top and bottom
// to its height.
func resolvePositionPercentages(box boxes.Box, cb containingBlock) {
	b := box.Base()
	b.Left = resolveOnePercentage(b.Style, props.PropLeft, cb.Width)
	b.Right = resolveOnePercentage(b.Style, props.PropRight, cb.Width)
	b.Top = resolveOnePercentage(b.Style, props.PropTop, cb.Height)
	b.Bottom = resolveOnePercentage(b.Style, props.PropBottom, cb.Height)
}

// resolvePercentages sets the used margins, paddings, border widths, sizes
// and size limits of a box from its computed style. Percentages refer to
// the width of the containing block, including those of vertical margins
// and paddings (CSS 2.1 §8.3, §8.4); page boxes use its height for them
// instead. The sizes are then reduced to content sizes according to
// box-sizing.
func resolvePercentages(box boxes.Box, cb containingBlock) {
	b := box.Base()
	style := b.Style
	verticalRef := cb.Width
	if box.Kind() == boxes.KindPage {
		verticalRef = cb.Height
	}
	b.MarginLeft = resolveOnePercentage(style, props.PropMarginLeft, cb.Width)
	b.MarginRight = resolveOnePercentage(style, props.PropMarginRight, cb.Width)
	b.MarginTop = resolveOnePercentage(style, props.PropMarginTop, verticalRef)
	b.MarginBottom = resolveOnePercentage(style, props.PropMarginBottom, verticalRef)
	b.PaddingLeft = resolveOnePercentage(style, props.PropPaddingLeft, cb.Width).Or(0)
	b.PaddingRight = resolveOnePercentage(style, props.PropPaddingRight, cb.Width).Or(0)
	b.PaddingTop = resolveOnePercentage(style, props.PropPaddingTop, verticalRef).Or(0)
	b.PaddingBottom = resolveOnePercentage(style, props.PropPaddingBottom, verticalRef).Or(0)
	b.Width = resolveOnePercentage(style, props.PropWidth, cb.Width)
	b.MinWidth = resolveOnePercentage(style, props.PropMinWidth, cb.Width)
	b.MaxWidth = resolveOnePercentage(style, props.PropMaxWidth, cb.Width)

	if cb.Height.Auto {
		// The height of the containing block depends on its content, so
		// percentages of it are ignored (CSS 2.1 §10.5, §10.7).
		height := style.Height()
		if height.IsKeyword() || height.Calc != nil || height.Dimension.Unit == "%" {
			b.Height = units.Auto
		} else {
			b.Height = units.F(height.Dimension.Value)
		}
		b.MinHeight = resolveOnePercentage(style, props.PropMinHeight, units.F(0))
		b.MaxHeight = resolveOnePercentage(style, props.PropMaxHeight, units.F(math.Inf(1)))
	} else {
		b.Height = resolveOnePercentage(style, props.PropHeight, cb.Height)
		b.MinHeight = resolveOnePercentage(style, props.PropMinHeight, cb.Height)
		b.MaxHeight = resolveOnePercentage(style, props.PropMaxHeight, cb.Height)
	}

	if !(style.BorderCollapse() == "collapse" && b.CollapsedBordersResolved) {
		b.BorderTopWidth = float64(style.BorderTopWidth())
		b.BorderRightWidth = float64(style.BorderRightWidth())
		b.BorderBottomWidth = float64(style.BorderBottomWidth())
		b.BorderLeftWidth = float64(style.BorderLeftWidth())
	}

	adjustBoxSizing(b, false)
	adjustBoxSizing(b, true)
}

// resolveRadiiPercentages sets the used radii of the corners of a box.
// Percentages refer to the width and height of the border box. A corner
// whose radius is zero in one direction is square, and so is one next to a
// side whose decoration was removed (CSS Backgrounds 3 §5.1).
func resolveRadiiPercentages(box boxes.Box) {
	b := box.Base()
	style := b.Style
	corners := [...]struct {
		sides boxes.DecorationSides
		value props.LengthPair
		dst   *boxes.Radius
	}{
		{boxes.SideTop | boxes.SideLeft, style.BorderTopLeftRadius(), &b.BorderTopLeftRadius},
		{boxes.SideTop | boxes.SideRight, style.BorderTopRightRadius(), &b.BorderTopRightRadius},
		{boxes.SideBottom | boxes.SideRight, style.BorderBottomRightRadius(), &b.BorderBottomRightRadius},
		{boxes.SideBottom | boxes.SideLeft, style.BorderBottomLeftRadius(), &b.BorderBottomLeftRadius},
	}
	for _, c := range corners {
		rx, ry := c.value.X, c.value.Y
		if isZeroPixels(rx) || isZeroPixels(ry) || b.RemoveDecorationSides&c.sides != 0 {
			*c.dst = boxes.Radius{}
			continue
		}
		*c.dst = boxes.Radius{
			X: percentage(rx, style, units.F(b.BorderWidth())).Or(0),
			Y: percentage(ry, style, units.F(b.BorderHeight())).Or(0),
		}
	}
}

func isZeroPixels(l props.Length) bool {
	return !l.IsKeyword() && l.Calc == nil && l.Dimension.Value == 0 &&
		(l.Dimension.Unit == "px" || l.Dimension.Unit == "")
}

// adjustBoxSizing turns the width or the height, and their limits, from the
// box the box-sizing property refers to into the size of the content box
// (CSS Box Sizing 3 §3). Sizes never become negative.
func adjustBoxSizing(b *boxes.BoxFields, vertical bool) {
	size, minSize, maxSize := &b.Width, &b.MinWidth, &b.MaxWidth
	padding := b.PaddingLeft + b.PaddingRight
	borders := [2]float64{b.BorderLeftWidth, b.BorderRightWidth}
	if vertical {
		size, minSize, maxSize = &b.Height, &b.MinHeight, &b.MaxHeight
		padding = b.PaddingTop + b.PaddingBottom
		borders = [2]float64{b.BorderTopWidth, b.BorderBottomWidth}
	}

	var delta float64
	switch b.Style.BoxSizing() {
	case "border-box":
		delta = padding + borders[0] + borders[1]
	case "padding-box":
		delta = padding
	}
	if !(delta > 0) {
		return
	}
	if !size.Auto {
		*size = units.F(firstMax(0, size.V-delta))
	}
	if !maxSize.Auto {
		*maxSize = units.F(firstMax(0, maxSize.V-delta))
	}
	if !minSize.Auto {
		*minSize = units.F(firstMax(0, minSize.V-delta))
	}
}
