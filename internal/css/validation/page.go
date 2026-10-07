package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	register(validateSize, flagUnstable, "size")
	register(singleToken(plain(validatePage)), flagUnstable, "page")
	register(singleToken(plain(validateBleed)), flagUnstable, "bleed-left", "bleed-right", "bleed-top", "bleed-bottom")
	register(validateMarks, flagUnstable, "marks")
	register(keywordOf("auto", "keep", "discard"), flagUnstable, "margin-break")
	register(keywordOf("auto", "avoid", "avoid-page", "page", "left", "right", "recto", "verso", "avoid-column", "column", "always"),
		0, "break-before", "break-after")
	register(keywordOf("auto", "avoid", "avoid-page", "avoid-column"), 0, "break-inside")
	register(keywordOf("slice", "clone"), 0, "box-decoration-break")
	register(singleToken(plain(validateOrphansWidows)), 0, "orphans", "widows")
	register(singleToken(plain(validateColumnWidth)), flagUnstable, "column-width")
	register(keywordOf("all", "none"), flagUnstable, "column-span")
	register(singleToken(plain(validateColumnCount)), flagUnstable, "column-count")
	register(keywordOf("auto", "balance"), flagUnstable, "column-fill")
	register(singleToken(plain(validateGap)), flagUnstable, "column-gap", "row-gap")
	register(validateColorScheme, 0, "color-scheme")
}

func pageSizePair(size props.PageSize, landscape bool) props.LengthPair {
	w := props.Length{Dimension: props.Dimension{Value: size.Width, Unit: size.Unit}}
	h := props.Length{Dimension: props.Dimension{Value: size.Height, Unit: size.Unit}}
	if landscape {
		w, h = h, w
	}
	return props.LengthPair{X: w, Y: h}
}

// validateSize validates size (CSS Page 3 §7.1).
func validateSize(tokens []syntax.Node, _ string) (props.Value, error) {
	if lengths, ok := lengthList(tokens, false, false); ok {
		switch len(lengths) {
		case 1:
			return props.LengthPair{X: lengths[0], Y: lengths[0]}, nil
		case 2:
			return props.LengthPair{X: lengths[0], Y: lengths[1]}, nil
		}
	}

	keywords := make([]props.Keyword, len(tokens))
	for i, token := range tokens {
		keywords[i] = props.Keyword(tk.GetKeyword(token))
	}
	switch len(keywords) {
	case 1:
		keyword := keywords[0]
		if size, ok := props.PageSizes[keyword]; ok {
			return pageSizePair(size, false), nil
		}
		switch keyword {
		case "auto", "portrait":
			return pageSizePair(props.InitialPageSize, false), nil
		case "landscape":
			return pageSizePair(props.InitialPageSize, true), nil
		}
	case 2:
		var orientation, pageSize props.Keyword
		isOrientation := func(k props.Keyword) bool { return k == "portrait" || k == "landscape" }
		switch {
		case isOrientation(keywords[0]):
			orientation, pageSize = keywords[0], keywords[1]
		case isOrientation(keywords[1]):
			pageSize, orientation = keywords[0], keywords[1]
		}
		if size, ok := props.PageSizes[pageSize]; ok {
			return pageSizePair(size, orientation != "portrait"), nil
		}
	}
	return nil, nil
}

func validatePage(token syntax.Node) props.Value {
	if ident, ok := token.(*syntax.IdentToken); ok {
		if ident.LowerValue == "auto" {
			return props.Keyword("auto")
		}
		return props.Keyword(ident.Value)
	}
	return nil
}

func validateBleed(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "auto" {
		return auto
	}
	if l, ok := tk.GetLength(token, true, false); ok {
		return l
	}
	return nil
}

func validateMarks(tokens []syntax.Node, _ string) (props.Value, error) {
	switch len(tokens) {
	case 2:
		k0, k1 := tk.GetKeyword(tokens[0]), tk.GetKeyword(tokens[1])
		if (k0 == "crop" && k1 == "cross") || (k0 == "cross" && k1 == "crop") {
			return props.Keywords{props.Keyword(k0), props.Keyword(k1)}, nil
		}
	case 1:
		switch k := tk.GetKeyword(tokens[0]); k {
		case "crop", "cross":
			return props.Keywords{props.Keyword(k)}, nil
		case "none":
			return props.Keywords{}, nil
		}
	}
	return nil, nil
}

func validateOrphansWidows(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, false, true); ok && n.Value >= 1 {
		return props.Integer(int(n.Value))
	}
	return nil
}

func validateColumnWidth(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, false); ok {
		return l
	}
	if tk.GetKeyword(token) == "auto" {
		return auto
	}
	return nil
}

func validateColumnCount(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, false, true); ok {
		if n.Value >= 1 {
			return props.IntOrKeyword{N: int(n.Value)}
		}
		return nil
	}
	if tk.GetKeyword(token) == "auto" {
		return props.IntOrKeyword{Keyword: "auto"}
	}
	return nil
}

func validateGap(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	if tk.GetKeyword(token) == "normal" {
		return props.LengthKeyword("normal")
	}
	return nil
}

// validateColorScheme validates color-scheme (CSS Color Adjust 1 §3): normal,
// or a list of scheme names with an optional only at either end.
func validateColorScheme(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) == 1 {
		switch k := tk.GetSingleKeyword(tokens); k {
		case "":
			return nil, nil
		case "normal":
			return props.Keywords{"normal"}, nil
		case "only":
			return nil, nil
		default:
			return props.Keywords{props.Keyword(k)}, nil
		}
	}
	var keywords props.Keywords
	only := false
	for i, token := range tokens {
		switch k := tk.GetKeyword(token); k {
		case "only":
			if only || (i != 0 && i != len(tokens)-1) {
				return nil, nil
			}
			only = true
		case "normal", "":
			return nil, nil
		default:
			keywords = append(keywords, props.Keyword(k))
		}
	}
	if only {
		keywords = append(keywords, "only")
	}
	return keywords, nil
}
