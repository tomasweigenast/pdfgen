package hyphen

import (
	"strings"
	"unicode"
)

// isCased reports whether r has a case distinction.
func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r)
}

// isCaseIgnorable approximates the Unicode Case_Ignorable property used by
// the Final_Sigma rule.
func isCaseIgnorable(r rune) bool {
	switch r {
	case '\'', '.', ':', '^', '`', '¨', '­', '¯', '´', '·', '¸', '’':
		return true
	}
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk)
}

// lower lowercases s without changing the code point count, except for the
// two cases where full Unicode lowercasing does: U+0130 expands to two code
// points and a word-final capital sigma becomes a final sigma.
func lower(s string) string {
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range runes {
		switch {
		case r == 'İ':
			b.WriteString("i̇")
		case r == 'Σ':
			if finalSigma(runes, i) {
				b.WriteRune('ς')
			} else {
				b.WriteRune('σ')
			}
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// finalSigma implements the Final_Sigma condition of Unicode SpecialCasing:
// a cased letter precedes the sigma and none follows it.
func finalSigma(runes []rune, i int) bool {
	before := false
	for j := i - 1; j >= 0; j-- {
		if isCaseIgnorable(runes[j]) {
			continue
		}
		before = isCased(runes[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(runes); j++ {
		if isCaseIgnorable(runes[j]) {
			continue
		}
		return !isCased(runes[j])
	}
	return true
}

// isUpper reports whether s has at least one cased letter and all of them
// are uppercase.
func isUpper(s string) bool {
	upper := false
	for _, r := range s {
		switch {
		case unicode.IsLower(r) || unicode.IsTitle(r):
			return false
		case unicode.IsUpper(r):
			upper = true
		}
	}
	return upper
}
