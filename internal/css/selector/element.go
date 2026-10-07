package selector

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// Namespace URLs that influence matching.
const (
	// XHTMLNamespace is the namespace of HTML elements.
	XHTMLNamespace = "http://www.w3.org/1999/xhtml"
	// XMLNamespace is the namespace of the xml: attributes.
	XMLNamespace = "http://www.w3.org/XML/1998/namespace"
)

// Element is the view of a document element needed to match selectors.
//
// Only element nodes take part in the tree: comments, text and processing
// instructions are never reported as parents or siblings. Relations that are
// absent must be returned as a nil interface value, not as a typed nil
// pointer. Implementations are expected to be comparable with ==, which is
// how elements are told apart.
type Element interface {
	// LocalName returns the tag name without any namespace prefix.
	LocalName() string
	// Namespace returns the namespace URL, or "" for no namespace.
	Namespace() string
	// Attr returns the value of the attribute with the given namespace URL
	// ("" for no namespace) and local name. Names are compared exactly.
	Attr(namespace, name string) (value string, ok bool)
	// ID returns the id attribute, or "" when there is none.
	ID() string
	// Classes returns the whitespace-separated tokens of the class
	// attribute.
	Classes() []string
	// Lang returns the language of the element in ASCII lower case, or ""
	// when unknown. See InheritedLang.
	Lang() string
	// Parent returns the parent element, or nil for the root.
	Parent() Element
	// PreviousSibling and NextSibling return the adjacent sibling elements,
	// or nil.
	PreviousSibling() Element
	NextSibling() Element
	// FirstChild returns the first child element, or nil.
	FirstChild() Element
	// Index returns the position among the sibling elements, counting from
	// 0. The root is at index 0.
	Index() int
	// IsEmpty reports whether the element has neither child elements nor
	// text (Selectors Level 4 §13.1). Comments are ignored.
	IsEmpty() bool
	// InHTMLDocument reports whether the document is parsed as HTML, in
	// which case type selectors and attribute names are matched ASCII
	// case-insensitively.
	InHTMLDocument() bool
}

// SplitWhitespace splits s at the whitespace characters of Selectors Level 4
// §3 (space, tab, CR, LF and form feed), dropping empty fields.
func SplitWhitespace(s string) []string {
	return strings.FieldsFunc(s, isSelectorSpace)
}

func isSelectorSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\f'
}

// ParseContentLanguage extracts the language from a Content-Language header
// or meta value. A value holding several languages yields "".
func ParseContentLanguage(value string) string {
	if strings.Contains(value, ",") {
		return ""
	}
	parts := SplitWhitespace(value)
	if len(parts) != 1 {
		return ""
	}
	return parts[0]
}

// InheritedLang implements the language lookup for Element.Lang
// (HTML Living Standard, "language"): xml:lang, then lang for HTML elements,
// then the parent's language. For the root, rootLang supplies the document
// default (a Content-Language meta element or the transport header); it may
// be nil. The result is in ASCII lower case.
func InheritedLang(el Element, rootLang func() string) string {
	if v, ok := el.Attr(XMLNamespace, "lang"); ok {
		return syntax.AsciiLower(v)
	}
	if el.InHTMLDocument() || el.Namespace() == XHTMLNamespace {
		if v, ok := el.Attr("", "lang"); ok {
			return syntax.AsciiLower(v)
		}
	}
	if p := el.Parent(); p != nil {
		return p.Lang()
	}
	if rootLang == nil {
		return ""
	}
	return syntax.AsciiLower(rootLang())
}
