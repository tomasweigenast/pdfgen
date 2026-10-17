package computed

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// Size is a width and a height in CSS pixels.
type Size struct{ Width, Height float64 }

// Context is shared by the styles of a document: the fonts that measure
// font-relative units, and the page sizes that viewport units refer to. The
// measures of the fonts are cached by the font configuration.
type Context struct {
	// FontConfig gives the fonts of the document.
	FontConfig *fonts.FontConfiguration
	// PageBox and PageArea are the sizes of the box and of the content area
	// of the first page, which the pv* and v* units refer to. They are nil
	// until the first page is laid out.
	PageBox, PageArea *Size

	initial *Style
}

// NewContext returns a context measuring fonts with fc. A nil fc uses the
// installed fonts.
func NewContext(fc *fonts.FontConfiguration) *Context {
	if fc == nil {
		fc = fonts.NewFontConfiguration()
	}
	c := &Context{FontConfig: fc}
	c.initial = &Style{kind: initialStyle, ctx: c}
	c.initial.init()
	return c
}

// Initial returns the style holding the initial value of every property,
// which serves as root style for the root element.
func (c *Context) Initial() *Style { return c.initial }

// FontDescription returns the font a style selects, at its font size.
func (s *Style) FontDescription() fonts.Description {
	return fonts.Description{
		Families:   s.FontFamily(),
		Weight:     int(s.FontWeight()),
		Style:      string(s.FontStyle()),
		Stretch:    string(s.FontStretch()),
		Size:       float64(s.FontSize()),
		Variations: s.FontVariationSettings(),
	}
}

// FeatureStyle returns the properties of the style that select OpenType
// features.
func (s *Style) FeatureStyle() fonts.FeatureStyle {
	return fonts.FeatureStyle{
		Kerning: s.FontKerning(),
		FontVariant: fonts.FontVariant{
			Ligatures:  s.FontVariantLigatures(),
			Position:   s.FontVariantPosition(),
			Caps:       s.FontVariantCaps(),
			Numeric:    s.FontVariantNumeric(),
			Alternates: s.FontVariantAlternates(),
			EastAsian:  s.FontVariantEastAsian(),
		},
		Settings: s.FontFeatureSettings(),
	}
}

// FontFeatures returns the OpenType features the style turns on or off.
func (s *Style) FontFeatures() []fonts.Feature { return fonts.Features(s.FeatureStyle()) }

// FontSpec returns everything that selects and shapes the font of the
// style.
func (s *Style) FontSpec() fonts.Spec {
	return fonts.Spec{
		Font:             s.FontDescription(),
		Features:         s.FeatureStyle(),
		Lang:             string(s.Lang()),
		LanguageOverride: s.FontLanguageOverride(),
	}
}

// CharacterRatio returns the size of the ex, ch, cap or ic unit as a
// fraction of the font size, measured on the font of the style.
func (s *Style) CharacterRatio(unit string) float64 {
	return s.ctx.FontConfig.CharacterRatio(s.FontSpec(), unit)
}

// Strut returns the used value of line-height and the position of the
// baseline from the top of the line, for an empty line in the first
// available font of the style (CSS 2.1 §10.8.1).
func (s *Style) Strut() (lineHeight, baseline float64) {
	if s.FontSize() == 0 {
		return 0, 0
	}
	return s.ctx.FontConfig.Strut(s.FontSpec(), s.fontLineHeight())
}

// fontLineHeight returns line-height as a strut needs it. A math function
// has its percentages refer to the font size.
func (s *Style) fontLineHeight() fonts.LineHeight {
	lh := s.LineHeight()
	switch {
	case lh.Keyword == "normal":
		return fonts.LineHeight{Normal: true}
	case lh.Calc != nil:
		return fonts.LineHeight{Value: s.resolveCalc(lh.Calc, props.PropLineHeight, float64(s.FontSize()))}
	case lh.Dimension.Unit == "":
		return fonts.LineHeight{Number: true, Value: lh.Dimension.Value}
	}
	return fonts.LineHeight{Value: lh.Dimension.Value}
}
