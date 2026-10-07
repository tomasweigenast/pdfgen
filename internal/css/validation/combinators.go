package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

// keywordOf validates a single keyword out of a closed set.
func keywordOf(allowed ...string) validator {
	return func(tokens []syntax.Node, _ string) (props.Value, error) {
		if k := tk.GetSingleKeyword(tokens); k != "" && contains(allowed, k) {
			return props.Keyword(k), nil
		}
		return nil, nil
	}
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// singleToken wraps a validator of one token.
func singleToken(f func(token syntax.Node, baseURL string) (props.Value, error)) validator {
	return func(tokens []syntax.Node, baseURL string) (props.Value, error) {
		if len(tokens) == 1 {
			return f(tokens[0], baseURL)
		}
		return nil, nil
	}
}

// plain adapts a function that cannot fail and does not need the base URL.
func plain(f func(token syntax.Node) props.Value) func(syntax.Node, string) (props.Value, error) {
	return func(token syntax.Node, _ string) (props.Value, error) { return f(token), nil }
}

// commaSeparated wraps the validator of an item into the validator of a
// comma-separated list of them. L is the list type, such as props.Images.
func commaSeparated[L interface {
	~[]E
	props.Value
}, E any](f item[E]) validator {
	return func(tokens []syntax.Node, baseURL string) (props.Value, error) {
		list, ok, err := parseCommaList[E](tokens, baseURL, f)
		if err != nil || !ok {
			return nil, err
		}
		return L(list), nil
	}
}

func parseCommaList[E any](tokens []syntax.Node, baseURL string, f item[E]) ([]E, bool, error) {
	var list []E
	for _, part := range tk.SplitOnComma(tokens) {
		e, ok, err := f(tk.RemoveWhitespace(part), baseURL)
		if err != nil || !ok {
			return nil, false, err
		}
		list = append(list, e)
	}
	return list, true, nil
}

// keywordItem is an item that accepts one keyword of a set.
func keywordItem(allowed ...string) item[props.Keyword] {
	return func(tokens []syntax.Node, _ string) (props.Keyword, bool, error) {
		if k := tk.GetSingleKeyword(tokens); k != "" && contains(allowed, k) {
			return props.Keyword(k), true, nil
		}
		return "", false, nil
	}
}

// valid reports whether the validator accepts the tokens, passing errors on.
func valid(f validator, baseURL string, tokens ...syntax.Node) (bool, error) {
	v, err := f(tokens, baseURL)
	return v != nil && err == nil, err
}

// firstError returns the first non-nil error.
func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
