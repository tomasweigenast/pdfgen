package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// renderWithHints lays out a document with the default user agent
// stylesheets and the page setup of the hints tests, and returns the html
// and body boxes of its only page. Hints are applied when withHints is set.
func renderWithHints(t *testing.T, source string, withHints bool) (html, body boxes.Box) {
	t.Helper()
	opts := []boxtest.Option{boxtest.WithDefaultUA(), boxtest.WithStylesheet(hintsSheet)}
	if withHints {
		opts = append(opts, boxtest.WithPresentationalHints())
	}
	pages := boxtest.FakeHTML(t, source, opts...).RenderPages(t)
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	return descend(t, pages[0], 0), descend(t, pages[0], 0, 0)
}

func TestHorizontalRuleSizeAttributeIsIgnoredWithoutHints(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `<hr size=100 />`, false)
	children := boxes.ChildrenOf(body)
	if len(children) != 1 {
		t.Fatalf("body has %d children, want 1", len(children))
	}
	if h := children[0].BorderHeight(); h == 100 {
		t.Errorf("hr border height = %g, want the size attribute to be ignored", h)
	}
}

func TestTableAlignAttributeIsIgnoredWithoutHints(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `
      <hr size=100 />
      <table align=right width=100><td>0</td></table>
    `, false)
	children := boxes.ChildrenOf(body)
	if len(children) != 2 {
		t.Fatalf("body has %d children, want 2", len(children))
	}
	if h := children[0].BorderHeight(); h == 100 {
		t.Errorf("hr border height = %g, want the size attribute to be ignored", h)
	}
	if x := children[1].Base().PositionX; x != 0 {
		t.Errorf("table position-x = %g, want 0", x)
	}
}

func TestBodyMarginAttributesGiveTheMarginsOfLaidOutBody(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `
      <body marginheight=2 leftmargin=5 bgcolor=red text=blue />
    `, true)
	b := body.Base()
	got := [4]float64{b.MarginTop.V, b.MarginBottom.V, b.MarginLeft.V, b.MarginRight.V}
	if want := [4]float64{2, 2, 5, 5}; got != want {
		t.Errorf("margins (top bottom left right) = %v, want %v", got, want)
	}
}

func TestLineBreakHintsMakeOneLinePerBreak(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `
      <br clear=left>
      <br clear=right />
      <br clear=both />
      <br clear=all />
      <font color=red face=testfont size=7></font>
      <Font size=4></Font>
      <font size=+5 ></font>
      <font size=-5 ></font>
    `, true)
	lines := boxes.ChildrenOf(body)
	if len(lines) != 5 {
		t.Fatalf("body has %d lines, want 5", len(lines))
	}
	for i, line := range lines[:4] {
		if n := len(boxes.ChildrenOf(line)); n != 1 {
			t.Errorf("line %d has %d children, want the br", i+1, n)
		}
	}
	if n := len(boxes.ChildrenOf(lines[4])); n != 4 {
		t.Errorf("last line has %d children, want the four fonts", n)
	}
}

func TestTableHintsGiveTheSizeOfRowsAndCells(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `
      <table align=left rules=none></table>
      <table align=right rules=groups></table>
      <table align=center rules=rows></table>
      <table border=10 cellspacing=3 bordercolor=green>
        <thead>
          <tr>
            <th valign=top></th>
          </tr>
        </thead>
        <tr>
          <td nowrap><h1 align=right></h1><p align=center></p></td>
        </tr>
        <tr height=0>
        </tr>
        <tfoot align=justify>
          <tr>
            <td width=1 height=10></td>
          </tr>
        </tfoot>
      </table>
    `, true)
	wrappers := boxes.ChildrenOf(body)
	if len(wrappers) != 4 {
		t.Fatalf("body has %d children, want 4", len(wrappers))
	}
	rowsGroup := descend(t, wrappers[3], 0, 1)
	rows := boxes.ChildrenOf(rowsGroup)
	if len(rows) != 2 {
		t.Fatalf("body group has %d rows, want 2", len(rows))
	}
	if h := rows[1].Base().Height.V; h != 0 {
		t.Errorf("empty row height = %g, want 0", h)
	}
	td := descend(t, wrappers[3], 0, 2, 0, 0).Base()
	if td.Height.V != 10 || td.Width.V != 1 {
		t.Errorf("cell size = %gx%g, want 1x10", td.Width.V, td.Height.V)
	}
}

func TestHorizontalRuleHintsGiveItsGeometry(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `
      <hr align=left>
      <hr align=right />
      <hr align=both color=red />
      <hr align=center noshade size=10 />
      <hr align=all size=8 width=100 />
    `, true)
	rules := boxes.ChildrenOf(body)
	if len(rules) != 5 {
		t.Fatalf("body has %d children, want 5", len(rules))
	}
	if m := rules[0].Base().MarginLeft.V; m != 0 {
		t.Errorf("hr 1 margin-left = %g, want 0", m)
	}
	if m := rules[1].Base().MarginRight.V; m != 0 {
		t.Errorf("hr 2 margin-right = %g, want 0", m)
	}
	if h := rules[3].BorderHeight(); h != 10 {
		t.Errorf("hr 4 border height = %g, want 10", h)
	}
	hr5 := rules[4].Base()
	if h := rules[4].BorderHeight(); h != 8 {
		t.Errorf("hr 5 border height = %g, want 8", h)
	}
	if hr5.Height.V != 6 || hr5.Width.V != 100 {
		t.Errorf("hr 5 content size = %gx%g, want 100x6", hr5.Width.V, hr5.Height.V)
	}
}

func TestEmbeddedContentHints(t *testing.T) {
	t.Skip("not supported: SVG images")
	testutil.AssertNoLogs(t)
	_, body := renderWithHints(t, `
      <object data="data:image/svg+xml,<svg></svg>"
              align=top hspace=10 vspace=20></object>
      <img src="data:image/svg+xml,<svg></svg>" alt=text
              align=right width=10 height=20 />
      <embed src="data:image/svg+xml,<svg></svg>" align=texttop />
    `, true)
	line := descend(t, body, 0)
	items := boxes.ChildrenOf(line)
	if len(items) != 5 {
		t.Fatalf("line has %d children, want 5", len(items))
	}
	object, img, embed := items[0], items[2], items[3]
	if v := embed.Base().Style.VerticalAlign(); v != props.LengthKeyword("text-top") {
		t.Errorf("embed vertical-align = %v, want text-top", v)
	}
	if v := object.Base().Style.VerticalAlign(); v != props.LengthKeyword("top") {
		t.Errorf("object vertical-align = %v, want top", v)
	}
	if m := object.Base().MarginTop.V; m != 20 {
		t.Errorf("object margin-top = %g, want 20", m)
	}
	if m := object.Base().MarginLeft.V; m != 10 {
		t.Errorf("object margin-left = %g, want 10", m)
	}
	if f := img.Base().Style.Float(); f != "right" {
		t.Errorf("img float = %s, want right", f)
	}
	if w, h := img.Base().Width.V, img.Base().Height.V; w != 10 || h != 20 {
		t.Errorf("img size = %gx%g, want 10x20", w, h)
	}
}
