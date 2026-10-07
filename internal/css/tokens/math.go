package tokens

import (
	"math"
	"strconv"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// Style gives the context that relative units of math functions need.
// ResolveMath is given nil when only absolute values can be resolved.
type Style interface {
	// ToPixels converts a font or viewport relative length of the given
	// unit, in the context of the named property, to CSS pixels.
	ToPixels(value float64, unit, property string) float64
}

// Constants of math functions (CSS Values 4 §10.7.1).
var (
	numberE       = newNumber(math.E, "e")
	numberPi      = newNumber(math.Pi, "π")
	plusInfinity  = newNumber(math.Inf(1), "∞")
	minusInfinity = newNumber(math.Inf(-1), "-∞")
	notANumber    = newNumber(math.NaN(), "NaN")
)

func newNumber(v float64, repr string) *syntax.NumberToken {
	return syntax.NewNumberToken(0, 0, syntax.Numeric{Value: v, Representation: repr})
}

// formatNumber writes v like a CSS number representation.
func formatNumber(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	case math.IsNaN(v):
		return "nan"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// tokenize makes a token of the given unit: a number for the empty unit, a
// percentage for "%" and a dimension otherwise.
func tokenize(v float64, unit string) syntax.Node {
	n := syntax.Numeric{Value: v}
	if v == math.Trunc(v) && !math.IsInf(v, 0) {
		n.IsInteger = true
		n.IntValue = int(v)
		n.Representation = strconv.Itoa(n.IntValue)
	} else {
		n.Representation = formatNumber(v)
	}
	switch unit {
	case "":
		return syntax.NewNumberToken(0, 0, n)
	case "%":
		return syntax.NewPercentageToken(0, 0, n)
	}
	return syntax.NewDimensionToken(0, 0, n, unit)
}

// tokenizeLike makes a token of the same kind and unit as like, with a new
// value.
func tokenizeLike(like syntax.Node, v float64) syntax.Node {
	switch t := like.(type) {
	case *syntax.PercentageToken:
		return tokenize(v, "%")
	case *syntax.DimensionToken:
		return tokenize(v, t.LowerUnit)
	}
	return tokenize(v, "")
}

func valueOf(token syntax.Node) float64 {
	n, _ := numeric(token)
	if n == nil {
		return math.NaN()
	}
	return n.Value
}

// unitOf returns the lowercase unit of a dimension, "%" for a percentage and
// the empty string otherwise.
func unitOf(token syntax.Node) string {
	switch t := token.(type) {
	case *syntax.DimensionToken:
		return t.LowerUnit
	case *syntax.PercentageToken:
		return "%"
	}
	return ""
}

func toPixels(token syntax.Node, style Style, property string) (float64, bool) {
	v, unit := valueOf(token), unitOf(token)
	switch {
	case v == 0:
		return 0, true
	case unit == "px":
		return v, true
	case units.IsAbsolute(unit):
		p, _ := units.AbsoluteToPixels(v, unit)
		return p, true
	case units.IsRelative(unit) && style != nil:
		return style.ToPixels(v, unit, property), true
	}
	return 0, false
}

func isLiteralIn(token syntax.Node, chars string) bool {
	l, ok := token.(*syntax.LiteralToken)
	return ok && l.Value != "" && strings.Contains(chars, l.Value)
}

// group is an operand or an operator of a calc() sum or product.
type group struct {
	operator string
	tokens   []syntax.Node
}

func resolveCalcSum(style Style, tokens []syntax.Node, property string, referTo *float64) (syntax.Node, error) {
	groups := []group{{}}
	for _, token := range tokens {
		switch t := token.(type) {
		case *syntax.LiteralToken:
			if isLiteralIn(t, "+-") {
				groups = append(groups, group{operator: t.Value}, group{})
				continue
			}
		case *syntax.ParenthesesBlock:
			result, err := resolveCalcSum(style, RemoveWhitespace(t.Content), property, referTo)
			if err != nil || result == nil {
				return nil, err
			}
			last := &groups[len(groups)-1]
			last.tokens = append(last.tokens, result)
			continue
		}
		last := &groups[len(groups)-1]
		last.tokens = append(last.tokens, token)
	}

	value, sign, unit := 0.0, "+", ""
	unitSet := false
	var pending error
	for _, g := range groups {
		if sign == "" {
			sign = g.operator
			continue
		}
		product, err := resolveCalcProduct(style, g.tokens, property, referTo)
		if err == ErrRelativeLengthInMath {
			// Assume pixels and go on, to find out whether
			// ErrPercentageInMath has to be reported first.
			if unitSet && unit == "%" {
				return nil, ErrPercentageInMath
			}
			pending = err
			unit, unitSet = "px", true
			sign = ""
			continue
		}
		if err != nil || product == nil {
			return nil, err
		}
		switch product.(type) {
		case *syntax.DimensionToken:
			switch {
			case !unitSet:
				unit, unitSet = unitOf(product), true
			case unit == "%":
				return nil, ErrPercentageInMath
			case unit != unitOf(product):
				return nil, nil
			}
		case *syntax.PercentageToken:
			if referTo != nil {
				product = tokenize(valueOf(product)/100**referTo, "px")
				unit, unitSet = "px", true
			} else if !unitSet || unit == "%" {
				unit, unitSet = "%", true
			} else {
				return nil, ErrPercentageInMath
			}
		}
		if sign == "+" {
			value += valueOf(product)
		} else {
			value -= valueOf(product)
		}
		sign = ""
	}

	// Reported only if ErrPercentageInMath did not come first.
	if pending != nil {
		return nil, pending
	}
	return tokenize(value, unit), nil
}

func resolveCalcProduct(style Style, tokens []syntax.Node, property string, referTo *float64) (syntax.Node, error) {
	groups := []group{{}}
	for _, token := range tokens {
		last := &groups[len(groups)-1]
		switch t := token.(type) {
		case *syntax.LiteralToken:
			if isLiteralIn(t, "*/") {
				groups = append(groups, group{operator: t.Value}, group{})
				continue
			}
			return nil, nil
		case *syntax.NumberToken:
			last.tokens = append(last.tokens, t)
		case *syntax.DimensionToken:
			switch {
			case units.IsLength(t.LowerUnit):
				if style == nil && units.IsRelative(t.LowerUnit) {
					return nil, ErrRelativeLengthInMath
				}
				pixels, ok := toPixels(t, style, property)
				if !ok {
					return nil, nil
				}
				last.tokens = append(last.tokens, tokenize(pixels, "px"))
			case units.IsAngle(t.LowerUnit):
				radians, _ := units.ToRadians(t.Value, t.LowerUnit)
				last.tokens = append(last.tokens, tokenize(radians, "rad"))
			default:
				return nil, nil
			}
		case *syntax.PercentageToken:
			last.tokens = append(last.tokens, tokenize(t.Value, "%"))
		case *syntax.IdentToken:
			last.tokens = append(last.tokens, t)
		default:
			return nil, nil
		}
	}

	value, sign, unit := 1.0, "*", ""
	for _, g := range groups {
		if sign == "" {
			sign = g.operator
			continue
		}
		calc := resolveCalcValue(g.tokens)
		if calc == nil {
			return nil, nil
		}
		switch calc.(type) {
		case *syntax.DimensionToken:
			if unit != "" && unit != "%" {
				return nil, nil
			}
			unit = unitOf(calc)
		case *syntax.PercentageToken:
			if unit == "" {
				unit = "%"
			}
		}
		switch {
		case sign == "*":
			value *= valueOf(calc)
		case valueOf(calc) == 0:
			// Division by zero gives infinity or NaN, as IEEE 754 and
			// CSS Values 4 require, instead of an error.
			if value == 0 {
				value = math.NaN()
			} else {
				value = math.Copysign(math.Inf(1), value)
			}
		default:
			value /= valueOf(calc)
		}
		sign = ""
	}
	return tokenize(value, unit), nil
}

func resolveCalcValue(tokens []syntax.Node) syntax.Node {
	if len(tokens) != 1 {
		return nil
	}
	switch t := tokens[0].(type) {
	case *syntax.NumberToken, *syntax.DimensionToken, *syntax.PercentageToken:
		return t
	case *syntax.IdentToken:
		switch t.LowerValue {
		case "e":
			return numberE
		case "pi":
			return numberPi
		case "infinity":
			return plusInfinity
		case "-infinity":
			return minusInfinity
		case "nan":
			return notANumber
		}
	}
	return nil
}

// ResolveMath resolves the math functions in token (CSS Values 4 §10). It
// returns nil when token holds no math function or one that cannot be
// resolved, and otherwise a number, percentage or dimension token, or a
// function whose arguments were resolved.
//
// With a nil style, ErrRelativeLengthInMath is returned for font and
// viewport units. With a nil referTo, ErrPercentageInMath is returned when a
// percentage is combined with lengths. The latter is reported first.
func ResolveMath(token syntax.Node, style Style, property string, referTo *float64) (syntax.Node, error) {
	if !CheckMath(token) {
		return nil, nil
	}
	fnToken := token.(*syntax.FunctionBlock)
	f := NewFunction(token)

	var args [][]syntax.Node
	for _, part := range f.SplitCommaGroups(false) {
		resolvedPart := make([]syntax.Node, 0, len(part))
		for _, arg := range part {
			if CheckMath(arg) {
				resolved, err := ResolveMath(arg, style, property, referTo)
				if err != nil || resolved == nil {
					return nil, err
				}
				arg = resolved
			}
			resolvedPart = append(resolvedPart, arg)
		}
		args = append(args, resolvedPart)
	}

	sum := func(tokens []syntax.Node) (syntax.Node, error) {
		return resolveCalcSum(style, tokens, property, referTo)
	}
	// resolveAll resolves every argument as a sum; it returns nil when one
	// of them cannot be resolved.
	resolveAll := func() ([]syntax.Node, error) {
		out := make([]syntax.Node, len(args))
		for i, a := range args {
			t, err := sum(a)
			if err != nil || t == nil {
				return nil, err
			}
			out[i] = t
		}
		return out, nil
	}
	isNumber := func(t syntax.Node) bool { _, ok := t.(*syntax.NumberToken); return ok }

	switch f.Name {
	case "calc":
		return sum(args[0])

	case "min", "max":
		var target, targetToken syntax.Node
		unit := ""
		unitSet := false
		for _, tokens := range args {
			t, err := sum(tokens)
			if err != nil || t == nil {
				return nil, err
			}
			var value syntax.Node
			switch tt := t.(type) {
			case *syntax.PercentageToken:
				if referTo == nil {
					if unitSet && (unit == "px" || unit == "") {
						return nil, ErrPercentageInMath
					}
					unit, unitSet = "%", true
					value = t
				} else {
					unit, unitSet = "px", true
					t = tokenize(tt.Value/100**referTo, "px")
					value = t
				}
			case *syntax.NumberToken:
				if unitSet && unit == "%" {
					return nil, ErrPercentageInMath
				} else if unitSet && unit == "px" {
					return nil, nil
				}
				unit, unitSet = "", true
				value = tokenize(tt.Value, "px")
			default:
				if unitSet && unit == "%" {
					return nil, ErrPercentageInMath
				} else if unitSet && unit == "" {
					return nil, nil
				}
				unit, unitSet = "px", true
				pixels, ok := toPixels(t, style, property)
				if !ok {
					return nil, nil
				}
				value = tokenize(pixels, "px")
			}
			if target == nil ||
				(f.Name == "min" && valueOf(value) < valueOf(target)) ||
				(f.Name == "max" && valueOf(value) > valueOf(target)) {
				target, targetToken = value, t
			}
		}
		return targetToken, nil

	case "round":
		strategy, multiple := "nearest", 1.0
		var number syntax.Node
		var err error
		switch len(args) {
		case 1:
			number, err = sum(args[0])
		case 2:
			if len(args[0]) == 1 && isRoundingStrategy(args[0][0]) {
				strategy = identValue(args[0][0])
				number, err = sum(args[1])
				break
			}
			var m syntax.Node
			number, err = sum(args[0])
			if err == nil && number != nil {
				m, err = sum(args[1])
			}
			if err != nil || number == nil || m == nil {
				return nil, err
			}
			if number.Type() != m.Type() {
				return nil, nil
			}
			multiple = valueOf(m)
		case 3:
			strategy = identValue(args[0][0])
			var m syntax.Node
			number, err = sum(args[1])
			if err == nil && number != nil {
				m, err = sum(args[2])
			}
			if err != nil || number == nil || m == nil {
				return nil, err
			}
			if number.Type() != m.Type() {
				return nil, nil
			}
			multiple = valueOf(m)
		}
		if err != nil || number == nil {
			return nil, err
		}
		var fn func(float64) float64
		switch strategy {
		case "nearest":
			fn = func(x float64) float64 { return math.Floor(x + 0.5) }
		case "up":
			fn = math.Ceil
		case "down":
			fn = math.Floor
		case "to-zero":
			if valueOf(number) > 0 {
				fn = math.Floor
			} else {
				fn = math.Ceil
			}
		default:
			return nil, nil
		}
		return tokenizeLike(number, fn(valueOf(number)/multiple)*multiple), nil

	case "mod", "rem":
		ts, err := resolveAll()
		if err != nil || ts == nil {
			return nil, err
		}
		number, parameter := ts[0], ts[1]
		if number.Type() != parameter.Type() {
			return nil, nil
		}
		n, p := valueOf(number), valueOf(parameter)
		// The result of a modulo takes the sign of the divisor.
		value := n - p*math.Floor(n/p)
		if f.Name == "rem" && n*p < 0 {
			value += math.Abs(p)
		}
		return tokenizeLike(number, value), nil

	case "sin", "cos", "tan":
		number, err := sum(args[0])
		if err != nil || number == nil {
			return nil, err
		}
		var angle float64
		if isNumber(number) {
			angle = valueOf(number)
		} else {
			var ok bool
			if angle, ok = GetAngle(number); !ok {
				return nil, nil
			}
		}
		switch f.Name {
		case "sin":
			return tokenize(math.Sin(angle), ""), nil
		case "cos":
			return tokenize(math.Cos(angle), ""), nil
		}
		return tokenize(math.Tan(angle), ""), nil

	case "asin", "acos", "atan":
		number, err := sum(args[0])
		if err != nil || number == nil || !isNumber(number) {
			return nil, err
		}
		v := valueOf(number)
		var r float64
		switch f.Name {
		case "asin":
			r = math.Asin(v)
		case "acos":
			r = math.Acos(v)
		default:
			r = math.Atan(v)
		}
		if math.IsNaN(r) {
			return nil, nil
		}
		return tokenize(r, "rad"), nil

	case "atan2":
		ts, err := resolveAll()
		if err != nil || ts == nil || !isNumber(ts[0]) || !isNumber(ts[1]) {
			return nil, err
		}
		return tokenize(math.Atan2(valueOf(ts[0]), valueOf(ts[1])), "rad"), nil

	case "clamp":
		var pixelsList []syntax.Node
		unit := ""
		unitSet := false
		for _, tokens := range args {
			t, err := sum(tokens)
			if err != nil || t == nil {
				return nil, err
			}
			var value syntax.Node
			if p, ok := t.(*syntax.PercentageToken); ok {
				if referTo == nil {
					if unitSet && unit == "px" {
						return nil, ErrPercentageInMath
					}
					unit, unitSet = "%", true
					value = t
				} else {
					unit, unitSet = "px", true
					value = tokenize(p.Value/100**referTo, "px")
				}
			} else {
				if unitSet && unit == "%" {
					return nil, ErrPercentageInMath
				}
				unit, unitSet = "px", true
				pixels, ok := toPixels(t, style, property)
				if !ok {
					return nil, nil
				}
				value = tokenize(pixels, "px")
			}
			pixelsList = append(pixelsList, value)
		}
		minToken, t, maxToken := pixelsList[0], pixelsList[1], pixelsList[2]
		if valueOf(t) < valueOf(minToken) {
			t = minToken
		}
		if valueOf(t) > valueOf(maxToken) {
			t = maxToken
		}
		return t, nil

	case "pow":
		ts, err := resolveAll()
		if err != nil || ts == nil || !isNumber(ts[0]) || !isNumber(ts[1]) {
			return nil, err
		}
		return tokenizeLike(ts[0], math.Pow(valueOf(ts[0]), valueOf(ts[1]))), nil

	case "sqrt":
		number, err := sum(args[0])
		if err != nil || number == nil || !isNumber(number) {
			return nil, err
		}
		return tokenizeLike(number, math.Sqrt(valueOf(number))), nil

	case "hypot":
		ts, err := resolveAll()
		if err != nil || ts == nil {
			return nil, err
		}
		h := 0.0
		for _, t := range ts {
			h = math.Hypot(h, valueOf(t))
		}
		return tokenizeLike(ts[0], h), nil

	case "log":
		number, err := sum(args[0])
		if err != nil || number == nil || !isNumber(number) {
			return nil, err
		}
		base := math.E
		if len(args) == 2 {
			b, err := sum(args[1])
			if err != nil || b == nil || !isNumber(b) {
				return nil, err
			}
			base = valueOf(b)
		}
		return tokenizeLike(number, math.Log(valueOf(number))/math.Log(base)), nil

	case "exp":
		number, err := sum(args[0])
		if err != nil || number == nil || !isNumber(number) {
			return nil, err
		}
		return tokenizeLike(number, math.Exp(valueOf(number))), nil

	case "abs":
		number, err := sum(args[0])
		if err != nil || number == nil {
			return nil, err
		}
		return tokenizeLike(number, math.Abs(valueOf(number))), nil

	case "sign":
		number, err := sum(args[0])
		if err != nil || number == nil {
			return nil, err
		}
		v := valueOf(number)
		switch {
		case v == 0:
			return tokenize(0, ""), nil
		case v > 0:
			return tokenize(1, ""), nil
		}
		return tokenize(-1, ""), nil
	}

	// Not a math function itself: resolve the ones in its arguments.
	arguments := make([]syntax.Node, 0, len(fnToken.Arguments))
	for _, argument := range fnToken.Arguments {
		if _, ok := argument.(*syntax.FunctionBlock); ok {
			result, err := ResolveMath(argument, style, property, referTo)
			if err != nil || result == nil {
				return nil, err
			}
			argument = result
		}
		arguments = append(arguments, argument)
	}
	resolved := syntax.NewFunctionBlock(fnToken.Line, fnToken.Column, fnToken.Name, arguments)
	if result, err := ResolveMath(resolved, style, property, referTo); err != nil || result != nil {
		return result, err
	}
	return resolved, nil
}

func isRoundingStrategy(token syntax.Node) bool {
	switch identValue(token) {
	case "nearest", "up", "down", "to-zero":
		return true
	}
	return false
}

func identValue(token syntax.Node) string {
	if ident, ok := token.(*syntax.IdentToken); ok {
		return ident.Value
	}
	return ""
}
