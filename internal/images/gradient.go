package images

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// gradientKind is the shape a gradient is painted with.
type gradientKind int

const (
	// gradientNone paints nothing, for boxes and sizes without a gradient.
	gradientNone gradientKind = iota
	gradientSolid
	gradientLinear
	gradientRadial
)

// gradientLayout is a gradient resolved against the size of its box.
type gradientLayout struct {
	// scaleY is the vertical scale applied before painting, which turns the
	// circles of a radial gradient into ellipses.
	scaleY float64
	kind   gradientKind
	// points are the coordinates of the start and end of a linear gradient
	// (x0, y0, x1, y1) or the starting and ending circles of a radial one
	// (cx0, cy0, r0, cx1, cy1, r1).
	points []float64
	// positions are the stops between 0 at the start and 1 at the end.
	positions []float64
	colors    []syntax.Color
	// hints are the exponents of the interpolation between each pair of
	// stops.
	hints []float64
	// extend paints the first and last colors beyond the vector, which is
	// what repeating gradients must not do.
	extend bool
}

func solidLayout(c syntax.Color) gradientLayout {
	return gradientLayout{scaleY: 1, kind: gradientSolid, colors: []syntax.Color{c}}
}

// gradient holds what linear and radial gradients have in common.
type gradient struct {
	colors    []syntax.Color
	stops     []*props.Length
	hints     []props.Length
	repeating bool
}

func newGradient(g props.Gradient, current syntax.Color) gradient {
	out := gradient{repeating: g.Repeating}
	for _, stop := range g.ColorStops {
		c := stop.Color.Color
		if c.IsCurrentColor() {
			c = current
		}
		out.colors = append(out.colors, c)
		out.stops = append(out.stops, stop.Position)
	}
	if len(out.colors) == 0 {
		out.colors = []syntax.Color{syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)}
		out.stops = []*props.Length{nil}
	}
	out.hints = append(out.hints, g.ColorHints...)
	for len(out.hints) < len(out.colors)-1 {
		out.hints = append(out.hints, props.Length{Dimension: props.Dimension{Value: 50, Unit: "%"}})
	}
	return out
}

// GradientFromProps returns the gradient img computes to. currentColor
// replaces currentcolor in the color stops. It reports false when img is not
// a gradient.
func GradientFromProps(img props.Image, currentColor syntax.Color) (Image, bool) {
	switch {
	case img.Type == "linear-gradient" && img.Linear != nil:
		return NewLinearGradient(img.Linear, currentColor), true
	}
	return nil, false
}

// IntrinsicSize reports that gradients have no natural size or ratio.
func (g *gradient) IntrinsicSize(resolution, fontSize float64) (w, h, ratio units.MaybeFloat) {
	return units.Auto, units.Auto, units.Auto
}

// paint draws a laid-out gradient into a w×h box at the origin of s. When a
// color stop is not opaque the gradient is painted through a luminosity soft
// mask holding the alphas (ISO 32000-1 §11.6.5.2), as shadings carry no
// alpha of their own.
func paint(s *stream.Stream, w, h float64, l gradientLayout) {
	switch l.kind {
	case gradientNone:
		return
	case gradientSolid:
		s.Rectangle(0, 0, w, h)
		s.SetColor(l.colors[0], false)
		s.Fill(false)
		return
	}

	n := len(l.colors)
	alphas := make([]float64, n)
	for i, c := range l.colors {
		alphas[i] = c.Alpha
	}
	hint := func(i int) float64 {
		if i < len(l.hints) {
			return l.hints[i]
		}
		return 1
	}

	type couple struct {
		c0, c1 [3]float64
		hint   float64
	}
	couples := make([]couple, n-1)
	for i := range couples {
		var c couple
		c.c0[0], c.c0[1], c.c0[2] = srgbComponents(l.colors[i])
		c.c1[0], c.c1[1], c.c1[2] = srgbComponents(l.colors[i+1])
		c.hint = hint(i)
		couples[i] = c
	}
	for i, alpha := range alphas {
		if alpha != 0 {
			continue
		}
		if i > 0 {
			couples[i-1].c1 = couples[i-1].c0
		}
		if i < n-1 {
			couples[i].c0 = couples[i].c1
		}
	}
	for i := range couples {
		a0, a1 := alphas[i], alphas[i+1]
		if a0 != 0 && a1 != 0 && (a0 != 1 || a1 != 1) {
			couples[i].hint = a0 / a1
		}
	}

	shadingType := 2
	if l.kind == gradientRadial {
		shadingType = 3
	}
	domain := []float64{l.positions[0], l.positions[len(l.positions)-1]}
	encode := make([]float64, 0, 2*(n-1))
	for range n - 1 {
		encode = append(encode, 0, 1)
	}
	bounds := l.positions[1 : len(l.positions)-1]
	extend := l.extend

	subs := make([]stream.Function, len(couples))
	for i, c := range couples {
		subs[i] = stream.InterpolationFunction([]float64{0, 1}, c.c0[:], c.c1[:], c.hint)
	}
	fn := stream.StitchingFunction(domain, encode, bounds, subs)
	shading := s.AddShading(shadingType, domain, l.points, extend, fn, "")
	s.Transform(1, 0, 0, l.scaleY, 0, 0)

	if hasAlpha(alphas) {
		mask := s.SetAlphaState(0, 0, w, h/l.scaleY, "luminosity")
		alphaSubs := make([]stream.Function, len(couples))
		for i := range alphaSubs {
			alphaSubs[i] = stream.InterpolationFunction([]float64{0, 1}, []float64{alphas[i]}, []float64{alphas[i+1]}, hint(i))
		}
		alphaFn := stream.StitchingFunction(domain, encode, bounds, alphaSubs)
		alphaShading := mask.AddShading(shadingType, domain, l.points, extend, alphaFn, "DeviceGray")
		mask.PaintShading(alphaShading)
	}
	s.PaintShading(shading)
}

func hasAlpha(alphas []float64) bool {
	for _, a := range alphas {
		if a != 1 {
			return true
		}
	}
	return false
}
