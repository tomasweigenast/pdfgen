package draw_test

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

// pageContent returns the content stream of the only page of the document.
func pageContent(t *testing.T, html string) string {
	t.Helper()
	pdf := string(rendertest.PDF(t, html))
	_, rest, ok := strings.Cut(pdf, "stream\n")
	if !ok {
		t.Fatal("no content stream in the PDF")
	}
	content, _, ok := strings.Cut(rest, "\nendstream")
	if !ok {
		t.Fatal("unterminated content stream")
	}
	return content
}

func numbers(t *testing.T, fields string) []float64 {
	t.Helper()
	var values []float64
	for _, f := range strings.Fields(fields) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			t.Fatalf("parse %q: %v", f, err)
		}
		values = append(values, v)
	}
	return values
}

func assertNumbers(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", name, got, want)
		return
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-4 {
			t.Errorf("%s = %v, want %v", name, got, want)
			return
		}
	}
}

func TestUniformSolidBorderIsOneEvenOddFill(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 20px } body { margin: 0 }</style>
		<div style="width: 12px; height: 12px; border: 2px solid red"></div>`)

	const want = "1 0 0 rg\n2 2 12 12 re\n0 0 16 16 re\nf*"
	if !strings.Contains(content, want) {
		t.Errorf("content does not paint the border ring with one fill:\n%s", content)
	}
	if n := strings.Count(content, "f*"); n != 1 {
		t.Errorf("got %d even-odd fills, want 1", n)
	}
}

func TestDoubleBorderFillsThreeRings(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 20px } body { margin: 0 }</style>
		<div style="width: 12px; height: 12px; border: 3px double red"></div>`)

	const want = "1 0 0 rg\n3 3 12 12 re\n1 1 16 16 re\n2 2 14 14 re\n0 0 18 18 re\nf*"
	if !strings.Contains(content, want) {
		t.Errorf("content does not paint the double border with one fill:\n%s", content)
	}
}

var strokedLine = regexp.MustCompile(`\[([\d. ]+)\] 0 d\n([\d. -]+) m\n([\d. -]+) l\nS`)

func TestUniformDashedBorderStrokesOneLinePerSide(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 40px } body { margin: 0 }</style>
		<div style="width: 32px; height: 32px; border: 2px dashed red"></div>`)

	if !strings.Contains(content, "1 0 0 RG\n2 w\n0 J\n") {
		t.Errorf("dashed border does not set a butt cap and the width:\n%s", content)
	}
	// Dashes of about three widths, as many as fit with equal gaps: four
	// dashes and three gaps along the 36px of each side.
	const dash = 36. / 7
	lines := strokedLine.FindAllStringSubmatch(content, -1)
	if len(lines) != 4 {
		t.Fatalf("got %d stroked lines, want 4:\n%s", len(lines), content)
	}
	ends := [][]float64{
		{0, 1, 36, 1},
		{35, 0, 35, 36},
		{0, 35, 36, 35},
		{1, 0, 1, 36},
	}
	for i, m := range lines {
		assertNumbers(t, "dash", numbers(t, m[1]), []float64{dash, dash})
		assertNumbers(t, "line", numbers(t, m[2]+" "+m[3]), ends[i])
	}
}

func TestUniformDottedBorderStrokesRoundDots(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 40px } body { margin: 0 }</style>
		<div style="width: 32px; height: 32px; border: 2px dotted red"></div>`)

	if !strings.Contains(content, "1 0 0 RG\n2 w\n1 J\n") {
		t.Errorf("dotted border does not set a round cap and the width:\n%s", content)
	}
	// A dot on each corner and nine gaps between them along each side.
	const gap = 34. / 9
	lines := strokedLine.FindAllStringSubmatch(content, -1)
	if len(lines) != 4 {
		t.Fatalf("got %d stroked lines, want 4:\n%s", len(lines), content)
	}
	ends := [][]float64{
		{1, 1, 35, 1},
		{35, 1, 35, 35},
		{1, 35, 35, 35},
		{1, 1, 1, 35},
	}
	for i, m := range lines {
		assertNumbers(t, "dash", numbers(t, m[1]), []float64{0, gap})
		assertNumbers(t, "line", numbers(t, m[2]+" "+m[3]), ends[i])
	}
}

func TestMixedColorBordersAreClippedSideBySide(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 20px } body { margin: 0 }</style>
		<div style="width: 12px; height: 12px; border: 2px solid;
		            border-color: red lime blue black"></div>`)

	// The page background is white; then bottom, left, right and top.
	colors := regexp.MustCompile(`(?m)^([\d. ]+) rg$`).FindAllStringSubmatch(content, -1)
	var got []string
	for _, m := range colors {
		got = append(got, m[1])
	}
	want := []string{"1 1 1", "0 0 1", "0 0 0", "0 1 0", "1 0 0"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("fill colors = %v, want %v", got, want)
	}
	if n := strings.Count(content, "W*\nn\n"); n != 4 {
		t.Errorf("got %d segment clips, want 4", n)
	}
	if n := strings.Count(content, "f*"); n != 4 {
		t.Errorf("got %d even-odd fills, want 4", n)
	}
}

func TestTransparentSideIsNotDrawn(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 20px } body { margin: 0 }</style>
		<div style="width: 12px; height: 12px; border: 2px solid;
		            border-color: red transparent red red"></div>`)

	if n := strings.Count(content, "f*"); n != 3 {
		t.Errorf("got %d even-odd fills, want 3 (one per visible side)", n)
	}
}

func TestHiddenBoxHasNoBorder(t *testing.T) {
	testutil.AssertNoLogs(t)
	content := pageContent(t, `
		<style>@page { size: 20px } body { margin: 0 }</style>
		<div style="width: 12px; height: 12px; border: 2px solid red; visibility: hidden"></div>`)

	if strings.Contains(content, "f*") {
		t.Errorf("hidden box paints a border:\n%s", content)
	}
}
