package images

import (
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

// gradientFor returns the gradient that background-image: css computes to on
// a div, with the default text color as currentcolor.
func gradientFor(t testing.TB, css string) Image {
	t.Helper()
	doc := csstest.Parse(t, `<div style="background-image: `+css+`"></div>`)
	styles := csstest.StyleFor(t, doc, cascade.Options{})
	style := styles.Get(csstest.Element(t, doc, "div"), "")
	img, ok := GradientFromProps(style.BackgroundImage()[0], style.Color().Color)
	if !ok {
		t.Fatalf("background-image: %s is not a gradient", css)
	}
	return img
}

type layouter interface {
	layout(width, height float64) gradientLayout
}

// gradientLayoutFor lays out the gradient of css in a 400×300 box.
func gradientLayoutFor(t testing.TB, css string) gradientLayout {
	t.Helper()
	l, ok := gradientFor(t, css).(layouter)
	if !ok {
		t.Fatalf("%s: gradient has no layout", css)
	}
	return l.layout(400, 300)
}

var (
	red  = rgba(1, 0, 0, 1)
	lime = rgba(0, 1, 0, 1)
	blue = rgba(0, 0, 1, 1)
)

func approx(a, b float64) bool {
	return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b))
}

func colorsNear(a, b syntax.Color) bool {
	return approxSlice(a.Coordinates, b.Coordinates) && approx(a.Alpha, b.Alpha)
}

func approxSlice(got, want []float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !approx(got[i], want[i]) {
			return false
		}
	}
	return true
}
