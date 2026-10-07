package html

import (
	"math"
	"strconv"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// asciiWhitespace is the set of ASCII whitespace of the Infra standard
// (https://infra.spec.whatwg.org/#ascii-whitespace).
const asciiWhitespace = " \t\n\f\r"

func isASCIIWhitespace(c byte) bool { return strings.IndexByte(asciiWhitespace, c) >= 0 }

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

// ParseURL parses a valid URL potentially surrounded by spaces, as found in
// attributes such as src and href (HTML Living Standard §2.4.1). Tabs and
// newlines inside the value are dropped. ok is false when the value is
// empty, or when it is relative, there is no baseURL and allowRelative is
// not set.
func ParseURL(s, baseURL string, allowRelative bool) (url string, ok bool) {
	s = strings.Trim(s, asciiWhitespace)
	s = strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(s)
	if s == "" {
		return "", false
	}
	return urls.JoinChecked(baseURL, s, allowRelative, s)
}

// ParseInteger parses an integer attribute value with the rules of HTML
// Living Standard §2.3.4.1: leading whitespace, an optional sign and digits;
// anything after the digits is ignored. Values beyond the range of int32 are
// clamped.
func ParseInteger(s string) (n int, ok bool) {
	i := 0
	for i < len(s) && isASCIIWhitespace(s[i]) {
		i++
	}
	negative := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		negative = s[i] == '-'
		i++
	}
	start := i
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i == start {
		return 0, false
	}
	v, err := strconv.ParseInt(s[start:i], 10, 64)
	if err != nil || v > math.MaxInt32 {
		v = math.MaxInt32
	}
	if negative {
		v = -v
	}
	return int(v), true
}

// ParseNonNegativeInteger is like ParseInteger but fails for negative
// values (HTML Living Standard §2.3.4.2). The sign of zero is not
// significant.
func ParseNonNegativeInteger(s string) (n int, ok bool) {
	n, ok = ParseInteger(s)
	if !ok || n < 0 {
		return 0, false
	}
	return n, true
}

// Dimension is a length or percentage attribute value.
type Dimension struct {
	Value float64
	// Unit is "px" or "%".
	Unit string
}

// ParseDimensionValue parses the digits, optional fraction and optional
// percent sign of a width or height attribute (HTML Living Standard
// §2.3.4.4, restricted to non-negative values).
func ParseDimensionValue(s string) (d Dimension, ok bool) {
	i := 0
	for i < len(s) && isASCIIWhitespace(s[i]) {
		i++
	}
	start := i
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i == start {
		return Dimension{}, false
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && isDigit(s[i]) {
			i++
		}
	}
	v, err := strconv.ParseFloat(s[start:i], 64)
	if err != nil {
		return Dimension{}, false
	}
	d = Dimension{Value: v, Unit: "px"}
	if i < len(s) && s[i] == '%' {
		d.Unit = "%"
	}
	return d, true
}

// String formats the dimension as CSS, the value always carrying a decimal
// point or exponent.
func (d Dimension) String() string { return formatFloat(d.Value) + d.Unit }

func formatFloat(v float64) string {
	if v != 0 && (math.Abs(v) < 1e-4 || math.Abs(v) >= 1e16) {
		return strconv.FormatFloat(v, 'e', -1, 64)
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// ParseLegacyColor parses a color attribute such as bgcolor into a CSS
// "#rrggbb" string. The keywords transparent and currentcolor, and
// anything that is not a CSS color, are rejected.
func ParseLegacyColor(s string) (color string, ok bool) {
	s = strings.Trim(s, asciiWhitespace)
	if lower := strings.ToLower(s); lower == "transparent" || lower == "currentcolor" {
		return "", false
	}
	c, _, ok := syntax.ParseColor3String(s)
	if !ok {
		return "", false
	}
	channel := func(v float64) int { return int(math.RoundToEven(max(0, min(1, v)) * 255)) }
	return "#" + hex2(channel(c.R)) + hex2(channel(c.G)) + hex2(channel(c.B)), true
}

func hex2(n int) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[n>>4], digits[n&15]})
}

// MapToPixelLength maps an attribute value to a CSS length in pixels, such
// as "12px". The fractional part and the unit of the value are ignored.
func MapToPixelLength(s string) (length string, ok bool) {
	n, ok := ParseNonNegativeInteger(s)
	if !ok {
		return "", false
	}
	return strconv.Itoa(n) + "px", true
}

// MapToDimensionProperty maps an attribute value to a CSS length or
// percentage.
func MapToDimensionProperty(s string) (value string, ok bool) {
	d, ok := ParseDimensionValue(s)
	if !ok {
		return "", false
	}
	return d.String(), true
}

// MapToDimensionPropertyIgnoringZero is like MapToDimensionProperty but
// rejects zero, which presentational attributes such as width treat as
// unset.
func MapToDimensionPropertyIgnoringZero(s string) (value string, ok bool) {
	d, ok := ParseDimensionValue(s)
	if !ok || d.Value == 0 {
		return "", false
	}
	return d.String(), true
}
