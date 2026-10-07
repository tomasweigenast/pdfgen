package boxes_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

var (
	P = boxtest.Parent
	T = boxtest.Text
	R = boxtest.Replaced
)

func TestBoxTree(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.Parse(t, "<p>"), []boxtest.Node{P("p", "Block")})
	boxtest.AssertTree(t, boxtest.Parse(t, `
      <style>
        span { display: inline-block }
      </style>
      <p>Hello <em>World <img src="pattern.png"><span>L</span></em>!</p>`), []boxtest.Node{
		P("p", "Block",
			T("p", "Hello "),
			P("em", "Inline",
				T("em", "World "),
				R("img", "InlineReplaced"),
				P("span", "InlineBlock",
					T("span", "L"))),
			T("p", "!")),
	})
}

func TestHTMLEntities(t *testing.T) {
	testutil.AssertNoLogs(t)
	for _, quote := range []string{`"`, "&quot;", "&#x22;", "&#34;"} {
		boxtest.AssertTree(t, boxtest.Parse(t, "<p>"+quote+"abc"+quote), []boxtest.Node{
			P("p", "Block", T("p", `"abc"`)),
		})
	}
}

func TestInlineInBlockWrapsInlinesBeforeABlock(t *testing.T) {
	testutil.AssertNoLogs(t)
	box := boxtest.Parse(t, "<div>Hello, <em>World</em>!\n<p>Lipsum.</p></div>")
	box = boxes.InlineInBlock(box)
	boxtest.AssertTree(t, box, []boxtest.Node{
		P("div", "Block",
			P("div", "Block",
				P("div", "Line",
					T("div", "Hello, "),
					P("em", "Inline", T("em", "World")),
					T("div", "!\n"))),
			P("p", "Block",
				P("p", "Line", T("p", "Lipsum.")))),
	})
}

func TestInlineInBlockWrapsInlinesAfterABlock(t *testing.T) {
	testutil.AssertNoLogs(t)
	box := boxtest.Parse(t, "<div><p>Lipsum.</p>Hello, <em>World</em>!\n</div>")
	box = boxes.InlineInBlock(box)
	boxtest.AssertTree(t, box, []boxtest.Node{
		P("div", "Block",
			P("p", "Block",
				P("p", "Line", T("p", "Lipsum."))),
			P("div", "Block",
				P("div", "Line",
					T("div", "Hello, "),
					P("em", "Inline", T("em", "World")),
					T("div", "!\n")))),
	})
}

func TestInlineInBlockKeepsAbsolutesInLines(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Absolutely positioned boxes stay in the lines to get their static
	// position later.
	box := boxtest.Parse(t, `<p>Hello <em style="position:absolute;
                                    display: block">World</em>!</p>`)
	expected := []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				T("p", "Hello "),
				P("em", "Block",
					P("em", "Line", T("em", "World"))),
				T("p", "!"))),
	}
	box = boxes.InlineInBlock(box)
	boxtest.AssertTree(t, box, expected)
	box = boxes.BlockInInline(box)
	boxtest.AssertTree(t, box, expected)
}

func TestInlineInBlockKeepsFloatsInLines(t *testing.T) {
	testutil.AssertNoLogs(t)
	box := boxtest.Parse(t, `<p>Hello <em style="float: left">World</em>!</p>`)
	box = boxes.InlineInBlock(box)
	box = boxes.BlockInInline(box)
	boxtest.AssertTree(t, box, []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				T("p", "Hello "),
				P("em", "Block",
					P("em", "Line", T("em", "World"))),
				T("p", "!"))),
	})
}

func TestBlockInInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	box := boxtest.Parse(t, `
      <style>
        p { display: inline-block; }
        span, i { display: block; }
      </style>
      <p>Lorem <em>ipsum <strong>dolor <span>sit</span>
      <span>amet,</span></strong><span><em>conse<i>`)
	box = boxes.InlineInBlock(box)
	boxtest.AssertTree(t, box, []boxtest.Node{
		P("body", "Line",
			P("p", "InlineBlock",
				P("p", "Line",
					T("p", "Lorem "),
					P("em", "Inline",
						T("em", "ipsum "),
						P("strong", "Inline",
							T("strong", "dolor "),
							P("span", "Block",
								P("span", "Line", T("span", "sit"))),
							T("strong", "\n      "),
							P("span", "Block",
								P("span", "Line", T("span", "amet,")))),
						P("span", "Block",
							P("span", "Line",
								P("em", "Inline",
									T("em", "conse"),
									P("i", "Block")))))))),
	})

	box = boxes.BlockInInline(box)
	boxtest.AssertTree(t, box, []boxtest.Node{
		P("body", "Line",
			P("p", "InlineBlock",
				P("p", "Block",
					P("p", "Line",
						T("p", "Lorem "),
						P("em", "Inline",
							T("em", "ipsum "),
							P("strong", "Inline", T("strong", "dolor "))))),
				P("span", "Block",
					P("span", "Line", T("span", "sit"))),
				P("p", "Block",
					P("p", "Line",
						P("em", "Inline",
							P("strong", "Inline", T("strong", "\n      "))))),
				P("span", "Block",
					P("span", "Line", T("span", "amet,"))),
				P("p", "Block",
					P("p", "Line",
						P("em", "Inline",
							P("strong", "Inline")))),
				P("span", "Block",
					P("span", "Block",
						P("span", "Line",
							P("em", "Inline", T("em", "conse")))),
					P("i", "Block"),
					P("span", "Block",
						P("span", "Line",
							P("em", "Inline")))),
				P("p", "Block",
					P("p", "Line",
						P("em", "Inline"))))),
	})
}

func TestAnonymousBoxesInheritStyles(t *testing.T) {
	testutil.AssertNoLogs(t)
	box := boxtest.Parse(t, `
      <style>
        span { display: block; }
        * { margin: 42px }
        html { color: blue }
      </style>
      <p>Lorem <em>ipsum <strong>dolor <span>sit</span>
        <span>amet,</span></strong><span>consectetur</span></em></p>`)
	box = boxes.InlineInBlock(box)
	box = boxes.BlockInInline(box)

	var descendants []boxes.Box
	for d := range boxes.Descendants(box, false) {
		descendants = append(descendants, d)
	}
	if len(descendants) != 31 {
		t.Fatalf("%d descendants, want 31", len(descendants))
	}
	if descendants[0] != box {
		t.Error("the box is not its first descendant")
	}
	for _, d := range descendants {
		style := d.Base().Style
		// All boxes inherit the color.
		if c := style.Color(); c.Coordinates[0] != 0 || c.Coordinates[1] != 0 || c.Coordinates[2] != 1 || c.Alpha != 1 {
			t.Errorf("%v: color %v, want blue", d, c)
		}
		// Only boxes that are not anonymous have margins.
		if m := style.MarginTop(); m != props.Px(0) && m != props.Px(42) {
			t.Errorf("%v: margin-top %#v", d, m)
		}
	}
}

func TestWhitespace(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <p>Lorem `+"\t\r\n"+`  ipsum`+"\t"+`<strong>  dolor
        <img src=pattern.png> sit
        <span style="position: absolute"></span> <em> amet </em>
        consectetur</strong>.</p>
      <pre>`+"\t"+`  foo`+"\n"+`</pre>
      <pre style="white-space: pre-wrap">`+"\t"+`  foo`+"\n"+`</pre>
      <pre style="white-space: pre-line">`+"\t"+`  foo`+"\n"+`</pre>
    `), []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				T("p", "Lorem ipsum "),
				P("strong", "Inline",
					T("strong", "dolor "),
					R("img", "InlineReplaced"),
					T("strong", " sit "),
					P("span", "Block"),
					P("em", "Inline", T("em", "amet ")),
					T("strong", "consectetur")),
				T("p", "."))),
		P("pre", "Block",
			P("pre", "Line", T("pre", "\t  foo\n"))),
		P("pre", "Block",
			P("pre", "Line", T("pre", "\t  foo\n"))),
		P("pre", "Block",
			P("pre", "Line", T("pre", "foo\n"))),
	})
}

func TestPageStyle(t *testing.T) {
	tests := []struct {
		page                     cascade.PageType
		top, right, bottom, left float64
	}{
		{cascade.PageType{Side: "left"}, 20, 3, 3, 10},
		{cascade.PageType{Side: "right"}, 20, 10, 3, 3},
		{cascade.PageType{Side: "left", Index: 1}, 10, 3, 3, 10},
		{cascade.PageType{Side: "right", Index: 1}, 10, 10, 3, 3},
		{cascade.PageType{Side: "right", Name: "name", Index: 1, Groups: []cascade.PageGroup{{Name: "name", Index: 0}}}, 5, 10, 3, 15},
		{cascade.PageType{Side: "right", Name: "name", Index: 1, Groups: []cascade.PageGroup{{Name: "name", Index: 1}}}, 5, 10, 4, 15},
		{cascade.PageType{Side: "right", Name: "name", Index: 2, Groups: []cascade.PageGroup{{Name: "name", Index: 0}}}, 5, 10, 1, 15},
		{cascade.PageType{Side: "right", Name: "name", Index: 8, Groups: []cascade.PageGroup{{Name: "name", Index: 8}}}, 5, 10, 2, 15},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%+v", tt.page), func(t *testing.T) {
			testutil.AssertNoLogs(t)
			doc := csstest.Parse(t, `
              <style>
                @page { margin: 3px }
                @page name { margin-left: 15px; margin-top: 5px }
                @page :nth(3) { margin-bottom: 1px }
                @page :nth(5n+4) { margin-bottom: 2px }
                @page :nth(2 of name) { margin-bottom: 4px }
                @page :first { margin-top: 20px }
                @page :right { margin-right: 10px; margin-top: 10px }
                @page :left { margin-left: 10px; margin-top: 10px }
              </style>`)
			sf := csstest.StyleFor(t, doc, cascade.Options{})
			// The styles of a page type are only computed when a page
			// is made.
			sf.SetPageTypeComputedStyles(tt.page)
			style := sf.Page(tt.page, "")
			got := []props.Length{style.MarginTop(), style.MarginRight(), style.MarginBottom(), style.MarginLeft()}
			want := []float64{tt.top, tt.right, tt.bottom, tt.left}
			for i := range got {
				if got[i] != props.Px(want[i]) {
					t.Errorf("margins = %v, want %v", got, want)
					break
				}
			}
		})
	}
}

func TestImagesFallBackToTheirAlternativeText(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	result := boxtest.ParseAll(t, `
          <p><img src=pattern.png
            /><img alt="No src"
            /><img src=inexistent.jpg alt="Inexistent src" /></p>
        `)
	lines := logs.Lines()
	if len(lines) != 1 || !strings.Contains(lines[0], "ERROR: Failed to load image") || !strings.Contains(lines[0], "inexistent.jpg") {
		t.Errorf("logs = %q", lines)
	}
	boxtest.AssertTree(t, result, []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				R("img", "InlineReplaced"),
				P("img", "Inline", T("img", "No src")),
				P("img", "Inline", T("img", "Inexistent src")))),
	})
}

func TestImagesWithoutBaseURL(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	result := boxtest.ParseAllWith(t, `<p><img src=pattern.png alt="No base_url">`, boxtest.Options{NoBaseURL: true})
	lines := logs.Lines()
	if len(lines) != 1 || !strings.Contains(lines[0], "ERROR: Relative URI reference without a base URI") {
		t.Errorf("logs = %q", lines)
	}
	boxtest.AssertTree(t, result, []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				P("img", "Inline", T("img", "No base_url")))),
	})
}

func TestTablesOrderRowGroupsAndRemoveWhiteSpace(t *testing.T) {
	testutil.AssertNoLogs(t)
	// CSS 2.1 §17.2.1 rule 1.3, and the order of the table model.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        x-table { display: table }
        x-tr { display: table-row }
        x-td, x-th { display: table-cell }
        x-thead { display: table-header-group }
        x-tfoot { display: table-footer-group }
        x-col { display: table-column }
        x-caption { display: table-caption }
      </style>
      <x-table>
        <x-tr>
          <x-th>foo</x-th>
          <x-th>bar</x-th>
        </x-tr>
        <x-tfoot></x-tfoot>
        <x-thead><x-th></x-th></x-thead>
        <x-caption style="caption-side: bottom"></x-caption>
        <x-thead></x-thead>
        <x-col></x-col>
        <x-caption>top caption</x-caption>
        <x-tr>
          <x-td>baz</x-td>
        </x-tr>
      </x-table>
    `), []boxtest.Node{
		P("x-table", "Block",
			P("x-caption", "TableCaption",
				P("x-caption", "Line", T("x-caption", "top caption"))),
			P("x-table", "Table",
				P("x-table", "TableColumnGroup",
					P("x-col", "TableColumn")),
				P("x-thead", "TableRowGroup",
					P("x-thead", "TableRow",
						P("x-th", "TableCell"))),
				P("x-table", "TableRowGroup",
					P("x-tr", "TableRow",
						P("x-th", "TableCell",
							P("x-th", "Line", T("x-th", "foo"))),
						P("x-th", "TableCell",
							P("x-th", "Line", T("x-th", "bar"))))),
				P("x-thead", "TableRowGroup"),
				P("x-table", "TableRowGroup",
					P("x-tr", "TableRow",
						P("x-td", "TableCell",
							P("x-td", "Line", T("x-td", "baz"))))),
				P("x-tfoot", "TableRowGroup")),
			P("x-caption", "TableCaption")),
	})
}

func TestTablesWrapLooseCells(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rules 1.4 and 3.1.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <span style="display: table-cell">foo</span>
      <span style="display: table-cell">bar</span>
    `), []boxtest.Node{
		P("body", "Block",
			P("body", "Table",
				P("body", "TableRowGroup",
					P("body", "TableRow",
						P("span", "TableCell",
							P("span", "Line", T("span", "foo"))),
						P("span", "TableCell",
							P("span", "Line", T("span", "bar"))))))),
	})
}

func TestTablesKeepOnlyColumnsInColumnGroups(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rules 1.1 and 1.2, and column groups have at least one column.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <span style="display: table-column-group">
        1
        <em style="display: table-column">
          2
          <strong>3</strong>
        </em>
        <strong>4</strong>
      </span>
      <ins style="display: table-column-group"></ins>
    `), []boxtest.Node{
		P("body", "Block",
			P("body", "Table",
				P("span", "TableColumnGroup",
					P("em", "TableColumn")),
				P("ins", "TableColumnGroup",
					P("ins", "TableColumn")))),
	})
}

func TestTablesWrapContentInRowsThenCells(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rules 2.1 then 2.3.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `<x-table style="display:table">foo <div></div></x-table>`), []boxtest.Node{
		P("x-table", "Block",
			P("x-table", "Table",
				P("x-table", "TableRowGroup",
					P("x-table", "TableRow",
						P("x-table", "TableCell",
							P("x-table", "Block",
								P("x-table", "Line", T("x-table", "foo "))),
							P("div", "Block")))))),
	})
}

func TestTablesWrapRowGroupContentInRows(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rule 2.2.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `<x-thead style="display: table-header-group"><div></div>`+
		`<x-td style="display: table-cell"></x-td></x-thead>`), []boxtest.Node{
		P("body", "Block",
			P("body", "Table",
				P("x-thead", "TableRowGroup",
					P("x-thead", "TableRow",
						P("x-thead", "TableCell",
							P("div", "Block")),
						P("x-td", "TableCell"))))),
	})
}

func TestTablesInInlineBoxesAreInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rule 3.2.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `<span><x-tr style="display: table-row"></x-tr></span>`), []boxtest.Node{
		P("body", "Line",
			P("span", "Inline",
				P("span", "InlineBlock",
					P("span", "InlineTable",
						P("span", "TableRowGroup",
							P("x-tr", "TableRow")))))),
	})
}

func TestTablesKeepWhiteSpaceAroundAnonymousTables(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rule 3.1; rule 1.3 does not apply, the white space before and after
	// is kept, then collapsed into a single space.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <span>
        <em style="display: table-cell"></em>
        <em style="display: table-cell"></em>
      </span>
    `), []boxtest.Node{
		P("body", "Line",
			P("span", "Inline",
				T("span", " "),
				P("span", "InlineBlock",
					P("span", "InlineTable",
						P("span", "TableRowGroup",
							P("span", "TableRow",
								P("em", "TableCell"),
								P("em", "TableCell"))))),
				T("span", " "))),
	})
}

func TestTablesWrapLooseRows(t *testing.T) {
	testutil.AssertNoLogs(t)
	// Rule 3.2.
	boxtest.AssertTree(t, boxtest.ParseAll(t, `<x-tr style="display: table-row"></x-tr>`+"\t"+
		`<x-tr style="display: table-row"></x-tr>`), []boxtest.Node{
		P("body", "Block",
			P("body", "Table",
				P("body", "TableRowGroup",
					P("x-tr", "TableRow"),
					P("x-tr", "TableRow")))),
	})
}

func TestTablesWrapLooseColumns(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `<x-col style="display: table-column"></x-col>`+"\n"+
		`<x-colgroup style="display: table-column-group"></x-colgroup>`), []boxtest.Node{
		P("body", "Block",
			P("body", "Table",
				P("body", "TableColumnGroup",
					P("x-col", "TableColumn")),
				P("x-colgroup", "TableColumnGroup",
					P("x-colgroup", "TableColumn")))),
	})
}

// wrappedTable returns the table of a document whose body only has a
// table.
func wrappedTable(t *testing.T, root boxes.Box) (wrapper boxes.Box, table *boxes.TableBox) {
	t.Helper()
	body := onlyChild(t, root)
	wrapper = onlyChild(t, body)
	table, ok := onlyChild(t, wrapper).(*boxes.TableBox)
	if !ok {
		t.Fatalf("wrapper child is %v, want a table", boxes.ChildrenOf(wrapper)[0])
	}
	return wrapper, table
}

func onlyChild(t *testing.T, box boxes.Box) boxes.Box {
	t.Helper()
	children := boxes.ChildrenOf(box)
	if len(children) != 1 {
		t.Fatalf("%v has %d children, want 1", box, len(children))
	}
	return children[0]
}

func TestTableStyleIsSplitWithTheWrapper(t *testing.T) {
	testutil.AssertNoLogs(t)
	wrapper, table := wrappedTable(t, boxtest.ParseAll(t, `<table style="margin: 1px; padding: 2px"></table>`))
	if wrapper.Kind() != boxes.KindBlock {
		t.Errorf("wrapper is %v", wrapper)
	}
	if got := wrapper.Base().Style.MarginTop(); got != props.Px(1) {
		t.Errorf("wrapper margin-top = %#v", got)
	}
	if got := wrapper.Base().Style.PaddingTop(); got != props.Px(0) {
		t.Errorf("wrapper padding-top = %#v", got)
	}
	if got := table.Style.MarginTop(); got != props.Px(0) {
		t.Errorf("table margin-top = %#v", got)
	}
	if got := table.Style.PaddingTop(); got != props.Px(2) {
		t.Errorf("table padding-top = %#v", got)
	}
}

func TestColumnsWithSpanAreRepeated(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, table := wrappedTable(t, boxtest.ParseAll(t, `
      <table>
        <col span=3 style="width: 10px"></col>
        <col span=2></col>
      </table>
    `))
	if len(table.ColumnGroups) != 1 {
		t.Fatalf("%d column groups, want 1", len(table.ColumnGroups))
	}
	columns := table.ColumnGroups[0].Children
	var widths []props.Length
	var gridX []int
	for _, c := range columns {
		widths = append(widths, c.Base().Style.Width())
		gridX = append(gridX, c.(*boxes.TableColumnBox).GridX)
	}
	ten, auto := props.Px(10), props.LengthKeyword("auto")
	want := []props.Length{ten, ten, ten, auto, auto}
	if fmt.Sprint(widths) != fmt.Sprint(want) {
		t.Errorf("widths = %v, want %v", widths, want)
	}
	if fmt.Sprint(gridX) != "[0 1 2 3 4]" {
		t.Errorf("grid x = %v", gridX)
	}
	// The repeated columns are copies.
	if columns[0] == columns[1] {
		t.Error("repeated columns are the same box")
	}
}

func TestNestedGridX(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, table := wrappedTable(t, boxtest.ParseAll(t, `
      <table>
        <col span=2></col>
        <colgroup span=2></colgroup>
        <colgroup>
          <col></col>
          <col span=2></col>
        </colgroup>
        <col></col>
      </table>
    `))
	var got []string
	for _, g := range table.ColumnGroups {
		var cols []int
		for _, c := range g.Children {
			cols = append(cols, c.(*boxes.TableColumnBox).GridX)
		}
		got = append(got, fmt.Sprint(g.GridX, cols))
	}
	if want := "[0 [0 1] 2 [2 3] 4 [4 5 6] 7 [7]]"; fmt.Sprint(got) != want {
		t.Errorf("grid = %v, want %v", got, want)
	}
}

// cellGrid returns the grid x, colspan and rowspan of the cells of each row
// of the only row group of a table.
func cellGrid(t *testing.T, table *boxes.TableBox) (gridX, colspan, rowspan string) {
	t.Helper()
	group := onlyChild(t, table)
	var xs, cs, rs [][]int
	for _, row := range boxes.ChildrenOf(group) {
		var x, c, r []int
		for _, child := range boxes.ChildrenOf(row) {
			cell := child.(*boxes.TableCellBox)
			x, c, r = append(x, cell.GridX), append(c, cell.ColSpan), append(r, cell.RowSpan)
		}
		xs, cs, rs = append(xs, x), append(cs, c), append(rs, r)
	}
	return fmt.Sprint(xs), fmt.Sprint(cs), fmt.Sprint(rs)
}

func TestColspanAndRowspanPlaceCells(t *testing.T) {
	testutil.AssertNoLogs(t)
	// +---+---+---+
	// | A | B | C | X
	// +---+---+---+
	// | D |     E | X
	// +---+---+   +---+
	// |  F ...|   |   |   <-- overlap
	// +---+---+---+   +
	// | H | X   X | G |
	// +---+---+   +   +
	// | I | J | X |   |
	// +---+---+   +---+
	//
	// X: empty cells
	_, table := wrappedTable(t, boxtest.ParseAll(t, `
      <table>
        <tr>
          <td>A <td>B <td>C
        </tr>
        <tr>
          <td>D <td colspan=2 rowspan=2>E
        </tr>
        <tr>
          <td colspan=2>F <td rowspan=0>G
        </tr>
        <tr>
          <td>H
        </tr>
        <tr>
          <td>I <td>J
        </tr>
      </table>
    `))
	gridX, colspan, rowspan := cellGrid(t, table)
	if want := "[[0 1 2] [0 1] [0 3] [0] [0 1]]"; gridX != want {
		t.Errorf("grid x = %s, want %s", gridX, want)
	}
	if want := "[[1 1 1] [1 2] [2 1] [1] [1 1]]"; colspan != want {
		t.Errorf("colspan = %s, want %s", colspan, want)
	}
	if want := "[[1 1 1] [1 2] [1 3] [1] [1 1]]"; rowspan != want {
		t.Errorf("rowspan = %s, want %s", rowspan, want)
	}
}

func TestRowspanStopsAtTheLastRow(t *testing.T) {
	testutil.AssertNoLogs(t)
	_, table := wrappedTable(t, boxtest.ParseAll(t, `
        <table>
            <tr>
                <td rowspan=5></td>
                <td></td>
            </tr>
            <tr>
                <td></td>
            </tr>
        </table>
    `))
	gridX, colspan, rowspan := cellGrid(t, table)
	if want := "[[0 1] [1]]"; gridX != want {
		t.Errorf("grid x = %s, want %s", gridX, want)
	}
	if want := "[[1 1] [1]]"; colspan != want {
		t.Errorf("colspan = %s, want %s", colspan, want)
	}
	if want := "[[2 1] [1]]"; rowspan != want {
		t.Errorf("rowspan = %s, want %s", rowspan, want)
	}
}

func TestPseudoElementsWithoutContentMakeNoBox(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p:before { content: normal }
        div:before { content: none }
        section::before { color: black }
      </style>
      <p></p>
      <div></div>
      <section></section>
    `), []boxtest.Node{
		P("p", "Block"),
		P("div", "Block"),
		P("section", "Block"),
	})
}

func TestBeforeAndAfterContent(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p:before { content: 'a' 'b' }
        p::after { content: 'd' 'e' }
      </style>
      <p> c </p>
    `), []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				P("p::before", "Inline", T("p::before", "ab")),
				T("p", " c "),
				P("p::after", "Inline", T("p::after", "de")))),
	})
}

func TestBeforeWithAttr(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        a[href]:before { content: '[' attr(href) '] ' }
      </style>
      <p><a href="some url">some text</a></p>
    `), []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				P("a", "Inline",
					P("a::before", "Inline", T("a::before", "[some url] ")),
					T("a", "some text")))),
	})
}

// quotesTree is the tree of a paragraph with nested quotes whose ::before
// and ::after have the given texts.
func quotesTree(before1, before2, after2, after1 string) []boxtest.Node {
	return []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				P("q", "Inline",
					P("q::before", "Inline", T("q::before", before1)),
					T("q", "Lorem ipsum "),
					P("q", "Inline",
						P("q::before", "Inline", T("q::before", before2)),
						T("q", "dolor"),
						P("q::after", "Inline", T("q::after", after2))),
					T("q", " sit amet"),
					P("q::after", "Inline", T("q::after", after1))))),
	}
}

// nbsp is a no-break space, which white space processing keeps.
const nbsp = "\u00a0"

func TestQuotes(t *testing.T) {
	tests := []struct {
		name, quotes, lang string
		want               []boxtest.Node
	}{
		{"strings", `body { quotes: '«' '»' '“' '”' }`, "", quotesTree("«"+nbsp, "“"+nbsp, nbsp+"”", nbsp+"»")},
		{"auto", `body { quotes: auto }`, "", quotesTree("“"+nbsp, "‘"+nbsp, nbsp+"’", nbsp+"”")},
		{"none", `body { quotes: none }`, "", quotesTree(nbsp, nbsp, nbsp, nbsp)},
		{"language", "", "fr", quotesTree("«"+nbsp, "«"+nbsp, nbsp+"»", nbsp+"»")},
		{"language variant", "", "fr_CH", quotesTree("«"+nbsp, "‹"+nbsp, nbsp+"›", nbsp+"»")},
		{"parent language", "", "fr_CH_alt", quotesTree("«"+nbsp, "‹"+nbsp, nbsp+"›", nbsp+"»")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			lang := ""
			if tt.lang != "" {
				lang = ` lang="` + tt.lang + `"`
			}
			boxtest.AssertTree(t, boxtest.ParseAll(t, `
              <style>
                `+tt.quotes+`
                q:before { content: open-quote '`+nbsp+`'}
                q:after { content: '`+nbsp+`' close-quote }
              </style>
              <p`+lang+`><q>Lorem ipsum <q>dolor</q> sit amet</q></p>
            `), tt.want)
		})
	}
}

func TestBeforeWithImageAndInvalidDeclaration(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
          <style>
            p:before {
              content: 'a' url(pattern.png) 'b';
              /* Invalid, ignored in favor of the one above. */
              content: some-function(nested-function(something));
            }
          </style>
          <p>c</p>
        `), []boxtest.Node{
		P("p", "Block",
			P("p", "Line",
				P("p::before", "Inline",
					T("p::before", "a"),
					R("p::before", "InlineReplaced"),
					T("p::before", "b")),
				T("p", "c"))),
	})
	lines := logs.Lines()
	if len(lines) != 1 || !strings.Contains(lines[0], "nested-function(") || !strings.Contains(lines[0], "invalid value") {
		t.Errorf("logs = %q", lines)
	}
}

func TestDisplayNoneRoot(t *testing.T) {
	for _, source := range []string{
		`<html style="display: none">`,
		`<html style="display: none">abc`,
		`<html style="display: none"><p>abc`,
		`<body style="display: none"><p>abc`,
	} {
		t.Run(source, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			box := boxtest.ParseAll(t, source)
			if d := box.Base().Style.Display(); fmt.Sprint(d) != "[block flow]" {
				t.Errorf("display = %v, want [block flow]", d)
			}
			if children := boxes.ChildrenOf(box); len(children) != 0 {
				t.Errorf("root has children %v", children)
			}
		})
	}
}
