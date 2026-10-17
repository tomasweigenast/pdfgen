package layout

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestCollapseMargin(t *testing.T) {
	tests := []struct {
		margins []float64
		want    float64
	}{
		{nil, 0},
		{[]float64{10, 15}, 15},
		{[]float64{-10, 15}, 5},
		{[]float64{10, -15}, -5},
		{[]float64{-10, -15}, -15},
		{[]float64{0, 0}, 0},
		{[]float64{3, 7, -2, -9, 5}, -2},
	}
	for _, tt := range tests {
		if got := collapseMargin(tt.margins); got != tt.want {
			t.Errorf("collapseMargin(%v) = %g, want %g", tt.margins, got, tt.want)
		}
	}
}

func TestMarginListIsSharedByReference(t *testing.T) {
	shared := &marginList{}
	other := shared
	shared.add(10)
	other.add(-4)
	if got := shared.collapse(); got != 6 || len(other.values) != 2 {
		t.Fatalf("collapse = %g after %v", got, other.values)
	}
	var none *marginList
	if none.collapse() != 0 || !none.isEmpty() {
		t.Fatal("a missing list must act as an empty one")
	}
}

func breakStyleBoxes(t *testing.T, css string) (before, after boxes.Box) {
	t.Helper()
	list := parseBoxes(t, `<style>`+css+`</style><div id=a></div><div id=b></div>`, "div")
	return list[0], list[1]
}

func TestBlockLevelPageBreakPrecedence(t *testing.T) {
	tests := []struct {
		before, after string
		want          props.Keyword
	}{
		{"auto", "auto", "auto"},
		{"avoid", "auto", "avoid"},
		{"auto", "avoid", "avoid"},
		{"page", "avoid", "page"},
		{"avoid", "page", "page"},
		{"avoid-page", "page", "page"},
		{"left", "avoid", "left"},
		{"left", "right", "right"},
		{"right", "left", "left"},
		{"recto", "page", "recto"},
		{"page", "verso", "verso"},
		{"column", "avoid", "column"},
		{"avoid-column", "auto", "avoid-column"},
		{"page", "column", "page"},
	}
	for _, tt := range tests {
		t.Run(tt.before+"/"+tt.after, func(t *testing.T) {
			before, after := breakStyleBoxes(t, fmt.Sprintf("#a { break-after: %s } #b { break-before: %s }", tt.before, tt.after))
			if got := blockLevelPageBreak(before, after); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestBlockLevelPageBreakLooksIntoNestedEdges(t *testing.T) {
	list := parseBoxes(t, `
		<style>
			#last { break-after: page }
			#first { break-before: avoid }
		</style>
		<div id=a><div id=last></div></div>
		<div id=b><div id=first></div></div>`, "div")
	before, after := list[0], list[2]
	if got := blockLevelPageBreak(before, after); got != "page" {
		t.Fatalf("break = %s, want page", got)
	}
	if got := blockLevelPageBreak(nil, after); got != "avoid" {
		t.Fatalf("break without a sibling = %s, want avoid", got)
	}
}

func TestBlockLevelPageBreakIgnoresOutOfFlowBoxes(t *testing.T) {
	list := parseBoxes(t, `
		<style>#a { position: absolute; break-after: page }</style>
		<div id=a></div><div id=b></div>`, "div")
	if got := blockLevelPageBreak(list[0], list[1]); got != "auto" {
		t.Fatalf("break = %s, want auto", got)
	}
}

func TestBlockLevelPageNameKeepsNilWhenSiblingsAgree(t *testing.T) {
	list := parseBoxes(t, `
		<style>#b { page: chapter } #c { page: chapter }</style>
		<div id=a></div><div id=b></div><div id=c></div>`, "div")
	if name := blockLevelPageName(list[1], list[2]); name != nil {
		t.Fatalf("same page name returned %q", *name)
	}
	if name := blockLevelPageName(list[0], list[1]); name == nil || *name != "chapter" {
		t.Fatalf("page name = %v, want chapter", name)
	}
	if name := blockLevelPageName(list[2], list[0]); name == nil || *name != "" {
		t.Fatalf("leaving a named page = %v, want explicit empty name", name)
	}
}

func TestPageBreakKeywordsInsideColumns(t *testing.T) {
	page, column := &Context{}, &Context{InColumn: true}
	tests := []struct {
		value                 props.Keyword
		avoid, force          bool
		avoidInColumn, force2 bool
	}{
		{"avoid", true, false, true, false},
		{"avoid-page", true, false, true, false},
		{"avoid-column", false, false, true, false},
		{"page", false, true, false, true},
		{"column", false, false, false, true},
		{"left", false, true, false, true},
		{"auto", false, false, false, false},
	}
	for _, tt := range tests {
		if avoidPageBreak(tt.value, page) != tt.avoid || forcePageBreak(tt.value, page) != tt.force {
			t.Errorf("%s in a page: avoid/force = %v/%v", tt.value, avoidPageBreak(tt.value, page), forcePageBreak(tt.value, page))
		}
		if avoidPageBreak(tt.value, column) != tt.avoidInColumn || forcePageBreak(tt.value, column) != tt.force2 {
			t.Errorf("%s in a column: avoid/force = %v/%v", tt.value, avoidPageBreak(tt.value, column), forcePageBreak(tt.value, column))
		}
	}
}

func TestFindLastChildKinds(t *testing.T) {
	list := parseBoxes(t, `
		<style>#b { float: left } #c { position: absolute }</style>
		<div id=a></div><div id=b></div><div id=c></div>`, "div")
	if got := findLastInFlowChild(list); got != list[0] {
		t.Fatalf("last in-flow child = %v", got)
	}
	if got := findLastClassABreakPointChild(list); got != list[1] {
		t.Fatalf("last class A break child = %v", got)
	}
	if findLastInFlowChild(nil) != nil || findLastClassABreakPointChild(list[2:]) != nil {
		t.Fatal("expected no child")
	}
}

func TestRemovePlaceholdersForgetsDescendantsEffects(t *testing.T) {
	list := parseBoxes(t, `
		<style>#abs { position: absolute } #fix { position: fixed } #flt { float: left }</style>
		<div id=outer><p id=abs></p><p id=fix></p><p id=flt></p></div>`, "div")
	outer := list[0]
	paragraphs := parseBoxes(t, `
		<style>#abs { position: absolute } #fix { position: fixed } #flt { float: left }</style>
		<p id=abs></p><p id=fix></p><p id=flt></p>`, "p")
	absolute := boxes.NewAbsolutePlaceholder(paragraphs[0])
	fixed := boxes.NewAbsolutePlaceholder(paragraphs[1])
	other := boxes.NewAbsolutePlaceholder(paragraphs[0].Copy())
	float := paragraphs[2]
	parent := outer.(boxes.Parent)
	parent.ParentBase().Children = []boxes.Box{absolute, fixed, float}

	ctx := &Context{}
	ctx.CreateBFC(nil, []boxes.Box{float})
	ctx.addBrokenOutOfFlow(float, float, outer, ResumeAt{1: nil})
	positioned := &positionedBoxes{
		Absolute: []*boxes.AbsolutePlaceholder{other, absolute},
		Fixed:    []*boxes.AbsolutePlaceholder{fixed},
	}
	ctx.removePlaceholders([]boxes.Box{outer}, positioned)

	if len(positioned.Absolute) != 1 || positioned.Absolute[0] != other || len(positioned.Fixed) != 0 {
		t.Fatalf("positioned boxes after removal = %v", positioned)
	}
	if len(*ctx.ExcludedShapes) != 0 || len(ctx.brokenOutOfFlow) != 0 {
		t.Fatal("float and continuation were kept")
	}
}

func TestFindEarlierPageBreakBetweenBlocks(t *testing.T) {
	list := parseBoxes(t, `
		<style>#b { break-after: avoid }</style>
		<body><div id=a></div><div id=b></div><div id=c></div></body>`, "div")
	for i, box := range list {
		box.Base().Index = i
	}
	ctx := &Context{}
	ctx.CreateBFC(nil, nil)

	kept, resume, ok := findEarlierPageBreak(ctx, list, &positionedBoxes{})
	if !ok || len(kept) != 1 || kept[0] != list[0] || !resume.Equal(ResumeAt{1: nil}) {
		t.Fatalf("earlier break = %v, %v, %v", kept, resume, ok)
	}

	if _, _, ok := findEarlierPageBreak(ctx, list[1:], &positionedBoxes{}); ok {
		t.Fatal("found a break between blocks that cannot be separated")
	}
	if _, _, ok := findEarlierPageBreak(ctx, nil, &positionedBoxes{}); ok {
		t.Fatal("found a break in an empty list")
	}
}

func TestFindEarlierPageBreakDescendsIntoBreakableBlocks(t *testing.T) {
	list := parseBoxes(t, `
		<style>#inner1, #inner2 { height: 5px } #outer { break-inside: auto; break-after: avoid } #keep { break-inside: avoid }</style>
		<body><div id=outer><div id=inner1></div><div id=inner2></div></div>
		<div id=keep><div id=k></div><div id=k2></div></div></body>`, "div")
	outer, inner1, inner2, keep := list[0], list[1], list[2], list[3]
	for i, box := range []boxes.Box{inner1, inner2} {
		box.Base().Index = i
	}
	outer.Base().Index = 0
	keep.Base().Index = 1
	ctx := &Context{}
	ctx.CreateBFC(nil, nil)

	kept, resume, ok := findEarlierPageBreak(ctx, []boxes.Box{outer, keep}, &positionedBoxes{})
	if !ok || len(kept) != 1 || !resume.Equal(ResumeAt{0: {1: nil}}) {
		t.Fatalf("earlier break = %v, %v, %v", kept, resume, ok)
	}
	if got := boxes.ChildrenOf(kept[0]); len(got) != 1 || got[0] != inner1 {
		t.Fatalf("kept children = %v", got)
	}
	if boxes.ChildrenOf(outer)[1] != inner2 {
		t.Fatal("the original block lost a child")
	}
}

func TestFindEarlierPageBreakKeepsLinesForOrphansAndWidows(t *testing.T) {
	line := func(orphans, widows string) []boxes.Box {
		list := parseBoxes(t, `<p style="orphans: `+orphans+`; widows: `+widows+`">a`, "p")
		inner := boxes.ChildrenOf(list[0])[0]
		return []boxes.Box{inner, inner.Copy(), inner.Copy(), inner.Copy()}
	}
	ctx := &Context{}
	ctx.CreateBFC(nil, nil)
	lines := line("2", "1")
	for i, l := range lines {
		ctx.setLineResumeAt(l, ResumeAt{i: nil})
	}
	kept, resume, ok := findEarlierPageBreak(ctx, lines, &positionedBoxes{})
	if !ok || len(kept) != 3 || !resume.Equal(ResumeAt{0: {2: nil}}) {
		t.Fatalf("earlier break = %d lines, %v, %v", len(kept), resume, ok)
	}
	if _, _, ok := findEarlierPageBreak(ctx, line("3", "2"), &positionedBoxes{}); ok {
		t.Fatal("orphans and widows left no room for a break")
	}
}

func TestRelativePositioning(t *testing.T) {
	tests := []struct {
		name         string
		style        string
		wantX, wantY float64
	}{
		{"left and top", "left: 5px; top: 7px", 5, 7},
		{"right and bottom", "right: 5px; bottom: 7px", -5, -7},
		{"left wins over right in ltr", "left: 5px; right: 9px", 5, 0},
		{"right wins over left in rtl", "left: 5px; right: 9px; direction: rtl", -9, 0},
		{"top wins over bottom", "top: 3px; bottom: 8px", 0, 3},
		{"percentages", "left: 10%; top: 50%", 20, 50},
		{"static", "position: static; left: 5px", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := parseBox(t, `<div style="position: relative; `+tt.style+`">`, "div")
			box.Base().PositionX, box.Base().PositionY = 100, 200
			relativePositioning(box, containingBlock{Width: units.F(200), Height: units.F(100)})
			if x, y := box.Base().PositionX-100, box.Base().PositionY-200; x != tt.wantX || y != tt.wantY {
				t.Fatalf("moved by (%g,%g), want (%g,%g)", x, y, tt.wantX, tt.wantY)
			}
		})
	}
}
