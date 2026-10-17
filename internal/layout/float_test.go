package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestFloatCollisionMovesBelowInsufficientSpace(t *testing.T) {
	containing := parseBox(t, `<body style="width: 290px">`, "body")
	resolvePercentages(containing, size(290, 500))
	containing.Base().Height = units.F(500)
	shape := parseBox(t, `<div style="float: left; width: 100px; height: 60px">`, "div")
	resolvePercentages(shape, size(290, 500))
	second := shape.Copy()
	second.Base().PositionX = 100
	ctx := &Context{}
	ctx.CreateBFC(nil, []boxes.Box{shape, second})
	x, y, width := avoidCollisions(ctx, shape, containing, true)
	if x != 0 || y != 60 || width != 290 {
		t.Fatalf("collision = (%g,%g,%g), want (0,60,290)", x, y, width)
	}
	line := boxes.NewLineBox("body", containing.Base().Style, nil, nil)
	line.Width, line.Height = units.F(60), units.F(60)
	x, y, width = avoidCollisions(ctx, line, containing, false)
	if x != 200 || y != 0 || width != 90 {
		t.Fatalf("line collision = (%g,%g,%g), want (200,0,90)", x, y, width)
	}
}

func TestFloatClearanceMatchesLogicalDirections(t *testing.T) {
	shape := parseBox(t, `<div style="float: inline-start; height: 60px">`, "div")
	resolvePercentages(shape, size(100, 500))
	block := parseBox(t, `<div style="clear: right; margin-top: 10px">`, "div")
	resolvePercentages(block, size(100, 500))
	ctx := &Context{}
	ctx.CreateBFC(nil, []boxes.Box{shape})
	if clearance := getClearance(ctx, block, props.Keyword("ltr"), 10); clearance != nil {
		t.Fatalf("unexpected clearance: %g", *clearance)
	}
	if clearance := getClearance(ctx, block, props.Keyword("rtl"), 10); clearance == nil || *clearance != 50 {
		t.Fatalf("clearance = %v, want 50", clearance)
	}
}
