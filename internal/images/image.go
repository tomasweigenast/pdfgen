package images

import (
	"fmt"

	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// Image is anything that can fill a box: a raster image, and later vector
// images and gradients.
type Image interface {
	// IntrinsicSize returns the natural width and height in CSS pixels and
	// the aspect ratio. Each may be auto when the image does not have it.
	IntrinsicSize(resolution, fontSize float64) (w, h, ratio units.MaybeFloat)
	// Draw paints the image into a w×h box at the origin of s. rendering is
	// the computed image-rendering value: only "auto" smooths the image.
	Draw(s *stream.Stream, w, h float64, rendering string)
}

// Orientation is the computed value of image-orientation (CSS Images 4 §5.2).
// The zero value is none.
type Orientation struct {
	// FromImage applies the orientation recorded in the file's Exif data.
	FromImage bool
	// Angle is a counter-clockwise rotation in degrees, a multiple of 90.
	Angle int
	// Flip mirrors the image horizontally after the rotation.
	Flip bool
}

// OrientationFromImage is the initial value of image-orientation.
var OrientationFromImage = Orientation{FromImage: true}

func (o Orientation) String() string {
	if o.FromImage {
		return "from-image"
	}
	return fmt.Sprintf("%d,%t", o.Angle, o.Flip)
}

// Options are the document-wide settings that shape how images are stored.
type Options struct {
	// OptimizeImages recompresses images as tightly as possible.
	OptimizeImages bool
	// JPEGQuality, from 1 to 95, recompresses JPEG images at that quality.
	// Zero embeds JPEG files unchanged.
	JPEGQuality int
	// DPI is the highest resolution, in dots per inch of the printed page,
	// that images keep; larger ones are scaled down. Zero means no limit.
	DPI int
}
