package images

import (
	"math"
	"slices"
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
)

// maxRepeats bounds how many times a repeating linear gradient is unrolled.
// Gradients that would need more are painted with their average color.
const maxRepeats = 1 << 14

// LinearGradient is a linear-gradient() or repeating-linear-gradient() image
// (CSS Images 3 §3.1).
type LinearGradient struct {
	gradient
	// angle is in radians, 0 pointing upwards and growing clockwise. It is
	// ignored when corner is set.
	angle  float64
	corner string
}

// NewLinearGradient returns the gradient g; current replaces currentcolor in
// its color stops.
func NewLinearGradient(g *props.LinearGradient, current syntax.Color) *LinearGradient {
	out := &LinearGradient{gradient: newGradient(g.Gradient, current), angle: g.Angle}
	if g.DirectionType == "corner" {
		out.corner = string(g.Corner)
	}
	return out
}

// Draw paints the gradient into a w×h box at the origin of s.
func (g *LinearGradient) Draw(s *stream.Stream, w, h float64, rendering string) {
	paint(s, w, h, g.layout(w, h))
}

// layout resolves the gradient against a width×height box whose top left
// corner is the origin.
func (g *LinearGradient) layout(width, height float64) gradientLayout {
	if len(g.colors) == 1 {
		return solidLayout(g.colors[0])
	}

	// (dx, dy) is the unit vector of the gradient line, y growing downwards.
	var dx, dy float64
	if g.corner != "" {
		vertical, horizontal, _ := strings.Cut(g.corner, "_")
		factorX, factorY := 1.0, 1.0
		if horizontal == "left" {
			factorX = -1
		}
		if vertical == "top" {
			factorY = -1
		}
		// The gradient line is perpendicular to the diagonal through the
		// neighbouring corners, hence the swap of height and width.
		diagonal := math.Hypot(width, height)
		dx = factorX * height / diagonal
		dy = factorY * width / diagonal
	} else {
		dx, dy = math.Sin(g.angle), -math.Cos(g.angle)
	}
	dx, dy = roundTo9(dx), roundTo9(dy)

	vectorLength := math.Abs(width*dx) + math.Abs(height*dy)
	if vectorLength == 0 {
		return gradientLayout{kind: gradientNone}
	}
	colors := append([]syntax.Color(nil), g.colors...)
	positions, hints := processColorStops(vectorLength, g.stops, g.hints)
	if !g.repeating {
		// PDF does not extend stops that are outside of the shading domain,
		// so give the boundaries explicit colors.
		if positions[0] == positions[1] {
			positions = slices.Insert(positions, 0, positions[0]-1)
			colors = slices.Insert(colors, 0, colors[0])
			hints = slices.Insert(hints, 0, 1)
		}
		if n := len(positions); positions[n-2] == positions[n-1] {
			positions = append(positions, positions[n-1]+1)
			colors = append(colors, colors[len(colors)-1])
			hints = append(hints, 1)
		}
	}
	first, last, positions := normalizeStopPositions(positions)

	if g.repeating {
		// CSS Images 3 §3.6: an empty range paints the average color.
		if first == last || vectorLength/(last-first) > maxRepeats {
			return solidLayout(gradientAverageColor(colors, positions))
		}
		colors, positions, hints, first, last = repeatLinear(colors, positions, hints, first, last, vectorLength)
	}

	startX := (width - dx*vectorLength) / 2
	startY := (height - dy*vectorLength) / 2
	return gradientLayout{
		scaleY: 1,
		kind:   gradientLinear,
		points: []float64{
			startX + dx*first, startY + dy*first,
			startX + dx*last, startY + dy*last,
		},
		positions: positions,
		colors:    colors,
		hints:     hints,
		extend:    !g.repeating,
	}
}

// repeatLinear unrolls a repeating gradient whose stops, normalized to the
// range from 0 to 1, span first to last along the vector, until it covers
// the whole vector. It returns the stops with the new first and last
// positions along the vector.
func repeatLinear(colors []syntax.Color, positions, hints []float64, first, last, vectorLength float64) ([]syntax.Color, []float64, []float64, float64, float64) {
	stopLength := last - first
	steps := make([]float64, len(positions)-1)
	for i := range steps {
		steps[i] = positions[i+1] - positions[i]
	}
	if len(hints) == 0 {
		hints = []float64{1}
	}

	nextSteps := newCycler(append([]float64{0}, steps...))
	nextColors := newCycler(colors)
	nextHints := newCycler(hints)
	previousSteps := newCycler(append([]float64{0}, reversed(steps)...))
	previousColors := newCycler(reversed(colors))
	previousHints := newCycler(reversed(hints))
	for last < vectorLength {
		step := nextSteps.next()
		colors = append(colors, nextColors.next())
		hints = append(hints, nextHints.next())
		positions = append(positions, positions[len(positions)-1]+step)
		last += step * stopLength
	}

	// The stops before the first one come from the end of the cycle, walked
	// backwards; they are collected outwards and prepended once.
	var beforeColors []syntax.Color
	var beforeHints, beforePositions []float64
	edge := positions[0]
	for first > 0 {
		step := previousSteps.next()
		beforeColors = append(beforeColors, previousColors.next())
		beforeHints = append(beforeHints, previousHints.next())
		edge -= step
		beforePositions = append(beforePositions, edge)
		first -= step * stopLength
	}
	colors = append(reversed(beforeColors), colors...)
	hints = append(reversed(beforeHints), hints...)
	positions = append(reversed(beforePositions), positions...)
	return colors, positions, hints, first, last
}

// cycler yields the items of a slice over and over.
type cycler[T any] struct {
	items []T
	i     int
}

func newCycler[T any](items []T) *cycler[T] { return &cycler[T]{items: items} }

func (c *cycler[T]) next() T {
	v := c.items[c.i]
	c.i = (c.i + 1) % len(c.items)
	return v
}

func reversed[T any](s []T) []T {
	out := make([]T, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

func roundTo9(v float64) float64 { return math.Round(v*1e9) / 1e9 }
