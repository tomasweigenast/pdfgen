package layout

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// leaderIndex returns the path of child indexes from box to its first leader
// box, and that box. The path is nil when there is no leader.
func leaderIndex(box boxes.Box) ([]int, boxes.Box) {
	parent, ok := box.(boxes.Parent)
	if !ok {
		return nil, nil
	}
	for i, child := range parent.ParentBase().Children {
		if child.Base().IsLeader {
			return []int{i}, child
		}
		if path, leader := leaderIndex(child); path != nil {
			return append([]int{i}, path...), leader
		}
	}
	return nil, nil
}

// handleLeader fills the free space of a line with the repetitions of the
// text of its leader, if it has one, and moves what follows the leader to
// the end of the line (CSS Generated Content for Paged Media, leader()).
func handleLeader(ctx *Context, line *boxes.LineBox, containing boxes.Box) {
	path, leader := leaderIndex(line)
	if path == nil {
		return
	}
	extraWidth := 0.0
	if children := boxes.ChildrenOf(leader); len(children) != 0 {
		text := children[0]
		textWidth := text.Base().Width.V
		if textWidth <= 0 {
			return
		}

		containingWidth := containingBlockOf(containing).Width.V
		extraWidth = containingWidth
		for _, child := range line.Children {
			if child.Base().IsInNormalFlow() {
				extraWidth -= child.MarginWidth()
			}
		}
		for _, shape := range excludedShapes(ctx) {
			s := shape.Base()
			if s.PositionY+s.Height.V > line.PositionY {
				extraWidth -= s.Width.V
			}
		}

		availableWidth := extraWidth + textWidth
		line.Width = units.F(containingWidth)

		var repeated []boxes.Box
		x := line.PositionX + containingWidth
		for range int(math.Floor(containingWidth / textWidth)) {
			x -= textWidth
			if x < leader.Base().PositionX || x+textWidth > leader.Base().PositionX+availableWidth {
				// Repetitions never go behind the text around the leader.
				continue
			}
			text = text.Copy()
			text.Base().PositionX = x
			repeated = append(repeated, text)
		}
		leader.(boxes.Parent).ParentBase().Children = repeated

		if line.Style.Direction() == "rtl" {
			leader.Translate(-extraWidth, 0, false)
		}
	}

	rtl := line.Style.Direction() == "rtl"
	var box boxes.Box = line
	for _, i := range path {
		children := box.(boxes.Parent).ParentBase().Children
		for _, child := range children[i+1:] {
			if child.Base().IsInNormalFlow() {
				if rtl {
					child.Translate(-extraWidth, 0, false)
				} else {
					child.Translate(extraWidth, 0, false)
				}
			}
		}
		box = children[i]
		box.Base().Width = units.F(box.Base().Width.V + extraWidth)
	}
}
