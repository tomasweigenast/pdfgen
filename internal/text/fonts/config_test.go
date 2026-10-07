package fonts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// resourceFetcher serves the files of testdata/resources by name.
func resourceFetcher(t testing.TB) Fetcher {
	t.Helper()
	return func(name string) ([]byte, error) {
		name, _, _ = strings.Cut(name, "#")
		return os.ReadFile(testutil.ResourcePath(t, name))
	}
}

func newTestConfiguration(t testing.TB) *FontConfiguration {
	t.Helper()
	return NewFontConfiguration(WithoutSystemFonts())
}

// addTestFont registers a resource font under a family.
func addTestFont(t testing.TB, fc *FontConfiguration, rule FontFaceRule) {
	t.Helper()
	if len(rule.Src) == 0 {
		rule.Src = []Source{{Value: "testfont.otf"}}
	}
	if err := fc.AddFontFace(rule, resourceFetcher(t)); err != nil {
		t.Fatalf("AddFontFace(%q): %v", rule.Family, err)
	}
}

func TestFaceOutlivesConfiguration(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "bug"})
	f := fc.Query(Description{Families: []string{"bug"}, Size: 10}, "", 'a')
	fc = nil
	runtime.GC()
	if got := textWidth(t, f, 10, "ab"); got != 20 {
		t.Errorf("width = %v, want 20", got)
	}
}

func query(fc *FontConfiguration, d Description, r rune) *Face { return fc.Query(d, "", r) }

func family(name string) Description {
	return Description{Families: []string{name}, Size: 16}
}

// withSystemFonts returns a configuration with installed fonts, and skips
// the test when the machine has none.
func withSystemFonts(t testing.TB) *FontConfiguration {
	t.Helper()
	fc := NewFontConfiguration(WithCacheDir(t.TempDir()))
	if fc.system() == nil || len(fc.footprints) == 0 {
		t.Skip("no system fonts")
	}
	return fc
}

func TestFontFaceGlyphWidth(t *testing.T) {
	testutil.AssertNoLogs(t)
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	f := query(fc, family("testfont"), 'a')
	if f == nil {
		t.Fatal("no face")
	}
	if got := textWidth(t, f, 16, "abc"); got != 3*16 {
		t.Errorf("width = %v, want %v", got, 3*16)
	}
}

func TestKerningAndLigaturesAreOnByDefault(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	f := query(fc, family("testfont"), 'k')
	for text, want := range map[string]float64{"kk": 1.5 * 16, "liga": 1.5 * 16} {
		if got := textWidth(t, f, 16, text); got != want {
			t.Errorf("width of %q = %v, want %v", text, got, want)
		}
	}
}

func TestKerningDeactivatedByFontFaceSettings(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{
		Family:          "no-kern",
		FeatureSettings: props.FontSettings{Items: []props.FontSetting{{Tag: "kern", Value: 0}}},
	})
	addTestFont(t, fc, FontFaceRule{Family: "kern"})
	if got := textWidth(t, query(fc, family("kern"), 'k'), 16, "kk"); got != 1.5*16 {
		t.Errorf("kern: width = %v, want %v", got, 1.5*16)
	}
	if got := textWidth(t, query(fc, family("no-kern"), 'k'), 16, "kk"); got != 2*16 {
		t.Errorf("no-kern: width = %v, want %v", got, 2*16)
	}
}

func TestKerningAndLigaturesDeactivatedByFontFaceRule(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{
		Family:          "no-kern-liga",
		FeatureSettings: props.FontSettings{Items: []props.FontSetting{{Tag: "kern", Value: 0}}},
		Variant:         FontVariant{Ligatures: props.Keywords{"no-common-ligatures"}},
	})
	addTestFont(t, fc, FontFaceRule{Family: "kern-liga"})
	if got := textWidth(t, query(fc, family("kern-liga"), 'k'), 16, "kk liga"); got != (1.5+1+1.5)*16 {
		t.Errorf("kern-liga: width = %v, want %v", got, (1.5+1+1.5)*16)
	}
	if got := textWidth(t, query(fc, family("no-kern-liga"), 'k'), 16, "kk liga"); got != (2+1+4)*16 {
		t.Errorf("no-kern-liga: width = %v, want %v", got, (2+1+4)*16)
	}
}

func TestFontFaceVariantDescriptorsSelectFeatures(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{
		Family: "testfont-variant",
		Variant: FontVariant{
			Position:  "sub",
			Ligatures: props.Keywords{"discretionary-ligatures"},
			Numeric:   props.Keywords{"oldstyle-nums", "slashed-zero"},
		},
	})
	f := query(fc, family("testfont-variant"), 'k')
	for _, text := range []string{"kk", "subs", "dlig", "onum", "zero"} {
		if got := textWidth(t, f, 16, text); got != 1.5*16 {
			t.Errorf("width of %q = %v, want %v", text, got, 1.5*16)
		}
	}
}

func TestStyleFeaturesApplyAfterFontFaceFeatures(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	f := query(fc, family("testfont"), 'k')
	if got := textWidth(t, f, 16, "kk", Feature{"kern", 0}); got != 2*16 {
		t.Errorf("width with kerning off = %v, want %v", got, 2*16)
	}
	smcp := Features(FeatureStyle{Kerning: "auto", FontVariant: FontVariant{Caps: "small-caps"}})
	if got := textWidth(t, f, 16, "kk liga camps", smcp...); got != 120*16/16 {
		t.Errorf("width with small caps = %v, want 120", got)
	}
}

func TestWOFFFontFaceMatchesOpenType(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont-otf"})
	addTestFont(t, fc, FontFaceRule{Family: "testfont-woff", Src: []Source{{Value: "testfont.woff"}}})
	addTestFont(t, fc, FontFaceRule{Family: "testfont-woff-again", Src: []Source{{Value: "testfont.woff"}}})
	otf := textWidth(t, query(fc, family("testfont-otf"), 'w'), 16, "woff font")
	for _, name := range []string{"testfont-woff", "testfont-woff-again"} {
		if got := textWidth(t, query(fc, family(name), 'w'), 16, "woff font"); got != otf {
			t.Errorf("%s: width = %v, want %v", name, got, otf)
		}
	}
}

func TestInstalledFontDiffersFromFontFace(t *testing.T) {
	fc := withSystemFonts(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	loaded := textWidth(t, query(fc, family("testfont"), 'w'), 16, "woff font")
	sans := query(fc, family("sans-serif"), 'w')
	if sans == nil {
		t.Fatal("no face for sans-serif")
	}
	if got := textWidth(t, sans, 16, "woff font"); got == loaded {
		t.Errorf("installed font has the width of the loaded font: %v", got)
	}
}

func TestQueryReturnsTheSameFace(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	if a, b := query(fc, family("TestFont"), 'a'), query(fc, family("testfont"), 'b'); a == nil || a != b {
		t.Errorf("faces differ: %p, %p", a, b)
	}
}

func TestQueryWithoutFonts(t *testing.T) {
	fc := newTestConfiguration(t)
	if f := query(fc, family("nothing"), 'a'); f != nil {
		t.Errorf("got face %q, want none", f.ID)
	}
	if f := fc.Query(Description{}, "", 'a'); f != nil {
		t.Errorf("got face %q for no families", f.ID)
	}
}

func TestQueryFamilyOrder(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "first", Variant: FontVariant{Position: "sub"}})
	addTestFont(t, fc, FontFaceRule{Family: "second"})
	d := Description{Families: []string{"missing", "second", "first"}, Size: 16}
	f := query(fc, d, 'a')
	if f == nil || len(f.Features) != 1 {
		t.Fatalf("got %+v, want the face of second", f)
	}
}

func TestUnicodeRangeLimitsFace(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{
		Family:       "digits",
		UnicodeRange: []Range{{'0', '9'}},
		Variant:      FontVariant{Position: "sub"},
	})
	addTestFont(t, fc, FontFaceRule{Family: "letters"})
	d := Description{Families: []string{"digits", "letters"}, Size: 16}
	digit, letter := query(fc, d, '5'), query(fc, d, 'a')
	if digit == nil || letter == nil {
		t.Fatal("missing face")
	}
	if len(digit.Features) != 2 || len(letter.Features) != 1 {
		t.Errorf("digit face has %v, letter face has %v", digit.Features, letter.Features)
	}
}

func TestUnicodeRangeWithoutFallbackKeepsNoFace(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "digits", UnicodeRange: []Range{{'0', '9'}}})
	if f := query(fc, family("digits"), 'a'); f != nil {
		t.Errorf("got face %q outside of the unicode range", f.ID)
	}
}

// matchTag identifies which registered face a query returned: the faces of
// the matching tests share one font, and carry their index as a feature.
func matchTag(t *testing.T, f *Face) int {
	t.Helper()
	if f == nil {
		t.Fatal("no face")
	}
	for _, ft := range f.Features {
		if ft.Tag == "ss01" {
			return ft.Value
		}
	}
	t.Fatalf("face has no tag: %v", f.Features)
	return 0
}

func addTagged(t *testing.T, fc *FontConfiguration, tag int, rule FontFaceRule) {
	t.Helper()
	rule.Family = "f"
	rule.FeatureSettings = props.FontSettings{Items: []props.FontSetting{{Tag: "ss01", Value: float64(tag)}}}
	addTestFont(t, fc, rule)
}

func TestFontFaceMatchingByWeight(t *testing.T) {
	tests := []struct {
		name    string
		weights []int
		want    int
		pick    int
	}{
		{"exact", []int{300, 400, 700}, 400, 1},
		{"400 prefers 500 over lighter", []int{300, 500, 700}, 400, 1},
		{"500 prefers 400", []int{300, 400, 700}, 500, 1},
		{"400 without 500 goes lighter", []int{300, 700}, 400, 0},
		{"600 goes heavier first", []int{400, 700, 900}, 600, 1},
		{"600 goes lighter when nothing is heavier", []int{300, 400}, 600, 1},
		{"200 goes lighter first", []int{100, 300, 400}, 200, 0},
		{"200 goes heavier when nothing is lighter", []int{300, 400}, 200, 0},
		{"800 goes heavier first", []int{400, 700, 900}, 800, 2},
		{"default weight is 400", []int{400, 700}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := newTestConfiguration(t)
			for i, w := range tc.weights {
				addTagged(t, fc, i, FontFaceRule{Weight: w, Src: []Source{{Value: fmt.Sprintf("testfont.otf#%d", i)}}})
			}
			d := family("f")
			d.Weight = tc.want
			if got := matchTag(t, query(fc, d, 'a')); got != tc.pick {
				t.Errorf("picked face %d (weight %d), want face %d (weight %d)", got, tc.weights[got], tc.pick, tc.weights[tc.pick])
			}
		})
	}
}

func TestFontFaceMatchingByStyle(t *testing.T) {
	tests := []struct {
		name   string
		styles []string
		want   string
		pick   int
	}{
		{"normal", []string{"italic", "normal"}, "normal", 1},
		{"italic", []string{"normal", "italic"}, "italic", 1},
		{"italic falls back to oblique", []string{"normal", "oblique"}, "italic", 1},
		{"oblique falls back to italic", []string{"normal", "italic"}, "oblique", 1},
		{"italic falls back to normal", []string{"normal"}, "italic", 0},
		{"normal falls back to oblique before italic", []string{"italic", "oblique"}, "normal", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := newTestConfiguration(t)
			for i, s := range tc.styles {
				addTagged(t, fc, i, FontFaceRule{Style: s, Src: []Source{{Value: fmt.Sprintf("testfont.otf#%d", i)}}})
			}
			d := family("f")
			d.Style = tc.want
			if got := matchTag(t, query(fc, d, 'a')); got != tc.pick {
				t.Errorf("picked face %d, want %d", got, tc.pick)
			}
		})
	}
}

func TestFontFaceMatchingByStretch(t *testing.T) {
	tests := []struct {
		name     string
		stretchs []string
		want     string
		pick     int
	}{
		{"exact", []string{"condensed", "normal", "expanded"}, "normal", 1},
		{"condensed goes narrower first", []string{"ultra-condensed", "normal"}, "semi-condensed", 0},
		{"condensed goes wider if nothing is narrower", []string{"normal", "expanded"}, "condensed", 0},
		{"expanded goes wider first", []string{"normal", "ultra-expanded"}, "semi-expanded", 1},
		{"expanded goes narrower if nothing is wider", []string{"condensed", "normal"}, "expanded", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := newTestConfiguration(t)
			for i, s := range tc.stretchs {
				addTagged(t, fc, i, FontFaceRule{Stretch: s, Src: []Source{{Value: fmt.Sprintf("testfont.otf#%d", i)}}})
			}
			d := family("f")
			d.Stretch = tc.want
			if got := matchTag(t, query(fc, d, 'a')); got != tc.pick {
				t.Errorf("picked face %d, want %d", got, tc.pick)
			}
		})
	}
}

func TestFontFaceMatchingPrefersStretchThenStyleThenWeight(t *testing.T) {
	fc := newTestConfiguration(t)
	rules := []FontFaceRule{
		{Weight: 400, Style: "normal", Stretch: "condensed"},
		{Weight: 700, Style: "italic", Stretch: "normal"},
		{Weight: 400, Style: "normal", Stretch: "normal"},
	}
	for i, r := range rules {
		r.Src = []Source{{Value: fmt.Sprintf("testfont.otf#%d", i)}}
		addTagged(t, fc, i, r)
	}
	d := family("f")
	d.Weight, d.Style = 700, "italic"
	if got := matchTag(t, query(fc, d, 'a')); got != 1 {
		t.Errorf("picked face %d, want 1", got)
	}
	d.Stretch = "condensed"
	if got := matchTag(t, query(fc, d, 'a')); got != 0 {
		t.Errorf("with stretch condensed: picked face %d, want 0", got)
	}
}

func TestFontFaceUsesFontPropertiesWhenDescriptorsAreUnset(t *testing.T) {
	fc := newTestConfiguration(t)
	addTagged(t, fc, 0, FontFaceRule{Src: []Source{{Value: "testfont.otf#a"}}, Weight: 700})
	addTagged(t, fc, 1, FontFaceRule{Src: []Source{{Value: "testfont.otf#b"}}})
	d := family("f")
	d.Weight = 700
	if got := matchTag(t, query(fc, d, 'a')); got != 0 {
		t.Errorf("weight 700: picked %d, want 0", got)
	}
	d.Weight = 400
	if got := matchTag(t, query(fc, d, 'a')); got != 1 {
		t.Errorf("weight 400: picked %d, want 1", got)
	}
}

func TestLaterFontFaceWinsOverEqualOne(t *testing.T) {
	fc := newTestConfiguration(t)
	addTagged(t, fc, 0, FontFaceRule{Src: []Source{{Value: "testfont.otf#a"}}})
	addTagged(t, fc, 1, FontFaceRule{Src: []Source{{Value: "testfont.otf#b"}}})
	if got := matchTag(t, query(fc, family("f"), 'a')); got != 1 {
		t.Errorf("picked %d, want 1", got)
	}
}

func TestAddFontFaceTriesSourcesInOrder(t *testing.T) {
	fc := newTestConfiguration(t)
	good := readResource(t, "testfont.otf")
	fetch := func(url string) ([]byte, error) {
		switch url {
		case "missing":
			return nil, errors.New("404")
		case "garbage":
			return []byte("not a font"), nil
		}
		return good, nil
	}
	err := fc.AddFontFace(FontFaceRule{Family: "x", Src: []Source{
		{Value: ""},
		{Value: "missing"},
		{Value: "garbage"},
		{Kind: SourceLocal, Value: "No Such Font"},
		{Value: "good", Format: "opentype"},
	}}, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if query(fc, family("x"), 'a') == nil {
		t.Error("font was not registered")
	}
}

func TestAddFontFaceFailure(t *testing.T) {
	fc := newTestConfiguration(t)
	fetch := func(string) ([]byte, error) { return nil, errors.New("offline") }
	err := fc.AddFontFace(FontFaceRule{Family: "lost", Src: []Source{{Value: "a"}, {Value: "b"}}}, fetch)
	if !errors.Is(err, ErrFontFaceUnavailable) {
		t.Fatalf("err = %v, want ErrFontFaceUnavailable", err)
	}
	if !strings.Contains(err.Error(), "lost") {
		t.Errorf("error %q does not name the family", err)
	}
	if err := fc.AddFontFace(FontFaceRule{Family: "lost", Src: []Source{{Value: "a"}}}, nil); !errors.Is(err, ErrFontFaceUnavailable) {
		t.Errorf("without a fetcher: err = %v", err)
	}
	if err := fc.AddFontFace(FontFaceRule{Family: "none"}, nil); !errors.Is(err, ErrFontFaceUnavailable) {
		t.Errorf("without sources: err = %v", err)
	}
}

func TestAddFontFaceIgnoresRepeatedRule(t *testing.T) {
	fc := newTestConfiguration(t)
	rule := FontFaceRule{Family: "again", Src: []Source{{Value: "testfont.otf"}}}
	calls := 0
	fetch := func(string) ([]byte, error) {
		calls++
		return readResource(t, "testfont.otf"), nil
	}
	for range 3 {
		if err := fc.AddFontFace(rule, fetch); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 || len(fc.faces["again"]) != 1 {
		t.Errorf("fetched %d times and registered %d faces, want 1 and 1", calls, len(fc.faces["again"]))
	}
}

// fullNameOf returns the full name of the first face of a font file.
func fullNameOf(data []byte) string {
	loaders, err := opentype.NewLoaders(bytes.NewReader(data))
	if err != nil || len(loaders) == 0 {
		return ""
	}
	raw, err := loaders[0].RawTable(opentype.MustNewTag("name"))
	if err != nil {
		return ""
	}
	names, _, err := tables.ParseName(raw)
	if err != nil {
		return ""
	}
	return names.Name(4)
}

func TestLocalSourceFindsInstalledFont(t *testing.T) {
	fc := withSystemFonts(t)
	var name string
	for _, fp := range fc.footprints {
		if fp.Location.Index != 0 {
			continue
		}
		data, err := os.ReadFile(fp.Location.File)
		if err != nil {
			continue
		}
		if name = fullNameOf(data); name != "" && strings.HasPrefix(squash(name), squash(fp.Family)) {
			break
		}
		name = ""
	}
	if name == "" {
		t.Skip("no installed font with a readable full name")
	}
	rule := FontFaceRule{Family: "installed", Src: []Source{{Kind: SourceLocal, Value: strings.ToUpper(name)}}}
	if err := fc.AddFontFace(rule, nil); err != nil {
		t.Fatalf("local(%q): %v", name, err)
	}
	if query(fc, family("installed"), 'a') == nil {
		t.Error("font was not registered")
	}
}

func TestLocalSourceRequiresExactName(t *testing.T) {
	fc := withSystemFonts(t)
	rule := FontFaceRule{Family: "x", Src: []Source{{Kind: SourceLocal, Value: "Definitely Not Installed"}}}
	if err := fc.AddFontFace(rule, nil); !errors.Is(err, ErrFontFaceUnavailable) {
		t.Errorf("err = %v, want ErrFontFaceUnavailable", err)
	}
}

func TestCharacterRatioThroughConfiguration(t *testing.T) {
	fc := newTestConfiguration(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont"})
	d := family("testfont")
	for unit, want := range map[string]float64{"ex": 0.7998, "ch": 1, "cap": 1, "ic": 2} {
		if got := fc.CharacterRatio(d, "", nil, unit); got != want {
			t.Errorf("CharacterRatio(%q) = %v, want %v", unit, got, want)
		}
	}
	if got := fc.CharacterRatio(Description{Families: []string{"none"}}, "", nil, "ex"); got != 0.5 {
		t.Errorf("without fonts: %v, want 0.5", got)
	}
}

// A unit is measured with a fallback font when the first family has no
// glyph for its character.
func TestCharacterRatioUsesFallbackFontForMissingCharacter(t *testing.T) {
	fc := withSystemFonts(t)
	addTestFont(t, fc, FontFaceRule{Family: "testfont", UnicodeRange: []Range{{'A', 'A'}}})
	d := Description{Families: []string{"testfont", "serif"}, Size: 16}
	got := fc.CharacterRatio(d, "", nil, "ex")
	if got == 0.7998 || got <= 0 || got >= 1 {
		t.Errorf("ex = %v, want the ratio of an installed font", got)
	}
}

func TestGenericFamilyResolvesToInstalledFont(t *testing.T) {
	fc := withSystemFonts(t)
	for _, name := range []string{"serif", "sans-serif", "monospace"} {
		d := family(name)
		f := query(fc, d, 'a')
		if f == nil || !f.HasGlyph('a') || len(f.Raw) == 0 {
			t.Errorf("%s: face %+v", name, f)
		}
	}
}

func TestNamedFamilyMatchesExactlyButFallbackStillFindsGlyphs(t *testing.T) {
	fc := withSystemFonts(t)
	if f := fc.systemFace("no such family anywhere", family("no such family anywhere"), 'a'); f != nil {
		t.Errorf("unknown family matched %q", f.Family)
	}
	d := Description{Families: []string{"no such family anywhere", "serif"}, Size: 16}
	if f := fc.Query(d, "", 'a'); f == nil || !f.HasGlyph('a') {
		t.Error("the generic family after an unknown one was not used")
	}
}
