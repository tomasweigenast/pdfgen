package pdfgen

import (
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Fetcher retrieves the resources a document refers to: stylesheets,
// images, fonts and other documents. Implement it to serve them from
// somewhere other than the file system and the network.
type Fetcher = urls.Fetcher

// Response is what a Fetcher returns for a URL.
type Response = urls.Response

// DefaultFetcher fetches file:, http:, https: and data: URLs.
type DefaultFetcher = urls.DefaultFetcher

// NewDefaultFetcher returns a fetcher with a 10 second timeout that follows
// redirects.
func NewDefaultFetcher() *DefaultFetcher { return urls.NewDefaultFetcher() }

// DefaultMediaType is the media type that media queries are evaluated for
// unless WithMediaType says otherwise.
const DefaultMediaType = "print"

// SourceOption adjusts how an HTML or CSS source is read.
type SourceOption func(*sourceConfig)

type sourceConfig struct {
	baseURL   string
	encoding  string
	mediaType string
	fetcher   Fetcher
}

func newSourceConfig(opts []SourceOption) *sourceConfig {
	c := &sourceConfig{}
	for _, o := range opts {
		o(c)
	}
	return c
}

// fetcherOrDefault returns the configured fetcher, or a new default one.
func (c *sourceConfig) fetcherOrDefault() Fetcher {
	if c.fetcher != nil {
		return c.fetcher
	}
	return urls.NewDefaultFetcher()
}

// WithBaseURL sets the URL that relative references of the source resolve
// against. A file path is accepted and converted to a file URL.
func WithBaseURL(url string) SourceOption {
	return func(c *sourceConfig) { c.baseURL = url }
}

// WithEncoding forces the character encoding of the source, overriding the
// one announced by the transport or declared in the document.
func WithEncoding(label string) SourceOption {
	return func(c *sourceConfig) { c.encoding = label }
}

// WithFetcher sets the fetcher used to load the source and its resources.
// The default is a DefaultFetcher.
func WithFetcher(f Fetcher) SourceOption {
	return func(c *sourceConfig) { c.fetcher = f }
}

// WithMediaType sets the media type that media queries are evaluated for,
// "print" by default.
func WithMediaType(mediaType string) SourceOption {
	return func(c *sourceConfig) { c.mediaType = mediaType }
}
