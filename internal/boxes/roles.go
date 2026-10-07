package boxes

// Parent is a box that has children.
type Parent interface {
	Box
	ParentBase() *ParentFields
	// CopyWithChildren returns a copy of the box that has the given
	// children. The decoration removed from the original is reset on it,
	// as it is when a box is split between pages.
	CopyWithChildren(children []Box) Box
}

// BlockLevel is a box that participates in a block formatting context.
type BlockLevel interface {
	Box
	blockLevel()
}

// InlineLevel is a box that participates in an inline formatting context.
type InlineLevel interface {
	Box
	inlineLevel()
}

// BlockContainer is a box that contains only block-level boxes or only line
// boxes.
type BlockContainer interface {
	Parent
	blockContainer()
}

// AtomicInline is an inline-level box that cannot be split across lines.
type AtomicInline interface {
	InlineLevel
	atomic()
}

// Replaced is a box whose content is replaced, such as an image.
type Replaced interface {
	Box
	ReplacedBase() *ReplacedFields
}

// FlexContainer is a box that contains only flex items.
type FlexContainer interface {
	Parent
	flexContainer()
}

// GridContainer is a box that contains only grid items.
type GridContainer interface {
	Parent
	GridBase() *GridContainerBox
}

// TableLike is a table box, block-level or inline-level.
type TableLike interface {
	Parent
	TableBase() *TableBox
}

// TableChild is a box that can be the child of a table without being
// wrapped in an anonymous box: a row group, row, column group, column or
// caption (CSS 2.1 §17.2.1).
type TableChild interface {
	Box
	properTableChild()
}

// InternalTableOrCaption is a table-internal box or a caption: a row group,
// row, column group, column, cell or caption.
type InternalTableOrCaption interface {
	Box
	internalTableOrCaption()
}

// TabularContainer is a table, row group or row.
type TabularContainer interface {
	Box
	tabularContainer()
}

// Marker types embedded in the concrete boxes to give them their roles.
// Their methods only exist to make the interfaces above unforgeable.
type (
	// BlockLevelBox gives a box the block-level outside behaviour.
	BlockLevelBox struct{}
	// InlineLevelBox gives a box the inline-level outside behaviour.
	InlineLevelBox struct{}
	// AtomicInlineLevelBox is an inline-level box that cannot be split.
	AtomicInlineLevelBox struct{ InlineLevelBox }

	properTableChildMark       struct{}
	internalTableOrCaptionMark struct{}
	tabularContainerMark       struct{}
)

func (BlockLevelBox) blockLevel()                          {}
func (InlineLevelBox) inlineLevel()                        {}
func (AtomicInlineLevelBox) atomic()                       {}
func (properTableChildMark) properTableChild()             {}
func (internalTableOrCaptionMark) internalTableOrCaption() {}
func (tabularContainerMark) tabularContainer()             {}

// ProperParents returns the kinds of boxes that can be the parent of a
// table child without a wrapper, or nil if the box is not a table child.
func ProperParents(b Box) []Kind {
	switch b.Kind() {
	case KindTableRowGroup, KindTableColumnGroup, KindTableCaption:
		return []Kind{KindTable, KindInlineTable}
	case KindTableRow:
		return []Kind{KindTable, KindInlineTable, KindTableRowGroup}
	case KindTableColumn:
		return []Kind{KindTable, KindInlineTable, KindTableColumnGroup}
	}
	return nil
}
