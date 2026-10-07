package boxtest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

// Node is a box serialized for comparison: its element tag, its type name
// without the Box suffix, and its content. The content is the text of a text
// box, "<replaced>" for a replaced box, and the nodes of the children,
// preceded by the column groups for a table, for the other boxes.
type Node struct {
	Tag, Type string
	Text      string
	Children  []Node
}

// ReplacedText is the content of a replaced box.
const ReplacedText = "<replaced>"

// Parent returns the node of a box with children.
func Parent(tag, typ string, children ...Node) Node {
	return Node{Tag: tag, Type: typ, Children: children}
}

// Text returns the node of a text box.
func Text(tag, text string) Node { return Node{Tag: tag, Type: "Text", Text: text} }

// Replaced returns the node of a replaced box of the given type.
func Replaced(tag, typ string) Node { return Node{Tag: tag, Type: typ, Text: ReplacedText} }

// String formats the node as a nested tuple.
func (n Node) String() string {
	var b strings.Builder
	n.write(&b, 0)
	return b.String()
}

func (n Node) write(b *strings.Builder, depth int) {
	indent := strings.Repeat("  ", depth)
	if n.Text != "" {
		fmt.Fprintf(b, "%s(%q, %q, %q)", indent, n.Tag, n.Type, n.Text)
		return
	}
	if len(n.Children) == 0 {
		fmt.Fprintf(b, "%s(%q, %q, [])", indent, n.Tag, n.Type)
		return
	}
	fmt.Fprintf(b, "%s(%q, %q, [\n", indent, n.Tag, n.Type)
	for _, c := range n.Children {
		c.write(b, depth+1)
		b.WriteString(",\n")
	}
	fmt.Fprintf(b, "%s])", indent)
}

func (n Node) equal(o Node) bool {
	if n.Tag != o.Tag || n.Type != o.Type || n.Text != o.Text || len(n.Children) != len(o.Children) {
		return false
	}
	for i := range n.Children {
		if !n.Children[i].equal(o.Children[i]) {
			return false
		}
	}
	return true
}

func format(nodes []Node) string {
	var b strings.Builder
	b.WriteString("[\n")
	for _, n := range nodes {
		n.write(&b, 1)
		b.WriteString(",\n")
	}
	b.WriteString("]")
	return b.String()
}

// Serialize converts boxes into nodes that are easy to compare in tests.
// Absolute placeholders are serialized as the boxes they hold.
func Serialize(list []boxes.Box) []Node {
	out := make([]Node, 0, len(list))
	for _, b := range list {
		if ph, ok := b.(*boxes.AbsolutePlaceholder); ok {
			b = ph.Box
		}
		n := Node{Tag: b.Base().ElementTag, Type: b.Kind().String()}
		switch box := b.(type) {
		case *boxes.TextBox:
			n.Text = box.Text
		case boxes.Replaced:
			n.Text = ReplacedText
		case boxes.TableLike:
			table := box.TableBase()
			children := make([]boxes.Box, 0, len(table.ColumnGroups)+len(table.Children))
			for _, g := range table.ColumnGroups {
				children = append(children, g)
			}
			n.Children = Serialize(append(children, table.Children...))
		default:
			n.Children = Serialize(boxes.ChildrenOf(b))
		}
		out = append(out, n)
	}
	return out
}

// AssertTree checks that the children of the <body> block of the tree rooted
// at box, which must be the <html> block, serialize to expected.
func AssertTree(t testing.TB, box boxes.Box, expected []Node) {
	t.Helper()
	if box.Base().ElementTag != "html" || box.Kind() != boxes.KindBlock {
		t.Fatalf("root is %v, want a Block for html", box)
	}
	children := boxes.ChildrenOf(box)
	if len(children) != 1 {
		t.Fatalf("html has %d children, want 1", len(children))
	}
	body := children[0]
	if body.Base().ElementTag != "body" || body.Kind() != boxes.KindBlock {
		t.Fatalf("html child is %v, want a Block for body", body)
	}
	got := Serialize(boxes.ChildrenOf(body))
	if len(got) == len(expected) {
		equal := true
		for i := range got {
			equal = equal && got[i].equal(expected[i])
		}
		if equal {
			return
		}
	}
	t.Errorf("box tree differs\ngot:\n%s\nwant:\n%s", format(got), format(expected))
}

// TreePosition returns the indexes that lead to the first box, depth first,
// for which match is true, or nil if there is none.
func TreePosition(list []boxes.Box, match func(boxes.Box) bool) []int {
	for i, b := range list {
		if match(b) {
			return []int{i}
		}
		if pos := TreePosition(boxes.ChildrenOf(b), match); pos != nil {
			return append([]int{i}, pos...)
		}
	}
	return nil
}
