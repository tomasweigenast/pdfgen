package boxtest

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/layout"
)

// Option changes how FakeHTML prepares a document.
type Option func(*Options)

// WithBaseURL sets the base URL of the document.
func WithBaseURL(url string) Option { return func(o *Options) { o.BaseURL = url } }

// WithoutBaseURL leaves the document without a base URL.
func WithoutBaseURL() Option { return func(o *Options) { o.NoBaseURL = true } }

// WithDefaultUA uses the user agent stylesheets of the engine instead of
// the test one.
func WithDefaultUA() Option { return func(o *Options) { o.DefaultUA = true } }

// WithPresentationalHints turns presentational HTML attributes into styles.
func WithPresentationalHints() Option { return func(o *Options) { o.PresentationalHints = true } }

// WithStylesheet adds a user stylesheet.
func WithStylesheet(css string) Option {
	return func(o *Options) { o.Stylesheets = append(o.Stylesheets, css) }
}

// FakeHTML prepares a document the way the rendering tests need it: with
// the test user agent stylesheet, which declares the test font, and the
// test resources as base URL.
func FakeHTML(t testing.TB, source string, opts ...Option) *Document {
	t.Helper()
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	return NewDocument(t, source, options)
}

// RenderPages builds the boxes of the document and lays them out in pages,
// failing the test on a fatal error of the layout or when a page breaks the
// rules about what boxes can contain.
func (d *Document) RenderPages(t testing.TB) []*boxes.PageBox {
	t.Helper()
	root := d.Build(t)
	pages, err := layout.LayoutDocument(d.HTML, root, layout.NewContext(d.Builder), 0)
	if err != nil {
		t.Fatalf("lay out pages: %v", err)
	}
	for _, page := range pages {
		SanityChecks(t, page)
	}
	return pages
}

// RenderPages lays out an HTML document with the test user agent stylesheet
// and returns its page boxes.
func RenderPages(t testing.TB, source string) []*boxes.PageBox {
	t.Helper()
	return FakeHTML(t, source).RenderPages(t)
}
