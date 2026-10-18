package anchors

import "github.com/tomasweigenast/pdfgen/internal/matrix"

// Node is an entry of the outline tree. Page is the zero-based index of the
// page that holds the position (X, Y).
type Node struct {
	Label    string
	Page     int
	X, Y     float64
	Children []Node
	State    string
}

// TreeBuilder assembles the outline tree from the bookmarks of consecutive
// pages. Bookmark levels may skip values, as an h4 right after an h1 does:
// the tree has no empty nodes, the skipped levels are remembered so that a
// later heading of a shallower level finds its parent.
type TreeBuilder struct {
	root []Node
	// skipped records, for each depth of the output, how many source levels
	// were skipped to reach it.
	skipped []int
	// lastByDepth[i] is the list that receives the children of the last
	// node at depth i.
	lastByDepth []*[]Node
	previous    int
}

// NewTreeBuilder returns an empty builder.
func NewTreeBuilder() *TreeBuilder {
	b := &TreeBuilder{}
	b.lastByDepth = []*[]Node{&b.root}
	return b
}

// AddPage appends the bookmarks of the page with the given index, mapping
// their positions with m.
func (b *TreeBuilder) AddPage(bookmarks []Bookmark, page int, m matrix.Matrix) {
	for _, bm := range bookmarks {
		if bm.Level > b.previous {
			b.skipped = append(b.skipped, bm.Level-b.previous-1)
		} else {
			level := bm.Level
			for level < b.previous && len(b.skipped) > 0 {
				last := len(b.skipped) - 1
				level += 1 + b.skipped[last]
				b.skipped = b.skipped[:last]
			}
			if level > b.previous {
				// Too many skips were removed: add some back.
				b.skipped = append(b.skipped, level-b.previous-1)
			}
		}
		b.previous = bm.Level

		depth := bm.Level
		for _, s := range b.skipped {
			depth -= s
		}
		depth = max(depth, 1)
		if depth > len(b.lastByDepth) {
			depth = len(b.lastByDepth)
		}

		x, y := m.TransformPoint(bm.X, bm.Y)
		siblings := b.lastByDepth[depth-1]
		*siblings = append(*siblings, Node{Label: bm.Label, Page: page, X: x, Y: y, State: bm.State})
		b.lastByDepth = b.lastByDepth[:depth]
		children := &(*siblings)[len(*siblings)-1].Children
		b.lastByDepth = append(b.lastByDepth, children)
	}
}

// Tree returns the nodes of the first level.
func (b *TreeBuilder) Tree() []Node { return b.root }
