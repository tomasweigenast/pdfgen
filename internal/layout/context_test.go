package layout

import (
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestResumeCloneKeepsIndependentPaths(t *testing.T) {
	original := ResumeAt{1: {2: {7: nil}}, 4: nil}
	cloned := original.Clone()
	if !original.Equal(cloned) || (ResumeAt(nil)).Equal(ResumeAt{}) {
		t.Fatal("clone changed the continuation")
	}
	cloned[1][2][9] = nil
	delete(cloned, 4)
	if original.Equal(cloned) || len(original[1][2]) != 1 || len(original) != 2 {
		t.Fatal("continuation paths share mutable maps")
	}
}

func TestPageStateCloneSeparatesCountersScopesAndGroups(t *testing.T) {
	original := PageState{
		Build: &boxes.State{
			QuoteDepth:    2,
			CounterValues: targets.CounterValues{"page": {1, 3}},
			CounterScopes: []map[string]bool{{"page": true}},
		},
		Groups: []pageGroup{{Name: "chapter", Index: 2, ResumeAt: ResumeAt{1: {2: nil}}}},
	}
	cloned := original.Clone()
	if !original.Equal(cloned) {
		t.Fatal("clone changed page state")
	}
	cloned.Build.CounterValues["page"][1] = 7
	cloned.Build.CounterScopes[0]["chapter"] = true
	cloned.Groups[0].ResumeAt[1][8] = nil
	cloned.Groups[0].Index = 9
	if original.Equal(cloned) || original.Build.CounterValues["page"][1] != 3 ||
		original.Build.CounterScopes[0]["chapter"] || len(original.Groups[0].ResumeAt[1]) != 1 {
		t.Fatal("page states share mutable storage")
	}
}

func TestNamedStringSelection(t *testing.T) {
	ctx := &Context{StringSet: map[string]map[int][]string{
		"title": {2: {"2-first", "2-last"}, 3: {"3-first", "3-last"}, 5: {"5"}},
	}}
	page := boxes.NewPageBox(cascade.PageType{}, computed.New(nil))
	cases := []struct {
		keyword string
		want    []string
	}{
		{keyword: "first", want: []string{"", "2-first", "3-first", "3-last", "5"}},
		{keyword: "last", want: []string{"", "2-last", "3-last", "3-last", "5"}},
		{keyword: "first-except", want: []string{"", "", "", "3-last", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.keyword, func(t *testing.T) {
			for i, want := range tc.want {
				ctx.CurrentPage = i + 1
				got := ctx.StringSetFor(page, "title", tc.keyword)
				if want == "" && len(got) == 0 {
					continue
				}
				if !slices.Equal(got, []string{want}) {
					t.Fatalf("page %d: got %q, want %q", i+1, got, want)
				}
			}
		})
	}
	ctx.CurrentPage = 3
	if got := ctx.StringSetFor(page, "title", "all-once"); !slices.Equal(got, []string{"3-first", "3-last"}) {
		t.Fatalf("all-once: %q", got)
	}
	ctx.CurrentPage = 4
	if got := ctx.StringSetFor(page, "title", "all-once"); len(got) != 0 {
		t.Fatalf("all-once inherited %q", got)
	}
}

func TestNamedStringStartChecksOnlyFirstChildPath(t *testing.T) {
	ctx := &Context{CurrentPage: 2, StringSet: map[string]map[int][]string{
		"title": {1: {"previous"}, 2: {"current"}},
	}}
	style := computed.New(nil)
	assigned := style.Copy()
	assigned.Set(props.PropStringSet, props.StringSet{{Name: "title"}})
	first := boxes.NewBlockBox("div", style, nil, nil)
	second := boxes.NewBlockBox("h1", assigned, nil, nil)
	page := boxes.NewPageBox(cascade.PageType{}, style)
	page.Children = []boxes.Box{first, second}
	if got := ctx.StringSetFor(page, "title", "start"); !slices.Equal(got, []string{"previous"}) {
		t.Fatalf("start: %q", got)
	}
	first.Children = []boxes.Box{second}
	if got := ctx.StringSetFor(page, "title", "start"); !slices.Equal(got, []string{"current"}) {
		t.Fatalf("start: %q", got)
	}
}

func TestFormattingContextsRestoreExclusionsAndContainFloats(t *testing.T) {
	ctx := &Context{}
	style := computed.New(nil)
	root := boxes.NewBlockBox("div", style, nil, nil)
	root.PositionY, root.Height = 10, units.F(20)
	outerFloat := boxes.NewBlockBox("p", style, nil, nil)
	innerFloat := boxes.NewBlockBox("p", style, nil, nil)
	innerFloat.PositionY, innerFloat.Height = 15, units.F(50)
	ctx.CreateBFC(nil, []boxes.Box{outerFloat})
	ctx.CreateBFC(root, []boxes.Box{innerFloat})
	ctx.FinishBFC(root)
	if root.Height.V != 55 {
		t.Fatalf("height = %v, want 55", root.Height)
	}
	if !slices.Equal(*ctx.ExcludedShapes, []boxes.Box{outerFloat}) {
		t.Fatal("outer exclusions lost")
	}
	ctx.FinishBFC(nil)
	if ctx.ExcludedShapes != nil {
		t.Fatal("exclusions remain outside a formatting context")
	}
}

func TestFormattingContextPreservesUnchangedFractionalHeight(t *testing.T) {
	for _, withFloat := range []bool{false, true} {
		ctx := &Context{}
		style := computed.New(nil)
		root := boxes.NewBlockBox("div", style, nil, nil)
		root.PositionY, root.Height = 0.1, units.F(0.2)
		shapes := []boxes.Box{}
		if withFloat {
			shape := boxes.NewBlockBox("p", style, nil, nil)
			shape.PositionY, shape.Height = 0.1, units.F(0.1)
			shapes = append(shapes, shape)
		}
		ctx.CreateBFC(root, shapes)
		ctx.FinishBFC(root)
		if root.Height.V != 0.2 {
			t.Fatalf("float %v: height = %.17g, want 0.2", withFloat, root.Height.V)
		}
	}
}
