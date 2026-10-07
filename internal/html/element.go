package html

import (
	"strings"

	nethtml "golang.org/x/net/html"

	"github.com/tomasweigenast/pdfgen/internal/css/selector"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// Element is an element of a Document. It implements selector.Element, and
// pointers to it can be compared to tell elements apart.
type Element struct {
	doc       *Document
	node      *nethtml.Node
	parent    *Element
	children  []*Element
	index     int
	namespace string
	lang      *string
	classes   []string
}

var _ selector.Element = (*Element)(nil)

// Document returns the document the element belongs to.
func (e *Element) Document() *Document { return e.doc }

// Node returns the element in the parse tree, which also holds the text and
// comment nodes that elements leave out.
func (e *Element) Node() *nethtml.Node { return e.node }

// LocalName returns the tag name without namespace. HTML tag names are in
// lower case.
func (e *Element) LocalName() string { return e.node.Data }

// Namespace returns the namespace URL of the element. Elements of the HTML
// syntax are in the XHTML namespace unless the parser put them in the SVG or
// MathML one.
func (e *Element) Namespace() string { return e.namespace }

// Attr implements selector.Element.
func (e *Element) Attr(namespace, name string) (string, bool) {
	for _, a := range e.node.Attr {
		if a.Key == name && attrNamespace(a.Namespace) == namespace {
			return a.Val, true
		}
	}
	return "", false
}

func attrNamespace(prefix string) string {
	switch prefix {
	case "":
		return ""
	case "xlink":
		return XLinkNamespace
	case "xml":
		return XMLNamespace
	case "xmlns":
		return XMLNSNamespace
	}
	return prefix
}

// Lookup returns the value of an attribute that has no namespace.
func (e *Element) Lookup(name string) (string, bool) { return e.Attr("", name) }

// Get returns the value of an attribute that has no namespace, or "" when
// it is absent.
func (e *Element) Get(name string) string {
	v, _ := e.Attr("", name)
	return v
}

// ID implements selector.Element.
func (e *Element) ID() string { return e.Get("id") }

// Classes implements selector.Element.
func (e *Element) Classes() []string {
	if e.classes == nil {
		e.classes = selector.SplitWhitespace(e.Get("class"))
		if e.classes == nil {
			e.classes = []string{}
		}
	}
	return e.classes
}

// Lang implements selector.Element.
func (e *Element) Lang() string {
	if e.lang == nil {
		l := selector.InheritedLang(e, nil)
		e.lang = &l
	}
	return *e.lang
}

// Parent implements selector.Element.
func (e *Element) Parent() selector.Element {
	if e.parent == nil {
		return nil
	}
	return e.parent
}

// ParentElement returns the parent as an *Element, nil for the root.
func (e *Element) ParentElement() *Element { return e.parent }

// Children returns the child elements in order. The slice must not be
// modified.
func (e *Element) Children() []*Element { return e.children }

// PreviousSibling implements selector.Element.
func (e *Element) PreviousSibling() selector.Element {
	if e.parent == nil || e.index == 0 {
		return nil
	}
	return e.parent.children[e.index-1]
}

// NextSibling implements selector.Element.
func (e *Element) NextSibling() selector.Element {
	if e.parent == nil || e.index+1 >= len(e.parent.children) {
		return nil
	}
	return e.parent.children[e.index+1]
}

// FirstChild implements selector.Element.
func (e *Element) FirstChild() selector.Element {
	if len(e.children) == 0 {
		return nil
	}
	return e.children[0]
}

// Index implements selector.Element.
func (e *Element) Index() int { return e.index }

// IsEmpty implements selector.Element.
func (e *Element) IsEmpty() bool {
	for c := e.node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode || c.Type == nethtml.TextNode && c.Data != "" {
			return false
		}
	}
	return true
}

// InHTMLDocument implements selector.Element.
func (e *Element) InHTMLDocument() bool { return true }

// ChildText returns the text nodes that are direct children of the
// element, concatenated, leaving out the text of descendants.
func (e *Element) ChildText() string {
	var b strings.Builder
	for c := e.node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}

// isHTML reports whether the element is an HTML element with the given tag
// name.
func (e *Element) isHTML(tag string) bool {
	return e.namespace == XHTMLNamespace && e.node.Data == tag
}

// HasLinkType reports whether the rel attribute has the given link type,
// compared ASCII case-insensitively (HTML Living Standard §4.6.7).
func (e *Element) HasLinkType(linkType string) bool {
	for _, token := range selector.SplitWhitespace(e.Get("rel")) {
		if syntax.AsciiLower(token) == linkType {
			return true
		}
	}
	return false
}
