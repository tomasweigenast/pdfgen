package boxes

import (
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func isKind(k Kind) func(Box) bool { return func(b Box) bool { return b.Kind() == k } }

func isTableChild(b Box) bool {
	_, ok := b.(TableChild)
	return ok
}

func isInternalTableBox(b Box) bool {
	_, ok := b.(InternalTableOrCaption)
	return ok
}

// anonymousTableBoxes removes and adds boxes according to the table model
// in box and its descendants (CSS 2.1 §17.2.1). Tables are replaced by
// their wrapper boxes.
func anonymousTableBoxes(box Box) Box {
	p, ok := box.(Parent)
	if !ok || box.IsRunning() {
		return box
	}
	old := p.ParentBase().Children
	children := make([]Box, len(old))
	for i, child := range old {
		children[i] = anonymousTableBoxes(child)
	}
	return tableBoxesChildren(p, children)
}

// tableBoxesChildren applies the rules of the table model to the children
// of box, which have already been processed, and sets them.
func tableBoxesChildren(box Parent, children []Box) Box {
	switch box.Kind() {
	case KindTableColumn:
		// Rule 1.1: columns have no children.
		children = nil
	case KindTableColumnGroup:
		// Rule 1.2: column groups only have columns, at least one of them.
		span := box.(*TableColumnGroupBox).Span()
		children = slices.DeleteFunc(children, func(c Box) bool { return c.Kind() != KindTableColumn })
		if len(children) == 0 {
			for range max(span, 1) {
				children = append(children, AnonymousFrom(KindTableColumn, box, nil))
			}
		}
	}

	// Rule 1.3: white space at the start and end of tabular containers is
	// removed; rule 1.4 removes it between internal boxes.
	if _, ok := box.(TabularContainer); ok && len(children) >= 2 {
		if n := len(children); isInternalTableBox(children[n-2]) && IsWhitespace(children[n-1]) {
			children = children[:n-1]
		}
		if len(children) >= 2 && isInternalTableBox(children[1]) && IsWhitespace(children[0]) {
			children = children[1:]
		}
	}
	kept := make([]Box, 0, len(children))
	for i, child := range children {
		if i > 0 && i < len(children)-1 && isInternalTableBox(children[i-1]) &&
			isInternalTableBox(children[i+1]) && IsWhitespace(child) {
			continue
		}
		kept = append(kept, child)
	}
	children = kept

	kind := box.Kind()
	switch kind {
	case KindTable, KindInlineTable:
		// Rule 2.1
		children = wrapImproper(box, children, KindTableRow, isTableChild)
	case KindTableRowGroup:
		// Rule 2.2
		children = wrapImproper(box, children, KindTableRow, isKind(KindTableRow))
	}
	if kind == KindTableRow {
		// Rule 2.3
		children = wrapImproper(box, children, KindTableCell, isKind(KindTableCell))
	} else {
		// Rule 3.1
		children = wrapImproper(box, children, KindTableRow, func(c Box) bool { return c.Kind() != KindTableCell })
	}
	// Rule 3.2
	if kind == KindInline {
		children = wrapImproper(box, children, KindInlineTable, func(c Box) bool { return !isTableChild(c) })
	} else {
		children = wrapImproper(box, children, KindTable, func(c Box) bool {
			return !isTableChild(c) || slices.Contains(ProperParents(c), kind)
		})
	}

	if t, ok := box.(TableLike); ok {
		return wrapTable(t, children)
	}
	box.ParentBase().Children = children
	return box
}

// wrapImproper wraps the consecutive children that fail proper in
// anonymous boxes of the given kind, to which the rules of the table model
// apply in turn.
func wrapImproper(box Parent, children []Box, kind Kind, proper func(Box) bool) []Box {
	var out, improper []Box
	flush := func() {
		wrapper := AnonymousFrom(kind, box, nil)
		out = append(out, tableBoxesChildren(wrapper, improper))
		improper = nil
	}
	for _, child := range children {
		if !proper(child) {
			improper = append(improper, child)
			continue
		}
		if len(improper) > 0 {
			flush()
		}
		out = append(out, child)
	}
	if len(improper) > 0 {
		flush()
	}
	return out
}

// wrapTable returns a table in its wrapper box, with its children grouped
// and ordered, and grid positions given to its columns and cells (CSS 2.1
// §17.4, §17.5). The grid y of a cell is the index of its row.
func wrapTable(box TableLike, children []Box) Box {
	var columns, rows, top, bottom []Box
	for _, child := range children {
		switch child.Kind() {
		case KindTableColumn, KindTableColumnGroup:
			columns = append(columns, child)
		case KindTableRow, KindTableRowGroup:
			rows = append(rows, child)
		case KindTableCaption:
			if child.Base().Style.CaptionSide() == "bottom" {
				bottom = append(bottom, child)
			} else {
				top = append(top, child)
			}
		}
	}

	var groups []*TableColumnGroupBox
	gridX := 0
	for _, g := range wrapImproper(box, columns, KindTableColumnGroup, isKind(KindTableColumnGroup)) {
		group := g.(*TableColumnGroupBox)
		groups = append(groups, group)
		group.GridX = gridX
		if len(group.Children) == 0 {
			gridX += group.Span()
			continue
		}
		// A span attribute already made one column for each spanned one.
		for _, c := range group.Children {
			c.(*TableColumnBox).GridX = gridX
			gridX++
		}
	}
	gridWidth := gridX

	var header, footer Box
	var body []Box
	for _, group := range wrapImproper(box, rows, KindTableRowGroup, isKind(KindTableRowGroup)) {
		display := group.Base().Style.Display()
		g := group.(*TableRowGroupBox)
		switch {
		case len(display) == 1 && display[0] == "table-header-group" && header == nil:
			g.IsHeader = true
			header = group
		case len(display) == 1 && display[0] == "table-footer-group" && footer == nil:
			g.IsFooter = true
			footer = group
		default:
			body = append(body, group)
		}
	}
	var rowGroups []Box
	if header != nil {
		rowGroups = append(rowGroups, header)
	}
	rowGroups = append(rowGroups, body...)
	if footer != nil {
		rowGroups = append(rowGroups, footer)
	}

	// Rowspans cannot extend beyond their row group, so each group is
	// independent (CSS 2.1 §17.5).
	gridHeight := 0
	for _, group := range rowGroups {
		rowsOfGroup := ChildrenOf(group)
		// The columns occupied in each row by cells of the rows above.
		occupied := make([]map[int]bool, len(rowsOfGroup))
		for i := range occupied {
			occupied[i] = map[int]bool{}
		}
		for i, row := range rowsOfGroup {
			x := 0
			for _, child := range ChildrenOf(row) {
				cell, ok := child.(*TableCellBox)
				if !ok {
					continue
				}
				for occupied[i][x] {
					x++
				}
				cell.GridX = x
				next := x + cell.ColSpan
				// A rowspan of zero spans the rest of the group (HTML 4.01
				// §11.2.6).
				if cell.RowSpan != 1 {
					maxRowSpan := len(rowsOfGroup) - i
					var spanned []map[int]bool
					if cell.RowSpan == 0 {
						spanned = occupied[i+1:]
						cell.RowSpan = maxRowSpan
					} else {
						cell.RowSpan = min(cell.RowSpan, maxRowSpan)
						spanned = occupied[i+1 : i+cell.RowSpan]
					}
					for _, cells := range spanned {
						for xx := x; xx < next; xx++ {
							cells[xx] = true
						}
					}
				}
				x = next
				gridWidth = max(gridWidth, x)
			}
		}
		gridHeight += len(rowsOfGroup)
	}

	copied := box.CopyWithChildren(rowGroups)
	table := copied.(TableLike).TableBase()
	table.Style = table.Style.Copy()
	table.ColumnGroups = groups
	if table.Style.BorderCollapse() == "collapse" {
		table.CollapsedBorderGrid = CollapseTableBorders(table, gridWidth, gridHeight)
	}

	wrapperKind := KindBlock
	if box.Kind() == KindInlineTable {
		wrapperKind = KindInlineBlock
	}
	wrapperChildren := append(append(top, copied), bottom...)
	wrapper := AnonymousFrom(wrapperKind, box, wrapperChildren)
	base := wrapper.Base()
	base.Style = base.Style.Copy()
	base.IsTableWrapper = true
	// The non-inherited properties of the table element apply either to the
	// wrapper or to the table; the other one gets the initial value (CSS
	// 2.1 §17.4).
	for id := range props.PropID(props.NumProps()) {
		if id.TableWrapper() {
			base.Style.Set(id, table.Style.Get(id))
			table.Style.Set(id, id.Initial())
		}
	}
	return wrapper
}
