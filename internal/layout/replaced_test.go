package layout

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func parseReplaced(t *testing.T, source string) *boxes.ReplacedFields {
	t.Helper()
	r, ok := parseBox(t, source, "img").(boxes.Replaced)
	if !ok {
		t.Fatalf("img in %q made a box that is not replaced", source)
	}
	return r.ReplacedBase()
}

type testImage struct{ w, h, ratio units.MaybeFloat }

func (i testImage) IntrinsicSize(resolution, _ float64) (w, h, ratio units.MaybeFloat) {
	w, h = units.Auto, units.Auto
	if !i.w.Auto {
		w = units.F(i.w.V / resolution)
	}
	if !i.h.Auto {
		h = units.F(i.h.V / resolution)
	}
	return w, h, i.ratio
}

func (testImage) Draw(*stream.Stream, float64, float64, string) {}

func TestDefaultImageSizing(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		intrinsic             testImage
		width, height         units.MaybeFloat
		wantWidth, wantHeight float64
	}{
		{"specified", testImage{units.Auto, units.Auto, units.Auto}, units.F(40), units.F(30), 40, 30},
		{"specified width with ratio", testImage{units.Auto, units.Auto, units.F(2)}, units.F(40), units.Auto, 40, 20},
		{"specified width with height", testImage{units.Auto, units.F(25), units.Auto}, units.F(40), units.Auto, 40, 25},
		{"specified width with fallback", testImage{units.Auto, units.Auto, units.Auto}, units.F(40), units.Auto, 40, 150},
		{"specified height with ratio", testImage{units.Auto, units.Auto, units.F(2)}, units.Auto, units.F(30), 60, 30},
		{"specified height with width", testImage{units.F(25), units.Auto, units.Auto}, units.Auto, units.F(30), 25, 30},
		{"specified height with fallback", testImage{units.Auto, units.Auto, units.Auto}, units.Auto, units.F(30), 300, 30},
		{"intrinsic dimensions", testImage{units.F(40), units.F(30), units.F(2)}, units.Auto, units.Auto, 40, 30},
		{"intrinsic width with ratio", testImage{units.F(40), units.Auto, units.F(2)}, units.Auto, units.Auto, 40, 20},
		{"intrinsic height with ratio", testImage{units.Auto, units.F(30), units.F(2)}, units.Auto, units.Auto, 60, 30},
		{"ratio alone", testImage{units.Auto, units.Auto, units.F(1)}, units.Auto, units.Auto, 150, 150},
		{"fallback", testImage{units.Auto, units.Auto, units.Auto}, units.Auto, units.Auto, 300, 150},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, h := defaultImageSizing(
				tt.intrinsic.w,
				tt.intrinsic.h,
				tt.intrinsic.ratio,
				tt.width,
				tt.height,
				300,
				150,
			)
			if w != tt.wantWidth || h != tt.wantHeight {
				t.Fatalf("size = %g × %g, want %g × %g", w, h, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestImageConstraintSizing(t *testing.T) {
	for _, tt := range []struct {
		ratio                                                units.MaybeFloat
		containWidth, containHeight, coverWidth, coverHeight float64
	}{
		{units.Auto, 100, 50, 100, 50},
		{units.F(1), 50, 50, 100, 100},
		{units.F(2), 100, 50, 100, 50},
		{units.F(4), 100, 25, 200, 50},
		{units.F(0), 0, 50, 100, math.Inf(1)},
	} {
		w, h := containConstraintImageSizing(100, 50, tt.ratio)
		if w != tt.containWidth || h != tt.containHeight {
			t.Fatalf("contain(%+v) = %g × %g", tt.ratio, w, h)
		}
		w, h = coverConstraintImageSizing(100, 50, tt.ratio)
		if w != tt.coverWidth || h != tt.coverHeight {
			t.Fatalf("cover(%+v) = %g × %g", tt.ratio, w, h)
		}
	}
}

func TestReplacedObjectFitAndPosition(t *testing.T) {
	for _, tt := range []struct {
		css  string
		want boxes.Rect
	}{
		{"object-fit:fill", boxes.Rect{X: 17, Y: 27, Width: 100, Height: 50}},
		{"object-fit:contain", boxes.Rect{X: 42, Y: 27, Width: 50, Height: 50}},
		{"object-fit:cover", boxes.Rect{X: 17, Y: 2, Width: 100, Height: 100}},
		{"object-fit:none", boxes.Rect{X: 57, Y: 42, Width: 20, Height: 20}},
		{"object-fit:scale-down", boxes.Rect{X: 57, Y: 42, Width: 20, Height: 20}},
		{"object-fit:none; object-position:right 10px bottom 5px", boxes.Rect{X: 87, Y: 52, Width: 20, Height: 20}},
		{"object-fit:cover; object-position:25% 100%", boxes.Rect{X: 17, Y: -23, Width: 100, Height: 100}},
	} {
		t.Run(tt.css, func(t *testing.T) {
			style := parseBox(t, `<div style="width:100px; height:50px; margin:3px; padding:2px; border:2px solid; `+tt.css+`"></div>`, "div").Base().Style
			box := boxes.NewInlineReplacedBox("img", style, nil, testImage{units.F(20), units.F(20), units.F(1)})
			resolvePercentages(box, size(200, 100))
			box.PositionX, box.PositionY = 10, 20
			if got := ReplacedBoxLayout(box); got != tt.want {
				t.Fatalf("rectangle = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReplacedObjectFitWithIncompleteIntrinsicSize(t *testing.T) {
	style := parseBox(t, `<div style="width:100px; height:50px; object-fit:none"></div>`, "div").Base().Style
	box := boxes.NewInlineReplacedBox("img", style, nil, testImage{units.F(20), units.Auto, units.F(1)})
	resolvePercentages(box, size(200, 100))
	if got := ReplacedBoxLayout(box); got != (boxes.Rect{X: 25, Y: 0, Width: 50, Height: 50}) {
		t.Fatalf("rectangle = %+v", got)
	}
	style.Set(props.PropObjectFit, props.Keyword("scale-down"))
	box.Replacement = testImage{units.F(200), units.F(200), units.F(1)}
	if got := ReplacedBoxLayout(box); got != (boxes.Rect{X: 25, Y: 0, Width: 50, Height: 50}) {
		t.Fatalf("scaled rectangle = %+v", got)
	}
}

func TestReplacedSizingUsesIntrinsicSizeAndLimits(t *testing.T) {
	for _, tt := range []struct {
		css           string
		width, height float64
	}{
		{"", 4, 4}, {"width:40px", 40, 40}, {"height:40px", 40, 40},
		{"width:20%", 40, 40}, {"min-width:40px", 40, 40}, {"max-width:2px", 2, 2},
		{"min-height:30px; max-width:2px", 2, 30},
		{"width:40px; max-height:10px", 40, 10},
		{"height:40px; max-width:10px", 10, 40},
		{"image-resolution:2dppx", 2, 2},
	} {
		t.Run(tt.css, func(t *testing.T) {
			r := parseReplaced(t, `<img src="pattern.png" style="`+tt.css+`">`)
			box := boxes.NewInlineReplacedBox("img", r.Style, r.Element, r.Replacement)
			resolvePercentages(box, size(200, 100))
			inlineReplacedBoxLayout(&box.ReplacedFields, size(200, 100), "ltr")
			if box.Width != units.F(tt.width) || box.Height != units.F(tt.height) {
				t.Fatalf("size = %+v × %+v, want %g × %g", box.Width, box.Height, tt.width, tt.height)
			}
		})
	}
}

func TestReplacedSizingWithMissingIntrinsicDimensions(t *testing.T) {
	for _, tt := range []struct {
		name, css             string
		image                 testImage
		wantWidth, wantHeight float64
	}{
		{"fallback", "", testImage{units.Auto, units.Auto, units.Auto}, 300, 150},
		{"width alone", "", testImage{units.F(60), units.Auto, units.Auto}, 60, 150},
		{"height alone", "", testImage{units.Auto, units.F(40), units.Auto}, 300, 40},
		{"height and ratio", "", testImage{units.Auto, units.F(40), units.F(2)}, 80, 40},
		{"width and ratio", "", testImage{units.F(60), units.Auto, units.F(2)}, 60, 30},
		{"ratio alone", "", testImage{units.Auto, units.Auto, units.F(2)}, 200, 100},
		{"specified height and ratio", "height:40px", testImage{units.Auto, units.Auto, units.F(2)}, 80, 40},
	} {
		t.Run(tt.name, func(t *testing.T) {
			style := parseBox(t, `<div style="`+tt.css+`"></div>`, "div").Base().Style
			box := boxes.NewInlineReplacedBox("img", style, nil, tt.image)
			resolvePercentages(box, size(200, 100))
			inlineReplacedBoxWidthHeight(&box.ReplacedFields, size(200, 100), "ltr")
			if box.Width != units.F(tt.wantWidth) || box.Height != units.F(tt.wantHeight) {
				t.Fatalf("size = %+v × %+v, want %g × %g", box.Width, box.Height, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestInlineReplacedMarginsResolveToZero(t *testing.T) {
	r := parseReplaced(t, `<img src="pattern.png" style="margin:auto">`)
	box := boxes.NewInlineReplacedBox("img", r.Style, r.Element, r.Replacement)
	resolvePercentages(box, size(200, 100))
	inlineReplacedBoxLayout(&box.ReplacedFields, size(200, 100), "ltr")
	if box.MarginTop != units.F(0) || box.MarginRight != units.F(0) || box.MarginBottom != units.F(0) || box.MarginLeft != units.F(0) {
		t.Fatalf("margins = %+v %+v %+v %+v", box.MarginTop, box.MarginRight, box.MarginBottom, box.MarginLeft)
	}
}

func TestBlockReplacedWidthRecomputesAutoMargins(t *testing.T) {
	r := parseReplaced(t, `<img src="pattern.png" style="width:40px; max-width:20px; margin:10px auto; display:block">`)
	box := boxes.NewBlockReplacedBox("img", r.Style, r.Element, r.Replacement)
	resolvePercentages(box, size(100, 100))
	blockReplacedWidth(&box.ReplacedFields, size(100, 100), "ltr")
	if box.Width != units.F(20) || box.MarginLeft != units.F(30) || box.MarginRight != units.F(30) {
		t.Fatalf("width/margins = %+v %+v %+v", box.Width, box.MarginLeft, box.MarginRight)
	}
}

func TestAutoReplacedLimitsPreserveRatioWhenPossible(t *testing.T) {
	for _, tt := range []struct {
		name                                                                           string
		width, height, minWidth, maxWidth, minHeight, maxHeight, wantWidth, wantHeight float64
	}{
		{"inside", 100, 50, 0, 200, 0, 100, 100, 50},
		{"width max", 100, 50, 0, 80, 0, 100, 80, 40},
		{"width min", 100, 50, 120, 200, 0, 100, 120, 60},
		{"height max", 100, 50, 0, 200, 0, 40, 80, 40},
		{"height min", 100, 50, 0, 200, 60, 100, 120, 60},
		{"both max width stronger", 100, 50, 0, 60, 0, 40, 60, 30},
		{"both max height stronger", 100, 50, 0, 80, 0, 30, 60, 30},
		{"both min height stronger", 100, 50, 120, 200, 80, 100, 160, 80},
		{"both min width stronger", 100, 50, 160, 200, 60, 100, 160, 80},
		{"width min height max", 100, 50, 120, 200, 0, 40, 120, 40},
		{"width max height min", 100, 50, 0, 80, 60, 100, 80, 60},
		{"minimum wins", 100, 50, 120, 80, 60, 40, 120, 60},
		{"zero width", 0, 50, 10, 100, 0, 100, 10, 100},
		{"zero height", 100, 0, 0, 200, 10, 100, 200, 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			box := &boxes.ReplacedFields{BoxFields: boxes.BoxFields{
				Width: units.F(tt.width), Height: units.F(tt.height),
				MinWidth: units.F(tt.minWidth), MaxWidth: units.F(tt.maxWidth),
				MinHeight: units.F(tt.minHeight), MaxHeight: units.F(tt.maxHeight),
			}}
			minMaxAutoReplaced(box)
			if box.Width != units.F(tt.wantWidth) || box.Height != units.F(tt.wantHeight) {
				t.Fatalf("size = %+v × %+v, want %g × %g", box.Width, box.Height, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestReplacedRatioOnlyWidthUsesContainingDirection(t *testing.T) {
	style := parseBox(t, `<div style="max-width:100px"></div>`, "div").Base().Style
	box := boxes.NewInlineReplacedBox("img", style, nil, testImage{w: units.Auto, h: units.Auto, ratio: units.F(2)})
	resolvePercentages(box, size(200, 100))
	inlineReplacedBoxWidthHeight(&box.ReplacedFields, size(200, 100), "rtl")
	if box.Width != units.F(100) || box.Height != units.F(50) || box.PositionX != 100 {
		t.Fatalf("size/position = %+v %+v %g", box.Width, box.Height, box.PositionX)
	}
}

func TestReplacedIntrinsicSizeIncludesOrientationAndResolution(t *testing.T) {
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 6, 4))); err != nil {
		t.Fatal(err)
	}
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
	for _, tt := range []struct {
		orientation   string
		width, height float64
	}{
		{"none", 3, 2}, {"90deg", 2, 3}, {"180deg", 3, 2}, {"270deg flip", 2, 3},
	} {
		t.Run(tt.orientation, func(t *testing.T) {
			r := parseReplaced(t, `<img src="`+url+`" style="image-resolution:2dppx; image-orientation:`+tt.orientation+`">`)
			box := boxes.NewInlineReplacedBox("img", r.Style, r.Element, r.Replacement)
			resolvePercentages(box, size(200, 100))
			inlineReplacedBoxWidthHeight(&box.ReplacedFields, size(200, 100), "ltr")
			if box.Width != units.F(tt.width) || box.Height != units.F(tt.height) {
				t.Fatalf("size = %+v × %+v, want %g × %g", box.Width, box.Height, tt.width, tt.height)
			}
		})
	}
}
