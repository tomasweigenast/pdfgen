package boxes

import (
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/images"
)

// Background holds the resolved color and layers painted behind a box.
type Background struct {
	Color  props.Color
	Layers []BackgroundLayer
	Style  *computed.Style
}

// BackgroundLayer holds the size, position and clipping of one background image.
// Size and Position are relative to the positioning area, in CSS pixels.
type BackgroundLayer struct {
	// Source retains gradient descriptors until painting constructs the image.
	Source props.Image
	Image  images.Image

	Size, Position [2]float64
	Repeat         props.Repeat
	Unbounded      bool

	PaintingArea, PositioningArea Rect
	ClippedBoxes                  []RoundedBox
}
