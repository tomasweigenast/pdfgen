package shape

import "unicode/utf8"

// Spacing holds the letter-spacing and word-spacing of a run in 1/1024 px.
// A letter-spacing of normal is zero.
type Spacing struct {
	Letter, Word int32
}

// Edges tells which ends of the laid-out text a run touches. The ends of a
// text get only half of the spacing of their character, so that spacing
// goes between characters and not around the text.
type Edges struct {
	// Start is true for the run holding the first character of the text
	// and End for the one holding the last character.
	Start, End bool
}

// ApplySpacing adds letter-spacing and word-spacing to glyphs, which must
// come from shaping text in visual order.
//
// Every cluster is widened by the letter-spacing, half on each side, and
// word separators (U+0020 and U+00A0) by the word-spacing as well. The side
// of the first character of the text that faces the outside of the text and
// the side of the last one are left out (CSS Text 3 §8.1 asks for the
// spacing between characters). A separator at an end gets twice the spacing
// first, so that it keeps the whole of it.
//
// Clusters are the unit of spacing: a base character with its combining
// marks is spaced once.
func ApplySpacing(glyphs []Glyph, text []rune, rtl bool, s Spacing, e Edges) {
	if len(glyphs) == 0 || (s.Letter == 0 && s.Word == 0) {
		return
	}
	// Character starting at each byte offset, to classify clusters in
	// constant time.
	at := make(map[int]rune, len(text))
	size, lastChar := 0, 0
	for _, c := range text {
		at[size] = c
		lastChar = size
		size += utf8.RuneLen(c)
	}

	// In visual order the start of the text is on the right for RTL.
	leftEdge, rightEdge := e.Start, e.End
	if rtl {
		leftEdge, rightEdge = e.End, e.Start
	}

	for i := 0; i < len(glyphs); {
		j := i + 1
		for j < len(glyphs) && glyphs[j].Cluster == glyphs[i].Cluster {
			j++
		}
		first, last := i, j-1
		i = j

		c := glyphs[first].Cluster
		sep := at[c] == ' ' || at[c] == '\u00a0'
		space := s.Letter
		if sep {
			space += s.Word
		}
		if sep && ((e.Start && c == 0) || (e.End && c == lastChar)) {
			space *= 2
		}
		left := space / 2
		right := space - left

		atLeft := leftEdge && first == 0
		atRight := rightEdge && last == len(glyphs)-1
		if !atLeft {
			glyphs[first].XOffset += left
			glyphs[first].XAdvance += left
		}
		if !atRight {
			glyphs[last].XAdvance += right
		}
	}
}
