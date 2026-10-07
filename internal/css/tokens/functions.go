package tokens

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

// Function is a CSS functional notation (CSS Values 4 §2.5). The zero value
// stands for a token that is not a function.
type Function struct {
	Name      string
	Arguments []syntax.Node
	ok        bool
}

// NewFunction returns the function held by token. When token is not a
// function, IsFunction reports false.
func NewFunction(token syntax.Node) Function {
	if fn, ok := token.(*syntax.FunctionBlock); ok {
		return Function{Name: fn.LowerName, Arguments: fn.Arguments, ok: true}
	}
	return Function{}
}

// IsFunction reports whether f was made from a function token.
func (f Function) IsFunction() bool { return f.ok }

// SplitSpace returns the arguments without whitespace and comments.
func (f Function) SplitSpace() []syntax.Node {
	if !f.ok {
		return nil
	}
	return RemoveWhitespace(f.Arguments)
}

func (f Function) commaParts() [][]syntax.Node {
	parts := [][]syntax.Node{nil}
	for _, token := range f.Arguments {
		if syntax.IsLiteral(token, ",") {
			parts = append(parts, nil)
			continue
		}
		if t := token.Type(); t != "comment" && t != "whitespace" {
			parts[len(parts)-1] = append(parts[len(parts)-1], token)
		}
	}
	return parts
}

// SplitComma splits the arguments on commas when each one is a single token,
// and returns those tokens. It returns nil when the function is malformed:
// an argument has no token or several.
func (f Function) SplitComma() []syntax.Node {
	if !f.ok {
		return nil
	}
	parts := f.commaParts()
	tokens := make([]syntax.Node, len(parts))
	for i, part := range parts {
		if len(part) != 1 {
			return nil
		}
		tokens[i] = part[0]
	}
	return tokens
}

// SplitCommaGroups splits the arguments on commas, keeping the tokens of each
// argument. Every argument must have at least one token, except the last one
// when trailing is set. It returns nil when the function is malformed.
func (f Function) SplitCommaGroups(trailing bool) [][]syntax.Node {
	if !f.ok {
		return nil
	}
	parts := f.commaParts()
	check := parts
	if trailing {
		check = parts[:len(parts)-1]
	}
	for _, part := range check {
		if len(part) == 0 {
			return nil
		}
	}
	return parts
}

// CheckAttr parses an attr() function. When allowedType is not empty the
// type or unit given to attr() has to be that one.
func CheckAttr(token syntax.Node, allowedType string) (props.ContentItem, bool) {
	f := NewFunction(token)
	if f.Name != "attr" {
		return props.ContentItem{}, false
	}
	parts := f.SplitCommaGroups(true)
	var nameAndType []syntax.Node
	fallback := ""
	switch len(parts) {
	case 1:
		nameAndType = parts[0]
	case 2:
		nameAndType = parts[0]
		if rest := parts[1]; len(rest) >= 1 {
			if s, ok := rest[0].(*syntax.StringToken); ok {
				fallback = s.Value
			}
		}
	default:
		return props.ContentItem{}, false
	}
	if len(nameAndType) == 0 {
		return props.ContentItem{}, false
	}
	for _, t := range nameAndType {
		if _, ok := t.(*syntax.IdentToken); !ok {
			return props.ContentItem{}, false
		}
	}
	name := nameAndType[0].(*syntax.IdentToken).Value
	typeOrUnit := "string"
	if len(nameAndType) == 2 {
		typeOrUnit = nameAndType[1].(*syntax.IdentToken).Value
	}
	if allowedType != "" && allowedType != typeOrUnit {
		return props.ContentItem{}, false
	}
	return props.ContentItem{Type: "attr()", Args: []string{name, typeOrUnit, fallback}}, true
}

// CheckCounter parses counter() and counters().
func CheckCounter(token syntax.Node) (props.ContentItem, bool) {
	f := NewFunction(token)
	arguments := f.SplitComma()
	switch f.Name {
	case "counter":
		if len(arguments) != 1 && len(arguments) != 2 {
			return props.ContentItem{}, false
		}
	case "counters":
		if len(arguments) != 2 && len(arguments) != 3 {
			return props.ContentItem{}, false
		}
	default:
		return props.ContentItem{}, false
	}

	ident, ok := arguments[0].(*syntax.IdentToken)
	if !ok {
		return props.ContentItem{}, false
	}
	arguments = arguments[1:]
	item := props.ContentItem{Type: f.Name + "()", Args: []string{ident.Value}}

	if f.Name == "counters" {
		s, ok := arguments[0].(*syntax.StringToken)
		if !ok {
			return props.ContentItem{}, false
		}
		item.Args = append(item.Args, s.Value)
		arguments = arguments[1:]
	}

	style := props.ListStyleType{Name: "decimal"}
	if len(arguments) > 0 {
		s, ok := ParseListStyleType(arguments[0])
		if !ok {
			return props.ContentItem{}, false
		}
		style = s
	}
	item.Style = &style
	return item, true
}

// CheckContent parses content().
func CheckContent(token syntax.Node) (props.ContentItem, bool) {
	f := NewFunction(token)
	if f.Name != "content" {
		return props.ContentItem{}, false
	}
	arguments := f.SplitComma()
	switch len(arguments) {
	case 0:
		return props.ContentItem{Type: "content()", Args: []string{"text"}}, true
	case 1:
		if ident, ok := arguments[0].(*syntax.IdentToken); ok {
			switch ident.LowerValue {
			case "text", "before", "after", "first-letter", "marker":
				return props.ContentItem{Type: "content()", Args: []string{ident.LowerValue}}, true
			}
		}
	}
	return props.ContentItem{}, false
}

// CheckStringOrElement parses string() or element(), according to name.
func CheckStringOrElement(name string, token syntax.Node) (props.ContentItem, bool) {
	f := NewFunction(token)
	arguments := f.SplitComma()
	if f.Name != name || (len(arguments) != 1 && len(arguments) != 2) {
		return props.ContentItem{}, false
	}
	customIdent, ok := arguments[0].(*syntax.IdentToken)
	if !ok {
		return props.ContentItem{}, false
	}
	which := "first"
	if len(arguments) == 2 {
		ident, ok := arguments[1].(*syntax.IdentToken)
		if !ok {
			return props.ContentItem{}, false
		}
		switch ident.LowerValue {
		case "first", "start", "last", "first-except", "all-once":
			which = ident.LowerValue
		default:
			return props.ContentItem{}, false
		}
	}
	return props.ContentItem{Type: name + "()", Args: []string{customIdent.Value, which}}, true
}

// CheckVar reports whether token is, or contains, a var() reference with a
// custom property name.
func CheckVar(token syntax.Node) bool {
	if block, ok := token.(*syntax.ParenthesesBlock); ok {
		for _, item := range block.Content {
			if CheckVar(item) {
				return true
			}
		}
		return false
	}
	f := NewFunction(token)
	if !f.IsFunction() {
		return false
	}
	arguments := f.SplitSpace()
	if f.Name == "var" {
		if len(arguments) == 0 {
			return false
		}
		ident, ok := arguments[0].(*syntax.IdentToken)
		return ok && len(ident.Value) >= 2 && ident.Value[:2] == "--"
	}
	for _, argument := range arguments {
		if CheckVar(argument) {
			return true
		}
	}
	return false
}

// CheckMath reports whether token is a math function with a valid number of
// arguments, or a function that contains one (CSS Values 4 §10).
func CheckMath(token syntax.Node) bool {
	f := NewFunction(token)
	if !f.IsFunction() {
		return false
	}
	n := len(f.SplitCommaGroups(false))
	switch f.Name {
	case "calc", "sin", "cos", "tan", "asin", "acos", "atan", "sqrt", "exp", "abs", "sign":
		return n == 1
	case "min", "max", "hypot":
		return n >= 1
	case "clamp":
		return n == 3
	case "round", "log":
		return 1 <= n && n <= 3 && (f.Name == "round" || n <= 2)
	case "mod", "rem", "atan2", "pow":
		return n == 2
	}
	for _, argument := range f.SplitSpace() {
		if CheckMath(argument) {
			return true
		}
	}
	return false
}
