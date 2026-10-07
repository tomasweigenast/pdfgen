package boxes

// Copy, DeepCopy and CopyWithChildren are written out for each type because a
// copy has to keep the concrete type of the box, which embedding cannot do.

// Copy returns a shallow copy of the box.
func (b *BlockBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *LineBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *InlineBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *InlineBlockBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *InlineTableBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableRowGroupBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableRowBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableColumnGroupBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableColumnBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableCellBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TableCaptionBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *PageBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *MarginBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *FootnoteAreaBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *FlexBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *InlineFlexBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *GridBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *InlineGridBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *TextBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *BlockReplacedBox) Copy() Box { return copyBox(b) }

// Copy returns a shallow copy of the box.
func (b *InlineReplacedBox) Copy() Box { return copyBox(b) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *BlockBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *BlockBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *LineBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *LineBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *InlineBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *InlineBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *InlineBlockBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *InlineBlockBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *InlineTableBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *InlineTableBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableRowGroupBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableRowGroupBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableRowBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableRowBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableColumnGroupBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableColumnGroupBox) CopyWithChildren(children []Box) Box {
	return copyWithChildren(b, children)
}

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableColumnBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableColumnBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableCellBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableCellBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *TableCaptionBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *TableCaptionBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *PageBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *PageBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *MarginBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *MarginBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *FootnoteAreaBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *FootnoteAreaBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *FlexBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *FlexBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *InlineFlexBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *InlineFlexBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *GridBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *GridBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box with copies of its descendants.
func (b *InlineGridBox) DeepCopy() Box { return deepCopyParent(b) }

// CopyWithChildren returns a copy of the box with the given children.
func (b *InlineGridBox) CopyWithChildren(children []Box) Box { return copyWithChildren(b, children) }

// DeepCopy returns a copy of the box.
func (b *TextBox) DeepCopy() Box { return b.Copy() }

// DeepCopy returns a copy of the box.
func (b *BlockReplacedBox) DeepCopy() Box { return b.Copy() }

// DeepCopy returns a copy of the box.
func (b *InlineReplacedBox) DeepCopy() Box { return b.Copy() }

var (
	_ Parent = (*BlockBox)(nil)

	_ Parent = (*LineBox)(nil)

	_ Parent = (*InlineBox)(nil)

	_ Parent = (*InlineBlockBox)(nil)

	_ Parent = (*TableBox)(nil)

	_ Parent = (*InlineTableBox)(nil)

	_ Parent = (*TableRowGroupBox)(nil)

	_ Parent = (*TableRowBox)(nil)

	_ Parent = (*TableColumnGroupBox)(nil)

	_ Parent = (*TableColumnBox)(nil)

	_ Parent = (*TableCellBox)(nil)

	_ Parent = (*TableCaptionBox)(nil)

	_ Parent = (*PageBox)(nil)

	_ Parent = (*MarginBox)(nil)

	_ Parent = (*FootnoteAreaBox)(nil)

	_ Parent = (*FlexBox)(nil)

	_ Parent = (*InlineFlexBox)(nil)

	_ Parent = (*GridBox)(nil)

	_ Parent = (*InlineGridBox)(nil)

	_ Box = (*TextBox)(nil)

	_ Box = (*BlockReplacedBox)(nil)

	_ Box = (*InlineReplacedBox)(nil)
)
