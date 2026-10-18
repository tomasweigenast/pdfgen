package draw

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawBoxShadows paints the outer or, when inset is set, the inner box
// shadows of style around the boxes that clip the background.
func drawBoxShadows(s *stream.Stream, clipped []boxes.RoundedBox, style *computed.Style, inset bool) {
	if style == nil {
		return
	}
	shadows := style.BoxShadow()
	// The first shadow is the closest to the viewer: paint it last.
	for i := len(shadows) - 1; i >= 0; i-- {
		sh := shadows[i]
		if sh.Inset != inset {
			continue
		}
		spread := sh.Spread.Dimension.Value
		if inset {
			spread = -spread
		}
		c := sh.Color.Color
		if sh.Color.IsCurrentColor() {
			c = getColor(style, props.PropColor)
		}
		shadow := boxShadow{
			tx: sh.X.Dimension.Value, ty: sh.Y.Dimension.Value,
			blur: sh.Blur.Dimension.Value, spread: spread,
			color: c, inset: inset,
			rendering: string(style.ImageRendering()),
		}
		for _, b := range clipped {
			s.PushState()
			shadow.draw(s, b)
			s.PopState()
		}
	}
}

type boxShadow struct {
	tx, ty, blur, spread float64
	color                syntax.Color
	inset                bool
	rendering            string
}

// spreadBox grows b by spread on every side and moves it by (tx, ty). A
// corner keeps a radius only if it stays rounded after the spread (CSS
// Backgrounds 3 §7.2).
func spreadBox(b boxes.RoundedBox, spread, tx, ty float64) boxes.RoundedBox {
	if spread == 0 && tx == 0 && ty == 0 {
		return b
	}
	spreadRadius := func(r boxes.Radius) boxes.Radius {
		if r.X >= -spread && r.Y >= -spread && r.X != 0 && r.Y != 0 {
			return boxes.Radius{X: r.X + spread, Y: r.Y + spread}
		}
		return boxes.Radius{}
	}
	return boxes.RoundedBox{
		Rect: boxes.Rect{
			X:      b.X + tx - spread,
			Y:      b.Y + ty - spread,
			Width:  math.Max(0, b.Width+2*spread),
			Height: math.Max(0, b.Height+2*spread),
		},
		TopLeft:     spreadRadius(b.TopLeft),
		TopRight:    spreadRadius(b.TopRight),
		BottomRight: spreadRadius(b.BottomRight),
		BottomLeft:  spreadRadius(b.BottomLeft),
	}
}

func (sh boxShadow) draw(s *stream.Stream, b boxes.RoundedBox) {
	s.SetColor(sh.color, false)

	shadowBox := spreadBox(b, sh.spread, sh.tx, sh.ty)
	innerBox := spreadBox(b, sh.spread-sh.blur, sh.tx, sh.ty)
	outerBox := spreadBox(b, sh.spread+sh.blur, sh.tx, sh.ty)

	// The shadow never paints inside the box of an outer shadow, nor
	// outside the box of an inset one.
	if !sh.inset {
		roundedBox(s, outerBox)
	}
	roundedBox(s, b)
	s.Clip(true)
	s.EndPath()

	if sh.blur == 0 {
		if sh.inset {
			roundedBox(s, b)
		}
		roundedBox(s, shadowBox)
		s.Fill(true)
		return
	}

	// The solid part: everything beyond the box for an inset shadow, the
	// inner box for an outer one.
	if sh.inset {
		roundedBox(s, b)
		roundedBox(s, outerBox)
		s.Fill(true)
	} else {
		roundedBox(s, innerBox)
		s.Fill(false)
	}

	transparent := sh.color
	transparent.Alpha = 0
	inColor, outColor := sh.color, transparent
	if sh.inset {
		inColor, outColor = transparent, sh.color
	}
	sh.drawBlurSides(s, innerBox, inColor, outColor)
	sh.drawBlurCorners(s, innerBox, outerBox, inColor, outColor)
}

// drawBlurSides fades the straight parts of the shadow edge with linear
// gradients, one per side.
func (sh boxShadow) drawBlurSides(s *stream.Stream, inner boxes.RoundedBox, inColor, outColor syntax.Color) {
	blur := sh.blur
	x, y, w, h := inner.X, inner.Y, inner.Width, inner.Height
	sides := [...]struct{ angle, e, f, w, h float64 }{
		{0, x + inner.TopLeft.X, y - 2*blur, w - inner.TopLeft.X - inner.TopRight.X, 2 * blur},
		{math.Pi / 2, x + w, y + inner.TopRight.Y, 2 * blur, h - inner.TopRight.Y - inner.BottomRight.Y},
		{math.Pi, x + inner.BottomLeft.X, y + h, w - inner.BottomLeft.X - inner.BottomRight.X, 2 * blur},
		{3 * math.Pi / 2, x - 2*blur, y + inner.TopLeft.Y, 2 * blur, h - inner.TopLeft.Y - inner.BottomLeft.Y},
	}
	for _, side := range sides {
		gradient := images.NewLinearGradient(&props.LinearGradient{
			Gradient: props.Gradient{
				ColorStops: []props.ColorStop{{Color: props.Color{Color: inColor}}, {Color: props.Color{Color: outColor}}},
				ColorHints: []props.Length{props.Px(blur * 1.25)},
			},
			DirectionType: "angle",
			Angle:         side.angle,
		}, sh.color)
		s.PushState()
		s.Transform(1, 0, 0, 1, side.e, side.f)
		gradient.Draw(s, side.w, side.h, sh.rendering)
		s.PopState()
	}
}

// drawBlurCorners fades the corners of the shadow edge with elliptical
// radial gradients, clipped to the band between the inner and outer boxes.
func (sh boxShadow) drawBlurCorners(s *stream.Stream, inner, outer boxes.RoundedBox, inColor, outColor syntax.Color) {
	blur := sh.blur
	x, y, w, h := inner.X, inner.Y, inner.Width, inner.Height
	corners := [...]struct {
		left, top, e, f float64
		r               boxes.Radius
	}{
		{100, 100, x - 2*blur, y - 2*blur, inner.TopLeft},
		{0, 100, x + w - inner.TopRight.X, y - 2*blur, inner.TopRight},
		{0, 0, x + w - inner.BottomRight.X, y + h - inner.BottomRight.Y, inner.BottomRight},
		{100, 0, x - 2*blur, y + h - inner.BottomLeft.Y, inner.BottomLeft},
	}

	s.PushState()
	defer s.PopState()
	roundedBox(s, inner)
	roundedBox(s, outer)
	s.Clip(true)
	s.EndPath()
	for _, c := range corners {
		width, height := 2*blur+c.r.X, 2*blur+c.r.Y
		gradient := images.NewRadialGradient(&props.RadialGradient{
			Gradient: props.Gradient{
				ColorStops: []props.ColorStop{
					{Color: props.Color{Color: inColor}, Position: lengthPtr(props.Px(c.r.X))},
					{Color: props.Color{Color: outColor}},
				},
				ColorHints: []props.Length{props.Px(c.r.X + blur*1.25)},
			},
			Shape:        "ellipse",
			SizeType:     "explicit",
			SizeExplicit: [2]props.Length{props.Px(width), props.Px(height)},
			Center: props.Position{
				OriginX: "left", X: percent(c.left),
				OriginY: "top", Y: percent(c.top),
			},
		}, sh.color)
		s.PushState()
		s.Transform(1, 0, 0, 1, c.e, c.f)
		gradient.Draw(s, width, height, sh.rendering)
		s.PopState()
	}
}

func lengthPtr(l props.Length) *props.Length { return &l }

func percent(v float64) props.Length {
	return props.Length{Dimension: props.Dimension{Value: v, Unit: "%"}}
}
