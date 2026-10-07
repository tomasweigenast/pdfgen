package boxtest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

// recorder collects the failures of an assertion instead of failing the test.
type recorder struct {
	testing.TB
	msgs []string
}

type stopped struct{}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	panic(stopped{})
}

func (r *recorder) run(f func(testing.TB)) {
	defer func() {
		if p := recover(); p != nil {
			if _, ok := p.(stopped); !ok {
				panic(p)
			}
		}
	}()
	f(r)
}

var style = computed.New(nil)

// document returns html > body > children.
func document(children ...boxes.Box) *boxes.BlockBox {
	body := boxes.NewBlockBox("body", style, nil, children)
	return boxes.NewBlockBox("html", style, nil, []boxes.Box{body})
}

func TestSerializeFollowsTheShapeOfTheTree(t *testing.T) {
	text := boxes.NewTextBox("p", style, nil, "hello")
	line := boxes.NewLineBox("p", style, nil, []boxes.Box{text})
	img := boxes.NewInlineReplacedBox("img", style, nil, nil)
	line2 := boxes.NewLineBox("div", style, nil, []boxes.Box{img})
	p := boxes.NewBlockBox("p", style, nil, []boxes.Box{line})
	div := boxes.NewBlockBox("div", style, nil, []boxes.Box{line2})
	br := boxes.NewBlockReplacedBox("img", style, nil, nil)

	got := boxtest.Serialize([]boxes.Box{p, div, br})
	want := []boxtest.Node{
		boxtest.Parent("p", "Block", boxtest.Parent("p", "Line", boxtest.Text("p", "hello"))),
		boxtest.Parent("div", "Block", boxtest.Parent("div", "Line", boxtest.Replaced("img", "InlineReplaced"))),
		boxtest.Replaced("img", "BlockReplaced"),
	}
	r := &recorder{TB: t}
	r.run(func(tb testing.TB) { boxtest.AssertTree(tb, document(p, div, br), want) })
	if len(r.msgs) != 0 {
		t.Errorf("AssertTree failed: %v", r.msgs)
	}
	if len(got) != 3 || got[2].Text != boxtest.ReplacedText {
		t.Errorf("Serialize = %v", got)
	}
}

func TestSerializeListsColumnGroupsBeforeRows(t *testing.T) {
	row := boxes.NewTableRowBox("tr", style, nil, nil)
	group := boxes.NewTableColumnGroupBox("colgroup", style, nil, []boxes.Box{boxes.NewTableColumnBox("col", style, nil, nil)})
	table := boxes.NewTableBox("table", style, nil, []boxes.Box{row})
	table.ColumnGroups = []*boxes.TableColumnGroupBox{group}
	wrapper := boxes.NewBlockBox("table", style, nil, []boxes.Box{table})

	want := []boxtest.Node{boxtest.Parent("table", "Block", boxtest.Parent("table", "Table",
		boxtest.Parent("colgroup", "TableColumnGroup", boxtest.Parent("col", "TableColumn")),
		boxtest.Parent("tr", "TableRow"),
	))}
	r := &recorder{TB: t}
	r.run(func(tb testing.TB) { boxtest.AssertTree(tb, document(wrapper), want) })
	if len(r.msgs) != 0 {
		t.Errorf("AssertTree failed: %v", r.msgs)
	}
}

func TestSerializeSeesThroughPlaceholders(t *testing.T) {
	abs := boxes.NewBlockBox("p", style, nil, nil)
	got := boxtest.Serialize([]boxes.Box{boxes.NewAbsolutePlaceholder(abs)})
	if len(got) != 1 || got[0].Type != "Block" || got[0].Tag != "p" {
		t.Errorf("Serialize = %v", got)
	}
}

func TestAssertTreeReportsDifferences(t *testing.T) {
	tree := document(boxes.NewBlockBox("p", style, nil, nil))
	r := &recorder{TB: t}
	r.run(func(tb testing.TB) {
		boxtest.AssertTree(tb, tree, []boxtest.Node{boxtest.Parent("p", "Inline")})
	})
	if len(r.msgs) != 1 || !strings.Contains(r.msgs[0], `("p", "Block", [])`) || !strings.Contains(r.msgs[0], `("p", "Inline", [])`) {
		t.Errorf("messages: %q", r.msgs)
	}
}

func TestAssertTreeChecksTheRoot(t *testing.T) {
	tests := []struct {
		name string
		root boxes.Box
	}{
		{"not html", boxes.NewBlockBox("div", style, nil, nil)},
		{"not a block", boxes.NewInlineBox("html", style, nil, nil)},
		{"no body", boxes.NewBlockBox("html", style, nil, nil)},
		{"body is not a block", boxes.NewBlockBox("html", style, nil, []boxes.Box{boxes.NewInlineBox("body", style, nil, nil)})},
	}
	for _, tt := range tests {
		r := &recorder{TB: t}
		r.run(func(tb testing.TB) { boxtest.AssertTree(tb, tt.root, nil) })
		if len(r.msgs) != 1 {
			t.Errorf("%s: messages %q", tt.name, r.msgs)
		}
	}
}

func TestTreePosition(t *testing.T) {
	a := boxes.NewTextBox("a", style, nil, "x")
	b := boxes.NewTextBox("b", style, nil, "x")
	c := boxes.NewTextBox("c", style, nil, "x")
	inline := boxes.NewInlineBox("em", style, nil, []boxes.Box{b, c})
	root := []boxes.Box{a, inline}
	match := func(tag string) func(boxes.Box) bool {
		return func(x boxes.Box) bool { return x.Base().ElementTag == tag }
	}
	if got := boxtest.TreePosition(root, match("c")); len(got) != 2 || got[0] != 1 || got[1] != 1 {
		t.Errorf("position of c = %v", got)
	}
	if got := boxtest.TreePosition(root, match("a")); len(got) != 1 || got[0] != 0 {
		t.Errorf("position of a = %v", got)
	}
	if got := boxtest.TreePosition(root, match("zzz")); got != nil {
		t.Errorf("position of a missing box = %v", got)
	}
}

// absolute returns a block box with position: absolute.
func absolute(t *testing.T) boxes.Box {
	t.Helper()
	doc := csstest.Parse(t, "<style>p{position:absolute}</style><p></p>")
	sf := csstest.StyleFor(t, doc, cascade.Options{})
	el := csstest.Element(t, doc, "p")
	return boxes.NewBlockBox("p", sf.Get(el, ""), el, nil)
}

func TestSanityChecks(t *testing.T) {
	text := func() boxes.Box { return boxes.NewTextBox("p", style, nil, "x") }
	line := func(children ...boxes.Box) boxes.Box { return boxes.NewLineBox("p", style, nil, children) }
	block := func(children ...boxes.Box) boxes.Box { return boxes.NewBlockBox("p", style, nil, children) }
	tests := []struct {
		name string
		box  boxes.Box
		ok   bool
	}{
		{"blocks in a block", block(block(), block()), true},
		{"lines in a block", block(line(text())), true},
		{"text in a block", block(text()), false},
		{"mixed blocks and lines", block(block(), line(text())), false},
		{"block in a line", block(line(block())), false},
		{"inline box in a line", block(line(boxes.NewInlineBox("em", style, nil, []boxes.Box{text()}))), true},
		{"row in a row group", boxes.NewTableRowGroupBox("t", style, nil, []boxes.Box{boxes.NewTableRowBox("tr", style, nil, nil)}), true},
		{"cell in a row group", boxes.NewTableRowGroupBox("t", style, nil, []boxes.Box{boxes.NewTableCellBox("td", style, nil, nil)}), false},
		{"row group in a table", boxes.NewTableBox("t", style, nil, []boxes.Box{boxes.NewTableRowGroupBox("t", style, nil, nil)}), true},
		{"block in a table", boxes.NewTableBox("t", style, nil, []boxes.Box{block()}), false},
		{"flex containers are not checked", boxes.NewFlexBox("f", style, nil, []boxes.Box{text()}), true},
		{"placeholders", block(boxes.NewAbsolutePlaceholder(absolute(t)), block()), true},
	}
	for _, tt := range tests {
		r := &recorder{TB: t}
		r.run(func(tb testing.TB) { boxtest.SanityChecks(tb, tt.box) })
		if ok := len(r.msgs) == 0; ok != tt.ok {
			t.Errorf("%s: passed = %v, want %v (%q)", tt.name, ok, tt.ok, r.msgs)
		}
	}
}
