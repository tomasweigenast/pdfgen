package testutil

import (
	"fmt"
	"strings"
)

// Pixel is an RGB color. When Any is set, it matches every color.
type Pixel struct {
	R, G, B uint8
	Any     bool
}

// PixelsByChar maps the characters of a pixel-art string to colors.
//
// "r" is not half red on purpose: in pixel strings it has better contrast
// with "B" than "R" does (compare "rBBBrrBrB" with "RBBBRRBRB").
var PixelsByChar = map[byte]Pixel{
	'_': {R: 255, G: 255, B: 255}, // white
	'R': {R: 255},                 // red
	'B': {B: 255},                 // blue
	'G': {G: 255},                 // lime green
	'V': {R: 191, B: 64},          // average of 1*B and 3*R
	'S': {R: 255, G: 63, B: 63},   // R above R above _
	'C': {G: 255, B: 255},         // cyan
	'M': {R: 255, B: 255},         // magenta
	'Y': {R: 255, G: 255},         // yellow
	'K': {},                       // black
	'P': {R: 50, G: 76, B: 103},   // blue CMYK with ICC
	'Q': {R: 63, G: 88, B: 110},   // blue CMYK without ICC
	'r': {R: 255},                 // red
	'g': {G: 128},                 // half green
	'b': {B: 128},                 // half blue
	'v': {R: 128, B: 128},         // average of B and R
	's': {R: 255, G: 127, B: 127}, // R above _
	't': {R: 127, G: 255, B: 127}, // G above _
	'u': {R: 128, B: 127},         // r above B above _
	'h': {R: 64, B: 64},           // half average of B and R
	'a': {B: 254},                 // R in lossy JPG
	'p': {R: 192, B: 63},          // R above R above B above _
	'z': {Any: true},              // any color
}

// ParsePixels parses a pixel-art string: one line per row, one character per
// pixel, "#" starting a comment and blank lines ignored. All rows must have
// the same width.
func ParsePixels(art string) (width, height int, pixels []Pixel, err error) {
	var rows []string
	for _, line := range strings.Split(art, "\n") {
		line, _, _ = strings.Cut(line, "#")
		if line = strings.TrimSpace(line); line != "" {
			rows = append(rows, line)
		}
	}
	if len(rows) == 0 {
		return 0, 0, nil, fmt.Errorf("pixel art is empty")
	}
	width = len(rows[0])
	for _, row := range rows {
		if len(row) != width {
			return 0, 0, nil, fmt.Errorf("all lines of pixels must have the same width")
		}
	}
	pixels = make([]Pixel, 0, width*len(rows))
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			p, ok := PixelsByChar[row[x]]
			if !ok {
				return 0, 0, nil, fmt.Errorf("unknown pixel character %q at (%d, %d)", row[x], x, y)
			}
			pixels = append(pixels, p)
		}
	}
	return width, len(rows), pixels, nil
}
