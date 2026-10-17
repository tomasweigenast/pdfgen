package layout_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// renderPages lays out a document and fails the test if anything is logged.
func renderPages(t *testing.T, source string) []*boxes.PageBox {
	t.Helper()
	testutil.AssertNoLogs(t)
	return boxtest.RenderPages(t, source)
}

// onlyPage fails the test unless there is exactly one page, and returns it.
func onlyPage(t testing.TB, pages []*boxes.PageBox) *boxes.PageBox {
	t.Helper()
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	return pages[0]
}

// pagesOf fails the test unless there are n pages, and returns them.
func pagesOf(t testing.TB, pages []*boxes.PageBox, n int) []*boxes.PageBox {
	t.Helper()
	if len(pages) != n {
		t.Fatalf("got %d pages, want %d", len(pages), n)
	}
	return pages
}

// children returns the children of a box, failing the test unless there are
// n of them. An absolute placeholder stands for the box it holds.
func children(t testing.TB, box boxes.Box, n int) []boxes.Box {
	t.Helper()
	if placeholder, ok := box.(*boxes.AbsolutePlaceholder); ok {
		box = placeholder.Box
	}
	list := boxes.ChildrenOf(box)
	if len(list) != n {
		t.Fatalf("%v has %d children, want %d: %v", box, len(list), n, list)
	}
	return list
}

// child returns the only child of a box.
func child(t testing.TB, box boxes.Box) boxes.Box {
	t.Helper()
	return children(t, box, 1)[0]
}

// pageBody returns the html and body boxes of a page that has only the
// content of the document.
func pageBody(t testing.TB, page boxes.Box) (html, body boxes.Box) {
	t.Helper()
	html = child(t, page)
	if tag := html.Base().ElementTag; tag != "html" {
		t.Fatalf("the first child of the page is %q, want html", tag)
	}
	return html, child(t, html)
}

// textOf returns the text of a box and its descendants.
func textOf(box boxes.Box) string {
	var text strings.Builder
	for d := range boxes.Descendants(box, false) {
		if t, ok := d.(*boxes.TextBox); ok {
			text.WriteString(t.Text)
		}
	}
	return text.String()
}

// tagOf returns the element tag of a box.
func tagOf(box boxes.Box) string { return box.Base().ElementTag }

// pageSize returns the width and the height of the page box.
func pageSize(page *boxes.PageBox) [2]float64 {
	return [2]float64{page.Base().Width.V, page.Base().Height.V}
}
