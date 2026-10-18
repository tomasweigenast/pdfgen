package draw

import (
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/layout"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
)

// minFontSize is the smallest font size worth drawing: the PDF writer keeps
// six decimals.
const minFontSize = 1e-6

// drawText paints the text of tb, moved by offsetX. textOverflow and
// blockEllipsis are the values of the line that contains it.
func drawText(s *stream.Stream, tb *boxes.TextBox, offsetX float64, textOverflow props.Keyword, blockEllipsis props.KeywordOrString) {
	style := tb.Style
	lay := tb.Layout
	fontSize := float64(style.FontSize())
	if lay == nil || fontSize < minFontSize || style.Visibility() != "visible" {
		return
	}

	lines := style.TextDecorationLine()
	hasLine := func(name props.Keyword) bool {
		return slices.Contains(lines, name)
	}
	metrics := lay.Metrics()
	baseline := 0.0
	if tb.Baseline != nil {
		baseline = *tb.Baseline
	}
	color := getColor(style, props.PropTextDecorationColor)
	decorate := func(offsetY, thickness float64) {
		y := tb.PositionY + offsetY
		drawLine(s, tb.PositionX, y, tb.PositionX+tb.Width.V, y, thickness,
			style.TextDecorationStyle(), [2]syntax.Color{color, color}, offsetX)
	}

	// Underline and overline are drawn under the text, line-through over it.
	var thickness float64
	if hasLine("underline") || hasLine("overline") {
		thickness = unitsToPx(metrics.UnderlineThickness)
		if t := style.TextDecorationThickness(); !t.IsKeyword() {
			thickness = lengthOrPercent(t, style, fontSize)
		}
	}
	if hasLine("overline") {
		decorate(baseline-metrics.Baseline()+thickness/2, thickness)
	}
	if hasLine("underline") {
		offset := -unitsToPx(metrics.UnderlinePosition)
		if o := style.TextUnderlineOffset(); !o.IsAuto() {
			offset = lengthOrPercent(o, style, fontSize)
		}
		decorate(baseline+offset+thickness/2, thickness)
	}

	s.SetColor(getColor(style, props.PropColor), false)
	lay.Reactivate(style.LineBreakStyle())
	s.BeginText()
	drawFirstLine(s, tb, textOverflow, blockEllipsis, tb.PositionX, tb.PositionY+baseline)
	s.EndText()

	if hasLine("line-through") {
		decorate(baseline-unitsToPx(metrics.StrikethroughPosition), unitsToPx(metrics.StrikethroughThickness))
	}
}

// lengthOrPercent returns l in px, a percentage being of referTo.
func lengthOrPercent(l props.Length, style *computed.Style, referTo float64) float64 {
	if v := layout.Percentage(l, style, referTo); !v.Auto {
		return v.V
	}
	return 0
}

func unitsToPx(u int32) float64 { return float64(u) / 1024 }

// drawFirstLine shows the first line of the layout of tb with its baseline
// origin at (x, y). The page is y-down and glyphs y-up, so the text matrix
// flips the vertical axis.
func drawFirstLine(s *stream.Stream, tb *boxes.TextBox, textOverflow props.Keyword, blockEllipsis props.KeywordOrString, x, y float64) {
	lay := tb.Layout
	// Lines with only invisible characters have nothing to show.
	if strings.TrimSpace(lay.Text) == "" {
		return
	}

	style := tb.Style
	lang := string(style.Lang())
	var line *linebreak.Line
	switch {
	case textOverflow == "ellipsis":
		if lay.MaxWidth == nil {
			logging.Warnf("text-overflow: ellipsis needs a width, ignored")
			line = lay.FirstLine()
			break
		}
		lay.EllipsizeEnd(*lay.MaxWidth)
		line = lay.FirstLine()
	case blockEllipsis.Keyword != "none" && blockEllipsis != (props.KeywordOrString{}):
		line = blockEllipsized(lay, tb, blockEllipsis, lang)
	default:
		line = lay.FirstLine()
	}

	s.SetTextMatrix(1, 0, 0, -1, x, y)
	text := lay.Text
	for _, run := range line.Runs {
		showRun(s, run, text)
	}
}

// blockEllipsized wraps the text at its available width and ends it with the
// ellipsis, shortening it a word at a time until what follows the first
// line is gone (CSS Overflow 4 §3.3).
func blockEllipsized(lay *linebreak.Layout, tb *boxes.TextBox, blockEllipsis props.KeywordOrString, lang string) *linebreak.Line {
	if lay.MaxWidth == nil {
		return lay.FirstLine()
	}
	lay.Wrap(*lay.MaxWidth)

	ellipsis := "…"
	if blockEllipsis.Keyword != "auto" {
		ellipsis = blockEllipsis.String
	}

	// A hyphenated word loses its last word along with the hyphen.
	text := lay.Text
	if hyphen := string(tb.Style.HyphenateCharacter()); hyphen != "" && strings.HasSuffix(text, hyphen) {
		text = text[:len(text)-len(hyphen)]
		if end, ok := linebreak.LastWordEnd(text, lang); ok && end > 0 {
			text = text[:end]
		}
	}
	lay.SetText(text + ellipsis)
	line, rest := lay.FirstLineAndRemainder()
	for rest >= 0 {
		end, ok := linebreak.LastWordEnd(strings.TrimSuffix(lay.Text, ellipsis), lang)
		if !ok {
			break
		}
		lay.SetText(lay.Text[:end] + ellipsis)
		line, rest = lay.FirstLineAndRemainder()
	}
	return line
}

// showRun shows the glyphs of one run, in the PDF font of its face. text is
// the text of the layout the clusters of the run point into.
func showRun(s *stream.Stream, run *linebreak.Run, text string) {
	if len(run.Glyphs) == 0 || run.Size < minFontSize {
		return
	}
	font := s.AddFont(run.Face)
	size := run.Size
	s.SetFont(font, size)

	// toTJ converts px to thousandths of the font size.
	toTJ := func(px float64) float64 { return px * 1000 / size }

	var elements []fonts.TJElement
	flush := func() {
		s.ShowGlyphs(elements)
		elements = elements[:0]
	}
	// shift is movement still owed to the next glyph, in px, to the right.
	// Empty glyphs advance without being drawn.
	shift := 0.0
	for i, g := range run.Glyphs {
		advance := g.XAdvancePx()
		if g.Empty {
			shift += advance
			continue
		}

		var cid uint16
		switch {
		case g.Missing() && run.Ellipsis == "":
			r, _ := utf8.DecodeRuneInString(text[min(run.Start+g.Cluster, len(text)):])
			logging.Warnf(".notdef glyph rendered for Unicode string unsupported by fonts: %q (U+%04X)", string(r), r)
			cid = font.MissingCID(r, math.RoundToEven(toTJ(advance)))
		default:
			cid = font.CID(uint16(g.ID))
			if !font.HasText(cid) {
				font.SetText(cid, clusterText(run, i, text))
			}
		}

		offset := toTJ(g.XOffsetPx())
		logical := font.LogicalWidth(cid, size)
		el := fonts.TJElement{
			CID:    cid,
			Before: -(toTJ(shift) + offset),
			Adjust: logical + offset - toTJ(advance),
		}
		shift = 0
		if rise := g.YOffsetPx(); rise != 0 {
			flush()
			s.SetTextRise(rise)
			s.ShowGlyphs([]fonts.TJElement{el})
			s.SetTextRise(0)
			continue
		}
		elements = append(elements, el)
	}
	flush()
}

// clusterText returns the characters glyph i of run stands for, which are
// shown when text is copied from the PDF. A glyph that shares its cluster
// with others, as ligature parts do, gets the text only when it is the
// first of the cluster in reading order.
func clusterText(run *linebreak.Run, i int, text string) string {
	if run.Ellipsis != "" {
		return run.Ellipsis
	}
	glyphs := run.Glyphs
	cluster := glyphs[i].Cluster
	rtl := run.RTL()
	// The glyph that comes before in reading order.
	if prev := i - 1; !rtl && prev >= 0 && glyphs[prev].Cluster == cluster {
		return ""
	}
	if next := i + 1; rtl && next < len(glyphs) && glyphs[next].Cluster == cluster {
		return ""
	}
	// The end of the cluster is the nearest other cluster after it in
	// reading order.
	end := run.End - run.Start
	for j := range glyphs {
		c := glyphs[j].Cluster
		if c > cluster && c < end {
			end = c
		}
	}
	lo := min(run.Start+cluster, len(text))
	hi := min(run.Start+end, len(text))
	return text[lo:hi]
}
