package draw

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// getColor returns the color property id of style, with currentcolor
// replaced by the value of color.
func getColor(style *computed.Style, id props.PropID) syntax.Color {
	return style.ResolvedColor(id).Color
}

// darken returns a darker version of c, keeping its alpha.
func darken(c syntax.Color) syntax.Color {
	h, s, v := rgbToHSV(srgbChannels(c))
	return hsvColor(h, s/1.25, v/1.5, c.Alpha)
}

// lighten returns a lighter version of c, keeping its alpha.
func lighten(c syntax.Color) syntax.Color {
	h, s, v := rgbToHSV(srgbChannels(c))
	if s != 0 {
		s = 1 - (1-s)/1.25
	}
	return hsvColor(h, s, 1-(1-v)/1.5, c.Alpha)
}

// styledColor returns the colors that draw a border side of the given style:
// inset and outset shade the color according to the side, and ridge and
// groove give the outer and the inner half of the border their own shade.
// For the other styles both results are c.
func styledColor(style props.Keyword, c syntax.Color, side string) (outer, inner syntax.Color) {
	topLeft := side == "top" || side == "left"
	switch style {
	case "inset", "outset":
		if topLeft != (style == "inset") {
			c = lighten(c)
		} else {
			c = darken(c)
		}
		return c, c
	case "ridge", "groove":
		if topLeft != (style == "ridge") {
			return lighten(c), darken(c)
		}
		return darken(c), lighten(c)
	}
	return c, c
}

// srgbChannels returns the red, green and blue channels of c in [0, 1]. A
// color that cannot be converted to sRGB is read as if it were sRGB.
func srgbChannels(c syntax.Color) (r, g, b float64) {
	if converted, ok := c.To(syntax.SRGB); ok {
		c = converted
	}
	if len(c.Coordinates) < 3 {
		return 0, 0, 0
	}
	return c.Coordinates[0], c.Coordinates[1], c.Coordinates[2]
}

func hsvColor(h, s, v, alpha float64) syntax.Color {
	r, g, b := hsvToRGB(h, s, v)
	return syntax.NewColor(syntax.SRGB, []float64{r, g, b}, alpha)
}

func rgbToHSV(r, g, b float64) (h, s, v float64) {
	maxc := math.Max(r, math.Max(g, b))
	minc := math.Min(r, math.Min(g, b))
	v = maxc
	if minc == maxc {
		return 0, 0, v
	}
	span := maxc - minc
	s = span / maxc
	rc, gc, bc := (maxc-r)/span, (maxc-g)/span, (maxc-b)/span
	switch maxc {
	case r:
		h = bc - gc
	case g:
		h = 2 + rc - bc
	default:
		h = 4 + gc - rc
	}
	h = math.Mod(h/6, 1)
	if h < 0 {
		h++
	}
	return h, s, v
}

func hsvToRGB(h, s, v float64) (r, g, b float64) {
	if s == 0 {
		return v, v, v
	}
	// The conversion keeps the compiler from fusing the product with the
	// subtraction, which gives a tiny negative remainder for h = 2/3.
	sector := float64(h * 6)
	i := math.Floor(sector)
	f := sector - i
	p, q, t := v*(1-s), v*(1-s*f), v*(1-s*(1-f))
	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}
