package fonts

import (
	"math"
	"strconv"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"
)

// Metrics are the vertical metrics of a face at one size, in 1/1024 px.
// Positions are measured upward from the baseline, so the underline position
// is usually negative.
type Metrics struct {
	UnitsPerEm int
	// Size is the font size in 1/1024 px the metrics were scaled to.
	Size int32

	Ascent, Descent int32

	UnderlinePosition, UnderlineThickness         int32
	StrikethroughPosition, StrikethroughThickness int32
}

// Metrics returns the metrics of the face at a size in px.
//
// The ascent and descent come from the hhea table, or from the OS/2 typo
// values when the font sets USE_TYPO_METRICS (OpenType OS/2 fsSelection bit
// 7). The line gap is deliberately left out: the height of a line is
// ascent + descent.
func (f *Face) Metrics(size float64) Metrics {
	units := ToUnits(size)
	h := f.HB(units)
	ext := h.ExtentsForDirection(harfbuzz.LeftToRight)
	upem := int(f.GoText.Upem())
	m := Metrics{
		UnitsPerEm: upem,
		Size:       units,
		Ascent:     int32(ext.Ascender),
		Descent:    int32(-ext.Descender),
	}
	m.UnderlinePosition = f.scaleMetric(f.GoText.LineMetric(font.UnderlinePosition), units)
	m.UnderlineThickness = f.scaleMetric(f.GoText.LineMetric(font.UnderlineThickness), units)
	m.StrikethroughPosition = f.scaleMetric(f.GoText.LineMetric(font.StrikethroughPosition), units)
	m.StrikethroughThickness = f.scaleMetric(f.GoText.LineMetric(font.StrikethroughThickness), units)
	return m
}

// scaleMetric scales a value in font units the way the shaper scales table
// values. Static values use a 16.16 multiplier rounded half up, and values
// adjusted by variations are scaled in floating point and rounded.
func (f *Face) scaleMetric(v float32, size int32) int32 {
	upem := int64(f.GoText.Upem())
	if upem == 0 {
		return 0
	}
	if len(f.GoText.Coords()) != 0 {
		return int32(math.Round(float64(v) * float64(size) / float64(upem)))
	}
	mult := (int64(size) << 16) / upem
	return int32((int64(math.Round(float64(v)))*mult + 32768) >> 16)
}

// Height returns the logical height of a line of text in px, ascent plus
// descent.
func (m Metrics) Height() float64 { return FromUnits(m.Ascent + m.Descent) }

// Baseline returns the distance in px from the top of the line to the
// baseline.
func (m Metrics) Baseline() float64 { return FromUnits(m.Ascent) }

// Strut returns the height of a line box and the baseline offset from its
// top. A nil lineHeight means line-height: normal, which uses the font
// metrics; otherwise lineHeight is the used value in px and the content is
// centered in it.
func (m Metrics) Strut(lineHeight *float64) (height, baseline float64) {
	if lineHeight == nil {
		return m.Height(), m.Baseline()
	}
	return *lineHeight, m.Baseline() + (*lineHeight-m.Height())/2
}

// ratioSize is the font size in px used to measure character ratios, large
// enough that 1/1024 px rounding is irrelevant.
const ratioSize = 1000

// RatioCharacter returns the character whose measure defines a font-relative
// unit (ex, cap, ic and ch), and false for other units.
func RatioCharacter(unit string) (rune, bool) {
	switch unit {
	case "ex":
		return 'x', true
	case "cap":
		return 'O', true
	case "ic":
		return '水', true
	case "ch":
		return '0', true
	}
	return 0, false
}

// CharacterRatio returns the size of a font-relative unit as a fraction of
// the font size, measured on face. The face must be an instance for the size
// of 1000 px, as returned by Query for a Description of that size.
//
// The ex unit is the height of the ink of "x". The other units are the
// advance of their character, which includes cap: it measures the advance
// of "O" and not the capital height. The ratio is rounded to five decimals,
// and falls back to 0.5 when the measure is zero.
func (f *Face) CharacterRatio(unit, lang string, features []Feature) float64 {
	r, ok := RatioCharacter(unit)
	if !ok {
		return 0
	}
	ink, advance := f.measureRune(r, ToUnits(ratioSize), lang, features)
	measure := advance
	if unit == "ex" {
		measure = ink
	}
	return ratio(FromUnits(measure))
}

func ratio(measure float64) float64 {
	v := roundHalfEven(measure/ratioSize, 5)
	if v == 0 {
		return 0.5
	}
	return v
}

// roundHalfEven rounds x to n decimals, resolving the exact binary value of
// x with ties to even.
func roundHalfEven(x float64, n int) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return v
}

// measureRune shapes one character and returns the height of its ink above
// the baseline and its advance, in 1/1024 px.
func (f *Face) measureRune(r rune, size int32, lang string, features []Feature) (ink, advance int32) {
	h := f.HB(size)
	buf := harfbuzz.NewBuffer()
	buf.AddRunes([]rune{r}, 0, 1)
	buf.ClusterLevel = harfbuzz.MonotoneCharacters
	buf.Props.Script = language.LookupScript(r)
	buf.Props.Language = language.NewLanguage(lang)
	buf.Props.Direction = harfbuzz.LeftToRight
	buf.Shape(h, HBFeatures(append(append([]Feature(nil), f.Features...), features...)))

	for i, info := range buf.Info {
		pos := buf.Pos[i]
		if i == 0 {
			if e, ok := h.GlyphExtents(info.Glyph); ok && e.Width != 0 && e.Height != 0 {
				ink = e.YBearing + pos.YOffset
			}
		}
		advance += pos.XAdvance
	}
	return ink, advance
}
