package units

import (
	"math"
	"strings"
)

// AngleToRadians gives the size of one unit of each angle unit in radians
// (CSS Values 4 §6.1).
var AngleToRadians = map[string]float64{
	"rad":  1,
	"turn": 2 * math.Pi,
	"deg":  math.Pi / 180,
	"grad": math.Pi / 200,
}

// LengthsToPixels gives the size of one unit of each absolute length unit in
// CSS pixels (CSS 2.1 §4.3.2).
var LengthsToPixels = map[string]float64{
	"px": 1,
	"pt": quotient(1, 0.75),
	"pc": 16,
	"in": 96,
	"cm": quotient(96, 2.54),
	"mm": quotient(96, 25.4),
	"q":  quotient(quotient(96, 25.4), 4),
}

// quotient divides at run time. A constant expression such as 96 / 25.4 is
// evaluated exactly and rounded once, which can land one unit in the last
// place away from the rounded divisions every length computation starts
// from, and lengths that sum to a container width would then stop fitting.
func quotient(a, b float64) float64 { return a / b }

// ResolutionToDppx gives the size of one unit of each resolution unit in dots
// per pixel (CSS Values 4 §6.4).
var ResolutionToDppx = map[string]float64{
	"dppx": 1,
	"x":    1,
	"dpi":  1 / LengthsToPixels["in"],
	"dpcm": 1 / LengthsToPixels["cm"],
}

var (
	fontUnits = setOf(
		"em", "ex", "cap", "ch", "ic", "lh",
		"rem", "rex", "rcap", "rch", "ric", "rlh",
	)
	viewportUnits = setOf(
		"vw", "vh", "vi", "vb", "vmin", "vmax",
		"lvw", "lvh", "lvi", "lvb", "lvmin", "lvmax",
		"svw", "svh", "svi", "svb", "svmin", "svmax",
		"dvw", "dvh", "dvi", "dvb", "dvmin", "dvmax",
		"pvw", "pvh", "pvi", "pvb", "pvmin", "pvmax",
	)
)

func setOf(units ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(units))
	for _, u := range units {
		m[u] = struct{}{}
	}
	return m
}

func in[V any](m map[string]V, unit string) bool {
	_, ok := m[strings.ToLower(unit)]
	return ok
}

// IsAbsolute reports whether unit is an absolute length unit.
func IsAbsolute(unit string) bool { return in(LengthsToPixels, unit) }

// IsFontRelative reports whether unit is relative to font metrics.
func IsFontRelative(unit string) bool { return in(fontUnits, unit) }

// IsViewportRelative reports whether unit is relative to the page size.
func IsViewportRelative(unit string) bool { return in(viewportUnits, unit) }

// IsRelative reports whether unit is a font or viewport relative length unit.
func IsRelative(unit string) bool { return IsFontRelative(unit) || IsViewportRelative(unit) }

// IsLength reports whether unit is any length unit.
func IsLength(unit string) bool { return IsAbsolute(unit) || IsRelative(unit) }

// IsAngle reports whether unit is an angle unit.
func IsAngle(unit string) bool { return in(AngleToRadians, unit) }

// ToRadians converts an angle in the given unit to radians. The boolean is
// false when unit is not an angle unit.
func ToRadians(value float64, unit string) (float64, bool) {
	f, ok := AngleToRadians[strings.ToLower(unit)]
	return value * f, ok
}

// AbsoluteToPixels converts an absolute length to CSS pixels. The boolean is
// false when unit is not an absolute length unit.
func AbsoluteToPixels(value float64, unit string) (float64, bool) {
	f, ok := LengthsToPixels[strings.ToLower(unit)]
	return value * f, ok
}
