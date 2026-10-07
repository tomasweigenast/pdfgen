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

// Strut returns the height of a line box for the style and the position
// of its baseline from the top (CSS 2.1 §10.8.1). A normal line-height
// gives the height of the font.
func Strut(ctx *Context, style *Style) (height, baseline float64) {
	if style.Font.Size == 0 {
		return 0, 0
	}
	layout := newLayout(ctx, style, 0, nil)
	layout.SetText(" ")
	line := layout.FirstLine()
	_, textHeight := LineSize(line, style)
	baseline = layout.Baseline()
	lh := style.LineHeight
	if lh.Normal {
		return textHeight, baseline
	}
	height = lh.Value
	if lh.Number {
		height *= style.Font.Size
	}
	return height, baseline + (height-textHeight)/2
}

// CharacterRatio returns the size of a font-relative unit ("ex", "ch",
// "cap" or "ic") as a fraction of the font size.
func CharacterRatio(ctx *Context, style *Style, unit string) float64 {
	return ctx.Fonts.CharacterRatio(style.Font, style.shapingLanguage(), style.features(), unit)
}

// Metrics returns the vertical metrics of the font of the layout, used to
// draw text decorations.
func (l *Layout) Metrics() fonts.Metrics {
	face := l.primaryFace()
	if face == nil {
		return fonts.Metrics{}
	}
	return face.Metrics(l.Style.Font.Size)
}
