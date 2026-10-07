package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestCopyIsShallow(t *testing.T) {
	s := plainStyle()
	child := boxes.NewTextBox("p", s, nil, "a")
	box := boxes.NewBlockBox("div", s, nil, []boxes.Box{child})
	box.PositionX = 5

	c := box.Copy().(*boxes.BlockBox)
	if c == box {
		t.Fatal("Copy returned the same box")
	}
	if c.Style != box.Style || c.Children[0] != boxes.Box(child) {
		t.Error("the copy does not share the style and the children")
	}
	c.PositionX = 9
	c.Width = units.F(3)
	if box.PositionX != 5 || box.Width != (units.MaybeFloat{}) {
		t.Errorf("changing the copy changed the original: %+v", box.BoxFields)
	}
}

func TestCopyWithChildrenReplacesChildren(t *testing.T) {
	s := plainStyle()
	a := boxes.NewTextBox("p", s, nil, "a")
	b := boxes.NewTextBox("p", s, nil, "b")
	box := boxes.NewBlockBox("div", s, nil, []boxes.Box{a, b})
	c := box.CopyWithChildren([]boxes.Box{b}).(*boxes.BlockBox)
	if len(c.Children) != 1 || c.Children[0] != boxes.Box(b) || len(box.Children) != 2 {
		t.Errorf("children: copy %v, original %v", c.Children, box.Children)
	}
}

func TestCopyWithChildrenKeepsRemovedDecorationAndResetsTheOriginal(t *testing.T) {
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	box.RemoveDecoration(true, false)
	c := box.CopyWithChildren(nil).(*boxes.BlockBox)
	if !c.RemoveDecorationSides.Has(boxes.SideTop) {
		t.Error("the copy lost the removed decoration")
	}
	if box.RemoveDecorationSides != 0 {
		t.Error("the original kept the removed decoration")
	}
}

func TestDeepCopyCopiesDescendants(t *testing.T) {
	s := plainStyle()
	leaf := boxes.NewTextBox("p", s, nil, "a")
	inline := boxes.NewInlineBox("em", s, nil, []boxes.Box{leaf})
	line := boxes.NewLineBox("div", s, nil, []boxes.Box{inline})
	box := boxes.NewBlockBox("div", s, nil, []boxes.Box{line})

	c := box.DeepCopy().(*boxes.BlockBox)
	cLeaf := c.Children[0].(*boxes.LineBox).Children[0].(*boxes.InlineBox).Children[0].(*boxes.TextBox)
	if cLeaf == leaf {
		t.Fatal("descendants are shared")
	}
	cLeaf.Text = "changed"
	if leaf.Text != "a" {
		t.Error("changing a descendant of the copy changed the original")
	}
}

func TestDeepCopyOfATableSharesColumnGroups(t *testing.T) {
	s := plainStyle()
	group := boxes.NewTableColumnGroupBox("colgroup", s, nil, nil)
	table := boxes.NewTableBox("table", s, nil, []boxes.Box{boxes.NewTableRowGroupBox("tbody", s, nil, nil)})
	table.ColumnGroups = []*boxes.TableColumnGroupBox{group}

	c := table.DeepCopy().(*boxes.TableBox)
	if c.Children[0] == table.Children[0] {
		t.Error("row groups are shared")
	}
	if c.ColumnGroups[0] != group {
		t.Error("column groups were copied")
	}
}

func TestGridAdvancementsAreSharedByCopies(t *testing.T) {
	grid := boxes.NewGridBox("div", plainStyle(), nil, nil)
	c := grid.Copy().(*boxes.GridBox)
	c.Advancements[[2]int{1, 2}] = 4
	if grid.Advancements[[2]int{1, 2}] != 4 {
		t.Error("copies do not share the advancements")
	}
}

func TestCopyWithTextChangesOnlyTheText(t *testing.T) {
	b := boxes.NewTextBox("p", plainStyle(), nil, "abc")
	b.JustificationSpacing = 2
	c := b.CopyWithText("ab")
	if c.Text != "ab" || b.Text != "abc" || c.JustificationSpacing != 2 {
		t.Errorf("text %q, original %q, spacing %v", c.Text, b.Text, c.JustificationSpacing)
	}
}

func TestCopiesKeepTheirType(t *testing.T) {
	s := plainStyle()
	for _, b := range []boxes.Box{
		boxes.NewInlineTableBox("", s, nil, nil),
		boxes.NewTableCaptionBox("", s, nil, nil),
		boxes.NewFootnoteAreaBox(nil, s),
		boxes.NewMarginBox("@top", s),
		boxes.NewPageBox(pageType(), s),
		boxes.NewInlineReplacedBox("", s, nil, nil),
	} {
		for name, c := range map[string]boxes.Box{"Copy": b.Copy(), "DeepCopy": b.DeepCopy()} {
			if c.Kind() != b.Kind() {
				t.Errorf("%s of a %v box is a %v box", name, b.Kind(), c.Kind())
			}
		}
	}
}

func TestPlaceholderCopiesTheBoxItHolds(t *testing.T) {
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	p := boxes.NewAbsolutePlaceholder(box)
	c := p.Copy().(*boxes.AbsolutePlaceholder)
	if c == p || c.Box == boxes.Box(box) {
		t.Error("the placeholder or its box is shared")
	}
	if c.Kind() != boxes.KindBlock {
		t.Errorf("Kind = %v", c.Kind())
	}
}
