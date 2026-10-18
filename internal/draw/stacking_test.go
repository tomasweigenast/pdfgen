package draw_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/draw"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// serializeStacking describes a context as its element, the elements of the
// blocks and cells of its tree, and the contexts of z-index 0.
func serializeStacking(c *draw.StackingContext) string {
	var b strings.Builder
	b.WriteString(c.Box.Base().ElementTag)
	b.WriteString(" [")
	b.WriteString(strings.Join(flattenBlocks(c.BlocksAndCells), " "))
	b.WriteString("] {")
	for i, child := range c.ZeroZ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(serializeStacking(child))
	}
	b.WriteString("}")
	return b.String()
}

func flattenBlocks(entries []draw.BlockEntry) []string {
	var tags []string
	for _, e := range entries {
		tags = append(tags, e.Box.Base().ElementTag)
		tags = append(tags, flattenBlocks(e.Children)...)
	}
	return tags
}

func TestStackingContextTree(t *testing.T) {
	tests := []struct {
		name, source, want string
	}{
		{
			"positioned box nests a context",
			`<p id=lorem></p>
			 <div style="position: relative">
			   <p id=lipsum></p>
			 </div>`,
			"html [body p] {div [p] {}}",
		},
		{
			"positioned child of a positioned box is a sibling context",
			`<div style="position: relative">
			   <p style="position: relative"></p>
			 </div>`,
			"html [body] {div [] {}, p [] {}}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			page := boxtest.RenderPages(t, tt.source)[0]
			root := page.Children[0]
			if got := serializeStacking(draw.FromBox(root, page)); got != tt.want {
				t.Errorf("stacking context = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestImageStackingContexts(t *testing.T) {
	testutil.AssertNoLogs(t)
	page := boxtest.RenderPages(t, `<body>Some text: <img style="position: relative" src=pattern.png>`)[0]
	context := draw.FromBox(page.Children[0], page)

	// The image is not in this context.
	got := boxtest.Serialize([]boxes.Box{context.Box})
	want := []boxtest.Node{
		boxtest.Parent("html", "Block",
			boxtest.Parent("body", "Block",
				boxtest.Parent("body", "Line",
					boxtest.Text("body", "Some text: ")))),
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("context box = %v, want %v", got, want)
	}

	// It is in a context of its own.
	var children []boxes.Box
	for _, c := range context.ZeroZ {
		children = append(children, c.Box)
	}
	got = boxtest.Serialize(children)
	want = []boxtest.Node{boxtest.Replaced("img", "InlineReplaced")}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("sub-contexts = %v, want %v", got, want)
	}
}
