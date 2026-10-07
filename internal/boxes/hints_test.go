package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// hintsSheet is the user stylesheet of the documents that test
// presentational hints.
const hintsSheet = `
@page {margin: 0; size: 1000px 1000px}
body {margin: 0}
`

// parseWithHints builds the box tree of a document with the default user
// agent stylesheets and presentational hints, and returns the children of
// its body.
func parseWithHints(t *testing.T, source string) []boxes.Box {
	t.Helper()
	return body(t, boxtest.ParseAllWith(t, source, hintsOptions))
}

var hintsOptions = boxtest.Options{
	DefaultUA:           true,
	PresentationalHints: true,
	Stylesheets:         []string{hintsSheet},
}

// elementBoxes returns the boxes of the elements with the given tag in a
// tree, in tree order, leaving out the boxes of their pseudo-elements.
func elementBoxes(list []boxes.Box, tag string) []boxes.Box {
	var out []boxes.Box
	for _, root := range list {
		for d := range boxes.Descendants(root, false) {
			if d.Base().ElementTag == tag && !d.Base().Style.IsAnonymous() {
				out = append(out, d)
			}
		}
	}
	return out
}

func isRGB(c props.Color, r, g, b float64) bool {
	return len(c.Coordinates) == 3 && c.Coordinates[0] == r && c.Coordinates[1] == g && c.Coordinates[2] == b && c.Alpha == 1
}

func TestHintsOfTheBody(t *testing.T) {
	testutil.AssertNoLogs(t)
	root := boxtest.ParseAllWith(t, `
      <body marginheight=2 leftmargin=5 bgcolor=red text=blue />
    `, hintsOptions)
	style := onlyChild(t, root).Base().Style
	if c := style.BackgroundColor(); !isRGB(c, 1, 0, 0) {
		t.Errorf("background-color = %v", c)
	}
	if c := style.Color(); !isRGB(c, 0, 0, 1) {
		t.Errorf("color = %v", c)
	}
}

func TestHintsOfFlowContent(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
      <pre wrap></pre>
      <center></center>
      <div align=center></div>
      <div align=middle></div>
      <div align=left></div>
      <div align=right></div>
      <div align=justify></div>
    `)
	if len(children) != 7 {
		t.Fatalf("body has %d children, want 7", len(children))
	}
	if ws := children[0].Base().Style.WhiteSpace(); ws != "pre-wrap" {
		t.Errorf("pre white-space = %s", ws)
	}
	for i, want := range []props.Keyword{"center", "center", "center", "left", "right", "justify"} {
		if got := children[i+1].Base().Style.TextAlignAll(); got != want {
			t.Errorf("child %d text-align = %s, want %s", i+1, got, want)
		}
	}
}

func TestHintsOfPhrasingContent(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
      <br clear=left>
      <br clear=right />
      <br clear=both />
      <br clear=all />
      <font color=red face=testfont size=7></font>
      <Font size=4></Font>
      <font size=+5 ></font>
      <font size=-5 ></font>
    `)
	brs := elementBoxes(children, "br")
	fonts := elementBoxes(children, "font")
	if len(brs) != 4 || len(fonts) != 4 {
		t.Fatalf("%d br and %d font boxes, want 4 of each", len(brs), len(fonts))
	}
	for i, want := range []props.Keyword{"left", "right", "both", "both"} {
		if got := brs[i].Base().Style.Clear(); got != want {
			t.Errorf("br %d clear = %s, want %s", i+1, got, want)
		}
	}
	first := fonts[0].Base().Style
	if c := first.Color(); !isRGB(c, 1, 0, 0) {
		t.Errorf("font color = %v", c)
	}
	if f := first.FontFamily(); len(f) != 1 || f[0] != "testfont" {
		t.Errorf("font-family = %v", f)
	}
	for i, want := range []float64{1.5 * 2 * 16, 6.0 / 5 * 16, 1.5 * 2 * 16, 8.0 / 9 * 16} {
		if got := fonts[i].Base().Style.FontSize(); float64(got) != want {
			t.Errorf("font %d size = %v, want %v", i+1, got, want)
		}
	}
}

func listStyleTypes(t *testing.T, list []boxes.Box) []string {
	t.Helper()
	var out []string
	for _, b := range list {
		out = append(out, b.Base().Style.ListStyleType().Name)
	}
	return out
}

func TestHintsOfListItems(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
      <ol>
        <li type=A></li>
        <li type=1></li>
        <li type=a></li>
        <li type=i></li>
        <li type=I></li>
      </ol>
      <ul>
        <li type=circle></li>
        <li type=disc></li>
        <li type=square></li>
      </ul>
    `)
	if len(children) != 2 {
		t.Fatalf("body has %d children, want 2", len(children))
	}
	got := append(listStyleTypes(t, boxes.ChildrenOf(children[0])), listStyleTypes(t, boxes.ChildrenOf(children[1]))...)
	want := []string{"upper-alpha", "decimal", "lower-alpha", "lower-roman", "upper-roman", "circle", "disc", "square"}
	if len(got) != len(want) {
		t.Fatalf("list style types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("list style types = %v, want %v", got, want)
			break
		}
	}
}

func TestHintsOfLists(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
      <ol type=A></ol>
      <ol type=1></ol>
      <ol type=a></ol>
      <ol type=i></ol>
      <ol type=I></ol>
      <ul type=circle></ul>
      <ul type=disc></ul>
      <ul type=square></ul>
    `)
	got := listStyleTypes(t, children)
	want := []string{"upper-alpha", "decimal", "lower-alpha", "lower-roman", "upper-roman", "circle", "disc", "square"}
	if len(got) != len(want) {
		t.Fatalf("list style types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("list style types = %v, want %v", got, want)
			break
		}
	}
}

func isListItemReset(style *computed.Style, value int) bool {
	items := style.CounterReset().Items
	return len(items) == 1 && items[0] == props.CounterOp{Name: "list-item", Value: value}
}

func TestHintsOfListStart(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
      <ol start=3>
        <li></li>
      </ol>
      <ul>
        <li value=3></li>
      </ul>
    `)
	if len(children) != 2 {
		t.Fatalf("body has %d children, want 2", len(children))
	}
	if style := children[0].Base().Style; !isListItemReset(style, 3) {
		t.Errorf("ol counter-reset = %v", style.CounterReset())
	}
	if style := onlyChild(t, children[1]).Base().Style; !isListItemReset(style, 3) {
		t.Errorf("li counter-reset = %v", style.CounterReset())
	}
}

func TestHintsOfTables(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
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
    `)
	if len(children) != 4 {
		t.Fatalf("body has %d children, want 4", len(children))
	}
	if f := children[0].Base().Style.Float(); f != "left" {
		t.Errorf("wrapper 1 float = %s", f)
	}
	if f := children[1].Base().Style.Float(); f != "right" {
		t.Errorf("wrapper 2 float = %s", f)
	}
	auto := props.LengthKeyword("auto")
	if s := children[2].Base().Style; s.MarginLeft() != auto || s.MarginRight() != auto {
		t.Errorf("wrapper 3 margins = %v %v", s.MarginLeft(), s.MarginRight())
	}
	for i, wrapper := range children[:3] {
		if s := boxes.ChildrenOf(wrapper)[0].Base().Style.BorderLeftStyle(); s != "hidden" {
			t.Errorf("table %d border-left-style = %s", i+1, s)
		}
		if c := wrapper.Base().Style.BorderCollapse(); c != "collapse" {
			t.Errorf("wrapper %d border-collapse = %s", i+1, c)
		}
	}

	table := onlyChild(t, children[3]).Base().Style
	if s := table.BorderTopStyle(); s != "outset" {
		t.Errorf("table border-top-style = %s", s)
	}
	if w := table.BorderTopWidth(); w != 10 {
		t.Errorf("table border-top-width = %v", w)
	}
	if s := table.BorderSpacing(); s != (props.PixelPair{X: 3, Y: 3}) {
		t.Errorf("table border-spacing = %v", s)
	}
	if c := table.BorderLeftColor().Coordinates; !(c[1] > c[0] && c[1] > c[2]) {
		t.Errorf("table border-left-color = %v", c)
	}

	groups := boxes.ChildrenOf(onlyChild(t, children[3]))
	if len(groups) != 3 {
		t.Fatalf("table has %d row groups, want 3", len(groups))
	}
	th := onlyChild(t, onlyChild(t, groups[0])).Base().Style
	if v := th.VerticalAlign(); v != props.LengthKeyword("top") {
		t.Errorf("th vertical-align = %v", v)
	}
	rows := boxes.ChildrenOf(groups[1])
	if len(rows) != 2 {
		t.Fatalf("body group has %d rows, want 2", len(rows))
	}
	td := onlyChild(t, rows[0])
	if s := td.Base().Style; s.WhiteSpace() != "nowrap" || s.BorderTopWidth() != 1 || s.BorderTopStyle() != "inset" {
		t.Errorf("td white-space %s, border-top %v %s", s.WhiteSpace(), s.BorderTopWidth(), s.BorderTopStyle())
	}
	blocks := boxes.ChildrenOf(td)
	if len(blocks) != 2 {
		t.Fatalf("td has %d children, want 2", len(blocks))
	}
	if a := blocks[0].Base().Style.TextAlignAll(); a != "right" {
		t.Errorf("h1 text-align = %s", a)
	}
	if a := blocks[1].Base().Style.TextAlignAll(); a != "center" {
		t.Errorf("p text-align = %s", a)
	}
	footTD := onlyChild(t, onlyChild(t, groups[2]))
	if a := footTD.Base().Style.TextAlignAll(); a != "justify" {
		t.Errorf("tfoot td text-align = %s", a)
	}
}

func TestHintsOfHorizontalRules(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := parseWithHints(t, `
      <hr align=left>
      <hr align=right />
      <hr align=both color=red />
      <hr align=center noshade size=10 />
      <hr align=all size=8 width=100 />
    `)
	if len(children) != 5 {
		t.Fatalf("body has %d children, want 5", len(children))
	}
	auto := props.LengthKeyword("auto")
	style := func(i int) *computed.Style { return children[i].Base().Style }
	if style(0).MarginRight() != auto || style(1).MarginLeft() != auto {
		t.Errorf("hr 1 margin-right %v, hr 2 margin-left %v", style(0).MarginRight(), style(1).MarginLeft())
	}
	for _, i := range []int{2, 3} {
		if style(i).MarginLeft() != auto || style(i).MarginRight() != auto {
			t.Errorf("hr %d margins %v %v", i+1, style(i).MarginLeft(), style(i).MarginRight())
		}
	}
	if c := style(2).Color(); !isRGB(c, 1, 0, 0) {
		t.Errorf("hr 3 color = %v", c)
	}
	if w := style(3).BorderTopWidth(); w != 5 {
		t.Errorf("hr 4 border-top-width = %v", w)
	}
	if w := style(4).BorderTopWidth(); w != 1 {
		t.Errorf("hr 5 border-top-width = %v", w)
	}
}
