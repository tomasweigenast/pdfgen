package linebreak

import (
	"unicode"

	"github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// item is a piece of a paragraph shaped as a whole: one face, one bidi
// level, one script and one letter-spacing. Indices are in characters.
type item struct {
	start, end int
	face       *fonts.Face
	// fontPosition ranks the face: 0 for the face of the text, 1 for a
	// fallback face.
	fontPosition int
	level        bidi.Level
	script       language.Script
	spacing      int
	// needHyphen appends a hyphen when shaping, for a line broken after
	// the item.
	needHyphen bool
}

func (it *item) numChars() int { return it.end - it.start }
func (it *item) rtl() bool     { return it.level%2 == 1 }

// split cuts the first n characters off it and returns them as a new item.
func (it *item) split(n int) *item {
	head := *it
	head.end = it.start + n
	it.start += n
	return &head
}

// considerAsSpace tells the characters that take the face of the text
// around them instead of selecting one.
func considerAsSpace(r rune) bool {
	switch {
	case unicode.Is(unicode.Cc, r), unicode.Is(unicode.Cs, r),
		unicode.Is(unicode.Zl, r), unicode.Is(unicode.Zp, r):
		return true
	case unicode.Is(unicode.Cf, r):
		return !((r >= 0x600 && r <= 0x6ff) || r == 0x70f || r == 0x8e2)
	case unicode.Is(unicode.Zs, r):
		return r != 0x1680
	}
	return (r >= 0xfe00 && r <= 0xfe0f) || (r >= 0xe0100 && r <= 0xe01ef)
}

// bracketPairs maps opening brackets to their closing ones. A closing
// bracket takes the script of its opening bracket.
var bracketPairs = map[rune]rune{
	'(': ')', '[': ']', '{': '}', 0xab: 0xbb, 0x2018: 0x2019, 0x201c: 0x201d,
	0x2039: 0x203a, 0x2045: 0x2046, 0x207d: 0x207e, 0x208d: 0x208e,
	0x3008: 0x3009, 0x300a: 0x300b, 0x300c: 0x300d, 0x300e: 0x300f,
	0x3010: 0x3011, 0x3014: 0x3015, 0x3016: 0x3017, 0x3018: 0x3019,
	0x301a: 0x301b, 0xff08: 0xff09, 0xff3b: 0xff3d, 0xff5b: 0xff5d,
	0xff5f: 0xff60, 0xff62: 0xff63,
}

var closingBrackets = func() map[rune]rune {
	m := make(map[rune]rune, len(bracketPairs))
	for open, close := range bracketPairs {
		m[close] = open
	}
	return m
}()

func isRealScript(s language.Script) bool {
	return s != language.Common && s != language.Inherited
}

// scriptRuns resolves the script of every character (UAX #24): Common and
// Inherited characters take the script of the run they are in, the first
// real script of a run applies to its beginning, and closing brackets take
// the script of the matching opening bracket.
func scriptRuns(text []rune) []language.Script {
	out := make([]language.Script, len(text))
	type paren struct {
		open   rune
		script language.Script
	}
	var stack []paren
	runStart := 0
	current := language.Common
	for i, r := range text {
		sc := language.LookupScript(r)
		if _, ok := bracketPairs[r]; ok {
			stack = append(stack, paren{r, current})
		} else if open, ok := closingBrackets[r]; ok {
			for len(stack) > 0 && stack[len(stack)-1].open != open {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				sc = stack[len(stack)-1].script
				stack = stack[:len(stack)-1]
			}
		}
		switch {
		case !isRealScript(sc) || sc == current:
		case !isRealScript(current):
			current = sc
			for j := runStart; j < i; j++ {
				out[j] = current
			}
			for k := range stack {
				if !isRealScript(stack[k].script) {
					stack[k].script = current
				}
			}
		default:
			runStart = i
			current = sc
		}
		out[i] = current
	}
	return out
}

// itemize splits the paragraph text[start:end] into items.
func (l *Layout) itemize(start, end int) []*item {
	if start == end {
		return nil
	}
	para := l.runes[start:end]
	var p bidi.Paragraph
	dir := bidi.LeftToRight
	if l.Style.rtl() {
		dir = bidi.RightToLeft
	}
	levels := make([]bidi.Level, len(para))
	runs := p.Segment(para, dir)
	for i := 0; i < runs.NumRuns(); i++ {
		run := runs.Run(i)
		for j := run.Start; j < run.End; j++ {
			levels[j] = run.Level
		}
	}
	scripts := scriptRuns(para)

	desc := l.description()
	lang := l.Style.shapingLanguage()
	spaceFace := l.ctx.Fonts.Query(desc, lang, ' ')

	var items []*item
	var cur *item
	firstSpace := -1
	lastWasForcedBreak := false
	for i := range para {
		r := para[i]
		pos := start + i
		newRun := i == 0 || levels[i] != levels[i-1] || scripts[i] != scripts[i-1] ||
			l.spacing[pos] != l.spacing[pos-1]
		if newRun {
			if cur != nil && cur.face == nil {
				fillFace(items, spaceFace, 0)
			}
			cur = nil
			firstSpace = -1
		}
		forced := r == '\t' || r == '\u2028'
		forceBreak := forced || lastWasForcedBreak
		lastWasForcedBreak = forced

		var face *fonts.Face
		position := 0xffff
		isSpace := considerAsSpace(r)
		if !isSpace {
			face = l.ctx.Fonts.Query(desc, lang, r)
			position = 1
			if face == spaceFace {
				position = 0
			}
		}
		prevFirstSpace := firstSpace
		if isSpace {
			if firstSpace < 0 {
				firstSpace = pos
			}
		} else {
			firstSpace = -1
		}

		if cur != nil {
			if cur.face == nil && face != nil {
				fillFace(items, face, position)
			} else if cur.face != nil && face == nil {
				face, position = cur.face, cur.fontPosition
			}
			if !forceBreak && cur.face == face {
				cur.end++
				continue
			}
			// A fallback item ending with spaces gives them to the next
			// item when its face is less of a fallback.
			startAt := pos
			if prevFirstSpace > cur.start && position < cur.fontPosition {
				startAt = prevFirstSpace
			}
			cur.end = startAt
			cur = &item{start: startAt, end: pos + 1, face: face, fontPosition: position,
				level: levels[i], script: scripts[i], spacing: l.spacing[pos]}
			items = append(items, cur)
			continue
		}
		cur = &item{start: pos, end: pos + 1, face: face, fontPosition: position,
			level: levels[i], script: scripts[i], spacing: l.spacing[pos]}
		items = append(items, cur)
	}
	if cur != nil && cur.face == nil {
		fillFace(items, spaceFace, 0)
	}
	return items
}

// fillFace gives a face to the items at the end of items that have none:
// items holding only spaces, tabs or controls take the face of what
// follows them.
func fillFace(items []*item, face *fonts.Face, position int) {
	for i := len(items) - 1; i >= 0 && items[i].face == nil; i-- {
		items[i].face, items[i].fontPosition = face, position
	}
}

func (l *Layout) description() fonts.Description {
	d := l.Style.Font
	d.Size = fonts.FromUnits(fonts.ToUnits(d.Size))
	return d
}
