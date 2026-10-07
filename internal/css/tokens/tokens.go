package tokens

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// Percentages used as the position of keywords such as top or center.
var (
	zeroPercent    = props.Dimension{Value: 0, Unit: "%"}
	fiftyPercent   = props.Dimension{Value: 50, Unit: "%"}
	hundredPercent = props.Dimension{Value: 100, Unit: "%"}
)

func percentLength(d props.Dimension) props.Length { return props.Length{Dimension: d} }

var positionKeywordPercentages = map[string]props.Length{
	"top":    percentLength(zeroPercent),
	"left":   percentLength(zeroPercent),
	"center": percentLength(fiftyPercent),
	"bottom": percentLength(hundredPercent),
	"right":  percentLength(hundredPercent),
}

// RemoveWhitespace removes the top-level whitespace and comments of a token
// list.
func RemoveWhitespace(tokens []syntax.Node) []syntax.Node {
	out := make([]syntax.Node, 0, len(tokens))
	for _, t := range tokens {
		if k := t.Type(); k != "whitespace" && k != "comment" {
			out = append(out, t)
		}
	}
	return out
}

// SplitOnComma splits a token list on its top-level commas. Commas inside
// functions and blocks are not splitting points. The result always has at
// least one, possibly empty, part.
func SplitOnComma(tokens []syntax.Node) [][]syntax.Node {
	parts := [][]syntax.Node{nil}
	for _, t := range tokens {
		if syntax.IsLiteral(t, ",") {
			parts = append(parts, nil)
		} else {
			parts[len(parts)-1] = append(parts[len(parts)-1], t)
		}
	}
	return parts
}

// GetKeyword returns the lowercase name of token if it is an identifier, and
// the empty string otherwise.
func GetKeyword(token syntax.Node) string {
	if ident, ok := token.(*syntax.IdentToken); ok {
		return ident.LowerValue
	}
	return ""
}

// GetCustomIdent returns the name of token, case preserved, if it is an
// identifier, and the empty string otherwise.
func GetCustomIdent(token syntax.Node) string {
	if ident, ok := token.(*syntax.IdentToken); ok {
		return ident.Value
	}
	return ""
}

// GetSingleKeyword returns the keyword of a one-token list, or the empty
// string.
func GetSingleKeyword(tokens []syntax.Node) string {
	if len(tokens) == 1 {
		return GetKeyword(tokens[0])
	}
	return ""
}

// numeric returns the numeric part of a number, percentage or dimension.
func numeric(token syntax.Node) (*syntax.Numeric, bool) {
	switch t := token.(type) {
	case *syntax.NumberToken:
		return &t.Numeric, true
	case *syntax.PercentageToken:
		return &t.Numeric, true
	case *syntax.DimensionToken:
		return &t.Numeric, true
	}
	return nil, false
}

// GetNumber parses a <number>, or an <integer> when integer is set. The
// result has an empty unit. Negative numbers are rejected unless negative is
// set, except for the result of a math function, which is clamped to zero.
func GetNumber(token syntax.Node, negative, integer bool) (props.Dimension, bool) {
	if CheckMath(token) {
		resolved, err := ResolveMath(token, nil, "", nil)
		if err != nil || resolved == nil {
			return props.Dimension{}, false
		}
		n, ok := resolved.(*syntax.NumberToken)
		if !ok {
			return props.Dimension{}, false
		}
		value := n.Value
		if !negative && value < 0 {
			value = 0
		}
		if integer {
			value = math.RoundToEven(value)
		}
		return props.Dimension{Value: value}, true
	}
	n, ok := token.(*syntax.NumberToken)
	if !ok {
		return props.Dimension{}, false
	}
	if integer {
		if n.IsInteger && (negative || n.IntValue >= 0) {
			return props.Dimension{Value: float64(n.IntValue)}, true
		}
	} else if negative || n.Value >= 0 {
		return props.Dimension{Value: n.Value}, true
	}
	return props.Dimension{}, false
}

// GetPercentage parses a <percentage>.
func GetPercentage(token syntax.Node, negative bool) (props.Dimension, bool) {
	if CheckMath(token) {
		resolved, err := ResolveMath(token, nil, "", nil)
		if err != nil {
			return props.Dimension{}, false
		}
		if resolved != nil {
			token = resolved
		}
		if p, ok := token.(*syntax.PercentageToken); ok && !negative && p.Value < 0 {
			token = tokenize(0, "%")
		}
	}
	if p, ok := token.(*syntax.PercentageToken); ok && (negative || p.Value >= 0) {
		return props.Dimension{Value: p.Value, Unit: "%"}, true
	}
	return props.Dimension{}, false
}

// GetLength parses a <length>, or a <length-percentage> when percentage is
// set. Negative values are rejected unless negative is set. A math function
// that depends on the style, because it has relative units, is returned
// unresolved.
func GetLength(token syntax.Node, negative, percentage bool) (props.Length, bool) {
	if CheckMath(token) {
		resolved, err := ResolveMath(token, nil, "", nil)
		switch err {
		case nil:
			if resolved != nil {
				token = resolved
			}
			if !negative {
				switch t := token.(type) {
				case *syntax.DimensionToken, *syntax.PercentageToken:
					// Clamped like max(0, v), which also turns NaN into 0.
					v, _ := numeric(t)
					if !(v.Value > 0) {
						token = tokenizeLike(t, 0)
					}
				}
			}
		case ErrPercentageInMath:
			// Reported first so that properties without percentages
			// discard the value.
			if !percentage {
				return props.Length{}, false
			}
			return calcLength(token)
		default:
			return calcLength(token)
		}
	}
	switch t := token.(type) {
	case *syntax.PercentageToken:
		if percentage && (negative || t.Value >= 0) {
			return props.Length{Dimension: props.Dimension{Value: t.Value, Unit: "%"}}, true
		}
	case *syntax.DimensionToken:
		if units.IsLength(t.LowerUnit) && (negative || t.Value >= 0) {
			return props.Length{Dimension: props.Dimension{Value: t.Value, Unit: t.LowerUnit}}, true
		}
	case *syntax.NumberToken:
		if t.Value == 0 {
			return props.Length{Dimension: props.Dimension{}}, true
		}
	}
	return props.Length{}, false
}

func calcLength(token syntax.Node) (props.Length, bool) {
	fn, ok := token.(*syntax.FunctionBlock)
	if !ok {
		return props.Length{}, false
	}
	return props.Length{Calc: &props.Calc{Function: fn}}, true
}

// GetAngle parses an <angle> and returns it in radians.
func GetAngle(token syntax.Node) (float64, bool) {
	resolved, err := ResolveMath(token, nil, "", nil)
	if err != nil {
		return 0, false
	}
	if resolved != nil {
		token = resolved
	}
	switch t := token.(type) {
	case *syntax.NumberToken:
		// Legacy syntax for zero (CSS Values 4 §6.1).
		if t.Value == 0 {
			return 0, true
		}
	case *syntax.DimensionToken:
		return units.ToRadians(t.Value, t.LowerUnit)
	}
	return 0, false
}

// GetResolution parses a <resolution> and returns it in dots per pixel.
func GetResolution(token syntax.Node) (float64, bool) {
	resolved, err := ResolveMath(token, nil, "", nil)
	if err != nil {
		return 0, false
	}
	if resolved != nil {
		token = resolved
	}
	if t, ok := token.(*syntax.DimensionToken); ok {
		if factor, ok := units.ResolutionToDppx[t.LowerUnit]; ok {
			return t.Value * factor, true
		}
	}
	return 0, false
}

// Parse2DPosition parses the syntax shared by background-position and
// transform-origin: one or two keywords, lengths or percentages, in
// horizontal then vertical order (CSS Backgrounds 3 §3.6).
func Parse2DPosition(tokens []syntax.Node) (x, y props.Length, ok bool) {
	if len(tokens) == 1 {
		tokens = []syntax.Node{tokens[0], syntax.NewIdentToken(0, 0, "center")}
	} else if len(tokens) != 2 {
		return
	}

	length1, ok1 := GetLength(tokens[0], true, true)
	length2, ok2 := GetLength(tokens[1], true, true)
	if ok1 && ok2 {
		return length1, length2, true
	}
	keyword1, keyword2 := GetKeyword(tokens[0]), GetKeyword(tokens[1])
	switch {
	case ok1 && isOneOf(keyword2, "top", "center", "bottom"):
		return length1, positionKeywordPercentages[keyword2], true
	case ok2 && isOneOf(keyword1, "left", "center", "right"):
		return positionKeywordPercentages[keyword1], length2, true
	case isOneOf(keyword1, "left", "center", "right") && isOneOf(keyword2, "top", "center", "bottom"):
		return positionKeywordPercentages[keyword1], positionKeywordPercentages[keyword2], true
	case isOneOf(keyword1, "top", "center", "bottom") && isOneOf(keyword2, "left", "center", "right"):
		// Swapped: they have to be in horizontal, vertical order.
		return positionKeywordPercentages[keyword2], positionKeywordPercentages[keyword1], true
	}
	return
}

func isOneOf(s string, options ...string) bool {
	for _, o := range options {
		if s == o {
			return true
		}
	}
	return false
}

// ParsePosition parses background-position and object-position (CSS
// Backgrounds 3 §3.6, CSS Images 3 §5.3).
func ParsePosition(tokens []syntax.Node) (props.Position, bool) {
	if x, y, ok := Parse2DPosition(tokens); ok {
		return props.Position{OriginX: "left", X: x, OriginY: "top", Y: y}, true
	}

	if len(tokens) == 4 {
		keyword1 := GetKeyword(tokens[0])
		keyword2 := GetKeyword(tokens[2])
		length1, ok1 := GetLength(tokens[1], true, true)
		length2, ok2 := GetLength(tokens[3], true, true)
		if ok1 && ok2 {
			if isOneOf(keyword1, "left", "right") && isOneOf(keyword2, "top", "bottom") {
				return props.Position{OriginX: props.Keyword(keyword1), X: length1, OriginY: props.Keyword(keyword2), Y: length2}, true
			}
			if isOneOf(keyword2, "left", "right") && isOneOf(keyword1, "top", "bottom") {
				return props.Position{OriginX: props.Keyword(keyword2), X: length2, OriginY: props.Keyword(keyword1), Y: length1}, true
			}
		}
	}

	if len(tokens) == 3 {
		var keyword, otherKeyword string
		length, ok := GetLength(tokens[2], true, true)
		if ok {
			keyword = GetKeyword(tokens[1])
			otherKeyword = GetKeyword(tokens[0])
		} else {
			length, ok = GetLength(tokens[1], true, true)
			otherKeyword = GetKeyword(tokens[2])
			keyword = GetKeyword(tokens[0])
		}
		if ok {
			switch {
			case otherKeyword == "center":
				if isOneOf(keyword, "top", "bottom") {
					return props.Position{OriginX: "left", X: percentLength(fiftyPercent), OriginY: props.Keyword(keyword), Y: length}, true
				}
				if isOneOf(keyword, "left", "right") {
					return props.Position{OriginX: props.Keyword(keyword), X: length, OriginY: "top", Y: percentLength(fiftyPercent)}, true
				}
			case isOneOf(keyword, "left", "right") && isOneOf(otherKeyword, "top", "bottom"):
				return props.Position{OriginX: props.Keyword(keyword), X: length, OriginY: props.Keyword(otherKeyword), Y: percentLength(zeroPercent)}, true
			case isOneOf(keyword, "top", "bottom") && isOneOf(otherKeyword, "left", "right"):
				return props.Position{OriginX: props.Keyword(otherKeyword), X: percentLength(zeroPercent), OriginY: props.Keyword(keyword), Y: length}, true
			}
		}
	}
	return props.Position{}, false
}

// GetQuote parses an open-quote, close-quote, no-open-quote or
// no-close-quote keyword.
func GetQuote(token syntax.Node) (string, bool) {
	keyword := GetKeyword(token)
	if isOneOf(keyword, "open-quote", "close-quote", "no-open-quote", "no-close-quote") {
		return keyword, true
	}
	return "", false
}
