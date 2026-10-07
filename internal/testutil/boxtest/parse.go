package boxtest

import (
	"context"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Options change how a test document is prepared.
type Options struct {
	// BaseURL replaces the base URL of test documents, a file in the test
	// resources. NoBaseURL leaves the document without one.
	BaseURL   string
	NoBaseURL bool
	// DefaultUA uses the user agent stylesheets of the engine instead of
	// the test one.
	DefaultUA bool
	// PresentationalHints turns presentational HTML attributes into
	// styles.
	PresentationalHints bool
	// Stylesheets are user stylesheets.
	Stylesheets []string
}

// Document is an HTML document with its computed styles and a builder for
// its box tree.
type Document struct {
	HTML     *html.Document
	StyleFor *cascade.StyleFor
	Builder  *boxes.Builder
}

// NewDocument parses an HTML document and computes its styles, by default
// with the test user agent stylesheet and its test font.
func NewDocument(t testing.TB, source string, opts Options) *Document {
	t.Helper()
	baseURL := opts.BaseURL
	switch {
	case opts.NoBaseURL:
		baseURL = ""
	case baseURL == "":
		baseURL = csstest.BaseURL(t)
	}
	doc, err := html.ParseString(source, baseURL)
	if err != nil {
		t.Fatalf("parse HTML: %v", err)
	}

	env := csstest.Environment()
	collector := targets.NewCollector()
	cascadeOpts := cascade.Options{
		Environment:         env,
		PresentationalHints: opts.PresentationalHints,
		Targets:             collector,
	}
	if !opts.DefaultUA {
		cascadeOpts.UAStylesheets = []*cascade.Stylesheet{csstest.UAStylesheet(t, env)}
	}
	for _, css := range opts.Stylesheets {
		cascadeOpts.UserStylesheets = append(cascadeOpts.UserStylesheets, cascade.ParseStylesheet(css, baseURL, env))
	}
	styleFor, err := cascade.ComputeAll(doc, cascadeOpts)
	if err != nil {
		t.Fatalf("compute styles: %v", err)
	}

	fetcher := urls.NewDefaultFetcher()
	return &Document{
		HTML:     doc,
		StyleFor: styleFor,
		Builder: &boxes.Builder{
			StyleFor:     styleFor,
			GetImage:     images.NewGetter(context.Background(), fetcher, nil, images.Options{}),
			Targets:      collector,
			CounterStyle: env.CounterStyle,
		},
	}
}

// Build builds the complete box tree of the document and checks that it
// follows the rules about what boxes can contain.
func (d *Document) Build(t testing.TB) boxes.Box {
	t.Helper()
	box, err := d.Builder.Build(d.HTML)
	if err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	SanityChecks(t, box)
	return box
}

// Parse builds the boxes of the root element of an HTML document, without
// processing white space or adding anonymous boxes. The root element must
// make exactly one box.
func Parse(t testing.TB, source string) boxes.Box {
	t.Helper()
	d := NewDocument(t, source, Options{})
	list := d.Builder.ElementToBox(d.HTML.Root, nil)
	if err := d.Builder.Err(); err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("root element made %d boxes, want 1", len(list))
	}
	return list[0]
}

// ParseAll builds the complete box tree of a document and checks that it
// follows the rules about what boxes can contain.
func ParseAll(t testing.TB, source string) boxes.Box {
	t.Helper()
	return NewDocument(t, source, Options{}).Build(t)
}

// ParseAllWith is ParseAll with options.
func ParseAllWith(t testing.TB, source string, opts Options) boxes.Box {
	t.Helper()
	return NewDocument(t, source, opts).Build(t)
}

// RenderPages lays out a document and returns its page boxes.
func RenderPages(t testing.TB, html string) []*boxes.PageBox {
	t.Helper()
	t.Skip("rendering pipeline not implemented")
	return nil
}
