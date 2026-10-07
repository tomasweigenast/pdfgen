package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestTranslateMovesDescendants(t *testing.T) {
	s := plainStyle()
	leaf := boxes.NewTextBox("p", s, nil, "a")
	leaf.PositionX, leaf.PositionY = 1, 2
	line := boxes.NewLineBox("div", s, nil, []boxes.Box{leaf})
	line.PositionX, line.PositionY = 3, 4
	box := boxes.NewBlockBox("div", s, nil, []boxes.Box{line})

	box.Translate(10, 20, false)
	if box.PositionX != 10 || line.PositionX != 13 || line.PositionY != 24 || leaf.PositionX != 11 || leaf.PositionY != 22 {
		t.Errorf("positions: box %v,%v line %v,%v leaf %v,%v",
			box.PositionX, box.PositionY, line.PositionX, line.PositionY, leaf.PositionX, leaf.PositionY)
	}
}

func TestTranslateByNothingDoesNothing(t *testing.T) {
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	box.PositionX = 7
	box.Translate(0, 0, false)
	if box.PositionX != 7 || box.PositionY != 0 {
		t.Errorf("position = %v, %v", box.PositionX, box.PositionY)
	}
}

func TestTranslateCanSkipFloats(t *testing.T) {
	s := newStyled(t, `<style>.f{float:left}</style><div><p class=f></p><p></p></div>`)
	ps := csstest.Elements(s.doc, "p")
	floated := boxes.NewBlockBox("p", s.sf.Get(ps[0], ""), ps[0], nil)
	flow := boxes.NewBlockBox("p", s.sf.Get(ps[1], ""), ps[1], nil)
	if !floated.IsFloated() || flow.IsFloated() {
		t.Fatalf("floated %v, in flow %v", floated.IsFloated(), flow.IsFloated())
	}
	parent := s.block("div", floated, flow)

	parent.Translate(5, 6, true)
	if floated.PositionX != 0 || flow.PositionX != 5 || parent.PositionX != 5 {
		t.Errorf("x positions: float %v, in flow %v, parent %v", floated.PositionX, flow.PositionX, parent.PositionX)
	}
	parent.Translate(1, 1, false)
	if floated.PositionX != 1 {
		t.Errorf("float x = %v after moving without ignoring floats", floated.PositionX)
	}
}

func TestTranslateMovesTableColumnsAndColumnGroups(t *testing.T) {
	s := plainStyle()
	col := boxes.NewTableColumnBox("col", s, nil, nil)
	group := boxes.NewTableColumnGroupBox("colgroup", s, nil, []boxes.Box{col})
	row := boxes.NewTableRowBox("tr", s, nil, nil)
	table := boxes.NewTableBox("table", s, nil, []boxes.Box{row})
	table.ColumnGroups = []*boxes.TableColumnGroupBox{group}
	table.ColumnPositions = []float64{0, 10, 25}

	table.Translate(3, 4, false)
	if got := table.ColumnPositions; got[0] != 3 || got[1] != 13 || got[2] != 28 {
		t.Errorf("column positions = %v", got)
	}
	if group.PositionY != 4 || row.PositionX != 3 {
		t.Errorf("group y = %v, row x = %v", group.PositionY, row.PositionX)
	}
	if col.PositionX != 3 {
		t.Errorf("column x = %v, want it moved with its group", col.PositionX)
	}
}

func TestAbsolutePlaceholderMovesOnlyItsBoxBeforeLayout(t *testing.T) {
	s := plainStyle()
	child := boxes.NewBlockBox("p", s, nil, nil)
	box := boxes.NewBlockBox("div", s, nil, []boxes.Box{child})
	p := boxes.NewAbsolutePlaceholder(box)

	p.Translate(2, 3, false)
	if box.PositionX != 2 || child.PositionX != 0 {
		t.Errorf("before layout: box %v, child %v", box.PositionX, child.PositionX)
	}

	laid := box.Copy().(*boxes.BlockBox)
	laid.Children = []boxes.Box{child}
	p.SetLaidOutBox(laid)
	p.Translate(1, 1, false)
	if laid.PositionX != 3 || child.PositionX != 1 {
		t.Errorf("after layout: box %v, child %v", laid.PositionX, child.PositionX)
	}
}

func names(seq func(func(boxes.Box) bool)) []string {
	var out []string
	for b := range seq {
		out = append(out, b.Base().ElementTag)
	}
	return out
}

func TestDescendantsAreDepthFirstAndIncludeTheBox(t *testing.T) {
	s := plainStyle()
	a := boxes.NewTextBox("a", s, nil, "x")
	c := boxes.NewTextBox("c", s, nil, "x")
	b := boxes.NewInlineBox("b", s, nil, []boxes.Box{c})
	root := boxes.NewBlockBox("root", s, nil, []boxes.Box{a, b})

	got := names(boxes.Descendants(root, false))
	want := []string{"root", "a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	n := 0
	for range boxes.Descendants(root, false) {
		if n++; n == 2 {
			break
		}
	}
	if n != 2 {
		t.Errorf("iteration did not stop at the break: %d", n)
	}
}

func TestDescendantsOfPlaceholders(t *testing.T) {
	s := plainStyle()
	inner := boxes.NewTextBox("inner", s, nil, "x")
	abs := boxes.NewBlockBox("abs", s, nil, []boxes.Box{inner})
	root := boxes.NewBlockBox("root", s, nil, []boxes.Box{boxes.NewAbsolutePlaceholder(abs)})

	if got := names(boxes.Descendants(root, false)); len(got) != 2 || got[1] != "abs" {
		t.Errorf("without placeholders: %v", got)
	}
	got := names(boxes.Descendants(root, true))
	if len(got) != 3 || got[1] != "abs" || got[2] != "inner" {
		t.Errorf("with placeholders: %v", got)
	}
}

func TestAllChildrenOfATableListColumnGroupsLast(t *testing.T) {
	s := plainStyle()
	row := boxes.NewTableRowBox("tr", s, nil, nil)
	group := boxes.NewTableColumnGroupBox("colgroup", s, nil, nil)
	table := boxes.NewTableBox("table", s, nil, []boxes.Box{row})
	table.ColumnGroups = []*boxes.TableColumnGroupBox{group}
	got := boxes.AllChildren(table)
	if len(got) != 2 || got[0] != boxes.Box(row) || got[1] != boxes.Box(group) {
		t.Errorf("AllChildren = %v", got)
	}
	if boxes.ChildrenOf(table)[0] != boxes.Box(row) || len(boxes.ChildrenOf(table)) != 1 {
		t.Error("ChildrenOf includes the column groups")
	}
	if boxes.ChildrenOf(boxes.NewTextBox("p", s, nil, "a")) != nil {
		t.Error("a text box has children")
	}
}

func TestGetWrappedTable(t *testing.T) {
	s := plainStyle()
	inline := boxes.NewInlineTableBox("table", s, nil, nil)
	wrapper := boxes.NewBlockBox("table", s, nil, []boxes.Box{boxes.NewTableCaptionBox("caption", s, nil, nil), inline})
	wrapper.IsTableWrapper = true
	got := boxes.GetWrappedTable(wrapper)
	if got == nil || got.Kind() != boxes.KindInlineTable {
		t.Errorf("GetWrappedTable = %v", got)
	}
	if boxes.GetWrappedTable(boxes.NewBlockBox("div", s, nil, nil)) != nil {
		t.Error("found a table in an empty box")
	}
}

func TestPageValuesComeFromTheFirstAndLastBoxesInFlow(t *testing.T) {
	s := newStyled(t, `<style>.a{page:first} .b{page:second} .c{page:third}</style>
<div><i class=a></i><i class=b></i><i class=c></i></div>`)
	is := csstest.Elements(s.doc, "i")
	mk := func(el *html.Element, children ...boxes.Box) *boxes.BlockBox {
		return boxes.NewBlockBox(el.LocalName(), s.sf.Get(el, ""), el, children)
	}
	div := s.el("div")
	a, b, c := mk(is[0]), mk(is[1]), mk(is[2])
	parent := mk(div, a, b, c)
	if start, end := parent.PageValues(); start != "first" || end != "third" {
		t.Errorf("page values = %q, %q", start, end)
	}
	if start, end := mk(div, b).PageValues(); start != "second" || end != "second" {
		t.Errorf("single child page values = %q, %q", start, end)
	}
	if start, end := mk(div).PageValues(); start != "" || end != "" {
		t.Errorf("empty box page values = %q, %q", start, end)
	}
	table := boxes.NewTableBox("table", s.style("div"), nil, []boxes.Box{a})
	if start, _ := table.PageValues(); start != "" {
		t.Errorf("a table took the page of its rows: %q", start)
	}
}
