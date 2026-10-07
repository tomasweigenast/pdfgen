package selector

import (
	"reflect"
	"testing"
)

func addAll(t *testing.T, m *Matcher, css string, ns map[string]string) {
	t.Helper()
	sels, err := CompileString(css, ns)
	if err != nil {
		t.Fatalf("compiling %q: %v", css, err)
	}
	for i, s := range sels {
		label := css
		if len(sels) > 1 {
			label = css + "#" + string(rune('0'+i))
		}
		m.Add(s, label)
	}
}

func payloads(ms []Match) []any {
	out := []any{}
	for _, m := range ms {
		out = append(out, m.Payload)
	}
	return out
}

func TestMatcherOrdersBySpecificityThenInsertion(t *testing.T) {
	root := parseXML(t, `<html xmlns="http://www.w3.org/1999/xhtml"><p id="x" class="c d" lang="en"/></html>`, true)
	p := root.children[0]

	m := &Matcher{}
	for _, css := range []string{
		"#x", "p", ".c", "*", "p.c", "[lang]", ".d", "p, .c", "html > p",
		":lang(en)", "#nope", "p:hover",
	} {
		addAll(t, m, css, nil)
	}
	got := payloads(m.Match(p))
	want := []any{
		"*",
		"p", "p, .c#0",
		"html > p",
		".c", "[lang]", ".d", "p, .c#1", ":lang(en)",
		"p.c",
		"#x",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

func TestMatcherResultCarriesSelectorData(t *testing.T) {
	root := parseXML(t, `<html xmlns="http://www.w3.org/1999/xhtml"><p/></html>`, true)
	m := &Matcher{}
	addAll(t, m, "p", nil)
	addAll(t, m, ":hover", nil)
	addAll(t, m, "p::marker", nil)
	got := m.Match(root.children[0])
	want := []Match{
		{Specificity: [3]int{0, 0, 1}, Order: 1, Payload: "p"},
		{Specificity: [3]int{0, 0, 2}, Order: 3, PseudoElement: "marker", Payload: "p::marker"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestMatcherIndexesDoNotHideSelectors(t *testing.T) {
	const doc = `<html xmlns="http://www.w3.org/1999/xhtml" xmlns:o="urn:o">
	<div id="a" class="x x y"><o:item o:k="1"/><b lang="fr"/></div></html>`
	root := parseXML(t, doc, true)
	ns := map[string]string{"o": "urn:o"}

	m := &Matcher{}
	for _, css := range []string{
		"#a", ".x", ".y", "div", "DIV", "o|item", "o|*", "[lang=fr]",
		"[lang|=fr]:not(.z)", "b[lang]", "*", "[o|k]", "#other",
	} {
		addAll(t, m, css, ns)
	}

	div := root.children[0]
	if got := payloads(m.Match(div)); !reflect.DeepEqual(got, []any{"*", "div", "DIV", ".x", ".y", "#a"}) {
		t.Errorf("div: got %v", got)
	}
	item := div.children[0]
	if got := payloads(m.Match(item)); !reflect.DeepEqual(got, []any{"o|*", "*", "o|item", "[o|k]"}) {
		t.Errorf("item: got %v", got)
	}
	b := div.children[1]
	if got := payloads(m.Match(b)); !reflect.DeepEqual(got, []any{"*", "[lang=fr]", "b[lang]", "[lang|=fr]:not(.z)"}) {
		t.Errorf("b: got %v", got)
	}
}
