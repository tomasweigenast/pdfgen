package pdf

import (
	"slices"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/draw"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/anchors"
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
	// Anchors, Links and Bookmarks are the positions of the page that the
	// document structures point at, in CSS pixels from its top-left corner.
	anchors.Data
}

// NewPage returns the page of a page box, setting the transformation matrices
// of its boxes and gathering its anchors, links and bookmarks.
func NewPage(box *boxes.PageBox) *Page {
	boxes.SetTransformationMatrices(box)
	return &Page{
		Width:  box.MarginWidth(),
		Height: box.MarginHeight(),
		Bleed:  box.Bleed(),
		Box:    box,
		Data:   anchors.Gather(box),
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

// Copy returns a document made of a subset of the pages of d, with the same
// metadata. Nil or empty pages mean every page.
func (d *Document) Copy(pages []*Page) *Document {
	if len(pages) == 0 {
		pages = d.Pages
	}
	return &Document{Pages: slices.Clone(pages), Metadata: d.Metadata}
}

// BookmarkNode is an entry of the outline of the document. Page is the
// zero-based index of the page that holds the position (X, Y).
type BookmarkNode = anchors.Node

// BookmarkTree returns the outline of the document. Positions are scaled by
// scale; with transformPages they are also flipped to have the origin at the
// bottom-left corner of the page, as the PDF user space does.
func (d *Document) BookmarkTree(scale float64, transformPages bool) []BookmarkNode {
	tree := anchors.NewTreeBuilder()
	for i, page := range d.Pages {
		m := matrix.New(scale, 0, 0, scale, 0, 0)
		if transformPages {
			m = pageMatrix(scale, page.Height)
		}
		tree.AddPage(page.Bookmarks, i, m)
	}
	return tree.Tree()
}

// pageMatrix maps the CSS pixels of a page, whose origin is its top-left
// corner, to the PDF user space of a page of the given height.
func pageMatrix(scale, height float64) matrix.Matrix {
	return matrix.New(scale, 0, 0, -scale, 0, height*scale)
}
