package boxes

// AbsolutePlaceholder stands in the tree for an absolutely positioned box
// that was taken out of the flow. Until the box is laid out its descendants
// have no position; SetLaidOutBox replaces it with the result.
//
// The placeholder behaves as the box it holds for every method of [Box], but
// it is none of the role interfaces: code that walks the tree sees an opaque
// child and unwraps it explicitly through the Box field.
type AbsolutePlaceholder struct {
	Box
	// LayoutDone tells that Box is the laid-out box.
	LayoutDone bool
}

// NewAbsolutePlaceholder returns a placeholder for box.
func NewAbsolutePlaceholder(box Box) *AbsolutePlaceholder {
	return &AbsolutePlaceholder{Box: box}
}

// SetLaidOutBox replaces the box by its laid-out version.
func (p *AbsolutePlaceholder) SetLaidOutBox(box Box) {
	p.Box = box
	p.LayoutDone = true
}

// Translate moves the box. Before layout only the box itself has a position,
// so its descendants are not moved.
func (p *AbsolutePlaceholder) Translate(dx, dy float64, ignoreFloats bool) {
	if dx == 0 && dy == 0 {
		return
	}
	if p.LayoutDone {
		p.Box.Translate(dx, dy, ignoreFloats)
		return
	}
	b := p.Box.Base()
	b.PositionX += dx
	b.PositionY += dy
}

// Copy returns a placeholder for a copy of the box.
func (p *AbsolutePlaceholder) Copy() Box {
	return &AbsolutePlaceholder{Box: p.Box.Copy(), LayoutDone: p.LayoutDone}
}

// DeepCopy returns a placeholder for a deep copy of the box.
func (p *AbsolutePlaceholder) DeepCopy() Box {
	return &AbsolutePlaceholder{Box: p.Box.DeepCopy(), LayoutDone: p.LayoutDone}
}
