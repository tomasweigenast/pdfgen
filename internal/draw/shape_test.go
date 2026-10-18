package draw

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

func newTestStream() *stream.Stream {
	return stream.New(stream.NewResources(nil), stream.Options{})
}

func TestRoundedBoxPaths(t *testing.T) {
	rect := boxes.Rect{X: 1, Y: 2, Width: 10, Height: 6}
	tests := []struct {
		name string
		box  boxes.RoundedBox
		want string
	}{
		{"no radius", boxes.RoundedBox{Rect: rect}, "1 2 10 6 re"},
		{
			"corners with a zero component stay square",
			boxes.RoundedBox{Rect: rect, TopLeft: boxes.Radius{X: 3}, BottomRight: boxes.Radius{Y: 2}},
			"1 2 10 6 re",
		},
		{
			"one rounded corner",
			boxes.RoundedBox{Rect: rect, TopLeft: boxes.Radius{X: 2, Y: 2}},
			"3 2 m\n11 2 l\n11 2 11 2 11 2 c\n11 8 l\n11 8 11 8 11 8 c\n" +
				"1 8 l\n1 8 1 8 1 8 c\n1 4 l\n1 2.9 1.9 2 3 2 c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStream()
			roundedBox(s, tt.box)
			if got := string(s.Bytes()); got != tt.want {
				t.Errorf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClipRoundedBoxEndsPath(t *testing.T) {
	s := newTestStream()
	clipRoundedBox(s, boxes.RoundedBox{Rect: boxes.Rect{Width: 4, Height: 4}})
	if got, want := string(s.Bytes()), "0 0 4 4 re\nW\nn"; got != want {
		t.Errorf("clip = %q, want %q", got, want)
	}
}
