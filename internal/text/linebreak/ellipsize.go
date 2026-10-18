package linebreak

import (
	"slices"

	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/shape"
)

// EllipsizeEnd makes the first line of the layout end with an ellipsis when it
// is wider than maxWidth px, instead of wrapping. The layout must not wrap
// (see Wrap with a negative width, or Reactivate): the line is the whole text
// up to a line feed.
func (l *Layout) EllipsizeEnd(maxWidth float64) {
	l.ellipsize = true
	l.ellipsizeWidth = int(max(0, maxWidth) * fonts.UnitsPerPx)
	l.lines, l.built = nil, false
}

// Wrap sets the width in px that lines wrap at. A negative width turns
// wrapping off.
func (l *Layout) Wrap(width float64) {
	if width < 0 || width >= maxWrapWidth {
		l.setWidth(-1)
		return
	}
	l.setWidth(int(width * fonts.UnitsPerPx))
}

// FirstLineAndRemainder returns the first line and the byte offset where the
// second line starts, or -1 when the text ends on the first line.
func (l *Layout) FirstLineAndRemainder() (*Line, int) { return l.firstLine() }

// ellipsizeLine drops whole glyph clusters from the end of the line, which
// is its right side for a left-to-right paragraph and its left side
// otherwise, until the rest and an ellipsis fit in the limit. The ellipsis
// is set in the font of the last cluster kept. When nothing fits, only the
// ellipsis remains. The end of a line mixing directions is taken from the
// direction of its paragraph.
func (l *Layout) ellipsizeLine(line *Line) {
	if line.Width() <= l.ellipsizeWidth || len(line.Runs) == 0 {
		return
	}
	// Runs are copied so that the glyph slices can be cut without touching
	// the shaped items.
	runs := make([]*Run, len(line.Runs))
	for i, r := range line.Runs {
		c := *r
		runs[i] = &c
	}
	width := line.Width()
	atEnd := func() *Run { // the run holding the end of the line
		if line.RTL {
			return runs[0]
		}
		return runs[len(runs)-1]
	}
	dropEnd := func() { // forget the run at the end of the line
		if line.RTL {
			runs = runs[1:]
		} else {
			runs = runs[:len(runs)-1]
		}
	}

	var last *Run // font of the ellipsis when nothing is kept
	removedAt := -1
	var ellipsis *Run
	for {
		var ew int
		face := last
		if len(runs) > 0 {
			face = atEnd()
		}
		ellipsis = l.ellipsisRun(face, line)
		ew = ellipsis.Width()
		if width+ew <= l.ellipsizeWidth || len(runs) == 0 {
			break
		}
		r := atEnd()
		if len(r.Glyphs) == 0 {
			dropEnd()
			continue
		}
		last = r
		n := clusterAtEnd(r.Glyphs, line.RTL)
		var cut []Glyph
		if line.RTL {
			cut, r.Glyphs = r.Glyphs[:n], r.Glyphs[n:]
		} else {
			cut, r.Glyphs = r.Glyphs[len(r.Glyphs)-n:], r.Glyphs[:len(r.Glyphs)-n]
		}
		for _, g := range cut {
			width -= int(g.XAdvance)
		}
		if !r.RTL() && !line.RTL {
			removedAt = r.Start + cut[0].Cluster
			r.End = removedAt
		}
		if len(r.Glyphs) == 0 {
			dropEnd()
		}
	}

	if line.RTL {
		runs = slices.Insert(runs, 0, ellipsis)
	} else {
		runs = append(runs, ellipsis)
	}
	line.Runs = runs
	if removedAt >= 0 {
		line.Length = removedAt - line.Start
	}
}

// clusterAtEnd returns the number of glyphs of the cluster at the end of
// glyphs: the last ones, or the first ones when atStart.
func clusterAtEnd(glyphs []Glyph, atStart bool) int {
	if atStart {
		n := 1
		for n < len(glyphs) && glyphs[n].Cluster == glyphs[0].Cluster {
			n++
		}
		return n
	}
	n := 1
	for n < len(glyphs) && glyphs[len(glyphs)-1-n].Cluster == glyphs[len(glyphs)-1].Cluster {
		n++
	}
	return n
}

// ellipsisRun shapes the ellipsis for a line, in the font of like. It is
// "…" when a font has it and three full stops otherwise.
func (l *Layout) ellipsisRun(like *Run, line *Line) *Run {
	face, size := l.primaryFace(), l.Style.Font.Size
	if like != nil {
		face, size = like.Face, like.Size
	}
	text := "…"
	if face == nil || !face.HasGlyph('…') {
		if alt := l.ctx.Fonts.Query(l.description(), l.Style.shapingLanguage(), '…'); alt != nil && alt.HasGlyph('…') {
			face = alt
		} else {
			text = "..."
		}
	}
	var lang language.Language
	if s := l.Style.shapingLanguage(); s != "" {
		lang = language.NewLanguage(s)
	}
	runes := []rune(text)
	glyphs := l.ctx.Shaper.Shape(shape.Run{
		Text: runes, End: len(runes), Face: face, Size: size,
		Features: l.Style.features(), Lang: lang,
	})
	run := &Run{Face: face, Size: size, Ellipsis: text}
	if line.RTL {
		run.Level = 1
	}
	for _, g := range glyphs {
		run.Glyphs = append(run.Glyphs, Glyph{Glyph: g})
	}
	if like != nil {
		run.Start, run.End = like.End, like.End
	}
	return run
}
