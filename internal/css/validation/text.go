package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

// hyphenCharacter is U+2010, the character chosen for hyphenate-character:
// auto.
const hyphenCharacter = "‐"

func init() {
	register(singleToken(plain(validateSpacing)), 0, "letter-spacing", "word-spacing")
	register(keywordOf("left", "right", "center", "justify", "start", "end"), 0, "text-align-all")
	register(keywordOf("auto", "left", "right", "center", "justify", "start", "end"), 0, "text-align-last")
	register(validateTextDecorationLine, 0, "text-decoration-line")
	register(keywordOf("solid", "double", "dotted", "dashed", "wavy"), 0, "text-decoration-style")
	register(singleToken(plain(validateTextDecorationThickness)), 0, "text-decoration-thickness")
	register(singleToken(plain(lengthOrNone(true))), 0, "text-indent")
	register(keywordOf("none", "uppercase", "lowercase", "capitalize", "full-width"), 0, "text-transform")
	register(keywordOf("normal", "pre", "nowrap", "pre-wrap", "pre-line"), 0, "white-space")
	register(keywordOf("anywhere", "normal", "break-word"), 0, "overflow-wrap")
	register(keywordOf("normal", "break-all"), 0, "word-break")
	register(singleToken(plain(validateTabSize)), 0, "tab-size")
	register(singleToken(plain(validateLineHeight)), 0, "line-height")
	register(singleToken(plain(validateHyphens)), flagUnstable, "hyphens")
	register(singleToken(plain(validateHyphenateCharacter)), flagUnstable, "hyphenate-character")
	register(singleToken(plain(validateHyphenateLimitZone)), flagUnstable, "hyphenate-limit-zone")
	register(validateHyphenateLimitChars, flagUnstable, "hyphenate-limit-chars")
	register(singleToken(plain(validateBlockEllipsis)), 0, "block-ellipsis")
	register(keywordOf("auto", "discard"), flagUnstable, "continue")
	register(singleToken(plain(validateMaxLines)), flagUnstable, "max-lines")
	register(keywordOf("auto", "crisp-edges", "pixelated"), flagUnstable, "image-rendering")
	register(keywordOf("fill", "contain", "cover", "none", "scale-down"), 0, "object-fit")
	register(singleToken(plain(validateImageResolution)), flagUnstable, "image-resolution")
	register(validateImageOrientation, flagUnstable, "image-orientation")
	register(singleToken(plain(validateAnchor)), flagProprietary, "anchor")
	register(singleToken(plain(validateLang)), flagProprietary, "lang")
}

func validateSpacing(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "normal" {
		return props.LengthKeyword("normal")
	}
	if l, ok := tk.GetLength(token, true, false); ok {
		return l
	}
	return nil
}

func validateTextDecorationLine(tokens []syntax.Node, _ string) (props.Value, error) {
	var keywords props.Keywords
	seen := map[string]bool{}
	for _, token := range tokens {
		k := tk.GetKeyword(token)
		if !seen[k] {
			keywords = append(keywords, props.Keyword(k))
		}
		seen[k] = true
	}
	if len(seen) == 1 && seen["none"] {
		return props.Keywords{"none"}, nil
	}
	if len(tokens) != len(seen) {
		return nil, nil
	}
	for k := range seen {
		if !contains([]string{"underline", "overline", "line-through", "blink"}, k) {
			return nil, nil
		}
	}
	return keywords, nil
}

func validateTextDecorationThickness(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, true, true); ok {
		return l
	}
	switch k := tk.GetKeyword(token); k {
	case "auto", "from-font":
		return props.LengthKeyword(props.Keyword(k))
	}
	return nil
}

// validateTabSize validates tab-size: a number of characters or a length
// (CSS Text 3 §7.2).
func validateTabSize(token syntax.Node) props.Value {
	if n, ok := token.(*syntax.NumberToken); ok && n.IsInteger && n.IntValue != 0 {
		return props.LengthNumber(float64(n.IntValue))
	}
	if l, ok := tk.GetLength(token, false, false); ok {
		return l
	}
	return nil
}

func validateLineHeight(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "normal" {
		return props.LengthKeyword("normal")
	}
	if n, ok := tk.GetNumber(token, false, false); ok {
		return props.Length{Dimension: n}
	}
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	return nil
}

func validateHyphens(token syntax.Node) props.Value {
	switch k := tk.GetKeyword(token); k {
	case "none", "manual", "auto":
		return props.Keyword(k)
	}
	return nil
}

func validateHyphenateCharacter(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "auto" {
		return props.String(hyphenCharacter)
	}
	if s, ok := token.(*syntax.StringToken); ok {
		return props.String(s.Value)
	}
	return nil
}

func validateHyphenateLimitZone(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, true); ok {
		return l
	}
	return nil
}

// validateHyphenateLimitChars validates hyphenate-limit-chars, giving the
// minimum word length and the characters before and after a hyphen. Omitted
// values and auto are 5, 2 and 2 (CSS Text 4 §6.3).
func validateHyphenateLimitChars(tokens []syntax.Node, _ string) (props.Value, error) {
	const total, side = 5, 2
	integer := func(token syntax.Node) (int, bool) {
		n, ok := tk.GetNumber(token, true, true)
		return int(n.Value), ok
	}
	isAuto := func(token syntax.Node) bool { return tk.GetKeyword(token) == "auto" }

	switch len(tokens) {
	case 1:
		if isAuto(tokens[0]) {
			return props.IntTriple{total, side, side}, nil
		}
		if n, ok := integer(tokens[0]); ok {
			return props.IntTriple{n, side, side}, nil
		}
	case 2:
		first, second := tokens[0], tokens[1]
		if n, ok := integer(first); ok {
			if left, ok := integer(second); ok {
				return props.IntTriple{n, left, left}, nil
			}
			if isAuto(second) {
				return props.IntTriple{n, side, side}, nil
			}
		} else if isAuto(first) {
			if left, ok := integer(second); ok {
				return props.IntTriple{total, left, left}, nil
			}
			if isAuto(second) {
				return props.IntTriple{total, side, side}, nil
			}
		}
	case 3:
		var result props.IntTriple
		for i, token := range tokens {
			switch n, ok := integer(token); {
			case isAuto(token):
				result[i] = side
				if i == 0 {
					result[i] = total
				}
			case ok:
				result[i] = n
			default:
				return nil, nil
			}
		}
		return result, nil
	}
	return nil, nil
}

func validateBlockEllipsis(token syntax.Node) props.Value {
	if s, ok := token.(*syntax.StringToken); ok {
		return props.KeywordOrString{String: s.Value}
	}
	switch k := tk.GetKeyword(token); k {
	case "none", "auto":
		return props.KeywordOrString{Keyword: props.Keyword(k)}
	}
	return nil
}

func validateMaxLines(token syntax.Node) props.Value {
	if n, ok := tk.GetNumber(token, false, true); ok {
		return props.IntOrKeyword{N: int(n.Value)}
	}
	if tk.GetKeyword(token) == "none" {
		return props.IntOrKeyword{Keyword: "none"}
	}
	return nil
}

func validateImageResolution(token syntax.Node) props.Value {
	if r, ok := tk.GetResolution(token); ok {
		return props.Number(r)
	}
	return nil
}

// validateImageOrientation validates image-orientation: none, from-image, or
// an angle with an optional flip (CSS Images 4 §5.2).
func validateImageOrientation(tokens []syntax.Node, _ string) (props.Value, error) {
	switch k := tk.GetSingleKeyword(tokens); k {
	case "none", "from-image":
		return props.ImageOrientation{Keyword: props.Keyword(k)}, nil
	}
	var angle float64
	haveAngle, flip, haveFlip := false, false, false
	for _, token := range tokens {
		if tk.GetKeyword(token) == "flip" {
			if haveFlip {
				return nil, nil
			}
			flip, haveFlip = true, true
			continue
		}
		if !haveAngle {
			if a, ok := tk.GetAngle(token); ok {
				angle, haveAngle = a, true
				continue
			}
		}
		return nil, nil
	}
	return props.ImageOrientation{Angle: angle, Flip: flip}, nil
}

func validateAnchor(token syntax.Node) props.Value {
	if s, ok := token.(*syntax.StringToken); ok {
		return props.String(s.Value)
	}
	return nil
}

func validateLang(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "none" {
		return props.Keyword("none")
	}
	if s, ok := token.(*syntax.StringToken); ok {
		return props.String(s.Value)
	}
	return nil
}
