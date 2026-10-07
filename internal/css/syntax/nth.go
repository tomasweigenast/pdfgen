package syntax

import (
	"strconv"
	"strings"
)

// ParseNth parses an <An+B> expression (CSS Syntax 3 §6), as found in
// :nth-child() and related pseudo-classes. ok is false when the input is
// invalid.
func ParseNth[T Input](input T) (a, b int, ok bool) {
	it := toIter(input, true)
	tok := it.nextSignificant()
	if tok == nil {
		return 0, 0, false
	}
	switch t := tok.(type) {
	case *NumberToken:
		if t.IsInteger {
			return nthEnd(it, 0, t.IntValue)
		}
	case *DimensionToken:
		if !t.IsInteger {
			break
		}
		switch unit := t.LowerUnit; unit {
		case "n":
			return nthB(it, t.IntValue)
		case "n-":
			return nthSignlessB(it, t.IntValue, -1)
		default:
			if v, ok := nDashDigits(unit); ok {
				return nthEnd(it, t.IntValue, v)
			}
		}
	case *IdentToken:
		switch ident := t.LowerValue; ident {
		case "even":
			return nthEnd(it, 2, 0)
		case "odd":
			return nthEnd(it, 2, 1)
		case "n":
			return nthB(it, 1)
		case "-n":
			return nthB(it, -1)
		case "n-":
			return nthSignlessB(it, 1, -1)
		case "-n-":
			return nthSignlessB(it, -1, -1)
		default:
			if strings.HasPrefix(ident, "-") {
				if v, ok := nDashDigits(ident[1:]); ok {
					return nthEnd(it, -1, v)
				}
			} else if v, ok := nDashDigits(ident); ok {
				return nthEnd(it, 1, v)
			}
		}
	case *LiteralToken:
		if t.Value != "+" {
			break
		}
		// Whitespace after an initial '+' is invalid.
		next, more := it.next()
		id, isIdent := next.(*IdentToken)
		if !more || !isIdent {
			break
		}
		switch ident := id.LowerValue; ident {
		case "n":
			return nthB(it, 1)
		case "n-":
			return nthSignlessB(it, 1, -1)
		default:
			if v, ok := nDashDigits(ident); ok {
				return nthEnd(it, 1, v)
			}
		}
	}
	return 0, 0, false
}

// nDashDigits matches ^n(-[0-9]+)$ and returns the signed number.
func nDashDigits(s string) (int, bool) {
	if len(s) < 3 || s[0] != 'n' || s[1] != '-' {
		return 0, false
	}
	for _, c := range s[2:] {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	v, err := strconv.Atoi(s[1:])
	return v, err == nil
}

func nthB(it *tokenIter, a int) (int, int, bool) {
	tok := it.nextSignificant()
	switch t := tok.(type) {
	case nil:
		return a, 0, true
	case *LiteralToken:
		switch t.Value {
		case "+":
			return nthSignlessB(it, a, 1)
		case "-":
			return nthSignlessB(it, a, -1)
		}
	case *NumberToken:
		if t.IsInteger && (t.Representation[0] == '-' || t.Representation[0] == '+') {
			return nthEnd(it, a, t.IntValue)
		}
	}
	return 0, 0, false
}

func nthSignlessB(it *tokenIter, a, bSign int) (int, int, bool) {
	t, ok := it.nextSignificant().(*NumberToken)
	if ok && t.IsInteger && t.Representation[0] != '-' && t.Representation[0] != '+' {
		return nthEnd(it, a, bSign*t.IntValue)
	}
	return 0, 0, false
}

func nthEnd(it *tokenIter, a, b int) (int, int, bool) {
	if it.nextSignificant() == nil {
		return a, b, true
	}
	return 0, 0, false
}
