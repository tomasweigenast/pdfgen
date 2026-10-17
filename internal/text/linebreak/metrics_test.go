package linebreak

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

func TestStrut(t *testing.T) {
	ctx := newTestContext(t)
	const baseline = 12.796875
	tests := []struct {
		lineHeight         LineHeight
		height, baselineAt float64
	}{
		{LineHeight{Normal: true}, 16, baseline},
		{LineHeight{Number: true, Value: 1.5}, 24, baseline + 4},
		{LineHeight{Value: 20}, 20, baseline + 2},
		{LineHeight{Value: 10}, 10, baseline - 3},
	}
	for _, tt := range tests {
		style := testStyle()
		style.LineHeight = tt.lineHeight
		h, b := Strut(ctx, style)
		if h != tt.height || b != tt.baselineAt {
			t.Errorf("%+v: strut %v, %v; want %v, %v", tt.lineHeight, h, b, tt.height, tt.baselineAt)
		}
	}
	style := testStyle()
	style.Font.Size = 0
	if h, b := Strut(ctx, style); h != 0 || b != 0 {
		t.Errorf("zero font size: strut %v, %v", h, b)
	}
}

func TestStrutIgnoresSpacing(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.WordSpacing = 10
	style.LetterSpacing = 3
	if h, b := Strut(ctx, style); h != 16 || b != 12.796875 {
		t.Errorf("strut %v, %v; want 16, 12.796875", h, b)
	}
}

// The strut of a style is the line of a space laid out with that style.
func TestStrutIsTheLineOfASpace(t *testing.T) {
	ctx := newTestContext(t)
	for _, size := range []float64{16, 13.333, 7.5, 2, 100} {
		style := NewStyle(size, "testfont")
		layout := newLayout(ctx, style, 0, nil)
		layout.SetText(" ")
		_, wantHeight := LineSize(layout.FirstLine(), style)
		wantBaseline := layout.Baseline()
		if h, b := Strut(ctx, style); h != wantHeight || b != wantBaseline {
			t.Errorf("size %v: strut %v, %v; the line has %v, %v", size, h, b, wantHeight, wantBaseline)
		}
	}
}

func TestCharacterRatioOfStyle(t *testing.T) {
	ctx := newTestContext(t)
	if r := CharacterRatio(ctx, testStyle(), "ex"); r != 0.7998 {
		t.Errorf("ex = %v, want 0.7998", r)
	}
	if r := CharacterRatio(ctx, testStyle(), "ch"); r != 1 {
		t.Errorf("ch = %v, want 1", r)
	}
}

func TestCharacterRatioIgnoresSpacingAndFontSize(t *testing.T) {
	ctx := newTestContext(t)
	style := NewStyle(40, "testfont")
	style.LetterSpacing, style.WordSpacing = 5, 7
	if r := CharacterRatio(ctx, style, "ex"); r != 0.7998 {
		t.Errorf("ex = %v, want 0.7998", r)
	}
}

// Styles of different packages that share the fonts share their measures.
func TestMeasuresAreTakenByTheFontConfiguration(t *testing.T) {
	ctx := newTestContext(t)
	other := NewContext(ctx.Fonts)
	spec := fonts.Spec{Font: fonts.Description{Families: []string{"testfont"}, Weight: 400, Size: 16}}

	style := testStyle()
	h, b := Strut(ctx, style)
	wantH, wantB := ctx.Fonts.Strut(spec, fonts.LineHeight{Normal: true})
	if h != wantH || b != wantB {
		t.Errorf("Strut = %v, %v; the font configuration gives %v, %v", h, b, wantH, wantB)
	}
	if h2, b2 := Strut(other, style); h2 != h || b2 != b {
		t.Errorf("Strut on another context = %v, %v; want %v, %v", h2, b2, h, b)
	}
	if got, want := CharacterRatio(other, style, "ex"), ctx.Fonts.CharacterRatio(spec, "ex"); got != want {
		t.Errorf("CharacterRatio = %v; the font configuration gives %v", got, want)
	}
}

func TestStyleFontSpec(t *testing.T) {
	style := NewStyle(20, "a", "b")
	style.Lang = "fr"
	style.LanguageOverride = props.KeywordOrString{String: "TRK"}
	style.Font.Weight = 700
	style.Features.Kerning = "none"
	spec := style.fontSpec()
	if spec.Font.Size != 20 || spec.Font.Weight != 700 || len(spec.Font.Families) != 2 {
		t.Errorf("font = %+v", spec.Font)
	}
	if spec.Lang != "fr" || spec.Language() != "tur" {
		t.Errorf("lang %q, language %q", spec.Lang, spec.Language())
	}
	if got := fonts.FeatureString(spec.FontFeatures()); got != "kern 0" {
		t.Errorf("features = %q, want \"kern 0\"", got)
	}
}

func TestNewStyleHasTheInitialHyphenation(t *testing.T) {
	s := NewStyle(16, "testfont")
	if s.HyphenateCharacter != "‐" {
		t.Errorf("initial hyphenate-character = %q", s.HyphenateCharacter)
	}
	if s.HyphenateLimitChars != (HyphenateLimitChars{5, 2, 2}) {
		t.Errorf("initial hyphenate-limit-chars = %+v", s.HyphenateLimitChars)
	}
}
