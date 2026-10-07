package cascade_test

import (
	"math"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

// usedLength resolves a computed length, percentages referring to
// referTo, as the layout does.
func usedLength(t *testing.T, s *computed.Style, l props.Length, property string, referTo float64) float64 {
	t.Helper()
	if l.Calc != nil {
		token, err := tk.ResolveMath(l.Calc.Function, s, property, &referTo)
		if err != nil || token == nil {
			t.Fatalf("cannot resolve %s: %v", l.Calc.Function.Serialize(), err)
		}
		switch tok := token.(type) {
		case *syntax.DimensionToken:
			return s.ToPixels(tok.Value, tok.LowerUnit, property)
		case *syntax.PercentageToken:
			return tok.Value * referTo / 100
		case *syntax.NumberToken:
			return tok.Value
		}
		t.Fatalf("unexpected resolved token %v", token)
	}
	if l.Dimension.Unit == "%" {
		return l.Dimension.Value * referTo / 100
	}
	return l.Dimension.Value
}

// divStyle computes the style of the div of a test document whose page is
// 400×500 with 100px margins, as the layout would set it.
func divStyle(t *testing.T, source string) *computed.Style {
	t.Helper()
	doc := csstest.Parse(t, source)
	sf := csstest.StyleFor(t, doc, cascade.Options{})
	sf.Context().PageBox = &computed.Size{Width: 400, Height: 500}
	sf.Context().PageArea = &computed.Size{Width: 200, Height: 300}
	return sf.Get(csstest.Element(t, doc, "div"), "")
}

func TestMathFunctions(t *testing.T) {
	widths := []string{
		"calc(100px)", "calc(10em)", "calc(50vw)", "calc(20pvh)", "calc(50%)",
		"calc(10px + 90px)", "calc(5em + 50px)", "calc(2 * 5em)", "calc(2 * (3em + 20px))",
		"calc(25% * (1 + 1))", "calc(20% * (1 + 1) + 20px)", "calc(100px",
		"max(100px)", "max(30%, 2em, 100px)", "max(-30%, -2em, 10em)", "calc(max(-1, 1, 2) * 50px)",
		"min(100px)", "min(100%, 20em, 100px)", "calc(min(4, 2) * 50px)",
		"calc(sqrt(4) * 50px)", "calc(pow(2, 2) * 25px)", "calc(hypot(2) * 50px)",
		"calc(hypot(3, 4) * 20px)", "calc(hypot(2px) * 50)", "calc(hypot(3px, 4px) * 20)",
		"calc(log(e) * 100px)", "calc(log(100, 10) * 50px)", "calc(exp(1) / e * 100px)",
		"abs(-100px)", "calc(abs(-100) * 1px)", "calc(sign(-100) * -100px)", "calc(sign(-100px) * -100px)",
		"calc(sqrt(16) * min(25px, 100%))",
		"clamp(calc(-infinity * 1px), 10em, calc(infinity * 1px))", "clamp(50px, 10em, 500px)",
		"clamp(100px, 2em, 500px)", "clamp(10px, 100em, 10em)", "clamp(10px, 100%, 10em)",
		"round(100.4px)", "round(145.4px, 100px)", "round(nearest, 100px)",
		"round(down, 195px, 100px)", "round(up, 5px, 100px)", "round(to-zero, 195px, 100px)",
		"mod(300px, 200px)", "calc(mod(300px, -200px) * -1)", "calc(mod(-300px, -200px) * -1)",
		"rem(300px, 200px)", "rem(300px, -200px)", "calc(rem(-300px, -200px) * -1)",
		"calc(sin(30deg) * 200px)", "calc(cos(60deg) * 200px)", "calc(tan(45deg) * 100px)",
		"calc(tan(calc(pi / 4)) * 100px)", "calc(sin(asin(0.5)) * 200px)", "calc(cos(acos(0.5)) * 200px)",
		"calc(tan(atan(1)) * 100px)", "calc(tan(atan2(1, 1)) * 100px)",
		"calc(100px * var(--one))", "calc(50% * var(--one))", "calc(100px * sqrt(var(--one)))",
	}
	for _, width := range widths {
		t.Run(width, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			div := divStyle(t, `<style>body { font-size: 10px; width: 200px }</style>
              <div style="--one: 1; height: 1px; width: `+width+`"></div>`)
			if got := usedLength(t, div, div.Width(), "width", 200); math.Abs(got-100) > 1e-9 {
				t.Errorf("width = %v, want 100", got)
			}
		})
	}
}

func TestCalcDivisionByZero(t *testing.T) {
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
	width := func(value string) float64 {
		div := divStyle(t, `<div style="font-size: 10px; height: 1px; width: `+value+`"></div>`)
		return usedLength(t, div, div.Width(), "width", 0)
	}
	for _, tt := range tests {
		testutil.AssertNoLogs(t)
		got, want := width(tt[0]), width(tt[1])
		if got != want && !(math.IsNaN(got) && math.IsNaN(want)) {
			t.Errorf("%s = %v, %s = %v", tt[0], got, tt[1], want)
		}
	}
}

func TestMathFunctionErrors(t *testing.T) {
	widths := []string{
		"calc", "(calc)", "calc(", "calc()", `calc("100px")`, "calc(100)", "calc(100px 100px)",
		"calc(100px, 100px)", "calc(100px * 100px)", "calc(100 * 100)", "calc(calc(100unknown))",
		"calc(0.1)", "calc(-1)", "min()", "min(10)", `min("10px")`, "min(10, 5px)",
		"calc(min(1, 5px) * 10px)", "max()", "max(10)", `max("10px")`, "max(10, 50px)",
		"calc(max(100, 5px) * 10px)", "calc(100* - max(56px, 1rem)", "clamp()", "clamp(10px)",
		"clamp(10px, 50px)", "clamp(10px, 50px, 100px, 200px)", `clamp(10px, "50px", 100px)`,
		"round()", "round(100)", "round(100, 10)", "round(nearest, 100, 10)", "round(100px, 10)",
		`round(100px, "10px")`, "round(nearest, 100px, 10)", "round(100px, 10px, 1)",
		"round(nearest, 100px, 10px, 1)", "round(unknown, 100px)", "round(unknown, 100px, 10px)",
		"mod()", "mod(10px)", "mod(100px, 10)", `mod(100px, "10px")`, "calc(mod(300px, 200) * -1)",
		"mod(100px, 10px, 1px)", "rem()", "rem(10px)", "rem(100px, 10)", `rem(100px, "10px")`,
		"calc(rem(300px, 200) * -1)", "rem(100px, 10px, 1px)",
		"sin()", "sin(10)", "sin(10%)", "sin(10deg)", "calc(sin(10) * 1)",
		"cos()", "cos(10)", "cos(10%)", "cos(10deg)", "calc(cos(10) * 1)",
		"tan()", "tan(10)", "tan(10%)", "tan(10deg)", "calc(tan(10) * 1)",
		"asin()", "asin(0)", "asin(0.5)", "asin(50deg)", "calc(sin(asin(50deg)) * 200px)",
		"calc(sin(asin(0.5)) * 200)", "calc(sin(asin(0.5, 2)) * 200px)", "calc(sin(asin(5)) * 200px)",
		"acos()", "acos(0)", "acos(0.5)", "acos(50deg)", "calc(cos(acos(50deg)) * 200px)",
		"calc(cos(acos(0.5)) * 200)", "calc(cos(acos(0.5, 2)) * 200px)", "calc(cos(acos(5)) * 200px)",
		"atan()", "atan(0)", "atan(0.5)", "atan(50deg)", "calc(tan(atan(50deg)) * 200px)",
		"calc(tan(atan(0.5)) * 200)", "calc(tan(atan(0.5, 2)) * 200px)",
		"atan2()", "atan2(0.5)", "atan2(0.5, 1)", "atan2(50deg, 1)",
		"calc(tan(atan2(50deg, 1)) * 200px)", "calc(tan(atan2(0.5, 1)) * 200)",
		"pow()", "pow(4, 3)", "pow(4px, 3)", "pow(4, 3, 4)",
		"sqrt()", "sqrt(4)", "sqrt(4px)", "sqrt(4, 2)", "hypoth()", "hypoth(3)", "hypoth(3, 4)",
		"log()", "log(10)", "log(10px)", "log(10, 10)", "log(10px, 10)", "log(10, 10, 10)",
		"exp()", "exp(10)", "exp(10px)", "exp(10, 10)", "exp(10px, 10)", "exp(10, 10, 10)",
		"abs()", "abs(10)", "abs(10px, 100)", "sign()", "sign(10)", "sign(10px)", "sign(10px, 10)",
	}
	for _, width := range widths {
		logs := logging.StartCapture()
		div := divStyle(t, `<style>body { font-size: 10px; width: 200px }</style>
          <div style="--one: 1; height: 1px; width: `+width+`"></div>`)
		div.Width()
		logs.Close()
		if lines := logs.Lines(); len(lines) != 1 {
			t.Errorf("width: %s: logs = %q, want one message", width, lines)
		}
	}
}

// computeAll reads every property of the styles of a document.
func computeAll(t *testing.T, source string) {
	t.Helper()
	doc := csstest.Parse(t, source)
	sf := csstest.StyleFor(t, doc, cascade.Options{})
	for _, el := range doc.Elements() {
		style := sf.Get(el, "")
		for id := 0; id < props.NumProps(); id++ {
			style.Get(props.PropID(id))
		}
		style.Strut()
	}
}

func TestMathWithPercentageAndFontUnitWarnsLikeItsParts(t *testing.T) {
	for _, p := range props.Table {
		if p.Validate == nil {
			continue
		}
		name := p.Name
		count := func(value string) int {
			logs := logging.StartCapture()
			computeAll(t, `<div style="`+name+`: `+value+`"></div>`)
			logs.Close()
			return len(logs.Lines())
		}
		mathLogs := count("calc(50% + 1em)")
		logs := count("50%")
		if logs == 0 {
			logs = count("1em")
		}
		if mathLogs != logs {
			t.Errorf("%s: %d messages with calc(), %d with its parts", name, mathLogs, logs)
		}
	}
}

func TestMathInStylesDoesNotFail(t *testing.T) {
	sources := []string{
		`<p style="line-height: calc(50% + 1em)">Hello</p>`,
		`<p style="line-height: calc(100% + 2px)">Hello</p>`,
		`<p style="line-height: calc(1.2em + 2px)">Hello</p>`,
		`<p style="line-height: calc(1.5 * 1)">Hello</p>`,
		`<div lang="en" style="hyphens: auto; hyphenate-limit-zone: calc(1em + 100%); width: 2em">absolute</div>`,
		`<div style="width: 10px; height: 10px; background: linear-gradient(blue calc(20% + 1em), red calc(80% + 1em))"></div>`,
		`<table style="margin: calc(1em + 10%)">`,
		`<table style="--spacing: 1em; --border-spacing: calc(var(--spacing) * 5);
		  border-spacing: var(--border-spacing) var(--border-spacing)"><tr><td>a`,
		`<table><tr><td style="vertical-align: calc(1em + 1%)">abc`,
		`<span style="vertical-align: calc(1em + 1%)">abc`,
	}
	for _, display := range []string{"block", "inline", "flex", "grid", "list", "list-item", "table",
		"table-row-group", "table-cell", "inline-block", "inline-table", "inline-flex", "inline-grid"} {
		sizes := "min-width: calc(50% + 1em); max-width: calc(50% + 1em); width: calc(50% + 1em); " +
			"min-height: calc(50% + 1em); max-height: calc(50% + 1em); height: calc(50% + 1em)"
		sources = append(sources, `<div style="display: `+display+`; `+sizes+`"><div style="`+sizes+`"></div></div>`)
	}
	for _, source := range sources {
		if strings.Contains(source, "display: list;") {
			// display: list is invalid and reported.
			logs := logging.StartCapture()
			computeAll(t, source)
			logs.Close()
			continue
		}
		t.Run("", func(t *testing.T) {
			testutil.AssertNoLogs(t)
			computeAll(t, source)
		})
	}
}
