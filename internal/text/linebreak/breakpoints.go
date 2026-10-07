package linebreak

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// textLogAttrs returns the break attributes of a standalone text.
func textLogAttrs(runes []rune, lang string) []logAttr {
	text := make([]rune, len(runes))
	for i, r := range runes {
		// Bidi embedding and override controls are not supported; they
		// are given the break properties of a zero-width space.
		if r >= '\u202a' && r <= '\u202e' {
			r = '\u200b'
		}
		text[i] = r
	}
	attrs := computeLogAttrs(text)
	tailorCatalan(text, attrs, lang)
	return attrs
}

// tailorCatalan keeps "l·l" in one word in Catalan.
func tailorCatalan(text []rune, attrs []logAttr, lang string) {
	if len(lang) < 3 || !strings.EqualFold(lang[:3], "ca-") {
		return
	}
	for i, r := range text {
		if r != '\u00b7' || i == 0 || i+1 >= len(text) {
			continue
		}
		if unicode.ToLower(text[i-1]) == 'l' && unicode.ToLower(text[i+1]) == 'l' {
			attrs[i].wordEnd = false
			attrs[i+1].wordStart = false
		}
	}
}

// nextBreakPoint returns the index of the first line break opportunity in
// attrs, or -1.
func nextBreakPoint(attrs []logAttr) int {
	for i, a := range attrs {
		if a.lineBreak {
			return i
		}
	}
	return -1
}

// nextBreakPointInText returns the first line break opportunity strictly
// inside text, as the number of characters before it minus one, or -1.
func nextBreakPointInText(runes []rune, lang string) int {
	if len(runes) < 2 {
		return -1
	}
	attrs := textLogAttrs(runes, lang)
	return nextBreakPoint(attrs[1:len(runes)])
}

// CanBreakText reports whether a line can be broken inside text, not at
// its ends (UAX #14).
func CanBreakText(text, lang string) bool {
	return nextBreakPointInText([]rune(text), lang) >= 0
}

// NextBreakPoint returns the byte offset of the first line break
// opportunity strictly inside text, and false when there is none.
func NextBreakPoint(text, lang string) (int, bool) {
	runes := []rune(text)
	i := nextBreakPointInText(runes, lang)
	if i < 0 {
		return 0, false
	}
	return byteOffset(text, i+1), true
}

// nextWordBoundaries returns the character offsets of the start and end
// of the first word of text.
func nextWordBoundaries(runes []rune, lang string) (start, end int, ok bool) {
	if len(runes) < 2 {
		return 0, 0, false
	}
	for i, a := range textLogAttrs(runes, lang) {
		if a.wordEnd {
			return start, i, true
		}
		if a.wordBoundary {
			start = i
		}
	}
	return 0, 0, false
}

// NextWordBoundaries returns the byte offsets of the start and end of the
// first word of text. Texts shorter than two characters have no word.
func NextWordBoundaries(text, lang string) (start, end int, ok bool) {
	s, e, ok := nextWordBoundaries([]rune(text), lang)
	if !ok {
		return 0, 0, false
	}
	return byteOffset(text, s), byteOffset(text, e), true
}

// LastWordEnd returns the byte offset of the end of the last word that
// ends before the end of text.
func LastWordEnd(text, lang string) (int, bool) {
	runes := []rune(text)
	if len(runes) < 2 {
		return 0, false
	}
	attrs := textLogAttrs(runes, lang)
	for i := len(attrs) - 2; i >= 0; i-- {
		if attrs[i].wordEnd {
			return byteOffset(text, i), true
		}
	}
	return 0, false
}

// byteOffset returns the byte offset of the character at index n of text.
func byteOffset(text string, n int) int {
	offset := 0
	for i := 0; i < n && offset < len(text); i++ {
		_, size := utf8.DecodeRuneInString(text[offset:])
		offset += size
	}
	return offset
}
