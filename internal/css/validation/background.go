package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	register(commaSeparated[props.Keywords](keywordItem("scroll", "fixed", "local")), 0, "background-attachment")
	register(commaSeparated[props.Images](backgroundImageItem), 0, "background-image")
	register(commaSeparated[props.Positions](positionItem), 0, "background-position", "object-position")
	register(commaSeparated[props.Repeats](backgroundRepeatItem), 0, "background-repeat")
	register(commaSeparated[props.BackgroundSizes](backgroundSizeItem), 0, "background-size")
	register(commaSeparated[props.Keywords](boxItem), 0, "background-clip", "background-origin")
	register(validateTransformOrigin, 0, "transform-origin")
}

var boxItem = keywordItem("border-box", "padding-box", "content-box")

func backgroundImageItem(tokens []syntax.Node, baseURL string) (props.Image, bool, error) {
	if len(tokens) != 1 {
		return props.Image{}, false, nil
	}
	if tk.GetKeyword(tokens[0]) == "none" {
		return props.Image{Type: "none"}, true, nil
	}
	return tk.GetImage(tokens[0], baseURL)
}

func positionItem(tokens []syntax.Node, _ string) (props.Position, bool, error) {
	p, ok := tk.ParsePosition(tokens)
	return p, ok, nil
}

var repeatKeywords = []string{"no-repeat", "repeat", "space", "round"}

func backgroundRepeatItem(tokens []syntax.Node, _ string) (props.Repeat, bool, error) {
	keywords := make([]string, len(tokens))
	for i, token := range tokens {
		keywords[i] = tk.GetKeyword(token)
	}
	switch {
	case len(keywords) == 1 && keywords[0] == "repeat-x":
		return props.Repeat{X: "repeat", Y: "no-repeat"}, true, nil
	case len(keywords) == 1 && keywords[0] == "repeat-y":
		return props.Repeat{X: "no-repeat", Y: "repeat"}, true, nil
	case len(keywords) == 1 && contains(repeatKeywords, keywords[0]):
		return props.Repeat{X: props.Keyword(keywords[0]), Y: props.Keyword(keywords[0])}, true, nil
	case len(keywords) == 2 && contains(repeatKeywords, keywords[0]) && contains(repeatKeywords, keywords[1]):
		return props.Repeat{X: props.Keyword(keywords[0]), Y: props.Keyword(keywords[1])}, true, nil
	}
	return props.Repeat{}, false, nil
}

func backgroundSizeItem(tokens []syntax.Node, _ string) (props.BackgroundSize, bool, error) {
	switch len(tokens) {
	case 1:
		token := tokens[0]
		switch keyword := tk.GetKeyword(token); keyword {
		case "contain", "cover":
			return props.BackgroundSize{Keyword: props.Keyword(keyword)}, true, nil
		case "auto":
			return props.BackgroundSize{Width: auto, Height: auto}, true, nil
		}
		if l, ok := tk.GetLength(token, false, true); ok {
			return props.BackgroundSize{Width: l, Height: auto}, true, nil
		}
	case 2:
		var values []props.Length
		for _, token := range tokens {
			if l, ok := tk.GetLength(token, false, true); ok {
				values = append(values, l)
			} else if tk.GetKeyword(token) == "auto" {
				values = append(values, auto)
			}
		}
		if len(values) == 2 {
			return props.BackgroundSize{Width: values[0], Height: values[1]}, true, nil
		}
	}
	return props.BackgroundSize{}, false, nil
}

func validateTransformOrigin(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) == 3 {
		// The third value is ignored as 3D transforms are not supported.
		tokens = tokens[:2]
	}
	x, y, ok := tk.Parse2DPosition(tokens)
	if !ok {
		return nil, nil
	}
	return props.LengthPair{X: x, Y: y}, nil
}
