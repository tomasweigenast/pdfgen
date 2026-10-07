package computed

import "github.com/tomasweigenast/pdfgen/internal/css/props"

// ResolvedColor returns the value of a color property with currentcolor
// replaced by the value of color (CSS Color 4 §6.4).
func (s *Style) ResolvedColor(id props.PropID) props.Color {
	c, _ := s.Get(id).(props.Color)
	if c.IsCurrentColor() {
		return s.Color()
	}
	return c
}
