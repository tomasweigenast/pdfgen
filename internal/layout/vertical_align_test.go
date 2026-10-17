package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestTranslateSubtree(t *testing.T) {
	paragraph := parseBox(t, `
      <p><span style="vertical-align: top"><b>a</b></span><span><b>b</b></span>`, "p")
	line := boxes.ChildrenOf(paragraph)[0]
	spans := boxes.ChildrenOf(line)
	top, baseline := spans[0], spans[1]
	topChild, baselineChild := boxes.ChildrenOf(top)[0], boxes.ChildrenOf(baseline)[0]
	for _, box := range []boxes.Box{top, topChild, baseline, baselineChild} {
		box.Base().PositionY = 10
	}

	translateSubtree(top, 5)
	translateSubtree(baseline, 5)

	// The children of a box aligned to the top move with it, as they are
	// placed relative to the line; the others were placed relative to their
	// parent, which moves them already.
	for name, want := range map[string]struct {
		box boxes.Box
		y   float64
	}{
		"top span":       {top, 15},
		"top child":      {topChild, 15},
		"baseline span":  {baseline, 15},
		"baseline child": {baselineChild, 10},
	} {
		if got := want.box.Base().PositionY; got != want.y {
			t.Errorf("%s y = %g, want %g", name, got, want.y)
		}
	}
}

func TestBoxWithoutBaselineAlignsAtTop(t *testing.T) {
	box := parseBox(t, `<p>a`, "p")
	if got := baselineOf(box); got != 0 {
		t.Fatalf("baseline of a box without one = %g, want 0", got)
	}
	baseline := 7.0
	box.Base().Baseline = &baseline
	if got := baselineOf(box); got != 7 {
		t.Fatalf("baseline = %g, want 7", got)
	}
}

func TestExtentCoversTheMarginBoxesIncluded(t *testing.T) {
	var e extent
	if e.set {
		t.Fatal("the zero extent is not empty")
	}
	e.merge(extent{})
	if e.set {
		t.Fatal("merging an empty extent changed the extent")
	}
	e.include(10, 20)
	e.include(-5, 15)
	e.merge(extent{min: 12, max: 40, set: true})
	if e.min != -5 || e.max != 40 {
		t.Fatalf("extent = [%g, %g], want [-5, 40]", e.min, e.max)
	}
}
