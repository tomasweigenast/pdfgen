package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	register(singleToken(listStyleImage), 0, "list-style-image")
	register(keywordOf("inside", "outside"), 0, "list-style-position")
	register(singleToken(plain(validateListStyleType)), 0, "list-style-type")
	register(counterValidator(1), 0, "counter-increment")
	register(counterValidator(0), 0, "counter-reset", "counter-set")
	register(validateContent, 0, "content")
	register(validateQuotes, 0, "quotes")
	register(commaSeparatedStringSet, flagUnstable, "string-set")
	register(validateBookmarkLabel, flagUnstable, "bookmark-label")
	register(singleToken(plain(validateBookmarkLevel)), flagUnstable, "bookmark-level")
	register(keywordOf("open", "closed"), flagUnstable, "bookmark-state")
	register(keywordOf("block", "inline", "compact"), flagUnstable, "footnote-display")
	register(keywordOf("auto", "line", "block"), flagUnstable, "footnote-policy")
}

func listStyleImage(token syntax.Node, baseURL string) (props.Value, error) {
	if tk.GetKeyword(token) == "none" {
		return props.Image{Type: "none"}, nil
	}
	url, ok, err := tk.GetURL(token, baseURL)
	if err != nil || !ok {
		return nil, err
	}
	if address, ok := url.External(); ok {
		return props.Image{Type: "url", URL: address}, nil
	}
	return nil, nil
}

func validateListStyleType(token syntax.Node) props.Value {
	if t, ok := tk.ParseListStyleType(token); ok {
		return t
	}
	return nil
}

// counterValidator validates counter-increment, counter-reset and
// counter-set. A counter name without an integer gets defaultInteger.
func counterValidator(defaultInteger int) validator {
	return func(tokens []syntax.Node, _ string) (props.Value, error) {
		if tk.GetSingleKeyword(tokens) == "none" {
			return props.Counters{}, nil
		}
		var results props.Counters
		for i := 0; i < len(tokens); {
			ident, ok := tokens[i].(*syntax.IdentToken)
			if !ok {
				return nil, nil
			}
			switch ident.Value {
			case "none", "initial", "inherit":
				return nil, tk.Invalid("Invalid counter name: %s", ident.Value)
			}
			i++
			integer := defaultInteger
			if i < len(tokens) {
				if n, ok := tk.GetNumber(tokens[i], true, true); ok {
					integer = int(n.Value)
					i++
				}
			}
			results.Items = append(results.Items, props.CounterOp{Name: ident.Value, Value: integer})
		}
		return results, nil
	}
}

// validateContent validates content (CSS Content 3 §2). Images given before
// the content list are checked and then dropped, as they are not rendered.
func validateContent(tokens []syntax.Node, baseURL string) (props.Value, error) {
	for len(tokens) >= 2 && syntax.IsLiteral(tokens[1], ",") {
		token := tokens[0]
		tokens = tokens[2:]
		if _, ok, err := tk.GetImage(token, baseURL); err != nil {
			return nil, err
		} else if ok {
			continue
		}
		if _, ok, err := tk.GetURL(token, baseURL); err != nil {
			return nil, err
		} else if !ok {
			return nil, nil
		}
	}
	if len(tokens) == 0 {
		return nil, nil
	}
	if len(tokens) >= 3 && syntax.IsLiteral(tokens[len(tokens)-2], "/") {
		if _, ok := tokens[len(tokens)-1].(*syntax.StringToken); ok {
			// The alternative text for speech is ignored.
			tokens = tokens[:len(tokens)-2]
		}
	}
	switch k := tk.GetSingleKeyword(tokens); k {
	case "normal", "none":
		return props.Content{Keyword: props.Keyword(k)}, nil
	}
	items, ok, err := tk.GetContentList(tokens, baseURL)
	if err != nil || !ok {
		return nil, err
	}
	return props.Content{Items: items}, nil
}

func validateQuotes(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) == 1 {
		switch k := tk.GetKeyword(tokens[0]); k {
		case "auto", "none":
			return props.Quotes{Keyword: props.Keyword(k)}, nil
		}
	}
	if len(tokens) == 0 || len(tokens)%2 != 0 {
		return nil, nil
	}
	var quotes props.Quotes
	for i, token := range tokens {
		s, ok := token.(*syntax.StringToken)
		if !ok {
			return nil, nil
		}
		// Open and close quotes alternate.
		if i%2 == 0 {
			quotes.Open = append(quotes.Open, s.Value)
		} else {
			quotes.Close = append(quotes.Close, s.Value)
		}
	}
	return quotes, nil
}

// commaSeparatedStringSet validates string-set. Spec asks for strings after
// the custom identifier, but any content list is allowed.
func commaSeparatedStringSet(tokens []syntax.Node, baseURL string) (props.Value, error) {
	entries, ok, err := parseCommaList(tokens, baseURL, stringSetItem)
	if err != nil || !ok {
		return nil, err
	}
	if len(entries) == 1 && entries[0].Name == "none" && len(entries[0].Content) == 0 {
		return props.StringSet{}, nil
	}
	return props.StringSet(entries), nil
}

func stringSetItem(tokens []syntax.Node, baseURL string) (props.StringSetEntry, bool, error) {
	if len(tokens) >= 2 {
		name := tk.GetCustomIdent(tokens[0])
		if name == "" {
			return props.StringSetEntry{}, false, nil
		}
		items, ok, err := tk.GetContentList(tokens[1:], baseURL)
		if err != nil || !ok {
			return props.StringSetEntry{}, false, err
		}
		return props.StringSetEntry{Name: name, Content: items}, true, nil
	}
	if len(tokens) > 0 && tk.GetKeyword(tokens[0]) == "none" {
		return props.StringSetEntry{Name: "none"}, true, nil
	}
	return props.StringSetEntry{}, false, nil
}

func validateBookmarkLabel(tokens []syntax.Node, baseURL string) (props.Value, error) {
	items, ok, err := tk.GetContentList(tokens, baseURL)
	if err != nil || !ok {
		return nil, err
	}
	return props.ContentList(items), nil
}

func validateBookmarkLevel(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, false, true); ok {
		if n.Value >= 1 {
			return props.IntOrKeyword{N: int(n.Value)}
		}
		return nil
	}
	if tk.GetKeyword(token) == "none" {
		return props.IntOrKeyword{Keyword: "none"}
	}
	return nil
}
