package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	registerExpander(generic([]string{"-line", "-color", "-style", "-thickness"}, textDecorationParts), "text-decoration")
	registerExpander(generic([]string{"break-after"}, pageBreakParts), "page-break-after")
	registerExpander(generic([]string{"break-before"}, pageBreakParts), "page-break-before")
	registerExpander(generic([]string{"break-inside"}, pageBreakInsideParts), "page-break-inside")
	registerExpander(generic([]string{"column-width", "column-count"}, columnsParts), "columns")
	registerExpander(generic([]string{"-alternates", "-caps", "-east-asian", "-ligatures", "-numeric", "-position"},
		func(tokens []syntax.Node, _, _ string) ([]part, error) { return expandFontVariant(tokens) }), "font-variant")
	registerExpander(generic([]string{"-style", "-variant-caps", "-weight", "-stretch", "-size", "line-height", "-family"}, fontParts), "font")
	registerExpander(generic([]string{"overflow-wrap"}, wordWrapParts), "word-wrap")
	registerExpander(generic([]string{"max-lines", "continue", "block-ellipsis"}, lineClampParts), "line-clamp")
	registerExpander(generic([]string{"-all", "-last"}, textAlignParts), "text-align")
}

func ident(like syntax.Node, name string) syntax.Node {
	line, column := like.Pos()
	return syntax.NewIdentToken(line, column, name)
}

func textDecorationParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	var line, color, style, thickness []syntax.Node
	noneInLine := false
	for _, token := range tokens {
		keyword := tk.GetKeyword(token)
		switch {
		case contains([]string{"none", "underline", "overline", "line-through", "blink"}, keyword):
			line = append(line, token)
			if noneInLine {
				return nil, tk.ErrInvalid
			} else if keyword == "none" {
				noneInLine = true
			}
		case contains([]string{"solid", "double", "dotted", "dashed", "wavy"}, keyword):
			if style != nil {
				return nil, tk.ErrInvalid
			}
			style = append(style, token)
		default:
			if _, ok := syntax.ParseColor(token); ok {
				if color != nil {
					return nil, tk.ErrInvalid
				}
				color = append(color, token)
			} else if ok, err := accepts("text-decoration-thickness", baseURL, token); err != nil {
				return nil, err
			} else if ok {
				if thickness != nil {
					return nil, tk.ErrInvalid
				}
				thickness = append(thickness, token)
			} else {
				return nil, tk.ErrInvalid
			}
		}
	}
	var parts []part
	for _, p := range []part{{"-line", line}, {"-color", color}, {"-style", style}, {"-thickness", thickness}} {
		if p.tokens != nil {
			parts = append(parts, p)
		}
	}
	return parts, nil
}

// pageBreakParts expands the legacy page-break-before and -after properties
// (CSS Fragmentation 3 §3.4).
func pageBreakParts(tokens []syntax.Node, name, _ string) ([]part, error) {
	newName := name[len("page-"):]
	switch tk.GetSingleKeyword(tokens) {
	case "auto", "left", "right", "avoid":
		return []part{{newName, tokens}}, nil
	case "always":
		return []part{{newName, []syntax.Node{ident(tokens[0], "page")}}}, nil
	}
	return nil, tk.ErrInvalid
}

func pageBreakInsideParts(tokens []syntax.Node, _, _ string) ([]part, error) {
	switch tk.GetSingleKeyword(tokens) {
	case "auto", "avoid":
		return []part{{"break-inside", tokens}}, nil
	}
	return nil, tk.ErrInvalid
}

func columnsParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	var parts []part
	if len(tokens) == 2 && tk.GetKeyword(tokens[0]) == "auto" {
		tokens = []syntax.Node{tokens[1], tokens[0]}
	}
	name := ""
	for _, token := range tokens {
		okWidth, err := accepts("column-width", baseURL, token)
		if err != nil {
			return parts, err
		}
		if okWidth && name != "column-width" {
			name = "column-width"
		} else if ok, err := accepts("column-count", baseURL, token); err != nil {
			return parts, err
		} else if ok {
			name = "column-count"
		} else {
			return parts, tk.ErrInvalid
		}
		parts = append(parts, part{name, []syntax.Node{token}})
	}
	if len(tokens) == 1 {
		if name == "column-count" {
			name = "column-width"
		} else {
			name = "column-count"
		}
		parts = append(parts, part{name, []syntax.Node{ident(tokens[0], "auto")}})
	}
	return parts, nil
}

// expandFontVariant splits the values of font-variant among the longhands
// (CSS Fonts 3 §6.12).
func expandFontVariant(tokens []syntax.Node) ([]part, error) {
	if keyword := tk.GetSingleKeyword(tokens); keyword == "normal" || keyword == "none" {
		normal := []syntax.Node{syntax.NewIdentToken(0, 0, "normal")}
		parts := []part{{"-alternates", normal}, {"-caps", normal}, {"-east-asian", normal}, {"-numeric", normal}, {"-position", normal}}
		ligatures := normal
		if keyword == "none" {
			ligatures = []syntax.Node{syntax.NewIdentToken(0, 0, "none")}
		}
		return append(parts, part{"-ligatures", ligatures}), nil
	}

	features := []string{"alternates", "caps", "east-asian", "ligatures", "numeric", "position"}
	found := map[string][]syntax.Node{}
	for _, token := range tokens {
		if tk.GetKeyword(token) == "normal" {
			// Only the specific values are allowed here.
			return nil, tk.ErrInvalid
		}
		matched := false
		for _, feature := range features {
			ok, err := accepts("font-variant-"+feature, "", token)
			if err != nil {
				return nil, err
			}
			if ok {
				found[feature] = append(found[feature], token)
				matched = true
				break
			}
		}
		if !matched {
			return nil, tk.ErrInvalid
		}
	}
	var parts []part
	for _, feature := range features {
		if t := found[feature]; len(t) > 0 {
			parts = append(parts, part{"-" + feature, t})
		}
	}
	return parts, nil
}

func fontParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	switch tk.GetSingleKeyword(tokens) {
	case "caption", "icon", "menu", "message-box", "small-caption", "status-bar":
		return nil, tk.Invalid("System fonts are not supported")
	}

	var parts []part
	// The stack has the next token on top.
	stack := reverse(tokens)
	pop := func() syntax.Node {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return t
	}

	// font-style, font-variant-caps, font-weight and font-stretch come in any
	// order and are all optional.
	var token syntax.Node
	finished := true
	for i := 0; i < 4; i++ {
		if len(stack) == 0 {
			return parts, tk.ErrInvalid
		}
		token = pop()
		if tk.GetKeyword(token) == "normal" {
			// The longhands that are not set are normal.
			continue
		}
		suffix := ""
		for _, c := range []struct{ suffix, name string }{
			{"-style", "font-style"}, {"-variant-caps", "font-variant-caps"},
			{"-weight", "font-weight"}, {"-stretch", "font-stretch"},
		} {
			ok, err := accepts(c.name, baseURL, token)
			if err != nil {
				return parts, err
			}
			if ok {
				suffix = c.suffix
				break
			}
		}
		if suffix == "" {
			// Done with these four, font-size follows.
			finished = false
			break
		}
		parts = append(parts, part{suffix, []syntax.Node{token}})
		if len(stack) == 0 {
			return parts, tk.ErrInvalid
		}
	}
	if finished {
		if len(stack) == 0 {
			return parts, tk.ErrInvalid
		}
		token = pop()
	}

	// font-size is mandatory.
	if ok, err := accepts("font-size", baseURL, token); err != nil || !ok {
		return parts, firstNonNil(err, tk.ErrInvalid)
	}
	parts = append(parts, part{"-size", []syntax.Node{token}})

	// line-height is optional, but font-family is not.
	if len(stack) == 0 {
		return parts, tk.ErrInvalid
	}
	token = pop()
	if syntax.IsLiteral(token, "/") {
		if len(stack) == 0 {
			return parts, tk.ErrInvalid
		}
		token = pop()
		if ok, err := accepts("line-height", baseURL, token); err != nil || !ok {
			return parts, firstNonNil(err, tk.ErrInvalid)
		}
		parts = append(parts, part{"line-height", []syntax.Node{token}})
	} else {
		stack = append(stack, token)
	}

	family := reverse(stack)
	if ok, err := accepts("font-family", baseURL, family...); err != nil || !ok {
		return parts, firstNonNil(err, tk.ErrInvalid)
	}
	return append(parts, part{"-family", family}), nil
}

func firstNonNil(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func wordWrapParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	if ok, err := accepts("overflow-wrap", baseURL, tokens...); err != nil || !ok {
		return nil, firstNonNil(err, tk.ErrInvalid)
	}
	return []part{{"overflow-wrap", tokens}}, nil
}

func numberToken(like syntax.Node, v int) syntax.Node {
	line, column := like.Pos()
	return syntax.NewNumberToken(line, column, syntax.Numeric{Value: float64(v), IntValue: v, IsInteger: true, Representation: itoa(v)})
}

func lineClampParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	switch len(tokens) {
	case 1:
		token := tokens[0]
		if tk.GetSingleKeyword(tokens) == "none" {
			none, auto := ident(token, "none"), ident(token, "auto")
			return []part{{"max-lines", []syntax.Node{none}}, {"continue", []syntax.Node{auto}}, {"block-ellipsis", []syntax.Node{none}}}, nil
		}
		if n, ok := token.(*syntax.NumberToken); ok && n.IsInteger {
			return []part{
				{"max-lines", tokens},
				{"continue", []syntax.Node{ident(token, "discard")}},
				{"block-ellipsis", []syntax.Node{ident(token, "auto")}},
			}, nil
		}
	case 2:
		if n, ok := tokens[0].(*syntax.NumberToken); ok {
			okEllipsis, err := accepts("block-ellipsis", baseURL, tokens[1])
			if err != nil {
				return nil, err
			}
			if n.IsInteger && n.IntValue != 0 && okEllipsis {
				return []part{
					{"max-lines", tokens[:1]},
					{"continue", []syntax.Node{ident(tokens[0], "discard")}},
					{"block-ellipsis", tokens[1:]},
				}, nil
			}
		}
	}
	return nil, tk.ErrInvalid
}

func textAlignParts(tokens []syntax.Node, _, _ string) ([]part, error) {
	if len(tokens) != 1 {
		return nil, tk.ErrInvalid
	}
	keyword := tk.GetSingleKeyword(tokens)
	if keyword == "" {
		return nil, tk.ErrInvalid
	}
	all := tokens
	if keyword == "justify-all" {
		all = []syntax.Node{ident(tokens[0], "justify")}
	}
	last := all
	if keyword == "justify" {
		last = []syntax.Node{ident(tokens[0], "start")}
	}
	return []part{{"-all", all}, {"-last", last}}, nil
}
