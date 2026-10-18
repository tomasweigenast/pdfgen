package draw

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/layout"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// borderSides lists the sides of a box in the order of its four-value
// properties.
var borderSides = [4]string{"top", "right", "bottom", "left"}

// drawBorder paints the borders of box and the rules between its columns.
func drawBorder(s *stream.Stream, box boxes.Box) {
	base := box.Base()
	style := base.Style
	if style.Visibility() != "visible" {
		return
	}

	if hasColumns(box) && style.ColumnRuleWidth() != 0 {
		drawColumnRules(s, box)
	}

	if style.BorderImageSource().Type != "none" && base.BorderImage != nil {
		drawBorderImage(s, box)
		return
	}

	widths := [4]float64{base.BorderTopWidth, base.BorderRightWidth, base.BorderBottomWidth, base.BorderLeftWidth}
	if widths == [4]float64{} {
		return
	}

	colors := [4]syntax.Color{
		getColor(style, props.PropBorderTopColor),
		getColor(style, props.PropBorderRightColor),
		getColor(style, props.PropBorderBottomColor),
		getColor(style, props.PropBorderLeftColor),
	}
	// A fully transparent side has nothing to draw, whatever its style.
	styles := [4]props.Keyword{
		style.BorderTopStyle(), style.BorderRightStyle(),
		style.BorderBottomStyle(), style.BorderLeftStyle(),
	}
	for i, c := range colors {
		if c.Alpha == 0 {
			styles[i] = ""
		}
	}

	sameStyle := styles[1] == styles[0] && styles[2] == styles[0] && styles[3] == styles[0]
	singleColor := sameColor(colors[0], colors[1]) && sameColor(colors[0], colors[2]) && sameColor(colors[0], colors[3])
	// A zero-width side makes the even-odd fill of the simple case leave
	// hairline gaps in PDF viewers.
	fourSides := widths[0] != 0 && widths[1] != 0 && widths[2] != 0 && widths[3] != 0

	if sameStyle && (styles[0] == "solid" || styles[0] == "double") && singleColor && fourSides {
		s.PushState()
		drawRoundedBorder(s, box, styles[0], [2]syntax.Color{colors[0], colors[0]})
		s.PopState()
		return
	}

	// Uniform dotted and dashed borders without rounded corners are one
	// stroked line per side with a dash array, instead of a clip and a path
	// per dash.
	border := box.RoundedBorderBox()
	oneWidth := widths[1] == widths[0] && widths[2] == widths[0] && widths[3] == widths[0]
	if sameStyle && (styles[0] == "dotted" || styles[0] == "dashed") && singleColor && fourSides && oneWidth && !hasCornerRadius(border) {
		s.PushState()
		drawDottedDashedBorder(s, border.Rect, styles[0], widths[0], colors[0])
		s.PopState()
		return
	}

	// The sides are drawn one after the other, clipped to their own
	// segment. The order is not specified; this one is close to what
	// browsers do.
	radii := [4]boxes.Radius{border.TopLeft, border.TopRight, border.BottomRight, border.BottomLeft}
	for _, i := range [...]int{2, 3, 1, 0} {
		if widths[i] == 0 || colors[i].Alpha == 0 {
			continue
		}
		s.PushState()
		clipBorderSegment(s, styles[i], borderSides[i], border.Rect, widths, radii)
		drawRoundedBorder(s, box, styles[i], styledColors(styles[i], colors[i], borderSides[i]))
		s.PopState()
	}
}

// hasColumns reports whether box is a multi-column container.
func hasColumns(box boxes.Box) bool {
	if _, ok := box.(boxes.BlockContainer); !ok {
		return false
	}
	style := box.Base().Style
	return !style.ColumnWidth().IsAuto() || style.ColumnCount().Keyword != "auto"
}

// hasCornerRadius reports whether any corner of b has a non-zero radius.
func hasCornerRadius(b boxes.RoundedBox) bool {
	for _, r := range [...]boxes.Radius{b.TopLeft, b.TopRight, b.BottomRight, b.BottomLeft} {
		if r.X != 0 || r.Y != 0 {
			return true
		}
	}
	return false
}

// sameColor reports whether a and b have the same space, coordinates and
// alpha. Missing coordinates compare equal to each other.
func sameColor(a, b syntax.Color) bool {
	if a.Space != b.Space || a.Alpha != b.Alpha || len(a.Coordinates) != len(b.Coordinates) {
		return false
	}
	for i, v := range a.Coordinates {
		w := b.Coordinates[i]
		if v != w && !(math.IsNaN(v) && math.IsNaN(w)) {
			return false
		}
	}
	return true
}

// styledColors is styledColor with its result as the pair that the border
// painting functions take.
func styledColors(style props.Keyword, c syntax.Color, side string) [2]syntax.Color {
	outer, inner := styledColor(style, c, side)
	return [2]syntax.Color{outer, inner}
}

// drawColumnRules paints the rule between each pair of adjacent columns of
// box. A column that follows a spanning element has no rule before it.
func drawColumnRules(s *stream.Stream, box boxes.Box) {
	parent, ok := box.(boxes.Parent)
	if !ok {
		return
	}
	style := box.Base().Style
	ruleWidth := float64(style.ColumnRuleWidth())
	ruleStyle := style.ColumnRuleStyle()
	widths := [4]float64{0, 0, 0, ruleWidth}
	color := styledColors(ruleStyle, getColor(style, props.PropColumnRuleColor), "left")

	gap := style.FontSize() // normal is 1em.
	if g := style.ColumnGap(); g.Keyword != "normal" {
		gap = props.Pixels(layout.Percentage(g, style, box.Base().Width.V).V)
	}

	skipNext := true
	for _, child := range parent.ParentBase().Children {
		cb := child.Base()
		if cb.Style.ColumnSpan() == "all" {
			skipNext = true
			continue
		}
		if skipNext {
			skipNext = false
			continue
		}
		s.PushState()
		rule := boxes.Rect{X: cb.PositionX - (ruleWidth+float64(gap))/2, Y: cb.PositionY, Width: ruleWidth, Height: cb.Height.V}
		clipBorderSegment(s, ruleStyle, "left", rule, widths, [4]boxes.Radius{})
		drawRectBorder(s, rule, widths, ruleStyle, color)
		s.PopState()
	}
}

// drawRoundedBorder fills the whole border ring of box with colors, which
// are the outer and the inner half of a ridge or groove and the same color
// for any other style.
func drawRoundedBorder(s *stream.Stream, box boxes.Box, style props.Keyword, colors [2]syntax.Color) {
	if style == "ridge" || style == "groove" {
		s.SetColor(colors[0], false)
		roundedBox(s, box.RoundedPaddingBox())
		roundedBox(s, box.RoundedBoxRatio(1./2))
		s.Fill(true)
		s.SetColor(colors[1], false)
		roundedBox(s, box.RoundedBoxRatio(1./2))
		roundedBox(s, box.RoundedBorderBox())
		s.Fill(true)
		return
	}
	s.SetColor(colors[0], false)
	roundedBox(s, box.RoundedPaddingBox())
	if style == "double" {
		roundedBox(s, box.RoundedBoxRatio(1./3))
		roundedBox(s, box.RoundedBoxRatio(2./3))
	}
	roundedBox(s, box.RoundedBorderBox())
	s.Fill(true)
}

// drawRectBorder fills the border ring between r and the rectangle inset by
// widths (top, right, bottom, left), like drawRoundedBorder without radii.
func drawRectBorder(s *stream.Stream, r boxes.Rect, widths [4]float64, style props.Keyword, colors [2]syntax.Color) {
	x, y, w, h := r.X, r.Y, r.Width, r.Height
	bt, br, bb, bl := widths[0], widths[1], widths[2], widths[3]
	if style == "ridge" || style == "groove" {
		s.SetColor(colors[0], false)
		s.Rectangle(x, y, w, h)
		s.Rectangle(x+bl/2, y+bt/2, w-(bl+br)/2, h-(bt+bb)/2)
		s.Fill(true)
		s.Rectangle(x+bl/2, y+bt/2, w-(bl+br)/2, h-(bt+bb)/2)
		s.Rectangle(x+bl, y+bt, w-bl-br, h-bt-bb)
		s.SetColor(colors[1], false)
		s.Fill(true)
		return
	}
	s.SetColor(colors[0], false)
	s.Rectangle(x, y, w, h)
	if style == "double" {
		s.Rectangle(x+bl/3, y+bt/3, w-(bl+br)/3, h-(bt+bb)/3)
		s.Rectangle(x+bl*2/3, y+bt*2/3, w-(bl+br)*2/3, h-(bt+bb)*2/3)
	}
	s.Rectangle(x+bl, y+bt, w-bl-br, h-bt-bb)
	s.Fill(true)
}

// drawDottedDashedBorder strokes a uniform dotted or dashed border of the
// given width around r, one line per side. A dot is a zero-length dash with
// a round cap, so the line ends are inset by half the width to keep the
// corner dots centered; dashes use a butt cap and run flush from corner to
// corner.
func drawDottedDashedBorder(s *stream.Stream, r boxes.Rect, style props.Keyword, width float64, color syntax.Color) {
	x, y, w, h := r.X, r.Y, r.Width, r.Height
	half := width / 2
	inset := 0.
	if style == "dotted" {
		inset = half
	}

	dash := func(outer float64) []float64 {
		if style == "dotted" {
			// A dot on each corner with equal spacing between them.
			count := math.Max(2, math.Floor(outer/(2*width))+1)
			return []float64{0, (outer - width) / (count - 1)}
		}
		// Dashes of about three widths, as many as fit with equal gaps.
		count := math.Floor(outer/(6*width)) + 1
		length := outer / (2*count - 1)
		return []float64{length, length}
	}

	s.SetColor(color, true)
	s.SetLineWidth(width)
	if style == "dotted" {
		s.SetLineCap(1)
	} else {
		s.SetLineCap(0)
	}
	for _, side := range [...]struct{ x1, y1, x2, y2, outer float64 }{
		{x + inset, y + half, x + w - inset, y + half, w},
		{x + w - half, y + inset, x + w - half, y + h - inset, h},
		{x + inset, y + h - half, x + w - inset, y + h - half, w},
		{x + half, y + inset, x + half, y + h - inset, h},
	} {
		s.SetDash(dash(side.outer), 0)
		s.MoveTo(side.x1, side.y1)
		s.LineTo(side.x2, side.y2)
		s.Stroke()
	}
}
