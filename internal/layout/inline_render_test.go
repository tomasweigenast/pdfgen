package layout

import (
	"context"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/units"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// lineDoc is a laid-out document: the root box of its only page.
type lineDoc struct {
	html boxes.Box
}

// renderPage lays out a document that fits on one page and fails the test if
// anything is logged.
func renderPage(t *testing.T, source string) *lineDoc {
	t.Helper()
	noLogs(t)
	page := onlyPage(t, paginate(t, source))
	html, _ := pageBody(t, page)
	return &lineDoc{html: html}
}

// body returns the box of the body element.
func (d *lineDoc) body(t *testing.T) boxes.Box {
	t.Helper()
	return kids(t, d.html, 1)[0]
}

// lineFixture is a block container with its line box, for the tests that
// call the line layout directly.
type lineFixture struct {
	ctx   *Context
	block boxes.Box
	line  *boxes.LineBox
}

// newLineFixture builds a document and returns the first block box made by
// an element called tag, which is width px wide and has a line box.
func newLineFixture(t *testing.T, source, tag string, width float64) *lineFixture {
	t.Helper()
	noLogs(t)
	doc := csstest.Parse(t, source)
	env := csstest.Environment()
	collector := targets.NewCollector()
	styles := csstest.StyleFor(t, doc, cascade.Options{Environment: env, Targets: collector})
	builder := &boxes.Builder{
		StyleFor:     styles,
		Targets:      collector,
		CounterStyle: env.CounterStyle,
		GetImage:     images.NewGetter(context.Background(), urls.NewDefaultFetcher(), nil, images.Options{}),
	}
	root, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	var block boxes.Box
	for b := range boxes.Descendants(root, false) {
		if b.Base().ElementTag == tag && b.Kind() == boxes.KindBlock {
			block = b
			break
		}
	}
	if block == nil {
		t.Fatalf("no %s block in %q", tag, source)
	}
	children := boxes.ChildrenOf(block)
	if len(children) == 0 {
		t.Fatalf("%v has no children", block)
	}
	line, ok := children[0].(*boxes.LineBox)
	if !ok {
		t.Fatalf("%v has no line box: %v", block, children)
	}
	ctx := NewContext(builder)
	ctx.CreateBFC(nil, nil)
	block.Base().Width, block.Base().Height = units.F(width), units.Auto
	return &lineFixture{ctx: ctx, block: block, line: line}
}

// layout lays out the lines of the block from positionY, the number of
// them that limit asks for or all when it is not positive.
func (f *lineFixture) layout(t *testing.T, opts blockOptions, positionY float64, limit int) []lineResult {
	t.Helper()
	opts.BottomSpace = math.Inf(-1)
	if opts.Positioned == nil {
		opts.Positioned = &positionedBoxes{}
	}
	var lines []lineResult
	for result := range iterLineBoxes(f.ctx, f.line, f.block, opts, positionY) {
		lines = append(lines, result)
		if len(lines) == limit {
			break
		}
	}
	if err := f.ctx.Err(); err != nil {
		t.Fatalf("layout: %v", err)
	}
	return lines
}
