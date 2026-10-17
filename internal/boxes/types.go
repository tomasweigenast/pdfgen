package boxes

import (
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// BlockContainerBox is a box that contains only block-level boxes or only
// line boxes. It is embedded by every box that behaves as a block on the
// inside.
type BlockContainerBox struct {
	ParentFields
}

func (*BlockContainerBox) blockContainer() {}

// BlockBox is a block-level box that is also a block container, made by
// display: block and list-item.
type BlockBox struct {
	BlockContainerBox
	BlockLevelBox
}

// LineBox is a line in an inline formatting context. It contains inline-level
// boxes only. While the box tree is built a line box holds all the inline
// content of a block; layout splits it into one line box for each line.
type LineBox struct {
	ParentFields
	// TextOverflow is the text-overflow of the block container, which only
	// applies when it does not have overflow: visible.
	TextOverflow props.Keyword
	// BlockEllipsis is the block-ellipsis of the container, set on its last
	// line when the content is clamped.
	BlockEllipsis props.KeywordOrString
	// TextIndent is the indentation of the first line.
	TextIndent float64
}

// InlineBox is an inline box with inline children, made by display: inline.
type InlineBox struct {
	InlineLevelBox
	ParentFields
}

// TextBox is a box that contains only text. All the text of a document ends
// up in text boxes, including what CSS calls anonymous inline boxes.
type TextBox struct {
	InlineLevelBox
	BoxFields
	// Text is the text of the box, never empty.
	Text string
	// JustificationSpacing is the space added to each word separator when
	// the line is justified.
	JustificationSpacing float64
	// Layout is the shaped text, set by layout.
	Layout *linebreak.Layout
}

// InlineBlockBox is an inline-level box that is also a block container,
// made by display: inline-block.
type InlineBlockBox struct {
	AtomicInlineLevelBox
	BlockContainerBox
}

// ReplacedFields is what a box whose content is replaced has: the content.
type ReplacedFields struct {
	BoxFields
	Replacement images.Image
}

// ReplacedBox is the name of the CSS 2.1 class that ReplacedFields implements.
type ReplacedBox = ReplacedFields

// ReplacedBase returns the fields of a replaced box.
func (r *ReplacedFields) ReplacedBase() *ReplacedFields { return r }

// BlockReplacedBox is a replaced box that is block-level.
type BlockReplacedBox struct {
	ReplacedFields
	BlockLevelBox
}

// InlineReplacedBox is a replaced box that is inline-level.
type InlineReplacedBox struct {
	ReplacedFields
	AtomicInlineLevelBox
}

// TableBox is a box made by display: table.
type TableBox struct {
	BlockLevelBox
	ParentFields
	tabularContainerMark

	// ColumnGroups are the column groups, which are not children of the
	// table, with the columns they contain.
	ColumnGroups []*TableColumnGroupBox
	// ColumnPositions are the x positions of the columns, and ColumnWidths
	// their widths, set by layout.
	ColumnPositions []float64
	ColumnWidths    []float64
	// CollapsedBorderGrid holds the resolved borders when border-collapse
	// is collapse.
	CollapsedBorderGrid *CollapsedBorderGrid
	// SkippedRows is the number of rows laid out on previous pages.
	SkippedRows int
	// SkipCellBorderTop and SkipCellBorderBottom tell that the borders of
	// the cells at the start or end of the table are drawn by the table.
	SkipCellBorderTop, SkipCellBorderBottom bool
}

// TableBase returns the table box.
func (t *TableBox) TableBase() *TableBox { return t }

// Translate moves the table with its rows, column groups and column
// positions.
func (t *TableBox) Translate(dx, dy float64, ignoreFloats bool) {
	if t.ColumnPositions != nil {
		moved := make([]float64, len(t.ColumnPositions))
		for i, p := range t.ColumnPositions {
			moved[i] = p + dx
		}
		t.ColumnPositions = moved
	}
	if dx == 0 && dy == 0 {
		return
	}
	t.PositionX += dx
	t.PositionY += dy
	translateAll(t.Children, dx, dy, ignoreFloats)
	for _, g := range t.ColumnGroups {
		if ignoreFloats && g.IsFloated() {
			continue
		}
		g.Translate(dx, dy, ignoreFloats)
	}
}

// PageValues returns the page of the table itself: its rows do not decide
// where the table starts.
func (t *TableBox) PageValues() (start, end string) { return t.BoxFields.PageValues() }

// InlineTableBox is a box made by display: inline-table. Like in CSS 2.1 it
// is a table box, and so a block-level box in the formatting structure.
type InlineTableBox struct {
	TableBox
}

// TableRowGroupBox is a box made by display: table-row-group,
// table-header-group and table-footer-group.
type TableRowGroupBox struct {
	ParentFields
	properTableChildMark
	internalTableOrCaptionMark
	tabularContainerMark

	IsHeader, IsFooter bool
}

// TableRowBox is a box made by display: table-row.
type TableRowBox struct {
	ParentFields
	properTableChildMark
	internalTableOrCaptionMark
	tabularContainerMark
}

// TableColumnGroupBox is a box made by display: table-column-group. Its
// margins and paddings are always zero.
type TableColumnGroupBox struct {
	ParentFields
	properTableChildMark
	internalTableOrCaptionMark

	// GridX is the index of the first column of the group.
	GridX int
}

// Cells returns the cells that originate in the columns of the group.
func (g *TableColumnGroupBox) Cells() []*TableCellBox {
	var cells []*TableCellBox
	for _, child := range g.Children {
		if c, ok := child.(*TableColumnBox); ok {
			cells = append(cells, c.Cells()...)
		}
	}
	return cells
}

// Span returns the number of columns of the group: the number of its
// column children or, if it has none, its span attribute.
func (g *TableColumnGroupBox) Span() int {
	if len(g.Children) > 0 {
		return len(g.Children)
	}
	return spanOf(g.Element)
}

// TableColumnBox is a box made by display: table-column. Its margins and
// paddings are always zero.
type TableColumnBox struct {
	ParentFields
	properTableChildMark
	internalTableOrCaptionMark

	// GridX is the index of the column.
	GridX int
	// GetCells returns the cells that originate in the column, set by
	// layout.
	GetCells func() []*TableCellBox
}

// Cells returns the cells that originate in the column.
func (c *TableColumnBox) Cells() []*TableCellBox {
	if c.GetCells == nil {
		return nil
	}
	return c.GetCells()
}

// Span returns the span attribute of the column, at least 1.
func (c *TableColumnBox) Span() int { return spanOf(c.Element) }

func spanOf(el *html.Element) int {
	if el == nil {
		return 1
	}
	return el.Span()
}

// TableCellBox is a box made by display: table-cell.
type TableCellBox struct {
	BlockContainerBox
	internalTableOrCaptionMark

	// GridX is the index of the first column of the cell.
	GridX int
	// ColSpan and RowSpan are the number of columns and rows of the cell.
	// A rowspan of zero extends the cell to the end of its row group.
	ColSpan, RowSpan int
	// ComputedHeight is the height of the cell before the content of the
	// row is laid out, and ContentHeight the height of its content.
	ComputedHeight units.MaybeFloat
	ContentHeight  float64
	// Empty tells that the cell has no content, for empty-cells.
	Empty bool
	// VerticalAlign is top, middle, bottom or baseline.
	VerticalAlign string
}

// TableCaptionBox is a box made by display: table-caption.
type TableCaptionBox struct {
	BlockBox
	properTableChildMark
	internalTableOrCaptionMark
}

// PageBox is a page. During layout a new page box is made after every page
// break. It is not linked to any element.
type PageBox struct {
	ParentFields
	PageType cascade.PageType
	// FixedBoxes are the placeholders of position: fixed boxes, which are
	// repeated on every page.
	FixedBoxes []Box
	// CanvasBackground is the background of the canvas, taken from the root
	// element, set by layout.
	CanvasBackground *Background
}

// Bleed returns the widths of the bleed area on each side.
func (p *PageBox) Bleed() Sides {
	return Sides{
		Top:    p.Style.BleedTop().Dimension.Value,
		Right:  p.Style.BleedRight().Dimension.Value,
		Bottom: p.Style.BleedBottom().Dimension.Value,
		Left:   p.Style.BleedLeft().Dimension.Value,
	}
}

// BleedArea returns the page together with its bleed area.
func (p *PageBox) BleedArea() Rect {
	b := p.Bleed()
	return Rect{
		-b.Left, -b.Top,
		p.MarginWidth() + b.Left + b.Right,
		p.MarginHeight() + b.Top + b.Bottom,
	}
}

// String describes the page, as <Page :first>.
func (p *PageBox) String() string { return "<Page " + p.PageType.Name + ">" }

// MarginBox is a box in the margins of a page (CSS Paged Media 3 §4). It
// is not linked to any element.
type MarginBox struct {
	BlockContainerBox
	// AtKeyword is the name of the margin at-rule, such as @top-left.
	AtKeyword string
	// IsGenerated tells whether the margin box has content.
	IsGenerated bool
}

// String describes the margin box, as <Margin @top-left>.
func (m *MarginBox) String() string { return "<Margin " + m.AtKeyword + ">" }

// FootnoteAreaBox displays the footnotes of a page (CSS GCPM).
type FootnoteAreaBox struct {
	BlockBox
	Page *PageBox
}

// String describes the footnote area.
func (f *FootnoteAreaBox) String() string { return "<FootnoteArea @footnote>" }

// FlexContainerBox is a box that contains only flex items.
type FlexContainerBox struct {
	ParentFields
}

func (*FlexContainerBox) flexContainer() {}

// FlexBox is a block-level flex container, made by display: flex.
type FlexBox struct {
	FlexContainerBox
	BlockLevelBox
}

// InlineFlexBox is an inline-level flex container, made by display:
// inline-flex.
type InlineFlexBox struct {
	FlexContainerBox
	InlineLevelBox
}

// GridContainerBox is a box that contains only grid items.
type GridContainerBox struct {
	ParentFields
	// Advancements is the height already laid out of the items split
	// between pages, by position of the item in the grid.
	Advancements map[[2]int]float64
}

// GridBase returns the grid container fields.
func (g *GridContainerBox) GridBase() *GridContainerBox { return g }

// GridBox is a block-level grid container, made by display: grid.
type GridBox struct {
	GridContainerBox
	BlockLevelBox
}

// InlineGridBox is an inline-level grid container, made by display:
// inline-grid.
type InlineGridBox struct {
	GridContainerBox
	InlineLevelBox
}

// CollapsedBorder is a border of the grid of collapsed borders, after the
// conflicts between adjacent borders were resolved (CSS 2.1 §17.6.2.1).
type CollapsedBorder struct {
	// Score orders borders in a conflict: hidden first, then width, then
	// style.
	Score [3]float64
	Style props.Keyword
	Width float64
	Color props.Color
}

// CollapsedBorderGrid holds the borders of a table in the collapsing border
// model, left to right and top to bottom: Vertical[y][x] is the border at
// the left of the cell in column x and row y, Horizontal[y][x] the border
// above the cell.
type CollapsedBorderGrid struct {
	Vertical, Horizontal [][]CollapsedBorder
}

func newParent(kind Kind, tag string, style *computed.Style, el *html.Element, children []Box) ParentFields {
	return ParentFields{BoxFields: newBoxFields(kind, tag, style, el), Children: children}
}
