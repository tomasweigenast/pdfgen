package layout

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestCanBreakInside(t *testing.T) {
	for _, tt := range []struct {
		name, contents string
		want           bool
	}{
		{"words", `<span>a b</span>`, true},
		{"one word", `<span>ab</span>`, false},
		{"nowrap", `<span style="white-space: nowrap">a b</span>`, false},
		{"pre", `<span style="white-space: pre">a b</span>`, false},
		{"pre-wrap", `<span style="white-space: pre-wrap">a b</span>`, true},
		{"nested with a lone space", `<span><b>ab</b> <i>cd</i></span>`, false},
		{"nested words", `<span><b>ab</b><i>c d</i></span>`, true},
		{"nested single word", `<span><b>ab</b><i>cd</i></span>`, false},
		{"inline-block", `<span style="display: inline-block">a b</span>`, false},
		{"image", `<img src="pattern.png">`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			line := boxes.ChildrenOf(parseBox(t, `<p style="font: 10px testfont">`+tt.contents, "p"))[0]
			if got := canBreakInside(boxes.ChildrenOf(line)[0]); got != tt.want {
				t.Fatalf("canBreakInside = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanBreakBefore(t *testing.T) {
	letter := func(r rune) boundary { return boundary{Kind: characterBoundary, Rune: r} }
	for _, tt := range []struct {
		name        string
		whiteSpace  string
		last, first boundary
		want        bool
	}{
		{"between letters", "normal", letter('a'), letter('b'), false},
		{"after a space", "normal", letter(' '), letter('b'), true},
		{"before a space", "normal", letter('a'), letter(' '), false},
		{"after an ideograph", "normal", letter('⺀'), letter('b'), true},
		{"nothing before", "normal", boundary{}, letter('b'), false},
		{"nothing after", "normal", letter(' '), boundary{}, false},
		{"after a collapsed space", "normal", collapsibleSpace, letter('b'), true},
		{"after a leader", "normal", boundary{Kind: leaderEndBoundary}, letter('b'), true},
		{"before a leader", "normal", letter('a'), collapsibleSpace, true},
		{"before a leader end", "normal", letter('a'), boundary{Kind: leaderEndBoundary}, false},
		{"no wrapping", "nowrap", letter(' '), letter('b'), false},
		{"preserved", "pre", letter(' '), letter('b'), false},
		{"pre-wrap", "pre-wrap", letter(' '), letter('b'), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			line := boxes.ChildrenOf(parseBox(t, fmt.Sprintf(
				`<p style="font: 10px testfont; white-space: %s"><span>a b</span>`, tt.whiteSpace), "p"))[0]
			s := &inlineSplit{box: line}
			child := boxes.ChildrenOf(line)[0]
			if got := s.canBreakBefore(child, tt.last, tt.first); got != tt.want {
				t.Fatalf("canBreakBefore = %v, want %v", got, tt.want)
			}
		})
	}
}
