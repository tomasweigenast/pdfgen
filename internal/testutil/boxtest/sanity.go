package boxtest

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

// properChildren returns the alternatives for what a box can contain: all
// the children in flow must satisfy one of the predicates. It returns nil for
// the boxes that are not checked.
func properChildren(b boxes.Box) []func(boxes.Box) bool {
	isKind := func(kinds ...boxes.Kind) func(boxes.Box) bool {
		return func(c boxes.Box) bool {
			for _, k := range kinds {
				if c.Kind() == k {
					return true
				}
			}
			return false
		}
	}
	switch b.Kind() {
	case boxes.KindLine, boxes.KindInline:
		return []func(boxes.Box) bool{func(c boxes.Box) bool { _, ok := c.(boxes.InlineLevel); return ok }}
	case boxes.KindTable, boxes.KindInlineTable:
		return []func(boxes.Box) bool{isKind(boxes.KindTableCaption, boxes.KindTableColumnGroup,
			boxes.KindTableColumn, boxes.KindTableRowGroup, boxes.KindTableRow)}
	case boxes.KindTableColumnGroup:
		return []func(boxes.Box) bool{isKind(boxes.KindTableColumn)}
	case boxes.KindTableRowGroup:
		return []func(boxes.Box) bool{isKind(boxes.KindTableRow)}
	case boxes.KindTableRow:
		return []func(boxes.Box) bool{isKind(boxes.KindTableCell)}
	}
	if _, ok := b.(boxes.BlockContainer); ok {
		return []func(boxes.Box) bool{
			func(c boxes.Box) bool { _, ok := c.(boxes.BlockLevel); return ok },
			isKind(boxes.KindLine),
		}
	}
	return nil
}

// SanityChecks fails the test if the tree breaks the rules about what boxes
// can contain: a block container has either only block-level boxes or only
// line boxes, line boxes and inline boxes only inline-level boxes, and table
// boxes only the boxes of their level. Boxes out of the normal flow are
// allowed anywhere.
func SanityChecks(t testing.TB, box boxes.Box) {
	t.Helper()
	if ph, ok := box.(*boxes.AbsolutePlaceholder); ok {
		box = ph.Box
	}
	if _, ok := box.(boxes.Parent); !ok {
		return
	}
	children := boxes.ChildrenOf(box)
	if alternatives := properChildren(box); alternatives != nil {
		ok := false
		for _, accept := range alternatives {
			all := true
			for _, c := range children {
				if !accept(c) && c.IsInNormalFlow() {
					all = false
					break
				}
			}
			if all {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("%v has improper children: %v", box, children)
		}
	}
	for _, c := range children {
		SanityChecks(t, c)
	}
}
