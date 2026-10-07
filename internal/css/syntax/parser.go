package syntax

import "fmt"

// Input is the input of the parser entry points: CSS text, which is
// tokenized first, or already tokenized component values. For a node list,
// the skipComments argument has no effect.
type Input interface {
	string | []Node
}

// tokenIter walks a list of nodes; pre holds nodes to yield before items.
type tokenIter struct {
	pre   []Node
	items []Node
	pos   int
}

func (it *tokenIter) next() (Node, bool) {
	if len(it.pre) > 0 {
		n := it.pre[0]
		it.pre = it.pre[1:]
		return n, true
	}
	if it.pos >= len(it.items) {
		return nil, false
	}
	n := it.items[it.pos]
	it.pos++
	return n, true
}

func toIter[T Input](input T, skipComments bool) *tokenIter {
	switch v := any(input).(type) {
	case string:
		return &tokenIter{items: ParseComponentValueList(v, skipComments)}
	case []Node:
		return &tokenIter{items: v}
	}
	return &tokenIter{}
}

func isBlank(n Node) bool {
	switch n.(type) {
	case *WhitespaceToken, *Comment:
		return true
	}
	return false
}

func (it *tokenIter) nextSignificant() Node {
	for {
		n, ok := it.next()
		if !ok {
			return nil
		}
		if !isBlank(n) {
			return n
		}
	}
}

// ParseOneComponentValue parses a single component value, ignoring
// surrounding whitespace and comments. It returns a ParseError node when the
// input is empty or holds more than one value.
func ParseOneComponentValue[T Input](input T, skipComments bool) Node {
	it := toIter(input, skipComments)
	first := it.nextSignificant()
	second := it.nextSignificant()
	if first == nil {
		return NewParseError(1, 1, "empty", "Input is empty")
	}
	if second != nil {
		l, c := second.Pos()
		return NewParseError(l, c, "extra-input", "Got more than one token")
	}
	return first
}

// ParseOneDeclaration parses a single declaration, returning a *Declaration
// or a *ParseError. Whitespace and comments before the colon are dropped.
func ParseOneDeclaration[T Input](input T, skipComments bool) Node {
	it := toIter(input, skipComments)
	first := it.nextSignificant()
	if first == nil {
		return NewParseError(1, 1, "empty", "Input is empty")
	}
	return parseDeclaration(first, it)
}

func consumeRemnants(it *tokenIter) {
	for {
		n, ok := it.next()
		if !ok || IsLiteral(n, ";") || IsLiteral(n, "}") {
			return
		}
	}
}

// parseDeclaration consumes it until the end of the declaration or the
// first error (CSS Syntax 3 §5.4.5).
func parseDeclaration(name Node, it *tokenIter) Node {
	nl, nc := name.Pos()
	ident, ok := name.(*IdentToken)
	if !ok {
		consumeRemnants(it)
		return NewParseError(nl, nc, "invalid",
			fmt.Sprintf("Expected <ident> for declaration name, got %s.", name.Type()))
	}

	colon := it.nextSignificant()
	if colon == nil {
		consumeRemnants(it)
		return NewParseError(nl, nc, "invalid", "Expected ':' after declaration name, got EOF")
	}
	cl, cc := colon.Pos()
	if !IsLiteral(colon, ":") {
		consumeRemnants(it)
		return NewParseError(cl, cc, "invalid",
			fmt.Sprintf("Expected ':' after declaration name, got %s.", colon.Type()))
	}

	const (
		stValue = iota
		stBang
		stImportant
	)
	value := []Node{}
	state := stValue
	nonWhitespace, simpleBlock := false, false
	bangPos := 0
	for i := 0; ; i++ {
		tok, ok := it.next()
		if !ok {
			break
		}
		switch {
		case state == stValue && IsLiteral(tok, "!"):
			state = stBang
			bangPos = i
		case state == stBang && isImportant(tok):
			state = stImportant
		case !isBlank(tok):
			state = stValue
			if _, isBlock := tok.(*CurlyBracketsBlock); isBlock && !nonWhitespace {
				nonWhitespace = true
			} else if isBlock {
				simpleBlock = true
			} else {
				nonWhitespace = true
			}
		}
		value = append(value, tok)
	}
	if state == stImportant {
		value = value[:bangPos]
	}
	if simpleBlock && nonWhitespace {
		return NewParseError(cl, cc, "invalid", "Declaration contains {} block")
	}
	d := NewDeclaration(nl, nc, ident.Value, value, state == stImportant)
	return d
}

func isImportant(n Node) bool {
	id, ok := n.(*IdentToken)
	return ok && id.LowerValue == "important"
}

func isCurly(n Node) (*CurlyBracketsBlock, bool) {
	b, ok := n.(*CurlyBracketsBlock)
	return b, ok
}

// consumeBlocksContent consumes either a declaration or a nested rule
// (CSS Syntax 3 §5.4.4).
func consumeBlocksContent(first Node, it *tokenIter) Node {
	var declTokens, semicolon []Node
	if _, curly := isCurly(first); !IsLiteral(first, ";") && !curly {
		for {
			tok, ok := it.next()
			if !ok {
				break
			}
			if IsLiteral(tok, ";") {
				semicolon = append(semicolon, tok)
				break
			}
			declTokens = append(declTokens, tok)
			if _, curly := isCurly(tok); curly {
				break
			}
		}
	}
	decl := parseDeclaration(first, &tokenIter{items: declTokens})
	if _, isDecl := decl.(*Declaration); isDecl {
		return decl
	}
	pre := make([]Node, 0, len(declTokens)+len(semicolon))
	pre = append(append(pre, declTokens...), semicolon...)
	chained := &tokenIter{pre: pre, items: it.items, pos: it.pos}
	rule := consumeQualifiedRule(first, chained, ";")
	it.pos = chained.pos
	return rule
}

// ParseBlocksContents parses the contents of a block: declarations, at-rules
// and nested qualified rules (CSS Syntax 3 §5.4.4). It is used for style rule
// bodies, @page rules and style attributes.
//
// Contexts that accept only some of those must reject the others as invalid.
// With skipWhitespace, top-level whitespace is dropped; whitespace inside
// declaration values and rules is kept.
func ParseBlocksContents[T Input](input T, skipComments, skipWhitespace bool) []Node {
	return parseList(input, skipComments, skipWhitespace, func(tok Node, it *tokenIter) (Node, bool) {
		switch {
		case isAtKeyword(tok):
			return consumeAtRule(tok.(*AtKeywordToken), it), true
		case !IsLiteral(tok, ";"):
			return consumeBlocksContent(tok, it), true
		}
		return nil, false
	})
}

// ParseDeclarationList parses a declaration list that may contain at-rules.
// It is the legacy form of ParseBlocksContents, which should be preferred.
func ParseDeclarationList[T Input](input T, skipComments, skipWhitespace bool) []Node {
	return parseList(input, skipComments, skipWhitespace, func(tok Node, it *tokenIter) (Node, bool) {
		switch {
		case isAtKeyword(tok):
			return consumeAtRule(tok.(*AtKeywordToken), it), true
		case !IsLiteral(tok, ";"):
			var rest []Node
			for {
				t, ok := it.next()
				if !ok || IsLiteral(t, ";") {
					break
				}
				rest = append(rest, t)
			}
			return parseDeclaration(tok, &tokenIter{items: rest}), true
		}
		return nil, false
	})
}

// ParseRuleList parses a non-top-level rule list, such as the content of
// @media. Unlike ParseStylesheet it keeps top-level <!-- and -->.
func ParseRuleList[T Input](input T, skipComments, skipWhitespace bool) []Node {
	return parseList(input, skipComments, skipWhitespace, func(tok Node, it *tokenIter) (Node, bool) {
		return consumeRule(tok, it), true
	})
}

// ParseStylesheet parses a stylesheet. Top-level <!-- and --> tokens are
// ignored, a legacy quirk of the <style> element.
func ParseStylesheet[T Input](input T, skipComments, skipWhitespace bool) []Node {
	return parseList(input, skipComments, skipWhitespace, func(tok Node, it *tokenIter) (Node, bool) {
		if IsLiteral(tok, "<!--") || IsLiteral(tok, "-->") {
			return nil, false
		}
		return consumeRule(tok, it), true
	})
}

func parseList[T Input](input T, skipComments, skipWhitespace bool,
	consume func(Node, *tokenIter) (Node, bool)) []Node {
	it := toIter(input, skipComments)
	result := []Node{}
	for {
		tok, ok := it.next()
		if !ok {
			return result
		}
		switch tok.(type) {
		case *WhitespaceToken:
			if !skipWhitespace {
				result = append(result, tok)
			}
		case *Comment:
			if !skipComments {
				result = append(result, tok)
			}
		default:
			if n, ok := consume(tok, it); ok {
				result = append(result, n)
			}
		}
	}
}

func isAtKeyword(n Node) bool {
	_, ok := n.(*AtKeywordToken)
	return ok
}

// ParseOneRule parses a single qualified rule or at-rule. Whitespace and
// comments around the rule are dropped.
func ParseOneRule[T Input](input T, skipComments bool) Node {
	it := toIter(input, skipComments)
	first := it.nextSignificant()
	if first == nil {
		return NewParseError(1, 1, "empty", "Input is empty")
	}
	rule := consumeRule(first, it)
	if next := it.nextSignificant(); next != nil {
		l, c := next.Pos()
		return NewParseError(l, c, "extra-input",
			fmt.Sprintf("Expected a single rule, got %s after the first rule.", next.Type()))
	}
	return rule
}

func consumeRule(first Node, it *tokenIter) Node {
	if at, ok := first.(*AtKeywordToken); ok {
		return consumeAtRule(at, it)
	}
	return consumeQualifiedRule(first, it, "")
}

func consumeAtRule(at *AtKeywordToken, it *tokenIter) Node {
	prelude := []Node{}
	var content []Node
	for {
		tok, ok := it.next()
		if !ok {
			break
		}
		if b, ok := isCurly(tok); ok {
			content = b.Content
			break
		}
		if IsLiteral(tok, ";") {
			break
		}
		prelude = append(prelude, tok)
	}
	return NewAtRule(at.Line, at.Column, at.Value, prelude, content)
}

func ruleError(tok Node, name string) Node {
	l, c := tok.Pos()
	return NewParseError(l, c, "invalid", name+" reached before {} block for a qualified rule.")
}

// consumeQualifiedRule consumes a qualified rule (CSS Syntax 3 §5.4.3).
// stop, when not empty, is a literal that ends parsing with an error.
func consumeQualifiedRule(first Node, it *tokenIter, stop string) Node {
	isStop := func(n Node) bool { return stop != "" && IsLiteral(n, stop) }
	if isStop(first) {
		return ruleError(first, "Stop token")
	}
	var prelude []Node
	var block *CurlyBracketsBlock
	if b, ok := isCurly(first); ok {
		prelude = []Node{}
		block = b
	} else {
		prelude = []Node{first}
		for {
			tok, ok := it.next()
			if !ok {
				return ruleError(prelude[len(prelude)-1], "EOF")
			}
			if isStop(tok) {
				return ruleError(tok, "Stop token")
			}
			if b, ok := isCurly(tok); ok {
				block = b
				break
			}
			prelude = append(prelude, tok)
		}
	}
	l, c := first.Pos()
	return NewQualifiedRule(l, c, prelude, block.Content)
}
