package counters

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
)

func symbols(values ...string) []validation.CounterSymbol {
	out := make([]validation.CounterSymbol, len(values))
	for i, v := range values {
		out[i] = stringSymbol(v)
	}
	return out
}

func smallRegistry() Registry {
	suffix := stringSymbol(" ")
	return Registry{
		"decimal": {
			System:  &validation.CounterSystem{Kind: "numeric"},
			Symbols: symbols("0", "1", "2", "3", "4", "5", "6", "7", "8", "9"),
		},
		"disc": {
			System:  &validation.CounterSystem{Kind: "cyclic"},
			Symbols: symbols("•"),
			Suffix:  &suffix,
		},
		"lower-roman": {
			System: &validation.CounterSystem{Kind: "additive"},
			Range:  []validation.CounterRange{{Lower: 1, Upper: 3999}},
			AdditiveSymbols: []validation.CounterPad{
				{Width: 1000, Symbol: stringSymbol("m")}, {Width: 900, Symbol: stringSymbol("cm")},
				{Width: 500, Symbol: stringSymbol("d")}, {Width: 400, Symbol: stringSymbol("cd")},
				{Width: 100, Symbol: stringSymbol("c")}, {Width: 90, Symbol: stringSymbol("xc")},
				{Width: 50, Symbol: stringSymbol("l")}, {Width: 40, Symbol: stringSymbol("xl")},
				{Width: 10, Symbol: stringSymbol("x")}, {Width: 9, Symbol: stringSymbol("ix")},
				{Width: 5, Symbol: stringSymbol("v")}, {Width: 4, Symbol: stringSymbol("iv")},
				{Width: 1, Symbol: stringSymbol("i")},
			},
		},
		"decimal-leading-zero": {
			System: &validation.CounterSystem{Kind: "extends", Extends: "decimal"},
			Pad:    &validation.CounterPad{Width: 2, Symbol: stringSymbol("0")},
		},
	}
}

func TestRenderValueSystems(t *testing.T) {
	r := smallRegistry()
	tests := []struct {
		style string
		value int
		want  string
	}{
		{"decimal", 0, "0"},
		{"decimal", 42, "42"},
		{"decimal", -7, "-7"},
		{"lower-roman", 1986, "mcmlxxxvi"},
		{"lower-roman", 4000, "4000"},
		{"lower-roman", 0, "0"},
		{"decimal-leading-zero", 3, "03"},
		{"decimal-leading-zero", -1, "-1"},
		{"decimal-leading-zero", -11, "-11"},
		{"disc", 3, "•"},
		{"unknown", 5, "5"},
	}
	for _, tt := range tests {
		if got := r.RenderValueOf(tt.value, tt.style); got != tt.want {
			t.Errorf("RenderValueOf(%d, %s) = %q, want %q", tt.value, tt.style, got, tt.want)
		}
	}
}

func TestAnonymousStyles(t *testing.T) {
	r := smallRegistry()
	sym := func(kind string, s ...string) props.ListStyleType {
		return props.ListStyleType{Kind: props.ListStyleSymbols, Name: kind, Symbols: s}
	}
	tests := []struct {
		t      props.ListStyleType
		values []string
	}{
		{sym("cyclic", "a", "b", "c"), []string{"a ", "b ", "c ", "a "}},
		{sym("symbolic", "a", "b"), []string{"a ", "b ", "aa ", "bb "}},
		{sym("alphabetic", "a", "b"), []string{"a ", "b ", "aa ", "ab "}},
		{sym("fixed", "a", "b"), []string{"a ", "b ", "3 ", "4 "}},
		{sym("numeric", "0", "1", "2"), []string{"1 ", "2 ", "10 ", "11 "}},
		{props.ListStyleType{Kind: props.ListStyleString, Name: "/"}, []string{"/", "/", "/", "/"}},
		{Named("decimal"), []string{"1. ", "2. ", "3. ", "4. "}},
	}
	for _, tt := range tests {
		for i, want := range tt.values {
			if got := r.RenderMarker(tt.t, i+1); got != want {
				t.Errorf("RenderMarker(%v, %d) = %q, want %q", tt.t, i+1, got, want)
			}
		}
	}
}

func TestParseName(t *testing.T) {
	r := Registry{"decimal": {}}
	tests := []struct {
		css  string
		want string
		ok   bool
	}{
		{"foo", "foo", true},
		{" Foo ", "Foo", true},
		{"decimal", "", false},
		{"disc", "disc", true},
		{"none", "", false},
		{"a b", "", false},
		{`"a"`, "", false},
	}
	for _, tt := range tests {
		got, ok := ParseName(syntax.ParseComponentValueList(tt.css, false), r)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseName(%q) = %q, %v; want %q, %v", tt.css, got, ok, tt.want, tt.ok)
		}
	}
}
