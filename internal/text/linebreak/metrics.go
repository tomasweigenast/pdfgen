package linebreak

import "github.com/tomasweigenast/pdfgen/internal/text/fonts"

// LineSize returns the logical width and height of a line in px. The
// letter-spacing that the ends of the line lose is added back to the
// width, as text boxes keep the spacing after their last character.
func LineSize(line *Line, style *Style) (width, height float64) {
	_, h := line.extents()
	width = fonts.FromUnits(int32(line.Width())) + style.LetterSpacing
	return width, fonts.FromUnits(int32(h))
}

// Baseline returns the distance in px from the top of the first line to
// its baseline.
func (l *Layout) Baseline() float64 {
	y, _ := l.FirstLine().extents()
	return fonts.FromUnits(int32(-y))
}
