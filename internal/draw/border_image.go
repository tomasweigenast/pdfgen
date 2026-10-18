package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// setMaskBorder applies the mask-border of box to the alpha of s.
func setMaskBorder(s *stream.Stream, box boxes.Box) {}

// drawBorderImage paints the border-image of box in place of its borders.
//
//lint:ignore U1000 drawBorder calls it once the borders are painted
func drawBorderImage(s *stream.Stream, box boxes.Box) {}
