package pdfgen

import (
	"errors"
	"fmt"
	"regexp"
)

// Options configure the rendering of a document and the PDF file written
// from it. The zero value is valid.
//
// Stylesheets, PresentationalHints, OptimizeImages, JPEGQuality, DPI and
// FontConfig apply when the document is rendered; Zoom, UncompressedPDF,
// PDFVersion and PDFIdentifier when the PDF file is written.
type Options struct {
	// Stylesheets are applied after the stylesheets of the document, as user
	// styles.
	Stylesheets []*CSS
	// Zoom scales the pages of the PDF file; zero means 1.
	Zoom float64
	// UncompressedPDF writes streams and objects as they are, which makes the
	// file larger and easier to debug.
	UncompressedPDF bool
	// PDFVersion is the version in the header of the file, such as "1.7".
	// Empty means 1.7.
	PDFVersion string
	// PDFIdentifier is the first element of the file identifier. Empty
	// derives it from the content.
	PDFIdentifier []byte
	// PresentationalHints turns presentational HTML attributes, such as
	// width and bgcolor, into styles.
	PresentationalHints bool
	// OptimizeImages recompresses images as tightly as possible, without
	// losing quality.
	OptimizeImages bool
	// JPEGQuality, from 1 to 95, recompresses JPEG images at that quality.
	// Zero embeds them unchanged.
	JPEGQuality int
	// DPI is the highest resolution, in dots per inch of the printed page,
	// that images keep; larger ones are scaled down. Zero means no limit.
	DPI int
	// FontConfig provides the fonts. Nil makes a new configuration for the
	// rendering.
	FontConfig *FontConfiguration
}

var pdfVersionPattern = regexp.MustCompile(`^[1-9]\.[0-9]$`)

// validateRender checks the options used when rendering.
func (o *Options) validateRender() error {
	if o.JPEGQuality < 0 || o.JPEGQuality > 95 {
		return fmt.Errorf("invalid JPEG quality %d: must be between 1 and 95, or 0 to keep JPEG files", o.JPEGQuality)
	}
	if o.DPI < 0 {
		return fmt.Errorf("invalid DPI %d: must not be negative", o.DPI)
	}
	for _, css := range o.Stylesheets {
		if css == nil {
			return errors.New("nil stylesheet in options")
		}
	}
	return nil
}

// validateWrite checks the options used when writing the file.
func (o *Options) validateWrite() error {
	if o.Zoom < 0 {
		return fmt.Errorf("invalid zoom %g: must not be negative", o.Zoom)
	}
	if o.PDFVersion != "" && !pdfVersionPattern.MatchString(o.PDFVersion) {
		return fmt.Errorf("invalid PDF version %q: want a version such as 1.7", o.PDFVersion)
	}
	return nil
}
