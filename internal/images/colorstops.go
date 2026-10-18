package images

import (
	"math"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// maxHint is the exponent of a PDF interpolation function that stands for a
// hint at the end of its segment, kept small enough to stay finite.
const maxHint = 1 << 32

// resolveLength returns the absolute value of l in px, percentages being
// relative to ref. It reports false for keywords and for lengths that cannot
// be resolved at this stage, which gradients treat as missing.
func resolveLength(l props.Length, ref float64) (float64, bool) {
	if l.IsKeyword() {
		return 0, false
	}
	if l.Calc != nil {
		logging.Warnf("Unresolved calc() in a gradient, using the default instead")
		return 0, false
	}
	switch l.Dimension.Unit {
	case "px", "":
		return l.Dimension.Value, true
	case "%":
		return ref * l.Dimension.Value / 100, true
	}
	logging.Warnf("Unsupported unit %q in a gradient, using the default instead", l.Dimension.Unit)
	return 0, false
}

// processColorStops gives the color stops positions along a gradient vector
// of the given length (CSS Images 3 §3.4.3), and turns the hints into the
// exponents of PDF interpolation functions (ISO 32000-1 §7.10.3). Unset
// positions are nil. Positions are returned in px.
func processColorStops(vectorLength float64, positions []*props.Length, hints []props.Length) ([]float64, []float64) {
	resolved := make([]float64, len(positions))
	for i, p := range positions {
		resolved[i] = math.NaN()
		if p == nil {
			continue
		}
		if v, ok := resolveLength(*p, vectorLength); ok {
			resolved[i] = v
		}
	}
	hintFractions := make([]float64, len(hints))
	for i, h := range hints {
		v, ok := resolveLength(h, vectorLength)
		if !ok {
			v = vectorLength / 2
		}
		hintFractions[i] = v / vectorLength
	}

	if math.IsNaN(resolved[0]) {
		resolved[0] = 0
	}
	if last := len(resolved) - 1; math.IsNaN(resolved[last]) {
		resolved[last] = vectorLength
	}

	previous := resolved[0]
	for i, p := range resolved {
		if math.IsNaN(p) {
			continue
		}
		if p < previous {
			resolved[i] = previous
		} else {
			previous = p
		}
	}

	previousSet := 0
	for i := 1; i < len(resolved); i++ {
		if math.IsNaN(resolved[i]) {
			continue
		}
		base := resolved[previousSet]
		increment := (resolved[i] - base) / float64(i-previousSet)
		for j := previousSet + 1; j < i; j++ {
			resolved[j] = base + float64(j)*increment
		}
		previousSet = i
	}

	pdfHints := make([]float64, len(hintFractions))
	for i, hint := range hintFractions {
		switch {
		case hint <= 0:
			hint = 0
		case hint >= 1:
			hint = maxHint
		default:
			before := resolved[i] / vectorLength
			after := resolved[i+1] / vectorLength
			if before == after || hint >= after || hint <= before {
				hint = 1
			} else {
				hint = math.Min(maxHint, math.Log(0.5)/math.Log((hint-before)/(after-before)))
			}
		}
		pdfHints[i] = hint
	}
	return resolved, pdfHints
}

// normalizeStopPositions maps positions to the range from 0 to 1. It returns
// the original first and last positions along with the normalized ones.
func normalizeStopPositions(positions []float64) (first, last float64, normalized []float64) {
	first, last = positions[0], positions[len(positions)-1]
	normalized = make([]float64, len(positions))
	total := last - first
	if total == 0 {
		return first, last, normalized
	}
	for i, p := range positions {
		normalized[i] = (p - first) / total
	}
	return first, last, normalized
}

// srgbComponents returns the color as non-premultiplied sRGB channels.
func srgbComponents(c syntax.Color) (r, g, b float64) {
	if converted, ok := c.To(syntax.SRGB); ok {
		c = converted
	}
	var rgb [3]float64
	for i := range rgb {
		if i < len(c.Coordinates) && !math.IsNaN(c.Coordinates[i]) {
			rgb[i] = c.Coordinates[i]
		}
	}
	return rgb[0], rgb[1], rgb[2]
}

// gradientAverageColor is the average color of a gradient (CSS Images 3
// §3.5.3), the weighted mean of premultiplied colors over the segments.
func gradientAverageColor(colors []syntax.Color, positions []float64) syntax.Color {
	total := positions[len(positions)-1] - positions[0]
	if total == 0 {
		positions = make([]float64, len(colors))
		for i := range positions {
			positions[i] = float64(i)
		}
		total = float64(len(colors) - 1)
	}
	var r, g, b, a float64
	totalWeight := 2 * total
	for i := 1; i < len(positions); i++ {
		weight := (positions[i] - positions[i-1]) / totalWeight
		for _, j := range [2]int{i - 1, i} {
			cr, cg, cb := srgbComponents(colors[j])
			alpha := colors[j].Alpha
			r += cr * alpha * weight
			g += cg * alpha * weight
			b += cb * alpha * weight
			a += alpha * weight
		}
	}
	if a == 0 {
		return syntax.NewColor(syntax.SRGB, []float64{0, 0, 0}, 0)
	}
	return syntax.NewColor(syntax.SRGB, []float64{r / a, g / a, b / a}, a)
}
