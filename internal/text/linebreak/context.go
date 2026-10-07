package linebreak

import (
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
	"github.com/tomasweigenast/pdfgen/internal/text/hyphen"
	"github.com/tomasweigenast/pdfgen/internal/text/shape"
)

// Context holds what text layout shares across a document: the fonts,
// the shaper and the hyphenation dictionaries in use. It is not safe for
// concurrent use.
type Context struct {
	Fonts  *fonts.FontConfiguration
	Shaper shape.Shaper

	hyphenators map[hyphenatorKey]*hyphen.Hyphenator
	charWidths  map[faceSize]int
}

type hyphenatorKey struct {
	lang          string
	before, after int
}

type faceSize struct {
	face *fonts.Face
	size int32
}

// NewContext returns a context laying out text with the fonts of fc.
func NewContext(fc *fonts.FontConfiguration) *Context {
	return &Context{
		Fonts:       fc,
		Shaper:      shape.GoText{},
		hyphenators: map[hyphenatorKey]*hyphen.Hyphenator{},
		charWidths:  map[faceSize]int{},
	}
}

// hyphenator returns the dictionary for a language, or nil when there is
// none.
func (c *Context) hyphenator(lang string, before, after int) *hyphen.Hyphenator {
	key := hyphenatorKey{lang, before, after}
	if h, ok := c.hyphenators[key]; ok {
		return h
	}
	h, err := hyphen.New(lang, before, after)
	if err != nil {
		h = nil
	}
	c.hyphenators[key] = h
	return h
}
