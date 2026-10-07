package tokens

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// URL is a <url>: a link resolved against the base URL, or an attr()
// function that yields one at used-value time.
type URL struct {
	Link urls.Link
	// Attr is set instead of Link for attr(name url).
	Attr *props.ContentItem
}

// GetURL parses a <url> token. A url that cannot be resolved because it is
// relative and there is no base URL is an error.
func GetURL(token syntax.Node, baseURL string) (URL, bool, error) {
	var link urls.Link
	var ok bool
	switch t := token.(type) {
	case *syntax.URLToken:
		link, ok = urls.GetURLTuple(t.Value, baseURL)
	case *syntax.FunctionBlock:
		switch {
		case t.Name == "attr":
			item, found := CheckAttr(token, "url")
			if !found {
				return URL{}, false, nil
			}
			return URL{Attr: &item}, true, nil
		case t.Name == "url" && (len(t.Arguments) == 1 || len(t.Arguments) == 2):
			// Url modifiers are ignored (CSS Values 3 §4.5).
			link, ok = urls.GetURLTuple(valueOfNode(t.Arguments[0]), baseURL)
		default:
			return URL{}, false, nil
		}
	default:
		return URL{}, false, nil
	}
	if !ok {
		return URL{}, false, Invalid("Relative URI reference without a base URI: None")
	}
	return URL{Link: link}, true, nil
}

// valueOfNode returns the text value of a token, or the empty string for
// tokens that do not have one.
func valueOfNode(n syntax.Node) string {
	switch t := n.(type) {
	case *syntax.StringToken:
		return t.Value
	case *syntax.IdentToken:
		return t.Value
	case *syntax.WhitespaceToken:
		return t.Value
	case *syntax.URLToken:
		return t.Value
	case *syntax.LiteralToken:
		return t.Value
	}
	return ""
}

// Item returns the content item of a url: a "url" item with the kind and
// address of the link, or the attr() item.
func (u URL) Item() props.ContentItem {
	if u.Attr != nil {
		return *u.Attr
	}
	kind := "external"
	if u.Link.Kind == urls.LinkInternal {
		kind = "internal"
	}
	return props.ContentItem{Type: "url", Args: []string{kind, u.Link.Target}}
}

// External returns the address of an external link, such as the one used
// where only those are allowed.
func (u URL) External() (string, bool) {
	if u.Attr != nil || u.Link.Kind != urls.LinkExternal {
		return "", false
	}
	return u.Link.Target, true
}

// GetString parses a <string>: a string token, or one of the functions that
// produce a string, such as attr(), counter() and string().
func GetString(token syntax.Node) (props.ContentItem, bool) {
	switch t := token.(type) {
	case *syntax.StringToken:
		return props.ContentItem{Type: "string", Args: []string{t.Value}}, true
	case *syntax.FunctionBlock:
		switch t.Name {
		case "attr":
			return CheckAttr(token, "string")
		case "counter", "counters":
			return CheckCounter(token)
		case "content":
			return CheckContent(token)
		case "string":
			return CheckStringOrElement("string", token)
		}
	}
	return props.ContentItem{}, false
}

// GetTarget parses target-counter(), target-counters() and target-text()
// (CSS Generated Content for Paged Media 3 §4.1).
func GetTarget(token syntax.Node, baseURL string) (props.ContentItem, bool, error) {
	f := NewFunction(token)
	arguments := f.SplitComma()
	switch f.Name {
	case "target-counter":
		if len(arguments) != 2 && len(arguments) != 3 {
			return props.ContentItem{}, false, nil
		}
	case "target-counters":
		if len(arguments) != 3 && len(arguments) != 4 {
			return props.ContentItem{}, false, nil
		}
	case "target-text":
		if len(arguments) != 1 && len(arguments) != 2 {
			return props.ContentItem{}, false, nil
		}
	default:
		return props.ContentItem{}, false, nil
	}

	item := props.ContentItem{Type: f.Name + "()"}
	link := arguments[0]
	arguments = arguments[1:]
	if s, ok := GetString(link); ok {
		item.Link = &s
	} else {
		url, ok, err := GetURL(link, baseURL)
		if err != nil || !ok {
			return props.ContentItem{}, false, err
		}
		u := url.Item()
		item.Link = &u
	}

	if f.Name == "target-text" {
		content := "content"
		if len(arguments) > 0 {
			content = GetKeyword(arguments[0])
			if !isOneOf(content, "content", "before", "after", "first-letter") {
				return props.ContentItem{}, false, nil
			}
		}
		item.Args = []string{content}
		return item, true, nil
	}

	ident, ok := arguments[0].(*syntax.IdentToken)
	if !ok {
		return props.ContentItem{}, false, nil
	}
	arguments = arguments[1:]
	item.Args = []string{ident.Value}
	if f.Name == "target-counters" {
		sep, ok := GetString(arguments[0])
		if !ok {
			return props.ContentItem{}, false, nil
		}
		item.Separator = &sep
		arguments = arguments[1:]
	}
	style := "decimal"
	if len(arguments) > 0 {
		style = GetKeyword(arguments[0])
	}
	item.Args = append(item.Args, style)
	return item, true, nil
}

// GetContentList parses the tokens of a <content-list> (CSS Content 3
// §2). It reports false if any token is not valid.
func GetContentList(tokens []syntax.Node, baseURL string) ([]props.ContentItem, bool, error) {
	items := make([]props.ContentItem, 0, len(tokens))
	for _, token := range tokens {
		item, ok, err := GetContentListToken(token, baseURL)
		if err != nil || !ok {
			return nil, false, err
		}
		items = append(items, item)
	}
	return items, true, nil
}

// GetContentListToken parses one item of a <content-list>.
func GetContentListToken(token syntax.Node, baseURL string) (props.ContentItem, bool, error) {
	if s, ok := GetString(token); ok {
		return s, true, nil
	}
	if GetKeyword(token) == "contents" {
		return props.ContentItem{Type: "content()", Args: []string{"text"}}, true, nil
	}
	url, ok, err := GetURL(token, baseURL)
	if err != nil {
		return props.ContentItem{}, false, err
	}
	if ok {
		return url.Item(), true, nil
	}
	if quote, ok := GetQuote(token); ok {
		return props.ContentItem{Type: "quote", Args: []string{quote}}, true, nil
	}
	target, ok, err := GetTarget(token, baseURL)
	if err != nil || ok {
		return target, ok, err
	}

	f := NewFunction(token)
	arguments := f.SplitComma()
	switch f.Name {
	case "leader":
		if len(arguments) != 1 {
			return props.ContentItem{}, false, nil
		}
		var s string
		switch arg := arguments[0].(type) {
		case *syntax.IdentToken:
			switch arg.Value {
			case "dotted":
				s = "."
			case "solid":
				s = "_"
			case "space":
				s = " "
			default:
				return props.ContentItem{}, false, nil
			}
		case *syntax.StringToken:
			s = arg.Value
		default:
			return props.ContentItem{}, false, nil
		}
		return props.ContentItem{Type: "leader()", Args: []string{s}}, true, nil
	case "element":
		item, ok := CheckStringOrElement("element", token)
		return item, ok, nil
	}
	return props.ContentItem{}, false, nil
}

// ParseListStyleType parses a list-style-type: a counter style name, a
// string or symbols().
func ParseListStyleType(token syntax.Node) (props.ListStyleType, bool) {
	switch t := token.(type) {
	case *syntax.IdentToken:
		return props.ListStyleType{Kind: props.ListStyleKeyword, Name: t.Value}, true
	case *syntax.StringToken:
		return props.ListStyleType{Kind: props.ListStyleString, Name: t.Value}, true
	case *syntax.FunctionBlock:
		if t.Name != "symbols" {
			return props.ListStyleType{}, false
		}
		arguments := RemoveWhitespace(t.Arguments)
		if len(arguments) == 0 {
			return props.ListStyleType{}, false
		}
		kind := "symbolic"
		index := 0
		if ident, ok := arguments[0].(*syntax.IdentToken); ok {
			switch ident.Value {
			case "cyclic", "numeric", "alphabetic", "symbolic", "fixed":
				kind = ident.Value
				index = 1
			default:
				return props.ListStyleType{}, false
			}
		}
		if len(arguments) < index+1 {
			return props.ListStyleType{}, false
		}
		var symbols []string
		for _, a := range arguments[index:] {
			s, ok := a.(*syntax.StringToken)
			if !ok {
				return props.ListStyleType{}, false
			}
			symbols = append(symbols, s.Value)
		}
		if (kind == "alphabetic" || kind == "numeric") && len(symbols) < 2 {
			return props.ListStyleType{}, false
		}
		return props.ListStyleType{Kind: props.ListStyleSymbols, Name: kind, Symbols: symbols}, true
	}
	return props.ListStyleType{}, false
}
