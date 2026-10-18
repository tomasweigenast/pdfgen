package draw

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

func lineOperators(style props.Keyword, x1, y1, x2, y2, thickness, offset float64) string {
	s := stream.New(stream.NewResources(writer.Ref{Num: 1}), stream.Options{})
	red := syntax.NewColor(syntax.SRGB, []float64{1, 0, 0}, 1)
	blue := syntax.NewColor(syntax.SRGB, []float64{0, 0, 1}, 1)
	drawLine(s, x1, y1, x2, y2, thickness, style, [2]syntax.Color{red, blue}, offset)
	return string(s.Bytes())
}

func TestDrawLineSolid(t *testing.T) {
	got := lineOperators("solid", 1, 2, 11, 2, 4, 0)
	for _, want := range []string{"1 0 0 RG", "4 w", "1 2 m", "11 2 l", "S"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestDrawLineDashed(t *testing.T) {
	got := lineOperators("dashed", 0, 0, 10, 0, 2, 3)
	if !strings.Contains(got, "[10] 3 d") {
		t.Errorf("dash array = %q, want [10] 3 d", got)
	}
}

func TestDrawLineDotted(t *testing.T) {
	got := lineOperators("dotted", 0, 0, 10, 0, 2, 0)
	for _, want := range []string{"1 J", "[0 4] 0 d"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestDrawLineDoubleStrokesTwoThinnerLines(t *testing.T) {
	got := lineOperators("double", 0, 3, 9, 3, 3, 0)
	if !strings.Contains(got, "1 w") || strings.Count(got, " m\n") != 2 {
		t.Errorf("double line = %q, want two segments 1 wide", got)
	}
	for _, want := range []string{"0 2 m", "9 2 l", "0 4 m", "9 4 l"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestDrawLineVerticalDoubleShiftsHorizontally(t *testing.T) {
	got := lineOperators("double", 5, 0, 5, 9, 3, 0)
	for _, want := range []string{"4 0 m", "4 9 l", "6 0 m", "6 9 l"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestDrawLineRidgeUsesBothColors(t *testing.T) {
	got := lineOperators("groove", 0, 0, 8, 0, 4, 0)
	if strings.Index(got, "1 0 0 RG") > strings.Index(got, "0 0 1 RG") || strings.Count(got, "S\n") != 2 {
		t.Errorf("groove line = %q, want the outer then the inner half", got)
	}
	if !strings.Contains(got, "2 w") || !strings.Contains(got, "0 1 m") || !strings.Contains(got, "0 -1 m") {
		t.Errorf("groove line = %q, want halves 2 wide shifted by a quarter", got)
	}
}

func TestDrawLineWavyClipsAndCoversTheLength(t *testing.T) {
	got := lineOperators("wavy", 0, 10, 12, 10, 2, 0)
	if !strings.Contains(got, "0 7 12 6 re") || !strings.Contains(got, "W") {
		t.Errorf("wavy line = %q, want a clip to the height of the wave", got)
	}
	// Each curve spans 2*0.75*thickness = 3px.
	if n := strings.Count(got, " c\n"); n != 4 {
		t.Errorf("curves = %d, want 4", n)
	}
}

func TestDrawLineWavyWithoutThicknessTerminates(t *testing.T) {
	lineOperators("wavy", 0, 10, 12, 10, 0, 0)
}

func TestDrawLineIgnoresDiagonalSegments(t *testing.T) {
	if got := lineOperators("solid", 0, 0, 5, 5, 1, 0); got != "" {
		t.Errorf("diagonal line drew %q", got)
	}
}
