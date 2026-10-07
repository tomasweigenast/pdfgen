package html

import (
	"fmt"
	"strings"
	"unicode"

	nethtml "golang.org/x/net/html"
)

// SVGMarkup serializes an element and its descendants as an XML document
// fragment, so that an inline svg can be handed to an SVG parser. The root
// declares the SVG and XLink namespaces, and an element that changes
// namespace, such as the HTML content of a foreignObject, declares its own.
// Comments are left out.
func (e *Element) SVGMarkup() string {
	var b strings.Builder
	writeXML(&b, e.node, "")
	return b.String()
}

func writeXML(b *strings.Builder, n *nethtml.Node, parentNS string) {
	switch n.Type {
	case nethtml.TextNode:
		escapeXML(b, n.Data, false)
		return
	case nethtml.ElementNode:
	default:
		return
	}

	ns := elementNamespace(n.Namespace)
	b.WriteByte('<')
	b.WriteString(n.Data)
	declared := map[string]bool{}
	for _, a := range n.Attr {
		name := a.Key
		switch a.Namespace {
		case "xlink", "xml", "xmlns":
			name = a.Namespace + ":" + a.Key
		}
		if !validXMLName(name) {
			continue
		}
		declared[name] = true
		b.WriteByte(' ')
		b.WriteString(name)
		b.WriteString(`="`)
		escapeXML(b, a.Val, true)
		b.WriteByte('"')
	}
	if ns != parentNS && !declared["xmlns"] {
		b.WriteString(` xmlns="`)
		escapeXML(b, ns, true)
		b.WriteByte('"')
	}
	if parentNS == "" && !declared["xmlns:xlink"] {
		b.WriteString(` xmlns:xlink="` + XLinkNamespace + `"`)
	}

	if n.FirstChild == nil {
		b.WriteString("/>")
		return
	}
	b.WriteByte('>')
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		writeXML(b, c, ns)
	}
	b.WriteString("</" + n.Data + ">")
}

func escapeXML(b *strings.Builder, s string, attr bool) {
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case attr && r == '"':
			b.WriteString("&quot;")
		case attr && (r == '\t' || r == '\n' || r == '\r'):
			fmt.Fprintf(b, "&#%d;", r)
		case r < 0x20 && r != '\t' && r != '\n' && r != '\r', r == 0xFFFE, r == 0xFFFF:
		default:
			b.WriteRune(r)
		}
	}
}

// validXMLName reports whether s can be written as an attribute name. The
// HTML parser accepts names that XML does not, and one of them would make
// the whole fragment unparseable.
func validXMLName(s string) bool {
	for i, r := range s {
		switch {
		case r == '_' || r == ':' || unicode.IsLetter(r):
		case i > 0 && (r == '-' || r == '.' || unicode.IsDigit(r)):
		default:
			return false
		}
	}
	return s != ""
}
