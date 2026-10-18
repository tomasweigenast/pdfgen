package draw

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestSpreadBox(t *testing.T) {
	rounded := boxes.RoundedBox{
		Rect:        boxes.Rect{X: 10, Y: 20, Width: 30, Height: 40},
		TopLeft:     boxes.Radius{X: 5, Y: 5},
		TopRight:    boxes.Radius{X: 5, Y: 8},
		BottomRight: boxes.Radius{},
		BottomLeft:  boxes.Radius{X: 2, Y: 2},
	}
	tests := []struct {
		name           string
		spread, tx, ty float64
		want           boxes.RoundedBox
	}{
		{"nothing to apply", 0, 0, 0, rounded},
		{
			"translation keeps radii", 0, 3, -4,
			boxes.RoundedBox{
				Rect:        boxes.Rect{X: 13, Y: 16, Width: 30, Height: 40},
				TopLeft:     boxes.Radius{X: 5, Y: 5},
				TopRight:    boxes.Radius{X: 5, Y: 8},
				BottomRight: boxes.Radius{},
				BottomLeft:  boxes.Radius{X: 2, Y: 2},
			},
		},
		{
			"positive spread grows radii but not square corners", 2, 0, 0,
			boxes.RoundedBox{
				Rect:        boxes.Rect{X: 8, Y: 18, Width: 34, Height: 44},
				TopLeft:     boxes.Radius{X: 7, Y: 7},
				TopRight:    boxes.Radius{X: 7, Y: 10},
				BottomRight: boxes.Radius{},
				BottomLeft:  boxes.Radius{X: 4, Y: 4},
			},
		},
		{
			"negative spread squares the corners it would remove", -3, 0, 0,
			boxes.RoundedBox{
				Rect:        boxes.Rect{X: 13, Y: 23, Width: 24, Height: 34},
				TopLeft:     boxes.Radius{X: 2, Y: 2},
				TopRight:    boxes.Radius{X: 2, Y: 5},
				BottomRight: boxes.Radius{},
				BottomLeft:  boxes.Radius{},
			},
		},
		{
			"size never goes negative", -30, 0, 0,
			boxes.RoundedBox{Rect: boxes.Rect{X: 40, Y: 50, Width: 0, Height: 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := spreadBox(rounded, tt.spread, tt.tx, tt.ty); got != tt.want {
				t.Errorf("spreadBox() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
