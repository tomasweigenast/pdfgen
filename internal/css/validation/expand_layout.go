package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	registerExpander(generic([]string{"-grow", "-shrink", "-basis"}, flexParts), "flex")
	registerExpander(generic([]string{"flex-direction", "flex-wrap"}, flexFlowParts), "flex-flow")
	registerExpander(generic([]string{"-columns", "-rows", "-areas"}, gridTemplateParts), "grid-template")
	registerExpander(generic([]string{"-template-columns", "-template-rows", "-template-areas", "-auto-columns", "-auto-rows", "-auto-flow"}, gridParts), "grid")
	registerExpander(generic([]string{"-start", "-end"}, gridColumnRowParts), "grid-column", "grid-row")
	registerExpander(generic([]string{"grid-row-start", "grid-row-end", "grid-column-start", "grid-column-end"}, gridAreaParts), "grid-area")
	registerExpander(generic([]string{"column-gap", "row-gap"}, gapParts), "grid-gap", "gap")
	registerExpander(generic([]string{"column-gap"}, legacyGapParts("column-gap")), "grid-column-gap")
	registerExpander(generic([]string{"row-gap"}, legacyGapParts("row-gap")), "grid-row-gap")
	// The place-* shorthands are not supported yet.
	for _, c := range []struct{ name, a, b string }{
		{"place-content", "align-content", "justify-content"},
		{"place-items", "align-items", "justify-items"},
		{"place-self", "align-self", "justify-self"},
	} {
		registerExpander(generic([]string{c.a, c.b}, func([]syntax.Node, string, string) ([]part, error) {
			return nil, tk.ErrInvalid
		}), c.name)
	}
}

func flexParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	if tk.GetSingleKeyword(tokens) == "none" {
		token := tokens[0]
		zero := numberToken(token, 0)
		return []part{{"-grow", []syntax.Node{zero}}, {"-shrink", []syntax.Node{zero}}, {"-basis", []syntax.Node{ident(token, "auto")}}}, nil
	}

	grow, shrink := 1.0, 1.0
	var basis syntax.Node
	growFound, shrinkFound, basisFound := false, false, false
	for _, token := range tokens {
		// A unitless zero that is not already preceded by two flex factors
		// must be interpreted as a flex factor (CSS Flexbox 1 §7.1.1).
		n, isNumber := token.(*syntax.NumberToken)
		forcedFlexFactor := isNumber && n.IsInteger && n.IntValue == 0 && !(growFound && shrinkFound)
		if !basisFound && !forcedFlexFactor {
			ok, err := accepts("flex-basis", baseURL, token)
			if err != nil {
				return nil, err
			}
			if ok {
				basis, basisFound = token, true
				continue
			}
		}
		if !growFound || !shrinkFound {
			v := validateFlexGrowShrink(token)
			if v == nil {
				return nil, tk.ErrInvalid
			}
			if !growFound {
				grow, growFound = float64(v.(props.Number)), true
			} else {
				shrink, shrinkFound = float64(v.(props.Number)), true
			}
			continue
		}
		return nil, tk.ErrInvalid
	}

	first := tokens[0]
	line, column := first.Pos()
	factor := func(v float64) syntax.Node {
		n := syntax.Numeric{Value: v, Representation: formatFloat(v)}
		if v == float64(int(v)) {
			n.IsInteger, n.IntValue = true, int(v)
		}
		return syntax.NewNumberToken(line, column, n)
	}
	if !basisFound {
		basis = syntax.NewDimensionToken(line, column, syntax.Numeric{Representation: "0", IsInteger: true}, "px")
	}
	return []part{
		{"-grow", []syntax.Node{factor(grow)}},
		{"-shrink", []syntax.Node{factor(shrink)}},
		{"-basis", []syntax.Node{basis}},
	}, nil
}

func flexFlowParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	switch len(tokens) {
	case 2:
		for _, sorted := range [][]syntax.Node{tokens, {tokens[1], tokens[0]}} {
			direction, err := accepts("flex-direction", baseURL, sorted[0])
			if err != nil {
				return nil, err
			}
			wrap, err := accepts("flex-wrap", baseURL, sorted[1])
			if err != nil {
				return nil, err
			}
			if direction && wrap {
				return []part{{"flex-direction", sorted[:1]}, {"flex-wrap", sorted[1:]}}, nil
			}
		}
	case 1:
		if ok, _ := accepts("flex-direction", baseURL, tokens[0]); ok {
			return []part{{"flex-direction", tokens}}, nil
		}
		if ok, _ := accepts("flex-wrap", baseURL, tokens[0]); ok {
			return []part{{"flex-wrap", tokens}}, nil
		}
	}
	return nil, tk.ErrInvalid
}

func splitOnSlash(tokens []syntax.Node) [][]syntax.Node {
	parts := [][]syntax.Node{nil}
	for _, t := range tokens {
		if syntax.IsLiteral(t, "/") {
			parts = append(parts, nil)
		} else {
			parts[len(parts)-1] = append(parts[len(parts)-1], t)
		}
	}
	return parts
}

func isTemplate(tokens []syntax.Node, baseURL string) bool {
	if len(tokens) == 0 {
		return false
	}
	ok, _ := accepts("grid-template-columns", baseURL, tokens...)
	return ok
}

func gridTemplateParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	none := []syntax.Node{ident(tokens[0], "none")}
	if len(tokens) == 1 && tk.GetKeyword(tokens[0]) == "none" {
		return []part{{"-columns", none}, {"-rows", none}, {"-areas", none}}, nil
	}
	if slashed := splitOnSlash(tokens); len(slashed) == 2 {
		if !isTemplate(slashed[1], baseURL) {
			return nil, tk.ErrInvalid
		}
		if isTemplate(slashed[0], baseURL) {
			return []part{{"-columns", slashed[1]}, {"-rows", slashed[0]}, {"-areas", none}}, nil
		}
	}
	// The syntax with areas is not supported yet.
	return nil, tk.ErrInvalid
}

func gridParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	token := tokens[0]
	auto, none := []syntax.Node{ident(token, "auto")}, []syntax.Node{ident(token, "none")}

	if template, err := gridTemplateParts(tokens, "grid-template", baseURL); err == nil {
		parts := make([]part, 0, 6)
		for _, p := range template {
			parts = append(parts, part{"-template" + p.name, p.tokens})
		}
		return append(parts, part{"-auto-columns", auto}, part{"-auto-rows", auto}, part{"-auto-flow", []syntax.Node{ident(token, "row")}}), nil
	}

	split := splitOnSlash(tokens)
	if len(split) != 2 {
		return nil, tk.ErrInvalid
	}
	autoTrack := ""
	var dense syntax.Node
	templates := map[string][]syntax.Node{}
	for i, track := range []string{"row", "column"} {
		line := split[i]
		autoFlow := false
		for j, t := range line {
			switch tk.GetKeyword(t) {
			case "dense":
				if dense != nil || (autoTrack != "" && autoTrack != track) {
					return nil, tk.ErrInvalid
				}
				dense, autoTrack = t, track
			case "auto-flow":
				if autoFlow || (autoTrack != "" && autoTrack != track) {
					return nil, tk.ErrInvalid
				}
				autoFlow, autoTrack = true, track
			default:
				if j != len(line)-1 {
					return nil, tk.ErrInvalid
				}
				templates[track] = append(templates[track], t)
			}
		}
	}
	if autoTrack == "" {
		return nil, tk.ErrInvalid
	}
	nonAuto := "row"
	if autoTrack == "row" {
		nonAuto = "column"
	}
	flow := []syntax.Node{ident(token, autoTrack)}
	if dense != nil {
		flow = append(flow, dense)
	}
	return []part{
		{"-auto-flow", flow},
		{"-auto-" + autoTrack + "s", templates[autoTrack]},
		{"-auto-" + nonAuto + "s", auto},
		{"-template-" + autoTrack + "s", none},
		{"-template-" + nonAuto + "s", templates[nonAuto]},
		{"-template-areas", none},
	}, nil
}

// gridLines splits the value of grid-column, grid-row and grid-area into the
// lines they set. A missing end copies the start when it is a name, and is
// auto otherwise (CSS Grid 2 §8.4).
func gridLines(tokens []syntax.Node, max int) ([][]syntax.Node, error) {
	lines := splitOnSlash(tokens)
	if len(lines) < 1 || len(lines) > max {
		return nil, tk.ErrInvalid
	}
	last := tokens[len(tokens)-1]
	auto := []syntax.Node{ident(last, "auto")}
	var out [][]syntax.Node
	var named []bool
	for _, line := range lines {
		v, err := validateGridLine(line, "")
		if err != nil || v == nil {
			return out, tk.ErrInvalid
		}
		g := v.(props.GridLine)
		named = append(named, !g.Auto && !g.Span && g.N == 0)
		out = append(out, line)
	}
	count := len(lines)
	copyOrAuto := func(i int) []syntax.Node {
		if named[i] {
			return lines[i]
		}
		return auto
	}
	if count <= 1 {
		out = append(out, copyOrAuto(0))
		lines = append(lines, lines[len(lines)-1])
		named = append(named, named[0])
	}
	if count <= 2 && 2 < max {
		out = append(out, copyOrAuto(0))
	}
	if count <= 3 && 3 < max {
		out = append(out, copyOrAuto(1))
	}
	return out, nil
}

func gridColumnRowParts(tokens []syntax.Node, _, _ string) ([]part, error) {
	lines, err := gridLines(tokens, 2)
	var parts []part
	for i, side := range []string{"-start", "-end"} {
		if i < len(lines) {
			parts = append(parts, part{side, lines[i]})
		}
	}
	return parts, err
}

func gridAreaParts(tokens []syntax.Node, _, _ string) ([]part, error) {
	lines, err := gridLines(tokens, 4)
	var parts []part
	for i, side := range []string{"row-start", "column-start", "row-end", "column-end"} {
		if i < len(lines) {
			parts = append(parts, part{"grid-" + side, lines[i]})
		}
	}
	return parts, err
}

func gapParts(tokens []syntax.Node, _, baseURL string) ([]part, error) {
	switch len(tokens) {
	case 1:
		if ok, err := accepts("column-gap", baseURL, tokens...); err != nil || !ok {
			return nil, firstNonNil(err, tk.ErrInvalid)
		}
		return []part{{"row-gap", tokens}, {"column-gap", tokens}}, nil
	case 2:
		for _, t := range tokens {
			if ok, err := accepts("column-gap", baseURL, t); err != nil || !ok {
				return nil, firstNonNil(err, tk.ErrInvalid)
			}
		}
		return []part{{"row-gap", tokens[:1]}, {"column-gap", tokens[1:]}}, nil
	}
	return nil, tk.ErrInvalid
}

func legacyGapParts(name string) partsFunc {
	return func(tokens []syntax.Node, _, baseURL string) ([]part, error) {
		if ok, err := accepts(name, baseURL, tokens...); err != nil || !ok {
			return nil, firstNonNil(err, tk.ErrInvalid)
		}
		return []part{{name, tokens}}, nil
	}
}
