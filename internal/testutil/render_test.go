package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useFakeRenderer makes HTMLToPixels turn the text of a document into
// pixel art for the duration of the test.
func useFakeRenderer(t *testing.T) {
	t.Helper()
	prev := HTMLToPixels
	t.Cleanup(func() { HTMLToPixels = prev })
	HTMLToPixels = func(t testing.TB, html string) (int, int, []Pixel) {
		t.Helper()
		w, h, p, err := ParsePixels(strings.ReplaceAll(html, "|", "\n"))
		if err != nil {
			t.Fatal(err)
		}
		return w, h, p
	}
}

func TestAssertPixelsUsesRenderer(t *testing.T) {
	useFakeRenderer(t)
	AssertPixels(t, "RB", "RB")
}

func TestAssertPixelsReportsSizeMismatch(t *testing.T) {
	useFakeRenderer(t)
	msg := runRecorded(t, func(tb testing.TB) { AssertPixels(tb, "RBB", "RB") })
	if !strings.Contains(msg, "expected: 3 × 1") || !strings.Contains(msg, "result: 2 × 1") {
		t.Errorf("message = %q", msg)
	}
}

func TestAssertSameRenderings(t *testing.T) {
	useFakeRenderer(t)
	AssertSameRenderings(t, 0, "RB", "RB", "RB")
	AssertSameRenderings(t, 255, "RB", "BR")
}

func TestAssertDifferentRenderings(t *testing.T) {
	useFakeRenderer(t)
	AssertDifferentRenderings(t, "RB", "BR", "RR")
	t.Cleanup(func() { os.Remove(filepath.Join(ResultsDir(t), ResultName(t)+"_1.png")) })
	msg := runRecorded(t, func(tb testing.TB) { AssertDifferentRenderings(tb, "RB", "BR", "RB") })
	if !strings.Contains(msg, "renderings 1 and 3 are the same") {
		t.Errorf("message = %q", msg)
	}
}
