package html

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Kind tells how the box builder treats an element beyond its style: the
// elements that are replaced by an image, carry a link, or take a span.
type Kind int

const (
	// KindNone is any element without special handling.
	KindNone Kind = iota
	// KindImg is an img element, replaced by its image or its alt text.
	KindImg
	// KindEmbed is an embed element, replaced by its image or by nothing.
	KindEmbed
	// KindObject is an object element, replaced by its image or by its
	// children.
	KindObject
	// KindLink is an a element, which may carry a hyperlink.
	KindLink
	// KindColgroup is a colgroup element, standing for span columns when
	// it has no col children.
	KindColgroup
	// KindCol is a col element, standing for span columns.
	KindCol
	// KindSVG is an inline svg element, replaced by the image it draws.
	KindSVG
)

type tagKey struct{ namespace, name string }

var handlers = map[tagKey]Kind{
	{XHTMLNamespace, "img"}:      KindImg,
	{XHTMLNamespace, "embed"}:    KindEmbed,
	{XHTMLNamespace, "object"}:   KindObject,
	{XHTMLNamespace, "a"}:        KindLink,
	{XHTMLNamespace, "colgroup"}: KindColgroup,
	{XHTMLNamespace, "col"}:      KindCol,
	{SVGNamespace, "svg"}:        KindSVG,
}

// Kind returns how the box builder handles the element.
func (e *Element) Kind() Kind { return handlers[tagKey{e.namespace, e.node.Data}] }

// ImgInfo is what an img element says about its image.
type ImgInfo struct {
	// Src is the absolute URL of the image, or "" when the element has no
	// usable src. The srcset attribute is not considered.
	Src string
	// Alt is the text shown when the image cannot be loaded.
	Alt string
}

// Img returns the image source and the alternative text of an img element.
func (e *Element) Img() ImgInfo {
	src, _ := ParseURL(e.Get("src"), e.doc.BaseURL, false)
	return ImgInfo{Src: src, Alt: e.Get("alt")}
}

// ResourceInfo locates the resource an embed or object element shows.
type ResourceInfo struct {
	// URL is the absolute URL of the resource.
	URL string
	// MIMEType is the declared type attribute, trimmed. When not empty it
	// overrides the type announced by the server.
	MIMEType string
}

// Embed returns the resource of an embed element; ok is false when it has
// no usable src.
func (e *Element) Embed() (info ResourceInfo, ok bool) { return e.resource("src") }

// Object returns the resource of an object element; ok is false when it
// has no usable data attribute, in which case the children are its
// fallback content.
func (e *Element) Object() (info ResourceInfo, ok bool) { return e.resource("data") }

func (e *Element) resource(attr string) (ResourceInfo, bool) {
	url, ok := ParseURL(e.Get(attr), e.doc.BaseURL, false)
	if !ok {
		return ResourceInfo{}, false
	}
	return ResourceInfo{URL: url, MIMEType: strings.TrimSpace(e.Get("type"))}, true
}

// Link returns the destination of an a element, or nil when it has no
// usable href.
func (e *Element) Link() *urls.Link {
	href, ok := ParseURL(e.Get("href"), e.doc.BaseURL, true)
	if !ok {
		return nil
	}
	return urls.GetLink(href, e.doc.BaseURL)
}

// Span returns the number of columns a col or colgroup element stands for:
// its span attribute, at least 1 (HTML Living Standard §4.9.4).
func (e *Element) Span() int {
	n, ok := ParseInteger(e.Get("span"))
	if !ok {
		return 1
	}
	return max(n, 1)
}

// ColSpan returns the number of columns a table cell spans, at least 1.
func (e *Element) ColSpan() int {
	n, ok := ParseInteger(e.Get("colspan"))
	if !ok {
		return 1
	}
	return max(n, 1)
}

// RowSpan returns the number of rows a table cell spans. Zero means that
// the cell extends to the end of its row group (HTML Living Standard
// §4.9.11).
func (e *Element) RowSpan() int {
	n, ok := ParseInteger(e.Get("rowspan"))
	if !ok {
		return 1
	}
	return max(n, 0)
}

// HasChildElement reports whether a direct child is an HTML element with
// the given tag name.
func (e *Element) HasChildElement(tag string) bool {
	for _, c := range e.children {
		if c.isHTML(tag) {
			return true
		}
	}
	return false
}
