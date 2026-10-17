package computed

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
)

// LineBreakStyle copies the properties used to measure and break text into
// lines. Math functions with percentages retain the style to resolve their
// units when the line width and page dimensions are known.
func (s *Style) LineBreakStyle() *linebreak.Style {
	spec := s.FontSpec()
	return &linebreak.Style{
		Font:             spec.Font,
		Features:         spec.Features,
		Lang:             spec.Lang,
		LanguageOverride: spec.LanguageOverride,
		Direction:        string(s.Direction()),

		LetterSpacing: s.letterSpacing(),
		WordSpacing:   float64(s.WordSpacing()),
		LineHeight:    s.fontLineHeight(),

		WhiteSpace:   string(s.WhiteSpace()),
		WordBreak:    string(s.WordBreak()),
		OverflowWrap: string(s.OverflowWrap()),

		Hyphens:             string(s.Hyphens()),
		HyphenateCharacter:  string(s.HyphenateCharacter()),
		HyphenateLimitChars: hyphenateLimitChars(s.HyphenateLimitChars()),
		HyphenateLimitZone:  s.hyphenateLimitZone(),

		TabSize: s.tabSize(),
	}
}

// letterSpacing returns letter-spacing in px, which is 0 for normal.
func (s *Style) letterSpacing() float64 {
	l := s.LetterSpacing()
	switch {
	case l.Keyword != "":
		return 0
	case l.Calc != nil:
		return s.resolveCalc(l.Calc, props.PropLetterSpacing, 0)
	}
	return l.Dimension.Value
}

func hyphenateLimitChars(t props.IntTriple) linebreak.HyphenateLimitChars {
	return linebreak.HyphenateLimitChars{Total: t[0], Before: t[1], After: t[2]}
}

// hyphenateLimitZone returns hyphenate-limit-zone. The percentages of a
// math function refer to the width of the line, which is only known when
// text is laid out.
func (s *Style) hyphenateLimitZone() linebreak.LengthPercentage {
	l := s.HyphenateLimitZone()
	switch {
	case l.Calc != nil:
		calc := l.Calc
		return linebreak.LengthPercentage{Calc: func(base float64) float64 {
			return s.resolveCalc(calc, props.PropHyphenateLimitZone, base)
		}}
	case l.Dimension.Unit == "%":
		return linebreak.LengthPercentage{Value: l.Dimension.Value, Percent: true}
	}
	return linebreak.LengthPercentage{Value: l.Dimension.Value}
}

// tabSize returns tab-size, a number of spaces or a length in px. The
// number of spaces is never zero: tab-size: 0 is a length (CSS Text 3 §7.2).
func (s *Style) tabSize() linebreak.TabSize {
	l := s.TabSize()
	switch {
	case l.Calc != nil:
		return linebreak.TabSize{Length: s.resolveCalc(l.Calc, props.PropTabSize, 0), IsLength: true}
	case l.Dimension.Unit == "" && l.Dimension.Value != 0:
		return linebreak.TabSize{Spaces: int(l.Dimension.Value)}
	}
	return linebreak.TabSize{Length: l.Dimension.Value, IsLength: true}
}
