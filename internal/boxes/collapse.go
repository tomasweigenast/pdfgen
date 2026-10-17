package boxes

import (
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// borderStyleScores ranks border styles in conflicts, the strongest last
// (CSS 2.1 §17.6.2.1).
var borderStyleScores = map[props.Keyword]float64{
	"none": 0, "inset": 1, "groove": 2, "outset": 3, "ridge": 4,
	"dotted": 5, "dashed": 6, "solid": 7, "double": 8, "hidden": 9,
}

// collapsedStyles maps the styles that have no meaning in the collapsing
// model to the ones they are drawn as (CSS 2.1 §17.6.2).
var collapsedStyles = map[props.Keyword]props.Keyword{"inset": "ridge", "outset": "groove"}

var transparent = props.Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)}

var (
	weakNullBorder   = CollapsedBorder{Score: [3]float64{0, 0, 0}, Style: "none", Color: transparent}
	strongNullBorder = CollapsedBorder{Score: [3]float64{1, 0, 9}, Style: "hidden", Color: transparent}
)

func lessScore(a, b [3]float64) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

type borderSide struct {
	style, width, color props.PropID
}

var (
	sideTop    = borderSide{props.PropBorderTopStyle, props.PropBorderTopWidth, props.PropBorderTopColor}
	sideRight  = borderSide{props.PropBorderRightStyle, props.PropBorderRightWidth, props.PropBorderRightColor}
	sideBottom = borderSide{props.PropBorderBottomStyle, props.PropBorderBottomWidth, props.PropBorderBottomColor}
	sideLeft   = borderSide{props.PropBorderLeftStyle, props.PropBorderLeftWidth, props.PropBorderLeftColor}
)

// wrapIndex turns a negative index into one counted from the end, as the
// right-to-left grids are addressed from their end.
func wrapIndex(i, n int) int {
	if i < 0 {
		return i + n
	}
	return i
}

// collapser resolves the border conflicts of one table.
type collapser struct {
	ltr                  bool
	vertical, horizontal [][]CollapsedBorder
}

func (c *collapser) setOne(grid [][]CollapsedBorder, style *computed.Style, side borderSide, x, y int) {
	borderStyle, _ := style.Get(side.style).(props.Keyword)
	width, _ := style.Get(side.width).(props.Pixels)
	hidden := 0.0
	if borderStyle == "hidden" {
		hidden = 1
	}
	score := [3]float64{hidden, float64(width), borderStyleScores[borderStyle]}
	if s, ok := collapsedStyles[borderStyle]; ok {
		borderStyle = s
	}
	row := grid[wrapIndex(y, len(grid))]
	x = wrapIndex(x, len(row))
	// Strictly greater, so that the box set first wins a tie.
	if lessScore(row[x].Score, score) {
		row[x] = CollapsedBorder{Score: score, Style: borderStyle, Width: float64(width), Color: style.ResolvedColor(side.color)}
	}
}

// setBorders sets the borders of a box covering w columns and h rows from
// the logical position x, y. The grid is always stored left to right.
func (c *collapser) setBorders(b Box, x, y, w, h int) {
	style := b.Base().Style
	if c.ltr {
		for yy := y; yy < y+h; yy++ {
			c.setOne(c.vertical, style, sideLeft, x, yy)
			c.setOne(c.vertical, style, sideRight, x+w, yy)
		}
		for xx := x; xx < x+w; xx++ {
			c.setOne(c.horizontal, style, sideTop, xx, y)
			c.setOne(c.horizontal, style, sideBottom, xx, y+h)
		}
		return
	}
	for yy := y; yy < y+h; yy++ {
		c.setOne(c.vertical, style, sideLeft, -1-w-x, yy)
		c.setOne(c.vertical, style, sideRight, -1-x, yy)
	}
	for xx := -1 - x; xx > -1-x-w; xx-- {
		c.setOne(c.horizontal, style, sideTop, xx, y)
		c.setOne(c.horizontal, style, sideBottom, xx, y+h)
	}
}

func (c *collapser) maxVerticalWidth(x, y1, y2 int) float64 {
	var m float64
	for i, row := range c.vertical[y1:y2] {
		if w := row[wrapIndex(x, len(row))].Width; i == 0 || w > m {
			m = w
		}
	}
	return m
}

// maxHorizontalWidth returns the widest border between x1 and x2 in row y.
// Negative bounds count from the end, and end == nil stands for the end.
func (c *collapser) maxHorizontalWidth(x1, y int, x2 *int) float64 {
	row := c.horizontal[y]
	n := len(row)
	start, end := clampIndex(x1, n), n
	if x2 != nil {
		end = clampIndex(*x2, n)
	}
	var m float64
	for i := start; i < end; i++ {
		if w := row[i].Width; i == start || w > m {
			m = w
		}
	}
	return m
}

func clampIndex(i, n int) int {
	if i < 0 {
		i += n
	}
	return max(0, min(i, n))
}

func setBorderWidths(b Box, top, right, bottom, left float64) {
	f := b.Base()
	f.CollapsedBordersResolved = true
	f.BorderTopWidth, f.BorderRightWidth = top, right
	f.BorderBottomWidth, f.BorderLeftWidth = bottom, left
}

// CollapseTableBorders resolves the conflicts between the borders of a table
// in the collapsing border model (CSS 2.1 §17.6.2). The cells, rows, row
// groups, columns and column groups of the table must have their grid
// positions. It sets the used border widths of the table and of its
// internal boxes: half of the widest border on each side for the table and
// the cells, and zero for the others, as the grid is drawn separately.
func CollapseTableBorders(table *TableBox, gridWidth, gridHeight int) *CollapsedBorderGrid {
	if gridWidth == 0 || gridHeight == 0 {
		return &CollapsedBorderGrid{}
	}
	c := &collapser{ltr: table.Style.Direction() == "ltr"}
	c.vertical = make([][]CollapsedBorder, gridHeight)
	for i := range c.vertical {
		c.vertical[i] = filledBorders(gridWidth + 1)
	}
	c.horizontal = make([][]CollapsedBorder, gridHeight+1)
	for i := range c.horizontal {
		c.horizontal[i] = filledBorders(gridWidth)
	}

	// "A style set on a cell wins over one on a row, which wins over a row
	// group, column, column group and, lastly, table" (CSS 2.1 §17.6.2.1).
	y := 0
	for _, group := range table.Children {
		for _, row := range ChildrenOf(group) {
			for _, child := range ChildrenOf(row) {
				cell, ok := child.(*TableCellBox)
				if !ok {
					continue
				}
				c.nullInsideCell(cell, y)
				c.setBorders(cell, cell.GridX, y, cell.ColSpan, cell.RowSpan)
			}
			y++
		}
	}
	y = 0
	for _, group := range table.Children {
		for _, row := range ChildrenOf(group) {
			c.setBorders(row, 0, y, gridWidth, 1)
			y++
		}
	}
	y = 0
	for _, group := range table.Children {
		rows := len(ChildrenOf(group))
		c.setBorders(group, 0, y, gridWidth, rows)
		y += rows
	}
	for _, group := range table.ColumnGroups {
		for _, child := range group.Children {
			if col, ok := child.(*TableColumnBox); ok {
				c.setBorders(col, col.GridX, 0, 1, gridHeight)
			}
		}
	}
	for _, group := range table.ColumnGroups {
		c.setBorders(group, group.GridX, 0, group.Span(), gridHeight)
	}
	c.setBorders(table, 0, 0, gridWidth, gridHeight)

	c.setUsedWidths(table, gridWidth, gridHeight)
	return &CollapsedBorderGrid{Vertical: c.vertical, Horizontal: c.horizontal}
}

func filledBorders(n int) []CollapsedBorder {
	row := make([]CollapsedBorder, n)
	for i := range row {
		row[i] = weakNullBorder
	}
	return row
}

// nullInsideCell removes the borders inside a cell that spans several
// columns or rows.
func (c *collapser) nullInsideCell(cell *TableCellBox, y int) {
	x, colspan, rowspan := cell.GridX, cell.ColSpan, cell.RowSpan
	var vertical, horizontal []int
	if c.ltr {
		for xx := x + 1; xx < x+colspan; xx++ {
			vertical = append(vertical, xx)
		}
		for xx := x; xx < x+colspan; xx++ {
			horizontal = append(horizontal, xx)
		}
	} else {
		for xx := -2 - x; xx > -1-x-colspan; xx-- {
			vertical = append(vertical, xx)
		}
		for xx := -1 - x; xx > -1-x-colspan; xx-- {
			horizontal = append(horizontal, xx)
		}
	}
	for _, xx := range vertical {
		for yy := y; yy < y+rowspan; yy++ {
			row := c.vertical[yy]
			row[wrapIndex(xx, len(row))] = strongNullBorder
		}
	}
	for _, xx := range horizontal {
		for yy := y + 1; yy < y+rowspan; yy++ {
			row := c.horizontal[yy]
			row[wrapIndex(xx, len(row))] = strongNullBorder
		}
	}
}

// setUsedWidths gives the table and its cells half of the collapsed borders
// around them, and removes the borders of the other internal boxes.
func (c *collapser) setUsedWidths(table *TableBox, gridWidth, gridHeight int) {
	y := 0
	for _, group := range table.Children {
		setBorderWidths(group, 0, 0, 0, 0)
		for _, row := range ChildrenOf(group) {
			setBorderWidths(row, 0, 0, 0, 0)
			for _, child := range ChildrenOf(row) {
				cell, ok := child.(*TableCellBox)
				if !ok {
					continue
				}
				x, colspan, rowspan := cell.GridX, cell.ColSpan, cell.RowSpan
				var top, right, bottom, left float64
				if c.ltr {
					end := x + colspan
					top = c.maxHorizontalWidth(x, y, &end)
					bottom = c.maxHorizontalWidth(x, y+rowspan, &end)
					left = c.maxVerticalWidth(x, y, y+rowspan)
					right = c.maxVerticalWidth(x+colspan, y, y+rowspan)
				} else {
					var end *int
					if x != 0 {
						e := -x
						end = &e
					}
					top = c.maxHorizontalWidth(-colspan-x, y, end)
					bottom = c.maxHorizontalWidth(-colspan-x, y+rowspan, end)
					left = c.maxVerticalWidth(-1-colspan-x, y, y+rowspan)
					right = c.maxVerticalWidth(-1-x, y, y+rowspan)
				}
				setBorderWidths(cell, top/2, right/2, bottom/2, left/2)
			}
			y++
		}
	}
	for _, group := range table.ColumnGroups {
		setBorderWidths(group, 0, 0, 0, 0)
		for _, col := range group.Children {
			setBorderWidths(col, 0, 0, 0, 0)
		}
	}
	// "UAs must compute an initial left and right border width for the
	// table by examining the first and last cells in the first row of the
	// table" (CSS 2.1 §17.6.2), so only the first row counts.
	setBorderWidths(table,
		c.maxHorizontalWidth(0, 0, &gridWidth)/2,
		c.maxVerticalWidth(gridWidth, 0, 1)/2,
		c.maxHorizontalWidth(0, gridHeight, &gridWidth)/2,
		c.maxVerticalWidth(0, 0, 1)/2)
}
