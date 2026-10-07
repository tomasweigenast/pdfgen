package testutil

import (
	"slices"
	"strings"
	"testing"
)

func TestParsePixels(t *testing.T) {
	art := `
		RB_   # first row
		  zKg
		# whole-line comment

	`
	w, h, got, err := ParsePixels(art)
	if err != nil {
		t.Fatal(err)
	}
	if w != 3 || h != 2 {
		t.Errorf("size = %d×%d, want 3×2", w, h)
	}
	want := []Pixel{
		{R: 255}, {B: 255}, {R: 255, G: 255, B: 255},
		{Any: true}, {}, {G: 128},
	}
	if !slices.Equal(got, want) {
		t.Errorf("pixels = %v, want %v", got, want)
	}
}

func TestParsePixelsErrors(t *testing.T) {
	tests := []struct {
		name, art, want string
	}{
		{"ragged rows", "RR\nR", "same width"},
		{"unknown character", "RX", "unknown pixel character"},
		{"empty", "  # only a comment\n", "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParsePixels(tt.art)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want one containing %q", err, tt.want)
			}
		})
	}
}

func TestEveryPixelCharacterIsDefined(t *testing.T) {
	for _, c := range "_RBGVSCMYKPQrgbvstuhapz" {
		if _, ok := PixelsByChar[byte(c)]; !ok {
			t.Errorf("missing character %q", c)
		}
	}
	if len(PixelsByChar) != 23 {
		t.Errorf("len(PixelsByChar) = %d, want 23", len(PixelsByChar))
	}
}
