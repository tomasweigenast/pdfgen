package selector

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// compiled is a node reduced to a predicate. Constant results are kept apart
// so that combinators can drop them at compile time: a selector that can
// never match is not even added to a Matcher.
type compiled struct {
	kind compiledKind
	fn   func(Element) bool
}

type compiledKind uint8

const (
	kindFunc compiledKind = iota
	kindNever
	kindAlways
)

var (
	never  = compiled{kind: kindNever}
	always = compiled{kind: kindAlways}
)

func predicate(fn func(Element) bool) compiled { return compiled{fn: fn} }

func (c compiled) eval(el Element) bool {
	switch c.kind {
	case kindNever:
		return false
	case kindAlways:
		return true
	}
	return c.fn(el)
}

func compileNode(n node) (compiled, error) {
	switch n := n.(type) {
	case *combined:
		return compileCombined(n)
	case *compound:
		return compileCompound(n)
	case *negation:
		return compileNegation(n)
	case *relational:
		return compileRelational(n)
	case *matchesAny:
		return compileAnyOf(n.list)
	case *zeroSpecificity:
		return compileAnyOf(n.list)
	case *localName:
		return compileLocalName(n), nil
	case *namespaceSel:
		ns := n.namespace
		return predicate(func(el Element) bool { return el.Namespace() == ns }), nil
	case *classSel:
		name := n.name
		return predicate(func(el Element) bool {
			for _, c := range el.Classes() {
				if c == name {
					return true
				}
			}
			return false
		}), nil
	case *idSel:
		id := n.id
		return predicate(func(el Element) bool { return el.ID() == id }), nil
	case *attrSel:
		return compileAttribute(n)
	case *pseudoClass:
		return compilePseudoClass(n.name)
	case *functionalPseudoClass:
		if n.name == "lang" {
			return compileLang(n.args)
		}
		return compileNth(n)
	}
	return never, errorf(nil, "unknown selector component %T", n)
}

func compileCombined(n *combined) (compiled, error) {
	left, err := compileNode(n.left)
	if err != nil {
		return never, err
	}
	if left.kind == kindNever {
		return never, nil
	}
	var leftTest func(Element) bool
	switch n.combinator {
	case ' ':
		if left.kind == kindAlways {
			leftTest = func(el Element) bool { return el.Parent() != nil }
		} else {
			leftTest = func(el Element) bool {
				for p := el.Parent(); p != nil; p = p.Parent() {
					if left.fn(p) {
						return true
					}
				}
				return false
			}
		}
	case '>':
		if left.kind == kindAlways {
			leftTest = func(el Element) bool { return el.Parent() != nil }
		} else {
			leftTest = func(el Element) bool {
				p := el.Parent()
				return p != nil && left.fn(p)
			}
		}
	case '+':
		if left.kind == kindAlways {
			leftTest = func(el Element) bool { return el.PreviousSibling() != nil }
		} else {
			leftTest = func(el Element) bool {
				p := el.PreviousSibling()
				return p != nil && left.fn(p)
			}
		}
	case '~':
		if left.kind == kindAlways {
			leftTest = func(el Element) bool { return el.PreviousSibling() != nil }
		} else {
			leftTest = func(el Element) bool {
				for p := el.PreviousSibling(); p != nil; p = p.PreviousSibling() {
					if left.fn(p) {
						return true
					}
				}
				return false
			}
		}
	default:
		return never, errorf(nil, "unknown combinator %q", n.combinator)
	}

	right, err := compileNode(n.right)
	if err != nil {
		return never, err
	}
	switch right.kind {
	case kindNever:
		return never, nil
	case kindAlways:
		return predicate(leftTest), nil
	}
	// Combinators are evaluated right to left: the subject is the cheaper
	// test and rules out most elements.
	return predicate(func(el Element) bool { return right.fn(el) && leftTest(el) }), nil
}

func compileCompound(n *compound) (compiled, error) {
	var subs []compiled
	for _, simple := range n.simples {
		c, err := compileNode(simple)
		if err != nil {
			return never, err
		}
		if c.kind != kindAlways {
			subs = append(subs, c)
		}
	}
	switch {
	case len(subs) == 1:
		return subs[0], nil
	case len(subs) == 0:
		return always, nil
	}
	for _, c := range subs {
		if c.kind == kindNever {
			return never, nil
		}
	}
	return predicate(func(el Element) bool {
		for _, c := range subs {
			if !c.fn(el) {
				return false
			}
		}
		return true
	}), nil
}

func compileSelectorTrees(list []*parsedSelector) ([]compiled, error) {
	out := make([]compiled, 0, len(list))
	for _, sel := range list {
		c, err := compileNode(sel.tree)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func compileNegation(n *negation) (compiled, error) {
	trees, err := compileSelectorTrees(n.list)
	if err != nil {
		return never, err
	}
	var subs []compiled
	for _, c := range trees {
		if c.kind != kindAlways {
			subs = append(subs, c)
		}
	}
	if len(subs) == 0 {
		return never, nil
	}
	return predicate(func(el Element) bool {
		for _, c := range subs {
			if c.eval(el) {
				return false
			}
		}
		return true
	}), nil
}

// compileAnyOf compiles :is() and :where(), which only differ in
// specificity.
func compileAnyOf(list []*parsedSelector) (compiled, error) {
	trees, err := compileSelectorTrees(list)
	if err != nil {
		return never, err
	}
	var subs []compiled
	for _, c := range trees {
		if c.kind != kindNever {
			subs = append(subs, c)
		}
	}
	if len(subs) == 0 {
		return never, nil
	}
	return predicate(func(el Element) bool {
		for _, c := range subs {
			if c.eval(el) {
				return true
			}
		}
		return false
	}), nil
}

type relativeTest struct {
	combinator byte
	test       compiled
}

func compileRelational(n *relational) (compiled, error) {
	var subs []relativeTest
	for _, sel := range n.list {
		c, err := compileNode(sel.tree)
		if err != nil {
			return never, err
		}
		if c.kind == kindNever {
			continue
		}
		subs = append(subs, relativeTest{sel.combinator, c})
	}
	if len(subs) == 0 {
		return never, nil
	}
	return predicate(func(el Element) bool {
		for _, s := range subs {
			if hasRelative(el, s.combinator, s.test) {
				return true
			}
		}
		return false
	}), nil
}

func hasRelative(el Element, combinator byte, t compiled) bool {
	switch combinator {
	case ' ':
		return anyDescendant(el, t)
	case '>':
		for c := el.FirstChild(); c != nil; c = c.NextSibling() {
			if t.eval(c) {
				return true
			}
		}
	case '+':
		next := el.NextSibling()
		return next != nil && t.eval(next)
	case '~':
		for s := el.NextSibling(); s != nil; s = s.NextSibling() {
			if t.eval(s) {
				return true
			}
		}
	}
	return false
}

func anyDescendant(el Element, t compiled) bool {
	for c := el.FirstChild(); c != nil; c = c.NextSibling() {
		if t.eval(c) || anyDescendant(c, t) {
			return true
		}
	}
	return false
}

func compileLocalName(n *localName) compiled {
	name, lower := n.name, n.lower
	if name == lower {
		return predicate(func(el Element) bool { return el.LocalName() == name })
	}
	return predicate(func(el Element) bool {
		if el.InHTMLDocument() {
			return el.LocalName() == lower
		}
		return el.LocalName() == name
	})
}

func compileAttribute(a *attrSel) (compiled, error) {
	if a.anyNamespace {
		return never, errorf(nil, "attribute selectors in any namespace are not supported")
	}
	ns, name, lower := a.namespace, a.name, a.lower
	get := func(el Element) (string, bool) {
		if name != lower && el.InHTMLDocument() {
			return el.Attr(ns, lower)
		}
		return el.Attr(ns, name)
	}
	value := a.value
	insensitive := a.mode == caseInsensitive
	if insensitive {
		value = strings.ToLower(value)
	}
	attrValue := func(el Element) string {
		v, _ := get(el)
		if insensitive {
			return strings.ToLower(v)
		}
		return v
	}

	switch a.operator {
	case "":
		return predicate(func(el Element) bool { _, ok := get(el); return ok }), nil
	case "=":
		return predicate(func(el Element) bool {
			_, ok := get(el)
			return ok && attrValue(el) == value
		}), nil
	case "~=":
		if len(strings.Fields(value)) != 1 || strings.TrimSpace(value) != value {
			return never, nil
		}
		return predicate(func(el Element) bool {
			for _, word := range SplitWhitespace(attrValue(el)) {
				if word == value {
					return true
				}
			}
			return false
		}), nil
	case "|=":
		return predicate(func(el Element) bool {
			v := attrValue(el)
			if _, ok := get(el); ok && v == value {
				return true
			}
			return strings.HasPrefix(v, value+"-")
		}), nil
	case "^=":
		if value == "" {
			return never, nil
		}
		return predicate(func(el Element) bool { return strings.HasPrefix(attrValue(el), value) }), nil
	case "$=":
		if value == "" {
			return never, nil
		}
		return predicate(func(el Element) bool { return strings.HasSuffix(attrValue(el), value) }), nil
	case "*=":
		if value == "" {
			return never, nil
		}
		return predicate(func(el Element) bool { return strings.Contains(attrValue(el), value) }), nil
	}
	return never, errorf(nil, "unknown attribute operator %q", a.operator)
}

// htmlTag returns a test for HTML elements with one of the given names. In an
// HTML document the namespace is not checked, since parsers may leave it
// out.
func htmlTag(names ...string) func(Element) bool {
	return func(el Element) bool {
		if !el.InHTMLDocument() && el.Namespace() != XHTMLNamespace {
			return false
		}
		local := el.LocalName()
		for _, n := range names {
			if local == n {
				return true
			}
		}
		return false
	}
}

func hasAttr(el Element, name string) bool {
	_, ok := el.Attr("", name)
	return ok
}

func isXHTMLElement(el Element, name string) bool {
	return el.Namespace() == XHTMLNamespace && el.LocalName() == name
}

// inDisabledFieldset reports whether el is a descendant of a disabled
// fieldset, other than in its first legend (HTML Living Standard, "actually
// disabled").
func inDisabledFieldset(el Element) bool {
	for {
		parent := el.Parent()
		if parent == nil {
			return false
		}
		if isXHTMLElement(parent, "fieldset") && hasAttr(parent, "disabled") {
			if !isXHTMLElement(el, "legend") {
				return true
			}
			for s := el.PreviousSibling(); s != nil; s = s.PreviousSibling() {
				if isXHTMLElement(s, "legend") {
					return true
				}
			}
		}
		el = parent
	}
}

// hasURLScheme reports whether href starts with a URL scheme.
func hasURLScheme(href string) bool {
	href = strings.TrimLeft(href, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	href = strings.NewReplacer("\t", "", "\r", "", "\n", "").Replace(href)
	i := strings.IndexByte(href, ':')
	if i <= 0 {
		return false
	}
	if c := href[0]; !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
		return false
	}
	for j := 0; j < i; j++ {
		c := href[j]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '+', c == '-', c == '.':
		default:
			return false
		}
	}
	return true
}

// neverMatching are the pseudo-classes that depend on user interaction or on
// a rendering state, none of which exists when printing.
var neverMatching = map[string]bool{
	"visited": true, "hover": true, "active": true, "focus": true,
	"focus-within": true, "focus-visible": true, "target": true,
	"target-within": true, "current": true, "past": true, "future": true,
	"playing": true, "paused": true, "seeking": true, "buffering": true,
	"stalled": true, "muted": true, "volume-locked": true,
	"user-valid": true, "user-invalid": true, "host": true,
}

func compilePseudoClass(name string) (compiled, error) {
	if neverMatching[name] {
		return never, nil
	}
	switch name {
	case "link", "any-link":
		isLink := htmlTag("a", "area", "link")
		return predicate(func(el Element) bool { return isLink(el) && hasAttr(el, "href") }), nil
	case "local-link":
		isLink := htmlTag("a", "area", "link")
		return predicate(func(el Element) bool {
			href, ok := el.Attr("", "href")
			return isLink(el) && ok && !hasURLScheme(href)
		}), nil
	case "enabled":
		input := htmlTag("button", "input", "select", "textarea", "option")
		group := htmlTag("optgroup", "menuitem", "fieldset")
		link := htmlTag("a", "area", "link")
		return predicate(func(el Element) bool {
			return input(el) && !hasAttr(el, "disabled") && !inDisabledFieldset(el) ||
				group(el) && !hasAttr(el, "disabled") ||
				link(el) && hasAttr(el, "href")
		}), nil
	case "disabled":
		input := htmlTag("button", "input", "select", "textarea", "option")
		group := htmlTag("optgroup", "menuitem", "fieldset")
		return predicate(func(el Element) bool {
			return input(el) && (hasAttr(el, "disabled") || inDisabledFieldset(el)) ||
				group(el) && hasAttr(el, "disabled")
		}), nil
	case "checked":
		input := htmlTag("input", "menuitem")
		option := htmlTag("option")
		return predicate(func(el Element) bool {
			if input(el) && hasAttr(el, "checked") {
				typ, _ := el.Attr("", "type")
				if typ = syntax.AsciiLower(typ); typ == "checkbox" || typ == "radio" {
					return true
				}
			}
			return option(el) && hasAttr(el, "selected")
		}), nil
	case "root", "scope":
		return predicate(func(el Element) bool { return el.Parent() == nil }), nil
	case "first-child":
		return predicate(func(el Element) bool { return el.Index() == 0 }), nil
	case "last-child":
		return predicate(func(el Element) bool { return el.NextSibling() == nil }), nil
	case "first-of-type":
		return predicate(func(el Element) bool { return countOfType(el, true) == 0 }), nil
	case "last-of-type":
		return predicate(func(el Element) bool { return countOfType(el, false) == 0 }), nil
	case "only-child":
		return predicate(func(el Element) bool {
			return el.PreviousSibling() == nil && el.NextSibling() == nil
		}), nil
	case "only-of-type":
		return predicate(func(el Element) bool {
			return countOfType(el, true) == 0 && countOfType(el, false) == 0
		}), nil
	case "empty":
		return predicate(func(el Element) bool { return el.IsEmpty() }), nil
	}
	return never, errorf(nil, "Unknown pseudo-class %s", name)
}

func sameType(a, b Element) bool {
	return a.LocalName() == b.LocalName() && a.Namespace() == b.Namespace()
}

// countOfType counts the siblings with the same tag as el, before it or after
// it.
func countOfType(el Element, before bool) int {
	n := 0
	for s := adjacent(el, before); s != nil; s = adjacent(s, before) {
		if sameType(s, el) {
			n++
		}
	}
	return n
}

func adjacent(el Element, before bool) Element {
	if before {
		return el.PreviousSibling()
	}
	return el.NextSibling()
}

func compileLang(args []syntax.Node) (compiled, error) {
	invalid := errorf(nil, "Invalid arguments for :lang()")
	var tokens []syntax.Node
	for _, t := range args {
		if ty := t.Type(); ty != "whitespace" && ty != "comment" {
			tokens = append(tokens, t)
		}
	}
	var langs []string
	for len(tokens) > 0 {
		switch t := tokens[0].(type) {
		case *syntax.IdentToken:
			langs = append(langs, t.LowerValue)
		case *syntax.StringToken:
			langs = append(langs, syntax.AsciiLower(t.Value))
		default:
			return never, invalid
		}
		tokens = tokens[1:]
		if len(tokens) > 0 {
			// An identifier is accepted in place of the comma and dropped.
			_, isIdent := tokens[0].(*syntax.IdentToken)
			if !isIdent && !syntax.IsLiteral(tokens[0], ",") {
				return never, invalid
			}
			tokens = tokens[1:]
		}
	}
	if len(langs) == 0 {
		return never, invalid
	}
	return predicate(func(el Element) bool {
		lang := el.Lang()
		for _, l := range langs {
			if lang == l || strings.HasPrefix(lang, l+"-") {
				return true
			}
		}
		return false
	}), nil
}

// compileNth compiles :nth-child() and the related functional
// pseudo-classes, with the optional "of S" filter (Selectors Level 4 §14.3).
func compileNth(f *functionalPseudoClass) (compiled, error) {
	invalid := errorf(nil, "Invalid arguments for :%s()", f.name)
	var nth, of []syntax.Node
	inOf := false
	for _, arg := range f.args {
		if id, ok := arg.(*syntax.IdentToken); ok && id.Value == "of" && !inOf {
			inOf = true
			continue
		}
		if inOf {
			of = append(of, arg)
		} else {
			nth = append(nth, arg)
		}
	}

	last := f.name == "nth-last-child" || f.name == "nth-last-of-type"
	ofType := f.name == "nth-of-type" || f.name == "nth-last-of-type"
	if !last && !ofType && f.name != "nth-child" {
		return never, errorf(nil, "Unknown pseudo-class %s", f.name)
	}

	var count func(Element) (int, bool)
	if len(of) > 0 {
		// The selector list of the filter is parsed without the namespace
		// declarations of the enclosing selector.
		list, err := parseSelectorList(of, nil, false, false)
		if err != nil {
			return never, err
		}
		trees, err := compileSelectorTrees(list)
		if err != nil {
			return never, err
		}
		filter := func(el Element) bool {
			for _, t := range trees {
				if !t.eval(el) {
					return false
				}
			}
			return true
		}
		count = func(el Element) (int, bool) {
			if !filter(el) {
				return 0, false
			}
			n := 0
			for s := adjacent(el, !last); s != nil; s = adjacent(s, !last) {
				if filter(s) && (!ofType || sameType(s, el)) {
					n++
				}
			}
			return n, true
		}
	} else {
		if inOf {
			return never, invalid
		}
		switch {
		case ofType:
			count = func(el Element) (int, bool) { return countOfType(el, !last), true }
		case last:
			count = func(el Element) (int, bool) {
				n := 0
				for s := el.NextSibling(); s != nil; s = s.NextSibling() {
					n++
				}
				return n, true
			}
		default:
			count = func(el Element) (int, bool) { return el.Index(), true }
		}
	}

	a, b, ok := syntax.ParseNth(nth)
	if !ok {
		return never, invalid
	}
	// The element matches when some n >= 0 gives count = a*n + b - 1, count
	// being the number of preceding (or following) siblings.
	offset := b - 1
	return predicate(func(el Element) bool {
		n, ok := count(el)
		if !ok {
			return false
		}
		if a == 0 {
			return n == offset
		}
		d := n - offset
		return d%a == 0 && d/a >= 0
	}), nil
}
