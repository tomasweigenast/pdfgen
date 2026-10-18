package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/layout"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawReplaced paints the content of a replaced box, such as an image.
func drawReplaced(s *stream.Stream, box boxes.Replaced) {
	r := box.ReplacedBase()
	if r.Style.Visibility() != "visible" || r.Width.V == 0 || r.Height.V == 0 || r.Replacement == nil {
		return
	}
	rect := layout.ReplacedBoxLayout(box)
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}

	s.PushState()
	s.SetAlpha(1, false, nil)
	s.Transform(1, 0, 0, 1, rect.X, rect.Y)
	s.PushState()
	r.Replacement.Draw(s, rect.Width, rect.Height, string(r.Style.ImageRendering()))
	s.PopState()
	s.PopState()
}
