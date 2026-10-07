package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

func init() {
	registerExpander(expandBackground, "background")
}

var backgroundLonghands = []string{
	"background-color", "background-image", "background-repeat",
	"background-attachment", "background-position", "background-size",
	"background-clip", "background-origin",
}

var transparent = props.Color{Color: syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)}

// layer holds the values found for one background layer by their longhand.
type layer map[string]props.Value

func (l layer) add(name string, v props.Value) (bool, error) {
	if v == nil {
		return false, nil
	}
	name = "background-" + name
	if _, dup := l[name]; dup {
		return false, tk.ErrInvalid
	}
	l[name] = v
	return true, nil
}

// initialLayer is the value that a layer gets for the longhands it does not
// set.
func initialLayer(name string) props.Value {
	switch name {
	case "background-image":
		return props.Image{Type: "none"}
	case "background-repeat":
		return props.Repeat{X: "repeat", Y: "repeat"}
	case "background-attachment":
		return props.Keyword("scroll")
	case "background-position":
		return props.Position{OriginX: "left", X: pctLength(0), OriginY: "top", Y: pctLength(0)}
	case "background-size":
		return props.BackgroundSize{Width: auto, Height: auto}
	case "background-clip":
		return props.Keyword("border-box")
	}
	return props.Keyword("padding-box")
}

func pctLength(v float64) props.Length {
	return props.Length{Dimension: props.Dimension{Value: v, Unit: "%"}}
}

// single runs the validator of one list item and returns nil when invalid.
func single[E props.Value](f item[E], baseURL string, tokens []syntax.Node) (props.Value, error) {
	v, ok, err := f(tokens, baseURL)
	if err != nil || !ok {
		return nil, err
	}
	return v, nil
}

func reverse(tokens []syntax.Node) []syntax.Node {
	out := make([]syntax.Node, len(tokens))
	for i, t := range tokens {
		out[len(tokens)-1-i] = t
	}
	return out
}

// parseBackgroundLayer parses one layer of background (CSS Backgrounds 3
// §2.10). Tokens are consumed from a stack, so its top is the next token.
func parseBackgroundLayer(tokens []syntax.Node, finalLayer bool, baseURL string) (props.Value, layer, error) {
	results := layer{}
	stack := reverse(tokens)
	pop := func(n int) { stack = stack[:len(stack)-n] }
	// top returns the n topmost tokens in source order.
	top := func(n int) []syntax.Node {
		if n > len(stack) {
			n = len(stack)
		}
		return reverse(stack[len(stack)-n:])
	}

	for len(stack) > 0 {
		v, err := single(backgroundRepeatItem, baseURL, top(2))
		if err != nil {
			return nil, nil, err
		}
		if ok, err := results.add("repeat", v); err != nil {
			return nil, nil, err
		} else if ok {
			pop(min(2, len(stack)))
			continue
		}

		token := top(1)
		if finalLayer {
			var c props.Value
			if len(token) == 1 {
				c = validateOtherColor(token[0])
			}
			if ok, err := results.add("color", c); err != nil {
				return nil, nil, err
			} else if ok {
				pop(1)
				continue
			}
		}
		if v, err = single(backgroundImageItem, baseURL, token); err != nil {
			return nil, nil, err
		}
		if ok, err := results.add("image", v); err != nil {
			return nil, nil, err
		} else if ok {
			pop(1)
			continue
		}
		if v, err = single(backgroundRepeatItem, baseURL, token); err != nil {
			return nil, nil, err
		}
		if ok, err := results.add("repeat", v); err != nil {
			return nil, nil, err
		} else if ok {
			pop(1)
			continue
		}
		if v, err = single(keywordItem("scroll", "fixed", "local"), baseURL, token); err != nil {
			return nil, nil, err
		}
		if ok, err := results.add("attachment", v); err != nil {
			return nil, nil, err
		} else if ok {
			pop(1)
			continue
		}

		found := false
		for n := min(len(stack), 4); n >= 1; n-- {
			position, ok, _ := positionItem(top(n), baseURL)
			if !ok {
				continue
			}
			found = true
			if _, err := results.add("position", position); err != nil {
				return nil, nil, err
			}
			pop(n)
			if len(stack) > 0 && syntax.IsLiteral(stack[len(stack)-1], "/") {
				sizes := []int{3, 2}
				if len(stack) < 2 {
					sizes = []int{2}
				}
				for _, n := range sizes {
					// n includes the "/".
					if n > len(stack) {
						continue
					}
					candidate := reverse(stack[len(stack)-n : len(stack)-1])
					size, ok, _ := backgroundSizeItem(candidate, baseURL)
					if ok {
						if _, err := results.add("size", size); err != nil {
							return nil, nil, err
						}
						pop(n)
					}
				}
			}
			break
		}
		if found {
			continue
		}

		if v, err = single(boxItem, baseURL, token); err != nil {
			return nil, nil, err
		}
		if ok, err := results.add("origin", v); err != nil {
			return nil, nil, err
		} else if ok {
			pop(1)
			next, err := single(boxItem, baseURL, top(1))
			if err != nil {
				return nil, nil, err
			}
			if ok, _ := results.add("clip", next); ok {
				pop(1)
			} else {
				// The same keyword sets both.
				results.add("clip", v)
			}
			continue
		}
		return nil, nil, tk.ErrInvalid
	}

	var color props.Value = transparent
	if c, ok := results["background-color"]; ok {
		color = c
		delete(results, "background-color")
	}
	for _, name := range backgroundLonghands[1:] {
		if _, ok := results[name]; !ok {
			results[name] = initialLayer(name)
		}
	}
	return color, results, nil
}

func expandBackground(tokens []syntax.Node, name, baseURL string) ([]pair, error) {
	if keyword := tk.GetSingleKeyword(tokens); keyword == "initial" || keyword == "inherit" {
		pairs := make([]pair, len(backgroundLonghands))
		for i, n := range backgroundLonghands {
			pairs[i] = pair{n, props.Keyword(keyword)}
		}
		return pairs, nil
	}
	if hasVar(tokens) {
		pending := &props.Pending{Name: name, Tokens: tokens}
		pairs := make([]pair, len(backgroundLonghands))
		for i, n := range backgroundLonghands {
			pairs[i] = pair{n, pending}
		}
		return pairs, nil
	}

	parts := tk.SplitOnComma(tokens)
	// The color is only allowed in the last layer, which is parsed first.
	color, last, err := parseBackgroundLayer(parts[len(parts)-1], true, baseURL)
	if err != nil {
		return nil, err
	}
	layers := []layer{last}
	for i := len(parts) - 2; i >= 0; i-- {
		_, l, err := parseBackgroundLayer(parts[i], false, baseURL)
		if err != nil {
			return nil, err
		}
		layers = append(layers, l)
	}

	pairs := make([]pair, 0, len(backgroundLonghands))
	for _, n := range backgroundLonghands[1:] {
		var values []props.Value
		for i := len(layers) - 1; i >= 0; i-- {
			values = append(values, layers[i][n])
		}
		pairs = append(pairs, pair{n, backgroundList(n, values)})
	}
	return append(pairs, pair{"background-color", color}), nil
}

// backgroundList makes the typed list of a longhand from the value of each
// layer.
func backgroundList(name string, values []props.Value) props.Value {
	switch name {
	case "background-image":
		list := make(props.Images, len(values))
		for i, v := range values {
			list[i] = v.(props.Image)
		}
		return list
	case "background-repeat":
		list := make(props.Repeats, len(values))
		for i, v := range values {
			list[i] = v.(props.Repeat)
		}
		return list
	case "background-position":
		list := make(props.Positions, len(values))
		for i, v := range values {
			list[i] = v.(props.Position)
		}
		return list
	case "background-size":
		list := make(props.BackgroundSizes, len(values))
		for i, v := range values {
			list[i] = v.(props.BackgroundSize)
		}
		return list
	}
	list := make(props.Keywords, len(values))
	for i, v := range values {
		list[i] = v.(props.Keyword)
	}
	return list
}
