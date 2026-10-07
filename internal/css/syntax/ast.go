package syntax

import "strings"

// Node is any element of the CSS syntax tree: a token, a block, a rule, a
// declaration, a comment or a parse error. Concrete types are used through
// pointers.
type Node interface {
	// Pos returns the 1-based line and column of the start of the node in
	// the source.
	Pos() (line, column int)
	// Type returns the node kind: "ident", "at-keyword", "hash", "string",
	// "url", "unicode-range", "number", "percentage", "dimension",
	// "whitespace", "comment", "literal", "function", "() block",
	// "[] block", "{} block", "declaration", "qualified-rule", "at-rule" or
	// "error".
	Type() string
	// Serialize returns the CSS text of the node.
	Serialize() string

	serializeTo(w *strings.Builder)
}

// Position is the source location shared by every node.
type Position struct {
	Line, Column int
}

// Pos implements Node.
func (p Position) Pos() (line, column int) { return p.Line, p.Column }

func serializeNode(n Node) string {
	var b strings.Builder
	n.serializeTo(&b)
	return b.String()
}

// AsciiLower lowercases only the ASCII letters of s, as CSS keywords are
// matched.
func AsciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if c := s[i]; 'A' <= c && c <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if 'A' <= b[j] && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}

// ParseError is a syntax error. It may occur anywhere in the tree.
type ParseError struct {
	Position
	// Kind is a machine-readable identifier such as "bad-url".
	Kind string
	// Message is a human-readable explanation.
	Message string
}

// NewParseError returns a ParseError node.
func NewParseError(line, column int, kind, message string) *ParseError {
	return &ParseError{Position{line, column}, kind, message}
}

func (*ParseError) Type() string        { return "error" }
func (n *ParseError) Serialize() string { return serializeNode(n) }
func (n *ParseError) serializeTo(w *strings.Builder) {
	switch n.Kind {
	case "bad-string":
		w.WriteString("\"[bad string]\n")
	case "bad-url":
		w.WriteString("url([bad url])")
	case ")", "]", "}":
		w.WriteString(n.Kind)
	}
}

// Comment is a CSS comment.
type Comment struct {
	Position
	// Value is the text between /* and */.
	Value string
}

// NewComment returns a Comment node.
func NewComment(line, column int, value string) *Comment {
	return &Comment{Position{line, column}, value}
}

func (*Comment) Type() string        { return "comment" }
func (n *Comment) Serialize() string { return serializeNode(n) }
func (n *Comment) serializeTo(w *strings.Builder) {
	w.WriteString("/*")
	w.WriteString(n.Value)
	w.WriteString("*/")
}

// WhitespaceToken is a run of whitespace.
type WhitespaceToken struct {
	Position
	// Value is the whitespace as found in the (newline-normalized) source.
	Value string
}

// NewWhitespaceToken returns a WhitespaceToken node.
func NewWhitespaceToken(line, column int, value string) *WhitespaceToken {
	return &WhitespaceToken{Position{line, column}, value}
}

func (*WhitespaceToken) Type() string        { return "whitespace" }
func (n *WhitespaceToken) Serialize() string { return serializeNode(n) }
func (n *WhitespaceToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Value)
}

// LiteralToken is one to four characters standing for a colon, semicolon,
// comma, CDO, CDC, a match operator, the column token or a delimiter.
type LiteralToken struct {
	Position
	Value string
}

// NewLiteralToken returns a LiteralToken node.
func NewLiteralToken(line, column int, value string) *LiteralToken {
	return &LiteralToken{Position{line, column}, value}
}

func (*LiteralToken) Type() string        { return "literal" }
func (n *LiteralToken) Serialize() string { return serializeNode(n) }
func (n *LiteralToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Value)
}

// IsLiteral reports whether n is a LiteralToken with the given value.
func IsLiteral(n Node, value string) bool {
	l, ok := n.(*LiteralToken)
	return ok && l.Value == value
}

// IdentToken is an identifier.
type IdentToken struct {
	Position
	// Value is the unescaped identifier.
	Value string
	// LowerValue is Value in ASCII lower case, for keyword comparison.
	LowerValue string
}

// NewIdentToken returns an IdentToken node.
func NewIdentToken(line, column int, value string) *IdentToken {
	return &IdentToken{Position{line, column}, value, AsciiLower(value)}
}

func (*IdentToken) Type() string        { return "ident" }
func (n *IdentToken) Serialize() string { return serializeNode(n) }
func (n *IdentToken) serializeTo(w *strings.Builder) {
	w.WriteString(SerializeIdentifier(n.Value))
}

// AtKeywordToken is an at-keyword such as @media.
type AtKeywordToken struct {
	Position
	// Value is the unescaped keyword without the leading @.
	Value      string
	LowerValue string
}

// NewAtKeywordToken returns an AtKeywordToken node.
func NewAtKeywordToken(line, column int, value string) *AtKeywordToken {
	return &AtKeywordToken{Position{line, column}, value, AsciiLower(value)}
}

func (*AtKeywordToken) Type() string        { return "at-keyword" }
func (n *AtKeywordToken) Serialize() string { return serializeNode(n) }
func (n *AtKeywordToken) serializeTo(w *strings.Builder) {
	w.WriteByte('@')
	w.WriteString(SerializeIdentifier(n.Value))
}

// HashToken is a # followed by a name.
type HashToken struct {
	Position
	// Value is the unescaped name without the leading #.
	Value string
	// IsIdentifier is true when the name is a valid identifier, which is
	// the only form valid as an ID selector.
	IsIdentifier bool
}

// NewHashToken returns a HashToken node.
func NewHashToken(line, column int, value string, isIdentifier bool) *HashToken {
	return &HashToken{Position{line, column}, value, isIdentifier}
}

func (*HashToken) Type() string        { return "hash" }
func (n *HashToken) Serialize() string { return serializeNode(n) }
func (n *HashToken) serializeTo(w *strings.Builder) {
	w.WriteByte('#')
	if n.IsIdentifier {
		w.WriteString(SerializeIdentifier(n.Value))
	} else {
		w.WriteString(SerializeName(n.Value))
	}
}

// StringToken is a quoted string.
type StringToken struct {
	Position
	// Value is the unescaped content without quotes.
	Value string
	// Representation is the serialized form, quotes included.
	Representation string
}

// NewStringToken returns a StringToken node.
func NewStringToken(line, column int, value, representation string) *StringToken {
	return &StringToken{Position{line, column}, value, representation}
}

func (*StringToken) Type() string        { return "string" }
func (n *StringToken) Serialize() string { return serializeNode(n) }
func (n *StringToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Representation)
}

// URLToken is an unquoted url(...) token.
type URLToken struct {
	Position
	// Value is the unescaped URL.
	Value string
	// Representation is the serialized form, url( and ) included.
	Representation string
}

// NewURLToken returns a URLToken node.
func NewURLToken(line, column int, value, representation string) *URLToken {
	return &URLToken{Position{line, column}, value, representation}
}

func (*URLToken) Type() string        { return "url" }
func (n *URLToken) Serialize() string { return serializeNode(n) }
func (n *URLToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Representation)
}

// UnicodeRangeToken is a unicode-range token such as U+0025-00FF.
type UnicodeRangeToken struct {
	Position
	// Start and End bound the range; they are equal for a single code point.
	Start, End int
}

// NewUnicodeRangeToken returns a UnicodeRangeToken node.
func NewUnicodeRangeToken(line, column, start, end int) *UnicodeRangeToken {
	return &UnicodeRangeToken{Position{line, column}, start, end}
}

func (*UnicodeRangeToken) Type() string        { return "unicode-range" }
func (n *UnicodeRangeToken) Serialize() string { return serializeNode(n) }
func (n *UnicodeRangeToken) serializeTo(w *strings.Builder) {
	if n.End == n.Start {
		w.WriteString("U+" + upperHex(n.Start))
	} else {
		w.WriteString("U+" + upperHex(n.Start) + "-" + upperHex(n.End))
	}
}

// Numeric holds the value of a number, percentage or dimension token.
type Numeric struct {
	// Value is the numeric value.
	Value float64
	// IntValue is the value when IsInteger is true (saturated at the int
	// range).
	IntValue int
	// IsInteger is true when the source was syntactically an integer.
	IsInteger bool
	// Representation is the number as written, without any unit.
	Representation string
}

// NumberToken is a number.
type NumberToken struct {
	Position
	Numeric
}

// NewNumberToken returns a NumberToken node.
func NewNumberToken(line, column int, n Numeric) *NumberToken {
	return &NumberToken{Position{line, column}, n}
}

func (*NumberToken) Type() string        { return "number" }
func (n *NumberToken) Serialize() string { return serializeNode(n) }
func (n *NumberToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Representation)
}

// PercentageToken is a number followed by %.
type PercentageToken struct {
	Position
	Numeric
}

// NewPercentageToken returns a PercentageToken node.
func NewPercentageToken(line, column int, n Numeric) *PercentageToken {
	return &PercentageToken{Position{line, column}, n}
}

func (*PercentageToken) Type() string        { return "percentage" }
func (n *PercentageToken) Serialize() string { return serializeNode(n) }
func (n *PercentageToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Representation)
	w.WriteByte('%')
}

// DimensionToken is a number followed by a unit.
type DimensionToken struct {
	Position
	Numeric
	// Unit is the unescaped unit; LowerUnit is Unit in ASCII lower case.
	Unit      string
	LowerUnit string
}

// NewDimensionToken returns a DimensionToken node.
func NewDimensionToken(line, column int, n Numeric, unit string) *DimensionToken {
	return &DimensionToken{Position{line, column}, n, unit, AsciiLower(unit)}
}

func (*DimensionToken) Type() string        { return "dimension" }
func (n *DimensionToken) Serialize() string { return serializeNode(n) }
func (n *DimensionToken) serializeTo(w *strings.Builder) {
	w.WriteString(n.Representation)
	// A unit starting like an exponent must be escaped to not merge into
	// the number when reparsed.
	u := n.Unit
	if u == "e" || u == "E" || strings.HasPrefix(u, "e-") || strings.HasPrefix(u, "E-") {
		w.WriteString("\\65 ")
		w.WriteString(SerializeName(u[1:]))
	} else {
		w.WriteString(SerializeIdentifier(u))
	}
}

// ParenthesesBlock is a ( ) block.
type ParenthesesBlock struct {
	Position
	Content []Node
}

// NewParenthesesBlock returns a ParenthesesBlock node.
func NewParenthesesBlock(line, column int, content []Node) *ParenthesesBlock {
	return &ParenthesesBlock{Position{line, column}, content}
}

func (*ParenthesesBlock) Type() string        { return "() block" }
func (n *ParenthesesBlock) Serialize() string { return serializeNode(n) }
func (n *ParenthesesBlock) serializeTo(w *strings.Builder) {
	w.WriteByte('(')
	serializeNodes(n.Content, w)
	w.WriteByte(')')
}

// SquareBracketsBlock is a [ ] block.
type SquareBracketsBlock struct {
	Position
	Content []Node
}

// NewSquareBracketsBlock returns a SquareBracketsBlock node.
func NewSquareBracketsBlock(line, column int, content []Node) *SquareBracketsBlock {
	return &SquareBracketsBlock{Position{line, column}, content}
}

func (*SquareBracketsBlock) Type() string        { return "[] block" }
func (n *SquareBracketsBlock) Serialize() string { return serializeNode(n) }
func (n *SquareBracketsBlock) serializeTo(w *strings.Builder) {
	w.WriteByte('[')
	serializeNodes(n.Content, w)
	w.WriteByte(']')
}

// CurlyBracketsBlock is a { } block.
type CurlyBracketsBlock struct {
	Position
	Content []Node
}

// NewCurlyBracketsBlock returns a CurlyBracketsBlock node.
func NewCurlyBracketsBlock(line, column int, content []Node) *CurlyBracketsBlock {
	return &CurlyBracketsBlock{Position{line, column}, content}
}

func (*CurlyBracketsBlock) Type() string        { return "{} block" }
func (n *CurlyBracketsBlock) Serialize() string { return serializeNode(n) }
func (n *CurlyBracketsBlock) serializeTo(w *strings.Builder) {
	w.WriteByte('{')
	serializeNodes(n.Content, w)
	w.WriteByte('}')
}

// FunctionBlock is a function call: a name followed by a parenthesized
// argument list. Commas are not special and appear as LiteralTokens.
type FunctionBlock struct {
	Position
	// Name is the unescaped name; LowerName is Name in ASCII lower case.
	Name      string
	LowerName string
	Arguments []Node
}

// NewFunctionBlock returns a FunctionBlock node.
func NewFunctionBlock(line, column int, name string, arguments []Node) *FunctionBlock {
	return &FunctionBlock{Position{line, column}, name, AsciiLower(name), arguments}
}

func (*FunctionBlock) Type() string        { return "function" }
func (n *FunctionBlock) Serialize() string { return serializeNode(n) }
func (n *FunctionBlock) serializeTo(w *strings.Builder) {
	w.WriteString(SerializeIdentifier(n.Name))
	w.WriteByte('(')
	serializeNodes(n.Arguments, w)
	// A string cut by EOF swallows the closing parenthesis when reparsed.
	var f Node = n
	for {
		fn, ok := f.(*FunctionBlock)
		if !ok || len(fn.Arguments) == 0 {
			break
		}
		last := fn.Arguments[len(fn.Arguments)-1]
		if e, ok := last.(*ParseError); ok && e.Kind == "eof-in-string" {
			return
		}
		f = last
	}
	w.WriteByte(')')
}

// Declaration is a property or descriptor declaration.
type Declaration struct {
	Position
	// Name is the unescaped name; LowerName is Name in ASCII lower case.
	Name      string
	LowerName string
	// Value holds the component values between the colon and the end of the
	// declaration or its !important flag.
	Value     []Node
	Important bool
}

// NewDeclaration returns a Declaration node.
func NewDeclaration(line, column int, name string, value []Node, important bool) *Declaration {
	return &Declaration{Position{line, column}, name, AsciiLower(name), value, important}
}

func (*Declaration) Type() string        { return "declaration" }
func (n *Declaration) Serialize() string { return serializeNode(n) }
func (n *Declaration) serializeTo(w *strings.Builder) {
	w.WriteString(SerializeIdentifier(n.Name))
	w.WriteByte(':')
	serializeNodes(n.Value, w)
	if n.Important {
		w.WriteString("!important")
	}
}

// QualifiedRule is a prelude followed by a {} block, such as a style rule.
type QualifiedRule struct {
	Position
	Prelude []Node
	Content []Node
}

// NewQualifiedRule returns a QualifiedRule node.
func NewQualifiedRule(line, column int, prelude, content []Node) *QualifiedRule {
	return &QualifiedRule{Position{line, column}, prelude, content}
}

func (*QualifiedRule) Type() string        { return "qualified-rule" }
func (n *QualifiedRule) Serialize() string { return serializeNode(n) }
func (n *QualifiedRule) serializeTo(w *strings.Builder) {
	serializeNodes(n.Prelude, w)
	w.WriteByte('{')
	serializeNodes(n.Content, w)
	w.WriteByte('}')
}

// AtRule is an at-keyword with a prelude, ended by a {} block or a semicolon.
type AtRule struct {
	Position
	// AtKeyword is the unescaped keyword without @; LowerAtKeyword is it in
	// ASCII lower case.
	AtKeyword      string
	LowerAtKeyword string
	Prelude        []Node
	// Content is the block content, or nil for a rule ended by a semicolon.
	// An empty block gives a non-nil empty slice.
	Content []Node
}

// NewAtRule returns an AtRule node.
func NewAtRule(line, column int, atKeyword string, prelude, content []Node) *AtRule {
	return &AtRule{Position{line, column}, atKeyword, AsciiLower(atKeyword), prelude, content}
}

func (*AtRule) Type() string        { return "at-rule" }
func (n *AtRule) Serialize() string { return serializeNode(n) }
func (n *AtRule) serializeTo(w *strings.Builder) {
	w.WriteByte('@')
	w.WriteString(SerializeIdentifier(n.AtKeyword))
	serializeNodes(n.Prelude, w)
	if n.Content == nil {
		w.WriteByte(';')
		return
	}
	w.WriteByte('{')
	serializeNodes(n.Content, w)
	w.WriteByte('}')
}
