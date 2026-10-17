package layout_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func TestBoxMarginTopRepagination(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page { size: 50px }
        :root { line-height: 1; font-size: 10px }
        a::before { content: target-counter(attr(href), page) }
        div { margin: 20px 0 0; background: yellow }
      </style>
      <p><a href="#title"></a></p>
      <div>1<br/>1<br/>2<br/>2</div>
      <h1 id="title">title</h1>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	children := func(box boxes.Box) []boxes.Box { return boxes.ChildrenOf(box) }

	body := children(children(pages[0])[0])[0]
	div := children(body)[1]
	if got := div.Base().MarginTop.V; got != 20 {
		t.Errorf("page 1 div margin-top = %g, want 20", got)
	}
	if got := div.PaddingBoxY(); got != 10+20 {
		t.Errorf("page 1 div padding box y = %g, want 30", got)
	}

	body = children(children(pages[1])[0])[0]
	div = children(body)[0]
	if got := div.Base().MarginTop.V; got != 0 {
		t.Errorf("page 2 div margin-top = %g, want 0", got)
	}
	if got := div.PaddingBoxY(); got != 0 {
		t.Errorf("page 2 div padding box y = %g, want 0", got)
	}
}
