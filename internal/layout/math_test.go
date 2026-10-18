package layout_test

import (
	"math"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/layout"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// renderPagesOrSkip lays out a document, skipping the test when it needs a
// layout mode that is not implemented. Warnings are collected and returned,
// not checked.
func renderPagesOrSkip(t *testing.T, source string) ([]*boxes.PageBox, []string) {
	t.Helper()
	logs := testutil.CaptureLogs(t)
	d := boxtest.FakeHTML(t, source)
	root := d.Build(t)
	pages, err := layout.LayoutDocument(d.HTML, root, layout.NewContext(d.Builder), 0)
	if err != nil {
		const prefix, suffix = "layout: layout of ", " is not supported yet"
		if msg := err.Error(); strings.HasPrefix(msg, prefix) && strings.HasSuffix(msg, suffix) {
			t.Skipf("not supported: %s", strings.TrimSuffix(strings.TrimPrefix(msg, prefix), suffix))
		}
		t.Fatalf("lay out pages: %v", err)
	}
	for _, page := range pages {
		boxtest.SanityChecks(t, page)
	}
	return pages, logs.Lines()
}

// divWidth lays out a document that holds only a division and returns its
// width.
func divWidth(t *testing.T, source string) float64 {
	t.Helper()
	_, body := renderedBody(t, source)
	return child(t, body).Base().Width.V
}

func TestMathFunctionsSetTheWidthOfBoxes(t *testing.T) {
	widths := []string{
		"calc(100px)", "calc(10em)", "calc(50vw)", "calc(20pvh)", "calc(50%)", "calc(10px + 90px)",
		"calc(5em + 50px)", "calc(2 * 5em)", "calc(2 * (3em + 20px))", "calc(25% * (1 + 1))",
		"calc(20% * (1 + 1) + 20px)", "calc(100px", "max(100px)", "max(30%, 2em, 100px)", "max(-30%, -2em, 10em)",
		"calc(max(-1, 1, 2) * 50px)", "min(100px)", "min(100%, 20em, 100px)", "calc(min(4, 2) * 50px)",
		"calc(sqrt(4) * 50px)", "calc(pow(2, 2) * 25px)", "calc(hypot(2) * 50px)", "calc(hypot(3, 4) * 20px)",
		"calc(hypot(2px) * 50)", "calc(hypot(3px, 4px) * 20)", "calc(log(e) * 100px)", "calc(log(100, 10) * 50px)",
		"calc(exp(1) / e * 100px)", "abs(-100px)", "calc(abs(-100) * 1px)", "calc(sign(-100) * -100px)",
		"calc(sign(-100px) * -100px)", "calc(sqrt(16) * min(25px, 100%))",
		"clamp(calc(-infinity * 1px), 10em, calc(infinity * 1px))", "clamp(50px, 10em, 500px)",
		"clamp(100px, 2em, 500px)", "clamp(10px, 100em, 10em)", "clamp(10px, 100%, 10em)", "round(100.4px)",
		"round(145.4px, 100px)", "round(nearest, 100px)", "round(down, 195px, 100px)", "round(up, 5px, 100px)",
		"round(to-zero, 195px, 100px)", "mod(300px, 200px)", "calc(mod(300px, -200px) * -1)",
		"calc(mod(-300px, -200px) * -1)", "rem(300px, 200px)", "rem(300px, -200px)",
		"calc(rem(-300px, -200px) * -1)", "calc(sin(30deg) * 200px)", "calc(cos(60deg) * 200px)",
		"calc(tan(45deg) * 100px)", "calc(tan(calc(pi / 4)) * 100px)", "calc(sin(asin(0.5)) * 200px)",
		"calc(cos(acos(0.5)) * 200px)", "calc(tan(atan(1)) * 100px)", "calc(tan(atan2(1, 1)) * 100px)",
		"calc(100px * var(--one))", "calc(50% * var(--one))", "calc(100px * sqrt(var(--one)))",
	}
	for _, width := range widths {
		t.Run(width, func(t *testing.T) {
			got := divWidth(t, `<style>
        @page { size: 400px 500px; margin: 100px }
        body { font-size: 10px; width: 200px }
      </style>
      <div style="--one: 1; height: 1px; width: `+width+`"></div>`)
			if math.Abs(got-100) > 1e-9*100 {
				t.Errorf("width = %v, want 100", got)
			}
		})
	}
}

func TestCalcDivisionByZeroLaysOutLikeInfinityAndNaN(t *testing.T) {
	tests := [][2]string{
		{"calc(100px / 0)", "calc(infinity * 1px)"},
		{"calc(50px / 0)", "calc(infinity * 1px)"},
		{"calc(10em / 0)", "calc(infinity * 1px)"},
		{"calc(-100px / 0)", "calc(-infinity * 1px)"},
		{"calc(0px / 0)", "calc(nan * 1px)"},
		{"calc(100px / (1 - 1))", "calc(infinity * 1px)"},
		{"calc(100px / 0 + 100px)", "calc(infinity * 1px)"},
		{"calc(-1px / 0 * 100)", "calc(-infinity * 1px)"},
	}
	width := func(t *testing.T, value string) float64 {
		return divWidth(t, `<div style="font-size: 10px; height: 1px; width: `+value+`"></div>`)
	}
	for _, tt := range tests {
		t.Run(tt[0], func(t *testing.T) {
			if got, want := width(t, tt[0]), width(t, tt[1]); got != want {
				t.Errorf("%s = %v, %s = %v", tt[0], got, tt[1], want)
			}
		})
	}
}

func TestInvalidMathFunctionsWarnOnce(t *testing.T) {
	widths := []string{
		"calc", "(calc)", "calc(", "calc()", `calc("100px")`, "calc(100)", "calc(100px 100px)",
		"calc(100px, 100px)", "calc(100px * 100px)", "calc(100 * 100)", "calc(calc(100unknown))", "calc(0.1)",
		"calc(-1)", "min()", "min(10)", `min("10px")`, "min(10, 5px)", "calc(min(1, 5px) * 10px)", "max()",
		"max(10)", `max("10px")`, "max(10, 50px)", "calc(max(100, 5px) * 10px)", "calc(100* - max(56px, 1rem)",
		"clamp()", "clamp(10px)", "clamp(10px, 50px)", "clamp(10px, 50px, 100px, 200px)",
		`clamp(10px, "50px", 100px)`, "round()", "round(100)", "round(100, 10)", "round(nearest, 100, 10)",
		"round(100px, 10)", `round(100px, "10px")`, "round(nearest, 100px, 10)", "round(100px, 10px, 1)",
		"round(nearest, 100px, 10px, 1)", "round(unknown, 100px)", "round(unknown, 100px, 10px)", "mod()",
		"mod(10px)", "mod(100px, 10)", `mod(100px, "10px")`, "calc(mod(300px, 200) * -1)", "mod(100px, 10px, 1px)",
		"rem()", "rem(10px)", "rem(100px, 10)", `rem(100px, "10px")`, "calc(rem(300px, 200) * -1)",
		"rem(100px, 10px, 1px)", "sin()", "sin(10)", "sin(10%)", "sin(10deg)", "calc(sin(10) * 1)", "cos()",
		"cos(10)", "cos(10%)", "cos(10deg)", "calc(cos(10) * 1)", "tan()", "tan(10)", "tan(10%)", "tan(10deg)",
		"calc(tan(10) * 1)", "asin()", "asin(0)", "asin(0.5)", "asin(50deg)", "calc(sin(asin(50deg)) * 200px)",
		"calc(sin(asin(0.5)) * 200)", "calc(sin(asin(0.5, 2)) * 200px)", "calc(sin(asin(5)) * 200px)", "acos()",
		"acos(0)", "acos(0.5)", "acos(50deg)", "calc(cos(acos(50deg)) * 200px)", "calc(cos(acos(0.5)) * 200)",
		"calc(cos(acos(0.5, 2)) * 200px)", "calc(cos(acos(5)) * 200px)", "atan()", "atan(0)", "atan(0.5)",
		"atan(50deg)", "calc(tan(atan(50deg)) * 200px)", "calc(tan(atan(0.5)) * 200)",
		"calc(tan(atan(0.5, 2)) * 200px)", "atan2()", "atan2(0.5)", "atan2(0.5, 1)", "atan2(50deg, 1)",
		"calc(tan(atan2(50deg, 1)) * 200px)", "calc(tan(atan2(0.5, 1)) * 200)", "pow()", "pow(4, 3)",
		"pow(4px, 3)", "pow(4, 3, 4)", "sqrt()", "sqrt(4)", "sqrt(4px)", "sqrt(4, 2)", "hypoth()", "hypoth(3)",
		"hypoth(3, 4)", "log()", "log(10)", "log(10px)", "log(10, 10)", "log(10px, 10)", "log(10, 10, 10)",
		"exp()", "exp(10)", "exp(10px)", "exp(10, 10)", "exp(10px, 10)", "exp(10, 10, 10)", "abs()", "abs(10)",
		"abs(10px, 100)", "sign()", "sign(10)", "sign(10px)", "sign(10px, 10)",
	}
	for _, width := range widths {
		t.Run(width, func(t *testing.T) {
			logs := testutil.CaptureLogs(t)
			boxtest.RenderPages(t, `<style>body { font-size: 10px; width: 200px }</style>
          <div style="--one: 1; height: 1px; width: `+width+`"></div>`)
			if lines := logs.Lines(); len(lines) != 1 {
				t.Errorf("logs = %q, want one message", lines)
			}
		})
	}
}

func TestMathWithPercentageAndFontUnitWarnsLikeItsParts(t *testing.T) {
	for _, p := range props.Table {
		if p.Validate == nil {
			continue
		}
		t.Run(p.Name, func(t *testing.T) {
			warnings := func(value string) int {
				_, lines := renderPagesOrSkip(t, `<div style="`+p.Name+`: `+value+`"></div>`)
				return len(lines)
			}
			mathLogs := warnings("calc(50% + 1em)")
			logs := warnings("50%")
			if logs == 0 {
				logs = warnings("1em")
			}
			if mathLogs != logs {
				t.Errorf("%d messages with calc(), %d with its parts", mathLogs, logs)
			}
		})
	}
}

func TestMathInLineHeightLaysOut(t *testing.T) {
	for _, lineHeight := range []string{"calc(50% + 1em)", "calc(100% + 2px)", "calc(1.2em + 2px)", "calc(1.5 * 1)"} {
		t.Run(lineHeight, func(t *testing.T) {
			renderPages(t, `<p style="line-height: `+lineHeight+`">Hello</p>`)
		})
	}
}

func TestMathInSizesOfEveryDisplay(t *testing.T) {
	displays := []string{
		"block", "inline", "flex", "grid", "list", "list-item", "table", "table-row-group", "table-cell",
		"inline-block", "inline-table", "inline-flex", "inline-grid",
	}
	const sizes = `min-width: calc(50% + 1em); max-width: calc(50% + 1em); width: calc(50% + 1em);
     min-height: calc(50% + 1em); max-height: calc(50% + 1em); height: calc(50% + 1em)`
	for _, display := range displays {
		t.Run(display, func(t *testing.T) {
			renderPagesOrSkip(t, `
    <div style="display: `+display+`;
     `+sizes+`
    ">
      <div style="
       `+sizes+`
      "></div>
    </div>`)
		})
	}
}

func TestMathInHyphenateLimitZoneLaysOut(t *testing.T) {
	renderPages(t, `
      <div lang="en"
        style="hyphens: auto; hyphenate-limit-zone: calc(1em + 100%); width: 2em">
        absolute
      </div>`)
}

func TestMathInGradientStopsLaysOut(t *testing.T) {
	renderPages(t, `
      <div style="width: 10px; height: 10px; background: linear-gradient(
        blue calc(20% + 1em),
        red calc(80% + 1em))"></div>`)
}

func TestMathInColorFunctions(t *testing.T) {
	t.Skip("not supported: calc() in color functions")
}

func TestMathInGradientColors(t *testing.T) {
	t.Skip("not supported: calc() in color functions")
}

func TestMathInImageSizesInTableCells(t *testing.T) {
	for _, style := range []string{
		`height: calc(10% + 1em); width: calc(10% + 1em); max-height: calc(10% + 1em);
            max-width: calc(10% + 1em); min-height: calc(10% + 1em); min-width: calc(10% + 1em);`,
		`height: calc(10% + 1em); max-height: calc(10% + 1em); max-width: calc(10% + 1em);
            min-height: calc(10% + 1em); min-width: calc(10% + 1em);`,
		`max-height: calc(10% + 1em); max-width: calc(10% + 1em); min-height: calc(10% + 1em);
            min-width: calc(10% + 1em);`,
	} {
		t.Run(style, func(t *testing.T) {
			renderPagesOrSkip(t, `<table><td><img src="pattern.png" style="`+style+`">`)
		})
	}
}

func TestMathInTableMargin(t *testing.T) {
	renderPagesOrSkip(t, `<table style="margin: calc(1em + 10%)">`)
}

func TestMathInGridPadding(t *testing.T) {
	renderPagesOrSkip(t, `
      <article style="display: grid">
        <div style="box-sizing: border-box; border: 1px solid;
                    padding: calc(2px + 10%); width: 7px">a</div>
      </article>`)
}

func TestMathInTableColumnWidths(t *testing.T) {
	renderPagesOrSkip(t, `
      <table style="width: 200px">
        <colgroup style="width: calc(1em + 10%)">
          <col />
        </colgroup>
        <col style="width: calc(1em + 10%)" />
        <tbody>
          <tr>
            <td>a</td>
            <td>a</td>
          </tr>
        </tbody>
      </table>`)
}

func TestMathInBorderSpacingWithFontUnit(t *testing.T) {
	renderPagesOrSkip(t, `
      <table style="
          --spacing: 1em; --border-spacing: calc(var(--spacing) * 5);
          border-spacing: var(--border-spacing) var(--border-spacing)">
        <tbody>
          <tr>
            <td>a</td>
            <td>a</td>
          </tr>
        </tbody>
      </table>`)
}

func TestMathInVerticalAlignOfTableCells(t *testing.T) {
	renderPagesOrSkip(t, `<table><tr><td style="vertical-align: calc(1em + 1%)">abc`)
}

func TestMathInVerticalAlignOfInlineBoxes(t *testing.T) {
	renderPages(t, `<span style="vertical-align: calc(1em + 1%)">abc`)
}

func TestMathInVerticalAlignOfMarginBoxes(t *testing.T) {
	renderPages(t, `
      <style>@page{@top-left{content: "a"; vertical-align: calc(1em + 1%)}}</style>
      <body>abc`)
}
