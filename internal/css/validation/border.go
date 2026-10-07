package validation

import (
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
)

var (
	zeroPixels    = props.Px(0)
	currentColor  = props.Color{Color: syntax.Color{Space: syntax.CurrentColor, Alpha: 1}}
	borderStyles  = []string{"none", "hidden", "dotted", "dashed", "double", "inset", "outset", "groove", "ridge", "solid"}
	outlineStyles = []string{"none", "dotted", "dashed", "double", "inset", "outset", "groove", "ridge", "solid"}
)

func init() {
	register(singleToken(plain(validateOtherColor)), 0,
		"background-color",
		"border-top-color", "border-right-color", "border-bottom-color", "border-left-color",
		"border-block-start-color", "border-block-end-color",
		"border-inline-start-color", "border-inline-end-color",
		"text-decoration-color")
	register(singleToken(plain(validateOtherColor)), flagUnstable, "column-rule-color")
	register(singleToken(plain(validateOutlineColor)), 0, "outline-color")
	register(singleToken(plain(validateColor)), 0, "color")
	register(validateBorderSpacing, 0, "border-spacing")
	register(validateBorderRadius, 0,
		"border-top-right-radius", "border-bottom-right-radius",
		"border-bottom-left-radius", "border-top-left-radius",
		"border-start-start-radius", "border-start-end-radius",
		"border-end-start-radius", "border-end-end-radius")
	register(keywordOf(borderStyles...), 0,
		"border-top-style", "border-right-style", "border-left-style", "border-bottom-style",
		"border-block-start-style", "border-block-end-style",
		"border-inline-start-style", "border-inline-end-style")
	register(keywordOf(borderStyles...), flagUnstable, "column-rule-style")
	register(keywordOf(outlineStyles...), 0, "outline-style")
	register(singleToken(plain(borderWidth)), 0,
		"border-top-width", "border-right-width", "border-left-width", "border-bottom-width",
		"border-block-start-width", "border-block-end-width",
		"border-inline-start-width", "border-inline-end-width",
		"outline-width")
	register(singleToken(plain(borderWidth)), flagUnstable, "column-rule-width")
	register(validateBoxShadow, 0, "box-shadow")
	register(singleToken(borderImageSource), 0, "border-image-source", "mask-border-source")
	register(validateBorderImageSlice, 0, "border-image-slice", "mask-border-slice")
	register(validateBorderImageWidth, 0, "border-image-width", "mask-border-width")
	register(validateBorderImageOutset, 0, "border-image-outset", "mask-border-outset")
	register(validateBorderImageRepeat, 0, "border-image-repeat", "mask-border-repeat")
	register(keywordOf("luminance", "alpha"), 0, "mask-border-mode")
}

// lightDark reports whether token uses light-dark(), which gives a different
// color depending on color-scheme.
func lightDark(token syntax.Node) bool {
	fn, ok := token.(*syntax.FunctionBlock)
	if !ok {
		return false
	}
	if fn.LowerName == "light-dark" {
		return true
	}
	for _, argument := range fn.Arguments {
		if lightDark(argument) {
			return true
		}
	}
	return false
}

// parseColor parses a color. The tokens of a color that depends on the color
// scheme are kept, to be parsed again once the scheme is known.
func parseColor(token syntax.Node) (props.Color, bool) {
	c, ok := syntax.ParseColor(token)
	if !ok {
		return props.Color{}, false
	}
	color := props.Color{Color: c}
	if lightDark(token) {
		color.Token = token
	}
	return color, true
}

func validateOtherColor(token syntax.Node) props.Value {
	if c, ok := parseColor(token); ok {
		return c
	}
	return nil
}

func validateOutlineColor(token syntax.Node) props.Value {
	if tk.GetKeyword(token) == "invert" {
		return currentColor
	}
	return validateOtherColor(token)
}

func validateColor(token syntax.Node) props.Value {
	if c, ok := parseColor(token); ok {
		if c.IsCurrentColor() {
			return props.Keyword("inherit")
		}
		return c
	}
	return nil
}

func validateBorderSpacing(tokens []syntax.Node, _ string) (props.Value, error) {
	lengths, ok := lengthList(tokens, false, false)
	switch {
	case !ok:
		return nil, nil
	case len(lengths) == 1:
		return props.LengthPair{X: lengths[0], Y: lengths[0]}, nil
	case len(lengths) == 2:
		return props.LengthPair{X: lengths[0], Y: lengths[1]}, nil
	}
	return nil, nil
}

func validateBorderRadius(tokens []syntax.Node, _ string) (props.Value, error) {
	lengths, ok := lengthList(tokens, false, true)
	switch {
	case !ok:
		return nil, nil
	case len(lengths) == 1:
		return props.LengthPair{X: lengths[0], Y: lengths[0]}, nil
	case len(lengths) == 2:
		return props.LengthPair{X: lengths[0], Y: lengths[1]}, nil
	}
	return nil, nil
}

// lengthList parses every token as a length. It reports false if any of them
// is not one.
func lengthList(tokens []syntax.Node, negative, percentage bool) ([]props.Length, bool) {
	lengths := make([]props.Length, len(tokens))
	for i, token := range tokens {
		l, ok := tk.GetLength(token, negative, percentage)
		if !ok {
			return nil, false
		}
		lengths[i] = l
	}
	return lengths, true
}

func borderWidth(token syntax.Node) props.Value {
	if l, ok := tk.GetLength(token, false, false); ok {
		return l
	}
	switch k := tk.GetKeyword(token); k {
	case "thin", "medium", "thick":
		return props.LengthKeyword(props.Keyword(k))
	}
	return nil
}

func validateBoxShadow(tokens []syntax.Node, _ string) (props.Value, error) {
	if tk.GetSingleKeyword(tokens) == "none" {
		return props.BoxShadows{}, nil
	}
	shadows := props.BoxShadows{}
	for len(tokens) > 0 {
		var lengths []props.Length
		var color *props.Color
		inset, lastLength := false, false
		for len(tokens) > 0 {
			token := tokens[0]
			tokens = tokens[1:]
			if syntax.IsLiteral(token, ",") && len(tokens) > 0 {
				break
			}
			if ident, ok := token.(*syntax.IdentToken); ok && ident.Value == "inset" && !inset {
				inset, lastLength = true, false
				continue
			}
			if l, ok := tk.GetLength(token, true, false); ok && !(len(lengths) > 0 && !lastLength) {
				lengths = append(lengths, l)
				lastLength = true
				continue
			}
			if c, ok := parseColor(token); ok && color == nil {
				color = &c
				lastLength = false
				continue
			}
			return nil, nil
		}
		if len(lengths) < 2 || len(lengths) > 4 {
			return nil, nil
		}
		shadow := props.BoxShadow{X: lengths[0], Y: lengths[1], Blur: zeroPixels, Spread: zeroPixels, Color: currentColor, Inset: inset}
		if len(lengths) > 2 {
			shadow.Blur = lengths[2]
			if shadow.Blur.Calc == nil && shadow.Blur.Dimension.Value < 0 {
				return nil, nil
			}
		}
		if len(lengths) > 3 {
			shadow.Spread = lengths[3]
		}
		if color != nil {
			shadow.Color = *color
		}
		shadows = append(shadows, shadow)
	}
	return shadows, nil
}

func borderImageSource(token syntax.Node, baseURL string) (props.Value, error) {
	if tk.GetKeyword(token) == "none" {
		return props.Image{Type: "none"}, nil
	}
	img, ok, err := tk.GetImage(token, baseURL)
	if err != nil || !ok {
		return nil, err
	}
	return img, nil
}

func validateBorderImageSlice(tokens []syntax.Node, _ string) (props.Value, error) {
	v, ok := parseBorderImageSlice(tokens)
	if !ok {
		return nil, nil
	}
	return v, nil
}

// parseBorderImageSlice parses border-image-slice, with the numbers or
// percentages expanded to the four sides.
func parseBorderImageSlice(tokens []syntax.Node) (props.ImageSlice, bool) {
	var values []props.Dimension
	fill := false
	for i, token := range tokens {
		// A length is not used, because a dimension with a unit is not
		// allowed.
		if p, ok := tk.GetPercentage(token, false); ok {
			values = append(values, p)
		} else if tk.GetKeyword(token) == "fill" && !fill && (i == 0 || i == len(tokens)-1) {
			fill = true
		} else if n, ok := tk.GetNumber(token, false, false); ok {
			values = append(values, n)
		} else {
			return props.ImageSlice{}, false
		}
	}
	if len(values) < 1 || len(values) > 4 {
		return props.ImageSlice{}, false
	}
	slice := props.ImageSlice{Fill: fill}
	edges := expandSides(values)
	copy(slice.Edges[:], edges)
	return slice, true
}

// expandSides repeats one to four values over the top, right, bottom and left
// sides (CSS Box 4 §7.1).
func expandSides[T any](values []T) []T {
	switch len(values) {
	case 1:
		return []T{values[0], values[0], values[0], values[0]}
	case 2:
		return []T{values[0], values[1], values[0], values[1]}
	case 3:
		return []T{values[0], values[1], values[2], values[1]}
	}
	return values
}

func validateBorderImageWidth(tokens []syntax.Node, _ string) (props.Value, error) {
	return parseLengthQuad(tokens, true), nil
}

func validateBorderImageOutset(tokens []syntax.Node, _ string) (props.Value, error) {
	return parseLengthQuad(tokens, false), nil
}

// parseLengthQuad parses the values of border-image-width (auto, numbers,
// lengths and percentages) and of border-image-outset (numbers and
// lengths), and expands them to four sides. It returns nil if invalid.
func parseLengthQuad(tokens []syntax.Node, widthSyntax bool) props.Value {
	var values []props.Length
	for _, token := range tokens {
		if widthSyntax && tk.GetKeyword(token) == "auto" {
			values = append(values, auto)
		} else if n, ok := tk.GetNumber(token, false, false); ok {
			values = append(values, props.Length{Dimension: n})
		} else if l, ok := tk.GetLength(token, false, widthSyntax); ok {
			values = append(values, l)
		} else {
			return nil
		}
	}
	if len(values) < 1 || len(values) > 4 {
		return nil
	}
	var quad props.LengthQuad
	copy(quad[:], expandSides(values))
	return quad
}

func validateBorderImageRepeat(tokens []syntax.Node, _ string) (props.Value, error) {
	if len(tokens) < 1 || len(tokens) > 2 {
		return nil, nil
	}
	var keywords []props.Keyword
	for _, token := range tokens {
		k := tk.GetKeyword(token)
		if !contains([]string{"stretch", "repeat", "round", "space"}, k) {
			return nil, nil
		}
		keywords = append(keywords, props.Keyword(k))
	}
	if len(keywords) == 1 {
		keywords = append(keywords, keywords[0])
	}
	return props.KeywordPair{keywords[0], keywords[1]}, nil
}
