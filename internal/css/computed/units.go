package computed

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// ToPixels converts a length to CSS pixels for the named property (CSS
// Values 4 §6). Font-relative units refer to the font of the style, and to
// the font of the parent for font-size itself; viewport units refer to the
// first page.
func (s *Style) ToPixels(value float64, unit, property string) float64 {
	unit = strings.ToLower(unit)
	if value == 0 {
		return 0
	}
	if unit == "px" {
		return value
	}
	if px, ok := units.AbsoluteToPixels(value, unit); ok {
		return px
	}
	switch {
	case units.IsFontRelative(unit):
		return s.fontRelativeToPixels(value, unit, property)
	case units.IsViewportRelative(unit):
		return s.viewportToPixels(value, unit)
	}
	return value
}

func (s *Style) fontRelativeToPixels(value float64, unit, property string) float64 {
	fontSize := float64(s.FontSize())
	if property == "font-size" {
		fontSize = pixelsOf(s.ParentValue(props.PropFontSize))
	}
	switch unit {
	case "lh":
		// The line height of these properties depends on the font size and
		// line height of the element itself, so the parent is used.
		strut := s
		if property == "font-size" || property == "line-height" {
			strut = s.parent
			if strut == nil {
				strut = s.root
			}
		}
		lineHeight, _ := strut.Strut()
		return value * lineHeight
	case "rlh":
		lineHeight, _ := s.root.Strut()
		return value * lineHeight
	case "em":
		return value * fontSize
	case "rem":
		return value * float64(s.root.FontSize())
	}
	if strings.HasPrefix(unit, "r") {
		ratio := s.root.CharacterRatio(unit[1:])
		return value * float64(s.root.FontSize()) * ratio
	}
	return value * fontSize * s.CharacterRatio(unit)
}

func (s *Style) viewportToPixels(value float64, unit string) float64 {
	size := s.ctx.PageArea
	if unit[0] == 'p' {
		size = s.ctx.PageBox
	}
	var width, height float64
	if size == nil {
		logging.Warnf("%s unit resolved before first page layout", unit)
		width, _ = units.AbsoluteToPixels(props.InitialPageSize.Width, props.InitialPageSize.Unit)
		height, _ = units.AbsoluteToPixels(props.InitialPageSize.Height, props.InitialPageSize.Unit)
	} else {
		width, height = size.Width, size.Height
	}
	// Writing modes are not supported: the inline axis is horizontal.
	switch {
	case strings.HasSuffix(unit, "vw"), strings.HasSuffix(unit, "vi"):
		return value / 100 * width
	case strings.HasSuffix(unit, "vh"), strings.HasSuffix(unit, "vb"):
		return value / 100 * height
	case strings.HasSuffix(unit, "vmin"):
		return value / 100 * min(width, height)
	case strings.HasSuffix(unit, "vmax"):
		return value / 100 * max(width, height)
	}
	return value
}

// pixelsOf returns the size in pixels of a computed length, and 0 for
// keywords and math functions.
func pixelsOf(v props.Value) float64 {
	switch v := v.(type) {
	case props.Pixels:
		return float64(v)
	case props.Length:
		if v.Keyword == "" && v.Calc == nil {
			return v.Dimension.Value
		}
	}
	return 0
}
