package layout

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// fallbackReplacedHeight is the height of a replaced element that has
// neither a height nor a ratio, nor anything to derive them from (CSS 2.1
// §10.6.2).
const fallbackReplacedHeight = 150

// intrinsicSize returns the natural size and aspect ratio of the content of
// a replaced box at the resolution and font size of its style. Any of them
// can be auto. The image already has the orientation of image-orientation
// applied, so the size is that of the image as displayed.
func intrinsicSize(r *boxes.ReplacedFields) (width, height, ratio units.MaybeFloat) {
	return r.Replacement.IntrinsicSize(float64(r.Style.ImageResolution()), float64(r.Style.FontSize()))
}

// defaultImageSizing returns the concrete size of an object given the
// width and height specified for it (auto when missing), its intrinsic
// size and ratio, and the size to use when it has nothing to go by (CSS
// Images 3 §5.3).
func defaultImageSizing(
	intrinsicWidth, intrinsicHeight, intrinsicRatio units.MaybeFloat,
	specifiedWidth, specifiedHeight units.MaybeFloat,
	defaultWidth, defaultHeight float64,
) (width, height float64) {
	switch {
	case !specifiedWidth.Auto && !specifiedHeight.Auto:
		return specifiedWidth.V, specifiedHeight.V
	case !specifiedWidth.Auto:
		switch {
		case !intrinsicRatio.Auto:
			return specifiedWidth.V, specifiedWidth.V / intrinsicRatio.V
		case !intrinsicHeight.Auto:
			return specifiedWidth.V, intrinsicHeight.V
		}
		return specifiedWidth.V, defaultHeight
	case !specifiedHeight.Auto:
		switch {
		case !intrinsicRatio.Auto:
			return specifiedHeight.V * intrinsicRatio.V, specifiedHeight.V
		case !intrinsicWidth.Auto:
			return intrinsicWidth.V, specifiedHeight.V
		}
		return defaultWidth, specifiedHeight.V
	}
	if !intrinsicWidth.Auto || !intrinsicHeight.Auto {
		return defaultImageSizing(
			intrinsicWidth,
			intrinsicHeight,
			intrinsicRatio,
			intrinsicWidth,
			intrinsicHeight,
			defaultWidth,
			defaultHeight,
		)
	}
	return containConstraintImageSizing(defaultWidth, defaultHeight, intrinsicRatio)
}

// containConstraintImageSizing returns the largest size with the ratio that
// fits in the constraint (CSS Images 3 §5.4).
func containConstraintImageSizing(constraintWidth, constraintHeight float64, ratio units.MaybeFloat) (width, height float64) {
	return constraintImageSizing(constraintWidth, constraintHeight, ratio, false)
}

// coverConstraintImageSizing returns the smallest size with the ratio that
// covers the constraint (CSS Images 3 §5.4).
func coverConstraintImageSizing(constraintWidth, constraintHeight float64, ratio units.MaybeFloat) (width, height float64) {
	return constraintImageSizing(constraintWidth, constraintHeight, ratio, true)
}

func constraintImageSizing(constraintWidth, constraintHeight float64, ratio units.MaybeFloat, cover bool) (width, height float64) {
	if ratio.Auto {
		return constraintWidth, constraintHeight
	}
	if cover != (constraintWidth > constraintHeight*ratio.V) {
		return constraintHeight * ratio.V, constraintHeight
	}
	return constraintWidth, constraintWidth / ratio.V
}

// ReplacedBoxLayout returns the rectangle that the content of a replaced
// box is drawn in, in the coordinates of the page: its size follows
// object-fit and its position object-position, inside the content box
// (CSS Images 3 §5.5). The width and height of the box must be resolved.
func ReplacedBoxLayout(box boxes.Replaced) boxes.Rect {
	r := box.ReplacedBase()
	style := r.Style
	boxWidth, boxHeight := r.Width.V, r.Height.V

	intrinsicWidth, intrinsicHeight, ratio := intrinsicSize(r)
	var naturalWidth, naturalHeight float64
	if intrinsicWidth.Auto || intrinsicHeight.Auto {
		naturalWidth, naturalHeight = containConstraintImageSizing(boxWidth, boxHeight, ratio)
	} else {
		naturalWidth, naturalHeight = intrinsicWidth.V, intrinsicHeight.V
	}

	objectFit := style.ObjectFit()
	var drawWidth, drawHeight float64
	switch objectFit {
	case "fill":
		drawWidth, drawHeight = boxWidth, boxHeight
	case "contain", "scale-down":
		drawWidth, drawHeight = containConstraintImageSizing(boxWidth, boxHeight, ratio)
	case "cover":
		drawWidth, drawHeight = coverConstraintImageSizing(boxWidth, boxHeight, ratio)
	default: // none
		drawWidth, drawHeight = naturalWidth, naturalHeight
	}
	if objectFit == "scale-down" {
		drawWidth = firstMin(drawWidth, naturalWidth)
		drawHeight = firstMin(drawHeight, naturalHeight)
	}

	position := objectPosition(style.ObjectPosition())
	freeWidth := boxWidth - drawWidth
	freeHeight := boxHeight - drawHeight
	x := percentage(position.X, style, units.F(freeWidth)).Or(0)
	y := percentage(position.Y, style, units.F(freeHeight)).Or(0)
	if position.OriginX == "right" {
		x = freeWidth - x
	}
	if position.OriginY == "bottom" {
		y = freeHeight - y
	}

	return boxes.Rect{
		X:      x + r.ContentBoxX(),
		Y:      y + r.ContentBoxY(),
		Width:  drawWidth,
		Height: drawHeight,
	}
}

// objectPosition returns the first position of the list, or the initial
// value, the center, when the list is empty.
func objectPosition(positions props.Positions) props.Position {
	if len(positions) > 0 {
		return positions[0]
	}
	half := props.Length{Dimension: props.Dimension{Value: 50, Unit: "%"}}
	return props.Position{OriginX: "left", X: half, OriginY: "top", Y: half}
}

func replacedBoxWidth(box *boxes.ReplacedFields, cb containingBlock, direction props.Keyword) {
	withMinMaxWidth(&box.BoxFields, func() { replacedBoxWidthWithoutMinMax(box, cb, direction) })
}

func replacedBoxWidthWithoutMinMax(box *boxes.ReplacedFields, cb containingBlock, direction props.Keyword) {
	width, height, ratio := intrinsicSize(box)
	if box.Height.Auto && box.Width.Auto {
		switch {
		case !width.Auto:
			box.Width = width
		case !ratio.Auto && !height.Auto:
			box.Width = units.F(height.V * ratio.V)
		case !ratio.Auto:
			blockLevelWidth(&box.BoxFields, cb, direction)
		}
	}
	if !box.Width.Auto {
		return
	}
	switch {
	case !ratio.Auto:
		box.Width = units.F(box.Height.V * ratio.V)
	case !width.Auto:
		box.Width = width
	default:
		box.Width = units.F(300)
	}
}

func blockReplacedWidth(box *boxes.ReplacedFields, cb containingBlock, direction props.Keyword) {
	withMinMaxWidth(&box.BoxFields, func() { blockReplacedWidthWithoutMinMax(box, cb, direction) })
}

func blockReplacedWidthWithoutMinMax(box *boxes.ReplacedFields, cb containingBlock, direction props.Keyword) {
	replacedBoxWidthWithoutMinMax(box, cb, direction)
	blockLevelWidthWithoutMinMax(&box.BoxFields, cb, direction)
}

func inlineReplacedBoxLayout(box *boxes.ReplacedFields, cb containingBlock, direction props.Keyword) {
	for _, margin := range []*units.MaybeFloat{&box.MarginTop, &box.MarginRight, &box.MarginBottom, &box.MarginLeft} {
		if margin.Auto {
			*margin = units.F(0)
		}
	}
	inlineReplacedBoxWidthHeight(box, cb, direction)
}

func inlineReplacedBoxWidthHeight(box *boxes.ReplacedFields, cb containingBlock, direction props.Keyword) {
	if box.Style.Width().IsAuto() && box.Style.Height().IsAuto() {
		replacedBoxWidthWithoutMinMax(box, cb, direction)
		replacedBoxHeightWithoutMinMax(box)
		minMaxAutoReplaced(box)
		return
	}
	replacedBoxWidth(box, cb, direction)
	replacedBoxHeight(box)
}

// replacedBoxHeight sets the used height of a replaced box, within its
// min-height and max-height (CSS 2.1 §10.6.2). Its width must be resolved.
func replacedBoxHeight(box *boxes.ReplacedFields) {
	withMinMaxHeight(&box.BoxFields, func() { replacedBoxHeightWithoutMinMax(box) })
}

func replacedBoxHeightWithoutMinMax(box *boxes.ReplacedFields) {
	b := &box.BoxFields
	_, height, ratio := intrinsicSize(box)
	hasRatio := !ratio.Auto && ratio.V != 0

	switch {
	case b.Height.Auto && b.Width.Auto:
		b.Height = height
	case b.Height.Auto && hasRatio:
		b.Height = units.F(b.Width.V / ratio.V)
	}

	switch {
	case b.Height.Auto && b.Width.Auto && !height.Auto:
		b.Height = height
	case !ratio.Auto && b.Height.Auto:
		b.Height = units.F(b.Width.V / ratio.V)
	case b.Height.Auto && !height.Auto:
		b.Height = height
	case b.Height.Auto:
		b.Height = units.F(fallbackReplacedHeight)
	}
}

// minMaxAutoReplaced applies min-width, max-width, min-height and
// max-height to a replaced box whose width and height are both auto, which
// keeps its ratio unless the limits make that impossible (CSS 2.1 §10.4,
// table of constraint violations). Its width and height must hold the
// sizes before the limits.
func minMaxAutoReplaced(box *boxes.ReplacedFields) {
	b := &box.BoxFields
	width, height := b.Width.V, b.Height.V
	minWidth, minHeight := b.MinWidth.V, b.MinHeight.V
	maxWidth := firstMax(minWidth, b.MaxWidth.V)
	maxHeight := firstMax(minHeight, b.MaxHeight.V)

	violationWidth := violationOf(width, minWidth, maxWidth)
	violationHeight := violationOf(height, minHeight, maxHeight)

	// Pathological sizes would divide by zero.
	if width == 0 {
		width = 1e-6
	}
	if height == 0 {
		height = 1e-6
	}

	set := func(w, h float64) { b.Width, b.Height = units.F(w), units.F(h) }
	switch [2]violation{violationWidth, violationHeight} {
	case [2]violation{violatesMax, noViolation}:
		set(maxWidth, firstMax(maxWidth*height/width, minHeight))
	case [2]violation{violatesMin, noViolation}:
		set(minWidth, firstMin(minWidth*height/width, maxHeight))
	case [2]violation{noViolation, violatesMax}:
		set(firstMax(maxHeight*width/height, minWidth), maxHeight)
	case [2]violation{noViolation, violatesMin}:
		set(firstMin(minHeight*width/height, maxWidth), minHeight)
	case [2]violation{violatesMax, violatesMax}:
		if maxWidth/width <= maxHeight/height {
			set(maxWidth, firstMax(minHeight, maxWidth*height/width))
		} else {
			set(firstMax(minWidth, maxHeight*width/height), maxHeight)
		}
	case [2]violation{violatesMin, violatesMin}:
		if minWidth/width <= minHeight/height {
			set(firstMin(maxWidth, minHeight*width/height), minHeight)
		} else {
			set(minWidth, firstMin(maxHeight, minWidth*height/width))
		}
	case [2]violation{violatesMin, violatesMax}:
		set(minWidth, maxHeight)
	case [2]violation{violatesMax, violatesMin}:
		set(maxWidth, minHeight)
	}
}

// violation tells which limit a size is outside of.
type violation uint8

const (
	noViolation violation = iota
	violatesMin
	violatesMax
)

func violationOf(size, minSize, maxSize float64) violation {
	switch {
	case size < minSize:
		return violatesMin
	case size > maxSize:
		return violatesMax
	}
	return noViolation
}
