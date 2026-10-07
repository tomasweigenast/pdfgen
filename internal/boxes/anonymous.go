package boxes

import "strings"

// CreateAnonymousBoxes adds the anonymous boxes required by the table, flex,
// grid and inline formatting models to box and its descendants, and returns
// the box that replaces it.
func CreateAnonymousBoxes(box Box) Box {
	box = anonymousTableBoxes(box)
	box = flexBoxes(box)
	box = gridBoxes(box)
	box = InlineInBlock(box)
	box = BlockInInline(box)
	return box
}

// Blockify returns the block-level equivalent of an inline-level box, or
// the box itself (CSS Display 3 §2.7). When layout is "flex" or "grid", the
// result is marked as an item of that layout.
func Blockify(box Box, layout string) Box {
	var block Box
	switch b := box.(type) {
	case *InlineBlockBox:
		block = AnonymousFrom(KindBlock, b, b.Children)
		block.Base().IsTableWrapper = b.IsTableWrapper
	case *InlineReplacedBox:
		block = AnonymousBlockReplaced(b, b.Replacement)
	case *TextBox:
		block = AnonymousFrom(KindBlock, b, []Box{b})
		setItem(b, layout, false)
	case *InlineFlexBox:
		block = AnonymousFrom(KindFlex, b, b.Children)
	case *InlineGridBox:
		block = AnonymousFrom(KindGrid, b, b.Children)
	case InlineLevel:
		block = AnonymousFrom(KindBlock, b, ChildrenOf(b))
	default:
		return box
	}
	block.Base().Style = box.Base().Style
	setItem(block, layout, true)
	return block
}

func setItem(box Box, layout string, item bool) {
	switch layout {
	case "flex":
		box.Base().IsFlexItem = item
	case "grid":
		box.Base().IsGridItem = item
	}
}

// isCollapsibleSpace reports whether a text box only has spaces, which
// makes no flex or grid item (CSS Flexbox 1 §4, CSS Grid 2 §6).
func isCollapsibleSpace(b Box) bool {
	t, ok := b.(*TextBox)
	return ok && strings.Trim(t.Text, " ") == ""
}

// flexBoxes blockifies the children of flex containers in box and its
// descendants, which become flex items (CSS Flexbox 1 §4).
func flexBoxes(box Box) Box {
	p, ok := box.(Parent)
	if !ok || box.IsRunning() {
		return box
	}
	old := p.ParentBase().Children
	children := make([]Box, len(old))
	for i, child := range old {
		children[i] = flexBoxes(child)
	}
	p.ParentBase().Children = children
	if _, ok := box.(FlexContainer); !ok {
		return box
	}
	items := make([]Box, 0, len(children))
	for _, child := range children {
		// Flex items are not floated (CSS Flexbox 1 §3).
		child.Base().NotFloated = true
		if child.IsInNormalFlow() {
			child.Base().IsFlexItem = true
		}
		if isCollapsibleSpace(child) {
			continue
		}
		items = append(items, Blockify(child, "flex"))
	}
	p.ParentBase().Children = items
	return box
}

// gridBoxes blockifies the children of grid containers in box and its
// descendants, which become grid items (CSS Grid 2 §6).
func gridBoxes(box Box) Box {
	p, ok := box.(Parent)
	if !ok || box.IsRunning() {
		return box
	}
	old := p.ParentBase().Children
	children := make([]Box, len(old))
	for i, child := range old {
		children[i] = gridBoxes(child)
	}
	p.ParentBase().Children = children
	if _, ok := box.(GridContainer); !ok {
		return box
	}
	items := make([]Box, 0, len(children))
	for _, child := range children {
		if child.IsInNormalFlow() {
			child.Base().IsGridItem = true
		}
		if isCollapsibleSpace(child) {
			continue
		}
		items = append(items, Blockify(child, "grid"))
	}
	p.ParentBase().Children = items
	return box
}

// InlineInBlock wraps the consecutive inline-level children of block
// containers in line boxes, in box and its descendants (CSS 2.1 §9.2.1.1).
// A line box with block-level siblings is wrapped in an anonymous block.
// Text boxes left empty by white space processing are removed; the spaces
// they collapsed are kept as flags for the line breaking. Absolutely
// positioned boxes stay in the lines, for their static position.
//
// While the tree is built, a line box holds all the inline content of its
// block; layout splits it into lines.
func InlineInBlock(box Box) Box {
	children := ChildrenOf(box)
	if len(children) == 0 || box.IsRunning() || box.IsNote() {
		return box
	}
	base := box.Base()
	if !base.LeadingCollapsibleSpace {
		base.LeadingCollapsibleSpace = children[0].Base().LeadingCollapsibleSpace
	}

	kept := make([]Box, 0, len(children))
	trailing := false
	for _, child := range children {
		if trailing {
			child.Base().LeadingCollapsibleSpace = true
		}
		if t, ok := child.(*TextBox); ok && t.Text == "" {
			trailing = t.LeadingCollapsibleSpace
			continue
		}
		trailing = false
		kept = append(kept, InlineInBlock(child))
	}
	if !base.TrailingCollapsibleSpace {
		base.TrailingCollapsibleSpace = trailing
	}

	if _, ok := box.(BlockContainer); !ok {
		setChildren(box, kept)
		return box
	}

	var line, blocks []Box
	for _, child := range kept {
		_, inline := child.(InlineLevel)
		switch {
		case len(line) > 0 && child.IsAbsolutelyPositioned():
			line = append(line, child)
		case inline || (len(line) > 0 && !child.IsInNormalFlow()):
			// White space at the start of a line would be removed by
			// layout.
			if len(line) > 0 || !isCollapsedSpace(child) {
				line = append(line, child)
			}
		default:
			if len(line) > 0 {
				blocks = append(blocks, AnonymousFrom(KindBlock, box, []Box{AnonymousFrom(KindLine, box, line)}))
				line = nil
			}
			blocks = append(blocks, child)
		}
	}
	if len(line) > 0 {
		lineBox := AnonymousFrom(KindLine, box, line)
		if len(blocks) > 0 {
			blocks = append(blocks, AnonymousFrom(KindBlock, box, []Box{lineBox}))
		} else {
			blocks = append(blocks, lineBox)
		}
	}
	setChildren(box, blocks)
	return box
}

// isCollapsedSpace reports whether b is a text box with the single space
// left by the collapsing of white space.
func isCollapsedSpace(b Box) bool {
	t, ok := b.(*TextBox)
	if !ok || t.Text != " " {
		return false
	}
	switch t.Style.WhiteSpace() {
	case "normal", "nowrap", "pre-line":
		return true
	}
	return false
}

func setChildren(box Box, children []Box) {
	if p, ok := box.(Parent); ok {
		p.ParentBase().Children = children
	}
}

// skipStack is the position after a block-level box found inside nested
// inline boxes: the index in the line, then in each inline box.
type skipStack struct {
	index int
	next  *skipStack
}

// BlockInInline splits the inline boxes that contain block-level boxes, in
// box and its descendants (CSS 2.1 §9.2.1.1). The parts of a line before,
// between and after its block-level boxes are wrapped in anonymous blocks,
// siblings of the block-level boxes. InlineInBlock must have made the lines.
func BlockInInline(box Box) Box {
	children := ChildrenOf(box)
	if len(children) == 0 || box.IsRunning() || box.IsNote() {
		return box
	}
	var out []Box
	changed := false
	for _, child := range children {
		var replacement Box
		if line, ok := child.(*LineBox); ok {
			// A line box has no siblings at this stage.
			var stack *skipStack
			var newLine Box
			for {
				var block Box
				newLine, block, stack = innerBlockInInline(line, stack)
				if block == nil {
					break
				}
				out = append(out, AnonymousFrom(KindBlock, box, []Box{newLine}), BlockInInline(block))
			}
			if len(out) > 0 {
				// The line became a block formatting context.
				replacement = AnonymousFrom(KindBlock, box, []Box{newLine})
			} else {
				replacement = newLine
			}
		} else {
			replacement = BlockInInline(child)
		}
		if replacement != child {
			changed = true
		}
		out = append(out, replacement)
	}
	if changed {
		setChildren(box, out)
	}
	return box
}

// innerBlockInInline finds the first block-level box in flow inside an
// inline formatting context, from the position skip. It returns a copy of
// box with the content before the block, the block, and the position after
// it. When there is no block, it returns the box with the content after
// skip, a nil block and a nil position.
func innerBlockInInline(box Box, skip *skipStack) (Box, Box, *skipStack) {
	var newChildren []Box
	var block Box
	var resume *skipStack
	changed := false

	start := 0
	if skip != nil {
		start, skip = skip.index, skip.next
	}
	for i, child := range ChildrenOf(box)[start:] {
		index := start + i
		if _, ok := child.(BlockLevel); ok && child.IsInNormalFlow() {
			block = child
			// Resume after the block.
			index++
		} else {
			var newChild Box
			if child.Kind() == KindInline {
				newChild, block, resume = innerBlockInInline(child, skip)
				skip = nil
			} else {
				newChild = BlockInInline(child)
			}
			if newChild != child {
				changed = true
			}
			newChildren = append(newChildren, newChild)
		}
		if block != nil {
			return box.(Parent).CopyWithChildren(newChildren), block, &skipStack{index, resume}
		}
	}
	if changed || start != 0 {
		box = box.(Parent).CopyWithChildren(newChildren)
	}
	return box, nil, nil
}
