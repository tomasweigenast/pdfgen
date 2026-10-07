package selector

import (
	"sort"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// Match is a selector that matched an element, with the payload it was added
// with.
type Match struct {
	Specificity   [3]int
	Order         int
	PseudoElement string
	Payload       any
}

type entry struct {
	selector *Selector
	order    int
	payload  any
}

// Matcher stores selectors and finds those that match an element. Selectors
// are indexed by the id, class, local name or namespace of their subject, so
// that matching an element only tests the selectors that can apply to it.
type Matcher struct {
	byID        map[string][]entry
	byClass     map[string][]entry
	byLocalName map[string][]entry
	byNamespace map[string][]entry
	langAttr    []entry
	other       []entry
	order       int
}

// Add stores a selector with an arbitrary payload, returned by Match.
// Selectors that can never match are counted for ordering but not stored.
func (m *Matcher) Add(sel Selector, payload any) {
	m.order++
	if sel.NeverMatches() {
		return
	}
	e := entry{&sel, m.order, payload}
	switch {
	case sel.id != "":
		m.byID = appendEntry(m.byID, sel.id, e)
	case sel.class != "":
		m.byClass = appendEntry(m.byClass, sel.class, e)
	case sel.localName != "":
		m.byLocalName = appendEntry(m.byLocalName, sel.lowerLocalName, e)
	case sel.hasNamespace:
		m.byNamespace = appendEntry(m.byNamespace, sel.namespace, e)
	case sel.requiresLangAttr:
		m.langAttr = append(m.langAttr, e)
	default:
		m.other = append(m.other, e)
	}
}

func appendEntry(index map[string][]entry, key string, e entry) map[string][]entry {
	if index == nil {
		index = map[string][]entry{}
	}
	index[key] = append(index[key], e)
	return index
}

// Match returns the selectors that match el, ordered from the lowest to the
// highest specificity and, among equal specificities, in the order they were
// added.
func (m *Matcher) Match(el Element) []Match {
	var out []Match

	if id := el.ID(); id != "" {
		out = collect(el, m.byID[id], out)
	}
	if len(m.byClass) > 0 {
		classes := el.Classes()
		for i, c := range classes {
			if !firstOccurrence(classes, i) {
				continue
			}
			out = collect(el, m.byClass[c], out)
		}
	}
	out = collect(el, m.byLocalName[syntax.AsciiLower(el.LocalName())], out)
	out = collect(el, m.byNamespace[el.Namespace()], out)
	if _, ok := el.Attr("", "lang"); ok {
		out = collect(el, m.langAttr, out)
	}
	out = collect(el, m.other, out)

	sort.Slice(out, func(i, j int) bool {
		if c := compareSpecificity(out[i].Specificity, out[j].Specificity); c != 0 {
			return c < 0
		}
		return out[i].Order < out[j].Order
	})
	return out
}

func firstOccurrence(list []string, i int) bool {
	for _, s := range list[:i] {
		if s == list[i] {
			return false
		}
	}
	return true
}

func collect(el Element, entries []entry, out []Match) []Match {
	for _, e := range entries {
		if e.selector.test.eval(el) {
			out = append(out, Match{
				Specificity:   e.selector.Specificity,
				Order:         e.order,
				PseudoElement: e.selector.PseudoElement,
				Payload:       e.payload,
			})
		}
	}
	return out
}
