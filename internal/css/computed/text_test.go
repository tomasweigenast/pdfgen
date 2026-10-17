package computed

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
)

// testFontContext returns a context whose only font is the test font,
// registered as "testfont". Its glyphs are one em wide, and its ascent and
// descent are 819 and 205 units of a 1024 units em.
func testFontContext(t testing.TB) *Context {
	t.Helper()
	fc := fonts.NewFontConfiguration(fonts.WithoutSystemFonts())
	rule := fonts.FontFaceRule{Family: "testfont", Src: []fonts.Source{{Value: "testfont.otf"}}}
	fetch := func(name string) ([]byte, error) {
		return os.ReadFile(testutil.ResourcePath(t, name))
	}
	if err := fc.AddFontFace(rule, fetch); err != nil {
		t.Fatalf("AddFontFace: %v", err)
	}
	return NewContext(fc)
}

// lineBreakStyleOf returns the line break style of the last of nested
// elements whose declarations are css.
func lineBreakStyleOf(t testing.TB, css ...string) *linebreak.Style {
	t.Helper()
	s := familyIn(t, NewContext(nil), css...)
	return s[len(s)-1].LineBreakStyle()
}

// featuresOf returns the OpenType features a style selects, as a string.
func featuresOf(s *linebreak.Style) string { return fonts.FeatureString(fonts.Features(s.Features)) }

func TestInitialStyleMapsToTheInitialLineBreakStyle(t *testing.T) {
	got := New(nil).LineBreakStyle()
	want := &linebreak.Style{
		Font: fonts.Description{
			Families: []string{"serif"}, Weight: 400, Style: "normal", Stretch: "normal", Size: 16,
			Variations: props.FontSettings{Normal: true},
		},
		Features: fonts.FeatureStyle{
			Kerning: "auto",
			FontVariant: fonts.FontVariant{
				Ligatures: props.Keywords{"normal"}, Position: "normal", Caps: "normal",
				Numeric: props.Keywords{"normal"}, Alternates: "normal", EastAsian: props.Keywords{"normal"},
			},
			Settings: props.FontSettings{Normal: true},
		},
		LanguageOverride:    props.KeywordOrString{Keyword: "normal"},
		Direction:           "ltr",
		LineHeight:          linebreak.LineHeight{Normal: true},
		WhiteSpace:          "normal",
		WordBreak:           "normal",
		OverflowWrap:        "normal",
		Hyphens:             "manual",
		HyphenateCharacter:  "‐",
		HyphenateLimitChars: linebreak.HyphenateLimitChars{Total: 5, Before: 2, After: 2},
		TabSize:             linebreak.TabSize{Spaces: 8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("initial style maps to\n%+v\nwant\n%+v", got, want)
	}
}

// The values that the style of the line break package starts from are the
// initial values of the properties.
func TestInitialStyleAgreesWithNewLineBreakStyle(t *testing.T) {
	got := New(nil).LineBreakStyle()
	want := linebreak.NewStyle(16, "serif")
	if got.Font.Size != want.Font.Size || got.Font.Weight != want.Font.Weight ||
		!reflect.DeepEqual(got.Font.Families, want.Font.Families) {
		t.Errorf("font = %+v, want %+v", got.Font, want.Font)
	}
	for name, pair := range map[string][2]any{
		"direction":             {got.Direction, want.Direction},
		"letter-spacing":        {got.LetterSpacing, want.LetterSpacing},
		"word-spacing":          {got.WordSpacing, want.WordSpacing},
		"line-height":           {got.LineHeight, want.LineHeight},
		"white-space":           {got.WhiteSpace, want.WhiteSpace},
		"word-break":            {got.WordBreak, want.WordBreak},
		"overflow-wrap":         {got.OverflowWrap, want.OverflowWrap},
		"hyphens":               {got.Hyphens, want.Hyphens},
		"hyphenate-character":   {got.HyphenateCharacter, want.HyphenateCharacter},
		"hyphenate-limit-chars": {got.HyphenateLimitChars, want.HyphenateLimitChars},
		"tab-size":              {got.TabSize, want.TabSize},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %v, want %v", name, pair[0], pair[1])
		}
	}
	if got.HyphenateLimitZone.Resolve(100) != 0 {
		t.Errorf("hyphenate-limit-zone = %v, want 0", got.HyphenateLimitZone.Resolve(100))
	}
}

func TestFontPropertiesMapToTheFontDescription(t *testing.T) {
	tests := []struct {
		name string
		css  []string
		want fonts.Description
	}{
		{
			"font shorthand",
			[]string{`font: italic bold 20px/1.5 "A B", sans-serif`},
			fonts.Description{
				Families: []string{"A B", "sans-serif"}, Weight: 700, Style: "italic", Stretch: "normal", Size: 20,
				Variations: props.FontSettings{Normal: true},
			},
		},
		{
			"oblique and stretch",
			[]string{"font-style: oblique; font-stretch: ultra-condensed; font-weight: 100"},
			fonts.Description{
				Families: []string{"serif"}, Weight: 100, Style: "oblique", Stretch: "ultra-condensed", Size: 16,
				Variations: props.FontSettings{Normal: true},
			},
		},
		{
			"variation settings",
			[]string{`font-variation-settings: "wght" 650, "opsz" 12.5`},
			fonts.Description{
				Families: []string{"serif"}, Weight: 400, Style: "normal", Stretch: "normal", Size: 16,
				Variations: props.FontSettings{Items: []props.FontSetting{{Tag: "wght", Value: 650}, {Tag: "opsz", Value: 12.5}}},
			},
		},
		{
			"font size in relative units",
			[]string{"font-size: 20px", "font-size: 1.5em"},
			fonts.Description{
				Families: []string{"serif"}, Weight: 400, Style: "normal", Stretch: "normal", Size: 30,
				Variations: props.FontSettings{Normal: true},
			},
		},
		{
			"inherited from the parent",
			[]string{"font: bold 10px testfont", ""},
			fonts.Description{
				Families: []string{"testfont"}, Weight: 700, Style: "normal", Stretch: "normal", Size: 10,
				Variations: props.FontSettings{Normal: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lineBreakStyleOf(t, tt.css...).Font; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Font = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFontFeaturesMapToTheFeatureStyle(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want string
	}{
		{"auto kerning", "", ""},
		{"normal kerning", "font-kerning: normal", "kern 1"},
		{"no kerning", "font-kerning: none", "kern 0"},
		{"no common ligatures", "font-variant-ligatures: no-common-ligatures", "liga 0,clig 0"},
		{"ligatures", "font-variant-ligatures: discretionary-ligatures contextual", "dlig 1,calt 1"},
		{"no ligatures", "font-variant-ligatures: none", "liga 0,clig 0,hlig 0,dlig 0,calt 0"},
		{"position", "font-variant-position: super", "sups 1"},
		{"caps", "font-variant-caps: all-small-caps", "c2sc 1,smcp 1"},
		{"numeric", "font-variant-numeric: oldstyle-nums tabular-nums", "onum 1,tnum 1"},
		{"alternates", "font-variant-alternates: historical-forms", "hist 1"},
		{"east asian", "font-variant-east-asian: full-width", "fwid 1"},
		{"font-variant shorthand", "font-variant: small-caps slashed-zero", "smcp 1,zero 1"},
		{"feature settings", `font-feature-settings: "liga" 0, "ss01"`, "liga 0,ss01 1"},
		{"normal feature settings", "font-feature-settings: normal", ""},
		{
			"settings replace variants and kerning",
			`font-kerning: none; font-variant-caps: small-caps; font-feature-settings: "kern" 1, "smcp" 0`,
			"kern 1,smcp 0",
		},
		{
			"variants follow kerning",
			"font-kerning: normal; font-variant-ligatures: no-common-ligatures; font-variant-position: sub",
			"kern 1,liga 0,clig 0,subs 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := familyIn(t, NewContext(nil), tt.css)[0]
			mapped := s.LineBreakStyle()
			if got := featuresOf(mapped); got != tt.want {
				t.Errorf("features = %q, want %q", got, tt.want)
			}
			if got := fonts.FeatureString(s.FontFeatures()); got != tt.want {
				t.Errorf("FontFeatures() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLanguagesMapToTheLineBreakStyle(t *testing.T) {
	tests := []struct {
		name         string
		css          string
		lang         string
		wantOverride props.KeywordOrString
		wantLanguage string
	}{
		{"none", "", "", props.KeywordOrString{Keyword: "normal"}, ""},
		{"lang", "", "fr", props.KeywordOrString{Keyword: "normal"}, "fr"},
		{"override", `font-language-override: "TRK"`, "fr", props.KeywordOrString{String: "TRK"}, "tur"},
		{"override without lang", `font-language-override: "TRK"`, "", props.KeywordOrString{String: "TRK"}, "tur"},
		{"normal override", "font-language-override: normal", "fr", props.KeywordOrString{Keyword: "normal"}, "fr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := familyIn(t, NewContext(nil), tt.css)[0]
			s.Set(props.PropLang, props.String(tt.lang))
			got := s.LineBreakStyle()
			if got.Lang != tt.lang || got.LanguageOverride != tt.wantOverride {
				t.Errorf("Lang, LanguageOverride = %q, %+v; want %q, %+v", got.Lang, got.LanguageOverride, tt.lang, tt.wantOverride)
			}
			if got := fonts.Language(got.LanguageOverride, got.Lang); got != tt.wantLanguage {
				t.Errorf("language = %q, want %q", got, tt.wantLanguage)
			}
		})
	}
}

func TestLangIsInherited(t *testing.T) {
	ctx := NewContext(nil)
	root := familyIn(t, ctx, "")[0]
	root.Set(props.PropLang, props.String("de"))
	if got := New(root).LineBreakStyle().Lang; got != "de" {
		t.Errorf("Lang = %q, want de", got)
	}
}

func TestLineHeightMapsToTheLineBreakStyle(t *testing.T) {
	tests := []struct {
		name string
		css  []string
		want linebreak.LineHeight
	}{
		{"normal", []string{"line-height: normal"}, linebreak.LineHeight{Normal: true}},
		{"number", []string{"line-height: 1.25"}, linebreak.LineHeight{Number: true, Value: 1.25}},
		{"length", []string{"line-height: 30px"}, linebreak.LineHeight{Value: 30}},
		{"relative length", []string{"font-size: 20px; line-height: 2em"}, linebreak.LineHeight{Value: 40}},
		{"percentage", []string{"font-size: 20px; line-height: 150%"}, linebreak.LineHeight{Value: 30}},
		{"number inherited", []string{"line-height: 1.5", "font-size: 30px"}, linebreak.LineHeight{Number: true, Value: 1.5}},
		{"length inherited as pixels", []string{"font-size: 10px; line-height: 2em", "font-size: 30px"}, linebreak.LineHeight{Value: 20}},
		{"math with a number", []string{"line-height: calc(1.5 * 2)"}, linebreak.LineHeight{Number: true, Value: 3}},
		{"math with lengths", []string{"font-size: 20px; line-height: calc(1.2em + 2px)"}, linebreak.LineHeight{Value: 26}},
		{
			"math with a percentage of the font size",
			[]string{"font-size: 20px; line-height: calc(50% + 1em)"},
			linebreak.LineHeight{Value: 30},
		},
		{
			"math with only a percentage",
			[]string{"font-size: 20px; line-height: calc(100% + 2px)"},
			linebreak.LineHeight{Value: 22},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lineBreakStyleOf(t, tt.css...).LineHeight; got != tt.want {
				t.Errorf("LineHeight = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSpacingMapsToPixels(t *testing.T) {
	tests := []struct {
		name           string
		css            []string
		letter, spaces float64
	}{
		{"normal", []string{""}, 0, 0},
		{"explicit normal", []string{"letter-spacing: normal; word-spacing: normal"}, 0, 0},
		{"lengths", []string{"letter-spacing: 2px; word-spacing: 3px"}, 2, 3},
		{"negative", []string{"letter-spacing: -1.5px; word-spacing: -4px"}, -1.5, -4},
		{"font relative", []string{"font-size: 20px; letter-spacing: 0.1em; word-spacing: 1em"}, 2, 20},
		{"absolute units", []string{"letter-spacing: 1in; word-spacing: 72pt"}, 96, 96},
		{"math", []string{"font-size: 10px; letter-spacing: calc(1px + 1em); word-spacing: calc(2 * 3px)"}, 11, 6},
		{"inherited", []string{"letter-spacing: 3px; word-spacing: 5px", ""}, 3, 5},
		{"inherited as pixels", []string{"font-size: 10px; letter-spacing: 1em", "font-size: 30px"}, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineBreakStyleOf(t, tt.css...)
			if got.LetterSpacing != tt.letter || got.WordSpacing != tt.spaces {
				t.Errorf("LetterSpacing, WordSpacing = %v, %v; want %v, %v", got.LetterSpacing, got.WordSpacing, tt.letter, tt.spaces)
			}
		})
	}
}

func TestWhiteSpaceAndBreakingMapToKeywords(t *testing.T) {
	tests := []struct {
		css                                          string
		whiteSpace, wordBreak, overflowWrap, hyphens string
		direction                                    string
	}{
		{"white-space: pre", "pre", "normal", "normal", "manual", "ltr"},
		{"white-space: nowrap", "nowrap", "normal", "normal", "manual", "ltr"},
		{"white-space: pre-wrap", "pre-wrap", "normal", "normal", "manual", "ltr"},
		{"white-space: pre-line", "pre-line", "normal", "normal", "manual", "ltr"},
		{"word-break: break-all", "normal", "break-all", "normal", "manual", "ltr"},
		{"overflow-wrap: anywhere", "normal", "normal", "anywhere", "manual", "ltr"},
		{"overflow-wrap: break-word", "normal", "normal", "break-word", "manual", "ltr"},
		{"hyphens: auto", "normal", "normal", "normal", "auto", "ltr"},
		{"hyphens: none", "normal", "normal", "normal", "none", "ltr"},
		{"direction: rtl", "normal", "normal", "normal", "manual", "rtl"},
		{
			"white-space: pre; word-break: break-all; overflow-wrap: anywhere; hyphens: auto; direction: rtl",
			"pre", "break-all", "anywhere", "auto", "rtl",
		},
	}
	for _, tt := range tests {
		t.Run(tt.css, func(t *testing.T) {
			got := lineBreakStyleOf(t, tt.css)
			have := []string{got.WhiteSpace, got.WordBreak, got.OverflowWrap, got.Hyphens, got.Direction}
			want := []string{tt.whiteSpace, tt.wordBreak, tt.overflowWrap, tt.hyphens, tt.direction}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("white-space, word-break, overflow-wrap, hyphens, direction = %v, want %v", have, want)
			}
		})
	}
}

func TestInheritedKeywordsReachTheLineBreakStyle(t *testing.T) {
	got := lineBreakStyleOf(t, "white-space: pre-wrap; hyphens: auto; overflow-wrap: anywhere", "")
	if got.WhiteSpace != "pre-wrap" || got.Hyphens != "auto" || got.OverflowWrap != "anywhere" {
		t.Errorf("WhiteSpace, Hyphens, OverflowWrap = %q, %q, %q", got.WhiteSpace, got.Hyphens, got.OverflowWrap)
	}
}

func TestHyphenationMapsToTheLineBreakStyle(t *testing.T) {
	tests := []struct {
		css       string
		character string
		limits    linebreak.HyphenateLimitChars
	}{
		{"", "‐", linebreak.HyphenateLimitChars{Total: 5, Before: 2, After: 2}},
		{"hyphenate-character: auto", "‐", linebreak.HyphenateLimitChars{Total: 5, Before: 2, After: 2}},
		{`hyphenate-character: "!"`, "!", linebreak.HyphenateLimitChars{Total: 5, Before: 2, After: 2}},
		{`hyphenate-character: "ù ù"`, "ù ù", linebreak.HyphenateLimitChars{Total: 5, Before: 2, After: 2}},
		{`hyphenate-character: ""`, "", linebreak.HyphenateLimitChars{Total: 5, Before: 2, After: 2}},
		{"hyphenate-limit-chars: 6 3 4", "‐", linebreak.HyphenateLimitChars{Total: 6, Before: 3, After: 4}},
		{"hyphenate-limit-chars: 7", "‐", linebreak.HyphenateLimitChars{Total: 7, Before: 2, After: 2}},
		{"hyphenate-limit-chars: auto 1", "‐", linebreak.HyphenateLimitChars{Total: 5, Before: 1, After: 1}},
		{"hyphenate-limit-chars: 0 0 0", "‐", linebreak.HyphenateLimitChars{}},
	}
	for _, tt := range tests {
		t.Run(tt.css, func(t *testing.T) {
			got := lineBreakStyleOf(t, tt.css)
			if got.HyphenateCharacter != tt.character {
				t.Errorf("HyphenateCharacter = %q, want %q", got.HyphenateCharacter, tt.character)
			}
			if got.HyphenateLimitChars != tt.limits {
				t.Errorf("HyphenateLimitChars = %+v, want %+v", got.HyphenateLimitChars, tt.limits)
			}
		})
	}
}

func TestHyphenateLimitZoneMapsToAResolvableLength(t *testing.T) {
	tests := []struct {
		name string
		css  string
		base float64
		want float64
	}{
		{"initial", "", 200, 0},
		{"length", "hyphenate-limit-zone: 12px", 200, 12},
		{"font relative length", "font-size: 20px; hyphenate-limit-zone: 2em", 200, 40},
		{"percentage", "hyphenate-limit-zone: 20%", 200, 40},
		{"math with lengths", "font-size: 20px; hyphenate-limit-zone: calc(1em + 2px)", 200, 22},
		{"math with a percentage", "font-size: 20px; hyphenate-limit-zone: calc(1em + 100%)", 200, 220},
		{"math with a percentage, other base", "font-size: 20px; hyphenate-limit-zone: calc(1em + 100%)", 50, 70},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone := lineBreakStyleOf(t, tt.css).HyphenateLimitZone
			if got := zone.Resolve(tt.base); got != tt.want {
				t.Errorf("Resolve(%v) = %v, want %v", tt.base, got, tt.want)
			}
		})
	}
}

func TestTabSizeMapsToSpacesOrLength(t *testing.T) {
	tests := []struct {
		css  string
		want linebreak.TabSize
	}{
		{"", linebreak.TabSize{Spaces: 8}},
		{"tab-size: 4", linebreak.TabSize{Spaces: 4}},
		{"tab-size: 1", linebreak.TabSize{Spaces: 1}},
		{"tab-size: 10px", linebreak.TabSize{Length: 10, IsLength: true}},
		{"tab-size: 0", linebreak.TabSize{IsLength: true}},
		{"font-size: 20px; tab-size: 2em", linebreak.TabSize{Length: 40, IsLength: true}},
		{"font-size: 10px; tab-size: calc(1px + 1em)", linebreak.TabSize{Length: 11, IsLength: true}},
	}
	for _, tt := range tests {
		t.Run(tt.css, func(t *testing.T) {
			if got := lineBreakStyleOf(t, tt.css).TabSize; got != tt.want {
				t.Errorf("TabSize = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLineBreakStyleFollowsChangesToTheStyle(t *testing.T) {
	s := New(nil)
	if got := s.LineBreakStyle().WhiteSpace; got != "normal" {
		t.Fatalf("WhiteSpace = %q", got)
	}
	s.Set(props.PropWhiteSpace, props.Keyword("pre"))
	if got := s.LineBreakStyle().WhiteSpace; got != "pre" {
		t.Errorf("WhiteSpace = %q after Set, want pre", got)
	}
}

// The strut and the units that depend on the font are the same whether
// they are asked of a computed style or of the line break style it maps to.
func TestStrutAndCharacterRatioAgreeWithTheLineBreakStyle(t *testing.T) {
	tests := []string{
		"",
		"font-size: 16px",
		"font-size: 13.333px; line-height: 1.5",
		"font-size: 10px; line-height: 30px",
		"font-size: 20px; line-height: 150%",
		"font-size: 20px; line-height: calc(50% + 1em)",
		"font-size: 7.5px; line-height: normal",
		"font-size: 0",
		"font-size: 12px; letter-spacing: 3px; word-spacing: 9px",
		"font-size: 40px; font-kerning: none; font-variant-caps: small-caps",
		`font-size: 16px; font-feature-settings: "smcp"; font-language-override: "TRK"`,
		"font-size: 16px; font-weight: bold; font-style: italic; font-stretch: condensed",
		"font-family: other, testfont",
		"font-family: missing",
	}
	for _, css := range tests {
		t.Run(css, func(t *testing.T) {
			ctx := testFontContext(t)
			lctx := linebreak.NewContext(ctx.FontConfig)
			if !strings.Contains(css, "font-family") {
				css += "; font-family: testfont"
			}
			s := familyIn(t, ctx, css)[0]
			mapped := s.LineBreakStyle()

			h, b := s.Strut()
			lh, lb := linebreak.Strut(lctx, mapped)
			if h != lh || b != lb {
				t.Errorf("Strut() = %v, %v; linebreak.Strut = %v, %v", h, b, lh, lb)
			}
			for _, unit := range []string{"ex", "ch", "cap", "ic"} {
				if got, want := s.CharacterRatio(unit), linebreak.CharacterRatio(lctx, mapped, unit); got != want {
					t.Errorf("CharacterRatio(%q) = %v; linebreak.CharacterRatio = %v", unit, got, want)
				}
			}
		})
	}
}

func TestStrutValues(t *testing.T) {
	const baseline = 12.796875
	tests := []struct {
		css          string
		height, base float64
	}{
		{"", 16, baseline},
		{"line-height: 1.5", 24, baseline + 4},
		{"line-height: 20px", 20, baseline + 2},
		{"line-height: 10px", 10, baseline - 3},
		{"line-height: calc(50% + 1em)", 24, baseline + 4},
		{"font-size: 0", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.css, func(t *testing.T) {
			ctx := testFontContext(t)
			s := familyIn(t, ctx, "font-family: testfont; "+tt.css)[0]
			if h, b := s.Strut(); h != tt.height || b != tt.base {
				t.Errorf("Strut() = %v, %v; want %v, %v", h, b, tt.height, tt.base)
			}
		})
	}
}

func TestCharacterRatioValues(t *testing.T) {
	ctx := testFontContext(t)
	s := familyIn(t, ctx, "font-family: testfont; font-size: 30px")[0]
	for unit, want := range map[string]float64{"ex": 0.7998, "ch": 1, "cap": 1, "ic": 2} {
		if got := s.CharacterRatio(unit); got != want {
			t.Errorf("CharacterRatio(%q) = %v, want %v", unit, got, want)
		}
	}
}

// Elements that do not differ in their font measure the same, and the
// measures follow a font added to the document later.
func TestMeasuresAreSharedByTheStylesOfADocument(t *testing.T) {
	ctx := testFontContext(t)
	styles := familyIn(t, ctx,
		"font-family: testfont; line-height: 2",
		"line-height: 1; font-kerning: none",
		"font-size: 32px",
	)
	if h, _ := styles[0].Strut(); h != 32 {
		t.Errorf("first strut height = %v, want 32", h)
	}
	if h, _ := styles[1].Strut(); h != 16 {
		t.Errorf("second strut height = %v, want 16", h)
	}
	if h, _ := styles[2].Strut(); h != 32 {
		t.Errorf("third strut height = %v, want 32", h)
	}
	if a, b := styles[0].CharacterRatio("ex"), styles[2].CharacterRatio("ex"); a != b || a != 0.7998 {
		t.Errorf("ex = %v and %v, want 0.7998", a, b)
	}
}
