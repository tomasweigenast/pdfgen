package linebreak

import (
	"unicode"

	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/segmenter"
)

// logAttr describes the position before a character: index i is between
// characters i-1 and i, and the last index is the end of the text.
type logAttr struct {
	lineBreak      bool // a line may be broken here (UAX #14)
	mandatoryBreak bool
	charBreak      bool // a line may be broken here when wrapping characters
	cursor         bool // grapheme cluster boundary (UAX #29)
	white          bool // the character after the position is white space
	wordStart      bool
	wordEnd        bool
	wordBoundary   bool // UAX #29 word boundary

	// insertsHyphen means that breaking here shows a hyphen at the end of
	// the line, and removesPreceding that the hyphen replaces the character
	// before the position.
	insertsHyphen    bool
	removesPreceding bool
}

const paragraphSeparator = '\u2029'

// computeLogAttrs returns the attributes of the len(text)+1 positions of a
// paragraph. The end of the text is treated as followed by a paragraph
// separator.
func computeLogAttrs(text []rune) []logAttr {
	attrs := make([]logAttr, len(text)+1)
	if len(text) == 0 {
		attrs[0] = logAttr{lineBreak: true, mandatoryBreak: true, charBreak: true, cursor: true, wordBoundary: true}
		return attrs
	}

	var seg segmenter.Segmenter
	seg.Init(text)
	lines := seg.LineIterator()
	for lines.Next() {
		l := lines.Line()
		end := l.Offset + len(l.Text)
		attrs[end].lineBreak = true
		attrs[end].mandatoryBreak = l.IsMandatoryBreak
	}
	graphemes := seg.GraphemeIterator()
	for graphemes.Next() {
		g := graphemes.Grapheme()
		attrs[g.Offset].cursor = true
		attrs[g.Offset+len(g.Text)].cursor = true
	}

	for i := range attrs {
		attrs[i].charBreak = attrs[i].cursor
		if i < len(text) {
			attrs[i].white = isWhite(text[i])
		}
	}
	initial := wordBoundaries(text, attrs)
	wordStartsAndEnds(text, attrs)
	hyphenInsertions(text, attrs, initial)

	// UAX #14 LB2 and LB3, UAX #29 GB1, GB2, WB1 and WB2.
	n := len(text)
	attrs[0].lineBreak = false
	attrs[n].lineBreak = true
	attrs[n].mandatoryBreak = true
	attrs[0].cursor, attrs[n].cursor = true, true
	attrs[0].charBreak, attrs[n].charBreak = true, true
	attrs[0].wordBoundary, attrs[n].wordBoundary = true, true
	return attrs
}

func isWhite(r rune) bool {
	switch r {
	case '\t', '\n', '\r', '\f':
		return true
	}
	return unicode.In(r, unicode.Zs, unicode.Zl, unicode.Zp)
}

type wordBreakClass uint8

const (
	wbOther wordBreakClass = iota
	wbNewline
	wbExtendFormat
	wbKatakana
	wbHebrewLetter
	wbALetter
	wbMidNumLet
	wbMidLetter
	wbMidNum
	wbNumeric
	wbExtendNumLet
	wbRIOdd
	wbRIEven
	wbWSegSpace
)

// isNumericLineBreak approximates the Line_Break=NU class: decimal digits
// except the full-width ones, which are ideographic.
func isNumericLineBreak(r rune) bool {
	return unicode.Is(unicode.Nd, r) && !(r >= 0xff10 && r <= 0xff19)
}

// isInfixSeparator lists the characters of Line_Break=IS.
func isInfixSeparator(r rune) bool {
	switch r {
	case ',', '.', ':', ';', 0x037e, 0x0589, 0x060c, 0x060d, 0x07f8, 0x2044,
		0xfe10, 0xfe13, 0xfe14:
		return true
	}
	return false
}

func isIdeographicException(r rune) bool {
	return r == 0x3006 || r == 0x3007 ||
		(r >= 0x3021 && r <= 0x3029) ||
		(r >= 0x3038 && r <= 0x303a) ||
		(r >= 0x3400 && r <= 0x4db5) ||
		(r >= 0x4e00 && r <= 0x9fc3) ||
		(r >= 0xf900 && r <= 0xfa2d) ||
		(r >= 0xfa30 && r <= 0xfa6a) ||
		(r >= 0xfa70 && r <= 0xfad9) ||
		(r >= 0x20000 && r <= 0x2a6d6) ||
		(r >= 0x2f800 && r <= 0x2fa1d)
}

func wordBreakClassOf(r rune, prev wordBreakClass) wordBreakClass {
	script := language.LookupScript(r)
	if script == language.Katakana {
		return wbKatakana
	}
	if script == language.Hebrew && unicode.Is(unicode.Lo, r) {
		return wbHebrewLetter
	}
	switch {
	case r == 0x00b8 || r == 0x058a || r == 0x070f:
		return wbALetter
	case r == 0x3031 || r == 0x3032 || r == 0x3033 || r == 0x3034 || r == 0x3035 ||
		r == 0x309b || r == 0x309c || r == 0x30a0 || r == 0x30fc || r == 0xff70:
		return wbKatakana
	case r == 0xff9e || r == 0xff9f:
		return wbExtendFormat
	}
	if isNumericLineBreak(r) && r != 0x066c {
		return wbNumeric
	}
	if isInfixSeparator(r) && r != ':' && r != 0xfe13 && r != '.' {
		return wbMidNum
	}
	switch {
	case unicode.Is(unicode.Cc, r):
		if r == '\r' || r == '\n' || r == '\v' || r == '\f' || r == 0x85 {
			return wbNewline
		}
	case unicode.In(r, unicode.Zl, unicode.Zp):
		return wbNewline
	case unicode.In(r, unicode.Cf, unicode.Mc, unicode.Me, unicode.Mn):
		return wbExtendFormat
	case unicode.Is(unicode.Pc, r):
		return wbExtendNumLet
	case unicode.In(r, unicode.Pi, unicode.Pf):
		if r == 0x2018 || r == 0x2019 {
			return wbMidNumLet
		}
	case unicode.Is(unicode.Po, r):
		switch {
		case (r >= 0x055a && r <= 0x055c) || r == 0x055e || r == 0x05f3:
			return wbALetter
		case r == 0x0027 || r == 0x002e || r == 0x2024 || r == 0xfe52 || r == 0xff07 || r == 0xff0e:
			return wbMidNumLet
		case r == 0x00b7 || r == 0x05f4 || r == 0x2027 || r == 0x003a || r == 0x0387 ||
			r == 0x055f || r == 0xfe13 || r == 0xfe55 || r == 0xff1a:
			return wbMidLetter
		case r == 0x066c || r == 0xfe50 || r == 0xfe54 || r == 0xff0c || r == 0xff1b:
			return wbMidNum
		}
	case unicode.Is(unicode.So, r):
		if r >= 0x24b6 && r <= 0x24e9 {
			return alphabetic(r, script)
		}
		if r >= 0x1f1e6 && r <= 0x1f1ff {
			if prev == wbRIOdd {
				return wbRIEven
			}
			return wbRIOdd
		}
	case unicode.In(r, unicode.Lo, unicode.Nl):
		if isIdeographicException(r) {
			return wbOther
		}
		return alphabetic(r, script)
	case unicode.In(r, unicode.Ll, unicode.Lm, unicode.Lt, unicode.Lu):
		return alphabetic(r, script)
	}
	if unicode.Is(unicode.Zs, r) && r != 0x00a0 && r != 0x2007 && r != 0x202f {
		return wbWSegSpace
	}
	return wbOther
}

// alphabetic classifies a letter, leaving out the scripts that are broken
// with a dictionary (Line_Break=SA) and Hiragana.
func alphabetic(r rune, script language.Script) wordBreakClass {
	if isComplexContext(script) || script == language.Hiragana {
		return wbOther
	}
	return wbALetter
}

func isComplexContext(s language.Script) bool {
	switch s {
	case language.Thai, language.Lao, language.Myanmar, language.Khmer,
		language.Tai_Tham, language.Tai_Viet, language.New_Tai_Lue, language.Tai_Le:
		return true
	}
	return false
}

func isRegionalIndicator(r rune) bool { return r >= 0x1f1e6 && r <= 0x1f1ff }

// wordBoundaries marks the word boundaries of UAX #29. The rules only look
// at grapheme cluster boundaries (WB3, WB4).
//
// WB6 and WB12 remove a boundary once the character after it is known. The
// returned slice holds the boundaries as they were before that, which is
// what decides where hyphens are shown.
func wordBoundaries(text []rune, attrs []logAttr) (initial []bool) {
	initial = make([]bool, len(attrs))
	prevPrev, prev := wbOther, wbOther
	prevIndex := -1
	var prevRune rune
	for i := 0; i <= len(text); i++ {
		r := rune(paragraphSeparator)
		if i < len(text) {
			r = text[i]
		}
		if i > 0 {
			prevRune = text[i-1]
		}
		if !attrs[i].cursor && !isRegionalIndicator(r) {
			continue
		}
		class := wordBreakClassOf(r, prev)
		isLetter := func(c wordBreakClass) bool {
			return c == wbALetter || c == wbHebrewLetter || c == wbNumeric
		}
		boundary := true
		switch {
		case prev == wbNewline && prevIndex+1 == i:
			boundary = true // WB3a
		case class == wbNewline:
			boundary = true // WB3b
		case prevRune == 0x200d && isExtendedPictographic(r):
			boundary = false // WB3c
		case prev == wbWSegSpace && class == wbWSegSpace && prevIndex+1 == i:
			boundary = false // WB3d
		case class == wbExtendFormat:
			boundary = false // WB4
		case isLetter(prev) && isLetter(class):
			boundary = false // WB5, WB8, WB9, WB10
		case prev == wbKatakana && class == wbKatakana:
			boundary = false // WB13
		case (isLetter(prev) || prev == wbKatakana || prev == wbExtendNumLet) && class == wbExtendNumLet:
			boundary = false // WB13a
		case prev == wbExtendNumLet && (isLetter(class) || class == wbKatakana):
			boundary = false // WB13b
		case (prevPrev == wbALetter || prevPrev == wbHebrewLetter) &&
			(class == wbALetter || class == wbHebrewLetter) &&
			(prev == wbMidLetter || prev == wbMidNumLet || prevRune == '\''):
			attrs[prevIndex].wordBoundary = false // WB6
			boundary = false                      // WB7
		case prev == wbHebrewLetter && r == '\'':
			boundary = false // WB7a
		case prevPrev == wbHebrewLetter && prevRune == '"' && class == wbHebrewLetter:
			attrs[prevIndex].wordBoundary = false // WB7b
			boundary = false                      // WB7c
		case prevPrev == wbNumeric && class == wbNumeric &&
			(prev == wbMidNum || prev == wbMidNumLet || prevRune == '\''):
			boundary = false                      // WB11
			attrs[prevIndex].wordBoundary = false // WB12
		case prev == wbRIOdd && class == wbRIEven:
			boundary = false // WB15, WB16
		}
		attrs[i].wordBoundary = boundary
		initial[i] = boundary
		if class != wbExtendFormat {
			prevPrev, prev = prev, class
			prevIndex = i
		}
	}
	return initial
}

func isExtendedPictographic(r rune) bool {
	// Approximation of the Extended_Pictographic property: the emoji and
	// pictograph blocks.
	return (r >= 0x1f000 && r <= 0x1faff) || (r >= 0x2600 && r <= 0x27bf) ||
		r == 0x00a9 || r == 0x00ae || r == 0x203c || r == 0x2049 || r == 0x2122 ||
		(r >= 0x2190 && r <= 0x21ff) || (r >= 0x2300 && r <= 0x23ff) ||
		(r >= 0x2b00 && r <= 0x2bff) || r == 0x3030 || r == 0x303d || r == 0x3297 || r == 0x3299
}

func isJapanese(r rune) bool { return r >= 0x2f00 && r <= 0x30ff }
func isKanji(r rune) bool    { return r >= 0x2f00 && r <= 0x2fdf }
func isHiragana(r rune) bool { return r >= 0x3040 && r <= 0x309f }
func isKatakana(r rune) bool { return r >= 0x30a0 && r <= 0x30ff }

// wordStartsAndEnds marks words as runs of letters or of numbers, ignoring
// marks and format characters inside them. Japanese words also end where
// the kind of kana or kanji changes.
func wordStartsAndEnds(text []rune, attrs []logAttr) {
	const (
		none = iota
		letters
		numbers
	)
	current := none
	var lastLetter rune
	for i := 0; i <= len(text); i++ {
		r := rune(paragraphSeparator)
		if i < len(text) {
			r = text[i]
		}
		isLetter := unicode.In(r, unicode.Ll, unicode.Lm, unicode.Lo, unicode.Lt, unicode.Lu)
		isNumber := unicode.In(r, unicode.Nd, unicode.Nl, unicode.No)
		if current != none {
			switch {
			case unicode.In(r, unicode.Mc, unicode.Me, unicode.Mn, unicode.Cf):
			case isLetter:
				if current == letters && (isJapanese(lastLetter) || isJapanese(r)) {
					if (isHiragana(lastLetter) && !isHiragana(r)) ||
						(isKatakana(lastLetter) && !(isKatakana(r) || isHiragana(r))) ||
						(isKanji(lastLetter) && !(isHiragana(r) || isKanji(r))) ||
						(isJapanese(lastLetter) && !isJapanese(r)) ||
						(!isJapanese(lastLetter) && isJapanese(r)) {
						attrs[i].wordStart = true
						attrs[i].wordEnd = true
					}
				}
				lastLetter = r
			case isNumber:
				lastLetter = r
			default:
				attrs[i].wordEnd = true
				current = none
			}
			continue
		}
		switch {
		case isLetter:
			current = letters
			lastLetter = r
			attrs[i].wordStart = true
		case isNumber:
			current = numbers
			lastLetter = r
			attrs[i].wordStart = true
		}
	}
}

func isHyphenCharacter(r rune) bool {
	switch r {
	case '-', 0x058a, 0x1400, 0x1806, 0x2010, 0x2e17, 0x2e40, 0x30a0, 0xfe63, 0xff0d:
		return true
	}
	return false
}

// hyphenInsertions marks the positions where breaking a line shows a
// hyphen: inside words of scripts that hyphenate, and after soft hyphens.
func hyphenInsertions(text []rune, attrs []logAttr, boundary []bool) {
	prevSpaceOrHyphen := false
	var prevRune rune
	prevScript := language.Common
	for i := 0; i <= len(text); i++ {
		r := rune(paragraphSeparator)
		if i < len(text) {
			r = text[i]
		}
		var insert bool
		switch prevScript {
		case language.Common:
			insert = prevRune == '\u00ad'
		case language.Han, language.Hangul, language.Hiragana, language.Katakana:
			insert = false
		default:
			insert = true
		}
		spaceOrHyphen := isWhite(r) || isHyphenCharacter(r)
		a := &attrs[i]
		a.insertsHyphen = !boundary[i] && !prevSpaceOrHyphen && !spaceOrHyphen && insert
		if prevRune == 0x2027 {
			a.insertsHyphen = true
			a.removesPreceding = true
		}
		prevSpaceOrHyphen = spaceOrHyphen
		prevRune = r
		prevScript = language.LookupScript(r)
	}
}

// disableHyphenInsertion removes the hyphens shown at breaks, as when
// overflow-wrap breaks words.
func disableHyphenInsertion(attrs []logAttr) {
	for i := 1; i < len(attrs); i++ {
		if !attrs[i].removesPreceding {
			attrs[i].insertsHyphen = false
		}
	}
}
