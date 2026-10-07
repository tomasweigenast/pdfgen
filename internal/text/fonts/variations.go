package fonts

import (
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// VariationsFor returns the variation coordinates for a variable face: wght
// from the font weight, opsz from the font size in px (truncated to the
// 1/1024 px grid), then the font-variation-settings, the last value of an
// axis winning. The result is empty for a static font.
func (f *Face) VariationsFor(weight int, size float64, settings props.FontSettings) []font.Variation {
	if !f.IsVariable() {
		return nil
	}
	if weight == 0 {
		weight = 400
	}
	values := map[font.Tag]float32{}
	var order []font.Tag
	set := func(tag font.Tag, v float32) {
		if _, ok := values[tag]; !ok {
			order = append(order, tag)
		}
		values[tag] = v
	}
	for _, a := range f.axes {
		switch a.Tag.String() {
		case "wght":
			set(a.Tag, float32(weight))
		case "opsz":
			set(a.Tag, float32(FromUnits(ToUnits(size))))
		}
	}
	if !settings.Normal {
		for _, item := range settings.Items {
			if len(item.Tag) != 4 {
				continue
			}
			set(opentype.MustNewTag(item.Tag), float32(item.Value))
		}
	}
	out := make([]font.Variation, len(order))
	for i, tag := range order {
		out[i] = font.Variation{Tag: tag, Value: values[tag]}
	}
	return out
}
