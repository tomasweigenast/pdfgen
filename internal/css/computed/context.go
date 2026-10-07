package computed

import (
	"fmt"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// Size is a width and a height in CSS pixels.
type Size struct{ Width, Height float64 }

// Context is shared by the styles of a document: the fonts that measure
// font-relative units, the page sizes that viewport units refer to, and the
// caches of those measures.
type Context struct {
	// FontConfig gives the fonts of the document.
	FontConfig *fonts.FontConfiguration
	// PageBox and PageArea are the sizes of the box and of the content area
	// of the first page, which the pv* and v* units refer to. They are nil
	// until the first page is laid out.
	PageBox, PageArea *Size

	initial *Style
	ratios  map[string]float64
	struts  map[string][2]float64
}

// NewContext returns a context measuring fonts with fc. A nil fc uses the
// installed fonts.
func NewContext(fc *fonts.FontConfiguration) *Context {
	if fc == nil {
		fc = fonts.NewFontConfiguration()
	}
	c := &Context{
		FontConfig: fc,
		ratios:     map[string]float64{},
		struts:     map[string][2]float64{},
	}
	c.initial = &Style{kind: initialStyle, ctx: c}
	c.initial.init()
	return c
}

// Initial returns the style holding the initial value of every property,
// which serves as root style for the root element.
func (c *Context) Initial() *Style { return c.initial }

// fontKey identifies what changes the font used to measure text, apart from
// its size.
func fontKey(s *Style) string {
	return fmt.Sprint(
		s.FontFamily(), s.FontStyle(), s.FontStretch(), s.FontWeight(),
		s.FontVariantLigatures(), s.FontVariantPosition(), s.FontVariantCaps(),
		s.FontVariantNumeric(), s.FontVariantAlternates(), s.FontVariantEastAsian(),
		s.FontFeatureSettings(), s.FontVariationSettings(),
		s.FontLanguageOverride(), s.Lang())
}

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

// FontFeatures returns the OpenType features the style turns on or off.
func (s *Style) FontFeatures() []fonts.Feature {
	return fonts.Features(fonts.FeatureStyle{
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
	})
}

// language returns the language used to select and shape the font.
func (s *Style) language() string {
	return fonts.Language(s.FontLanguageOverride(), string(s.Lang()))
}

// CharacterRatio returns the size of the ex, ch, cap or ic unit as a
// fraction of the font size, measured on the font of the style.
func (s *Style) CharacterRatio(unit string) float64 {
	key := unit + "\x00" + fontKey(s)
	if r, ok := s.ctx.ratios[key]; ok {
		return r
	}
	r := s.ctx.FontConfig.CharacterRatio(s.FontDescription(), s.language(), s.FontFeatures(), unit)
	s.ctx.ratios[key] = r
	return r
}

// Strut returns the used value of line-height and the position of the
// baseline from the top of the line, for an empty line in the first
// available font of the style (CSS 2.1 §10.8.1).
func (s *Style) Strut() (lineHeight, baseline float64) {
	size := float64(s.FontSize())
	if size == 0 {
		return 0, 0
	}
	lh := s.LineHeight()
	key := fmt.Sprint(fontKey(s), size, lh)
	if v, ok := s.ctx.struts[key]; ok {
		return v[0], v[1]
	}

	var metrics fonts.Metrics
	if face := s.ctx.FontConfig.Query(s.FontDescription(), s.language(), ' '); face != nil {
		metrics = face.Metrics(size)
	}
	var used *float64
	switch {
	case lh.Keyword == "normal":
	case lh.Calc != nil:
		v := s.resolveCalc(lh.Calc, props.PropLineHeight, size)
		used = &v
	case lh.Dimension.Unit == "":
		v := lh.Dimension.Value * size
		used = &v
	default:
		v := lh.Dimension.Value
		used = &v
	}
	lineHeight, baseline = metrics.Strut(used)
	s.ctx.struts[key] = [2]float64{lineHeight, baseline}
	return lineHeight, baseline
}
