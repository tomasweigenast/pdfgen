package layout

import (
	"context"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// paginate lays out a document in pages, as LayoutDocument does for the
// whole pipeline, and fails the test on a layout error.
func paginate(t testing.TB, source string) []*boxes.PageBox {
	t.Helper()
	pages, err := paginateWithError(t, source)
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	return pages
}

// paginateWithError is paginate for documents whose layout can fail: it
// returns the first fatal error of the layout.
func paginateWithError(t testing.TB, source string) ([]*boxes.PageBox, error) {
	t.Helper()
	doc := csstest.Parse(t, source)
	env := csstest.Environment()
	collector := targets.NewCollector()
	styles := csstest.StyleFor(t, doc, cascade.Options{Environment: env, Targets: collector})
	builder := &boxes.Builder{
		StyleFor:     styles,
		Targets:      collector,
		CounterStyle: env.CounterStyle,
		GetImage:     images.NewGetter(context.Background(), urls.NewDefaultFetcher(), nil, images.Options{}),
	}
	root, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	return LayoutDocument(doc, root, NewContext(builder), 0)
}

// expectChildren returns the children of a box, failing the test unless
// there are n of them.
func expectChildren(t testing.TB, box boxes.Box, n int) []boxes.Box {
	t.Helper()
	children := boxes.ChildrenOf(unwrapPlaceholder(box))
	if len(children) != n {
		t.Fatalf("%v has %d children, want %d: %v", box, len(children), n, children)
	}
	return children
}

// unwrapPlaceholder returns the box that a placeholder stands for, which
// is the box itself for any other.
func unwrapPlaceholder(box boxes.Box) boxes.Box {
	if placeholder, ok := box.(*boxes.AbsolutePlaceholder); ok {
		return placeholder.Box
	}
	return box
}

// onlyPage fails the test unless there is exactly one page, and returns it.
func onlyPage(t testing.TB, pages []*boxes.PageBox) *boxes.PageBox {
	t.Helper()
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	return pages[0]
}

// pageBody returns the html and body boxes of a page.
func pageBody(t testing.TB, page *boxes.PageBox) (html, body boxes.Box) {
	t.Helper()
	html = expectChildren(t, page, 1)[0]
	if tag := html.Base().ElementTag; tag != "html" {
		t.Fatalf("page child is %q, want html", tag)
	}
	return html, boxes.ChildrenOf(html)[0]
}

func noLogs(t testing.TB) { testutil.AssertNoLogs(t) }
