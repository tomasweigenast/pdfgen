package pdf

import (
	"fmt"
	"io"
	"math"

	"github.com/tomasweigenast/pdfgen/internal/pdf/anchors"
	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts"
	"github.com/tomasweigenast/pdfgen/internal/pdf/metadata"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// pointsPerPixel converts CSS pixels to PDF points: 72 points per inch over
// 96 pixels per inch.
const pointsPerPixel = 0.75

// maxBleedBox is the largest distance in points between the TrimBox and the
// BleedBox of a page.
const maxBleedBox = 10

// Options configure the PDF file. The zero value is valid.
type Options struct {
	// Zoom scales every length of the pages; zero means 1.
	Zoom float64
	// Uncompressed writes streams and objects as they are, for debugging.
	Uncompressed bool
	// PDFVersion is the version of the file header, such as "1.7"; empty
	// means 1.7.
	PDFVersion string
	// Identifier is the first element of the file identifier; empty derives
	// it from the content.
	Identifier []byte
}

// WritePDF writes the document as a PDF file to w.
func (d *Document) WritePDF(w io.Writer, opts Options) error {
	return Generate(d, w, opts)
}

// Generate writes doc as a PDF file to w.
func Generate(doc *Document, w io.Writer, opts Options) error {
	zoom := opts.Zoom
	if zoom == 0 {
		zoom = 1
	}
	scale := zoom * pointsPerPixel
	compress := !opts.Uncompressed

	file := writer.New()
	colorSpaces := file.Add(stream.LabColorSpaces())
	resourcesRef := file.Reserve()
	res := stream.NewResources(colorSpaces)
	res.Fonts = fonts.NewRegistry(false)

	data := make([]*anchors.Data, len(doc.Pages))
	for i, page := range doc.Pages {
		data[i] = &page.Data
	}
	resolved := anchors.ResolveLinks(data)

	var destinations []anchors.NamedDestination
	for i, page := range doc.Pages {
		dict := paintPage(file, page, res, resourcesRef, scale, compress)
		m := pageMatrix(scale, page.Height)
		if annots := anchors.AddLinks(file, resolved[i].Links, m); len(annots) > 0 {
			dict.Set("Annots", annots)
		}
		ref := file.AddPage(dict)
		destinations = append(destinations, anchors.Destinations(resolved[i].Destinations, ref, m)...)
	}
	anchors.AddOutlines(file, doc.BookmarkTree(scale, true), file.PageRefs())
	anchors.SetNameTree(file.Catalog(), destinations)
	metadata.SetInfo(file.Info(), file.Catalog(), doc.Metadata)

	fontDict, err := res.Fonts.Build(file, fonts.BuildOptions{Compress: compress, PDFVersion: opts.PDFVersion})
	if err != nil {
		return fmt.Errorf("embed fonts: %w", err)
	}
	res.Font = file.Add(fontDict)
	if err := res.Embed(file, resourcesRef); err != nil {
		return fmt.Errorf("embed resources: %w", err)
	}

	err = file.Write(w, writer.WriteOptions{
		Version:    opts.PDFVersion,
		Identifier: opts.Identifier,
		Compress:   compress,
	})
	if err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}
	return nil
}

// paintPage draws page and returns its page dictionary, whose MediaBox is the
// bleed area, TrimBox the page box and BleedBox a margin of at most 10 points
// around the TrimBox.
func paintPage(file *writer.PDF, page *Page, res *stream.Resources, resourcesRef writer.Ref,
	scale float64, compress bool) *writer.Dict {
	bleed := page.Bleed
	left, top := -scale*bleed.Left, -scale*bleed.Top
	right := left + scale*(page.Width+bleed.Left+bleed.Right)
	bottom := top + scale*(page.Height+bleed.Top+bleed.Bottom)

	s := stream.New(res, stream.Options{
		PageRectangle: [4]float64{left / scale, top / scale, (right - left) / scale, (bottom - top) / scale},
		Compress:      compress,
	})
	// The origin of the page is its top-left corner, with y going down.
	s.Transform(1, 0, 0, -1, 0, page.Height*scale)
	page.Paint(s, scale)

	trimLeft, trimTop := left+scale*bleed.Left, top+scale*bleed.Top
	trimRight, trimBottom := right-scale*bleed.Right, bottom-scale*bleed.Bottom
	margin := func(width float64) float64 { return math.Min(maxBleedBox, scale*width) }

	return writer.NewDict(
		"Type", writer.Name("Page"),
		"MediaBox", box(left, top, right, bottom),
		"TrimBox", box(trimLeft, trimTop, trimRight, trimBottom),
		"BleedBox", box(
			trimLeft-margin(bleed.Left), trimTop-margin(bleed.Top),
			trimRight+margin(bleed.Right), trimBottom+margin(bleed.Bottom)),
		"Contents", file.Add(s.Object()),
		"Resources", resourcesRef,
	)
}

func box(left, top, right, bottom float64) writer.Array {
	return writer.Array{writer.Real(left), writer.Real(top), writer.Real(right), writer.Real(bottom)}
}
