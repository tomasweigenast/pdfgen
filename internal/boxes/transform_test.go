package boxes_test

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// matrixOf lays out a 20x10 block at the top left of a page with the given
// transform declarations, and returns the matrix set on it.
func matrixOf(t *testing.T, declarations string) *matrix.Matrix {
	t.Helper()
	pages := boxtest.RenderPages(t, `
		<style>
		  @page { size: 100px; margin: 0 }
		  body { margin: 0 }
		  div { width: 20px; height: 10px; `+declarations+` }
		</style>
		<div></div>`)
	boxes.SetTransformationMatrices(pages[0])
	for b := range boxes.Descendants(pages[0], true) {
		if b.Base().ElementTag == "div" {
			return b.Base().TransformationMatrix
		}
	}
	t.Fatal("no div in the page")
	return nil
}

func TestTransformationMatrices(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want [6]float64
	}{
		{"translate", "transform: translate(5px, 6px)", [6]float64{1, 0, 0, 1, 5, 6}},
		{"translate percentages", "transform: translate(50%, 10%)", [6]float64{1, 0, 0, 1, 10, 1}},
		{"scale around the center", "transform: scale(2, 3)", [6]float64{2, 0, 0, 3, -10, -10}},
		{"rotate around the center", "transform: rotate(90deg)", [6]float64{0, 1, -1, 0, 15, -5}},
		{"skew around the center", "transform: skew(45deg)", [6]float64{1, 0, 1, 1, -5, 0}},
		{"skew both axes", "transform: skew(45deg, 45deg); transform-origin: 0 0", [6]float64{1, 1, 1, 1, 0, 0}},
		{"matrix at the origin", "transform: matrix(1, 2, 3, 4, 5, 6); transform-origin: 0 0", [6]float64{1, 2, 3, 4, 5, 6}},
		{"last function applies first", "transform: translate(5px, 0) scale(2); transform-origin: 0 0", [6]float64{2, 0, 0, 2, 5, 0}},
		{"origin in pixels", "transform: scale(2); transform-origin: 4px 2px", [6]float64{2, 0, 0, 2, -4, -2}},
		{"origin in percentages", "transform: scale(2); transform-origin: 100% 100%", [6]float64{2, 0, 0, 2, -20, -10}},
		{"calc in translate", "transform: translate(calc(50% + 1px), 0)", [6]float64{1, 0, 0, 1, 11, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := matrixOf(t, tt.css)
			if m == nil {
				t.Fatal("no matrix set")
			}
			a, b, c, d, e, f := m.Values()
			got := [6]float64{a, b, c, d, e, f}
			for i := range got {
				if math.Abs(got[i]-tt.want[i]) > 1e-9 {
					t.Fatalf("matrix = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestNoMatrixWithoutTransform(t *testing.T) {
	if m := matrixOf(t, "opacity: .5"); m != nil {
		t.Fatalf("matrix = %v, want none", m)
	}
}

func TestInlineBoxesAreNotTransformed(t *testing.T) {
	pages := boxtest.RenderPages(t, `
		<style>@page { size: 100px } span { transform: scale(2) }</style>
		<p><span>text</span> <b style="display: inline-block; transform: scale(2)">x</b>`)
	boxes.SetTransformationMatrices(pages[0])
	var inline, atomic int
	for b := range boxes.Descendants(pages[0], true) {
		if b.Base().TransformationMatrix == nil {
			continue
		}
		switch b.(type) {
		case *boxes.InlineBox:
			inline++
		case *boxes.InlineBlockBox:
			atomic++
		}
	}
	if inline != 0 || atomic != 1 {
		t.Fatalf("matrices on %d inline and %d inline-block boxes, want 0 and 1", inline, atomic)
	}
}

func TestTransformationMatricesAreIdempotent(t *testing.T) {
	pages := boxtest.RenderPages(t, `<style>div { transform: rotate(10deg) }</style><div>x</div>`)
	first := func() (out [6]float64) {
		boxes.SetTransformationMatrices(pages[0])
		for b := range boxes.Descendants(pages[0], true) {
			if m := b.Base().TransformationMatrix; m != nil {
				a, b, c, d, e, f := m.Values()
				return [6]float64{a, b, c, d, e, f}
			}
		}
		t.Fatal("no matrix")
		return
	}
	if a, b := first(), first(); a != b {
		t.Fatalf("second call gave %v, first gave %v", b, a)
	}
}
