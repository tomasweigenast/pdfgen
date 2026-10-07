package boxes

import (
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/images"
)

func newBoxFields(kind Kind, elementTag string, style *computed.Style, element *html.Element) BoxFields {
	return BoxFields{kind: kind, ElementTag: elementTag, Style: style, Element: element}
}

// NewBlockBox returns a block box.
func NewBlockBox(tag string, style *computed.Style, el *html.Element, children []Box) *BlockBox {
	return &BlockBox{BlockContainerBox: BlockContainerBox{newParent(KindBlock, tag, style, el, children)}}
}

// NewLineBox returns a line box.
func NewLineBox(tag string, style *computed.Style, el *html.Element, children []Box) *LineBox {
	return &LineBox{
		ParentFields:  newParent(KindLine, tag, style, el, children),
		TextOverflow:  "clip",
		BlockEllipsis: props.KeywordOrString{Keyword: "none"},
	}
}

// NewInlineBox returns an inline box.
func NewInlineBox(tag string, style *computed.Style, el *html.Element, children []Box) *InlineBox {
	return &InlineBox{ParentFields: newParent(KindInline, tag, style, el, children)}
}

// NewTextBox returns a text box. The text must not be empty.
func NewTextBox(tag string, style *computed.Style, el *html.Element, text string) *TextBox {
	return &TextBox{BoxFields: newBoxFields(KindText, tag, style, el), Text: text}
}

// NewInlineBlockBox returns an inline-block box.
func NewInlineBlockBox(tag string, style *computed.Style, el *html.Element, children []Box) *InlineBlockBox {
	return &InlineBlockBox{BlockContainerBox: BlockContainerBox{newParent(KindInlineBlock, tag, style, el, children)}}
}

// NewBlockReplacedBox returns a block-level replaced box.
func NewBlockReplacedBox(tag string, style *computed.Style, el *html.Element, replacement images.Image) *BlockReplacedBox {
	return &BlockReplacedBox{ReplacedFields: ReplacedFields{newBoxFields(KindBlockReplaced, tag, style, el), replacement}}
}

// NewInlineReplacedBox returns an inline-level replaced box.
func NewInlineReplacedBox(tag string, style *computed.Style, el *html.Element, replacement images.Image) *InlineReplacedBox {
	return &InlineReplacedBox{ReplacedFields: ReplacedFields{newBoxFields(KindInlineReplaced, tag, style, el), replacement}}
}

// NewTableBox returns a table box.
func NewTableBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableBox {
	return &TableBox{ParentFields: newParent(KindTable, tag, style, el, children)}
}

// NewInlineTableBox returns an inline table box.
func NewInlineTableBox(tag string, style *computed.Style, el *html.Element, children []Box) *InlineTableBox {
	return &InlineTableBox{TableBox{ParentFields: newParent(KindInlineTable, tag, style, el, children)}}
}

// NewTableRowGroupBox returns a table row group box.
func NewTableRowGroupBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableRowGroupBox {
	return &TableRowGroupBox{ParentFields: newParent(KindTableRowGroup, tag, style, el, children)}
}

// NewTableRowBox returns a table row box.
func NewTableRowBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableRowBox {
	return &TableRowBox{ParentFields: newParent(KindTableRow, tag, style, el, children)}
}

// NewTableColumnGroupBox returns a table column group box.
func NewTableColumnGroupBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableColumnGroupBox {
	return &TableColumnGroupBox{ParentFields: newParent(KindTableColumnGroup, tag, style, el, children)}
}

// NewTableColumnBox returns a table column box.
func NewTableColumnBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableColumnBox {
	return &TableColumnBox{ParentFields: newParent(KindTableColumn, tag, style, el, children)}
}

// NewTableCellBox returns a table cell box. Its colspan and rowspan come
// from the attributes of the element: HTML 4.01 gave colspan=0 a meaning
// that HTML 5 removed, but rowspan=0 is still valid.
func NewTableCellBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableCellBox {
	c := &TableCellBox{
		BlockContainerBox: BlockContainerBox{newParent(KindTableCell, tag, style, el, children)},
		ColSpan:           1,
		RowSpan:           1,
	}
	if el != nil {
		c.ColSpan, c.RowSpan = el.ColSpan(), el.RowSpan()
	}
	return c
}

// NewTableCaptionBox returns a table caption box.
func NewTableCaptionBox(tag string, style *computed.Style, el *html.Element, children []Box) *TableCaptionBox {
	return &TableCaptionBox{BlockBox: *NewBlockBox(tag, style, el, children).withKind(KindTableCaption)}
}

func (b *BlockBox) withKind(k Kind) *BlockBox {
	b.kind = k
	return b
}

// NewPageBox returns the box of a page of the given type.
func NewPageBox(pageType cascade.PageType, style *computed.Style) *PageBox {
	return &PageBox{ParentFields: newParent(KindPage, "", style, nil, nil), PageType: pageType}
}

// NewMarginBox returns the box of the margin at-rule atKeyword.
func NewMarginBox(atKeyword string, style *computed.Style) *MarginBox {
	return &MarginBox{
		BlockContainerBox: BlockContainerBox{newParent(KindMargin, "", style, nil, nil)},
		AtKeyword:         atKeyword,
	}
}

// NewFootnoteAreaBox returns the footnote area of a page.
func NewFootnoteAreaBox(page *PageBox, style *computed.Style) *FootnoteAreaBox {
	return &FootnoteAreaBox{BlockBox: *NewBlockBox("", style, nil, nil).withKind(KindFootnoteArea), Page: page}
}

// NewFlexBox returns a block-level flex container.
func NewFlexBox(tag string, style *computed.Style, el *html.Element, children []Box) *FlexBox {
	return &FlexBox{FlexContainerBox: FlexContainerBox{newParent(KindFlex, tag, style, el, children)}}
}

// NewInlineFlexBox returns an inline-level flex container.
func NewInlineFlexBox(tag string, style *computed.Style, el *html.Element, children []Box) *InlineFlexBox {
	return &InlineFlexBox{FlexContainerBox: FlexContainerBox{newParent(KindInlineFlex, tag, style, el, children)}}
}

func newGridContainer(kind Kind, tag string, style *computed.Style, el *html.Element, children []Box) GridContainerBox {
	return GridContainerBox{
		ParentFields: newParent(kind, tag, style, el, children),
		Advancements: map[[2]int]float64{},
	}
}

// NewGridBox returns a block-level grid container.
func NewGridBox(tag string, style *computed.Style, el *html.Element, children []Box) *GridBox {
	return &GridBox{GridContainerBox: newGridContainer(KindGrid, tag, style, el, children)}
}

// NewInlineGridBox returns an inline-level grid container.
func NewInlineGridBox(tag string, style *computed.Style, el *html.Element, children []Box) *InlineGridBox {
	return &InlineGridBox{GridContainerBox: newGridContainer(KindInlineGrid, tag, style, el, children)}
}

// New returns an empty-bodied box of a kind that has children, or nil for
// the other kinds: text and replaced boxes need a text or an image, and
// pages, margin boxes and footnote areas have no element.
func New(kind Kind, tag string, style *computed.Style, el *html.Element, children []Box) Parent {
	switch kind {
	case KindBlock:
		return NewBlockBox(tag, style, el, children)
	case KindLine:
		return NewLineBox(tag, style, el, children)
	case KindInline:
		return NewInlineBox(tag, style, el, children)
	case KindInlineBlock:
		return NewInlineBlockBox(tag, style, el, children)
	case KindTable:
		return NewTableBox(tag, style, el, children)
	case KindInlineTable:
		return NewInlineTableBox(tag, style, el, children)
	case KindTableRowGroup:
		return NewTableRowGroupBox(tag, style, el, children)
	case KindTableRow:
		return NewTableRowBox(tag, style, el, children)
	case KindTableColumnGroup:
		return NewTableColumnGroupBox(tag, style, el, children)
	case KindTableColumn:
		return NewTableColumnBox(tag, style, el, children)
	case KindTableCell:
		return NewTableCellBox(tag, style, el, children)
	case KindTableCaption:
		return NewTableCaptionBox(tag, style, el, children)
	case KindFlex:
		return NewFlexBox(tag, style, el, children)
	case KindInlineFlex:
		return NewInlineFlexBox(tag, style, el, children)
	case KindGrid:
		return NewGridBox(tag, style, el, children)
	case KindInlineGrid:
		return NewInlineGridBox(tag, style, el, children)
	}
	return nil
}

// AnonymousFrom returns an anonymous box of the given kind that inherits
// from parent: it has the element tag and element of parent and the
// anonymous style of its style (CSS 2.1 §9.2.1.1), where inherited
// properties come from the parent and the others have their initial value.
// It returns nil for the kinds that New does not make.
func AnonymousFrom(kind Kind, parent Box, children []Box) Parent {
	p := parent.Base()
	box := New(kind, p.ElementTag, p.Style.Anonymous(), p.Element, children)
	if line, ok := box.(*LineBox); ok && p.Style.Overflow() != "visible" {
		line.TextOverflow = p.Style.TextOverflow()
	}
	return box
}

// AnonymousText returns an anonymous text box that inherits from parent.
func AnonymousText(parent Box, text string) *TextBox {
	p := parent.Base()
	return NewTextBox(p.ElementTag, p.Style.Anonymous(), p.Element, text)
}

// AnonymousBlockReplaced returns an anonymous block-level replaced box that
// inherits from parent.
func AnonymousBlockReplaced(parent Box, replacement images.Image) *BlockReplacedBox {
	p := parent.Base()
	return NewBlockReplacedBox(p.ElementTag, p.Style.Anonymous(), p.Element, replacement)
}

// AnonymousInlineReplaced returns an anonymous inline-level replaced box
// that inherits from parent.
func AnonymousInlineReplaced(parent Box, replacement images.Image) *InlineReplacedBox {
	p := parent.Base()
	return NewInlineReplacedBox(p.ElementTag, p.Style.Anonymous(), p.Element, replacement)
}

// CopyWithText returns a copy of the text box with another text, which must
// not be empty.
func (b *TextBox) CopyWithText(text string) *TextBox {
	c := copyBox(b)
	c.Text = text
	return c
}
