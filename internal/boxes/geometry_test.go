package boxes_test

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func sampleFields() *boxes.BoxFields {
	return &boxes.BoxFields{
		PositionX: 100, PositionY: 200,
		Width: units.F(50), Height: units.F(30),
		MarginTop: units.F(1), MarginRight: units.F(2), MarginBottom: units.F(3), MarginLeft: units.F(4),
		PaddingTop: 5, PaddingRight: 6, PaddingBottom: 7, PaddingLeft: 8,
		BorderTopWidth: 9, BorderRightWidth: 10, BorderBottomWidth: 11, BorderLeftWidth: 12,
	}
}

func TestBoxSizesAddEachLayer(t *testing.T) {
	b := sampleFields()
	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"padding width", b.PaddingWidth(), 50 + 8 + 6},
		{"padding height", b.PaddingHeight(), 30 + 5 + 7},
		{"border width", b.BorderWidth(), 50 + 8 + 6 + 12 + 10},
		{"border height", b.BorderHeight(), 30 + 5 + 7 + 9 + 11},
		{"margin width", b.MarginWidth(), 50 + 8 + 6 + 12 + 10 + 4 + 2},
		{"margin height", b.MarginHeight(), 30 + 5 + 7 + 9 + 11 + 1 + 3},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestBoxCornersFollowTheLayers(t *testing.T) {
	b := sampleFields()
	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"border box x", b.BorderBoxX(), 100 + 4},
		{"border box y", b.BorderBoxY(), 200 + 1},
		{"padding box x", b.PaddingBoxX(), 100 + 4 + 12},
		{"padding box y", b.PaddingBoxY(), 200 + 1 + 9},
		{"content box x", b.ContentBoxX(), 100 + 4 + 12 + 8},
		{"content box y", b.ContentBoxY(), 200 + 1 + 9 + 5},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestAutoDimensionsCountAsZero(t *testing.T) {
	b := sampleFields()
	b.Width, b.MarginLeft = units.Auto, units.Auto
	if got, want := b.MarginWidth(), float64(8+6+12+10+2); got != want {
		t.Errorf("MarginWidth = %v, want %v", got, want)
	}
}

func TestHitAreaIsTheBorderBox(t *testing.T) {
	b := sampleFields()
	want := boxes.Rect{X: 104, Y: 201, Width: 86, Height: 62}
	if got := b.HitArea(); got != want {
		t.Errorf("HitArea = %+v, want %+v", got, want)
	}
}

func TestRoundedBoxesShrinkWithTheBorder(t *testing.T) {
	b := &boxes.BoxFields{
		Width: units.F(80), Height: units.F(40),
		BorderTopWidth: 5, BorderRightWidth: 5, BorderBottomWidth: 5, BorderLeftWidth: 5,
		BorderTopLeftRadius:     boxes.Radius{X: 20, Y: 10},
		BorderTopRightRadius:    boxes.Radius{X: 4, Y: 4},
		BorderBottomRightRadius: boxes.Radius{X: 0, Y: 0},
		BorderBottomLeftRadius:  boxes.Radius{X: 9, Y: 30},
	}
	border := b.RoundedBorderBox()
	if border.Rect != (boxes.Rect{X: 0, Y: 0, Width: 90, Height: 50}) || border.TopLeft != (boxes.Radius{X: 20, Y: 10}) {
		t.Errorf("border box = %+v", border)
	}
	padding := b.RoundedPaddingBox()
	if padding.Rect != (boxes.Rect{X: 5, Y: 5, Width: 80, Height: 40}) {
		t.Errorf("padding box rect = %+v", padding.Rect)
	}
	want := map[string][2]boxes.Radius{
		"top left":    {padding.TopLeft, {X: 15, Y: 5}},
		"top right":   {padding.TopRight, {X: 0, Y: 0}},
		"bottom left": {padding.BottomLeft, {X: 4, Y: 25}},
	}
	for name, w := range want {
		if w[0] != w[1] {
			t.Errorf("%s radius = %+v, want %+v", name, w[0], w[1])
		}
	}
	half := b.RoundedBoxRatio(0.5)
	if half.Rect != (boxes.Rect{X: 2.5, Y: 2.5, Width: 85, Height: 45}) {
		t.Errorf("half border box rect = %+v", half.Rect)
	}
}

func TestRoundedBoxScalesOverlappingRadii(t *testing.T) {
	b := &boxes.BoxFields{
		Width:               units.F(100),
		Height:              units.F(100),
		BorderTopLeftRadius: boxes.Radius{X: 80, Y: 10}, BorderTopRightRadius: boxes.Radius{X: 80, Y: 10},
	}
	got := b.RoundedBorderBox()
	want := boxes.Radius{X: 50, Y: 6.25}
	if got.TopLeft != want || got.TopRight != want {
		t.Errorf("radii = %+v and %+v, want %+v", got.TopLeft, got.TopRight, want)
	}
}

func TestRoundedBoxClampsHugeRadii(t *testing.T) {
	b := &boxes.BoxFields{Width: units.F(10), Height: units.F(10), BorderTopLeftRadius: boxes.Radius{X: math.Inf(1), Y: -3}}
	got := b.RoundedBorderBox()
	if math.IsInf(got.TopLeft.X, 0) || got.TopLeft.Y != 0 {
		t.Errorf("radius = %+v, want a finite x and a zero y", got.TopLeft)
	}
}
