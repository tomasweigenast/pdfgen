package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawOutline paints the outline of box and of its descendants. Outlines are
// never rounded and they are painted over the content (CSS UI 3 §5.1).
func drawOutline(s *stream.Stream, box boxes.Box) {
	base := box.Base()
	style := base.Style
	width := float64(style.OutlineWidth())
	offset := float64(style.OutlineOffset())
	color := getColor(style, props.PropOutlineColor)
	outlineStyle := style.OutlineStyle()

	if style.Visibility() == "visible" && width != 0 && color.Alpha != 0 {
		outer := boxes.Rect{
			X:      base.BorderBoxX() - width - offset,
			Y:      base.BorderBoxY() - width - offset,
			Width:  base.BorderWidth() + 2*width + 2*offset,
			Height: base.BorderHeight() + 2*width + 2*offset,
		}
		widths := [4]float64{width, width, width, width}
		for _, side := range borderSides {
			s.PushState()
			clipBorderSegment(s, outlineStyle, side, outer, widths, [4]boxes.Radius{})
			drawRectBorder(s, outer, widths, outlineStyle, styledColors(outlineStyle, color, side))
			s.PopState()
		}
	}

	for _, child := range boxes.ChildrenOf(box) {
		drawOutline(s, child)
	}
}
