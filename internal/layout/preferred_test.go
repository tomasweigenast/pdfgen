package layout

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func preferredContext(box boxes.Box) *Context {
	return &Context{Text: linebreak.NewContext(box.Base().Style.Context().FontConfig)}
}

func TestPreferredInlineNestedTrailingSpaces(t *testing.T) {
	for _, tt := range []struct {
		name, contents string
		width          float64
	}{
		{"empty inline block", `<span style="display:inline-block"></span> a`, 4},
		{"nested trailing spaces", `<span>a </span> `, 2},
		{"trailing space in nested", `<span style="display:inline-block"></span><span><span>a</span> </span>`, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			box := parseBox(t, `<style>div { font:2px testfont; float:left }</style><div>`+tt.contents+`</div>`, "div")
			ctx := preferredContext(box)
			if got := maxContentWidth(ctx, box, false); got != tt.width {
				t.Fatalf("max-content = %g, want %g", got, tt.width)
			}
			if got := shrinkToFit(ctx, box, 1000); got != tt.width {
				t.Fatalf("shrink-to-fit = %g, want %g", got, tt.width)
			}
		})
	}
}

// fortyPercent keeps 1-40/100 out of constant folding so the expectation
// rounds like the runtime division.
var fortyPercent = 40.0

func TestIntrinsicBlockWidths(t *testing.T) {
	for _, tt := range []struct {
		name, css, contents     string
		minimum, maximum, outer float64
	}{
		{"children", "", `<p>aa bbbb</p><p>c d</p>`, 8, 14, 14},
		{"absolute child", "", `<p>aa bbbb</p><p style="position:absolute;width:200px"></p>`, 8, 14, 14},
		{"fixed width", "width:40px", `aa bbbb`, 40, 40, 40},
		{"percentage width", "width:40%", `aa bbbb`, 8, 14, 14},
		{"fixed minimum", "min-width:20px", `aa bbbb`, 20, 20, 20},
		{"fixed maximum", "max-width:10px", `aa bbbb`, 8, 10, 10},
		{"percentage limits", "min-width:80%;max-width:90%", `aa bbbb`, 8, 14, 14},
		{"box sizing", "box-sizing:border-box;width:100px;padding:10px;border:5px solid", ``, 70, 70, 100},
		{"percentage padding", "box-sizing:border-box;width:100px;padding:10%;border:5px solid", ``, 72, 72, 102.5},
		{"percentage outer", "padding:10%;margin:10%", `aa bbbb`, 8, 14, 14 / (1 - fortyPercent/100)},
		{"unbounded percentage outer", "padding:50%", `aa bbbb`, 8, 14, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			box := parseBox(t, `<div style="font:2px testfont;`+tt.css+`">`+tt.contents+`</div>`, "div")
			ctx := preferredContext(box)
			minimum, maximum := minContentWidth(ctx, box, false), maxContentWidth(ctx, box, false)
			outer := maxContentWidth(ctx, box, true)
			if minimum != tt.minimum || maximum != tt.maximum || outer != tt.outer {
				t.Fatalf("min/max/outer = %g/%g/%g, want %g/%g/%g", minimum, maximum, outer, tt.minimum, tt.maximum, tt.outer)
			}
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
		})
	}
}

func TestIntrinsicInlineLineWidths(t *testing.T) {
	for _, tt := range []struct {
		name, css, contents string
		minimum, maximum    []float64
	}{
		{"words", "", "ab cdef", []float64{4, 8}, []float64{14}},
		{"nowrap", "white-space:nowrap", "ab cdef", []float64{14}, []float64{14}},
		{"preserved breaks", "white-space:pre", "ab\ncdef", []float64{4, 8}, []float64{4, 8}},
		{"nested fragments", "", `<span style="padding:0 2px">ab cdef</span> gh`, []float64{6, 10, 4}, []float64{24}},
		{"atomic breaks", "", `ab<span style="display:inline-block;width:10px"></span>cd`, []float64{4, 10, 4}, []float64{18}},
		{"indent", "text-indent:3px", `ab cdef`, []float64{7, 8}, []float64{17}},
		{"percentage indent", "text-indent:50%", `ab cdef`, []float64{4, 8}, []float64{14}},
		{"math indent", "text-indent:calc(50% + 3px)", `ab cdef`, []float64{7, 8}, []float64{17}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			box := parseBox(t, `<p style="font:2px testfont;`+tt.css+`">`+tt.contents+`</p>`, "p")
			line := boxes.ChildrenOf(box)[0].(boxes.Parent)
			ctx := preferredContext(box)
			minimum := inlineLineWidths(ctx, line, true, true, true, nil, false)
			maximum := inlineLineWidths(ctx, line, true, true, false, nil, false)
			if !slices.Equal(minimum, tt.minimum) || !slices.Equal(maximum, tt.maximum) {
				t.Fatalf("min/max lines = %v/%v, want %v/%v", minimum, maximum, tt.minimum, tt.maximum)
			}
		})
	}
}

func TestIntrinsicInlineContinuationUsesByteOffsets(t *testing.T) {
	box := parseBox(t, `<p style="font:2px testfont">é ab cdef</p>`, "p")
	line := boxes.ChildrenOf(box)[0]
	ctx := preferredContext(box)
	skip := ResumeAt{0: ResumeAt{3: nil}}
	if got := inlineMinContentWidth(ctx, line, false, skip, true, true); got != 4 {
		t.Fatalf("first continued min-content = %g, want 4", got)
	}
	if got := inlineMinContentWidth(ctx, line, false, skip, false, true); got != 8 {
		t.Fatalf("continued min-content = %g, want 8", got)
	}
	if !skip.Equal(ResumeAt{0: ResumeAt{3: nil}}) {
		t.Fatal("continuation was changed")
	}
}

func TestTrailingWhitespaceOnlyCollapsesWhereAllowed(t *testing.T) {
	for _, tt := range []struct {
		css, contents string
		want          float64
	}{
		{"white-space:normal", `<span>a </span> `, 2},
		{"white-space:pre", `a  `, 0},
		{"white-space:pre-line", "a\nb  ", 2},
		{"font-size:0", `a `, 0},
		{"", `<span>a</span>`, 0},
	} {
		box := parseBox(t, `<p style="font:2px testfont;`+tt.css+`">`+tt.contents+`</p>`, "p")
		line := boxes.ChildrenOf(box)[0]
		if got := trailingWhitespaceSize(preferredContext(box), line); got != tt.want {
			t.Fatalf("%s %q: whitespace = %g, want %g", tt.css, tt.contents, got, tt.want)
		}
	}
}

func TestReplacedIntrinsicWidthContributions(t *testing.T) {
	for _, tt := range []struct {
		css              string
		image            testImage
		minimum, maximum float64
	}{
		{"", testImage{w: units.Auto, h: units.Auto, ratio: units.Auto}, 0, 300},
		{"max-width:50%", testImage{w: units.F(40), h: units.F(20), ratio: units.F(2)}, 0, 40},
		{"height:10px", testImage{w: units.F(40), h: units.F(20), ratio: units.F(2)}, 20, 20},
		{"width:50%", testImage{w: units.F(40), h: units.F(20), ratio: units.F(2)}, 0, 0},
		{"min-height:30px", testImage{w: units.F(40), h: units.F(20), ratio: units.F(2)}, 60, 60},
		{"max-height:10px", testImage{w: units.F(40), h: units.F(20), ratio: units.F(2)}, 20, 20},
	} {
		style := parseBox(t, `<div style="font:2px testfont;`+tt.css+`"></div>`, "div").Base().Style
		box := boxes.NewInlineReplacedBox("img", style, nil, tt.image)
		ctx := preferredContext(box)
		minimum, maximum := minContentWidth(ctx, box, false), maxContentWidth(ctx, box, false)
		if minimum != tt.minimum || maximum != tt.maximum {
			t.Fatalf("%s: min/max = %g/%g, want %g/%g", tt.css, minimum, maximum, tt.minimum, tt.maximum)
		}
	}
}

func TestUnsupportedIntrinsicKindReportsAnError(t *testing.T) {
	style := parseBox(t, `<div></div>`, "div").Base().Style
	box := boxes.NewTableBox("table", style, nil, nil)
	ctx := preferredContext(box)
	minContentWidth(ctx, box, false)
	if ctx.Err() == nil {
		t.Fatal("table sizing did not report its unavailable algorithm")
	}
	first := errors.New("first failure")
	ctx.err = first
	maxContentWidth(ctx, box, false)
	if !errors.Is(ctx.Err(), first) {
		t.Fatal("intrinsic sizing overwrote the first error")
	}
}

func TestIntrinsicMathBoundsKeepOrderedNaNBehavior(t *testing.T) {
	box := parseBox(t, `<div style="min-width:5px"></div>`, "div")
	if got := preferredMinMax(box, math.NaN()); got != 5 {
		t.Fatalf("minimum with unordered candidate = %g, want 5", got)
	}
	box.Base().Style.Set(props.PropMaxWidth, props.Px(math.NaN()))
	if got := preferredMinMax(box, 10); got != 10 {
		t.Fatalf("unordered maximum = %g, want 10", got)
	}
}
