package computed

import (
	"errors"
	"math"
	"strconv"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// mathLengths returns the lengths of a value that is a length or a fixed
// list of lengths, and whether one of them is an unresolved math function.
func mathLengths(v props.Value) ([]props.Length, bool) {
	var lengths []props.Length
	switch v := v.(type) {
	case props.Length:
		lengths = []props.Length{v}
	case props.LengthPair:
		lengths = []props.Length{v.X, v.Y}
	case props.LengthQuad:
		lengths = v[:]
	case props.Clip:
		lengths = v
	default:
		return nil, false
	}
	for _, l := range lengths {
		if l.Calc != nil {
			return lengths, true
		}
	}
	return nil, false
}

// resolveMath resolves the math functions of a value that depend on the
// style, such as calc(1em + 2px), and validates the value again with the
// results (CSS Values 4 §10.12). Functions with percentages are kept for the
// layout, which knows what they refer to. When a function cannot be
// resolved, or its result is invalid for the property, the property gets its
// inherited or initial value with a warning. The boolean is false when the
// value has no math function to resolve.
func (s *Style) resolveMath(id props.PropID, value props.Value) (props.Value, bool) {
	lengths, ok := mathLengths(value)
	if !ok {
		return nil, false
	}
	name := id.Name()
	var last *syntax.FunctionBlock
	tokens := make([]syntax.Node, 0, len(lengths))
	invalid := false
	for _, l := range lengths {
		if l.Calc == nil {
			tokens = append(tokens, lengthToken(l))
			continue
		}
		last = l.Calc.Function
		resolved, err := tk.ResolveMath(last, s, name, nil)
		switch {
		case errors.Is(err, tk.ErrPercentageInMath):
			tokens = append(tokens, last)
		case err != nil || resolved == nil:
			invalid = true
		default:
			tokens = append(tokens, resolved)
		}
		if invalid {
			break
		}
	}
	if !invalid {
		if v, err := validation.Validate(tokens, name, s.baseURL); err == nil {
			return v, true
		}
	}

	line, column := last.Pos()
	logging.Warnf("Invalid math function at %d:%d: %s", line, column, last.Serialize())
	if id.Inherited() && s.parent != nil {
		v := s.parent.Get(id)
		s.store(id, v)
		return v, true
	}
	v := id.Initial()
	if !id.InitialNotComputed() {
		s.store(id, v)
	}
	return v, true
}

// resolveCalc resolves a math function to pixels, percentages referring to
// referTo. It returns 0 when the function cannot be resolved.
func (s *Style) resolveCalc(c *props.Calc, id props.PropID, referTo float64) float64 {
	resolved, err := tk.ResolveMath(c.Function, s, id.Name(), &referTo)
	if err != nil || resolved == nil {
		return 0
	}
	switch t := resolved.(type) {
	case *syntax.DimensionToken:
		return s.ToPixels(t.Value, t.LowerUnit, id.Name())
	case *syntax.PercentageToken:
		return t.Value * referTo / 100
	case *syntax.NumberToken:
		return t.Value
	}
	return 0
}

// lengthToken turns a length back into the token it was parsed from.
func lengthToken(l props.Length) syntax.Node {
	switch {
	case l.Keyword != "":
		return syntax.NewIdentToken(0, 0, string(l.Keyword))
	case l.Calc != nil:
		return l.Calc.Function
	}
	n := syntax.Numeric{Value: l.Dimension.Value, Representation: formatNumber(l.Dimension.Value)}
	if l.Dimension.Value == math.Trunc(l.Dimension.Value) && !math.IsInf(l.Dimension.Value, 0) {
		n.IsInteger, n.IntValue = true, int(l.Dimension.Value)
	}
	switch l.Dimension.Unit {
	case "":
		return syntax.NewNumberToken(0, 0, n)
	case "%":
		return syntax.NewPercentageToken(0, 0, n)
	}
	return syntax.NewDimensionToken(0, 0, n, l.Dimension.Unit)
}

func formatNumber(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
