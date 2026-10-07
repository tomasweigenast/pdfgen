package shape

import (
	"slices"
	"testing"
)

func TestLetterSpacingGoesBetweenCharacters(t *testing.T) {
	face := testFace(t)
	text := []rune("abc")
	base := shapeText(face, 16, "abc")
	glyphs := slices.Clone(base)
	ApplySpacing(glyphs, text, false, Spacing{Letter: 2048}, Edges{Start: true, End: true})

	want := []int32{base[0].XAdvance + 1024, base[1].XAdvance + 2048, base[2].XAdvance + 1024}
	if got := advances(glyphs); !slices.Equal(got, want) {
		t.Errorf("advances = %v, want %v", got, want)
	}
	if glyphs[0].XOffset != 0 || glyphs[1].XOffset != 1024 || glyphs[2].XOffset != 1024 {
		t.Errorf("offsets = %d %d %d, want 0 1024 1024", glyphs[0].XOffset, glyphs[1].XOffset, glyphs[2].XOffset)
	}
}

func TestLetterSpacingInTheMiddleOfATextKeepsBothHalves(t *testing.T) {
	face := testFace(t)
	base := shapeText(face, 16, "abc")
	glyphs := slices.Clone(base)
	ApplySpacing(glyphs, []rune("abc"), false, Spacing{Letter: 2048}, Edges{})
	for i, g := range glyphs {
		if g.XAdvance != base[i].XAdvance+2048 {
			t.Errorf("glyph %d advance = %d, want %d", i, g.XAdvance, base[i].XAdvance+2048)
		}
	}
}

func TestWordSpacingAddsToSeparators(t *testing.T) {
	face := testFace(t)
	text := []rune("a b c")
	base := shapeText(face, 16, string(text))
	glyphs := slices.Clone(base)
	ApplySpacing(glyphs, text, false, Spacing{Word: 4096}, Edges{Start: true, End: true})
	for i, g := range glyphs {
		want := base[i].XAdvance
		if text[i] == ' ' || text[i] == ' ' {
			want += 4096
		}
		if g.XAdvance != want {
			t.Errorf("glyph %d advance = %d, want %d", i, g.XAdvance, want)
		}
	}
}

func TestSeparatorAtTheEndsKeepsTheWholeWordSpacing(t *testing.T) {
	face := testFace(t)
	text := []rune(" a ")
	base := shapeText(face, 16, string(text))
	glyphs := slices.Clone(base)
	ApplySpacing(glyphs, text, false, Spacing{Word: 4096}, Edges{Start: true, End: true})
	var sum, baseSum int32
	for i := range glyphs {
		sum += glyphs[i].XAdvance
		baseSum += base[i].XAdvance
	}
	if sum-baseSum != 2*4096 {
		t.Errorf("total spacing = %d, want %d", sum-baseSum, 2*4096)
	}
}

func TestSpacingOfAFirstRunTrimsOnlyItsLeftEdge(t *testing.T) {
	face := testFace(t)
	text := []rune("ab")
	base := shapeText(face, 16, "ab")
	first := slices.Clone(base)
	ApplySpacing(first, text, false, Spacing{Letter: 2048}, Edges{Start: true})
	if first[0].XAdvance != base[0].XAdvance+1024 || first[1].XAdvance != base[1].XAdvance+2048 {
		t.Errorf("first run advances = %v", advances(first))
	}
}

func TestSpacingOnRTLRunTrimsTheVisualEnds(t *testing.T) {
	face := testFace(t)
	text := []rune("abc")
	glyphs := GoText{}.Shape(Run{Text: text, End: 3, Face: face, Size: 16, RTL: true})
	base := slices.Clone(glyphs)
	ApplySpacing(glyphs, text, true, Spacing{Letter: 2048}, Edges{Start: true, End: true})

	if glyphs[0].XOffset != 0 || glyphs[0].XAdvance != base[0].XAdvance+1024 {
		t.Errorf("visually first glyph: offset %d advance %d", glyphs[0].XOffset, glyphs[0].XAdvance)
	}
	last := len(glyphs) - 1
	if glyphs[last].XAdvance != base[last].XAdvance+1024 || glyphs[last].XOffset != 1024 {
		t.Errorf("visually last glyph: offset %d advance %d", glyphs[last].XOffset, glyphs[last].XAdvance)
	}
}

func TestSpacingTreatsAClusterAsOneUnit(t *testing.T) {
	face := testFace(t)
	text := []rune("liga")
	glyphs := shapeText(face, 16, "liga")
	base := slices.Clone(glyphs)
	ApplySpacing(glyphs, text, false, Spacing{Letter: 2048}, Edges{})
	if len(glyphs) != 1 || glyphs[0].XAdvance != base[0].XAdvance+2048 {
		t.Errorf("ligature advance = %d, want %d", glyphs[0].XAdvance, base[0].XAdvance+2048)
	}
}

func TestZeroSpacingChangesNothing(t *testing.T) {
	face := testFace(t)
	glyphs := shapeText(face, 16, "abc")
	base := slices.Clone(glyphs)
	ApplySpacing(glyphs, []rune("abc"), false, Spacing{}, Edges{Start: true, End: true})
	if !slices.Equal(glyphs, base) {
		t.Error("glyphs changed")
	}
}
