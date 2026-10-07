package fonts

import (
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func settings(items ...props.FontSetting) props.FontSettings {
	return props.FontSettings{Items: items}
}

func TestFeatures(t *testing.T) {
	tests := []struct {
		name  string
		style FeatureStyle
		want  []Feature
	}{
		{"kerning auto leaves kern to the font", FeatureStyle{Kerning: "auto"}, nil},
		{"kerning normal", FeatureStyle{Kerning: "normal"}, []Feature{{"kern", 1}}},
		{"unset kerning is normal", FeatureStyle{}, []Feature{{"kern", 1}}},
		{"kerning none", FeatureStyle{Kerning: "none"}, []Feature{{"kern", 0}}},
		{
			"ligatures none",
			FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Ligatures: props.Keywords{"none"}}},
			[]Feature{{"liga", 0}, {"clig", 0}, {"hlig", 0}, {"dlig", 0}, {"calt", 0}},
		},
		{
			"ligatures normal",
			FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Ligatures: props.Keywords{"normal"}}},
			nil,
		},
		{
			"common ligatures off, discretionary on",
			FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Ligatures: props.Keywords{"no-common-ligatures", "discretionary-ligatures"}}},
			[]Feature{{"liga", 0}, {"clig", 0}, {"dlig", 1}},
		},
		{
			"contextual off",
			FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Ligatures: props.Keywords{"no-contextual"}}},
			[]Feature{{"calt", 0}},
		},
		{"position sub", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Position: "sub"}}, []Feature{{"subs", 1}}},
		{"position super", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Position: "super"}}, []Feature{{"sups", 1}}},
		{"small caps", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "small-caps"}}, []Feature{{"smcp", 1}}},
		{"all small caps", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "all-small-caps"}}, []Feature{{"c2sc", 1}, {"smcp", 1}}},
		{"petite caps", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "petite-caps"}}, []Feature{{"pcap", 1}}},
		{"all petite caps", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "all-petite-caps"}}, []Feature{{"c2pc", 1}, {"pcap", 1}}},
		{"unicase", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "unicase"}}, []Feature{{"unic", 1}}},
		{"titling caps", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "titling-caps"}}, []Feature{{"titl", 1}}},
		{
			"numeric",
			FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Numeric: props.Keywords{"oldstyle-nums", "tabular-nums", "slashed-zero", "diagonal-fractions"}}},
			[]Feature{{"onum", 1}, {"tnum", 1}, {"zero", 1}, {"frac", 1}},
		},
		{"stacked fractions", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Numeric: props.Keywords{"stacked-fractions", "lining-nums", "proportional-nums"}}}, []Feature{{"afrc", 1}, {"lnum", 1}, {"pnum", 1}}},
		{"historical forms", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Alternates: "historical-forms"}}, []Feature{{"hist", 1}}},
		{"other alternates are not supported", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Alternates: "stylistic(1)"}}, nil},
		{"east asian", FeatureStyle{Kerning: "auto", FontVariant: FontVariant{EastAsian: props.Keywords{"full-width", "proportional-width"}}}, []Feature{{"fwid", 1}, {"pwid", 1}}},
		{
			"feature settings come last and override",
			FeatureStyle{Kerning: "normal", FontVariant: FontVariant{Caps: "small-caps"}, Settings: settings(
				props.FontSetting{Tag: "smcp", Value: 0}, props.FontSetting{Tag: "ss01", Value: 1})},
			[]Feature{{"kern", 1}, {"smcp", 0}, {"ss01", 1}},
		},
		{
			"overriding keeps the position of the first setting",
			FeatureStyle{Kerning: "normal", FontVariant: FontVariant{Ligatures: props.Keywords{"none"}}, Settings: settings(
				props.FontSetting{Tag: "liga", Value: 1})},
			[]Feature{{"kern", 1}, {"liga", 1}, {"clig", 0}, {"hlig", 0}, {"dlig", 0}, {"calt", 0}},
		},
		{"normal feature settings", FeatureStyle{Kerning: "auto", Settings: props.FontSettings{Normal: true}}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Features(tc.style)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Features = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFeatureString(t *testing.T) {
	got := FeatureString([]Feature{{"kern", 1}, {"liga", 0}, {"ss01", 3}})
	if want := "kern 1,liga 0,ss01 3"; got != want {
		t.Errorf("FeatureString = %q, want %q", got, want)
	}
	if got := FeatureString(nil); got != "" {
		t.Errorf("FeatureString(nil) = %q", got)
	}
}

func TestHBFeaturesSkipsMalformedTags(t *testing.T) {
	got := HBFeatures([]Feature{{"liga", 0}, {"toolong", 1}, {"ab", 1}, {"smcp", 1}})
	if len(got) != 2 || got[0].Value != 0 || got[1].Value != 1 {
		t.Errorf("HBFeatures = %+v", got)
	}
}

func TestLanguage(t *testing.T) {
	tests := []struct {
		name     string
		override props.KeywordOrString
		lang     string
		want     string
	}{
		{"normal uses lang", props.KeywordOrString{Keyword: "normal"}, "fr", "fr"},
		{"unset uses lang", props.KeywordOrString{}, "fr", "fr"},
		{"tag is mapped to ISO", props.KeywordOrString{String: "AFK"}, "fr", "afr"},
		{"mapped tag", props.KeywordOrString{String: "TRK"}, "", "tur"},
		{"lowercase lookup", props.KeywordOrString{String: "Far"}, "", "fas"},
		{"unknown tag passes through", props.KeywordOrString{String: "XYZ"}, "fr", "XYZ"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Language(tc.override, tc.lang); got != tc.want {
				t.Errorf("Language = %q, want %q", got, tc.want)
			}
		})
	}
}
