package pdfgen

import (
	"context"
	"fmt"
	"io"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// CSS is a stylesheet given by the user, to pass in Options.Stylesheets. It
// is parsed when a document is rendered, so that its @font-face rules
// register their fonts in the font configuration of that rendering.
type CSS struct {
	text      string
	baseURL   string
	mediaType string
	fetcher   Fetcher
}

// CSSFromString makes a stylesheet from a string. Relative URLs in it
// resolve against the base URL given with WithBaseURL, if any.
func CSSFromString(s string, o ...SourceOption) *CSS {
	c := newSourceConfig(o)
	return &CSS{text: s, baseURL: baseURLOf(c.baseURL), mediaType: c.mediaType, fetcher: c.fetcher}
}

// CSSFromFile reads a stylesheet from a file. Unless WithBaseURL says
// otherwise, relative URLs in it resolve against the file.
func CSSFromFile(path string, o ...SourceOption) (*CSS, error) {
	c := newSourceConfig(o)
	src, err := urls.OpenPath(path, c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("open stylesheet: %w", err)
	}
	return readCSS(src, c)
}

// CSSFromURL fetches a stylesheet. Unless WithBaseURL says otherwise,
// relative URLs in it resolve against the final URL of the response.
func CSSFromURL(url string, o ...SourceOption) (*CSS, error) {
	c := newSourceConfig(o)
	src, err := urls.OpenURL(context.Background(), c.fetcherOrDefault(), url, c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch stylesheet: %w", err)
	}
	return readCSS(src, c)
}

// CSSFromReader reads a stylesheet from r.
func CSSFromReader(r io.Reader, o ...SourceOption) (*CSS, error) {
	c := newSourceConfig(o)
	return readCSS(urls.FromReader(r, "", c.baseURL), c)
}

// CSSGuess opens s as a URL when it starts with a scheme, and as a file path
// otherwise.
func CSSGuess(s string, o ...SourceOption) (*CSS, error) {
	c := newSourceConfig(o)
	src, err := urls.Guess(context.Background(), c.fetcherOrDefault(), s, c.baseURL)
	if err != nil {
		return nil, err
	}
	return readCSS(src, c)
}

// readCSS decodes the body of src as browsers do, honoring a byte order
// mark, the transport encoding and @charset, and closes it.
func readCSS(src *urls.Source, c *sourceConfig) (*CSS, error) {
	defer src.Body.Close()
	data, err := io.ReadAll(src.Body)
	if err != nil {
		return nil, fmt.Errorf("read stylesheet: %w", err)
	}
	protocol := src.ProtocolEncoding
	if c.encoding != "" {
		protocol = c.encoding
	}
	text, _ := syntax.DecodeStylesheetBytes(data, protocol, nil)
	return &CSS{text: text, baseURL: src.BaseURL, mediaType: c.mediaType, fetcher: c.fetcher}, nil
}

// stylesheet parses the stylesheet in env. A stylesheet made with its own
// media type or fetcher keeps them for its @media and @import rules. The
// error is a fatal fetching error met while importing other stylesheets.
func (c *CSS) stylesheet(env *cascade.Environment) (*cascade.Stylesheet, error) {
	local := *env
	if c.mediaType != "" {
		local.MediaType = c.mediaType
	}
	if c.fetcher != nil {
		local.Fetcher = c.fetcher
	}
	sheet := cascade.ParseStylesheet(c.text, c.baseURL, &local)
	if err := local.Err(); err != nil {
		return nil, err
	}
	return sheet, nil
}
