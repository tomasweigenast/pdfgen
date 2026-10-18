package pdfgen

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/layout"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/pdf"
)

// Document is a laid-out document: its pages, ready to be written as a PDF
// file as many times as needed.
type Document struct {
	// Pages are the pages of the document, in order.
	Pages []*Page
	// Metadata is the information in the head of the HTML source.
	Metadata Metadata

	doc *pdf.Document
}

// Page is a page of a Document.
type Page struct {
	// Width and Height are the size of the page in CSS pixels, with the
	// zoom of the PDF file not applied.
	Width, Height float64
}

// Metadata is the document information found in the head of an HTML
// document. Fields the document does not set are empty.
type Metadata struct {
	Title       string
	Description string
	Generator   string
	Lang        string
	Authors     []string
	Keywords    []string
	// Created and Modified are W3C date-time strings.
	Created  string
	Modified string
}

// Render lays out the document in pages. Invalid or unsupported content,
// such as a broken declaration or an image that cannot be loaded, is
// reported through the logging output and skipped. The error is fatal: the
// document cannot be rendered at all.
func (h *HTML) Render(opts Options) (*Document, error) {
	if err := opts.validateRender(); err != nil {
		return nil, err
	}
	fontConfig := opts.FontConfig
	if fontConfig == nil {
		fontConfig = NewFontConfiguration()
	}
	env := &cascade.Environment{
		MediaType:  h.mediaType,
		Fetcher:    h.fetcher,
		FontConfig: fontConfig.config,
	}
	logging.Progressf("Step 2 - Fetching and parsing CSS")
	collector := targets.NewCollector()
	cascadeOpts := cascade.Options{
		Environment:         env,
		PresentationalHints: opts.PresentationalHints,
		Targets:             collector,
	}
	for _, css := range opts.Stylesheets {
		sheet, err := css.stylesheet(env)
		if err != nil {
			return nil, fmt.Errorf("load stylesheet: %w", err)
		}
		cascadeOpts.UserStylesheets = append(cascadeOpts.UserStylesheets, sheet)
	}
	styleFor, err := cascade.ComputeAll(h.doc, cascadeOpts)
	if err != nil {
		return nil, fmt.Errorf("compute styles: %w", err)
	}

	logging.Progressf("Step 4 - Creating formatting structure")
	builder := &boxes.Builder{
		StyleFor: styleFor,
		GetImage: images.NewGetter(context.Background(), h.fetcher, nil, images.Options{
			OptimizeImages: opts.OptimizeImages,
			JPEGQuality:    opts.JPEGQuality,
			DPI:            opts.DPI,
		}),
		Targets:      collector,
		CounterStyle: env.CounterStyle,
	}
	root, err := builder.Build(h.doc)
	if err != nil {
		return nil, fmt.Errorf("build boxes: %w", err)
	}

	logging.Progressf("Step 5 - Creating layout")
	pages, err := layout.LayoutDocument(h.doc, root, layout.NewContext(builder), 0)
	if err != nil {
		return nil, fmt.Errorf("lay out document: %w", err)
	}
	return newDocument(pdf.NewDocument(pages, h.doc.Metadata())), nil
}

func newDocument(doc *pdf.Document) *Document {
	d := &Document{Pages: make([]*Page, len(doc.Pages)), Metadata: metadataOf(doc.Metadata), doc: doc}
	for i, p := range doc.Pages {
		d.Pages[i] = &Page{Width: p.Width, Height: p.Height}
	}
	return d
}

func metadataOf(m *html.Metadata) Metadata {
	if m == nil {
		return Metadata{}
	}
	deref := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	return Metadata{
		Title:       deref(m.Title),
		Description: deref(m.Description),
		Generator:   deref(m.Generator),
		Lang:        deref(m.Lang),
		Authors:     m.Authors,
		Keywords:    m.Keywords,
		Created:     deref(m.Created),
		Modified:    deref(m.Modified),
	}
}

// WritePDF writes the document as a PDF file to w.
func (d *Document) WritePDF(w io.Writer, opts Options) error {
	if err := opts.validateWrite(); err != nil {
		return err
	}
	logging.Progressf("Step 6 - Drawing pages and generating PDF")
	err := d.doc.WritePDF(w, pdf.Options{
		Zoom:         opts.Zoom,
		Uncompressed: opts.UncompressedPDF,
		PDFVersion:   opts.PDFVersion,
		Identifier:   opts.PDFIdentifier,
	})
	if err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}
	return nil
}

// WriteFile writes the document as a PDF file at path. A partial file is
// removed when writing fails.
func (d *Document) WriteFile(path string, opts Options) (err error) {
	if err := opts.validateWrite(); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create PDF file: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	out := bufio.NewWriter(f)
	if err = d.WritePDF(out, opts); err == nil {
		err = out.Flush()
	}
	if cerr := f.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("close PDF file: %w", cerr)
	}
	return err
}

// WritePDF renders the document and writes it as a PDF file to w.
func (h *HTML) WritePDF(w io.Writer, opts Options) error {
	doc, err := h.Render(opts)
	if err != nil {
		return err
	}
	return doc.WritePDF(w, opts)
}

// WriteFile renders the document and writes it as a PDF file at path.
func (h *HTML) WriteFile(path string, opts Options) error {
	doc, err := h.Render(opts)
	if err != nil {
		return err
	}
	return doc.WriteFile(path, opts)
}
