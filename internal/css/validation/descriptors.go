package validation

import (
	"math"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Descriptor is a validated descriptor of an at-rule such as @font-face. The
// type of Value depends on the descriptor and is documented with it:
//
//	font-family, fallback: string
//	src (font-face): []FontSource
//	font-style, font-stretch: props.Keyword
//	font-weight: props.Keyword (normal or bold) or props.Integer
//	font-feature-settings: props.FontSettings
//	font-variant: []props.Value pairs, as FontVariant
//	unicode-range: []UnicodeRange
//	system: CounterSystem
//	negative: []CounterSymbol, prefix, suffix: CounterSymbol
//	range: []CounterRange
//	pad: CounterPad
//	symbols: []CounterSymbol
//	additive-symbols: []CounterPad
//	src (color-profile): urls.Link
//	rendering-intent: string
//	components: [][]string
type Descriptor struct {
	Name  string
	Value any
}

// FontSource is an item of the src descriptor of @font-face: a local font
// name, which is empty for local() without a name, or a link to a font file.
type FontSource struct {
	Local bool
	Name  string
	Link  urls.Link
}

// FontVariant is one longhand set by the font-variant descriptor.
type FontVariant struct {
	Name  string
	Value props.Value
}

// UnicodeRange is an inclusive range of code points.
type UnicodeRange struct{ Start, End int }

// CounterSystem is the system descriptor of @counter-style. For an extending
// system Kind is "extends" and Extends is the extended style; for "fixed"
// Number is the first symbol value.
type CounterSystem struct {
	Kind    string
	Extends string
	Number  int
}

// CounterSymbol is a string or an image of a counter style.
type CounterSymbol struct {
	// Kind is "string" or "url".
	Kind  string
	Value string
	Link  urls.Link
}

// CounterRange is an item of the range descriptor: auto, or inclusive bounds
// that may be infinite.
type CounterRange struct {
	Auto         bool
	Lower, Upper float64
}

// CounterPad is the value of pad, and an item of additive-symbols, where
// Width is the weight of the symbol.
type CounterPad struct {
	Width  int
	Symbol CounterSymbol
}

type descriptorValidator = func(tokens []syntax.Node, baseURL string) (any, error)

// descriptors maps the name of an at-rule to its descriptors.
var descriptors = map[string]map[string]descriptorValidator{
	"font-face":     {},
	"counter-style": {},
	"color-profile": {},
}

// notPrintDescriptors are ignored without a warning.
var notPrintDescriptors = setOf("font-display")

func registerDescriptor(rule string, fn descriptorValidator, names ...string) {
	for _, name := range names {
		descriptors[rule][name] = fn
	}
}

// PreprocessDescriptors validates the descriptors of an at-rule, which is
// "font-face", "counter-style" or "color-profile". Unsupported and invalid
// descriptors are ignored with a warning.
func PreprocessDescriptors(rule, baseURL string, declarations []syntax.Node) []Descriptor {
	var out []Descriptor
	for _, node := range declarations {
		d, ok := node.(*syntax.Declaration)
		if !ok || d.Important {
			continue
		}
		if notPrintDescriptors[d.Name] {
			continue
		}
		tokens := tk.RemoveWhitespace(d.Value)
		value, err := validateDescriptor(rule, d.Name, tokens, baseURL)
		if err != nil {
			line, column := d.Pos()
			logging.Warnf("Ignored `%s:%s` at %d:%d, %s.", d.Name, syntax.Serialize(d.Value), line, column, err.Error())
			continue
		}
		out = append(out, Descriptor{Name: d.Name, Value: value})
	}
	return out
}

func validateDescriptor(rule, name string, tokens []syntax.Node, baseURL string) (any, error) {
	fn, ok := descriptors[rule][name]
	if !ok {
		return nil, tk.Invalid("descriptor not supported")
	}
	value, err := fn(tokens, baseURL)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, tk.ErrInvalid
	}
	return value, nil
}

func init() {
	registerDescriptor("font-face", fontFaceFamily, "font-family")
	registerDescriptor("font-face", fontFaceSrc, "src")
	registerDescriptor("font-face", func(t []syntax.Node, b string) (any, error) {
		return asAny(keywordOf("normal", "italic", "oblique")(t, b))
	}, "font-style")
	registerDescriptor("font-face", fontFaceWeight, "font-weight")
	registerDescriptor("font-face", func(t []syntax.Node, b string) (any, error) {
		return asAny(keywordOf(fontStretchKeywords...)(t, b))
	}, "font-stretch")
	registerDescriptor("font-face", func(t []syntax.Node, b string) (any, error) {
		return asAny(validateFontFeatureSettings(t, b))
	}, "font-feature-settings")
	registerDescriptor("font-face", fontFaceVariant, "font-variant")
	registerDescriptor("font-face", fontFaceUnicodeRange, "unicode-range")

	registerDescriptor("counter-style", counterSystem, "system")
	registerDescriptor("counter-style", counterNegative, "negative")
	registerDescriptor("counter-style", counterPrefixSuffix, "prefix", "suffix")
	registerDescriptor("counter-style", counterRange, "range")
	registerDescriptor("counter-style", counterPad, "pad")
	registerDescriptor("counter-style", counterFallback, "fallback")
	registerDescriptor("counter-style", counterSymbols, "symbols")
	registerDescriptor("counter-style", counterAdditiveSymbols, "additive-symbols")

	registerDescriptor("color-profile", colorProfileSrc, "src")
	registerDescriptor("color-profile", func(t []syntax.Node, b string) (any, error) {
		v, err := keywordOf("relative-colorimetric", "absolute-colorimetric", "perceptual", "saturation")(t, b)
		if v == nil || err != nil {
			return nil, err
		}
		return string(v.(props.Keyword)), nil
	}, "rendering-intent")
	registerDescriptor("color-profile", colorProfileComponents, "components")
}

// asAny converts the result of a property validator, keeping a nil value
// untyped.
func asAny(v props.Value, err error) (any, error) {
	if v == nil || err != nil {
		return nil, err
	}
	return v, nil
}

// fontFamilyName validates the family of @font-face and of local(), which
// also allows whitespace between the identifiers.
func fontFamilyName(tokens []syntax.Node, allowSpaces bool) (string, bool) {
	if len(tokens) == 1 {
		if s, ok := tokens[0].(*syntax.StringToken); ok {
			return s.Value, true
		}
	}
	if len(tokens) == 0 {
		return "", false
	}
	var names []string
	for _, token := range tokens {
		switch t := token.(type) {
		case *syntax.IdentToken:
			names = append(names, t.Value)
		case *syntax.WhitespaceToken:
			if !allowSpaces {
				return "", false
			}
		default:
			return "", false
		}
	}
	return strings.Join(names, " "), true
}

func fontFaceFamily(tokens []syntax.Node, _ string) (any, error) {
	if name, ok := fontFamilyName(tokens, false); ok {
		return name, nil
	}
	return nil, nil
}

func fontFaceSrc(tokens []syntax.Node, baseURL string) (any, error) {
	sources, ok, err := parseCommaList(tokens, baseURL, func(tokens []syntax.Node, baseURL string) (FontSource, bool, error) {
		if len(tokens) != 1 && len(tokens) != 2 {
			return FontSource{}, false, nil
		}
		token := tokens[len(tokens)-1]
		tokens = tokens[:len(tokens)-1]
		if fn, ok := token.(*syntax.FunctionBlock); ok && fn.LowerName == "format" {
			if len(tokens) == 0 {
				return FontSource{}, false, nil
			}
			token = tokens[len(tokens)-1]
		}
		if fn, ok := token.(*syntax.FunctionBlock); ok && fn.LowerName == "local" {
			name, _ := fontFamilyName(fn.Arguments, true)
			return FontSource{Local: true, Name: name}, true, nil
		}
		url, ok, err := tk.GetURL(token, baseURL)
		if err != nil || !ok || url.Attr != nil {
			return FontSource{}, false, err
		}
		return FontSource{Link: url.Link}, true, nil
	})
	if err != nil || !ok {
		return nil, err
	}
	return sources, nil
}

func fontFaceWeight(tokens []syntax.Node, _ string) (any, error) {
	if len(tokens) != 1 {
		return nil, nil
	}
	switch k := tk.GetKeyword(tokens[0]); k {
	case "normal", "bold":
		return props.Keyword(k), nil
	}
	if n, ok := tk.GetNumber(tokens[0], true, true); ok {
		switch n.Value {
		case 100, 200, 300, 400, 500, 600, 700, 800, 900:
			return props.Integer(int(n.Value)), nil
		}
	}
	return nil, nil
}

func fontFaceVariant(tokens []syntax.Node, _ string) (any, error) {
	if len(tokens) == 1 {
		switch tk.GetKeyword(tokens[0]) {
		case "normal", "none", "inherit":
			return []FontVariant{}, nil
		}
	}
	parts, err := expandFontVariant(tokens)
	if err != nil {
		return nil, err
	}
	values := []FontVariant{}
	for _, p := range parts {
		name := "font-variant" + p.name
		v, err := validateNonShorthand(p.tokens, name, "", true)
		if err != nil {
			return nil, nil
		}
		values = append(values, FontVariant{Name: name, Value: v})
	}
	return values, nil
}

func fontFaceUnicodeRange(tokens []syntax.Node, baseURL string) (any, error) {
	ranges, ok, err := parseCommaList(tokens, baseURL, func(tokens []syntax.Node, _ string) (UnicodeRange, bool, error) {
		if len(tokens) == 1 {
			if r, ok := tokens[0].(*syntax.UnicodeRangeToken); ok {
				return UnicodeRange{r.Start, r.End}, true, nil
			}
		}
		return UnicodeRange{}, false, nil
	})
	if err != nil || !ok {
		return nil, err
	}
	return ranges, nil
}

func counterSystem(tokens []syntax.Node, _ string) (any, error) {
	if len(tokens) > 2 || len(tokens) == 0 {
		return nil, nil
	}
	switch keyword := tk.GetKeyword(tokens[0]); keyword {
	case "extends":
		if len(tokens) == 2 {
			if second := tk.GetKeyword(tokens[1]); second != "" {
				return CounterSystem{Kind: "extends", Extends: second}, nil
			}
		}
	case "fixed":
		if len(tokens) == 1 {
			return CounterSystem{Kind: "fixed", Number: 1}, nil
		}
		if n, ok := tk.GetNumber(tokens[1], true, true); ok {
			return CounterSystem{Kind: "fixed", Number: int(n.Value)}, nil
		}
	case "cyclic", "numeric", "alphabetic", "symbolic", "additive":
		if len(tokens) == 1 {
			return CounterSystem{Kind: keyword}, nil
		}
	}
	return nil, nil
}

// counterSymbol parses a string, identifier or image. The boolean is false
// for any other token.
func counterSymbol(token syntax.Node, baseURL string) (CounterSymbol, bool, error) {
	switch t := token.(type) {
	case *syntax.StringToken:
		return CounterSymbol{Kind: "string", Value: t.Value}, true, nil
	case *syntax.IdentToken:
		return CounterSymbol{Kind: "string", Value: t.Value}, true, nil
	}
	url, ok, err := tk.GetURL(token, baseURL)
	if err != nil || !ok || url.Attr != nil {
		return CounterSymbol{}, false, err
	}
	return CounterSymbol{Kind: "url", Link: url.Link}, true, nil
}

func counterNegative(tokens []syntax.Node, baseURL string) (any, error) {
	if len(tokens) > 2 {
		return nil, nil
	}
	var values []CounterSymbol
	for _, token := range tokens {
		s, ok, err := counterSymbol(token, baseURL)
		if err != nil {
			return nil, err
		}
		if ok {
			values = append(values, s)
		}
	}
	if len(values) == 1 {
		values = append(values, CounterSymbol{Kind: "string"})
	}
	if len(values) == 2 {
		return values, nil
	}
	return nil, nil
}

func counterPrefixSuffix(tokens []syntax.Node, baseURL string) (any, error) {
	if len(tokens) != 1 {
		return nil, nil
	}
	s, ok, err := counterSymbol(tokens[0], baseURL)
	if err != nil || !ok {
		return nil, err
	}
	return s, nil
}

func counterRange(tokens []syntax.Node, baseURL string) (any, error) {
	ranges, ok, err := parseCommaList(tokens, baseURL, func(tokens []syntax.Node, _ string) (CounterRange, bool, error) {
		switch len(tokens) {
		case 1:
			if tk.GetSingleKeyword(tokens) == "auto" {
				return CounterRange{Auto: true}, true, nil
			}
		case 2:
			var values []float64
			for i, token := range tokens {
				if ident, ok := token.(*syntax.IdentToken); ok && ident.Value == "infinite" {
					if i == 1 {
						values = append(values, math.Inf(1))
					} else {
						values = append(values, math.Inf(-1))
					}
				} else if n, ok := tk.GetNumber(token, true, true); ok {
					values = append(values, n.Value)
				}
			}
			if len(values) == 2 && values[0] <= values[1] {
				return CounterRange{Lower: values[0], Upper: values[1]}, true, nil
			}
		}
		return CounterRange{}, false, nil
	})
	if err != nil || !ok {
		return nil, err
	}
	return ranges, nil
}

func parseCounterPad(tokens []syntax.Node, baseURL string) (CounterPad, bool, error) {
	if len(tokens) != 2 {
		return CounterPad{}, false, nil
	}
	var pad CounterPad
	haveWidth, haveSymbol := false, false
	for _, token := range tokens {
		if n, ok := tk.GetNumber(token, false, true); ok {
			if !haveWidth {
				pad.Width, haveWidth = int(n.Value), true
			}
		} else if s, ok, _ := counterSymbol(token, baseURL); ok && s.Kind == "string" {
			pad.Symbol, haveSymbol = s, true
		}
		url, ok, err := tk.GetURL(token, baseURL)
		if err != nil {
			return CounterPad{}, false, err
		}
		if ok && url.Attr == nil {
			pad.Symbol, haveSymbol = CounterSymbol{Kind: "url", Link: url.Link}, true
		}
	}
	return pad, haveWidth && haveSymbol, nil
}

func counterPad(tokens []syntax.Node, baseURL string) (any, error) {
	pad, ok, err := parseCounterPad(tokens, baseURL)
	if err != nil || !ok {
		return nil, err
	}
	return pad, nil
}

func counterFallback(tokens []syntax.Node, _ string) (any, error) {
	if len(tokens) != 1 {
		return nil, nil
	}
	if name := tk.GetCustomIdent(tokens[0]); name != "" && name != "none" {
		return name, nil
	}
	return nil, nil
}

func counterSymbols(tokens []syntax.Node, baseURL string) (any, error) {
	values := []CounterSymbol{}
	for _, token := range tokens {
		s, ok, err := counterSymbol(token, baseURL)
		if err != nil || !ok {
			return nil, err
		}
		values = append(values, s)
	}
	return values, nil
}

func counterAdditiveSymbols(tokens []syntax.Node, baseURL string) (any, error) {
	var results []CounterPad
	for _, part := range tk.SplitOnComma(tokens) {
		pad, ok, err := parseCounterPad(tk.RemoveWhitespace(part), baseURL)
		if err != nil || !ok {
			return nil, err
		}
		if len(results) > 0 && results[len(results)-1].Width <= pad.Width {
			return nil, nil
		}
		results = append(results, pad)
	}
	return results, nil
}

func colorProfileSrc(tokens []syntax.Node, baseURL string) (any, error) {
	if len(tokens) != 1 {
		return nil, nil
	}
	url, ok, err := tk.GetURL(tokens[0], baseURL)
	if err != nil || !ok || url.Attr != nil {
		return nil, err
	}
	return url.Link, nil
}

func colorProfileComponents(tokens []syntax.Node, baseURL string) (any, error) {
	components, ok, err := parseCommaList(tokens, baseURL, func(tokens []syntax.Node, _ string) ([]string, bool, error) {
		names := []string{}
		for _, token := range tokens {
			ident, ok := token.(*syntax.IdentToken)
			if !ok {
				return nil, false, nil
			}
			names = append(names, ident.Value)
		}
		return names, true, nil
	})
	if err != nil || !ok {
		return nil, err
	}
	return components, nil
}
