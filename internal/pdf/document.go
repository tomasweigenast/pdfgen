package pdf

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/draw"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// Page is a laid-out page ready to be painted.
type Page struct {
	// Width and Height are the size of the margin box in CSS pixels.
	Width, Height float64
	// Bleed holds the widths of the bleed area, in CSS pixels.
	Bleed boxes.Sides
	// Box is the root of the box tree of the page.
	Box *boxes.PageBox
}

// NewPage returns the page of a page box, setting the transformation matrices
// of its boxes.
func NewPage(box *boxes.PageBox) *Page {
	boxes.SetTransformationMatrices(box)
	return &Page{
		Width:  box.MarginWidth(),
		Height: box.MarginHeight(),
		Bleed:  box.Bleed(),
		Box:    box,
	}
}

// Paint draws the page on s, whose unit is the point at the given scale: the
// number of points of a CSS pixel.
func (p *Page) Paint(s *stream.Stream, scale float64) {
	s.PushState()
	s.Transform(scale, 0, 0, scale, 0, 0)
	draw.DrawPage(s, p.Box)
	s.PopState()
}

// Document is a laid-out document: its pages and the metadata of its source.
type Document struct {
	Pages    []*Page
	Metadata *html.Metadata
}

// NewDocument returns the document made of pages. meta may be nil.
func NewDocument(pages []*boxes.PageBox, meta *html.Metadata) *Document {
	d := &Document{Pages: make([]*Page, len(pages)), Metadata: meta}
	for i, box := range pages {
		d.Pages[i] = NewPage(box)
	}
	return d
}
