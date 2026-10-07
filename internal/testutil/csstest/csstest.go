// Package csstest computes the styles of test documents the way the
// rendering tests do: with the test user agent stylesheet, which declares
// the test font, and the test resources as base URL.
package csstest

import (
	"os"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// BaseURL is the base URL of test documents: a file in the resources
// directory.
func BaseURL(t testing.TB) string {
	t.Helper()
	return testutil.ResourceURL(t, "<test>")
}

// Environment returns an environment with a new font configuration, whose
// registries are filled by the stylesheets loaded with it.
func Environment() *cascade.Environment {
	return &cascade.Environment{FontConfig: fonts.NewFontConfiguration()}
}

// UAStylesheet loads the test user agent stylesheet, registering the test
// font in the font configuration of env.
func UAStylesheet(t testing.TB, env *cascade.Environment) *cascade.Stylesheet {
	t.Helper()
	path := testutil.ResourcePath(t, "test_ua.css")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test stylesheet: %v", err)
	}
	return cascade.ParseStylesheet(string(data), testutil.ResourceURL(t, "test_ua.css"), env)
}

// Parse parses an HTML document with the test base URL.
func Parse(t testing.TB, source string) *html.Document {
	t.Helper()
	doc, err := html.ParseString(source, BaseURL(t))
	if err != nil {
		t.Fatalf("parse HTML: %v", err)
	}
	return doc
}

// StyleFor computes the styles of a document with the test user agent
// stylesheet. Options left empty get the test environment.
func StyleFor(t testing.TB, doc *html.Document, opts cascade.Options) *cascade.StyleFor {
	t.Helper()
	if opts.Environment == nil {
		opts.Environment = Environment()
	}
	if opts.UAStylesheets == nil {
		opts.UAStylesheets = []*cascade.Stylesheet{UAStylesheet(t, opts.Environment)}
	}
	sf, err := cascade.ComputeAll(doc, opts)
	if err != nil {
		t.Fatalf("compute styles: %v", err)
	}
	return sf
}

// Element returns the first element with the given local name in tree
// order, failing the test if there is none.
func Element(t testing.TB, doc *html.Document, name string) *html.Element {
	t.Helper()
	for _, el := range doc.Elements() {
		if el.LocalName() == name {
			return el
		}
	}
	t.Fatalf("no %s element", name)
	return nil
}

// Elements returns the elements with the given local name in tree order.
func Elements(doc *html.Document, name string) []*html.Element {
	var out []*html.Element
	for _, el := range doc.Elements() {
		if el.LocalName() == name {
			out = append(out, el)
		}
	}
	return out
}
