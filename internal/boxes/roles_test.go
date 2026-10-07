package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
)

type roleSet struct {
	blockLevel, inlineLevel, parent, blockContainer, atomic bool
	replaced, flex, grid, table                             bool
	tableChild, internalTable, tabular                      bool
}

func rolesOf(b boxes.Box) roleSet {
	var r roleSet
	_, r.blockLevel = b.(boxes.BlockLevel)
	_, r.inlineLevel = b.(boxes.InlineLevel)
	_, r.parent = b.(boxes.Parent)
	_, r.blockContainer = b.(boxes.BlockContainer)
	_, r.atomic = b.(boxes.AtomicInline)
	_, r.replaced = b.(boxes.Replaced)
	_, r.flex = b.(boxes.FlexContainer)
	_, r.grid = b.(boxes.GridContainer)
	_, r.table = b.(boxes.TableLike)
	_, r.tableChild = b.(boxes.TableChild)
	_, r.internalTable = b.(boxes.InternalTableOrCaption)
	_, r.tabular = b.(boxes.TabularContainer)
	return r
}

func TestBoxTypesHaveTheirRoles(t *testing.T) {
	style := plainStyle()
	tests := []struct {
		box  boxes.Box
		kind boxes.Kind
		name string
		want roleSet
	}{
		{boxes.NewBlockBox("", style, nil, nil), boxes.KindBlock, "Block",
			roleSet{blockLevel: true, parent: true, blockContainer: true}},
		{boxes.NewLineBox("", style, nil, nil), boxes.KindLine, "Line",
			roleSet{parent: true}},
		{boxes.NewInlineBox("", style, nil, nil), boxes.KindInline, "Inline",
			roleSet{inlineLevel: true, parent: true}},
		{boxes.NewTextBox("", style, nil, "a"), boxes.KindText, "Text",
			roleSet{inlineLevel: true}},
		{boxes.NewInlineBlockBox("", style, nil, nil), boxes.KindInlineBlock, "InlineBlock",
			roleSet{inlineLevel: true, parent: true, blockContainer: true, atomic: true}},
		{boxes.NewBlockReplacedBox("", style, nil, nil), boxes.KindBlockReplaced, "BlockReplaced",
			roleSet{blockLevel: true, replaced: true}},
		{boxes.NewInlineReplacedBox("", style, nil, nil), boxes.KindInlineReplaced, "InlineReplaced",
			roleSet{inlineLevel: true, atomic: true, replaced: true}},
		{boxes.NewTableBox("", style, nil, nil), boxes.KindTable, "Table",
			roleSet{blockLevel: true, parent: true, table: true, tabular: true}},
		{boxes.NewInlineTableBox("", style, nil, nil), boxes.KindInlineTable, "InlineTable",
			roleSet{blockLevel: true, parent: true, table: true, tabular: true}},
		{boxes.NewTableRowGroupBox("", style, nil, nil), boxes.KindTableRowGroup, "TableRowGroup",
			roleSet{parent: true, tableChild: true, internalTable: true, tabular: true}},
		{boxes.NewTableRowBox("", style, nil, nil), boxes.KindTableRow, "TableRow",
			roleSet{parent: true, tableChild: true, internalTable: true, tabular: true}},
		{boxes.NewTableColumnGroupBox("", style, nil, nil), boxes.KindTableColumnGroup, "TableColumnGroup",
			roleSet{parent: true, tableChild: true, internalTable: true}},
		{boxes.NewTableColumnBox("", style, nil, nil), boxes.KindTableColumn, "TableColumn",
			roleSet{parent: true, tableChild: true, internalTable: true}},
		{boxes.NewTableCellBox("", style, nil, nil), boxes.KindTableCell, "TableCell",
			roleSet{parent: true, blockContainer: true, internalTable: true}},
		{boxes.NewTableCaptionBox("", style, nil, nil), boxes.KindTableCaption, "TableCaption",
			roleSet{blockLevel: true, parent: true, blockContainer: true, tableChild: true, internalTable: true}},
		{boxes.NewPageBox(cascade.PageType{}, style), boxes.KindPage, "Page",
			roleSet{parent: true}},
		{boxes.NewMarginBox("@top-left", style), boxes.KindMargin, "Margin",
			roleSet{parent: true, blockContainer: true}},
		{boxes.NewFootnoteAreaBox(nil, style), boxes.KindFootnoteArea, "FootnoteArea",
			roleSet{blockLevel: true, parent: true, blockContainer: true}},
		{boxes.NewFlexBox("", style, nil, nil), boxes.KindFlex, "Flex",
			roleSet{blockLevel: true, parent: true, flex: true}},
		{boxes.NewInlineFlexBox("", style, nil, nil), boxes.KindInlineFlex, "InlineFlex",
			roleSet{inlineLevel: true, parent: true, flex: true}},
		{boxes.NewGridBox("", style, nil, nil), boxes.KindGrid, "Grid",
			roleSet{blockLevel: true, parent: true, grid: true}},
		{boxes.NewInlineGridBox("", style, nil, nil), boxes.KindInlineGrid, "InlineGrid",
			roleSet{inlineLevel: true, parent: true, grid: true}},
	}
	seen := map[boxes.Kind]bool{}
	for _, tt := range tests {
		seen[tt.kind] = true
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.box.Kind(); got != tt.kind {
				t.Errorf("Kind = %v, want %v", got, tt.kind)
			}
			if got := tt.kind.String(); got != tt.name {
				t.Errorf("Kind.String = %q, want %q", got, tt.name)
			}
			if got := rolesOf(tt.box); got != tt.want {
				t.Errorf("roles = %+v, want %+v", got, tt.want)
			}
		})
	}
	for k := boxes.KindBlock; k <= boxes.KindInlineGrid; k++ {
		if !seen[k] {
			t.Errorf("no box of kind %v in the table", k)
		}
	}
}

func TestProperTableParents(t *testing.T) {
	style := plainStyle()
	tests := []struct {
		box  boxes.Box
		want []boxes.Kind
	}{
		{boxes.NewTableRowGroupBox("", style, nil, nil), []boxes.Kind{boxes.KindTable, boxes.KindInlineTable}},
		{boxes.NewTableRowBox("", style, nil, nil), []boxes.Kind{boxes.KindTable, boxes.KindInlineTable, boxes.KindTableRowGroup}},
		{boxes.NewTableColumnBox("", style, nil, nil), []boxes.Kind{boxes.KindTable, boxes.KindInlineTable, boxes.KindTableColumnGroup}},
		{boxes.NewTableCaptionBox("", style, nil, nil), []boxes.Kind{boxes.KindTable, boxes.KindInlineTable}},
		{boxes.NewTableCellBox("", style, nil, nil), nil},
		{boxes.NewBlockBox("", style, nil, nil), nil},
	}
	for _, tt := range tests {
		got := boxes.ProperParents(tt.box)
		if len(got) != len(tt.want) {
			t.Errorf("ProperParents(%v) = %v, want %v", tt.box.Kind(), got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ProperParents(%v) = %v, want %v", tt.box.Kind(), got, tt.want)
			}
		}
	}
}
