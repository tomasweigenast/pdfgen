package media

import (
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// Query is a parsed media query list. Only media types are supported: a
// query is the list of lowercase media type names, "all" matching any
// device.
type Query []string

// Evaluate reports whether the query matches the media type of the device.
// Media features are not supported, so only the types are compared.
func Evaluate(q Query, deviceMediaType string) bool {
	for _, t := range q {
		if t == "all" || t == deviceMediaType {
			return true
		}
	}
	return false
}

// Parse parses the comma-separated media types of a media query list
// (Media Queries 4 §3). An empty list is "all" and a leading only is
// ignored. Anything other than a single identifier per query is reported
// with a warning, and the boolean is then false.
func Parse(tokens []syntax.Node) (Query, bool) {
	tokens = tk.RemoveWhitespace(tokens)
	if len(tokens) == 0 {
		return Query{"all"}, true
	}
	if ident, ok := tokens[0].(*syntax.IdentToken); ok && ident.LowerValue == "only" {
		tokens = tokens[1:]
	}
	q := Query{}
	for _, part := range tk.SplitOnComma(tokens) {
		if len(part) == 1 {
			if ident, ok := part[0].(*syntax.IdentToken); ok {
				q = append(q, ident.LowerValue)
				continue
			}
		}
		logging.Warnf("Expected a media type, got %s", logging.Repr(syntax.Serialize(part)))
		return nil, false
	}
	return q, true
}
