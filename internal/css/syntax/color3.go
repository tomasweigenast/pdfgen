package syntax

import (
	"math"
)

// RGBA is a color of CSS Color Level 3: four floats in [0, 1] for red, green,
// blue and alpha. Alpha is clipped, the other channels may be out of range
// (rgb(-10%, 120%, 0%) is {-0.1, 1.2, 0, 1}).
type RGBA struct {
	R, G, B, A float64
}

// ParseColor3 parses a color as defined by CSS Color Level 3. It reports
// false for anything that is not a valid color. For the currentcolor keyword
// it returns isCurrentColor true and a zero RGBA.
func ParseColor3(token Node) (c RGBA, isCurrentColor, ok bool) {
	switch t := token.(type) {
	case *IdentToken:
		switch t.LowerValue {
		case "currentcolor":
			return RGBA{}, true, true
		case "transparent":
			return RGBA{}, false, true
		}
		if rgb, found := color3Keywords[t.LowerValue]; found {
			return RGBA{float64(rgb[0]) / 255, float64(rgb[1]) / 255, float64(rgb[2]) / 255, 1}, false, true
		}
	case *HashToken:
		if ch, found := hexChannels(t.Value); found {
			c := RGBA{ch[0], ch[1], ch[2], 1}
			if len(ch) == 4 {
				c.A = ch[3]
			}
			return c, false, true
		}
	case *FunctionBlock:
		args, valid := commaSeparated(t.Arguments)
		if len(args) == 0 || !valid {
			return RGBA{}, false, false
		}
		var res RGBA
		var found bool
		switch t.LowerName {
		case "rgb":
			res, found = rgb3(args, 1)
		case "rgba":
			if a, good := alpha3(args[min(3, len(args)):]); good {
				res, found = rgb3(args[:min(3, len(args))], a)
			}
		case "hsl":
			res, found = hsl3(args, 1)
		case "hsla":
			if a, good := alpha3(args[min(3, len(args)):]); good {
				res, found = hsl3(args[:min(3, len(args))], a)
			}
		}
		return res, false, found
	}
	return RGBA{}, false, false
}

// ParseColor3String parses a color from CSS text; see ParseColor3.
func ParseColor3String(css string) (c RGBA, isCurrentColor, ok bool) {
	return ParseColor3(ParseOneComponentValue(css, true))
}

// commaSeparated returns the arguments made of a single token each and
// separated by mandatory commas, with whitespace and comments ignored.
// valid is false when the content does not have that shape.
func commaSeparated(nodes []Node) (args []Node, valid bool) {
	tokens := significant(nodes)
	if len(tokens) == 0 {
		return nil, true
	}
	if len(tokens)%2 == 1 && allCommas(everyOther(tokens, 1)) {
		return everyOther(tokens, 0), true
	}
	return nil, false
}

func alpha3(args []Node) (float64, bool) {
	if len(args) == 1 {
		if n, ok := args[0].(*NumberToken); ok {
			return math.Min(1, math.Max(0, n.Value)), true
		}
	}
	return 0, false
}

func rgb3(args []Node, alpha float64) (RGBA, bool) {
	if len(args) != 3 {
		return RGBA{}, false
	}
	var v [3]float64
	switch args[0].Type() {
	case "number":
		for i, a := range args {
			n, ok := a.(*NumberToken)
			if !ok || !n.IsInteger {
				return RGBA{}, false
			}
			v[i] = float64(n.IntValue) / 255
		}
	case "percentage":
		for i, a := range args {
			n, ok := a.(*PercentageToken)
			if !ok {
				return RGBA{}, false
			}
			v[i] = n.Value / 100
		}
	default:
		return RGBA{}, false
	}
	return RGBA{v[0], v[1], v[2], alpha}, true
}

func hsl3(args []Node, alpha float64) (RGBA, bool) {
	if len(args) != 3 {
		return RGBA{}, false
	}
	h, ok1 := args[0].(*NumberToken)
	s, ok2 := args[1].(*PercentageToken)
	l, ok3 := args[2].(*PercentageToken)
	if !ok1 || !ok2 || !ok3 {
		return RGBA{}, false
	}
	r, g, b := hlsToRGB(h.Value/360, l.Value/100, s.Value/100)
	return RGBA{r, g, b, alpha}, true
}
