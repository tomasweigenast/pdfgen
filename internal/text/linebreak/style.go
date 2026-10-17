package linebreak

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// Style holds the computed values of the properties that change how text
// is measured and broken into lines. css/computed fills it from the computed
// style of a text box (LineBreakStyle):
//
//	Font                 font-family, font-weight, font-style, font-stretch,
//	                     font-size and font-variation-settings
//	Features             font-kerning, font-variant-* and font-feature-settings
//	Lang                 the lang of the element ("" when unknown)
//	LanguageOverride     font-language-override
//	Direction            direction
//	LetterSpacing        letter-spacing in px, 0 for normal
//	WordSpacing          word-spacing in px
//	LineHeight           line-height
//	WhiteSpace           white-space
//	WordBreak            word-break
//	OverflowWrap         overflow-wrap
//	Hyphens              hyphens
//	HyphenateCharacter   hyphenate-character, "‐" for auto
//	HyphenateLimitChars  hyphenate-limit-chars, auto replaced by 5, 2 and 2
//	HyphenateLimitZone   hyphenate-limit-zone
//	TabSize              tab-size
//
// Empty keywords mean the initial value of the property. Hyphenate-character
// and hyphenate-limit-chars have no keyword: their empty and zero values are
// as valid as any other, and NewStyle gives their initial values.
type Style struct {
	Font     fonts.Description
	Features fonts.FeatureStyle

	Lang             string
	LanguageOverride props.KeywordOrString
	// Direction is "ltr" or "rtl". It sets the base direction of the
	// bidirectional algorithm.
	Direction string

	LetterSpacing float64
	WordSpacing   float64
	LineHeight    LineHeight

	WhiteSpace   string
	WordBreak    string
	OverflowWrap string

	Hyphens             string
	HyphenateCharacter  string
	HyphenateLimitChars HyphenateLimitChars
	HyphenateLimitZone  LengthPercentage

	TabSize TabSize
}

// LineHeight is the computed value of line-height.
type LineHeight = fonts.LineHeight

// LengthPercentage is a length in px or, when Percent is set, a percentage.
type LengthPercentage struct {
	Value   float64
	Percent bool
	// Calc, when not nil, replaces Value and Percent. It resolves a math
	// function whose percentages refer to the base given to Resolve.
	Calc func(base float64) float64
}

// Resolve returns the value in px, percentages being relative to base.
func (l LengthPercentage) Resolve(base float64) float64 {
	if l.Calc != nil {
		return l.Calc(base)
	}
	if l.Percent {
		return l.Value * base / 100
	}
	return l.Value
}

// HyphenateLimitChars holds the minimum length of a hyphenated word and the
// minimum numbers of characters before and after a hyphen.
type HyphenateLimitChars struct {
	Total, Before, After int
}

// TabSize is the computed value of tab-size: a number of spaces, or a
// length in px when IsLength is set.
type TabSize struct {
	Spaces   int
	Length   float64
	IsLength bool
}

// NewStyle returns a style with the initial values of the properties, for
// a font size in px and a list of families.
func NewStyle(size float64, families ...string) *Style {
	return &Style{
		Font:                fonts.Description{Families: families, Weight: 400, Size: size},
		Direction:           "ltr",
		LineHeight:          LineHeight{Normal: true},
		WhiteSpace:          "normal",
		WordBreak:           "normal",
		OverflowWrap:        "normal",
		Hyphens:             "manual",
		HyphenateCharacter:  "\u2010",
		HyphenateLimitChars: HyphenateLimitChars{5, 2, 2},
		TabSize:             TabSize{Spaces: 8},
	}
}

func (s *Style) whiteSpace() string {
	if s.WhiteSpace == "" {
		return "normal"
	}
	return s.WhiteSpace
}

// textWrap reports whether lines may wrap (CSS Text 3 §3).
func (s *Style) textWrap() bool {
	switch s.whiteSpace() {
	case "normal", "pre-wrap", "pre-line":
		return true
	}
	return false
}

// spaceCollapse reports whether white space collapses (CSS Text 3 §3).
func (s *Style) spaceCollapse() bool {
	switch s.whiteSpace() {
	case "normal", "nowrap", "pre-line":
		return true
	}
	return false
}

func (s *Style) hyphens() string {
	if s.Hyphens == "" {
		return "manual"
	}
	return s.Hyphens
}

func (s *Style) tabSize() TabSize {
	if s.TabSize == (TabSize{}) {
		return TabSize{Spaces: 8}
	}
	return s.TabSize
}

func (s *Style) rtl() bool { return s.Direction == "rtl" }

// overflowWrapBreaksWords reports whether overflow-wrap allows breaking
// inside words.
func (s *Style) overflowWrapBreaksWords() bool {
	return s.OverflowWrap == "anywhere" || s.OverflowWrap == "break-word"
}

// shapingLanguage is the language given to the shaper and used to pick
// fonts: font-language-override when set, else the language of the element.
func (s *Style) shapingLanguage() string {
	return fonts.Language(s.LanguageOverride, s.Lang)
}

func (s *Style) features() []fonts.Feature { return fonts.Features(s.Features) }

// fontSpec returns what selects and shapes the font of the style.
func (s *Style) fontSpec() fonts.Spec {
	return fonts.Spec{
		Font:             s.Font,
		Features:         s.Features,
		Lang:             s.Lang,
		LanguageOverride: s.LanguageOverride,
	}
}
