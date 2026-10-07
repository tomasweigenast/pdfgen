package boxes

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/units"
)

// PaddingWidth returns the width of the padding box. Like the other
// measures, it counts auto dimensions as zero: layout resolves them before
// measuring.
func (b *BoxFields) PaddingWidth() float64 {
	return b.Width.V + b.PaddingLeft + b.PaddingRight
}

// PaddingHeight returns the height of the padding box.
func (b *BoxFields) PaddingHeight() float64 {
	return b.Height.V + b.PaddingTop + b.PaddingBottom
}

// BorderWidth returns the width of the border box.
func (b *BoxFields) BorderWidth() float64 {
	return b.PaddingWidth() + b.BorderLeftWidth + b.BorderRightWidth
}

// BorderHeight returns the height of the border box.
func (b *BoxFields) BorderHeight() float64 {
	return b.PaddingHeight() + b.BorderTopWidth + b.BorderBottomWidth
}

// MarginWidth returns the width of the margin box, also called outer box.
func (b *BoxFields) MarginWidth() float64 {
	return b.BorderWidth() + b.MarginLeft.V + b.MarginRight.V
}

// MarginHeight returns the height of the margin box.
func (b *BoxFields) MarginHeight() float64 {
	return b.BorderHeight() + b.MarginTop.V + b.MarginBottom.V
}

// ContentBoxX returns the absolute horizontal position of the content box.
func (b *BoxFields) ContentBoxX() float64 {
	return b.PositionX + b.MarginLeft.V + b.PaddingLeft + b.BorderLeftWidth
}

// ContentBoxY returns the absolute vertical position of the content box.
func (b *BoxFields) ContentBoxY() float64 {
	return b.PositionY + b.MarginTop.V + b.PaddingTop + b.BorderTopWidth
}

// PaddingBoxX returns the absolute horizontal position of the padding box.
func (b *BoxFields) PaddingBoxX() float64 {
	return b.PositionX + b.MarginLeft.V + b.BorderLeftWidth
}

// PaddingBoxY returns the absolute vertical position of the padding box.
func (b *BoxFields) PaddingBoxY() float64 {
	return b.PositionY + b.MarginTop.V + b.BorderTopWidth
}

// BorderBoxX returns the absolute horizontal position of the border box.
func (b *BoxFields) BorderBoxX() float64 { return b.PositionX + b.MarginLeft.V }

// BorderBoxY returns the absolute vertical position of the border box.
func (b *BoxFields) BorderBoxY() float64 { return b.PositionY + b.MarginTop.V }

// HitArea returns the rectangle where the box is clickable: its border
// area, which is where hit-testing is done. Inline boxes use their line
// height instead of the height of their border box.
func (b *BoxFields) HitArea() Rect {
	if b.kind == KindInline {
		return Rect{b.BorderBoxX(), b.PositionY, b.BorderWidth(), b.MarginHeight()}
	}
	return Rect{b.BorderBoxX(), b.BorderBoxY(), b.BorderWidth(), b.BorderHeight()}
}

// RoundedBox is a rectangle with rounded corners.
type RoundedBox struct {
	Rect
	TopLeft, TopRight, BottomRight, BottomLeft Radius
}

// clampRadius limits a radius to what a corner can have after the border
// next to it is removed.
func clampRadius(r float64) float64 {
	if !(r > 0) {
		return 0
	}
	return math.Min(r, math.MaxInt64)
}

// RoundedBox returns the position, size and radii of the box inside the
// border box, where bt, br, bb and bl are the distances of its edges from
// the border box. Radii that overlap are scaled down (CSS Backgrounds 3
// §5.5).
func (b *BoxFields) RoundedBox(bt, br, bb, bl float64) RoundedBox {
	tl := Radius{clampRadius(b.BorderTopLeftRadius.X - bl), clampRadius(b.BorderTopLeftRadius.Y - bt)}
	tr := Radius{clampRadius(b.BorderTopRightRadius.X - br), clampRadius(b.BorderTopRightRadius.Y - bt)}
	brr := Radius{clampRadius(b.BorderBottomRightRadius.X - br), clampRadius(b.BorderBottomRightRadius.Y - bb)}
	blr := Radius{clampRadius(b.BorderBottomLeftRadius.X - bl), clampRadius(b.BorderBottomLeftRadius.Y - bb)}

	x := b.BorderBoxX() + bl
	y := b.BorderBoxY() + bt
	width := b.BorderWidth() - bl - br
	height := b.BorderHeight() - bt - bb

	ratio := 1.0
	for _, c := range [...]struct{ extent, sum float64 }{
		{width, tl.X + tr.X},
		{width, blr.X + brr.X},
		{height, tl.Y + blr.Y},
		{height, tr.Y + brr.Y},
	} {
		if c.sum > 0 {
			ratio = math.Min(ratio, c.extent/c.sum)
		}
	}
	scale := func(r Radius) Radius { return Radius{r.X * ratio, r.Y * ratio} }
	return RoundedBox{
		Rect:        Rect{x, y, width, height},
		TopLeft:     scale(tl),
		TopRight:    scale(tr),
		BottomRight: scale(brr),
		BottomLeft:  scale(blr),
	}
}

// RoundedBoxRatio returns the box inside the border box whose edges are at
// the given fraction of the border widths.
func (b *BoxFields) RoundedBoxRatio(ratio float64) RoundedBox {
	return b.RoundedBox(
		b.BorderTopWidth*ratio, b.BorderRightWidth*ratio,
		b.BorderBottomWidth*ratio, b.BorderLeftWidth*ratio)
}

// RoundedPaddingBox returns the position, size and radii of the padding box.
func (b *BoxFields) RoundedPaddingBox() RoundedBox {
	return b.RoundedBox(b.BorderTopWidth, b.BorderRightWidth, b.BorderBottomWidth, b.BorderLeftWidth)
}

// RoundedBorderBox returns the position, size and radii of the border box.
func (b *BoxFields) RoundedBorderBox() RoundedBox { return b.RoundedBox(0, 0, 0, 0) }

// RoundedContentBox returns the position, size and radii of the content box.
func (b *BoxFields) RoundedContentBox() RoundedBox {
	return b.RoundedBox(
		b.BorderTopWidth+b.PaddingTop, b.BorderRightWidth+b.PaddingRight,
		b.BorderBottomWidth+b.PaddingBottom, b.BorderLeftWidth+b.PaddingLeft)
}

// Translate moves the box. Boxes with children move them too.
func (b *BoxFields) Translate(dx, dy float64, ignoreFloats bool) {
	if dx == 0 && dy == 0 {
		return
	}
	b.PositionX += dx
	b.PositionY += dy
}

// resetSpacing sets to zero the margin, padding and border of a side.
func (b *BoxFields) resetSpacing(side DecorationSides) {
	b.RemoveDecorationSides |= side
	switch side {
	case SideTop:
		b.MarginTop = units.F(0)
		b.PaddingTop, b.BorderTopWidth = 0, 0
	case SideRight:
		b.MarginRight = units.F(0)
		b.PaddingRight, b.BorderRightWidth = 0, 0
	case SideBottom:
		b.MarginBottom = units.F(0)
		b.PaddingBottom, b.BorderBottomWidth = 0, 0
	case SideLeft:
		b.MarginLeft = units.F(0)
		b.PaddingLeft, b.BorderLeftWidth = 0, 0
	}
}

// RemoveDecoration removes the decoration at the start and the end of a box
// that is split, unless box-decoration-break is clone (CSS Fragmentation 3
// §5.4). Block-level boxes lose their top and bottom sides; inline-level
// boxes the left and right ones, according to the direction.
func (b *BoxFields) RemoveDecoration(start, end bool) {
	if b.Style.BoxDecorationBreak() == "clone" {
		return
	}
	startSide, endSide := SideTop, SideBottom
	if kinds[b.kind].inlineDecoration {
		startSide, endSide = SideLeft, SideRight
		if b.Style.Direction() != "ltr" {
			startSide, endSide = SideRight, SideLeft
		}
	}
	if start {
		b.resetSpacing(startSide)
	}
	if end {
		b.resetSpacing(endSide)
	}
}
