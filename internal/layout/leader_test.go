package layout

import (
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

// leaderLine lays out a document and returns its first line box.
func leaderLine(t *testing.T, source string) *boxes.LineBox {
	t.Helper()
	doc := renderPage(t, source)
	for d := range boxes.Descendants(doc.html, false) {
		if line, ok := d.(*boxes.LineBox); ok {
			return line
		}
	}
	t.Fatal("document has no line")
	return nil
}

// textsAt returns the text and the horizontal position of each text box
// under a box.
func textsAt(box boxes.Box) (texts []string, xs []float64) {
	for d := range boxes.Descendants(box, false) {
		if text, ok := d.(*boxes.TextBox); ok {
			texts = append(texts, text.Text)
			xs = append(xs, text.PositionX)
		}
	}
	return texts, xs
}

func TestLeaderFillsTheSpaceBetweenTextAndMovesWhatFollows(t *testing.T) {
	line := leaderLine(t, `
		<style>
			@page { size: 100px; margin: 0 }
			p { font: 10px/1 testfont; margin: 0 }
			span::before { content: leader('.') }
		</style>
		<p>ab<span></span>cd</p>`)
	texts, xs := textsAt(line)
	wantTexts := []string{"ab", ".", ".", ".", ".", ".", ".", "cd"}
	wantXs := []float64{0, 20, 30, 40, 50, 60, 70, 80}
	if !slices.Equal(texts, wantTexts) {
		t.Errorf("texts = %q, want %q", texts, wantTexts)
	}
	// The leader repeats from the right end of its space.
	if got := slices.Sorted(slices.Values(xs)); !slices.Equal(got, wantXs) {
		t.Errorf("positions = %v, want %v", got, wantXs)
	}
	if line.Width.V != 100 {
		t.Errorf("line width = %g, want 100", line.Width.V)
	}
}

func TestLeaderInNestedInlineBoxWidensItsParents(t *testing.T) {
	line := leaderLine(t, `
		<style>
			@page { size: 100px; margin: 0 }
			p { font: 10px/1 testfont; margin: 0 }
			b::after { content: leader('.') }
		</style>
		<p><b>ab</b>cd</p>`)
	bold := at(t, line, 0)
	if got := bold.Base().Width.V; got != 80 {
		t.Errorf("width of the box with the leader = %g, want 80", got)
	}
	if _, xs := textsAt(line); xs[len(xs)-1] != 80 {
		t.Errorf("text after the leader is at %g, want 80", xs[len(xs)-1])
	}
}

func TestLeaderWithoutFreeSpaceKeepsOneRepetition(t *testing.T) {
	line := leaderLine(t, `
		<style>
			@page { size: 50px; margin: 0 }
			p { font: 10px/1 testfont; margin: 0 }
			span::before { content: leader('.') }
		</style>
		<p>abc<span></span>d</p>`)
	texts, xs := textsAt(line)
	if want := []string{"abc", ".", "d"}; !slices.Equal(texts, want) {
		t.Errorf("texts = %q, want %q", texts, want)
	}
	if want := []float64{0, 30, 40}; !slices.Equal(xs, want) {
		t.Errorf("positions = %v, want %v", xs, want)
	}
}
