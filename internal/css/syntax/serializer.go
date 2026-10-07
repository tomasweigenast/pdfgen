package syntax

import (
	"strconv"
	"strings"
)

// Serialize converts nodes back to CSS text. Unlike serializing each node on
// its own, it inserts the separators needed to keep adjacent tokens from
// merging when reparsed (CSS Syntax 3 §9) and the semicolons between
// declarations.
func Serialize(nodes []Node) string {
	var b strings.Builder
	serializeNodes(nodes, &b)
	return b.String()
}

func upperHex(n int) string { return strings.ToUpper(strconv.FormatInt(int64(n), 16)) }

func isNameStartOrNonASCII(c rune) bool {
	return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c > 0x7F
}

func isNameChar(c rune) bool {
	return isNameStartOrNonASCII(c) || c == '-' || '0' <= c && c <= '9'
}

// SerializeIdentifier serializes s so that it reparses as an identifier
// whose value is s.
func SerializeIdentifier(s string) string {
	if s == "" {
		return ""
	}
	if s == "-" {
		return `\-`
	}
	if strings.HasPrefix(s, "--") {
		return "--" + SerializeName(s[2:])
	}
	var b strings.Builder
	if s[0] == '-' {
		b.WriteByte('-')
		s = s[1:]
	}
	first := true
	for _, c := range s {
		if !first {
			writeNameRune(&b, c)
			continue
		}
		first = false
		switch {
		case isNameStartOrNonASCII(c):
			b.WriteRune(c)
		case c == '\n':
			b.WriteString(`\A `)
		case c == '\r':
			b.WriteString(`\D `)
		case c == '\f':
			b.WriteString(`\C `)
		case '0' <= c && c <= '9':
			b.WriteString(`\` + upperHex(int(c)) + " ")
		default:
			b.WriteByte('\\')
			b.WriteRune(c)
		}
	}
	return b.String()
}

func writeNameRune(b *strings.Builder, c rune) {
	switch {
	case isNameChar(c):
		b.WriteRune(c)
	case c == '\n':
		b.WriteString(`\A `)
	case c == '\r':
		b.WriteString(`\D `)
	case c == '\f':
		b.WriteString(`\C `)
	default:
		b.WriteByte('\\')
		b.WriteRune(c)
	}
}

// SerializeName serializes s as the body of an identifier-like token such as
// a hash name.
func SerializeName(s string) string {
	var b strings.Builder
	for _, c := range s {
		writeNameRune(&b, c)
	}
	return b.String()
}

// SerializeStringValue escapes s for use between double quotes.
func SerializeStringValue(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\A `)
		case '\r':
			b.WriteString(`\D `)
		case '\f':
			b.WriteString(`\C `)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// SerializeURL escapes s for use inside an unquoted url().
func SerializeURL(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch c {
		case '\'':
			b.WriteString(`\'`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case ' ':
			b.WriteString(`\ `)
		case '\t':
			b.WriteString(`\9 `)
		case '\n':
			b.WriteString(`\A `)
		case '\r':
			b.WriteString(`\D `)
		case '\f':
			b.WriteString(`\C `)
		case '(':
			b.WriteString(`\(`)
		case ')':
			b.WriteString(`\)`)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

func serializationType(n Node) string {
	if l, ok := n.(*LiteralToken); ok {
		return l.Value
	}
	return n.Type()
}

func serializeNodes(nodes []Node, w *strings.Builder) {
	previous := ""
	for _, n := range nodes {
		t := serializationType(n)
		if badPairs[[2]string{previous, t}] {
			w.WriteString("/**/")
		} else if previous == `\` {
			ws, ok := n.(*WhitespaceToken)
			if !ok || !strings.HasPrefix(ws.Value, "\n") {
				w.WriteByte('\n')
			}
		}
		n.serializeTo(w)
		if t == "declaration" {
			w.WriteByte(';')
		}
		previous = t
	}
}

// badPairs lists the token sequences that need a comment between them to
// survive a round trip (CSS Syntax 3 §9, serialization table).
var badPairs = func() map[[2]string]bool {
	m := map[[2]string]bool{}
	add := func(as, bs []string) {
		for _, a := range as {
			for _, b := range bs {
				m[[2]string{a, b}] = true
			}
		}
	}
	add([]string{"ident", "at-keyword", "hash", "dimension", "#", "-", "number"},
		[]string{"ident", "function", "url", "number", "percentage", "dimension", "unicode-range"})
	add([]string{"ident", "at-keyword", "hash", "dimension"}, []string{"-", "-->"})
	add([]string{"#", "-", "number", "@"}, []string{"ident", "function", "url"})
	add([]string{"unicode-range", ".", "+"}, []string{"number", "percentage", "dimension"})
	add([]string{"@"}, []string{"ident", "function", "url", "unicode-range", "-"})
	add([]string{"unicode-range"}, []string{"ident", "function", "?"})
	add([]string{"$", "*", "^", "~", "|"}, []string{"="})
	add([]string{"ident"}, []string{"() block"})
	add([]string{"|"}, []string{"|"})
	add([]string{"/"}, []string{"*"})
	return m
}()
