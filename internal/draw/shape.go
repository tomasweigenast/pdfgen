package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// cornerControl is the fraction of a corner radius at which the Bézier
// control points of a rounded corner sit. It is not the circle constant
// (about 0.552): pixel output depends on this value.
const cornerControl = 0.45

// roundedBox appends the outline of b to the current path of s: a rectangle
// when no corner has a radius, else lines joined by one curve per corner.
func roundedBox(s *stream.Stream, b boxes.RoundedBox) {
	if !hasRadius(b) {
		s.Rectangle(b.X, b.Y, b.Width, b.Height)
		return
	}

	x, y, w, h := b.X, b.Y, b.Width, b.Height
	tl, tr, br, bl := b.TopLeft, b.TopRight, b.BottomRight, b.BottomLeft
	const r = cornerControl

	s.MoveTo(x+tl.X, y)
	s.LineTo(x+w-tr.X, y)
	s.CurveTo(x+w-tr.X*r, y, x+w, y+tr.Y*r, x+w, y+tr.Y)
	s.LineTo(x+w, y+h-br.Y)
	s.CurveTo(x+w, y+h-br.Y*r, x+w-br.X*r, y+h, x+w-br.X, y+h)
	s.LineTo(x+bl.X, y+h)
	s.CurveTo(x+bl.X*r, y+h, x, y+h-bl.Y*r, x, y+h-bl.Y)
	s.LineTo(x, y+tl.Y)
	s.CurveTo(x, y+tl.Y*r, x+tl.X*r, y, x+tl.X, y)
}

// hasRadius reports whether any corner of b is rounded. A corner whose
// horizontal or vertical radius is zero is square.
func hasRadius(b boxes.RoundedBox) bool {
	for _, c := range [...]boxes.Radius{b.TopLeft, b.TopRight, b.BottomRight, b.BottomLeft} {
		if c.X != 0 && c.Y != 0 {
			return true
		}
	}
	return false
}

// clipRoundedBox restricts the painting of s to the inside of b.
func clipRoundedBox(s *stream.Stream, b boxes.RoundedBox) {
	roundedBox(s, b)
	s.Clip(false)
	s.EndPath()
}
