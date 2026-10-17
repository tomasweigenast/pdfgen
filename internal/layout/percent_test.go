package layout

import (
	"fmt"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestPercentageLengths(t *testing.T) {
	for _, tt := range []struct {
		name      string
		length    props.Length
		reference units.MaybeFloat
		want      units.MaybeFloat
	}{
		{"pixels", props.Px(12), units.F(200), units.F(12)},
		{"percentage", props.Length{Dimension: props.Dimension{Value: 25, Unit: "%"}}, units.F(200), units.F(50)},
		{"negative reference", props.Length{Dimension: props.Dimension{Value: 25, Unit: "%"}}, units.F(-40), units.F(-10)},
		{"auto", props.LengthKeyword("auto"), units.F(200), units.Auto},
		{"indefinite percentage", props.Length{Dimension: props.Dimension{Value: 50, Unit: "%"}}, units.Auto, units.Auto},
		{"indefinite pixels", props.Px(12), units.Auto, units.F(12)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentage(tt.length, nil, tt.reference); got != tt.want {
				t.Fatalf("percentage = %+v, want %+v", got, tt.want)
			}
		})
	}
	if got := Percentage(props.Length{Dimension: props.Dimension{Value: 12.5, Unit: "%"}}, nil, 80); got != units.F(10) {
		t.Fatalf("Percentage = %+v, want 10", got)
	}
}

func TestPercentageMathUsesContainingSize(t *testing.T) {
	for _, tt := range []struct {
		css  string
		want float64
	}{
		{"calc(50% + 2em)", 132},
		{"min(50%, 40px)", 40},
		{"max(50%, 40px)", 100},
		{"clamp(20px, 50%, 80px)", 80},
	} {
		t.Run(tt.css, func(t *testing.T) {
			box := parseBox(t, `<div style="width:`+tt.css+`"></div>`, "div")
			resolvePercentages(box, size(200, 100))
			if got := box.Base().Width; got != units.F(tt.want) {
				t.Fatalf("width = %+v, want %g", got, tt.want)
			}
		})
	}
}

func TestPercentageHeightsWithIndefiniteContainingHeight(t *testing.T) {
	for _, tt := range []struct {
		css                      string
		height, minimum, maximum units.MaybeFloat
	}{
		{"height: 50%; min-height: 20%; max-height: 80%", units.Auto, units.F(0), units.F(math.Inf(1))},
		{"height: 40px; min-height: auto; max-height: none", units.F(40), units.F(0), units.F(math.Inf(1))},
		{"height: calc(20px + 10%); min-height: 12px; max-height: 60px", units.Auto, units.F(12), units.F(60)},
		{"max-height: 0%", units.Auto, units.F(0), units.F(0)},
	} {
		t.Run(tt.css, func(t *testing.T) {
			box := parseBox(t, `<div style="`+tt.css+`"></div>`, "div")
			resolvePercentages(box, autoHeight(200))
			b := box.Base()
			if !sameMaybe(b.Height, tt.height) || !sameMaybe(b.MinHeight, tt.minimum) || !sameMaybe(b.MaxHeight, tt.maximum) {
				t.Fatalf("height/min/max = %+v %+v %+v, want %+v %+v %+v", b.Height, b.MinHeight, b.MaxHeight, tt.height, tt.minimum, tt.maximum)
			}
		})
	}
	box := parseBox(t, `<body style="height: 50%"></body>`, "body")
	resolvePercentages(box, size(200, 300))
	if box.Base().Height != units.F(150) {
		t.Fatalf("height = %+v, want 150", box.Base().Height)
	}
}

func TestVerticalPercentagesUsePageHeight(t *testing.T) {
	box := parseBox(t, `<div style="margin: 10%; padding: 5%; width: 50%; height: 25%"></div>`, "div")
	page := boxes.NewPageBox(cascade.PageType{}, box.Base().Style)
	for _, tt := range []struct {
		box                             boxes.Box
		verticalMargin, verticalPadding float64
	}{
		{box, 20, 10}, {page, 10, 5},
	} {
		resolvePercentages(tt.box, size(200, 100))
		b := tt.box.Base()
		if b.MarginTop != units.F(tt.verticalMargin) || b.MarginBottom != b.MarginTop ||
			b.PaddingTop != tt.verticalPadding || b.PaddingBottom != b.PaddingTop ||
			b.MarginLeft != units.F(20) || b.PaddingLeft != 10 || b.Width != units.F(100) || b.Height != units.F(25) {
			t.Fatalf("incorrect percentage geometry: %+v", b)
		}
	}
	if got := containingBlockOf(box); got != size(100, 25) {
		t.Fatalf("containing size = %+v", got)
	}
}

func TestPositionPercentagesUseBothAxes(t *testing.T) {
	box := parseBox(t, `<div style="position: absolute; left: 10%; right: auto; top: 20%; bottom: 3px"></div>`, "div")
	resolvePositionPercentages(box, size(200, 100))
	b := box.Base()
	if b.Left != units.F(20) || b.Right != units.Auto || b.Top != units.F(20) || b.Bottom != units.F(3) {
		t.Fatalf("offsets = %+v %+v %+v %+v", b.Left, b.Right, b.Top, b.Bottom)
	}
	resolvePositionPercentages(box, autoHeight(200))
	if b.Top != units.Auto || b.Bottom != units.F(3) {
		t.Fatalf("indefinite offsets = %+v %+v", b.Top, b.Bottom)
	}
}

func TestBoxSizingResolvesContentSizesAndLimits(t *testing.T) {
	for _, tt := range []struct {
		sizing string
		want   float64
	}{
		{"content-box", 1000}, {"padding-box", 980}, {"border-box", 978},
	} {
		for _, css := range []string{
			"width: 10%; height: 1000px",
			"max-width: 10%; max-height: 1000px; height: 2000px",
			"width: 5%; min-width: 10%; min-height: 1000px",
			"width: 10%; height: 1000px; min-width: auto; max-height: none",
		} {
			t.Run(tt.sizing+"/"+css, func(t *testing.T) {
				box := parseBox(t, fmt.Sprintf(`<div style="%s; box-sizing: %s; margin: 100px; padding: 10px; border: 1px solid"></div>`, css, tt.sizing), "div")
				resolvePercentages(box, size(10000, 100000))
				b := box.Base()
				withMinMaxWidth(b, func() {
					if b.Width.Auto {
						b.Width = units.F(9800)
					}
				})
				withMinMaxHeight(b, func() {
					if b.Height.Auto {
						b.Height = units.F(0)
					}
				})
				wrongContent := b.Width != units.F(tt.want) || b.Height != units.F(tt.want)
				wrongOuter := b.PaddingWidth() != tt.want+20 || b.BorderHeight() != tt.want+22 ||
					b.MarginHeight() != tt.want+222
				if wrongContent || wrongOuter {
					t.Fatalf("sizes = %+v %+v, padding %g, border %g, margin %g", b.Width, b.Height, b.PaddingWidth(), b.BorderHeight(), b.MarginHeight())
				}
			})
		}
	}
}

func TestBoxSizingNeverMakesContentNegative(t *testing.T) {
	for _, sizing := range []string{"content-box", "padding-box", "border-box"} {
		box := parseBox(t, `<div style="box-sizing:`+sizing+`; width:0; height:0;
			min-width:0; max-width:0; min-height:0; max-height:0; padding:10px; border:1px solid"></div>`, "div")
		resolvePercentages(box, size(10000, 100000))
		b := box.Base()
		wrongContent := b.Width != units.F(0) || b.Height != units.F(0)
		wrongWidthLimits := b.MinWidth != units.F(0) || b.MaxWidth != units.F(0)
		wrongHeightLimits := b.MinHeight != units.F(0) || b.MaxHeight != units.F(0)
		if wrongContent || wrongWidthLimits || wrongHeightLimits || b.BorderWidth() != 22 {
			t.Fatalf("%s: content or limits became negative: %+v", sizing, b)
		}
	}
}

func TestRadiiUseBorderDimensionsAndRemovedDecorations(t *testing.T) {
	box := parseBox(t, `<div style="width:180px; height:80px; padding:5px; border:5px solid; border-radius:50% 20px / 25% 0"></div>`, "div")
	resolvePercentages(box, size(300, 200))
	resolveRadiiPercentages(box)
	b := box.Base()
	wrongRoundCorners := b.BorderTopLeftRadius != (boxes.Radius{X: 100, Y: 25}) ||
		b.BorderBottomRightRadius != b.BorderTopLeftRadius
	wrongSquareCorners := b.BorderTopRightRadius != (boxes.Radius{}) || b.BorderBottomLeftRadius != (boxes.Radius{})
	if wrongRoundCorners || wrongSquareCorners {
		t.Fatalf("radii = %+v %+v %+v %+v", b.BorderTopLeftRadius, b.BorderTopRightRadius, b.BorderBottomRightRadius, b.BorderBottomLeftRadius)
	}
	b.RemoveDecorationSides = boxes.SideTop
	resolveRadiiPercentages(box)
	if b.BorderTopLeftRadius != (boxes.Radius{}) || b.BorderBottomRightRadius != (boxes.Radius{X: 100, Y: 25}) {
		t.Fatalf("removed top radii = %+v %+v", b.BorderTopLeftRadius, b.BorderBottomRightRadius)
	}
}

func TestOrderedBoundsKeepFirstArgumentForNaN(t *testing.T) {
	nan := math.NaN()
	if firstMax(0, nan) != 0 || firstMin(0, nan) != 0 || !math.IsNaN(firstMax(nan, 0)) || !math.IsNaN(firstMin(nan, 0)) {
		t.Fatal("bounds changed unordered comparison semantics")
	}
}

func TestUnboundedPercentageLimitsRetainUnorderedValues(t *testing.T) {
	box := parseBox(t, `<div style="box-sizing:border-box; padding:1px"></div>`, "div")
	zeroPercent := props.Length{Dimension: props.Dimension{Unit: "%"}}
	box.Base().Style.Set(props.PropMaxHeight, zeroPercent)
	if got := resolveOnePercentage(box.Base().Style, props.PropMaxHeight, units.F(math.Inf(1))); !math.IsNaN(got.V) {
		t.Fatalf("unbounded zero percentage = %+v, want NaN", got)
	}
	resolvePercentages(box, autoHeight(200))
	if box.Base().MaxHeight != units.F(0) {
		t.Fatalf("adjusted maximum height = %+v, want 0", box.Base().MaxHeight)
	}
}

func TestEmptyCollapsedTableUsesComputedBorders(t *testing.T) {
	all := parseBoxes(t, `<table style="border-collapse: collapse; border: 8px solid"></table>`, "table")
	for _, box := range all {
		if box.Kind() != boxes.KindTable {
			continue
		}
		resolvePercentages(box, size(200, 100))
		b := box.Base()
		if b.BorderTopWidth != 8 || b.BorderRightWidth != 8 || b.BorderBottomWidth != 8 || b.BorderLeftWidth != 8 {
			t.Fatalf("table borders = %g %g %g %g, want 8 on every side", b.BorderTopWidth, b.BorderRightWidth, b.BorderBottomWidth, b.BorderLeftWidth)
		}
		return
	}
	t.Fatal("no table box")
}

func TestCollapsedCellPreservesResolvedZeroBorders(t *testing.T) {
	box := parseBox(t, `<table style="border-collapse: collapse; border: hidden"><tr><td style="border: 8px solid"></td></tr></table>`, "td")
	if !box.Base().CollapsedBordersResolved || box.Base().Style.BorderTopWidth() != 8 {
		t.Fatal("cell must have computed borders and resolved hidden conflicts")
	}
	resolvePercentages(box, size(200, 100))
	b := box.Base()
	if b.BorderTopWidth != 0 || b.BorderRightWidth != 0 || b.BorderBottomWidth != 0 || b.BorderLeftWidth != 0 {
		t.Fatalf("hidden cell borders = %g %g %g %g, want zero", b.BorderTopWidth, b.BorderRightWidth, b.BorderBottomWidth, b.BorderLeftWidth)
	}
}
