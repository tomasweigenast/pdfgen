package transform

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

// Language-specific tailorings are keyed by the primary language subtag of the
// element's language.
func isTurkic(lang string) bool { return lang == "tr" || lang == "az" }

// Transform applies the text-transform keyword function to text in the given
// language. Supported keywords are "uppercase", "lowercase", "capitalize" and
// "full-width"; ok is false for any other keyword, and text is then returned
// unchanged.
func Transform(function, text, lang string) (result string, ok bool) {
	switch function {
	case "uppercase":
		return Uppercase(text, lang), true
	case "lowercase":
		return Lowercase(text, lang), true
	case "capitalize":
		return Capitalize(text, lang), true
	case "full-width":
		return FullWidth(text), true
	}
	return text, false
}

// Uppercase converts text to upper case using the full Unicode case mapping,
// with the Greek and Turkic tailorings of CLDR's el-Upper and tr-Upper.
func Uppercase(text, lang string) string {
	var replacer *strings.Replacer
	switch {
	case lang == "el":
		replacer = strings.NewReplacer(
			"άι", "ΑΪ",
			"άυ", "ΑΫ",
			"όι", "ΟΪ",
			"όυ", "ΟΫ",
			"έυ", "ΗΫ",
		)
	case isTurkic(lang):
		replacer = strings.NewReplacer("i", "İ")
	}
	if replacer != nil {
		text = replacer.Replace(text)
	}

	if lang == "el" {
		text = stripGreekDiacritics(text)
	}
	return cases.Upper(language.Und).String(text)
}

// greekDiacritics are the combining marks dropped by el-Upper: psili, dasia,
// oxia/tonos, varia, vrachy, perispomeni, macron and ypogegrammeni.
var greekDiacritics = strings.NewReplacer(
	"̓", "", "̔", "", "́", "", "̀", "",
	"̆", "", "͂", "", "̄", "", "ͅ", "",
)

func stripGreekDiacritics(text string) string {
	return norm.NFC.String(greekDiacritics.Replace(norm.NFD.String(text)))
}

// Lowercase converts text to lower case using the full Unicode case mapping,
// including the final sigma rule, with the Turkic and Lithuanian tailorings of
// CLDR's tr-Lower and lt-Lower.
func Lowercase(text, lang string) string {
	var replacer *strings.Replacer
	switch {
	case isTurkic(lang):
		replacer = strings.NewReplacer("I", "ı", "İ", "i")
	case lang == "lt":
		replacer = strings.NewReplacer(
			"Ì", "i̇̀",
			"Í", "i̇́",
			"Ĩ", "i̇̃",
		)
	}
	if replacer != nil {
		text = replacer.Replace(text)
	}
	return cases.Lower(language.Und).String(text)
}

// Capitalize upper-cases the first letter or digit of each space-separated
// word (CSS Text 3 §2.1). Only separator characters (Unicode category Z)
// start a new word, so punctuation inside a word never does. In Dutch a
// leading "ij" is capitalized as the digraph "IJ".
func Capitalize(text, lang string) string {
	var out strings.Builder
	out.Grow(len(text))
	letterFound := false
	skipNext := false
	for i, r := range text {
		if skipNext {
			skipNext = false
			continue
		}
		switch {
		case !letterFound && (unicode.IsLetter(r) || unicode.IsNumber(r)):
			letterFound = true
			switch {
			case lang == "nl" && strings.HasPrefix(text[i:], "ij"):
				skipNext = true
				out.WriteString("IJ")
			case isTurkic(lang):
				out.WriteString(Uppercase(string(r), lang))
			default:
				out.WriteString(cases.Upper(language.Und).String(string(r)))
			}
			continue
		case unicode.Is(unicode.Z, r):
			letterFound = false
		}
		out.WriteRune(r)
	}
	return out.String()
}

// FullWidth maps ASCII printable characters to their full-width forms, the
// space to U+3000 and the hyphen-minus to U+2212 (CSS Text 3 §2.1).
func FullWidth(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ':
			return '　'
		case r == '-':
			return '−'
		case r > 0x20 && r < 0x7f:
			return r + 0xfee0
		}
		return r
	}, text)
}

// FirstLetter returns the prefix of text that ::first-letter applies to: any
// leading punctuation (Ps, Pe, Pi, Pf, Po) plus the first following character
// (CSS Pseudo-Elements 4, "Generated Content: ::first-letter").
func FirstLetter(text string) string {
	end := 0
	found := false
	for end < len(text) {
		r, size := utf8.DecodeRuneInString(text[end:])
		if !unicode.In(r, unicode.Ps, unicode.Pe, unicode.Pi, unicode.Pf, unicode.Po) {
			if found {
				break
			}
			found = true
		}
		end += size
	}
	return text[:end]
}
