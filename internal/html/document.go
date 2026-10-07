package html

import (
	"fmt"
	"io"
	"strings"

	nethtml "golang.org/x/net/html"

	"github.com/tomasweigenast/pdfgen/internal/css/selector"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Namespace URLs of the elements and attributes found in HTML documents.
const (
	XHTMLNamespace  = selector.XHTMLNamespace
	SVGNamespace    = "http://www.w3.org/2000/svg"
	MathMLNamespace = "http://www.w3.org/1998/Math/MathML"
	XLinkNamespace  = "http://www.w3.org/1999/xlink"
	XMLNamespace    = selector.XMLNamespace
	XMLNSNamespace  = "http://www.w3.org/2000/xmlns/"
)

// Document is a parsed HTML document.
type Document struct {
	// Root is the html element.
	Root *Element
	// BaseURL resolves the relative URLs of the document: the base URL of
	// the source, adjusted by the first base element that has an href. It
	// is "" when the document has no base.
	BaseURL string
	// Encoding is the canonical name of the encoding the bytes were
	// decoded with, or "" for a document parsed from a string.
	Encoding string

	elements    []*Element
	byNode      map[*nethtml.Node]*Element
	contentLang string
	rootLang    *string
}

// Option adjusts how a document is parsed.
type Option func(*options)

type options struct {
	transportEncoding string
	contentLanguage   string
}

// WithTransportEncoding sets the charset announced by the transport, such
// as the one in an HTTP Content-Type header. It ranks below an explicit
// encoding and above a meta declaration.
func WithTransportEncoding(label string) Option {
	return func(o *options) { o.transportEncoding = label }
}

// WithContentLanguage sets the language announced by the transport, such as
// an HTTP Content-Language header. It is the language of the document unless
// a meta element says otherwise.
func WithContentLanguage(value string) Option {
	return func(o *options) { o.contentLanguage = value }
}

// Parse reads an HTML document from r. The character encoding is found as
// described in DetectEncoding; a non-empty encoding forces it, and
// baseURL is the base URL of the source.
func Parse(r io.Reader, encoding, baseURL string, opts ...Option) (*Document, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read HTML: %w", err)
	}
	text, name, err := decode(data, encoding, o.transportEncoding)
	if err != nil {
		return nil, fmt.Errorf("decode HTML as %s: %w", name, err)
	}
	doc, err := parseText(text, baseURL, o)
	if err != nil {
		return nil, err
	}
	doc.Encoding = name
	return doc, nil
}

// ParseString parses HTML that is already text, so that no encoding
// detection takes place.
func ParseString(s, baseURL string, opts ...Option) (*Document, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return parseText(s, baseURL, o)
}

func parseText(s, baseURL string, o options) (*Document, error) {
	root, err := nethtml.ParseWithOptions(strings.NewReader(s), nethtml.ParseOptionEnableScripting(false))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	doc := &Document{byNode: map[*nethtml.Node]*Element{}, contentLang: o.contentLanguage}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode {
			doc.Root = doc.build(c, nil, 0)
			break
		}
	}
	if doc.Root == nil {
		return nil, fmt.Errorf("parse HTML: no root element")
	}
	doc.BaseURL = doc.findBaseURL(baseURL)
	return doc, nil
}

func (d *Document) build(n *nethtml.Node, parent *Element, index int) *Element {
	e := &Element{doc: d, node: n, parent: parent, index: index, namespace: elementNamespace(n.Namespace)}
	d.byNode[n] = e
	d.elements = append(d.elements, e)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode {
			e.children = append(e.children, d.build(c, e, len(e.children)))
		}
	}
	return e
}

func elementNamespace(ns string) string {
	switch ns {
	case "":
		return XHTMLNamespace
	case "svg":
		return SVGNamespace
	case "math":
		return MathMLNamespace
	}
	return ns
}

// findBaseURL applies the first base element with a non-empty href to the
// base URL of the source (HTML Living Standard §2.4.1, "document base URL").
func (d *Document) findBaseURL(fallback string) string {
	for _, e := range d.elements {
		if !e.isHTML("base") {
			continue
		}
		if href := strings.TrimSpace(e.Get("href")); href != "" {
			return urls.Join(fallback, href)
		}
		break
	}
	return fallback
}

// Elements returns every element in document order. The slice must not be
// modified.
func (d *Document) Elements() []*Element { return d.elements }

// Element returns the wrapper of an element node of the parse tree, or nil
// for any other node.
func (d *Document) Element(n *nethtml.Node) *Element { return d.byNode[n] }

// Language returns the language of the document as declared by a
// Content-Language meta element or the transport, in lower case, or "".
func (d *Document) Language() string {
	if d.rootLang != nil {
		return *d.rootLang
	}
	var lang *string
	for _, e := range d.elements {
		if !e.isHTML("meta") {
			continue
		}
		if !strings.EqualFold(e.Get("http-equiv"), "content-language") {
			continue
		}
		v, ok := e.Lookup("content")
		if !ok {
			lang = nil
			continue
		}
		p := selector.ParseContentLanguage(v)
		if p == "" {
			lang = nil
			continue
		}
		lang = &p
	}
	var result string
	if lang != nil {
		result = *lang
	} else {
		result = selector.ParseContentLanguage(d.contentLang)
	}
	result = syntax.AsciiLower(result)
	d.rootLang = &result
	return result
}
