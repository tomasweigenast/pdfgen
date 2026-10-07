package boxes

import "iter"

// ParentFields is what boxes with children have: the children.
type ParentFields struct {
	BoxFields
	Children []Box
}

// ParentBase returns the fields of a box with children.
func (p *ParentFields) ParentBase() *ParentFields { return p }

// ParentBox is the name of the CSS 2.1 class that ParentFields implements.
type ParentBox = ParentFields

// Translate moves the box and its children.
func (p *ParentFields) Translate(dx, dy float64, ignoreFloats bool) {
	if dx == 0 && dy == 0 {
		return
	}
	p.PositionX += dx
	p.PositionY += dy
	translateAll(p.Children, dx, dy, ignoreFloats)
}

func translateAll(children []Box, dx, dy float64, ignoreFloats bool) {
	for _, child := range children {
		if ignoreFloats && child.IsFloated() {
			continue
		}
		child.Translate(dx, dy, ignoreFloats)
	}
}

// PageValues returns the page names at the start and the end of the box,
// which are those of its first and last children when they are in flow or
// floated (CSS Break 4 §3.1.1).
func (p *ParentFields) PageValues() (start, end string) {
	start, end = p.BoxFields.PageValues()
	var children []Box
	for _, child := range p.Children {
		if child.IsInNormalFlow() || child.IsFloated() {
			children = append(children, child)
		}
	}
	switch len(children) {
	case 0:
	case 1:
		s, e := children[0].PageValues()
		start, end = orDefault(s, start), orDefault(e, end)
	default:
		s, _ := children[0].PageValues()
		_, e := children[len(children)-1].PageValues()
		start, end = orDefault(s, start), orDefault(e, end)
	}
	return start, end
}

func orDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

// ChildrenOf returns the children of a box, nil if it has none.
func ChildrenOf(b Box) []Box {
	if p, ok := b.(Parent); ok {
		return p.ParentBase().Children
	}
	return nil
}

// AllChildren returns the children of a box and, for a table, its column
// groups.
func AllChildren(b Box) []Box {
	if t, ok := b.(TableLike); ok {
		tb := t.TableBase()
		out := make([]Box, 0, len(tb.Children)+len(tb.ColumnGroups))
		out = append(out, tb.Children...)
		for _, g := range tb.ColumnGroups {
			out = append(out, g)
		}
		return out
	}
	return ChildrenOf(b)
}

// Descendants iterates over a box, then its children and their descendants,
// depth first. An absolute placeholder is yielded without descending into
// the box it stands for, unless placeholders is set, in which case the
// placeholder is replaced by that box.
func Descendants(b Box, placeholders bool) iter.Seq[Box] {
	return func(yield func(Box) bool) {
		descend(b, placeholders, yield)
	}
}

func descend(b Box, placeholders bool, yield func(Box) bool) bool {
	if ph, ok := b.(*AbsolutePlaceholder); ok {
		if !placeholders {
			return yield(b)
		}
		return descend(ph.Box, placeholders, yield)
	}
	if !yield(b) {
		return false
	}
	for _, child := range ChildrenOf(b) {
		if !descend(child, placeholders, yield) {
			return false
		}
	}
	return true
}

func copyBox[T any](b *T) *T {
	c := *b
	return &c
}

func copyWithChildren[P Parent](b P, children []Box) P {
	c := b.Copy().(P)
	c.ParentBase().Children = children
	// A fragment keeps the sides it already had; the original starts over,
	// so the next fragment cut from it begins without removed decoration.
	b.Base().RemoveDecorationSides = 0
	return c
}

func deepCopyParent[P Parent](b P) P {
	c := b.Copy().(P)
	pb := c.ParentBase()
	pb.Children = deepCopyAll(pb.Children)
	return c
}

func deepCopyAll(children []Box) []Box {
	out := make([]Box, len(children))
	for i, child := range children {
		out[i] = child.DeepCopy()
	}
	return out
}

// GetWrappedTable returns the table, inline-level or not, wrapped by a table
// wrapper box, or nil.
func GetWrappedTable(b Box) TableLike {
	for _, child := range ChildrenOf(b) {
		if t, ok := child.(TableLike); ok {
			return t
		}
	}
	return nil
}
