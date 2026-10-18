package anchors_test

import (
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/anchors"
)

func TestTreeBuilderNestsBySkippingMissingLevels(t *testing.T) {
	// h1 h4 h3 h2 h1: the h4 right after the h1 is its first child, as are
	// the h3 and the h2 after it.
	levels := []int{1, 4, 3, 2, 1}
	var bookmarks []anchors.Bookmark
	for i, level := range levels {
		bookmarks = append(bookmarks, anchors.Bookmark{Level: level, Label: string(rune('a' + i)), X: 1, Y: float64(i)})
	}
	b := anchors.NewTreeBuilder()
	b.AddPage(bookmarks, 0, matrix.Identity)

	node := func(label string, y float64) anchors.Node { return anchors.Node{Label: label, X: 1, Y: y} }
	want := []anchors.Node{
		{Label: "a", X: 1, Y: 0, Children: []anchors.Node{node("b", 1), node("c", 2), node("d", 3)}},
		node("e", 4),
	}
	if got := b.Tree(); !reflect.DeepEqual(got, want) {
		t.Errorf("tree = %+v, want %+v", got, want)
	}
}

func TestTreeBuilderContinuesAcrossPages(t *testing.T) {
	b := anchors.NewTreeBuilder()
	m := matrix.New(2, 0, 0, -2, 0, 100)
	b.AddPage([]anchors.Bookmark{{Level: 1, Label: "a", X: 10, Y: 20}}, 0, m)
	b.AddPage([]anchors.Bookmark{{Level: 2, Label: "b", X: 5, Y: 10, State: "closed"}}, 1, m)

	want := []anchors.Node{{
		Label: "a", Page: 0, X: 20, Y: 60,
		Children: []anchors.Node{{Label: "b", Page: 1, X: 10, Y: 80, State: "closed"}},
	}}
	if got := b.Tree(); !reflect.DeepEqual(got, want) {
		t.Errorf("tree = %+v, want %+v", got, want)
	}
}
