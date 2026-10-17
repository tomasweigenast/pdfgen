package fonts

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func testSpec(size float64) Spec {
	return Spec{Font: Description{Families: []string{"testfont"}, Weight: 400, Size: size}}
}

func newMeasureConfiguration(t testing.TB) *FontConfiguration {
	t.Helper()
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	return fc
}

func TestStrutOfSpec(t *testing.T) {
	fc := newMeasureConfiguration(t)
	const baseline = 12.796875
	tests := []struct {
		name         string
		size         float64
		lineHeight   LineHeight
		height, base float64
	}{
		{"normal", 16, LineHeight{Normal: true}, 16, baseline},
		{"number", 16, LineHeight{Number: true, Value: 1.5}, 24, baseline + 4},
		{"length", 16, LineHeight{Value: 20}, 20, baseline + 2},
		{"length smaller than the font", 16, LineHeight{Value: 10}, 10, baseline - 3},
		{"number follows the font size", 10, LineHeight{Number: true, Value: 2}, 20, 7.998046875 + 5},
		{"zero font size", 0, LineHeight{Normal: true}, 0, 0},
		{"zero font size with a length", 0, LineHeight{Value: 20}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, b := fc.Strut(testSpec(tt.size), tt.lineHeight)
			if h != tt.height || b != tt.base {
				t.Errorf("Strut = %v, %v; want %v, %v", h, b, tt.height, tt.base)
			}
		})
	}
}

func TestStrutWithoutFontsHasNoHeight(t *testing.T) {
	fc := newTestConfiguration(t)
	spec := Spec{Font: Description{Families: []string{"none"}, Size: 16}}
	if h, b := fc.Strut(spec, LineHeight{Normal: true}); h != 0 || b != 0 {
		t.Errorf("Strut = %v, %v; want 0, 0", h, b)
	}
	if h, b := fc.Strut(spec, LineHeight{Value: 20}); h != 20 || b != 10 {
		t.Errorf("Strut with a length = %v, %v; want 20, 10", h, b)
	}
}

func TestStrutMetricsAreReused(t *testing.T) {
	fc := newMeasureConfiguration(t)
	base := testSpec(16)
	fc.Strut(base, LineHeight{Normal: true})
	if n := len(fc.metrics); n != 1 {
		t.Fatalf("%d cached metrics after the first strut, want 1", n)
	}

	// Properties that do not change the font share its metrics.
	fc.Strut(base, LineHeight{Number: true, Value: 3})
	fc.Strut(base, LineHeight{Value: 40})
	kerning := base
	kerning.Features.Kerning = "none"
	fc.Strut(kerning, LineHeight{Normal: true})
	if n := len(fc.metrics); n != 1 {
		t.Errorf("%d cached metrics after changing line-height and font-kerning, want 1", n)
	}

	// The size, the family and the settings that select the font do not.
	for name, change := range map[string]func(*Spec){
		"font-size":    func(s *Spec) { s.Font.Size = 17 },
		"font-family":  func(s *Spec) { s.Font.Families = []string{"other", "testfont"} },
		"font-weight":  func(s *Spec) { s.Font.Weight = 700 },
		"font-style":   func(s *Spec) { s.Font.Style = "italic" },
		"font-stretch": func(s *Spec) { s.Font.Stretch = "condensed" },
		"variations": func(s *Spec) {
			s.Font.Variations = props.FontSettings{Items: []props.FontSetting{{Tag: "wght", Value: 500}}}
		},
		"features": func(s *Spec) { s.Features.Caps = "small-caps" },
		"feature list": func(s *Spec) {
			s.Features.Settings = props.FontSettings{Items: []props.FontSetting{{Tag: "liga", Value: 0}}}
		},
		"lang":          func(s *Spec) { s.Lang = "fr" },
		"lang override": func(s *Spec) { s.LanguageOverride = props.KeywordOrString{String: "TRK"} },
	} {
		before := len(fc.metrics)
		s := base
		change(&s)
		fc.Strut(s, LineHeight{Normal: true})
		if len(fc.metrics) != before+1 {
			t.Errorf("changing %s did not add metrics: %d entries, was %d", name, len(fc.metrics), before)
		}
	}
}

func TestStrutAgreesWithTheMetricsOfTheFont(t *testing.T) {
	fc := newMeasureConfiguration(t)
	spec := testSpec(13.333)
	face := fc.Query(spec.Font, "", ' ')
	if face == nil {
		t.Fatal("no face")
	}
	wantH, wantB := face.Metrics(13.333).Strut(nil)
	// Twice, the second time from the cache.
	for range 2 {
		if h, b := fc.Strut(spec, LineHeight{Normal: true}); h != wantH || b != wantB {
			t.Errorf("Strut = %v, %v; want %v, %v", h, b, wantH, wantB)
		}
	}
}

func TestFamilyListsDoNotShareCacheEntries(t *testing.T) {
	fc := newMeasureConfiguration(t)
	one, two := testSpec(16), testSpec(16)
	one.Font.Families = []string{"a b"}
	two.Font.Families = []string{"a", "b"}
	fc.Strut(one, LineHeight{Normal: true})
	fc.Strut(two, LineHeight{Normal: true})
	if n := len(fc.metrics); n != 2 {
		t.Errorf("%d cached metrics for [\"a b\"] and [\"a\", \"b\"], want 2", n)
	}
	fc.CharacterRatio(one, "ex")
	fc.CharacterRatio(two, "ex")
	if n := len(fc.ratios); n != 2 {
		t.Errorf("%d cached ratios for [\"a b\"] and [\"a\", \"b\"], want 2", n)
	}
}

func TestCharacterRatioOfSpec(t *testing.T) {
	fc := newMeasureConfiguration(t)
	tests := []struct {
		unit string
		want float64
	}{
		{"ex", 0.7998},
		{"ch", 1},
		{"cap", 1},
		{"ic", 2},
		{"em", 0},
	}
	for _, tt := range tests {
		// Twice, the second time from the cache.
		for range 2 {
			if got := fc.CharacterRatio(testSpec(16), tt.unit); got != tt.want {
				t.Errorf("CharacterRatio(%q) = %v, want %v", tt.unit, got, tt.want)
			}
		}
	}
	if got := fc.CharacterRatio(Spec{Font: Description{Families: []string{"none"}}}, "ex"); got != 0.5 {
		t.Errorf("without fonts: %v, want 0.5", got)
	}
}

func TestCharacterRatioIgnoresTheFontSize(t *testing.T) {
	fc := newMeasureConfiguration(t)
	for _, size := range []float64{1, 16, 80, 0} {
		if got := fc.CharacterRatio(testSpec(size), "ex"); got != 0.7998 {
			t.Errorf("size %v: ex = %v, want 0.7998", size, got)
		}
	}
	if n := len(fc.ratios); n != 1 {
		t.Errorf("%d cached ratios for one font at several sizes, want 1", n)
	}
}

func TestCharacterRatiosAreReused(t *testing.T) {
	fc := newMeasureConfiguration(t)
	base := testSpec(16)
	fc.CharacterRatio(base, "ex")
	fc.CharacterRatio(base, "ch")
	if n := len(fc.ratios); n != 2 {
		t.Fatalf("%d cached ratios for two units, want 2", n)
	}

	// A single glyph is never kerned, so the kerning of the style is not
	// part of the key.
	kerning := base
	kerning.Features.Kerning = "none"
	kerning.Font.Size = 40
	fc.CharacterRatio(kerning, "ex")
	if n := len(fc.ratios); n != 2 {
		t.Errorf("%d cached ratios after changing font-kerning and the size, want 2", n)
	}

	lang := base
	lang.Lang = "fr"
	fc.CharacterRatio(lang, "ex")
	if n := len(fc.ratios); n != 3 {
		t.Errorf("%d cached ratios after changing the language, want 3", n)
	}
}

func TestLanguageOverrideAndLangMeasureTheSameFont(t *testing.T) {
	fc := newMeasureConfiguration(t)
	a, b := testSpec(16), testSpec(16)
	a.Lang = "tur"
	b.LanguageOverride = props.KeywordOrString{String: "TRK"}
	if got, want := fc.CharacterRatio(a, "ex"), fc.CharacterRatio(b, "ex"); got != want {
		t.Errorf("ratios differ: %v and %v", got, want)
	}
}

func TestMeasuresFollowFontFacesAddedLater(t *testing.T) {
	fc := newTestConfiguration(t)
	spec := Spec{Font: Description{Families: []string{"later"}, Size: 16}}
	if h, _ := fc.Strut(spec, LineHeight{Normal: true}); h != 0 {
		t.Fatalf("strut height %v without the font, want 0", h)
	}
	if r := fc.CharacterRatio(spec, "ex"); r != 0.5 {
		t.Fatalf("ex = %v without the font, want 0.5", r)
	}
	addTestFont(t, fc, FontFaceRule{Family: "later"})
	if h, _ := fc.Strut(spec, LineHeight{Normal: true}); h != 16 {
		t.Errorf("strut height %v with the font, want 16", h)
	}
	if r := fc.CharacterRatio(spec, "ex"); r != 0.7998 {
		t.Errorf("ex = %v with the font, want 0.7998", r)
	}
}

func TestSpecLanguage(t *testing.T) {
	tests := []struct {
		name string
		spec Spec
		want string
	}{
		{"lang", Spec{Lang: "fr"}, "fr"},
		{"normal override", Spec{Lang: "fr", LanguageOverride: props.KeywordOrString{Keyword: "normal"}}, "fr"},
		{"override tag", Spec{Lang: "fr", LanguageOverride: props.KeywordOrString{String: "TRK"}}, "tur"},
		{"unknown override", Spec{Lang: "fr", LanguageOverride: props.KeywordOrString{String: "xx"}}, "xx"},
		{"none", Spec{}, ""},
	}
	for _, tt := range tests {
		if got := tt.spec.Language(); got != tt.want {
			t.Errorf("%s: Language() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSpecFeatures(t *testing.T) {
	spec := Spec{Features: FeatureStyle{
		Kerning:     "none",
		FontVariant: FontVariant{Caps: "small-caps"},
		Settings:    props.FontSettings{Items: []props.FontSetting{{Tag: "smcp", Value: 0}}},
	}}
	want := "kern 0,smcp 0"
	if got := FeatureString(spec.FontFeatures()); got != want {
		t.Errorf("features = %q, want %q", got, want)
	}
}

func TestLineHeightUsedValue(t *testing.T) {
	tests := []struct {
		lh   LineHeight
		want *float64
	}{
		{LineHeight{Normal: true}, nil},
		{LineHeight{Number: true, Value: 1.5}, ptr(24)},
		{LineHeight{Value: 30}, ptr(30)},
	}
	for _, tt := range tests {
		got := tt.lh.Used(16)
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%+v.Used(16) = %v, want %v", tt.lh, got, tt.want)
		}
	}
}

func ptr(v float64) *float64 { return &v }
