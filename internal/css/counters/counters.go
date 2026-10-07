package counters

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
)

// Style holds the descriptors of a counter style (CSS Counter Styles 3 §3).
// A nil or empty field is a descriptor that was not given, which an
// extending style takes from the style it extends.
type Style struct {
	System          *validation.CounterSystem
	Negative        []validation.CounterSymbol
	Prefix, Suffix  *validation.CounterSymbol
	Range           []validation.CounterRange
	Pad             *validation.CounterPad
	Fallback        string
	Symbols         []validation.CounterSymbol
	AdditiveSymbols []validation.CounterPad
}

// system returns the extended style or the algorithm of the style, with the
// first value of a fixed system. A style without system is symbolic.
func (s *Style) system() (extends bool, name string, fixed int) {
	if s.System == nil {
		return false, "symbolic", 0
	}
	if s.System.Kind == "extends" {
		return true, s.System.Extends, 0
	}
	return false, s.System.Kind, s.System.Number
}

// inherit fills the descriptors that s does not set from other.
func (s *Style) inherit(other *Style) {
	if s.Negative == nil {
		s.Negative = other.Negative
	}
	if s.Prefix == nil {
		s.Prefix = other.Prefix
	}
	if s.Suffix == nil {
		s.Suffix = other.Suffix
	}
	if s.Range == nil {
		s.Range = other.Range
	}
	if s.Pad == nil {
		s.Pad = other.Pad
	}
	if s.Fallback == "" {
		s.Fallback = other.Fallback
	}
	if s.Symbols == nil {
		s.Symbols = other.Symbols
	}
	if s.AdditiveSymbols == nil {
		s.AdditiveSymbols = other.AdditiveSymbols
	}
}

// Registry maps counter style names to their definitions. Names are case
// sensitive, except for the predefined decimal and disc (CSS Counter
// Styles 3 §3.1).
type Registry map[string]*Style

// Copy returns a registry holding the same styles. The styles themselves are
// never modified, so they are shared.
func (r Registry) Copy() Registry {
	c := make(Registry, len(r))
	for k, v := range r {
		c[k] = v
	}
	return c
}

// ParseName returns the name an @counter-style prelude defines, or false if
// the name is invalid: none, or decimal and disc once they are defined.
func ParseName(prelude []syntax.Node, r Registry) (string, bool) {
	tokens := tk.RemoveWhitespace(prelude)
	if len(tokens) != 1 {
		return "", false
	}
	ident, ok := tokens[0].(*syntax.IdentToken)
	if !ok {
		return "", false
	}
	switch ident.LowerValue {
	case "decimal", "disc":
		if _, defined := r[ident.LowerValue]; defined {
			return "", false
		}
		return ident.Value, true
	case "none":
		return "", false
	}
	return ident.Value, true
}

// Named returns the list-style-type value referring to a counter style by
// name.
func Named(name string) props.ListStyleType {
	return props.ListStyleType{Kind: props.ListStyleKeyword, Name: name}
}

func stringSymbol(s string) validation.CounterSymbol {
	return validation.CounterSymbol{Kind: "string", Value: s}
}

func symbolText(s *validation.CounterSymbol, fallback string) string {
	if s == nil {
		return fallback
	}
	if s.Kind == "string" {
		return s.Value
	}
	return ""
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Resolve returns the definition of a counter style, with the descriptors
// of the styles it extends filled in, or nil when the name is unknown or
// refers back to a style in visited. A string or a symbols() function
// defines an anonymous style (CSS Counter Styles 3 §6).
func (r Registry) Resolve(t props.ListStyleType) *Style {
	visited := []string{}
	return r.resolve(t, &visited)
}

func (r Registry) resolve(t props.ListStyleType, visited *[]string) *Style {
	switch t.Kind {
	case props.ListStyleString:
		suffix := stringSymbol("")
		return &Style{
			System:   &validation.CounterSystem{Kind: "cyclic"},
			Negative: []validation.CounterSymbol{stringSymbol("-"), stringSymbol("")},
			Prefix:   &validation.CounterSymbol{Kind: "string"},
			Suffix:   &suffix,
			Range:    []validation.CounterRange{{Auto: true}},
			Pad:      &validation.CounterPad{Symbol: stringSymbol("")},
			Fallback: "decimal",
			Symbols:  []validation.CounterSymbol{stringSymbol(t.Name)},
		}
	case props.ListStyleSymbols:
		system := &validation.CounterSystem{Kind: t.Name}
		if t.Name == "fixed" {
			system.Number = 1
		}
		symbols := make([]validation.CounterSymbol, len(t.Symbols))
		for i, s := range t.Symbols {
			symbols[i] = stringSymbol(s)
		}
		suffix := stringSymbol(" ")
		return &Style{
			System:          system,
			Negative:        []validation.CounterSymbol{stringSymbol("-"), stringSymbol("")},
			Prefix:          &validation.CounterSymbol{Kind: "string"},
			Suffix:          &suffix,
			Range:           []validation.CounterRange{{Auto: true}},
			Pad:             &validation.CounterPad{Symbol: stringSymbol("")},
			Fallback:        "decimal",
			Symbols:         symbols,
			AdditiveSymbols: []validation.CounterPad{},
		}
	}

	name := t.Name
	defined, ok := r[name]
	if !ok || contains(*visited, name) {
		return nil
	}
	*visited = append(*visited, name)

	style := *defined
	extends, system, _ := style.system()
	for extends {
		extended, ok := r[system]
		if !ok {
			return &style
		}
		style.System = extended.System
		*visited = append(*visited, system)
		extends, system, _ = style.system()
		if extends && contains(*visited, system) {
			// An extension loop falls back to decimal (CSS Counter Styles 3
			// §3.1.7).
			extends, system = true, "decimal"
			continue
		}
		style.inherit(extended)
	}
	return &style
}

// RenderValue returns the representation of value in the counter style t,
// without prefix and suffix (CSS Counter Styles 3 §11, "generate a counter
// representation").
func (r Registry) RenderValue(value int, t props.ListStyleType) string {
	return r.render(value, t.Name, r.Resolve(t), nil)
}

// RenderValueOf is RenderValue for a style given by name.
func (r Registry) RenderValueOf(value int, name string) string {
	return r.RenderValue(value, Named(name))
}

func (r Registry) renderDecimal(value int) string {
	return r.render(value, "decimal", r.Resolve(Named("decimal")), nil)
}

// render generates the representation of value with style, whose name is
// recorded in visited to stop fallback loops. A nil visited starts a new
// chain.
func (r Registry) render(value int, name string, style *Style, visited *[]string) string {
	if style == nil {
		if _, ok := r["decimal"]; ok {
			return r.renderDecimal(value)
		}
		// Only possible without the user agent stylesheet.
		return ""
	}
	extends, system, fixed := style.system()

	if visited == nil {
		visited = &[]string{}
	} else if contains(*visited, system) {
		return r.renderDecimal(value)
	}
	*visited = append(*visited, name)

	for extends {
		extended, ok := r[system]
		if !ok {
			return r.renderDecimal(value)
		}
		style.System = extended.System
		extends, system, fixed = style.system()
		if contains(*visited, system) {
			return r.renderDecimal(value)
		}
		*visited = append(*visited, system)
		style.inherit(extended)
	}

	fallback := func(v int) string {
		f := style.Fallback
		if f == "" {
			f = "decimal"
		}
		return r.render(v, f, r.resolve(Named(f), visited), visited)
	}

	if !inRange(value, system, style.Range) {
		return fallback(value)
	}

	isNegative := value < 0
	useNegative := false
	var negativePrefix, negativeSuffix string
	if isNegative {
		negative := style.Negative
		if negative == nil {
			negative = []validation.CounterSymbol{stringSymbol("-"), stringSymbol("")}
		}
		negativePrefix = symbolText(&negative[0], "")
		negativeSuffix = symbolText(&negative[1], "")
		switch system {
		case "symbolic", "alphabetic", "numeric", "additive":
			useNegative = true
			value = -value
		}
	}

	symbols := style.Symbols
	var initial string
	switch system {
	case "cyclic":
		if len(symbols) < 1 {
			return r.renderDecimal(value)
		}
		initial = symbolText(&symbols[floorMod(value-1, len(symbols))], "")
	case "fixed":
		if len(symbols) < 1 {
			return r.renderDecimal(value)
		}
		index := value - fixed
		if index < 0 || index >= len(symbols) {
			return fallback(value)
		}
		initial = symbolText(&symbols[index], "")
	case "symbolic":
		if len(symbols) < 1 {
			return r.renderDecimal(value)
		}
		n := len(symbols)
		index := floorMod(value-1, n)
		repeat := floorDiv(value-1, n) + 1
		initial = strings.Repeat(symbolText(&symbols[index], ""), max(repeat, 0))
	case "alphabetic":
		n := len(symbols)
		if n < 2 {
			return r.renderDecimal(value)
		}
		var parts []string
		for value != 0 {
			value--
			parts = append(parts, symbolText(&symbols[floorMod(value, n)], ""))
			value = floorDiv(value, n)
		}
		initial = joinReversed(parts)
	case "numeric":
		if value == 0 {
			if len(symbols) == 0 {
				return r.renderDecimal(value)
			}
			initial = symbolText(&symbols[0], "")
			break
		}
		n := len(symbols)
		if n < 2 {
			return r.renderDecimal(value)
		}
		if value < 0 {
			value = -value
		}
		var parts []string
		for value != 0 {
			parts = append(parts, symbolText(&symbols[value%n], ""))
			value /= n
		}
		initial = joinReversed(parts)
	case "additive":
		found := false
		if value == 0 {
			for _, s := range style.AdditiveSymbols {
				if s.Width == 0 {
					initial, found = symbolText(&s.Symbol, ""), true
				}
			}
		} else {
			if len(style.AdditiveSymbols) < 1 {
				return r.renderDecimal(value)
			}
			var b strings.Builder
			for _, s := range style.AdditiveSymbols {
				if s.Width == 0 {
					continue
				}
				repetitions := floorDiv(value, s.Width)
				b.WriteString(strings.Repeat(symbolText(&s.Symbol, ""), max(repetitions, 0)))
				value -= s.Width * repetitions
				if value == 0 {
					initial, found = b.String(), true
					break
				}
			}
		}
		if !found {
			// The remainder, not the original value, goes to the fallback.
			return fallback(value)
		}
	default:
		return r.renderDecimal(value)
	}

	pad := style.Pad
	if pad == nil {
		pad = &validation.CounterPad{Symbol: stringSymbol("")}
	}
	difference := pad.Width - utf8.RuneCountInString(initial)
	if isNegative && useNegative {
		difference -= utf8.RuneCountInString(negativePrefix) + utf8.RuneCountInString(negativeSuffix)
	}
	if difference > 0 {
		initial = strings.Repeat(symbolText(&pad.Symbol, ""), difference) + initial
	}
	if isNegative && useNegative {
		initial = negativePrefix + initial + negativeSuffix
	}
	return initial
}

// inRange reports whether value is in the range of the style, the default
// range of the system being used when the descriptor is auto or missing.
func inRange(value int, system string, ranges []validation.CounterRange) bool {
	auto := len(ranges) == 0
	for _, r := range ranges {
		if r.Auto {
			auto = true
		}
	}
	if auto {
		low := math.Inf(-1)
		switch system {
		case "alphabetic", "symbolic":
			low = 1
		case "additive":
			low = 0
		}
		return float64(value) >= low
	}
	for _, r := range ranges {
		if r.Lower <= float64(value) && float64(value) <= r.Upper {
			return true
		}
	}
	return false
}

// RenderMarker returns the content of a ::marker for value in the counter
// style t: the representation with the prefix and suffix of the style, the
// suffix defaulting to ". ".
func (r Registry) RenderMarker(t props.ListStyleType, value int) string {
	style := r.Resolve(t)
	if style == nil {
		if _, ok := r["decimal"]; ok {
			return r.RenderMarker(Named("decimal"), value)
		}
		return ""
	}
	return symbolText(style.Prefix, "") + r.RenderValue(value, t) + symbolText(style.Suffix, ". ")
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int) int { return a - floorDiv(a, b)*b }

func joinReversed(parts []string) string {
	var b strings.Builder
	for i := len(parts) - 1; i >= 0; i-- {
		b.WriteString(parts[i])
	}
	return b.String()
}
