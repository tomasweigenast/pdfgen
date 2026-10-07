package tokens

import (
	"math"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

type direction struct {
	typ    props.Keyword
	angle  float64
	corner props.Keyword
}

// Directions of linear-gradient() given as keywords (CSS Images 3 §3.1.1),
// keyed by the keywords joined with spaces. Angles are in radians, 0
// pointing upwards and growing clockwise.
var directionKeywords = map[string]direction{
	"to top":          {typ: "angle", angle: 0},
	"to right":        {typ: "angle", angle: math.Pi / 2},
	"to bottom":       {typ: "angle", angle: math.Pi},
	"to left":         {typ: "angle", angle: math.Pi * 3 / 2},
	"to top left":     {typ: "corner", corner: "top_left"},
	"to left top":     {typ: "corner", corner: "top_left"},
	"to top right":    {typ: "corner", corner: "top_right"},
	"to right top":    {typ: "corner", corner: "top_right"},
	"to bottom left":  {typ: "corner", corner: "bottom_left"},
	"to left bottom":  {typ: "corner", corner: "bottom_left"},
	"to bottom right": {typ: "corner", corner: "bottom_right"},
	"to right bottom": {typ: "corner", corner: "bottom_right"},
}

func parseColorHint(tokens []syntax.Node) (props.Length, bool) {
	if len(tokens) == 1 {
		return GetLength(tokens[0], true, true)
	}
	return props.Length{}, false
}

func parseColorStop(tokens []syntax.Node) (props.ColorStop, error) {
	switch len(tokens) {
	case 1:
		color, ok := syntax.ParseColor(tokens[0])
		if ok && color.IsCurrentColor() {
			// The current color is not known yet.
			color, _ = syntax.ParseColorString("black")
		}
		if ok {
			return props.ColorStop{Color: props.Color{Color: color}}, nil
		}
	case 2:
		color, ok := syntax.ParseColor(tokens[0])
		position, okPosition := GetLength(tokens[1], true, true)
		if ok && okPosition {
			return props.ColorStop{Color: props.Color{Color: color}, Position: &position}, nil
		}
	}
	return props.ColorStop{}, ErrInvalid
}

func parseColorStopsAndHints(parts [][]syntax.Node) ([]props.ColorStop, []props.Length, error) {
	if len(parts) == 0 {
		return nil, nil, ErrInvalid
	}
	first, err := parseColorStop(parts[0])
	if err != nil {
		return nil, nil, err
	}
	stops := []props.ColorStop{first}
	var hints []props.Length
	previousWasColorStop := true
	for _, tokens := range parts[1:] {
		if hint, ok := parseColorHint(tokens); ok {
			hints = append(hints, hint)
			previousWasColorStop = false
			continue
		}
		if previousWasColorStop {
			hints = append(hints, percentLength(fiftyPercent))
		}
		stop, err := parseColorStop(tokens)
		if err != nil {
			return nil, nil, err
		}
		stops = append(stops, stop)
		previousWasColorStop = true
	}
	if !previousWasColorStop {
		return nil, nil, ErrInvalid
	}
	return stops, hints, nil
}

func parseLinearGradientParameters(arguments [][]syntax.Node) (direction, [][]syntax.Node) {
	first := arguments[0]
	if len(first) == 1 {
		if angle, ok := GetAngle(first[0]); ok {
			return direction{typ: "angle", angle: angle}, arguments[1:]
		}
	} else {
		keywords := make([]string, len(first))
		for i, t := range first {
			keywords[i] = GetKeyword(t)
		}
		if d, ok := directionKeywords[strings.Join(keywords, " ")]; ok {
			return d, arguments[1:]
		}
	}
	// The default direction is "to bottom".
	return direction{typ: "angle", angle: math.Pi}, arguments
}

type radialParameters struct {
	shape      props.Keyword
	sizeType   props.Keyword
	sizeKw     props.Keyword
	sizeLength [2]props.Length
	position   props.Position
	rest       [][]syntax.Node
}

func parseRadialGradientParameters(arguments [][]syntax.Node) (radialParameters, bool) {
	var shape, sizeShape props.Keyword
	var sizeType, sizeKw props.Keyword
	var explicit [2]props.Length
	var position *props.Position

	// The first argument is consumed as a stack, from its end.
	stack := make([]syntax.Node, len(arguments[0]))
	for i, t := range arguments[0] {
		stack[len(stack)-1-i] = t
	}
	for len(stack) > 0 {
		token := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		keyword := GetKeyword(token)
		switch {
		case keyword == "at":
			rest := make([]syntax.Node, len(stack))
			for i, t := range stack {
				rest[len(stack)-1-i] = t
			}
			p, ok := ParsePosition(rest)
			if !ok {
				return radialParameters{}, false
			}
			position = &p
			stack = nil
		case (keyword == "circle" || keyword == "ellipse") && shape == "":
			shape = props.Keyword(keyword)
		case isOneOf(keyword, "closest-corner", "farthest-corner", "closest-side", "farthest-side") && sizeType == "":
			sizeType, sizeKw = "keyword", props.Keyword(keyword)
		default:
			if len(stack) > 0 && sizeType == "" {
				l1, ok1 := GetLength(token, true, true)
				l2, ok2 := GetLength(stack[len(stack)-1], true, true)
				if ok1 && ok2 {
					sizeType, explicit = "explicit", [2]props.Length{l1, l2}
					sizeShape = "ellipse"
					stack = stack[:len(stack)-1]
				}
			}
			if sizeType == "" {
				if l1, ok := GetLength(token, true, false); ok {
					sizeType, explicit = "explicit", [2]props.Length{l1, l1}
					sizeShape = "circle"
				}
			}
			if sizeType == "" {
				return radialParameters{}, false
			}
		}
	}
	// A circle cannot have two radii.
	if shape == "circle" && sizeShape == "ellipse" {
		return radialParameters{}, false
	}

	p := radialParameters{shape: shape, sizeType: sizeType, sizeKw: sizeKw, sizeLength: explicit, rest: arguments[1:]}
	if p.shape == "" {
		p.shape = sizeShape
	}
	if p.shape == "" {
		p.shape = "ellipse"
	}
	if p.sizeType == "" {
		p.sizeType, p.sizeKw = "keyword", "farthest-corner"
	}
	if position != nil {
		p.position = *position
	} else {
		p.position = centerPosition()
	}
	return p, true
}

func centerPosition() props.Position {
	return props.Position{OriginX: "left", X: percentLength(fiftyPercent), OriginY: "top", Y: percentLength(fiftyPercent)}
}

// GetImage parses an <image>: a url or a gradient (CSS Images 3 §2, §3).
func GetImage(token syntax.Node, baseURL string) (props.Image, bool, error) {
	url, ok, err := GetURL(token, baseURL)
	if err != nil {
		return props.Image{}, false, err
	}
	if ok {
		if address, ok := url.External(); ok {
			return props.Image{Type: "url", URL: address}, true, nil
		}
		return props.Image{}, false, nil
	}

	f := NewFunction(token)
	arguments := f.SplitCommaGroups(false)
	if len(arguments) == 0 {
		return props.Image{}, false, nil
	}
	repeating := strings.HasPrefix(f.Name, "repeating-")
	switch f.Name {
	case "linear-gradient", "repeating-linear-gradient":
		d, rest := parseLinearGradientParameters(arguments)
		stops, hints, err := parseColorStopsAndHints(rest)
		if err != nil {
			return props.Image{}, false, err
		}
		return props.Image{Type: "linear-gradient", Linear: &props.LinearGradient{
			Gradient:      props.Gradient{ColorStops: stops, ColorHints: hints, Repeating: repeating},
			DirectionType: d.typ, Angle: d.angle, Corner: d.corner,
		}}, true, nil
	case "radial-gradient", "repeating-radial-gradient":
		parameters, ok := parseRadialGradientParameters(arguments)
		if !ok {
			parameters = radialParameters{
				shape: "ellipse", sizeType: "keyword", sizeKw: "farthest-corner",
				position: centerPosition(), rest: arguments,
			}
		}
		stops, hints, err := parseColorStopsAndHints(parameters.rest)
		if err != nil {
			return props.Image{}, false, err
		}
		return props.Image{Type: "radial-gradient", Radial: &props.RadialGradient{
			Gradient: props.Gradient{ColorStops: stops, ColorHints: hints, Repeating: repeating},
			Shape:    parameters.shape, SizeType: parameters.sizeType, SizeKeyword: parameters.sizeKw,
			SizeExplicit: parameters.sizeLength, Center: parameters.position,
		}}, true, nil
	}
	return props.Image{}, false, nil
}
