package fonts

import (
	"strconv"
	"strings"

	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/harfbuzz"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// Feature is an OpenType feature setting. Tag is a four-letter tag such as
// "liga".
type Feature struct {
	Tag   string
	Value int
}

// String formats the feature as "tag value".
func (f Feature) String() string { return f.Tag + " " + strconv.Itoa(f.Value) }

// FeatureString joins features as "tag value,tag value".
func FeatureString(features []Feature) string {
	parts := make([]string, len(features))
	for i, f := range features {
		parts[i] = f.String()
	}
	return strings.Join(parts, ",")
}

// HBFeatures converts features to HarfBuzz features covering the whole text.
// Settings whose tag is not four bytes long are dropped.
func HBFeatures(features []Feature) []harfbuzz.Feature {
	out := make([]harfbuzz.Feature, 0, len(features))
	for _, f := range features {
		if len(f.Tag) != 4 {
			continue
		}
		out = append(out, harfbuzz.Feature{
			Tag:   opentype.MustNewTag(f.Tag),
			Value: uint32(max(f.Value, 0)),
			Start: harfbuzz.FeatureGlobalStart,
			End:   harfbuzz.FeatureGlobalEnd,
		})
	}
	return out
}

// FontVariant holds the longhands of font-variant, as computed keywords.
// An empty value means normal.
type FontVariant struct {
	Ligatures  props.Keywords
	Position   props.Keyword
	Caps       props.Keyword
	Numeric    props.Keywords
	Alternates props.Keyword
	EastAsian  props.Keywords
}

// FeatureStyle holds the computed properties that select OpenType features.
// An empty Kerning means normal, which turns kern on.
type FeatureStyle struct {
	Kerning props.Keyword
	FontVariant
	Settings props.FontSettings
}

var (
	ligatureKeys = map[props.Keyword][]string{
		"common-ligatures":        {"liga", "clig"},
		"historical-ligatures":    {"hlig"},
		"discretionary-ligatures": {"dlig"},
		"contextual":              {"calt"},
	}
	// ligatureOrder fixes the iteration order of font-variant-ligatures: none.
	ligatureOrder = []props.Keyword{
		"common-ligatures", "historical-ligatures", "discretionary-ligatures", "contextual",
	}
	capsKeys = map[props.Keyword][]string{
		"small-caps":      {"smcp"},
		"all-small-caps":  {"c2sc", "smcp"},
		"petite-caps":     {"pcap"},
		"all-petite-caps": {"c2pc", "pcap"},
		"unicase":         {"unic"},
		"titling-caps":    {"titl"},
	}
	numericKeys = map[props.Keyword]string{
		"lining-nums":        "lnum",
		"oldstyle-nums":      "onum",
		"proportional-nums":  "pnum",
		"tabular-nums":       "tnum",
		"diagonal-fractions": "frac",
		"stacked-fractions":  "afrc",
		"slashed-zero":       "zero",
	}
	eastAsianKeys = map[props.Keyword]string{
		"full-width":         "fwid",
		"proportional-width": "pwid",
	}
)

// featureSet is an insertion-ordered map of features: setting a tag again
// changes its value but keeps its position.
type featureSet struct {
	list  []Feature
	index map[string]int
}

func (s *featureSet) set(tag string, value int) {
	if i, ok := s.index[tag]; ok {
		s.list[i].Value = value
		return
	}
	if s.index == nil {
		s.index = map[string]int{}
	}
	s.index[tag] = len(s.list)
	s.list = append(s.list, Feature{tag, value})
}

func isNormal(k props.Keyword) bool { return k == "" || k == "normal" }

func isNormalList(ks props.Keywords) bool {
	return len(ks) == 0 || (len(ks) == 1 && ks[0] == "normal")
}

// Features returns the OpenType features selected by a style, in the order
// of the cascade of CSS Fonts 3 §6.12.1: font-kerning, then the font-variant
// longhands, then font-feature-settings.
func Features(s FeatureStyle) []Feature {
	var set featureSet

	if s.Kerning != "auto" {
		set.set("kern", b2i(isNormal(s.Kerning)))
	}

	switch {
	case len(s.Ligatures) == 1 && s.Ligatures[0] == "none":
		for _, k := range ligatureOrder {
			for _, tag := range ligatureKeys[k] {
				set.set(tag, 0)
			}
		}
	case !isNormalList(s.Ligatures):
		for _, k := range s.Ligatures {
			value := 1
			if rest, ok := strings.CutPrefix(string(k), "no-"); ok {
				value, k = 0, props.Keyword(rest)
			}
			for _, tag := range ligatureKeys[k] {
				set.set(tag, value)
			}
		}
	}

	switch s.Position {
	case "sub":
		set.set("subs", 1)
	case "super":
		set.set("sups", 1)
	}

	if !isNormal(s.Caps) {
		for _, tag := range capsKeys[s.Caps] {
			set.set(tag, 1)
		}
	}

	if !isNormalList(s.Numeric) {
		for _, k := range s.Numeric {
			if tag, ok := numericKeys[k]; ok {
				set.set(tag, 1)
			}
		}
	}

	if s.Alternates == "historical-forms" {
		set.set("hist", 1)
	}

	if !isNormalList(s.EastAsian) {
		for _, k := range s.EastAsian {
			if tag, ok := eastAsianKeys[k]; ok {
				set.set(tag, 1)
			}
		}
	}

	if !s.Settings.Normal {
		for _, item := range s.Settings.Items {
			set.set(item.Tag, int(item.Value))
		}
	}
	return set.list
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Language returns the language used for shaping and font selection:
// font-language-override when it is a string, converted from its OpenType
// language system tag to an ISO 639 code, else the lang of the element.
func Language(override props.KeywordOrString, lang string) string {
	if override.Keyword != "" || override.String == "" {
		return lang
	}
	if iso, ok := languageTagToISO[strings.ToLower(override.String)]; ok {
		return iso
	}
	return override.String
}
