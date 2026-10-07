package boxes

// Kind identifies the concrete type of a box.
type Kind uint8

// The kinds of boxes. KindInvalid is the zero value and never belongs to a
// box made by a constructor.
const (
	KindInvalid Kind = iota
	KindBlock
	KindLine
	KindInline
	KindText
	KindInlineBlock
	KindBlockReplaced
	KindInlineReplaced
	KindTable
	KindInlineTable
	KindTableRowGroup
	KindTableRow
	KindTableColumnGroup
	KindTableColumn
	KindTableCell
	KindTableCaption
	KindPage
	KindMargin
	KindFootnoteArea
	KindFlex
	KindInlineFlex
	KindGrid
	KindInlineGrid
	numKinds
)

// kindProps describes the roles of a kind that BoxFields methods depend on.
type kindProps struct {
	name string
	// atomic and replaced make a box monolithic.
	atomic, replaced bool
	// blockContainer and blockBox mirror the class hierarchy: a table
	// caption and a footnote area are block boxes, a table cell and a
	// margin box are block containers.
	blockContainer, blockBox bool
	// inlineDecoration selects the left and right sides when the
	// decoration of a split box is removed, instead of the top and bottom.
	inlineDecoration bool
}

var kinds = [numKinds]kindProps{
	KindInvalid:          {name: "Invalid"},
	KindBlock:            {name: "Block", blockContainer: true, blockBox: true},
	KindLine:             {name: "Line"},
	KindInline:           {name: "Inline", inlineDecoration: true},
	KindText:             {name: "Text", inlineDecoration: true},
	KindInlineBlock:      {name: "InlineBlock", atomic: true, blockContainer: true, inlineDecoration: true},
	KindBlockReplaced:    {name: "BlockReplaced", replaced: true},
	KindInlineReplaced:   {name: "InlineReplaced", atomic: true, replaced: true, inlineDecoration: true},
	KindTable:            {name: "Table"},
	KindInlineTable:      {name: "InlineTable"},
	KindTableRowGroup:    {name: "TableRowGroup"},
	KindTableRow:         {name: "TableRow"},
	KindTableColumnGroup: {name: "TableColumnGroup"},
	KindTableColumn:      {name: "TableColumn"},
	KindTableCell:        {name: "TableCell", blockContainer: true},
	KindTableCaption:     {name: "TableCaption", blockContainer: true, blockBox: true},
	KindPage:             {name: "Page"},
	KindMargin:           {name: "Margin", blockContainer: true},
	KindFootnoteArea:     {name: "FootnoteArea", blockContainer: true, blockBox: true},
	KindFlex:             {name: "Flex"},
	KindInlineFlex:       {name: "InlineFlex"},
	KindGrid:             {name: "Grid"},
	KindInlineGrid:       {name: "InlineGrid"},
}

// String returns the name of the box type without the Box suffix, such as
// "Block" or "InlineReplaced".
func (k Kind) String() string {
	if k >= numKinds {
		return "Invalid"
	}
	return kinds[k].name
}
