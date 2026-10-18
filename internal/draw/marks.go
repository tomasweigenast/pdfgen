package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// circleKappa is the control point distance that approximates a quarter
// circle with a cubic Bézier curve, as a fraction of the radius.
const circleKappa = 0.5522847498

// transformStep is one translate or scale of a mark's transform list. The
// list applies in SVG order: the last step is applied to the point first.
type transformStep struct {
	scale bool
	x, y  float64
}

func translate(x, y float64) transformStep { return transformStep{false, x, y} }
func scale(x, y float64) transformStep     { return transformStep{true, x, y} }

// drawPageMarks strokes the crop marks and registration crosses of a page
// in its bleed area with 1px black lines. area is the painting area of the
// page background: the whole page including bleed.
func drawPageMarks(s *stream.Stream, area boxes.Rect, bleed boxes.Sides, marks props.Keywords) {
	width, height := area.Width, area.Height
	half := boxes.Sides{Top: bleed.Top / 2, Right: bleed.Right / 2, Bottom: bleed.Bottom / 2, Left: bleed.Left / 2}

	s.PushState()
	defer s.PopState()
	s.Transform(1, 0, 0, 1, area.X, area.Y)
	s.SetColor(syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 1), true)
	s.SetLineWidth(1)

	line := func(x, y, dx, dy float64, steps ...transformStep) {
		stroked(s, steps, func() {
			s.MoveTo(x, y)
			s.LineTo(x+dx, y+dy)
		})
	}
	circle := func(r float64, steps ...transformStep) {
		stroked(s, steps, func() { circlePath(s, r) })
	}

	for _, mark := range marks {
		switch mark {
		case "crop":
			line(0, bleed.Top, half.Left, 0)
			line(0, bleed.Top, half.Right, 0, translate(width, 0), scale(-1, 1))
			line(0, bleed.Bottom, half.Right, 0, translate(width, height), scale(-1, -1))
			line(0, bleed.Bottom, half.Left, 0, translate(0, height), scale(1, -1))
			line(bleed.Left, 0, 0, half.Top)
			line(bleed.Right, 0, 0, half.Bottom, translate(width, height), scale(-1, -1))
			line(bleed.Left, 0, 0, half.Bottom, translate(0, height), scale(1, -1))
			line(bleed.Right, 0, 0, half.Top, translate(width, 0), scale(-1, 1))
		case "cross":
			circle(half.Top, scale(0.5, 0.5), translate(width, half.Top), scale(0.5, 0.5))
			stroked(s, []transformStep{scale(0.5, 0.5), translate(width, 0)}, func() {
				s.MoveTo(-half.Top, half.Top)
				s.LineTo(-half.Top+bleed.Top, half.Top)
				s.MoveTo(0, 0)
				s.LineTo(0, bleed.Top)
			})
			circle(half.Bottom, translate(0, height), scale(0.5, 0.5), translate(width, -half.Bottom), scale(0.5, 0.5))
			stroked(s, []transformStep{translate(0, height), scale(0.5, 0.5), translate(width, 0)}, func() {
				s.MoveTo(-half.Bottom, -half.Bottom)
				s.LineTo(-half.Bottom+bleed.Bottom, -half.Bottom)
				s.MoveTo(0, 0)
				s.LineTo(0, -bleed.Bottom)
			})
			circle(half.Left, scale(0.5, 0.5), translate(half.Left, height), scale(0.5, 0.5))
			stroked(s, []transformStep{scale(0.5, 0.5), translate(0, height)}, func() {
				s.MoveTo(half.Left, -half.Left)
				s.LineTo(half.Left, -half.Left+bleed.Left)
				s.MoveTo(0, 0)
				s.LineTo(bleed.Left, 0)
			})
			circle(half.Right, translate(width, 0), scale(0.5, 0.5), translate(-half.Right, height), scale(0.5, 0.5))
			stroked(s, []transformStep{translate(width, 0), scale(0.5, 0.5), translate(0, height)}, func() {
				s.MoveTo(-half.Right, -half.Right)
				s.LineTo(-half.Right, -half.Right+bleed.Right)
				s.MoveTo(0, 0)
				s.LineTo(-bleed.Right, 0)
			})
		}
	}
}

// stroked applies steps to the graphics state, strokes the path built by
// path, and restores the state.
func stroked(s *stream.Stream, steps []transformStep, path func()) {
	s.PushState()
	defer s.PopState()
	for _, st := range steps {
		if st.scale {
			s.Transform(st.x, 0, 0, st.y, 0, 0)
		} else {
			s.Transform(1, 0, 0, 1, st.x, st.y)
		}
	}
	path()
	s.Stroke()
}

func circlePath(s *stream.Stream, r float64) {
	k := r * circleKappa
	s.MoveTo(r, 0)
	s.CurveTo(r, k, k, r, 0, r)
	s.CurveTo(-k, r, -r, k, -r, 0)
	s.CurveTo(-r, -k, -k, -r, 0, -r)
	s.CurveTo(k, -r, r, -k, r, 0)
	s.ClosePath()
}
