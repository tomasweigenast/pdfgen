package images

import (
	"math"
	"slices"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// Zero and unbounded radii of degenerate radial gradients are replaced by
// these, which keep the scale of the ellipse finite.
const (
	tinyRadius = 1e-7
	hugeRadius = 1e7
)

// RadialGradient is a radial-gradient() or repeating-radial-gradient() image
// (CSS Images 3 §3.2).
type RadialGradient struct {
	gradient
	center       props.Position
	circle       bool
	sizeKeyword  string
	sizeExplicit *[2]props.Length
}

// NewRadialGradient returns the gradient g; current replaces currentcolor in
// its color stops.
func NewRadialGradient(g *props.RadialGradient, current syntax.Color) *RadialGradient {
	out := &RadialGradient{
		gradient: newGradient(g.Gradient, current),
		center:   g.Center,
		circle:   g.Shape == "circle",
	}
	if g.SizeType == "explicit" {
		size := g.SizeExplicit
		out.sizeExplicit = &size
	} else {
		out.sizeKeyword = string(g.SizeKeyword)
	}
	return out
}

// Draw paints the gradient into a w×h box at the origin of s.
func (g *RadialGradient) Draw(s *stream.Stream, w, h float64, rendering string) {
	paint(s, w, h, g.layout(w, h))
}

// layout resolves the gradient against a width×height box whose top left
// corner is the origin.
func (g *RadialGradient) layout(width, height float64) gradientLayout {
	if len(g.colors) == 1 {
		return solidLayout(g.colors[0])
	}

	centerX := resolveOr(g.center.X, width, width/2)
	centerY := resolveOr(g.center.Y, height, height/2)
	if g.center.OriginX == "right" {
		centerX = width - centerX
	}
	if g.center.OriginY == "bottom" {
		centerY = height - centerY
	}

	sizeX, sizeY := g.resolveSize(width, height, centerX, centerY)
	sizeX, sizeY = handleDegenerate(sizeX, sizeY)
	if math.IsNaN(sizeX) || math.IsNaN(sizeY) || sizeX < 0 || sizeY < 0 {
		logging.Warnf("Invalid radial gradient size %vx%v, nothing is drawn", sizeX, sizeY)
		return gradientLayout{kind: gradientNone}
	}
	scaleY := sizeY / sizeX

	colors := slices.Clone(g.colors)
	positions, hints := processColorStops(sizeX, g.stops, g.hints)
	if !g.repeating {
		// PDF does not extend stops that are outside of the shading domain,
		// so give the boundaries explicit colors.
		if positions[0] > 0 && positions[0] == positions[1] {
			positions = slices.Insert(positions, 0, 0)
			colors = slices.Insert(colors, 0, colors[0])
			hints = slices.Insert(hints, 0, 1)
		}
		if n := len(positions); positions[n-2] == positions[n-1] {
			positions = append(positions, positions[n-1]+1)
			colors = append(colors, colors[len(colors)-1])
			hints = append(hints, 1)
		}
	}
	if positions[0] < 0 {
		// PDF does not accept negative radii, so shift the stops into the
		// positive side.
		if g.repeating {
			vectorLength := positions[len(positions)-1] - positions[0]
			offset := vectorLength * (1 + math.Floor(-positions[0]/vectorLength))
			for i := range positions {
				positions[i] += offset
			}
		} else {
			if positions[len(positions)-1] <= 0 {
				return solidLayout(g.colors[len(g.colors)-1])
			}
			colors, positions = clipToPositive(colors, positions)
		}
	}
	first, last, positions := normalizeStopPositions(positions)

	if first == last && g.repeating {
		return solidLayout(gradientAverageColor(colors, positions))
	}

	points := []float64{centerX, centerY / scaleY, first, centerX, centerY / scaleY, last}
	if g.repeating {
		points, positions, colors, hints = repeatRadial(width, height, scaleY, points, positions, colors, hints)
	}
	return gradientLayout{
		scaleY:    scaleY,
		kind:      gradientRadial,
		points:    points,
		positions: positions,
		colors:    colors,
		hints:     hints,
		extend:    !g.repeating,
	}
}

// clipToPositive drops the part of a gradient that has negative radii. The
// color at radius 0 is interpolated when no stop is there.
func clipToPositive(colors []syntax.Color, positions []float64) ([]syntax.Color, []float64) {
	for i, p := range positions {
		if p == 0 {
			return colors[i:], positions[i:]
		}
		if p > 0 {
			previous := colors[i-1]
			color := gradientAverageColor(
				[]syntax.Color{previous, previous, colors[i], colors[i]},
				[]float64{positions[i-1], 0, 0, p})
			return append([]syntax.Color{color}, colors[i:]...), append([]float64{0}, positions[i:]...)
		}
	}
	return colors, positions
}

// repeatRadial unrolls a repeating gradient from the center to the farthest
// corner of the box. points holds the starting and ending circles, whose
// radii are the first and last stop.
func repeatRadial(width, height, scaleY float64, points, positions []float64, colors []syntax.Color, hints []float64) ([]float64, []float64, []syntax.Color, []float64) {
	originalColors := slices.Clone(colors)
	originalHints := slices.Clone(hints)
	originalPositions := slices.Clone(positions)
	gradientLength := points[5] - points[2]

	// The farthest corner tells how many times the stops repeat outside.
	maxDistance := math.Max(
		math.Max(
			math.Hypot(width-points[0], height/scaleY-points[1]),
			math.Hypot(width-points[0], -points[1]*scaleY)),
		math.Max(
			math.Hypot(-points[0], height/scaleY-points[1]),
			math.Hypot(-points[0], -points[1]*scaleY)))
	repeatAfter := int(math.Ceil((maxDistance - points[5]) / gradientLength))
	if repeatAfter > 0 {
		repeat := 1 + repeatAfter
		colors = repeatSlice(colors, repeat)
		hints = joinRepeated(hints, repeat)
		positions = positions[:0:0]
		for i := range repeat {
			for _, p := range originalPositions {
				positions = append(positions, float64(i)+p)
			}
		}
		points = slices.Clone(points)
		points[5] += gradientLength * float64(repeatAfter)
	}

	if points[2] == 0 {
		return points, positions, colors, hints
	}

	// The inner circle is moved to the center, and the stops before it are
	// the same stops repeated inwards.
	repeatBefore := points[2] / gradientLength
	points = slices.Clone(points)
	points[2] = 0

	fullRepeat := int(repeatBefore)
	if fullRepeat > 0 {
		colors = append(repeatSlice(originalColors, fullRepeat), colors...)
		var before []float64
		for range fullRepeat {
			before = append(before, 1)
			before = append(before, originalHints...)
		}
		hints = append(before, hints...)
		var inner []float64
		for i := range fullRepeat {
			for _, p := range originalPositions {
				inner = append(inner, float64(i-fullRepeat)+p)
			}
		}
		positions = append(inner, positions...)
	}

	partialRepeat := repeatBefore - float64(fullRepeat)
	if partialRepeat == 0 {
		return points, positions, colors, hints
	}

	// What is left is the part of the stops that ends at the inner circle,
	// found walking them backwards from the last one.
	ratio := 1 - partialRepeat
	n := len(originalPositions)
	for i := 1; i <= n; i++ {
		position := originalPositions[n-i]
		if position == ratio {
			colors = append(slices.Clone(originalColors[n-i:]), colors...)
			hints = append(append(slices.Clone(originalHints[len(originalHints)-min(i, len(originalHints)):]), 1), hints...)
			shifted := make([]float64, 0, i)
			for _, p := range originalPositions[n-i:] {
				shifted = append(shifted, p-float64(fullRepeat)-1)
			}
			return points, append(shifted, positions...), colors, hints
		}
		if position < ratio {
			color := originalColors[n-i]
			next := originalColors[n-i+1]
			nextPosition := originalPositions[n-i+1]
			zero := gradientAverageColor(
				[]syntax.Color{color, color, next, next},
				[]float64{position, ratio, ratio, nextPosition})
			tail := originalColors[n-i+1:]
			colors = slices.Concat([]syntax.Color{zero}, tail, colors)
			tailHints := originalHints[len(originalHints)-min(i-1, len(originalHints)):]
			hints = slices.Concat([]float64{1}, tailHints, []float64{1}, hints)
			shifted := []float64{ratio - 1 - float64(fullRepeat)}
			for _, p := range originalPositions[n-i+1:] {
				shifted = append(shifted, p-1-float64(fullRepeat))
			}
			return points, append(shifted, positions...), colors, hints
		}
	}
	return points, positions, colors, hints
}

func repeatSlice[T any](s []T, n int) []T {
	out := make([]T, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}
	return out
}

// joinRepeated repeats hints n times, separated by a linear hint, as the
// hints sit between colors.
func joinRepeated(hints []float64, n int) []float64 {
	var out []float64
	for i := range n {
		if i > 0 {
			out = append(out, 1)
		}
		out = append(out, hints...)
	}
	return out
}

// resolveOr resolves l against ref, or returns def when l cannot be resolved.
func resolveOr(l props.Length, ref, def float64) float64 {
	if v, ok := resolveLength(l, ref); ok {
		return v
	}
	return def
}

// resolveSize resolves the radii of the ending shape (CSS Images 3 §3.2.3).
func (g *RadialGradient) resolveSize(width, height, centerX, centerY float64) (float64, float64) {
	if g.sizeExplicit != nil {
		return resolveOr(g.sizeExplicit[0], width, 0), resolveOr(g.sizeExplicit[1], height, 0)
	}
	left := math.Abs(centerX)
	right := math.Abs(width - centerX)
	top := math.Abs(centerY)
	bottom := math.Abs(height - centerY)
	closest := strings.HasPrefix(g.sizeKeyword, "closest")
	pick := func(values ...float64) float64 {
		best := values[0]
		for _, v := range values[1:] {
			if (closest && v < best) || (!closest && v > best) {
				best = v
			}
		}
		return best
	}
	if strings.HasSuffix(g.sizeKeyword, "side") {
		if g.circle {
			size := pick(left, right, top, bottom)
			return size, size
		}
		return pick(left, right), pick(top, bottom)
	}
	if g.circle {
		size := pick(math.Hypot(left, top), math.Hypot(left, bottom), math.Hypot(right, top), math.Hypot(right, bottom))
		return size, size
	}
	corners := [4][2]float64{{left, top}, {left, bottom}, {right, top}, {right, bottom}}
	best := corners[0]
	for _, c := range corners[1:] {
		d, bd := math.Hypot(c[0], c[1]), math.Hypot(best[0], best[1])
		if (closest && d < bd) || (!closest && d > bd) {
			best = c
		}
	}
	return best[0] * math.Sqrt2, best[1] * math.Sqrt2
}

// handleDegenerate replaces zero radii by values that keep the scale of the
// ellipse finite (CSS Images 3 §3.2.4).
func handleDegenerate(sizeX, sizeY float64) (float64, float64) {
	switch {
	case sizeX == 0 && sizeY == 0:
		return tinyRadius, tinyRadius
	case sizeX == 0:
		return tinyRadius, hugeRadius
	case sizeY == 0:
		return hugeRadius, tinyRadius
	}
	return sizeX, sizeY
}
