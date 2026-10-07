package selector

import "github.com/tomasweigenast/pdfgen/internal/css/syntax"

// Selector is a compiled complex selector.
type Selector struct {
	// Specificity is the (a, b, c) specificity of Selectors Level 4 §17. A
	// pseudo-element counts as a type selector.
	Specificity [3]int
	// PseudoElement is the lower-case name of the pseudo-element the
	// selector ends with, or "".
	PseudoElement string

	test compiled

	// Keys of the rightmost compound selector, used by Matcher to avoid
	// testing selectors that cannot apply to an element.
	id, class, localName, lowerLocalName string
	namespace                            string
	hasNamespace                         bool
	requiresLangAttr                     bool
}

// NeverMatches reports whether the selector is known to match no element.
func (s *Selector) NeverMatches() bool { return s.test.kind == kindNever }

// Match reports whether el is the subject of the selector, ignoring the
// pseudo-element.
func (s *Selector) Match(el Element) bool { return s.test.eval(el) }

// Compile compiles a comma-separated list of selectors.
//
// namespaces maps the declared prefixes to namespace URLs; the empty prefix
// declares the default namespace for type selectors. A nil map declares
// nothing. The first invalid selector fails the whole list, with an *Error.
func Compile(tokens []syntax.Node, namespaces map[string]string) ([]Selector, error) {
	parsed, err := parseSelectorList(tokens, namespaces, false, false)
	if err != nil {
		return nil, err
	}
	out := make([]Selector, 0, len(parsed))
	for _, p := range parsed {
		sel, err := compileSelector(p)
		if err != nil {
			return nil, err
		}
		out = append(out, sel)
	}
	return out, nil
}

// CompileString is like Compile for selectors given as text.
func CompileString(css string, namespaces map[string]string) ([]Selector, error) {
	return Compile(syntax.ParseComponentValueList(css, false), namespaces)
}

func compileSelector(p *parsedSelector) (Selector, error) {
	test, err := compileNode(p.tree)
	if err != nil {
		return Selector{}, err
	}
	sel := Selector{Specificity: p.spec, PseudoElement: p.pseudoElement, test: test}

	subject := p.tree
	if c, ok := subject.(*combined); ok {
		subject = c.right
	}
	for _, simple := range subject.(*compound).simples {
		switch s := simple.(type) {
		case *idSel:
			sel.id = s.id
		case *classSel:
			sel.class = s.name
		case *localName:
			sel.localName, sel.lowerLocalName = s.name, s.lower
		case *namespaceSel:
			sel.namespace, sel.hasNamespace = s.namespace, true
		case *attrSel:
			if s.name == "lang" {
				sel.requiresLangAttr = true
			}
		}
	}
	return sel, nil
}
