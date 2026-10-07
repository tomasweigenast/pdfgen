package selector

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

// testElement is a minimal Element built from an XML document.
type testElement struct {
	name     xml.Name
	attrs    map[xml.Name]string
	parent   *testElement
	children []*testElement
	index    int
	hasText  bool
	html     bool
	// transportLang is the document-level language, set on the root only.
	transportLang string
}

var _ Element = (*testElement)(nil)

func (e *testElement) LocalName() string { return e.name.Local }
func (e *testElement) Namespace() string { return e.name.Space }
func (e *testElement) Attr(namespace, name string) (string, bool) {
	v, ok := e.attrs[xml.Name{Space: namespace, Local: name}]
	return v, ok
}
func (e *testElement) ID() string {
	v, _ := e.Attr("", "id")
	return v
}
func (e *testElement) Classes() []string {
	v, _ := e.Attr("", "class")
	return SplitWhitespace(v)
}
func (e *testElement) InHTMLDocument() bool { return e.html }
func (e *testElement) Index() int           { return e.index }
func (e *testElement) IsEmpty() bool        { return len(e.children) == 0 && !e.hasText }

func (e *testElement) Parent() Element {
	if e.parent == nil {
		return nil
	}
	return e.parent
}

func (e *testElement) PreviousSibling() Element {
	if e.parent == nil || e.index == 0 {
		return nil
	}
	return e.parent.children[e.index-1]
}

func (e *testElement) NextSibling() Element {
	if e.parent == nil || e.index+1 >= len(e.parent.children) {
		return nil
	}
	return e.parent.children[e.index+1]
}

func (e *testElement) FirstChild() Element {
	if len(e.children) == 0 {
		return nil
	}
	return e.children[0]
}

func (e *testElement) Lang() string {
	return InheritedLang(e, func() string {
		if e.html || e.name.Space == XHTMLNamespace {
			found := ""
			e.walk(func(el *testElement) {
				if el.name != (xml.Name{Space: XHTMLNamespace, Local: "meta"}) {
					return
				}
				equiv, _ := el.Attr("", "http-equiv")
				if syntaxLower(equiv) == "content-language" {
					content, _ := el.Attr("", "content")
					found = ParseContentLanguage(content)
				}
			})
			if found != "" {
				return found
			}
		}
		return ParseContentLanguage(e.transportLang)
	})
}

func syntaxLower(s string) string { return strings.ToLower(s) }

func (e *testElement) walk(fn func(*testElement)) {
	fn(e)
	for _, c := range e.children {
		c.walk(fn)
	}
}

// parseXML builds the element tree of an XML document. html selects HTML
// document semantics for matching.
func parseXML(t testing.TB, src string, html bool) *testElement {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(src))
	var root, cur *testElement
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parsing XML: %v", err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			el := &testElement{name: tok.Name, attrs: map[xml.Name]string{}, parent: cur, html: html}
			for _, a := range tok.Attr {
				if a.Name.Space == "xmlns" || a.Name.Space == "" && a.Name.Local == "xmlns" {
					continue
				}
				el.attrs[a.Name] = a.Value
			}
			if cur == nil {
				root = el
			} else {
				el.index = len(cur.children)
				cur.children = append(cur.children, el)
			}
			cur = el
		case xml.EndElement:
			cur = cur.parent
		case xml.CharData:
			if cur != nil && len(tok) > 0 {
				cur.hasText = true
			}
		}
	}
	if root == nil {
		t.Fatal("document has no root element")
	}
	return root
}

// subtree returns el and its descendants in tree order.
func (e *testElement) subtree() []*testElement {
	var out []*testElement
	e.walk(func(el *testElement) { out = append(out, el) })
	return out
}

// queryAll returns the elements of the subtree matching any of the
// selectors, in tree order. Selectors with a pseudo-element never match.
func queryAll(t testing.TB, root *testElement, css string) []*testElement {
	t.Helper()
	sels, err := CompileString(css, nil)
	if err != nil {
		t.Fatalf("compiling %q: %v", css, err)
	}
	return queryCompiled(root, sels)
}

func queryCompiled(root *testElement, sels []Selector) []*testElement {
	var out []*testElement
	for _, el := range root.subtree() {
		for i := range sels {
			if sels[i].PseudoElement == "" && sels[i].Match(el) {
				out = append(out, el)
				break
			}
		}
	}
	return out
}

func idsOf(els []*testElement) []string {
	ids := []string{}
	for _, el := range els {
		v, ok := el.Attr("", "id")
		if !ok {
			v = "nil"
		}
		ids = append(ids, v)
	}
	return ids
}
