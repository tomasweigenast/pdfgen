package draw

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// drawText paints the text of tb, moved by offsetX. textOverflow and
// blockEllipsis are the values of the line that contains it.
func drawText(s *stream.Stream, tb *boxes.TextBox, offsetX float64, textOverflow props.Keyword, blockEllipsis props.KeywordOrString) {
}
