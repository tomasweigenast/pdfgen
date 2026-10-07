package selector

import "github.com/tomasweigenast/pdfgen/internal/css/syntax"

// node is a parsed selector component with a specificity (Selectors Level 4
// §17).
type node interface {
	specificity() [3]int
}

// parsedSelector is a complex selector with an optional trailing
// pseudo-element. combinator is the leading combinator of a relative
// selector (the argument of :has()) and 0 otherwise.
type parsedSelector struct {
	tree          node
	pseudoElement string
	combinator    byte
	spec          [3]int
}

func newParsedSelector(tree node, pseudoElement string, combinator byte) *parsedSelector {
	spec := tree.specificity()
	if pseudoElement != "" {
		spec[2]++
	}
	return &parsedSelector{tree, pseudoElement, combinator, spec}
}

// combined is a complex selector: left, a combined or compound selector,
// related to the compound selector right by combinator, one of ' ', '>',
// '+' or '~'.
type combined struct {
	left       node
	combinator byte
	right      *compound
}

func (n *combined) specificity() [3]int {
	return addSpecificity(n.left.specificity(), n.right.specificity())
}

type compound struct {
	simples []node
}

func (n *compound) specificity() [3]int {
	var s [3]int
	for _, simple := range n.simples {
		s = addSpecificity(s, simple.specificity())
	}
	return s
}

func addSpecificity(a, b [3]int) [3]int {
	return [3]int{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func maxSpecificity(list []*parsedSelector) [3]int {
	var best [3]int
	for i, sel := range list {
		if i == 0 || compareSpecificity(sel.spec, best) > 0 {
			best = sel.spec
		}
	}
	return best
}

func compareSpecificity(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

type localName struct{ name, lower string }

func (*localName) specificity() [3]int { return [3]int{0, 0, 1} }

// namespaceSel matches the namespace URL; "" stands for no namespace.
type namespaceSel struct{ namespace string }

func (*namespaceSel) specificity() [3]int { return [3]int{} }

type idSel struct{ id string }

func (*idSel) specificity() [3]int { return [3]int{1, 0, 0} }

type classSel struct{ name string }

func (*classSel) specificity() [3]int { return [3]int{0, 1, 0} }

type caseMode uint8

const (
	// caseDefault is an attribute selector without i or s flag; values are
	// compared case-sensitively.
	caseDefault caseMode = iota
	caseSensitive
	caseInsensitive
)

// attrSel is an attribute selector. anyNamespace is set for the *|name
// form; otherwise namespace holds the URL, "" meaning no namespace.
type attrSel struct {
	anyNamespace bool
	namespace    string
	name, lower  string
	// operator is "" for [attr], else one of = ~= |= ^= $= *=.
	operator string
	value    string
	mode     caseMode
}

func (*attrSel) specificity() [3]int { return [3]int{0, 1, 0} }

type pseudoClass struct{ name string }

func (*pseudoClass) specificity() [3]int { return [3]int{0, 1, 0} }

type functionalPseudoClass struct {
	name string
	args []syntax.Node
}

func (*functionalPseudoClass) specificity() [3]int { return [3]int{0, 1, 0} }

// negation is :not().
type negation struct{ list []*parsedSelector }

func (n *negation) specificity() [3]int { return maxSpecificity(n.list) }

// matchesAny is :is().
type matchesAny struct{ list []*parsedSelector }

func (n *matchesAny) specificity() [3]int { return maxSpecificity(n.list) }

// relational is :has(); its list holds relative selectors.
type relational struct{ list []*parsedSelector }

func (n *relational) specificity() [3]int { return maxSpecificity(n.list) }

// zeroSpecificity is :where().
type zeroSpecificity struct{ list []*parsedSelector }

func (*zeroSpecificity) specificity() [3]int { return [3]int{} }
