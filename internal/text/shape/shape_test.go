package shape

import (
	"os"
	"slices"
	"testing"

	"github.com/go-text/typesetting/language"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

func loadFace(t testing.TB, path string) *fonts.Face {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	f, err := fonts.NewFace(data, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return f
}

func testFace(t testing.TB) *fonts.Face {
	t.Helper()
	return loadFace(t, testutil.ResourcePath(t, "testfont.otf"))
}

// systemFace loads the first font found among candidates, and skips the
// test when there is none.
func systemFace(t testing.TB, candidates ...string) *fonts.Face {
	t.Helper()
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return loadFace(t, p)
		}
	}
	t.Skipf("none of %v is installed", candidates)
	return nil
}

func shapeText(f *fonts.Face, size float64, text string, features ...fonts.Feature) []Glyph {
	runes := []rune(text)
	return GoText{}.Shape(Run{Text: runes, End: len(runes), Face: f, Size: size, Features: features})
}

func advances(glyphs []Glyph) []int32 {
	out := make([]int32, len(glyphs))
	for i, g := range glyphs {
		out[i] = g.XAdvance
	}
	return out
}

func TestShapingMatchesReferenceAdvances(t *testing.T) {
	face := testFace(t)
	for _, c := range testFontCases {
		t.Run(c.name, func(t *testing.T) {
			got := shapeText(face, c.size, c.text, c.features...)
			if !slices.Equal(got, c.glyphs) {
				t.Errorf("glyphs differ\n got %v\nwant %v", got, c.glyphs)
			}
		})
	}
}

func TestLigatureFeature(t *testing.T) {
	face := testFace(t)
	on := shapeText(face, 16, "liga")
	off := shapeText(face, 16, "liga", fonts.Feature{Tag: "liga", Value: 0})
	if len(on) != 1 || len(off) != 4 {
		t.Fatalf("liga on gives %d glyphs, off gives %d, want 1 and 4", len(on), len(off))
	}
	if on[0].Cluster != 0 {
		t.Errorf("ligature cluster = %d, want 0", on[0].Cluster)
	}
}

func TestKerningFeature(t *testing.T) {
	face := testFace(t)
	on := advances(shapeText(face, 16, "kk"))
	off := advances(shapeText(face, 16, "kk", fonts.Feature{Tag: "kern", Value: 0}))
	if on[0] >= off[0] {
		t.Errorf("kerned advance %d should be less than unkerned %d", on[0], off[0])
	}
}

func TestFaceFeaturesComeBeforeRunFeatures(t *testing.T) {
	face := testFace(t)
	withFace := *face
	withFace.Features = []fonts.Feature{{Tag: "liga", Value: 0}}
	if got := shapeText(&withFace, 16, "liga"); len(got) != 4 {
		t.Errorf("font-face liga 0 gives %d glyphs, want 4", len(got))
	}
	run := fonts.Feature{Tag: "liga", Value: 1}
	if got := shapeText(&withFace, 16, "liga", run); len(got) != 1 {
		t.Errorf("run liga 1 gives %d glyphs, want 1", len(got))
	}
}

func TestSizeIsTruncatedToTheUnitGrid(t *testing.T) {
	face := testFace(t)
	// 13.333 px is 13652.99 units, and the grid keeps 13652.
	got := shapeText(face, 13.333, "a")
	if got[0].XAdvance != 13652 {
		t.Errorf("advance = %d, want 13652", got[0].XAdvance)
	}
	if got[0].XAdvancePx() != 13652.0/1024 {
		t.Errorf("XAdvancePx = %v", got[0].XAdvancePx())
	}
}

func TestSubRunUsesTextAsContext(t *testing.T) {
	face := testFace(t)
	runes := []rune("kk kk")
	sub := GoText{}.Shape(Run{Text: runes, Start: 3, End: 5, Face: face, Size: 16})
	whole := shapeText(face, 16, "kk")
	if !slices.Equal(sub, whole) {
		t.Errorf("sub-run %v, want %v", sub, whole)
	}
}

func TestClustersAreByteOffsets(t *testing.T) {
	face := testFace(t)
	got := shapeText(face, 16, "aéb")
	var clusters []int
	for _, g := range got {
		clusters = append(clusters, g.Cluster)
	}
	if want := []int{0, 1, 3}; !slices.Equal(clusters, want) {
		t.Errorf("clusters = %v, want %v", clusters, want)
	}
}

func TestMissingGlyphIsReported(t *testing.T) {
	face := testFace(t)
	got := shapeText(face, 16, "a中b")
	if len(got) != 3 {
		t.Fatalf("got %d glyphs, want 3", len(got))
	}
	for i, want := range []bool{false, true, false} {
		if got[i].Missing() != want {
			t.Errorf("glyph %d Missing() = %v, want %v", i, got[i].Missing(), want)
		}
	}
	if got[1].Cluster != 1 {
		t.Errorf("missing glyph cluster = %d, want 1", got[1].Cluster)
	}
}

func TestRTLRunIsInVisualOrder(t *testing.T) {
	face := testFace(t)
	runes := []rune("abc")
	got := GoText{}.Shape(Run{Text: runes, End: 3, Face: face, Size: 16, RTL: true})
	var clusters []int
	for _, g := range got {
		clusters = append(clusters, g.Cluster)
	}
	if want := []int{2, 1, 0}; !slices.Equal(clusters, want) {
		t.Errorf("clusters = %v, want %v", clusters, want)
	}
}

func TestArabicRunJoinsLetters(t *testing.T) {
	face := systemFace(t,
		"/usr/share/fonts/truetype/noto/NotoSansArabic-Regular.ttf",
		"/usr/share/fonts/noto/NotoSansArabic-Regular.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
	)
	if !face.HasGlyph('م') {
		t.Skip("font has no Arabic")
	}
	runes := []rune("مرحبا بالعالم")
	got := GoText{}.Shape(Run{
		Text: runes, End: len(runes), Face: face, Size: 16, RTL: true,
		Lang: language.NewLanguage("ar"),
	})
	if len(got) == 0 {
		t.Fatal("no glyphs")
	}
	for i := 1; i < len(got); i++ {
		if got[i].Cluster > got[i-1].Cluster {
			t.Fatalf("clusters %d then %d: not in visual order", got[i-1].Cluster, got[i].Cluster)
		}
	}
	for _, g := range got {
		if g.Missing() {
			t.Fatal("missing glyph in Arabic text")
		}
	}
}

func TestLatinLigatureWithSystemFont(t *testing.T) {
	face := systemFace(t,
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
	)
	on := shapeText(face, 16, "fi")
	off := shapeText(face, 16, "fi", fonts.Feature{Tag: "liga", Value: 0})
	if len(on) != 1 || len(off) != 2 {
		t.Errorf("liga on gives %d glyphs, off gives %d, want 1 and 2", len(on), len(off))
	}
}
