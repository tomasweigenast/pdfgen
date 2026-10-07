package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// recorder captures Fatalf from AssertPixelsEqual instead of failing the
// enclosing test.
type recorder struct {
	testing.TB
	msg string
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.msg = strings.TrimSpace(fmt.Sprintf(format, args...))
	runtime.Goexit()
}

func (r *recorder) Logf(string, ...any) {}

func runRecorded(tb testing.TB, f func(testing.TB)) string {
	r := &recorder{TB: tb}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(r)
	}()
	<-done
	return r.msg
}

func TestAssertPixelsEqualReportsFirstDifference(t *testing.T) {
	_, _, want, _ := ParsePixels("RB\nBR")
	_, _, got, _ := ParsePixels("RB\nBB")

	name := "TestAssertPixelsEqualReportsFirstDifference_fixture"
	dir := ResultsDir(t)
	t.Cleanup(func() {
		os.Remove(filepath.Join(dir, name+".png"))
		os.Remove(filepath.Join(dir, name+".expected.png"))
	})

	msg := runRecorded(t, func(tb testing.TB) {
		AssertPixelsEqual(tb, name, 2, 2, got, want, 0)
	})
	if want := "Pixel (1, 1) in " + name + ": expected rgb(255, 0, 0), got rgb(0, 0, 255)"; msg != want {
		t.Errorf("message = %q, want %q", msg, want)
	}
	for _, f := range []string{name + ".png", name + ".expected.png"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("result image not written: %v", err)
		}
	}
}

func TestAssertPixelsEqualAcceptsWildcardsAndTolerance(t *testing.T) {
	want := []Pixel{{Any: true}, {R: 100}}
	got := []Pixel{{R: 7, G: 8, B: 9}, {R: 102}}

	if msg := runRecorded(t, func(tb testing.TB) { AssertPixelsEqual(tb, "x", 2, 1, got, want, 2) }); msg != "" {
		t.Errorf("unexpected failure: %s", msg)
	}
	if msg := runRecorded(t, func(tb testing.TB) { AssertPixelsEqual(tb, "TestTolerance_fixture", 2, 1, got, want, 1) }); msg == "" {
		t.Error("difference of 2 passed with tolerance 1")
	}
	dir := ResultsDir(t)
	os.Remove(filepath.Join(dir, "TestTolerance_fixture.png"))
	os.Remove(filepath.Join(dir, "TestTolerance_fixture.expected.png"))
}

func TestResultName(t *testing.T) {
	t.Run("sub test/with spaces", func(t *testing.T) {
		if got, want := ResultName(t), "TestResultName_sub_test_with_spaces"; got != want {
			t.Errorf("ResultName = %q, want %q", got, want)
		}
	})
}
