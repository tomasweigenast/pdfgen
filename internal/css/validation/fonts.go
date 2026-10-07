package validation

import (
	"strings"
	"unicode/utf8"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

var (
	fontStretchKeywords = []string{
		"ultra-condensed", "extra-condensed", "condensed", "semi-condensed",
		"normal",
		"semi-expanded", "expanded", "extra-expanded", "ultra-expanded",
	}
	fontVariantCapsKeywords = []string{
		"normal", "small-caps", "all-small-caps", "petite-caps",
		"all-petite-caps", "unicase", "titling-caps",
	}

	// The groups of mutually exclusive values of the font-variant longhands.
	ligatureGroups = [][]string{
		{"common-ligatures", "no-common-ligatures"},
		{"historical-ligatures", "no-historical-ligatures"},
		{"discretionary-ligatures", "no-discretionary-ligatures"},
		{"contextual", "no-contextual"},
	}
	numericGroups = [][]string{
		{"lining-nums", "oldstyle-nums"},
		{"proportional-nums", "tabular-nums"},
		{"diagonal-fractions", "stacked-fractions"},
		{"ordinal"}, {"slashed-zero"},
	}
	eastAsianGroups = [][]string{
		{"jis78", "jis83", "jis90", "jis04", "simplified", "traditional"},
		{"full-width", "proportional-width"},
		{"ruby"},
	}
)

func init() {
	register(commaSeparated[props.Strings](fontFamilyItem), 0, "font-family")
	register(keywordOf("auto", "normal", "none"), 0, "font-kerning")
	register(singleToken(plain(validateFontLanguageOverride)), 0, "font-language-override")
	register(validateFontVariantLigatures, 0, "font-variant-ligatures")
	register(keywordOf("normal", "sub", "super"), 0, "font-variant-position")
	register(keywordOf(fontVariantCapsKeywords...), 0, "font-variant-caps")
	register(validateFontVariantNumeric, 0, "font-variant-numeric")
	register(validateFontFeatureSettings, 0, "font-feature-settings")
	register(keywordOf("normal", "historical-forms"), 0, "font-variant-alternates")
	register(validateFontVariantEastAsian, 0, "font-variant-east-asian")
	register(validateFontVariationSettings, 0, "font-variation-settings")
	register(singleToken(plain(validateFontSize)), 0, "font-size")
	register(keywordOf("normal", "italic", "oblique"), 0, "font-style")
	register(keywordOf(fontStretchKeywords...), 0, "font-stretch")
	register(singleToken(plain(validateFontWeight)), 0, "font-weight")
}

// fontFamilyItem validates one family: a string, or identifiers that make up
// an unquoted name.
func fontFamilyItem(tokens []syntax.Node, _ string) (string, bool, error) {
	if len(tokens) == 1 {
		if s, ok := tokens[0].(*syntax.StringToken); ok {
			return s.Value, true, nil
		}
	}
	if len(tokens) == 0 {
		return "", false, nil
	}
	names := make([]string, len(tokens))
	for i, token := range tokens {
		ident, ok := token.(*syntax.IdentToken)
		if !ok {
			return "", false, nil
		}
		names[i] = ident.Value
	}
	return strings.Join(names, " "), true, nil
}

func validateFontLanguageOverride(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "normal" {
		return props.KeywordOrString{Keyword: "normal"}
	}
	if s, ok := token.(*syntax.StringToken); ok {
		return props.KeywordOrString{String: s.Value}
	}
	return nil
}

// fontVariantKeywords validates the keywords of a font-variant longhand: the
// keyword normal alone, or any number of values from different groups.
func fontVariantKeywords(tokens []syntax.Node, groups [][]string, alone ...string) props.Value {
	if len(tokens) == 1 {
		if k := tk.GetKeyword(tokens[0]); contains(alone, k) {
			return props.Keywords{props.Keyword(k)}
		}
	}
	var values props.Keywords
	for _, token := range tokens {
		ident, ok := token.(*syntax.IdentToken)
		if !ok {
			return nil
		}
		group := groupOf(groups, ident.Value)
		if group == nil {
			return nil
		}
		for _, v := range values {
			if contains(group, string(v)) {
				return nil
			}
		}
		values = append(values, props.Keyword(ident.Value))
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func groupOf(groups [][]string, value string) []string {
	for _, g := range groups {
		if contains(g, value) {
			return g
		}
	}
	return nil
}

func validateFontVariantLigatures(tokens []syntax.Node, _ string) (props.Value, error) {
	return fontVariantKeywords(tokens, ligatureGroups, "normal", "none"), nil
}

func validateFontVariantNumeric(tokens []syntax.Node, _ string) (props.Value, error) {
	return fontVariantKeywords(tokens, numericGroups, "normal"), nil
}

func validateFontVariantEastAsian(tokens []syntax.Node, _ string) (props.Value, error) {
	return fontVariantKeywords(tokens, eastAsianGroups, "normal"), nil
}

func validateFontFeatureSettings(tokens []syntax.Node, baseURL string) (props.Value, error) {
	if len(tokens) == 1 && tk.GetKeyword(tokens[0]) == "normal" {
		return props.FontSettings{Normal: true}, nil
	}
	items, ok, err := parseCommaList(tokens, baseURL, fontFeatureItem)
	if err != nil || !ok {
		return nil, err
	}
	return props.FontSettings{Items: items}, nil
}

func fontFeatureItem(tokens []syntax.Node, _ string) (props.FontSetting, bool, error) {
	var feature string
	var value float64
	haveFeature, haveValue := false, false

	if len(tokens) == 2 {
		token := tokens[1]
		tokens = tokens[:1]
		if ident, ok := token.(*syntax.IdentToken); ok {
			switch ident.Value {
			case "on":
				value, haveValue = 1, true
			case "off":
				value, haveValue = 0, true
			}
		} else if n, ok := tk.GetNumber(token, false, true); ok {
			value, haveValue = n.Value, true
		}
	} else if len(tokens) == 1 {
		value, haveValue = 1, true
	}

	if len(tokens) == 1 {
		if s, ok := tokens[0].(*syntax.StringToken); ok && utf8.RuneCountInString(s.Value) == 4 {
			printable := true
			for _, r := range s.Value {
				if r < 0x20 || r > 0x7f {
					printable = false
				}
			}
			if printable {
				feature, haveFeature = s.Value, true
			}
		}
	}
	if haveFeature && haveValue {
		return props.FontSetting{Tag: feature, Value: value}, true, nil
	}
	return props.FontSetting{}, false, nil
}

func validateFontVariationSettings(tokens []syntax.Node, baseURL string) (props.Value, error) {
	if len(tokens) == 1 && tk.GetKeyword(tokens[0]) == "normal" {
		return props.FontSettings{Normal: true}, nil
	}
	items, ok, err := parseCommaList(tokens, baseURL, func(tokens []syntax.Node, _ string) (props.FontSetting, bool, error) {
		if len(tokens) == 2 {
			key, okKey := tokens[0].(*syntax.StringToken)
			value, okValue := tokens[1].(*syntax.NumberToken)
			if okKey && okValue {
				return props.FontSetting{Tag: key.Value, Value: value.Value}, true, nil
			}
		}
		return props.FontSetting{}, false, nil
	})
	if err != nil || !ok {
		return nil, err
	}
	return props.FontSettings{Items: items}, nil
}

func validateFontSize(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	k := props.Keyword(tk.GetKeyword(token))
	if _, ok := props.FontSizeKeywords[k]; ok || k == "smaller" || k == "larger" {
		return props.LengthKeyword(k)
	}
	return nil
}

func validateFontWeight(token syntax.Node) props.Value {
	switch k := tk.GetKeyword(token); k {
	case "normal", "bold", "bolder", "lighter":
		return props.Keyword(k)
	}
	if n, ok := token.(*syntax.NumberToken); ok && n.IsInteger {
		switch n.IntValue {
		case 100, 200, 300, 400, 500, 600, 700, 800, 900:
			return props.Integer(n.IntValue)
		}
	}
	return nil
}
