package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawLine strokes the horizontal or vertical segment from (x1, y1) to
// (x2, y2) in the given line style. offset shifts the dash and wave patterns.
//
// colors[0] colors every style; ridge and groove use colors[0] for the outer
// half of the line and colors[1] for the inner one, as returned by
// styledColor. The wavy style only draws horizontal lines.
func drawLine(s *stream.Stream, x1, y1, x2, y2, thickness float64, style props.Keyword, colors [2]syntax.Color, offset float64) {
	vertical := x1 == x2
	if !vertical && y1 != y2 {
		return
	}
	// shifted draws the segment moved by d across its direction.
	shifted := func(d float64) {
		if vertical {
			s.MoveTo(x1+d, y1)
			s.LineTo(x2+d, y2)
		} else {
			s.MoveTo(x1, y1+d)
			s.LineTo(x2, y2+d)
		}
	}

	s.PushState()
	defer s.PopState()
	if style != "ridge" && style != "groove" {
		s.SetColor(colors[0], true)
	}

	switch style {
	case "dashed":
		s.SetDash([]float64{5 * thickness}, offset)
	case "dotted":
		s.SetLineCap(1)
		s.SetDash([]float64{0, 2 * thickness}, offset)
	}

	switch style {
	case "double":
		s.SetLineWidth(thickness / 3)
		shifted(-thickness / 3)
		shifted(thickness / 3)
	case "ridge", "groove":
		s.SetLineWidth(thickness / 2)
		s.SetColor(colors[0], true)
		shifted(thickness / 4)
		s.Stroke()
		s.SetColor(colors[1], true)
		shifted(-thickness / 4)
	case "wavy":
		if vertical {
			return
		}
		drawWave(s, x1, y1, x2, thickness, offset)
	default:
		s.SetLineWidth(thickness)
		s.MoveTo(x1, y1)
		s.LineTo(x2, y2)
	}
	s.Stroke()
}

// drawWave builds the path of a wave of the given thickness along y, clipped
// to its own height, from x1 to x2.
func drawWave(s *stream.Stream, x1, y, x2, thickness, offset float64) {
	radius := 0.75 * thickness
	// A wave without thickness would never advance.
	if !(radius > 0) {
		return
	}
	s.Rectangle(x1, y-2*radius, x2-x1, 4*radius)
	s.Clip(false)
	s.EndPath()

	up := 1.0
	x := x1 - offset
	s.MoveTo(x, y)
	s.SetLineWidth(thickness)
	for x < x2 {
		s.CurveTo(x+radius/2, y+up*radius, x+3*radius/2, y+up*radius, x+2*radius, y)
		x += 2 * radius
		up = -up
	}
}
