package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawBoxShadows paints the outer or, when inset is set, the inner box
// shadows of style around the boxes that clip the background.
func drawBoxShadows(s *stream.Stream, clipped []boxes.RoundedBox, style *computed.Style, inset bool) {
}
