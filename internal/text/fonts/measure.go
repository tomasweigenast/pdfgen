package fonts

import (
	"fmt"
	"strconv"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// Spec holds the computed properties that select and shape the font of a
// run of text. It is the input of the measures that depend on the font of a
// style: the strut and the size of font-relative units.
type Spec struct {
	// Font is the font-family, font-weight, font-style, font-stretch,
	// font-size and font-variation-settings.
	Font Description
	// Features are font-kerning, the font-variant longhands and
	// font-feature-settings.
	Features FeatureStyle
	// Lang is the language of the element, empty when unknown.
	Lang string
	// LanguageOverride is font-language-override.
	LanguageOverride props.KeywordOrString
}

// Language returns the language used to select and shape the font.
func (s Spec) Language() string { return Language(s.LanguageOverride, s.Lang) }

// FontFeatures returns the OpenType features the spec turns on or off.
func (s Spec) FontFeatures() []Feature { return Features(s.Features) }

// key identifies the font a spec selects, whatever its size. The language
// takes the place of lang and font-language-override, which only matter
// through it. Kerning is left out: the measures are taken on one glyph.
func (s Spec) key() string {
	d, f := s.Font, s.Features
	return fmt.Sprintf("%q|%d|%q|%q|%#v|%q|%q|%q|%q|%q|%q|%#v|%q",
		d.Families, d.Weight, d.Style, d.Stretch, d.Variations,
		f.Ligatures, f.Position, f.Caps, f.Numeric, f.Alternates, f.EastAsian,
		f.Settings, s.Language())
}

// LineHeight is the computed value of line-height, resolved to what a strut
// needs.
type LineHeight struct {
	// Normal is true for the normal keyword, which uses the metrics of the
	// font.
	Normal bool
	// Number is true when Value multiplies the font size; otherwise Value
	// is a length in px.
	Number bool
	Value  float64
}

// Used returns the line height in px for a font size, or nil for normal.
func (l LineHeight) Used(fontSize float64) *float64 {
	if l.Normal {
		return nil
	}
	v := l.Value
	if l.Number {
		v *= fontSize
	}
	return &v
}

// Strut returns the height of a line box for the font of a spec and the
// position of its baseline from the top (CSS 2.1 §10.8.1). A normal
// line-height gives the height of the font, which is measured on the first
// font available for a space.
//
// Metrics are kept per font and size, so styles that only differ in
// line-height measure the font once.
func (fc *FontConfiguration) Strut(s Spec, lineHeight LineHeight) (height, baseline float64) {
	size := s.Font.Size
	if size == 0 {
		return 0, 0
	}
	return fc.fontMetrics(s).Strut(lineHeight.Used(size))
}

func (fc *FontConfiguration) fontMetrics(s Spec) Metrics {
	key := s.key() + "|" + strconv.FormatFloat(s.Font.Size, 'g', -1, 64)
	if m, ok := fc.metrics[key]; ok {
		return m
	}
	var m Metrics
	if face := fc.Query(s.Font, s.Language(), ' '); face != nil {
		m = face.Metrics(s.Font.Size)
	}
	fc.metrics[key] = m
	return m
}

// CharacterRatio returns the size of a font-relative unit ("ex", "cap",
// "ic" or "ch") as a fraction of the font size, for the font of a spec. The
// size of the spec is ignored: the measure is taken at a large size, with
// the character searched through the families like any other, so a font
// without it is replaced by a fallback font. Other units give 0.
func (fc *FontConfiguration) CharacterRatio(s Spec, unit string) float64 {
	r, ok := RatioCharacter(unit)
	if !ok {
		return 0
	}
	key := unit + "|" + s.key()
	if v, ok := fc.ratios[key]; ok {
		return v
	}
	d := s.Font
	d.Size = ratioSize
	v := 0.5
	if face := fc.Query(d, s.Language(), r); face != nil {
		v = face.CharacterRatio(unit, s.Language(), s.FontFeatures())
	}
	fc.ratios[key] = v
	return v
}
