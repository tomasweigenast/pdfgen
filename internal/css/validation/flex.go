package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

// Keywords of the alignment properties (CSS Box Alignment 3). They share one
// grammar: a single keyword, baseline with an optional first or last, or an
// overflow position followed by a keyword.
var (
	alignContentKeywords = []string{
		"center", "space-between", "space-around", "space-evenly", "stretch",
		"normal", "flex-start", "flex-end", "start", "end",
	}
	justifyContentKeywords = []string{
		"center", "space-between", "space-around", "space-evenly", "stretch",
		"normal", "flex-start", "flex-end", "start", "end", "left", "right",
	}
	justifyContentSafe = []string{"center", "start", "end", "flex-start", "flex-end", "left", "right"}
	itemsKeywords      = []string{
		"normal", "stretch", "center", "start", "end", "self-start", "self-end",
		"flex-start", "flex-end",
	}
	itemsSafe = []string{"center", "start", "end", "self-start", "self-end", "flex-start", "flex-end"}
	// The justify properties also take the physical keywords after safe.
	justifyItemsSafe = append([]string{"left", "right"}, itemsSafe...)
)

func init() {
	register(singleToken(plain(validateFlexBasis)), 0, "flex-basis")
	register(keywordOf("row", "row-reverse", "column", "column-reverse"), 0, "flex-direction")
	register(keywordOf("nowrap", "wrap", "wrap-reverse"), 0, "flex-wrap")
	register(singleToken(plain(validateFlexGrowShrink)), 0, "flex-grow", "flex-shrink")
	register(singleToken(plain(validateOrder)), 0, "order")

	register(alignment(justifyContentKeywords, justifyContentSafe, false, false), 0, "justify-content")
	register(alignment(append([]string{"left", "right", "legacy"}, itemsKeywords...), justifyItemsSafe, true, true), 0, "justify-items")
	register(alignment(append([]string{"auto", "left", "right"}, itemsKeywords...), justifyItemsSafe, true, false), 0, "justify-self")
	register(alignment(itemsKeywords, itemsSafe, true, false), 0, "align-items")
	register(alignment(append([]string{"auto"}, itemsKeywords...), itemsSafe, true, false), 0, "align-self")
	register(alignment(alignContentKeywords, []string{"center", "start", "end", "flex-start", "flex-end"}, true, false), 0, "align-content")
}

// alignment validates the properties of CSS Box Alignment 3. keywords are
// accepted alone, safe follows safe or unsafe, baseline is accepted when
// baseline is set (as first baseline when alone) and legacy when legacy is.
func alignment(keywords, safe []string, baseline, legacy bool) validator {
	return func(tokens []syntax.Node, _ string) (props.Value, error) {
		switch len(tokens) {
		case 1:
			k := tk.GetKeyword(tokens[0])
			if contains(keywords, k) {
				return props.Keywords{props.Keyword(k)}, nil
			}
			if baseline && k == "baseline" {
				return props.Keywords{"first", "baseline"}, nil
			}
		case 2:
			k := [2]string{tk.GetKeyword(tokens[0]), tk.GetKeyword(tokens[1])}
			switch {
			case k[0] == "safe" || k[0] == "unsafe":
				if contains(safe, k[1]) {
					return props.Keywords{props.Keyword(k[0]), props.Keyword(k[1])}, nil
				}
			case baseline && (k[0] == "baseline" || k[1] == "baseline"):
				if k[0] == "first" || k[0] == "last" || k[1] == "first" || k[1] == "last" {
					return props.Keywords{props.Keyword(k[0]), props.Keyword(k[1])}, nil
				}
			case legacy && (k[0] == "legacy" || k[1] == "legacy"):
				for _, s := range k {
					if s == "left" || s == "right" || s == "center" {
						return props.Keywords{props.Keyword(k[0]), props.Keyword(k[1])}, nil
					}
				}
			}
		}
		return nil, nil
	}
}

func validateFlexBasis(token syntax.Node) props.Value {
	if v := widthHeight(token); v != nil {
		return v
	}
	if tk.GetKeyword(token) == "content" {
		return props.LengthKeyword("content")
	}
	return nil
}

func validateFlexGrowShrink(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, true, false); ok {
		return props.Number(n.Value)
	}
	return nil
}

func validateOrder(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, true, true); ok {
		return props.Integer(int(n.Value))
	}
	return nil
}
