package selector

import (
	"fmt"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// supportedPseudoElements lists the names accepted after "::".
var supportedPseudoElements = map[string]bool{
	// CSS Pseudo-Elements Level 4
	"first-line": true, "first-letter": true, "prefix": true, "postfix": true,
	"selection": true, "target-text": true, "spelling-error": true,
	"grammar-error": true, "before": true, "after": true, "marker": true,
	"placeholder": true, "file-selector-button": true,
	// CSS Generated Content for Paged Media
	"footnote-call": true, "footnote-marker": true,
	// CSS Notes
	"note-call": true, "note-marker": true, "note-callback": true,
	// CSS Scoping Level 1
	"content": true, "shadow": true,
}

// Error is the error returned for an invalid or unsupported selector.
type Error struct {
	// Token is the offending token, or nil when the input ended early.
	Token syntax.Node
	Msg   string
}

func (e *Error) Error() string {
	if e.Token != nil {
		line, col := e.Token.Pos()
		return fmt.Sprintf("%s (%d:%d)", e.Msg, line, col)
	}
	return e.Msg
}

func errorf(tok syntax.Node, format string, args ...any) error {
	return &Error{Token: tok, Msg: fmt.Sprintf(format, args...)}
}

func typeName(n syntax.Node) string {
	if n == nil {
		return "EOF"
	}
	return n.Type()
}

type tokenStream struct {
	tokens []syntax.Node
	pos    int
}

func (s *tokenStream) next() syntax.Node {
	if s.pos >= len(s.tokens) {
		return nil
	}
	n := s.tokens[s.pos]
	s.pos++
	return n
}

func (s *tokenStream) peek() syntax.Node {
	if s.pos >= len(s.tokens) {
		return nil
	}
	return s.tokens[s.pos]
}

func (s *tokenStream) skip(types ...string) bool {
	found := false
	for {
		p := s.peek()
		if p == nil {
			return found
		}
		match := false
		for _, t := range types {
			if p.Type() == t {
				match = true
				break
			}
		}
		if !match {
			return found
		}
		s.pos++
		found = true
	}
}

func (s *tokenStream) skipWhitespace() bool { return s.skip("whitespace") }
func (s *tokenStream) skipComment() bool    { return s.skip("comment") }
func (s *tokenStream) skipBlank() bool      { return s.skip("comment", "whitespace") }

// isCombinator reports whether n is one of the combinator delimiters and
// returns it.
func isCombinator(n syntax.Node) (byte, bool) {
	if l, ok := n.(*syntax.LiteralToken); ok && len(l.Value) == 1 {
		switch c := l.Value[0]; c {
		case '>', '+', '~':
			return c, true
		}
	}
	return 0, false
}

// parseSelectorList parses a comma-separated selector list. In forgiving
// mode (Selectors Level 4 §18.1) invalid selectors are dropped instead of
// failing the whole list; a list whose first selector is invalid is dropped
// entirely.
func parseSelectorList(tokens []syntax.Node, namespaces map[string]string, forgiving, relative bool) ([]*parsedSelector, error) {
	ts := &tokenStream{tokens: tokens}
	sel, err := parseSelector(ts, namespaces, relative)
	if err != nil {
		if forgiving {
			return nil, nil
		}
		return nil, err
	}
	list := []*parsedSelector{sel}
	for {
		tok := ts.next()
		if tok == nil {
			return list, nil
		}
		if syntax.IsLiteral(tok, ",") {
			sel, err := parseSelector(ts, namespaces, relative)
			if err != nil {
				if !forgiving {
					return nil, err
				}
				continue
			}
			list = append(list, sel)
		} else if !forgiving {
			return nil, errorf(tok, "unexpected %s token.", tok.Type())
		}
	}
}

func parseSelector(ts *tokenStream, namespaces map[string]string, relative bool) (*parsedSelector, error) {
	ts.skipBlank()
	initial := byte(0)
	if relative {
		initial = ' '
		if c, ok := isCombinator(ts.peek()); ok {
			initial = c
			ts.next()
		}
		ts.skipBlank()
	}
	var result node
	comp, pseudo, err := parseCompoundSelector(ts, namespaces)
	if err != nil {
		return nil, err
	}
	result = comp
	for {
		hasWhitespace := ts.skipWhitespace()
		for ts.skipComment() {
			if ts.skipWhitespace() {
				hasWhitespace = true
			}
		}
		sel := newParsedSelector(result, pseudo, initial)
		if pseudo != "" {
			return sel, nil
		}
		peek := ts.peek()
		var combinator byte
		if peek == nil || syntax.IsLiteral(peek, ",") {
			return sel, nil
		} else if c, ok := isCombinator(peek); ok {
			combinator = c
			ts.next()
		} else if hasWhitespace {
			combinator = ' '
		} else {
			return sel, nil
		}
		comp, pseudo, err = parseCompoundSelector(ts, namespaces)
		if err != nil {
			return nil, err
		}
		result = &combined{result, combinator, comp}
	}
}

func parseCompoundSelector(ts *tokenStream, namespaces map[string]string) (*compound, string, error) {
	simples, typed, err := parseTypeSelector(ts, namespaces)
	if err != nil {
		return nil, "", err
	}
	pseudo := ""
	for {
		simple, pe, err := parseSimpleSelector(ts, namespaces)
		if err != nil {
			return nil, "", err
		}
		if pe != "" {
			pseudo = pe
			break
		}
		if simple == nil {
			break
		}
		simples = append(simples, simple)
	}
	if len(simples) > 0 || typed || pseudo != "" {
		return &compound{simples}, pseudo, nil
	}
	peek := ts.peek()
	return nil, "", errorf(peek, "expected a compound selector, got %s", typeName(peek))
}

// qualifiedName is the result of parsing [ns|]name. hasNamespace is false
// for an unspecified or any (*|) namespace; hasLocal is false for *.
type qualifiedName struct {
	namespace    string
	hasNamespace bool
	local, lower string
	hasLocal     bool
}

func parseTypeSelector(ts *tokenStream, namespaces map[string]string) (simples []node, typed bool, err error) {
	ts.skipWhitespace()
	q, ok, err := parseQualifiedName(ts, namespaces, false)
	if err != nil || !ok {
		return nil, false, err
	}
	simples = []node{}
	if q.hasLocal {
		simples = append(simples, &localName{q.local, q.lower})
	}
	if q.hasNamespace {
		simples = append(simples, &namespaceSel{q.namespace})
	}
	return simples, true, nil
}

func parseSimpleSelector(ts *tokenStream, namespaces map[string]string) (node, string, error) {
	peek := ts.peek()
	switch p := peek.(type) {
	case nil:
		return nil, "", nil
	case *syntax.HashToken:
		if p.IsIdentifier {
			ts.next()
			return &idSel{p.Value}, "", nil
		}
		return nil, "", nil
	case *syntax.SquareBracketsBlock:
		ts.next()
		attr, err := parseAttributeSelector(&tokenStream{tokens: p.Content}, namespaces)
		if err != nil {
			return nil, "", err
		}
		return attr, "", nil
	case *syntax.LiteralToken:
		switch p.Value {
		case ".":
			ts.next()
			next := ts.next()
			ident, ok := next.(*syntax.IdentToken)
			if !ok {
				return nil, "", errorf(next, "Expected a class name, got %s", typeName(next))
			}
			return &classSel{ident.Value}, "", nil
		case ":":
			ts.next()
			return parseColon(ts, namespaces)
		}
	}
	return nil, "", nil
}

// parseColon parses what follows a colon: a pseudo-class, a pseudo-element
// or a functional pseudo-class.
func parseColon(ts *tokenStream, namespaces map[string]string) (node, string, error) {
	next := ts.next()
	switch n := next.(type) {
	case *syntax.LiteralToken:
		if n.Value != ":" {
			break
		}
		name := ts.next()
		ident, ok := name.(*syntax.IdentToken)
		if !ok {
			return nil, "", errorf(name, "Expected a pseudo-element name, got %s", typeName(name))
		}
		if !supportedPseudoElements[ident.LowerValue] {
			return nil, "", errorf(ident, "Expected a supported pseudo-element, got %s", ident.LowerValue)
		}
		return nil, ident.LowerValue, nil
	case *syntax.IdentToken:
		switch n.LowerValue {
		case "before", "after", "first-line", "first-letter":
			return nil, n.LowerValue, nil
		}
		return &pseudoClass{n.LowerValue}, "", nil
	case *syntax.FunctionBlock:
		switch n.LowerName {
		case "is", "where", "not", "has":
			sel, err := parseLogicalCombination(n, namespaces)
			return sel, "", err
		}
		return &functionalPseudoClass{n.LowerName, n.Arguments}, "", nil
	}
	return nil, "", errorf(next, "unexpected %s token.", typeName(next))
}

func parseLogicalCombination(fn *syntax.FunctionBlock, namespaces map[string]string) (node, error) {
	name := fn.LowerName
	forgiving := name != "not"
	relative := name == "has"
	parsed, err := parseSelectorList(fn.Arguments, namespaces, forgiving, relative)
	if err != nil {
		return nil, err
	}
	var list []*parsedSelector
	for _, sel := range parsed {
		if sel.pseudoElement == "" {
			list = append(list, sel)
		}
	}
	switch name {
	case "is":
		return &matchesAny{list}, nil
	case "where":
		return &zeroSpecificity{list}, nil
	case "not":
		return &negation{list}, nil
	default:
		return &relational{list}, nil
	}
}

var attributeOperators = []string{"=", "~=", "|=", "^=", "$=", "*="}

func parseAttributeSelector(ts *tokenStream, namespaces map[string]string) (node, error) {
	ts.skipWhitespace()
	q, ok, err := parseQualifiedName(ts, namespaces, true)
	if err != nil {
		return nil, err
	}
	if !ok {
		next := ts.next()
		return nil, errorf(next, "expected attribute name, got %s", typeName(next))
	}
	attr := &attrSel{
		anyNamespace: !q.hasNamespace,
		namespace:    q.namespace,
		name:         q.local,
		lower:        q.lower,
	}

	ts.skipWhitespace()
	peek := ts.peek()
	if peek != nil {
		op := ""
		if l, ok := peek.(*syntax.LiteralToken); ok {
			for _, candidate := range attributeOperators {
				if l.Value == candidate {
					op = candidate
				}
			}
		}
		if op == "" {
			return nil, errorf(peek, "expected attribute selector operator, got %s", peek.Serialize())
		}
		attr.operator = op
		ts.next()
		ts.skipWhitespace()
		next := ts.next()
		switch v := next.(type) {
		case *syntax.IdentToken:
			attr.value = v.Value
		case *syntax.StringToken:
			attr.value = v.Value
		default:
			return nil, errorf(next, "expected attribute value, got %s", typeName(next))
		}
	}

	ts.skipWhitespace()
	if next := ts.next(); next != nil {
		ident, ok := next.(*syntax.IdentToken)
		switch {
		case ok && syntax.AsciiLower(ident.Value) == "i":
			attr.mode = caseInsensitive
		case ok && syntax.AsciiLower(ident.Value) == "s":
			attr.mode = caseSensitive
		default:
			return nil, errorf(next, "expected ], got %s", next.Type())
		}
	}
	return attr, nil
}

// parseQualifiedName parses [namespace|]local, where either part may be *.
// ok is false when no name starts at the current position.
func parseQualifiedName(ts *tokenStream, namespaces map[string]string, isAttribute bool) (q qualifiedName, ok bool, err error) {
	peek := ts.peek()
	if peek == nil {
		return q, false, nil
	}
	var namespace string
	var hasNamespace bool
	switch p := peek.(type) {
	case *syntax.IdentToken:
		ts.next()
		if !syntax.IsLiteral(ts.peek(), "|") {
			q.local, q.lower, q.hasLocal = p.Value, p.LowerValue, true
			if isAttribute {
				q.hasNamespace = true
			} else {
				q.namespace, q.hasNamespace = namespaces[""]
			}
			return q, true, nil
		}
		ts.next()
		namespace, hasNamespace = namespaces[p.Value]
		if !hasNamespace {
			return q, false, errorf(p, "undefined namespace prefix: %s", p.Value)
		}
	case *syntax.LiteralToken:
		switch p.Value {
		case "*":
			ts.next()
			if !syntax.IsLiteral(ts.peek(), "|") {
				if isAttribute {
					return q, false, errorf(p, "expected local name, got %s", p.Type())
				}
				q.namespace, q.hasNamespace = namespaces[""]
				return q, true, nil
			}
			ts.next()
		case "|":
			ts.next()
			namespace, hasNamespace = "", true
		default:
			return q, false, nil
		}
	default:
		return q, false, nil
	}

	next := ts.next()
	q.namespace, q.hasNamespace = namespace, hasNamespace
	switch n := next.(type) {
	case *syntax.IdentToken:
		q.local, q.lower, q.hasLocal = n.Value, n.LowerValue, true
		return q, true, nil
	case *syntax.LiteralToken:
		if n.Value == "*" && !isAttribute {
			return q, true, nil
		}
	}
	return q, false, errorf(next, "expected local name, got %s", typeName(next))
}
