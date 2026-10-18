package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawBackground paints the background color of bg inside its painting area.
// The area is clipped to the boxes of the layer when clipBox is set. bleed
// and marks are the bleed area of a page and its crop marks and crosses.
func drawBackground(s *stream.Stream, bg *boxes.Background, clipBox bool, bleed *boxes.Sides, marks props.Keywords) {
	if bg == nil || len(bg.Layers) == 0 {
		return
	}
	last := bg.Layers[len(bg.Layers)-1]
	drawBoxShadows(s, last.ClippedBoxes, bg.Style, false)

	s.PushState()
	defer s.PopState()
	if clipBox {
		for _, b := range last.ClippedBoxes {
			roundedBox(s, b)
		}
		s.Clip(false)
		s.EndPath()
	}

	if bg.Color.Alpha > 0 {
		area := last.PaintingArea
		s.PushState()
		s.SetColor(bg.Color.Color, false)
		s.Rectangle(area.X, area.Y, area.Width, area.Height)
		s.Clip(false)
		s.EndPath()
		s.Rectangle(area.X, area.Y, area.Width, area.Height)
		s.Fill(false)
		s.PopState()
	}
}
