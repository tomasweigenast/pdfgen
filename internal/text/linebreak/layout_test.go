package linebreak

import (
	"testing"

	"github.com/go-text/typesetting/bidi"
)

func layoutOf(t *testing.T, ctx *Context, style *Style, text string, wrapWidth float64) *Layout {
	t.Helper()
	l := newLayout(ctx, style, 0, nil)
	if wrapWidth >= 0 {
		l.setWidth(int(wrapWidth * 1024))
	}
	l.SetText(text)
	return l
}

func TestLineSizeOfTestFont(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "abc", -1)
	w, h := LineSize(l.FirstLine(), l.Style)
	if w != 48 || h != 16 {
		t.Errorf("size = %vx%v, want 48x16", w, h)
	}
	if b := l.Baseline(); b != 12.796875 {
		t.Errorf("baseline = %v, want 12.796875", b)
	}
}

func TestEmptyTextHasTheHeightOfTheFont(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "", -1)
	w, h := LineSize(l.FirstLine(), l.Style)
	if w != 0 || h != 16 {
		t.Errorf("size = %vx%v, want 0x16", w, h)
	}
}

func TestLetterSpacingGoesBetweenCharactersOfALine(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.LetterSpacing = 2
	l := layoutOf(t, ctx, style, "abc", -1)
	line := l.FirstLine()
	// Two gaps of 2 px inside the line; the spacing after the last
	// character is added back by LineSize.
	if got := line.Width(); got != 52*1024 {
		t.Errorf("line width = %d, want %d", got, 52*1024)
	}
	if w, _ := LineSize(line, style); w != 54 {
		t.Errorf("LineSize width = %v, want 54", w)
	}
	var advances []int32
	for _, g := range line.Runs[0].Glyphs {
		advances = append(advances, g.XAdvance)
	}
	want := []int32{17 * 1024, 18 * 1024, 17 * 1024}
	for i := range want {
		if advances[i] != want[i] {
			t.Fatalf("advances = %v, want %v", advances, want)
		}
	}
}

func TestWordSpacingWidensSeparators(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.WordSpacing = 5
	l := layoutOf(t, ctx, style, "a b\u00a0c", -1)
	if w, _ := LineSize(l.FirstLine(), style); w != 5*16+2*5 {
		t.Errorf("width = %v, want %v", w, 5*16+2*5)
	}
	// A separator at an end of the text keeps the whole of its spacing.
	l = layoutOf(t, ctx, style, " a ", -1)
	if w, _ := LineSize(l.FirstLine(), style); w != 3*16+2*5 {
		t.Errorf("width = %v, want %v", w, 3*16+2*5)
	}
}

func TestSingleSpaceGetsWordSpacing(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.WordSpacing = 4
	style.LetterSpacing = 2
	l := layoutOf(t, ctx, style, " ", -1)
	if l.Text != " \u200b" {
		t.Errorf("text = %q, want a space and a zero-width space", l.Text)
	}
	if w, _ := LineSize(l.FirstLine(), style); w != 24 {
		t.Errorf("width = %v, want 24", w)
	}
}

func TestTabsExtendToTabStops(t *testing.T) {
	ctx := newTestContext(t)
	tests := []struct {
		text  string
		tab   TabSize
		width float64
	}{
		{"a\tb", TabSize{Spaces: 8}, 8*16 + 16},
		{"a\tb", TabSize{Spaces: 2}, 2*16 + 16},
		{"aaa\tb", TabSize{Spaces: 2}, 4*16 + 16},
		{"\t\tb", TabSize{Spaces: 1}, 3 * 16},
		{"a\tb", TabSize{Length: 20, IsLength: true}, 20 + 16},
		// Tab stops cannot be closer than 1 px.
		{"a\tb", TabSize{IsLength: true}, 2*16 + 1},
	}
	for _, tt := range tests {
		style := testStyle()
		style.TabSize = tt.tab
		l := layoutOf(t, ctx, style, tt.text, -1)
		if w, _ := LineSize(l.FirstLine(), style); w != tt.width {
			t.Errorf("%q with %+v: width = %v, want %v", tt.text, tt.tab, w, tt.width)
		}
	}
}

func TestTextIsCutAfterTheFirstLineFeed(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "ab\ncd\nef", -1)
	if l.Text != "ab\nc" {
		t.Errorf("text = %q, want %q", l.Text, "ab\nc")
	}
	line, next := l.firstLine()
	if line.Length != 2 || next != 3 {
		t.Errorf("line length %d, next line at %d; want 2 and 3", line.Length, next)
	}
}

func TestLinesWrapAtBreakOpportunities(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "aaa bbb ccc", 120)
	line, next := l.firstLine()
	if line.Length != 8 || next != 8 {
		t.Errorf("line length %d, next line at %d; want 8 and 8", line.Length, next)
	}
	// The space ending a wrapped line takes no room.
	if got := line.Width(); got != 7*16*1024 {
		t.Errorf("width = %d, want %d", got, 7*16*1024)
	}
}

func TestWordTooLongForTheLineOverflows(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "aaaaaa bb", 40)
	line, next := l.firstLine()
	if line.Length != 7 || next != 7 {
		t.Errorf("line length %d, next line at %d; want 7 and 7", line.Length, next)
	}
}

func TestCharacterWrappingBreaksWords(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.OverflowWrap = "anywhere"
	l := layoutOf(t, ctx, style, "aaaaaa", 40)
	l.setWrap(wrapChar)
	line, next := l.firstLine()
	if line.Length != 2 || next != 2 {
		t.Errorf("line length %d, next line at %d; want 2 and 2", line.Length, next)
	}
}

func TestSoftHyphenIsInvisibleUnlessALineEndsThere(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "aaa\u00adbbb", -1)
	if w := l.FirstLine().Width(); w != 6*16*1024 {
		t.Errorf("width = %d, want %d", w, 6*16*1024)
	}
	l = layoutOf(t, ctx, testStyle(), "aaa\u00adbbb", 70)
	line, next := l.firstLine()
	if line.Length != 5 || next != 5 {
		t.Errorf("line length %d, next line at %d; want 5 and 5", line.Length, next)
	}
	// The line shows a hyphen.
	if w := line.Width(); w != 4*16*1024 {
		t.Errorf("width = %d, want %d", w, 4*16*1024)
	}
}

func TestRightToLeftTextIsReordered(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "ab אב cd", -1)
	line := l.FirstLine()
	var levels []bidi.Level
	for _, r := range line.Runs {
		levels = append(levels, r.Level)
	}
	// The space after the Hebrew letters keeps their script, and is a run
	// of its own.
	if len(levels) != 4 || levels[0] != 0 || levels[1] != 1 || levels[2] != 0 || levels[3] != 0 {
		t.Fatalf("levels = %v, want [0 1 0 0]", levels)
	}
	hebrew := line.Runs[1]
	if hebrew.Glyphs[0].Cluster <= hebrew.Glyphs[1].Cluster {
		t.Errorf("right-to-left glyphs must be in visual order: %+v", hebrew.Glyphs)
	}

	style := testStyle()
	style.Direction = "rtl"
	l = layoutOf(t, ctx, style, "ab אב cd", -1)
	line = l.FirstLine()
	if !line.RTL || len(line.Runs) < 2 {
		t.Fatalf("expected a right-to-left line with several runs, got %+v", line)
	}
	if first := line.Runs[0]; l.Text[first.Start:first.End] != "cd" {
		t.Errorf("first visual run = %q, want %q", l.Text[first.Start:first.End], "cd")
	}
}

func TestReorderRunsByLevel(t *testing.T) {
	runs := func(levels ...bidi.Level) []*Run {
		out := make([]*Run, len(levels))
		for i, l := range levels {
			out[i] = &Run{Level: l, Start: i}
		}
		return out
	}
	order := func(rs []*Run) []int {
		out := make([]int, len(rs))
		for i, r := range rs {
			out[i] = r.Start
		}
		return out
	}
	tests := []struct {
		levels []bidi.Level
		want   []int
	}{
		{[]bidi.Level{0, 0, 0}, []int{0, 1, 2}},
		{[]bidi.Level{1, 1, 1}, []int{2, 1, 0}},
		{[]bidi.Level{0, 1, 1, 0}, []int{0, 2, 1, 3}},
		{[]bidi.Level{1, 2, 2, 1}, []int{3, 1, 2, 0}},
		{[]bidi.Level{0, 1, 2, 1, 0}, []int{0, 3, 2, 1, 4}},
	}
	for _, tt := range tests {
		got := order(reorderRuns(runs(tt.levels...)))
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("levels %v: order %v, want %v", tt.levels, got, tt.want)
				break
			}
		}
	}
}

func TestScriptRunsResolveCommonCharacters(t *testing.T) {
	text := []rune("(a) א [ב]")
	scripts := scriptRuns(text)
	latin, hebrew := scripts[1], scripts[4]
	want := []int{0, 0, 0, 0, 1, 1, 1, 1, 1}
	for i, w := range want {
		expected := latin
		if w == 1 {
			expected = hebrew
		}
		if scripts[i] != expected {
			t.Errorf("script of %q = %v, want %v", text[i], scripts[i], expected)
		}
	}
}
