package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestExcludedShapesBelongToTheCurrentFormattingContext(t *testing.T) {
	outer := parseBox(t, `<div style="float: left">`, "div")
	inner := outer.Copy()
	ctx := &Context{}
	ctx.CreateBFC(nil, nil)
	ctx.addExcludedShape(outer)
	ctx.CreateBFC(inner, nil)
	if len(*ctx.ExcludedShapes) != 0 {
		t.Fatal("a new formatting context inherited the floats of its parent")
	}
	ctx.addExcludedShape(inner)
	ctx.removeExcludedShape(inner)
	ctx.FinishBFC(nil)
	if shapes := *ctx.ExcludedShapes; len(shapes) != 1 || shapes[0] != outer {
		t.Fatalf("parent shapes after the nested context = %v", shapes)
	}
	ctx.removeExcludedShape(inner)
	if len(*ctx.ExcludedShapes) != 1 {
		t.Fatal("removing an unknown shape changed the list")
	}
}

func TestExcludedShapesAreIgnoredWithoutFormattingContext(t *testing.T) {
	ctx := &Context{}
	ctx.addExcludedShape(parseBox(t, `<div style="float: left">`, "div"))
	if ctx.Err() == nil {
		t.Fatal("a float outside of any formatting context was accepted")
	}
}

func TestBrokenOutOfFlowKeepsInsertionOrderAndReplacesByFragment(t *testing.T) {
	boxesByName := parseBoxes(t, `<div></div><div></div><div></div>`, "div")
	first, second, third := boxesByName[0], boxesByName[1], boxesByName[2]
	root := boxesByName[0].Copy()
	ctx := &Context{}
	ctx.CreateBFC(root, nil)
	resume := ResumeAt{1: nil}
	ctx.addBrokenOutOfFlow(first, third, second, resume)
	ctx.addBrokenOutOfFlow(second, third, second, ResumeAt{2: nil})
	ctx.addBrokenOutOfFlow(first, second, third, ResumeAt{3: nil})
	resume[1] = ResumeAt{9: nil}

	if len(ctx.brokenOutOfFlow) != 2 {
		t.Fatalf("records = %d, want 2", len(ctx.brokenOutOfFlow))
	}
	got := ctx.brokenOutOfFlow[0]
	if got.Fragment != first || got.Original != second || got.ContainingBlock != third ||
		got.BFCRoot != root || !got.ResumeAt.Equal(ResumeAt{3: nil}) {
		t.Fatalf("replaced record = %+v", got)
	}
	if ctx.brokenOutOfFlow[1].Fragment != second {
		t.Fatal("insertion order changed")
	}

	ctx.removeBrokenOutOfFlow(first)
	if len(ctx.brokenOutOfFlow) != 1 || ctx.brokenOutOfFlow[0].Fragment != second {
		t.Fatal("record was not removed by fragment")
	}
}

func TestBrokenOutOfFlowCopiesTheContinuation(t *testing.T) {
	div := parseBox(t, `<div></div>`, "div")
	ctx := &Context{}
	ctx.CreateBFC(nil, nil)
	resume := ResumeAt{1: {2: nil}}
	ctx.addBrokenOutOfFlow(div, div, div, resume)
	resume[1][2] = ResumeAt{5: nil}
	if !ctx.brokenOutOfFlow[0].ResumeAt.Equal(ResumeAt{1: {2: nil}}) {
		t.Fatal("record shares the continuation with its caller")
	}
}

func TestRunningElementsAreStoredPerNameAndPage(t *testing.T) {
	source := `<style>div { position: running(header) }</style><div></div><div></div>`
	list := parseBoxes(t, source, "div")
	ctx := &Context{CurrentPage: 2}
	ctx.addRunningElement(list[0])
	ctx.addRunningElement(list[1])
	if got := ctx.RunningElements["header"][2]; len(got) != 2 || got[0] != list[0] || got[1] != list[1] {
		t.Fatalf("running elements = %v", got)
	}
	ctx.removeRunningElement(list[0])
	ctx.removeRunningElement(list[0])
	ctx.CurrentPage = 3
	ctx.removeRunningElement(list[1])
	if got := ctx.RunningElements["header"][2]; len(got) != 1 || got[0] != list[1] {
		t.Fatalf("running elements after removal = %v", got)
	}
}

func TestRenderedNotesAreNotRunningElements(t *testing.T) {
	source := `<style>div { position: note(footnotes) }</style><div></div>`
	note := parseBox(t, source, "div")
	ctx := &Context{CurrentPage: 1, RenderedNotes: []boxes.Box{note}}
	ctx.addRunningElement(note)
	if len(ctx.RunningElements["footnotes"][1]) != 0 {
		t.Fatal("a note laid out on a previous page was added again")
	}
}
