package pdfgen

import (
	"context"
	"fmt"
	"io"

	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// HTML is a parsed HTML document, ready to be rendered. Rendering does not
// modify it, so it can be rendered more than once, with different options.
type HTML struct {
	doc       *html.Document
	fetcher   Fetcher
	mediaType string
}

// HTMLFromString parses HTML held in a string. Relative references resolve
// against the base URL given with WithBaseURL, if any.
func HTMLFromString(s string, o ...SourceOption) (*HTML, error) {
	c := newSourceConfig(o)
	logging.Progressf("Step 1 - Parsing HTML")
	doc, err := html.ParseString(s, baseURLOf(c.baseURL))
	if err != nil {
		return nil, err
	}
	return newHTML(doc, c), nil
}

// HTMLFromFile reads and parses an HTML file. Unless WithBaseURL says
// otherwise, relative references resolve against the file.
func HTMLFromFile(path string, o ...SourceOption) (*HTML, error) {
	c := newSourceConfig(o)
	src, err := urls.OpenPath(path, c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	return parseSource(src, c)
}

// HTMLFromURL fetches and parses an HTML document. Unless WithBaseURL says
// otherwise, relative references resolve against the final URL of the
// response, after redirects.
func HTMLFromURL(url string, o ...SourceOption) (*HTML, error) {
	c := newSourceConfig(o)
	c.fetcher = c.fetcherOrDefault()
	src, err := urls.OpenURL(context.Background(), c.fetcher, url, c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch document: %w", err)
	}
	return parseSource(src, c)
}

// HTMLFromReader reads and parses an HTML document from r. Without
// WithBaseURL the document has no base URL, so relative references cannot
// be resolved.
func HTMLFromReader(r io.Reader, o ...SourceOption) (*HTML, error) {
	c := newSourceConfig(o)
	return parseSource(urls.FromReader(r, "", c.baseURL), c)
}

// HTMLGuess opens s as a URL when it starts with a scheme, and as a file
// path otherwise.
func HTMLGuess(s string, o ...SourceOption) (*HTML, error) {
	c := newSourceConfig(o)
	c.fetcher = c.fetcherOrDefault()
	src, err := urls.Guess(context.Background(), c.fetcher, s, c.baseURL)
	if err != nil {
		return nil, err
	}
	return parseSource(src, c)
}

// parseSource parses the body of src and closes it.
func parseSource(src *urls.Source, c *sourceConfig) (*HTML, error) {
	defer src.Body.Close()
	logging.Progressf("Step 1 - Parsing HTML - %s", src.BaseURL)
	doc, err := html.Parse(src.Body, c.encoding, src.BaseURL, html.WithTransportEncoding(src.ProtocolEncoding))
	if err != nil {
		return nil, err
	}
	return newHTML(doc, c), nil
}

func newHTML(doc *html.Document, c *sourceConfig) *HTML {
	mediaType := c.mediaType
	if mediaType == "" {
		mediaType = DefaultMediaType
	}
	return &HTML{doc: doc, fetcher: c.fetcherOrDefault(), mediaType: mediaType}
}

// baseURLOf turns a base URL option into a URL: file paths become file URLs
// and an empty value stays empty.
func baseURLOf(base string) string {
	if base == "" {
		return ""
	}
	return urls.EnsureURL(base)
}
