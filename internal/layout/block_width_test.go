package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestBlockWidthResolvesAutoMargins(t *testing.T) {
	for _, tt := range []struct {
		css         string
		width, left float64
	}{
		{"", 94, 0},
		{"width:50px", 50, 0},
		{"margin:0 10px 0 20px", 64, 20},
		{"width:50px; margin-left:20px; margin-right:auto", 50, 20},
		{"width:50px; margin-left:auto; margin-right:20px", 50, 24},
		{"width:50px; margin:auto", 50, 22},
		{"margin-left:20px; margin-right:auto", 74, 20},
		{"margin-left:auto; margin-right:20px", 74, 0},
		{"margin:auto", 94, 0},
		{"width:200px; margin:auto", 200, 0},
		{"min-width:200px; margin:auto", 200, 0},
		{"max-width:50px; margin:auto", 50, 22},
		{"min-width:50px; margin:auto", 94, 0},
		{"width:70%", 70, 0},
	} {
		t.Run(tt.css, func(t *testing.T) {
			box := parseBox(t, `<p style="padding:2px; border:1px solid; `+tt.css+`"></p>`, "p")
			resolvePercentages(box, size(100, 2000))
			b := box.Base()
			blockLevelWidth(b, size(100, 2000), "ltr")
			if b.Width != units.F(tt.width) || b.MarginLeft != units.F(tt.left) || b.MarginRight.Auto {
				t.Fatalf("width/margins = %+v %+v %+v, want width %g left %g", b.Width, b.MarginLeft, b.MarginRight, tt.width, tt.left)
			}
		})
	}
}

func TestBlockWidthPositionsRightToLeft(t *testing.T) {
	for _, tt := range []struct {
		direction props.Keyword
		column    bool
		want      float64
	}{
		{"ltr", false, 10}, {"rtl", false, 54}, {"rtl", true, 10},
	} {
		box := parseBox(t, `<p style="width:50px; padding:2px; border:1px solid"></p>`, "p")
		resolvePercentages(box, size(100, 2000))
		b := box.Base()
		b.PositionX, b.IsColumn = 10, tt.column
		blockLevelWidth(b, size(100, 2000), tt.direction)
		if b.PositionX != tt.want || b.MarginRight != units.F(0) {
			t.Fatalf("%s column %t: x=%g, margin-right=%+v", tt.direction, tt.column, b.PositionX, b.MarginRight)
		}
	}
}

func TestBlockWidthCanIgnoreLimits(t *testing.T) {
	box := parseBox(t, `<p style="width:200px; min-width:300px; max-width:50px; margin:auto"></p>`, "p")
	resolvePercentages(box, size(100, 2000))
	b := box.Base()
	blockLevelWidthWithoutMinMax(b, size(100, 2000), "ltr")
	if b.Width != units.F(200) || b.MarginLeft != units.F(0) || b.MarginRight != units.F(0) {
		t.Fatalf("unlimited dimensions = %+v %+v %+v", b.Width, b.MarginLeft, b.MarginRight)
	}
}

func TestBlockWidthClampsBeforeRightToLeftPositioning(t *testing.T) {
	box := parseBox(t, `<p style="width:200px; max-width:50px"></p>`, "p")
	resolvePercentages(box, size(100, 2000))
	box.Base().PositionX = 10
	blockLevelWidth(box.Base(), size(100, 2000), "rtl")
	if box.Base().PositionX != 60 || box.Base().Width != units.F(50) {
		t.Fatalf("x=%g, width=%+v", box.Base().PositionX, box.Base().Width)
	}
}
