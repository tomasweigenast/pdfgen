package draw

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/layout"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// borderImage holds the parameters shared by border-image-* and mask-border-*.
type borderImage struct {
	image  images.Image
	slice  props.ImageSlice
	repeat props.KeywordPair
	outset props.LengthQuad
	width  props.LengthQuad
}

// sourceImage returns the image of a border-image-source or mask-border-source
// value: the loaded one, or a gradient built from the value. It is nil when
// there is none.
func sourceImage(loaded images.Image, source props.Image, style *computed.Style) images.Image {
	if source.Type == "none" {
		return nil
	}
	if loaded != nil {
		return loaded
	}
	if g, ok := images.GradientFromProps(source, style.ResolvedColor(props.PropColor).Color); ok {
		return g
	}
	return nil
}

// setMaskBorder applies the mask-border of box to the alpha of s.
func setMaskBorder(s *stream.Stream, box boxes.Box) {
	base := box.Base()
	style := base.Style
	img := sourceImage(base.MaskBorderImage, style.MaskBorderSource(), style)
	if img == nil {
		return
	}
	b := box.RoundedBorderBox()
	mode := string(style.MaskBorderMode())
	if mode == "luminance" {
		mode = "luminosity"
	}
	mask := s.SetAlphaState(b.X, b.Y, b.Width, b.Height, mode)
	drawBorderImageParts(mask, box, borderImage{
		image:  img,
		slice:  style.MaskBorderSlice(),
		repeat: style.MaskBorderRepeat(),
		outset: style.MaskBorderOutset(),
		width:  style.MaskBorderWidth(),
	})
}

// drawBorderImage paints the border-image of box in place of its borders.
func drawBorderImage(s *stream.Stream, box boxes.Box) {
	base := box.Base()
	style := base.Style
	img := sourceImage(base.BorderImage, style.BorderImageSource(), style)
	if img == nil {
		return
	}
	drawBorderImageParts(s, box, borderImage{
		image:  img,
		slice:  style.BorderImageSlice(),
		repeat: style.BorderImageRepeat(),
		outset: style.BorderImageOutset(),
		width:  style.BorderImageWidth(),
	})
}

// drawBorderImageParts cuts the image in nine regions and paints each of them
// in the matching part of the border image area (CSS Backgrounds 3 §6).
func drawBorderImageParts(s *stream.Stream, box boxes.Box, bi borderImage) {
	base := box.Base()
	style := base.Style
	w, h, ratio := bi.image.IntrinsicSize(float64(style.ImageResolution()), float64(style.FontSize()))
	intrinsicW, intrinsicH := layout.DefaultImageSizing(w, h, ratio, base.BorderWidth(), base.BorderHeight())

	sliceTop := sliceDimension(bi.slice.Edges[0], intrinsicH)
	sliceRight := sliceDimension(bi.slice.Edges[1], intrinsicW)
	sliceBottom := sliceDimension(bi.slice.Edges[2], intrinsicH)
	sliceLeft := sliceDimension(bi.slice.Edges[3], intrinsicW)

	border := box.RoundedBorderBox()
	padding := box.RoundedPaddingBox()
	x, y, bw, bh := border.X, border.Y, border.Width, border.Height
	borderLeft := padding.X - x
	borderTop := padding.Y - y
	borderRight := bw - padding.Width - borderLeft
	borderBottom := bh - padding.Height - borderTop

	outsetTop := outsetDimension(bi.outset[0], borderTop)
	outsetRight := outsetDimension(bi.outset[1], borderRight)
	outsetBottom := outsetDimension(bi.outset[2], borderBottom)
	outsetLeft := outsetDimension(bi.outset[3], borderLeft)

	x -= outsetLeft
	y -= outsetTop
	bw += outsetLeft + outsetRight
	bh += outsetTop + outsetBottom

	// Numeric outsets are relative to the border width, not to the
	// border-image-width, and the percentages of border-image-width refer to
	// the area that includes the outsets: widths are adjusted afterwards.
	borderTop = widthAdjustment(bi.width[0], borderTop, sliceTop, bh)
	borderRight = widthAdjustment(bi.width[1], borderRight, sliceRight, bw)
	borderBottom = widthAdjustment(bi.width[2], borderBottom, sliceBottom, bh)
	borderLeft = widthAdjustment(bi.width[3], borderLeft, sliceLeft, bw)

	rendering := string(style.ImageRendering())
	region := func(r imageRegion) (float64, float64) {
		r.image = bi.image
		r.intrinsicW, r.intrinsicH = intrinsicW, intrinsicH
		r.rendering = rendering
		return r.draw(s)
	}
	stretch := props.Keyword("stretch")
	repeatX, repeatY := bi.repeat[0], bi.repeat[1]

	scaleLeft, scaleTop := region(imageRegion{
		x: x, y: y, w: borderLeft, h: borderTop,
		sliceW: sliceLeft, sliceH: sliceTop, repeatX: stretch, repeatY: stretch,
	})
	region(imageRegion{
		x: x + bw - borderRight, y: y, w: borderRight, h: borderTop,
		sliceX: intrinsicW - sliceRight, sliceW: sliceRight, sliceH: sliceTop,
		repeatX: stretch, repeatY: stretch,
	})
	scaleRight, scaleBottom := region(imageRegion{
		x: x + bw - borderRight, y: y + bh - borderBottom, w: borderRight, h: borderBottom,
		sliceX: intrinsicW - sliceRight, sliceY: intrinsicH - sliceBottom,
		sliceW: sliceRight, sliceH: sliceBottom, repeatX: stretch, repeatY: stretch,
	})
	region(imageRegion{
		x: x, y: y + bh - borderBottom, w: borderLeft, h: borderBottom,
		sliceY: intrinsicH - sliceBottom, sliceW: sliceLeft, sliceH: sliceBottom,
		repeatX: stretch, repeatY: stretch,
	})

	xMiddle := sliceLeft+sliceRight < intrinsicW
	if xMiddle {
		region(imageRegion{
			x: x + borderLeft, y: y, w: bw - borderLeft - borderRight, h: borderTop,
			sliceX: sliceLeft, sliceW: intrinsicW - sliceLeft - sliceRight, sliceH: sliceTop,
			repeatX: repeatX, repeatY: stretch,
		})
		region(imageRegion{
			x: x + borderLeft, y: y + bh - borderBottom, w: bw - borderLeft - borderRight, h: borderBottom,
			sliceX: sliceLeft, sliceY: intrinsicH - sliceBottom,
			sliceW: intrinsicW - sliceLeft - sliceRight, sliceH: sliceBottom,
			repeatX: repeatX, repeatY: stretch,
		})
	}
	yMiddle := sliceTop+sliceBottom < intrinsicH
	if yMiddle {
		region(imageRegion{
			x: x + bw - borderRight, y: y + borderTop, w: borderRight, h: bh - borderTop - borderBottom,
			sliceX: intrinsicW - sliceRight, sliceY: sliceTop,
			sliceW: sliceRight, sliceH: intrinsicH - sliceTop - sliceBottom,
			repeatX: stretch, repeatY: repeatY,
		})
		region(imageRegion{
			x: x, y: y + borderTop, w: borderLeft, h: bh - borderTop - borderBottom,
			sliceY: sliceTop, sliceW: sliceLeft, sliceH: intrinsicH - sliceTop - sliceBottom,
			repeatX: stretch, repeatY: repeatY,
		})
	}
	if bi.slice.Fill && xMiddle && yMiddle {
		region(imageRegion{
			x: x + borderLeft, y: y + borderTop,
			w: bw - borderLeft - borderRight, h: bh - borderTop - borderBottom,
			sliceX: sliceLeft, sliceY: sliceTop,
			sliceW: intrinsicW - sliceLeft - sliceRight, sliceH: intrinsicH - sliceTop - sliceBottom,
			repeatX: repeatX, repeatY: repeatY,
			scaleX: firstNonZero(scaleLeft, scaleRight), scaleY: firstNonZero(scaleTop, scaleBottom),
		})
	}
}

func firstNonZero(a, b float64) float64 {
	if a != 0 {
		return a
	}
	return b
}

// sliceDimension resolves one value of border-image-slice against the image
// size: numbers are image pixels and percentages are capped at 100%.
func sliceDimension(d props.Dimension, intrinsic float64) float64 {
	if d.Unit == "%" {
		return math.Min(100, d.Value) / 100 * intrinsic
	}
	return math.Min(d.Value, intrinsic)
}

// outsetDimension resolves one value of border-image-outset: a number is a
// multiple of the border width.
func outsetDimension(l props.Length, border float64) float64 {
	if l.Dimension.Unit == "" {
		return l.Dimension.Value * border
	}
	return l.Dimension.Value
}

// widthAdjustment resolves one value of border-image-width. original is the
// border width, intrinsic the slice size and area the size of the border
// image area on that axis.
func widthAdjustment(l props.Length, original, intrinsic, area float64) float64 {
	switch {
	case l.IsAuto():
		return intrinsic
	case l.Dimension.Unit == "":
		return l.Dimension.Value * original
	case l.Dimension.Unit == "%":
		return l.Dimension.Value / 100 * area
	}
	return l.Dimension.Value
}

// imageRegion is one of the nine parts of a border image: the rectangle of
// the border image area it is painted in, and the slice of the image that
// fills it.
type imageRegion struct {
	image                  images.Image
	rendering              string
	intrinsicW, intrinsicH float64

	x, y, w, h       float64
	sliceX, sliceY   float64
	sliceW, sliceH   float64
	repeatX, repeatY props.Keyword
	scaleX, scaleY   float64
}

// draw paints the region and returns the scale applied to the slice on each
// axis, zero when nothing was painted.
func (r imageRegion) draw(s *stream.Stream) (scaleX, scaleY float64) {
	scaleX, scaleY = r.scaleX, r.scaleY
	var nX, nY int
	var extraDX, extraDY float64

	if r.intrinsicW == 0 || r.w == 0 || r.sliceW == 0 {
		scaleX = 0
	} else {
		if scaleX == 0 {
			scaleX = 1
			if r.h != 0 && r.sliceH != 0 {
				scaleX = r.h / r.sliceH
			}
		}
		switch r.repeatX {
		case "repeat":
			nX = int(math.Ceil(r.w / r.sliceW / scaleX))
		case "space":
			nX = int(math.Floor(r.w / r.sliceW / scaleX))
			// There is one more space than repeats: one before the first
			// and one after the last.
			extraDX = (r.w/scaleX - float64(nX)*r.sliceW) / float64(nX+1)
		case "round":
			nX = max(1, int(math.RoundToEven(r.w/r.sliceW/scaleX)))
			scaleX = r.w / (float64(nX) * r.sliceW)
		default:
			nX = 1
			scaleX = r.w / r.sliceW
		}
	}

	if r.intrinsicH == 0 || r.h == 0 || r.sliceH == 0 {
		scaleY = 0
	} else {
		if scaleY == 0 {
			scaleY = 1
			if r.w != 0 && r.sliceW != 0 {
				scaleY = r.w / r.sliceW
			}
		}
		switch r.repeatY {
		case "repeat":
			nY = int(math.Ceil(r.h / r.sliceH / scaleY))
		case "space":
			nY = int(math.Floor(r.h / r.sliceH / scaleY))
			extraDY = (r.h/scaleY - float64(nY)*r.sliceH) / float64(nY+1)
		case "round":
			nY = max(1, int(math.RoundToEven(r.h/r.sliceH/scaleY)))
			scaleY = r.h / (float64(nY) * r.sliceH)
		default:
			nY = 1
			scaleY = r.h / r.sliceH
		}
	}

	if scaleX == 0 || scaleY == 0 {
		return scaleX, scaleY
	}

	offsetX := r.intrinsicW * scaleX * r.sliceX / r.intrinsicW
	offsetY := r.intrinsicH * scaleY * r.sliceY / r.intrinsicH

	s.PushState()
	s.Rectangle(r.x, r.y, r.w, r.h)
	s.Clip(false)
	s.EndPath()
	s.Transform(1, 0, 0, 1, r.x-offsetX+extraDX, r.y-offsetY+extraDY)
	s.Transform(scaleX, 0, 0, scaleY, 0, 0)
	for i := 0; i < nX; i++ {
		for j := 0; j < nY; j++ {
			s.PushState()
			s.Transform(1, 0, 0, 1, float64(i)*(r.sliceW+extraDX), float64(j)*(r.sliceH+extraDY))
			s.Rectangle(offsetX/scaleX, offsetY/scaleY, r.sliceW, r.sliceH)
			s.Clip(false)
			s.EndPath()
			r.image.Draw(s, r.intrinsicW, r.intrinsicH, r.rendering)
			s.PopState()
		}
	}
	s.PopState()
	return scaleX, scaleY
}
