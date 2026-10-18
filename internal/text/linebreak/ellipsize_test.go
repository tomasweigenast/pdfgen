package linebreak

import (
	"testing"
)

func glyphCount(line *Line) (total, ellipsis int) {
	for _, r := range line.Runs {
		total += len(r.Glyphs)
		if r.Ellipsis != "" {
			ellipsis += len(r.Glyphs)
		}
	}
	return total, ellipsis
}

func TestEllipsizeEndKeepsWhatFitsWithTheEllipsis(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "abcde", -1)
	// Glyphs are 16 px wide: two letters and the ellipsis fit in 56 px.
	l.EllipsizeEnd(56)
	line := l.FirstLine()
	if total, ellipsis := glyphCount(line); total != 3 || ellipsis != 1 {
		t.Fatalf("glyphs = %d (ellipsis %d), want 3 (1)", total, ellipsis)
	}
	if w := line.Width(); w != 48*1024 {
		t.Errorf("width = %d, want %d", w, 48*1024)
	}
	if line.Length != 2 {
		t.Errorf("length = %d, want 2", line.Length)
	}
}

func TestEllipsizeEndLeavesShortLinesAlone(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "abc", -1)
	l.EllipsizeEnd(48)
	line := l.FirstLine()
	if total, ellipsis := glyphCount(line); total != 3 || ellipsis != 0 {
		t.Errorf("glyphs = %d (ellipsis %d), want 3 (0)", total, ellipsis)
	}
}

func TestEllipsizeEndWithNoRoomKeepsOnlyTheEllipsis(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "abcde", -1)
	l.EllipsizeEnd(4)
	if total, ellipsis := glyphCount(l.FirstLine()); total != ellipsis || total == 0 {
		t.Errorf("glyphs = %d (ellipsis %d), want only the ellipsis", total, ellipsis)
	}
}

func TestEllipsizeEndIsKeptWhenTheTextChanges(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "abcde", -1)
	l.EllipsizeEnd(56)
	l.SetText("abcdefg")
	if total, ellipsis := glyphCount(l.FirstLine()); total != 3 || ellipsis != 1 {
		t.Errorf("glyphs = %d (ellipsis %d), want 3 (1)", total, ellipsis)
	}
}

func TestFirstLineAndRemainderReportsOverflowingText(t *testing.T) {
	ctx := newTestContext(t)
	l := layoutOf(t, ctx, testStyle(), "ab cd", -1)
	l.Wrap(40)
	line, rest := l.FirstLineAndRemainder()
	if rest != 3 || line.Length != 3 {
		t.Errorf("rest = %d, length = %d, want 3 and 3", rest, line.Length)
	}
	l.Wrap(-1)
	if _, rest := l.FirstLineAndRemainder(); rest != -1 {
		t.Errorf("rest = %d without wrapping, want -1", rest)
	}
}
