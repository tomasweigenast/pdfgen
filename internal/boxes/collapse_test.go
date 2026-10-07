package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

// collapsedTable builds a table of one row group and one row with the given
// cells, styled by the document s.
func collapsedTable(s *styled, cells ...*html.Element) *boxes.TableBox {
	var children []boxes.Box
	for i, el := range cells {
		cell := boxes.NewTableCellBox("td", s.sf.Get(el, ""), el, nil)
		cell.GridX = i
		children = append(children, cell)
	}
	tr := s.el("tr")
	row := boxes.NewTableRowBox("tr", s.sf.Get(tr, ""), tr, children)
	tbody := s.el("tbody")
	group := boxes.NewTableRowGroupBox("tbody", s.sf.Get(tbody, ""), tbody, []boxes.Box{row})
	table := s.el("table")
	return boxes.NewTableBox("table", s.sf.Get(table, ""), table, []boxes.Box{group})
}

func TestCollapsedBordersKeepTheWidest(t *testing.T) {
	s := newStyled(t, `
      <table style="border-collapse: collapse; border: 2px solid">
        <tr><td style="border: 6px dotted"></td><td style="border-left: 4px double"></td></tr>
      </table>`)
	table := collapsedTable(s, csstest.Elements(s.doc, "td")...)
	grid := boxes.CollapseTableBorders(table, 2, 1)

	if got := len(grid.Vertical); got != 1 || len(grid.Vertical[0]) != 3 {
		t.Fatalf("vertical grid is %d×%d, want 1×3", got, len(grid.Vertical[0]))
	}
	if got := len(grid.Horizontal); got != 2 || len(grid.Horizontal[0]) != 2 {
		t.Fatalf("horizontal grid is %d×%d, want 2×2", got, len(grid.Horizontal[0]))
	}
	tests := []struct {
		name  string
		got   boxes.CollapsedBorder
		width float64
		style string
	}{
		{"table left", grid.Vertical[0][0], 6, "dotted"},
		{"between cells", grid.Vertical[0][1], 6, "dotted"},
		{"table right", grid.Vertical[0][2], 2, "solid"},
		{"first cell top", grid.Horizontal[0][0], 6, "dotted"},
		{"second cell top", grid.Horizontal[0][1], 2, "solid"},
	}
	for _, tt := range tests {
		if tt.got.Width != tt.width || string(tt.got.Style) != tt.style {
			t.Errorf("%s: %v %s, want %v %s", tt.name, tt.got.Width, tt.got.Style, tt.width, tt.style)
		}
	}

	if table.BorderLeftWidth != 3 || table.BorderRightWidth != 1 || table.BorderTopWidth != 3 {
		t.Errorf("table borders: left %v right %v top %v", table.BorderLeftWidth, table.BorderRightWidth, table.BorderTopWidth)
	}
	first := table.Children[0].(*boxes.TableRowGroupBox).Children[0].(*boxes.TableRowBox).Children[0].Base()
	if first.BorderRightWidth != 3 || first.BorderBottomWidth != 3 {
		t.Errorf("first cell borders: right %v bottom %v", first.BorderRightWidth, first.BorderBottomWidth)
	}
}

func TestCollapsedBordersOfEmptyTables(t *testing.T) {
	s := newStyled(t, `<table style="border-collapse: collapse"><tr><td></td></tr></table>`)
	table := collapsedTable(s)
	if grid := boxes.CollapseTableBorders(table, 0, 0); grid == nil || len(grid.Vertical) != 0 || len(grid.Horizontal) != 0 {
		t.Errorf("grid = %+v, want empty", grid)
	}
}

func TestCollapsedBordersOfRightToLeftTables(t *testing.T) {
	s := newStyled(t, `
      <table style="border-collapse: collapse; direction: rtl">
        <tr><td style="border-right: 6px solid"></td><td style="border-left: 4px solid"></td></tr>
      </table>`)
	table := collapsedTable(s, csstest.Elements(s.doc, "td")...)
	grid := boxes.CollapseTableBorders(table, 2, 1)
	widths := []float64{grid.Vertical[0][0].Width, grid.Vertical[0][1].Width, grid.Vertical[0][2].Width}
	if widths[0] != 4 || widths[1] != 0 || widths[2] != 6 {
		t.Errorf("vertical widths = %v, want [4 0 6]", widths)
	}
	if table.BorderRightWidth != 3 || table.BorderLeftWidth != 2 {
		t.Errorf("table borders: left %v right %v", table.BorderLeftWidth, table.BorderRightWidth)
	}
}
