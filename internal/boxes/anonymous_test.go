package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestAnonymousBoxesInheritFromTheirParent(t *testing.T) {
	s := newStyled(t, `<style>div{color:red;border:3px solid;width:50px}</style><div></div>`)
	parent := s.block("div")

	box := boxes.AnonymousFrom(boxes.KindBlock, parent, nil)
	if _, ok := box.(*boxes.BlockBox); !ok {
		t.Fatalf("AnonymousFrom made a %T", box)
	}
	b := box.Base()
	if b.ElementTag != "div" || b.Element != parent.Element {
		t.Errorf("tag %q, element %v", b.ElementTag, b.Element)
	}
	if b.Style != parent.Style.Anonymous() || !b.Style.IsAnonymous() {
		t.Error("the style is not the anonymous style of the parent")
	}
	if got := b.Style.BorderTopWidth(); got != 0 {
		t.Errorf("anonymous border width = %v, want 0", got)
	}
}

func TestAnonymousFromMakesEveryKindWithChildren(t *testing.T) {
	parent := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	kinds := []boxes.Kind{
		boxes.KindBlock, boxes.KindLine, boxes.KindInline, boxes.KindInlineBlock,
		boxes.KindTable, boxes.KindInlineTable, boxes.KindTableRowGroup, boxes.KindTableRow,
		boxes.KindTableColumnGroup, boxes.KindTableColumn, boxes.KindTableCell, boxes.KindTableCaption,
		boxes.KindFlex, boxes.KindInlineFlex, boxes.KindGrid, boxes.KindInlineGrid,
	}
	child := boxes.AnonymousText(parent, "x")
	for _, k := range kinds {
		box := boxes.AnonymousFrom(k, parent, []boxes.Box{child})
		if box == nil || box.Kind() != k || len(box.ParentBase().Children) != 1 {
			t.Errorf("AnonymousFrom(%v) = %v", k, box)
		}
	}
	for _, k := range []boxes.Kind{boxes.KindText, boxes.KindBlockReplaced, boxes.KindPage, boxes.KindInvalid} {
		if box := boxes.AnonymousFrom(k, parent, nil); box != nil {
			t.Errorf("AnonymousFrom(%v) = %v, want nil", k, box)
		}
	}
}

func TestAnonymousLineBoxKeepsTextOverflowOfClippingContainers(t *testing.T) {
	s := newStyled(t, `<style>.a{text-overflow:ellipsis} .b{text-overflow:ellipsis;overflow:hidden}</style>
<p class=a></p><p class=b></p>`)
	ps := s.doc.Elements()
	var visible, hidden *boxes.BlockBox
	for _, el := range ps {
		switch el.Get("class") {
		case "a":
			visible = boxes.NewBlockBox("p", s.sf.Get(el, ""), el, nil)
		case "b":
			hidden = boxes.NewBlockBox("p", s.sf.Get(el, ""), el, nil)
		}
	}
	if got := boxes.AnonymousFrom(boxes.KindLine, visible, nil).(*boxes.LineBox).TextOverflow; got != "clip" {
		t.Errorf("visible overflow: text-overflow = %q, want clip", got)
	}
	if got := boxes.AnonymousFrom(boxes.KindLine, hidden, nil).(*boxes.LineBox).TextOverflow; got != "ellipsis" {
		t.Errorf("hidden overflow: text-overflow = %q, want ellipsis", got)
	}
}

func TestNewLineBoxDefaults(t *testing.T) {
	l := boxes.NewLineBox("div", plainStyle(), nil, nil)
	if l.TextOverflow != "clip" || l.BlockEllipsis.Keyword != "none" {
		t.Errorf("text-overflow %q, block-ellipsis %+v", l.TextOverflow, l.BlockEllipsis)
	}
}

func TestAnonymousTextBox(t *testing.T) {
	parent := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	tb := boxes.AnonymousText(parent, "abc")
	if tb.Text != "abc" || tb.ElementTag != "div" || tb.Kind() != boxes.KindText {
		t.Errorf("text box = %+v", tb.BoxFields)
	}
}

func TestTableCellSpansComeFromTheElement(t *testing.T) {
	s := newStyled(t, `<table><tr><td colspan=3 rowspan=0></td><td colspan=0 rowspan=-2></td><td></td></tr></table>`)
	tds := csstest.Elements(s.doc, "td")
	want := [][2]int{{3, 0}, {1, 0}, {1, 1}}
	for i, td := range tds {
		c := boxes.NewTableCellBox("td", s.sf.Get(td, ""), td, nil)
		if c.ColSpan != want[i][0] || c.RowSpan != want[i][1] {
			t.Errorf("cell %d: colspan %d, rowspan %d, want %v", i, c.ColSpan, c.RowSpan, want[i])
		}
	}
	c := boxes.NewTableCellBox("", plainStyle(), nil, nil)
	if c.ColSpan != 1 || c.RowSpan != 1 {
		t.Errorf("cell without element: %d, %d", c.ColSpan, c.RowSpan)
	}
	anon := boxes.AnonymousFrom(boxes.KindTableCell, boxes.NewTableRowBox("tr", plainStyle(), nil, nil), nil).(*boxes.TableCellBox)
	if anon.ColSpan != 1 || anon.RowSpan != 1 {
		t.Errorf("anonymous cell: %d, %d", anon.ColSpan, anon.RowSpan)
	}
}

func TestColumnSpans(t *testing.T) {
	s := newStyled(t, `<table><colgroup span=3></colgroup><colgroup><col span=2><col></colgroup></table>`)
	groups := csstest.Elements(s.doc, "colgroup")
	cols := csstest.Elements(s.doc, "col")
	plain := boxes.NewTableColumnGroupBox("colgroup", plainStyle(), groups[0], nil)
	if plain.Span() != 3 {
		t.Errorf("group span = %d, want 3", plain.Span())
	}
	c1 := boxes.NewTableColumnBox("col", plainStyle(), cols[0], nil)
	c2 := boxes.NewTableColumnBox("col", plainStyle(), cols[1], nil)
	if c1.Span() != 2 || c2.Span() != 1 {
		t.Errorf("column spans = %d, %d", c1.Span(), c2.Span())
	}
	withColumns := boxes.NewTableColumnGroupBox("colgroup", plainStyle(), groups[1], []boxes.Box{c1, c2})
	if withColumns.Span() != 2 {
		t.Errorf("group with columns span = %d, want the number of columns", withColumns.Span())
	}
}

func TestColumnCellsComeFromLayout(t *testing.T) {
	cell := boxes.NewTableCellBox("td", plainStyle(), nil, nil)
	col := boxes.NewTableColumnBox("col", plainStyle(), nil, nil)
	group := boxes.NewTableColumnGroupBox("colgroup", plainStyle(), nil, []boxes.Box{col})
	if len(col.Cells()) != 0 || len(group.Cells()) != 0 {
		t.Error("cells without layout")
	}
	col.GetCells = func() []*boxes.TableCellBox { return []*boxes.TableCellBox{cell} }
	if got := group.Cells(); len(got) != 1 || got[0] != cell {
		t.Errorf("group cells = %v", got)
	}
}
