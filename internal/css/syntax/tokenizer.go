package syntax

import (
	"strconv"
	"strings"
)

// ParseComponentValueList tokenizes css into a list of component values
// (CSS Syntax 3 §5.4.7, §4). When skipComments is true no Comment nodes are
// produced, at any depth.
//
// Newlines are normalized before tokenizing, so positions refer to the
// normalized text. Columns count code points.
func ParseComponentValueList(css string, skipComments bool) []Node {
	src := normalizeInput(css)
	length := len(src)
	root := []Node{}
	cur := &root
	var endChar rune = -1
	type frame struct {
		list *[]Node
		end  rune
	}
	var stack []frame
	push := func(n Node, list *[]Node, end rune) {
		*cur = append(*cur, n)
		stack = append(stack, frame{cur, endChar})
		endChar = end
		cur = list
	}
	add := func(n Node) { *cur = append(*cur, n) }

	tokenStart, pos := 0, 0
	line := 1
	lastNewline := -1

	for pos < length {
		// Newlines consumed by the previous token.
		for i := tokenStart; i < pos; i++ {
			if src[i] == '\n' {
				line++
				lastNewline = i
			}
		}
		column := pos - lastNewline
		tokenStart = pos
		c := src[pos]

		switch {
		case isCSSSpace(c):
			pos++
			for pos < length && isCSSSpace(src[pos]) {
				pos++
			}
			add(NewWhitespaceToken(line, column, string(src[tokenStart:pos])))
			continue
		case (c == 'U' || c == 'u') && pos+2 < length && src[pos+1] == '+' &&
			(isHexDigit(src[pos+2]) || src[pos+2] == '?'):
			var start, end int
			start, end, pos = consumeUnicodeRange(src, pos+2)
			add(NewUnicodeRangeToken(line, column, start, end))
			continue
		case hasPrefixAt(src, pos, "-->"):
			add(NewLiteralToken(line, column, "-->"))
			pos += 3
			continue
		case isIdentStart(src, pos):
			var value string
			value, pos = consumeIdent(src, pos)
			if pos >= length || src[pos] != '(' {
				add(NewIdentToken(line, column, value))
				continue
			}
			pos++
			if AsciiLower(value) == "url" {
				urlPos := pos
				for urlPos < length && isCSSSpace(src[urlPos]) {
					urlPos++
				}
				if urlPos >= length || (src[urlPos] != '"' && src[urlPos] != '\'') {
					var v string
					var ok bool
					var perr *ParseError
					v, ok, pos, perr = consumeURL(src, pos)
					if ok {
						repr := "url(" + SerializeURL(v) + ")"
						if perr != nil {
							if perr.Kind == "eof-in-string" {
								repr = repr[:len(repr)-2]
							} else {
								repr = repr[:len(repr)-1]
							}
						}
						add(NewURLToken(line, column, v, repr))
					}
					if perr != nil {
						perr.Line, perr.Column = line, column
						add(perr)
					}
					continue
				}
			}
			fn := NewFunctionBlock(line, column, value, []Node{})
			push(fn, &fn.Arguments, ')')
			continue
		}

		if end, ok := matchNumber(src, pos); ok {
			repr := string(src[pos:end])
			pos = end
			num := makeNumeric(repr)
			switch {
			case pos < length && isIdentStart(src, pos):
				var unit string
				unit, pos = consumeIdent(src, pos)
				add(NewDimensionToken(line, column, num, unit))
			case pos < length && src[pos] == '%':
				pos++
				add(NewPercentageToken(line, column, num))
			default:
				add(NewNumberToken(line, column, num))
			}
			continue
		}

		switch {
		case c == '@':
			pos++
			if pos < length && isIdentStart(src, pos) {
				var value string
				value, pos = consumeIdent(src, pos)
				add(NewAtKeywordToken(line, column, value))
			} else {
				add(NewLiteralToken(line, column, "@"))
			}
		case c == '#':
			pos++
			if pos < length && (isNameChar(src[pos]) ||
				(src[pos] == '\\' && !hasPrefixAt(src, pos, "\\\n"))) {
				isIdent := isIdentStart(src, pos)
				var value string
				value, pos = consumeIdent(src, pos)
				add(NewHashToken(line, column, value, isIdent))
			} else {
				add(NewLiteralToken(line, column, "#"))
			}
		case c == '{':
			b := NewCurlyBracketsBlock(line, column, []Node{})
			push(b, &b.Content, '}')
			pos++
		case c == '[':
			b := NewSquareBracketsBlock(line, column, []Node{})
			push(b, &b.Content, ']')
			pos++
		case c == '(':
			b := NewParenthesesBlock(line, column, []Node{})
			push(b, &b.Content, ')')
			pos++
		case c == endChar:
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			cur, endChar = top.list, top.end
			pos++
		case c == '}' || c == ']' || c == ')':
			add(NewParseError(line, column, string(c), "Unmatched "+string(c)))
			pos++
		case c == '"' || c == '\'':
			value, ok, newPos, perr := consumeQuotedString(src, pos)
			pos = newPos
			if ok {
				repr := `"` + SerializeStringValue(value) + `"`
				if perr != nil {
					repr = repr[:len(repr)-1]
				}
				add(NewStringToken(line, column, value, repr))
			}
			if perr != nil {
				perr.Line, perr.Column = line, column
				add(perr)
			}
		case hasPrefixAt(src, pos, "/*"):
			end := indexAt(src, pos+2, "*/")
			if end == -1 {
				if !skipComments {
					add(NewComment(line, column, string(src[tokenStart+2:])))
				}
				return root
			}
			if !skipComments {
				add(NewComment(line, column, string(src[tokenStart+2:end])))
			}
			pos = end + 2
		case hasPrefixAt(src, pos, "<!--"):
			add(NewLiteralToken(line, column, "<!--"))
			pos += 4
		case hasPrefixAt(src, pos, "||"):
			add(NewLiteralToken(line, column, "||"))
			pos += 2
		case strings.ContainsRune("~|^$*", c):
			pos++
			if pos < length && src[pos] == '=' {
				pos++
				add(NewLiteralToken(line, column, string(c)+"="))
			} else {
				add(NewLiteralToken(line, column, string(c)))
			}
		default:
			add(NewLiteralToken(line, column, string(c)))
			pos++
		}
	}
	return root
}

// normalizeInput applies the preprocessing of CSS Syntax 3 §3.3 and returns
// the code points.
func normalizeInput(css string) []rune {
	in := []rune(css)
	out := make([]rune, 0, len(in))
	for i := 0; i < len(in); i++ {
		switch c := in[i]; c {
		case 0:
			out = append(out, '�')
		case '\r':
			if i+1 < len(in) && in[i+1] == '\n' {
				i++
			}
			out = append(out, '\n')
		case '\f':
			out = append(out, '\n')
		default:
			out = append(out, c)
		}
	}
	return out
}

func isCSSSpace(c rune) bool { return c == ' ' || c == '\n' || c == '\t' }

func isHexDigit(c rune) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func hasPrefixAt(src []rune, pos int, s string) bool {
	for _, r := range s {
		if pos >= len(src) || src[pos] != r {
			return false
		}
		pos++
	}
	return true
}

func indexAt(src []rune, from int, s string) int {
	for i := from; i < len(src); i++ {
		if hasPrefixAt(src, i, s) {
			return i
		}
	}
	return -1
}

// matchNumber matches [-+]?([0-9]*\.)?[0-9]+([eE][+-]?[0-9]+)? at pos.
func matchNumber(src []rune, pos int) (end int, ok bool) {
	i := pos
	n := len(src)
	if i < n && (src[i] == '-' || src[i] == '+') {
		i++
	}
	digits := func(j int) int {
		for j < n && '0' <= src[j] && src[j] <= '9' {
			j++
		}
		return j
	}
	j := digits(i)
	if j < n && src[j] == '.' && digits(j+1) > j+1 {
		i = digits(j + 1)
	} else if j > i {
		i = j
	} else {
		return 0, false
	}
	if i < n && (src[i] == 'e' || src[i] == 'E') {
		k := i + 1
		if k < n && (src[k] == '+' || src[k] == '-') {
			k++
		}
		if e := digits(k); e > k {
			i = e
		}
	}
	return i, true
}

func makeNumeric(repr string) Numeric {
	v, _ := strconv.ParseFloat(repr, 64)
	num := Numeric{Value: v, Representation: repr}
	if !strings.ContainsAny(repr, ".eE") {
		num.IsInteger = true
		iv, err := strconv.ParseInt(repr, 10, 64)
		if err != nil {
			if repr[0] == '-' {
				iv = -1 << 63
			} else {
				iv = 1<<63 - 1
			}
		}
		num.IntValue = int(iv)
	}
	return num
}

func isNameStart(c rune) bool { return isNameStartOrNonASCII(c) }

// isIdentStart reports whether an identifier starts at pos (CSS Syntax 3
// §4.3.9).
func isIdentStart(src []rune, pos int) bool {
	c := src[pos]
	switch {
	case isNameStart(c):
		return true
	case c == '-':
		pos++
		return (pos < len(src) && (isNameStart(src[pos]) || src[pos] == '-')) ||
			(pos < len(src) && src[pos] == '\\' && !hasPrefixAt(src, pos, "\\\n"))
	case c == '\\':
		return !hasPrefixAt(src, pos, "\\\n")
	}
	return false
}

// consumeIdent consumes a name starting at a valid identifier start (CSS
// Syntax 3 §4.3.11) and returns it unescaped.
func consumeIdent(src []rune, pos int) (string, int) {
	var b strings.Builder
	start := pos
	for pos < len(src) {
		c := src[pos]
		if isNameChar(c) {
			pos++
		} else if c == '\\' && !hasPrefixAt(src, pos, "\\\n") {
			b.WriteString(string(src[start:pos]))
			var r rune
			r, pos = consumeEscape(src, pos+1)
			b.WriteRune(r)
			start = pos
		} else {
			break
		}
	}
	b.WriteString(string(src[start:pos]))
	return b.String(), pos
}

// consumeQuotedString consumes a string token (CSS Syntax 3 §4.3.5). ok is
// false for a bad string, which has no value.
func consumeQuotedString(src []rune, pos int) (value string, ok bool, newPos int, perr *ParseError) {
	quote := src[pos]
	pos++
	var b strings.Builder
	start := pos
	for pos < len(src) {
		c := src[pos]
		switch {
		case c == quote:
			b.WriteString(string(src[start:pos]))
			return b.String(), true, pos + 1, nil
		case c == '\\':
			b.WriteString(string(src[start:pos]))
			pos++
			if pos < len(src) {
				if src[pos] == '\n' {
					pos++
				} else {
					var r rune
					r, pos = consumeEscape(src, pos)
					b.WriteRune(r)
				}
			}
			start = pos
		case c == '\n':
			return "", false, pos, NewParseError(0, 0, "bad-string", "Bad string token")
		default:
			pos++
		}
	}
	b.WriteString(string(src[start:pos]))
	return b.String(), true, pos, NewParseError(0, 0, "eof-in-string", "EOF in string")
}

// consumeEscape consumes an escape; pos is just after the backslash, which
// is not followed by a newline (CSS Syntax 3 §4.3.7).
func consumeEscape(src []rune, pos int) (rune, int) {
	i := pos
	for i < len(src) && i-pos < 6 && isHexDigit(src[i]) {
		i++
	}
	if i > pos {
		cp, _ := strconv.ParseInt(string(src[pos:i]), 16, 32)
		if i < len(src) && isCSSSpace(src[i]) {
			i++
		}
		if cp == 0 || cp > 0x10FFFF || (0xD800 <= cp && cp <= 0xDFFF) {
			return '�', i
		}
		return rune(cp), i
	}
	if pos < len(src) {
		return src[pos], pos + 1
	}
	return '�', pos
}

// consumeURL consumes an unquoted URL; pos is just after "url(" (CSS Syntax 3
// §4.3.6). ok is false for a bad URL.
func consumeURL(src []rune, pos int) (value string, ok bool, newPos int, perr *ParseError) {
	length := len(src)
	eof := func() *ParseError { return NewParseError(0, 0, "eof-in-url", "EOF in URL") }
	for pos < length && isCSSSpace(src[pos]) {
		pos++
	}
	if pos >= length {
		return "", true, pos, eof()
	}
	var val string
	valOK := false
	switch c := src[pos]; {
	case c == '"' || c == '\'':
		val, valOK, pos, perr = consumeQuotedString(src, pos)
	case c == ')':
		return "", true, pos + 1, nil
	default:
		var b strings.Builder
		start := pos
	loop:
		for {
			if pos >= length {
				b.WriteString(string(src[start:pos]))
				return b.String(), true, pos, eof()
			}
			c := src[pos]
			switch {
			case c == ')':
				b.WriteString(string(src[start:pos]))
				return b.String(), true, pos + 1, perr
			case isCSSSpace(c):
				b.WriteString(string(src[start:pos]))
				val, valOK = b.String(), true
				pos++
				break loop
			case c == '\\' && !hasPrefixAt(src, pos, "\\\n"):
				b.WriteString(string(src[start:pos]))
				var r rune
				r, pos = consumeEscape(src, pos+1)
				b.WriteRune(r)
				start = pos
			case isURLForbidden(c):
				valOK = false
				pos++
				break loop
			default:
				pos++
			}
		}
	}

	if valOK {
		for pos < length && isCSSSpace(src[pos]) {
			pos++
		}
		if pos < length {
			if src[pos] == ')' {
				return val, true, pos + 1, perr
			}
		} else {
			if perr == nil {
				perr = eof()
			}
			return val, true, pos, perr
		}
	}

	// Remnants of a bad URL (CSS Syntax 3 §4.3.14).
	for pos < length {
		if hasPrefixAt(src, pos, "\\)") {
			pos += 2
		} else if src[pos] == ')' {
			pos++
			break
		} else {
			pos++
		}
	}
	return "", false, pos, NewParseError(0, 0, "bad-url", "bad URL token")
}

func isURLForbidden(c rune) bool {
	switch {
	case c == '"' || c == '\'' || c == '(':
		return true
	case c <= 0x08 || c == 0x0b || (0x0e <= c && c <= 0x1f) || c == 0x7f:
		return true
	}
	return false
}

// consumeUnicodeRange consumes the part of a unicode-range token after "U+"
// (CSS Syntax 3 §4.3.8).
func consumeUnicodeRange(src []rune, pos int) (start, end, newPos int) {
	length := len(src)
	startPos := pos
	maxPos := min(pos+6, length)
	for pos < maxPos && isHexDigit(src[pos]) {
		pos++
	}
	startStr := string(src[startPos:pos])
	startPos = pos
	for pos < maxPos && src[pos] == '?' {
		pos++
	}
	questions := pos - startPos
	var endStr string
	switch {
	case questions > 0:
		endStr = startStr + strings.Repeat("F", questions)
		startStr += strings.Repeat("0", questions)
	case pos+1 < length && src[pos] == '-' && isHexDigit(src[pos+1]):
		pos++
		startPos = pos
		maxPos = min(pos+6, length)
		for pos < maxPos && isHexDigit(src[pos]) {
			pos++
		}
		endStr = string(src[startPos:pos])
	default:
		endStr = startStr
	}
	s, _ := strconv.ParseInt(startStr, 16, 64)
	e, _ := strconv.ParseInt(endStr, 16, 64)
	return int(s), int(e), pos
}
