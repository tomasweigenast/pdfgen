package linebreak

import (
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/shape"
)

// Glyph is a positioned glyph of a line.
type Glyph struct {
	shape.Glyph
	// Empty glyphs are not drawn: tabs, invisible characters and spaces
	// collapsed at the end of a wrapped line.
	Empty bool
}

// Run is a sequence of glyphs set with one face, in one direction.
type Run struct {
	Face *fonts.Face
	// Size is the font size in px, on the 1/1024 px grid.
	Size float64
	// Start and End are byte offsets of the characters of the run in the
	// text of the layout. Glyph clusters are relative to Start.
	Start, End int
	Level      bidi.Level
	Script     language.Script
	// Glyphs are in visual order. Advances include letter-spacing,
	// word-spacing and tab stops.
	Glyphs []Glyph

	item    *item
	tab     bool
	spacing int
}

// RTL reports whether the run is set from right to left.
func (r *Run) RTL() bool { return r.Level%2 == 1 }

// Width returns the advance of the run in 1/1024 px.
func (r *Run) Width() int {
	w := 0
	for _, g := range r.Glyphs {
		w += int(g.XAdvance)
	}
	return w
}

// Line is a line of a layout.
type Line struct {
	// Start and Length are the byte range of the line in the text of the
	// layout. A paragraph separator ending the line is not included.
	Start, Length int
	// Runs are in visual order.
	Runs []*Run
	// RTL is the resolved direction of the paragraph of the line.
	RTL bool

	layout         *Layout
	paragraphStart bool
}

// Width returns the logical width of the line in 1/1024 px.
func (l *Line) Width() int {
	w := 0
	for _, r := range l.Runs {
		w += r.Width()
	}
	return w
}

// extents returns the logical top of the line, measured upwards as a
// negative value from the baseline, and its height, in 1/1024 px.
func (l *Line) extents() (y, height int) {
	if len(l.Runs) == 0 {
		face := l.layout.primaryFace()
		if face == nil {
			return 0, 0
		}
		m := face.Metrics(l.layout.Style.Font.Size)
		return -int(m.Ascent), int(m.Ascent + m.Descent)
	}
	top, bottom := 0, 0
	for _, r := range l.Runs {
		ry, rh := r.extents()
		top = min(top, ry)
		bottom = max(bottom, ry+rh)
	}
	return top, bottom - top
}

func (r *Run) extents() (y, height int) {
	// Every glyph, empty ones included, spans the ascent and descent of
	// the font.
	if len(r.Glyphs) == 0 {
		return 0, 0
	}
	m := r.Face.Metrics(r.Size)
	return -int(m.Ascent), int(m.Ascent + m.Descent)
}

type wrapMode uint8

const (
	wrapWord wrapMode = iota
	wrapChar
)

// Layout holds text set in one style and broken into lines. Only the first
// line and the start of the second one are computed: that is all line
// breaking needs.
type Layout struct {
	// Text is the laid-out text.
	Text  string
	Style *Style
	// MaxWidth is the available width the layout was created for, nil
	// when unconstrained.
	MaxWidth             *float64
	JustificationSpacing float64

	ctx   *Context
	width int // in 1/1024 px, negative when lines do not wrap
	wrap  wrapMode

	runes   []rune
	offsets []int // byte offset of each character, and the length
	// spacing is the letter-spacing of each character, word-spacing
	// included for separators, in 1/1024 px.
	spacing       []int
	insertHyphens bool
	tabStop       int // distance between tab stops in 1/1024 px

	attrs []logAttr
	lines []*Line
	built bool
}

// newLayout returns an empty layout. Lines do not wrap until a width is
// set.
func newLayout(ctx *Context, style *Style, justificationSpacing float64, maxWidth *float64) *Layout {
	return &Layout{
		Style:                style,
		MaxWidth:             maxWidth,
		JustificationSpacing: justificationSpacing,
		ctx:                  ctx,
		width:                -1,
	}
}

// setWidth sets the width lines wrap at, in 1/1024 px. A negative width
// turns wrapping off.
func (l *Layout) setWidth(width int) {
	if width < 0 {
		width = -1
	}
	if width != l.width {
		l.width = width
		l.lines, l.built = nil, false
	}
}

func (l *Layout) setWrap(mode wrapMode) {
	if mode != l.wrap {
		l.wrap = mode
		l.lines, l.built = nil, false
	}
}

// SetText replaces the text of the layout. Only the first line break and
// the character after it are kept: that is enough to find where the first
// line ends.
func (l *Layout) SetText(text string) { l.setText(text, false) }

// setText sets the text. With justify, the justification spacing of the
// layout is added to word-spacing, as when drawing.
func (l *Layout) setText(text string, justify bool) {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		end := i + 1
		if end < len(text) {
			_, size := utf8.DecodeRuneInString(text[end:])
			end += size
		}
		text = text[:end]
	}

	wordSpacing := l.Style.WordSpacing
	if justify {
		wordSpacing += l.JustificationSpacing
	}
	letterSpacing := 0
	if l.Style.LetterSpacing != 0 {
		letterSpacing = int(l.Style.LetterSpacing * fonts.UnitsPerPx)
	}
	wordBreaking := l.Style.overflowWrapBreaksWords()

	// A single space cannot get word-spacing, so a zero-width space is
	// added after it. Letter-spacing only covers the text as given.
	letterSpaced := utf8.RuneCountInString(text)
	if text == " " && wordSpacing != 0 {
		text = " \u200b"
	}
	l.Text = text
	l.runes = []rune(text)
	l.offsets = make([]int, len(l.runes)+1)
	for i, r := range l.runes {
		l.offsets[i+1] = l.offsets[i] + utf8.RuneLen(r)
	}

	l.spacing = make([]int, len(l.runes)+1)
	if text != "" && (wordSpacing != 0 || l.Style.LetterSpacing != 0) {
		for i := range letterSpaced {
			l.spacing[i] = letterSpacing
		}
		if wordSpacing != 0 {
			spaceSpacing := int(wordSpacing*fonts.UnitsPerPx + float64(letterSpacing))
			last := len(text) - 1
			for i, r := range l.runes {
				if r != ' ' && r != '\u00a0' {
					continue
				}
				// Separators at the ends of the text lose half of their
				// spacing, so they get twice as much.
				factor := 1
				if l.offsets[i] == 0 || l.offsets[i] == last {
					factor = 2
				}
				l.spacing[i] = factor * spaceSpacing
			}
		}
	}
	l.insertHyphens = !(text != "" && wordBreaking)

	l.tabStop = 0
	if strings.ContainsRune(text, '\t') {
		l.tabStop = l.tabWidth()
	}
	l.attrs = nil
	l.lines, l.built = nil, false
}

// tabWidth returns the distance between tab stops in 1/1024 px.
func (l *Layout) tabWidth() int {
	var width int
	if ts := l.Style.tabSize(); ts.IsLength {
		width = int(ts.Length)
	} else {
		spaces := newLayout(l.ctx, l.Style, l.JustificationSpacing, nil)
		spaces.SetText(strings.Repeat(" ", ts.Spaces))
		w, _ := LineSize(spaces.FirstLine(), l.Style)
		width = int(math.RoundToEven(w))
	}
	// A zero distance would make every tab stop the same.
	if width == 0 {
		width = 1
	}
	return width * fonts.UnitsPerPx
}

func (l *Layout) primaryFace() *fonts.Face {
	return l.ctx.Fonts.Query(l.description(), l.Style.shapingLanguage(), ' ')
}

// FirstLine returns the first line of the layout.
func (l *Layout) FirstLine() *Line {
	l.build()
	return l.lines[0]
}

// firstLine returns the first line and the byte offset where the second
// line starts, or -1 when there is no second line.
func (l *Layout) firstLine() (*Line, int) {
	l.build()
	if len(l.lines) > 1 {
		return l.lines[0], l.lines[1].Start
	}
	return l.lines[0], -1
}

// logAttrs returns the break attributes of every position of the text.
func (l *Layout) logAttrs() []logAttr {
	l.build()
	return l.attrs
}

// paragraphBoundary returns the end of the paragraph starting at start,
// and the start of the next one. Paragraphs end at line feeds, carriage
// returns, CR LF pairs and paragraph separators.
func paragraphBoundary(text []rune, start int) (end, next int) {
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '\n', paragraphSeparator:
			return i, i + 1
		case '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				return i, i + 2
			}
			return i, i + 1
		}
	}
	return len(text), len(text)
}

func (l *Layout) build() {
	if l.built {
		return
	}
	l.built = true
	l.lines = nil
	needAttrs := l.attrs == nil
	if needAttrs {
		l.attrs = make([]logAttr, len(l.runes)+1)
	}
	type paragraph struct {
		start, end int
		items      []*item
	}
	var paragraphs []paragraph
	for start := 0; ; {
		end, next := paragraphBoundary(l.runes, start)
		items := l.itemize(start, end)
		if needAttrs {
			l.paragraphAttrs(start, next)
		}
		paragraphs = append(paragraphs, paragraph{start, end, items})
		if end == len(l.runes) {
			break
		}
		start = next
	}
	for _, p := range paragraphs {
		l.breakParagraph(p.start, p.items)
		if len(l.lines) >= 2 {
			break
		}
	}
}

// paragraphAttrs computes the break attributes of text[start:end], which
// includes the paragraph separator.
func (l *Layout) paragraphAttrs(start, end int) {
	attrs := computeLogAttrs(l.runes[start:end])
	tailorCatalan(l.runes[start:end], attrs, l.Style.shapingLanguage())
	if !l.insertHyphens {
		disableHyphenInsertion(attrs)
	}
	// The end of the previous paragraph stays a break.
	before := l.attrs[start]
	attrs[0].lineBreak = attrs[0].lineBreak || before.lineBreak
	attrs[0].mandatoryBreak = attrs[0].mandatoryBreak || before.mandatoryBreak
	attrs[0].cursor = attrs[0].cursor || before.cursor
	copy(l.attrs[start:], attrs)
}

// runeIndex returns the index of the character at a byte offset.
func (l *Layout) runeIndex(offset int) int {
	return sort.SearchInts(l.offsets, offset)
}

// shapeItem shapes an item and applies its letter-spacing.
func (l *Layout) shapeItem(it *item) []Glyph {
	if l.runes[it.start] == '\t' {
		return nil
	}
	size := fonts.FromUnits(fonts.ToUnits(l.Style.Font.Size))
	var lang language.Language
	if s := l.Style.shapingLanguage(); s != "" {
		lang = language.NewLanguage(s)
	}
	shaped := l.ctx.Shaper.Shape(shape.Run{
		Text: l.runes, Start: it.start, End: it.end,
		Face: it.face, Size: size, Features: l.Style.features(),
		Lang: lang, Script: it.script, RTL: it.rtl(), Hyphen: it.needHyphen,
	})
	glyphs := make([]Glyph, len(shaped))
	base := l.offsets[it.start]
	for i, g := range shaped {
		glyphs[i] = Glyph{Glyph: g}
		if g.XAdvance == 0 && isDefaultIgnorable(l.runes[l.runeIndex(base+g.Cluster)]) {
			glyphs[i].Empty = true
		}
	}
	if it.spacing != 0 && len(glyphs) > 0 {
		l.letterSpace(it, glyphs)
		left, right := distributeSpacing(it.spacing)
		glyphs[0].XAdvance += int32(left)
		glyphs[0].XOffset += int32(left)
		glyphs[len(glyphs)-1].XAdvance += int32(right)
	}
	return glyphs
}

// isDefaultIgnorable approximates Default_Ignorable_Code_Point: characters
// that are invisible unless a font explicitly draws them.
func isDefaultIgnorable(r rune) bool {
	if unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r) || unicode.Is(unicode.Variation_Selector, r) {
		return true
	}
	if unicode.Is(unicode.Cf, r) {
		return !unicode.Is(unicode.Prepended_Concatenation_Mark, r) &&
			!(r >= 0xfff9 && r <= 0xfffb) && !(r >= 0x13430 && r <= 0x1343f)
	}
	return r == 0x115f || r == 0x1160 || r == 0x3164 || r == 0xffa0
}

// distributeSpacing splits a letter-spacing between the two sides of a
// character. Whole pixels are kept whole on the left side, as hinted
// glyph positions would be.
func distributeSpacing(spacing int) (left, right int) {
	left = spacing / 2
	if spacing&(fonts.UnitsPerPx-1) == 0 {
		left = (left + fonts.UnitsPerPx/2) &^ (fonts.UnitsPerPx - 1)
	}
	return left, spacing - left
}

type cluster struct {
	first, last int // glyph indices, first <= last
	startChar   int // character index in the item
	endChar     int
}

// clusters returns the clusters of glyphs in logical order.
func (l *Layout) clusters(it *item, glyphs []Glyph) []cluster {
	base := l.offsets[it.start]
	var out []cluster
	for i := 0; i < len(glyphs); {
		j := i + 1
		for j < len(glyphs) && glyphs[j].Cluster == glyphs[i].Cluster {
			j++
		}
		c := cluster{first: i, last: j - 1, startChar: l.runeIndex(base+glyphs[i].Cluster) - it.start}
		out = append(out, c)
		i = j
	}
	if it.rtl() {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	for i := range out {
		if i+1 < len(out) {
			out[i].endChar = out[i+1].startChar
		} else {
			out[i].endChar = it.numChars()
		}
	}
	return out
}

// letterSpace adds the letter-spacing of an item between its grapheme
// clusters.
func (l *Layout) letterSpace(it *item, glyphs []Glyph) {
	left, right := distributeSpacing(it.spacing)
	n := it.numChars()
	for _, c := range l.clusters(it, glyphs) {
		if !l.attrs[it.start+c.startChar].cursor {
			// Marks shaped apart from their base are moved with it.
			if it.rtl() && glyphs[c.last].XAdvance == 0 {
				glyphs[c.last].XOffset += int32(left)
			} else if !it.rtl() && glyphs[c.first].XAdvance == 0 {
				glyphs[c.first].XOffset -= int32(right)
			}
			continue
		}
		if !it.rtl() {
			if c.startChar > 0 {
				glyphs[c.first].XAdvance += int32(left)
				glyphs[c.first].XOffset += int32(left)
			}
			if c.endChar < n {
				glyphs[c.last].XAdvance += int32(right)
			}
			continue
		}
		if c.startChar > 0 {
			glyphs[c.last].XAdvance += int32(right)
		}
		if c.endChar < n {
			glyphs[c.first].XOffset += int32(left)
			glyphs[c.first].XAdvance += int32(left)
		}
	}
}

// logicalWidths returns the width of each character of an item, the width
// of a cluster being shared by its characters.
func (l *Layout) logicalWidths(it *item, glyphs []Glyph) []int {
	widths := make([]int, it.numChars())
	if it.end > it.start && l.runes[it.start] == '\t' {
		for _, g := range glyphs {
			widths[0] += int(g.XAdvance)
		}
		return widths
	}
	for _, c := range l.clusters(it, glyphs) {
		w := 0
		for i := c.first; i <= c.last; i++ {
			w += int(glyphs[i].XAdvance)
		}
		n := c.endChar - c.startChar
		if n <= 0 {
			continue
		}
		each := w / n
		for i := c.startChar; i < c.endChar; i++ {
			widths[i] = each
		}
		widths[c.startChar] += w - each*n
	}
	return widths
}

func glyphsWidth(glyphs []Glyph) int {
	w := 0
	for _, g := range glyphs {
		w += int(g.XAdvance)
	}
	return w
}
