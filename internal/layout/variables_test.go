package layout_test

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// renderedBody lays out a one-page document, which must not log anything,
// and returns its html and body boxes.
func renderedBody(t *testing.T, source string) (html, body boxes.Box) {
	t.Helper()
	return pageBody(t, onlyPage(t, renderPages(t, source)))
}

// renderedBodyLogging is renderedBody for documents that are expected to log
// warnings, which it returns.
func renderedBodyLogging(t *testing.T, source string) (body boxes.Box, logs *logging.Capture) {
	t.Helper()
	logs = testutil.CaptureLogs(t)
	_, body = pageBody(t, onlyPage(t, boxtest.RenderPages(t, source)))
	return body, logs
}

// margins returns the margins of a box, clockwise from the top.
func margins(box boxes.Box) [4]float64 {
	f := box.Base()
	return [4]float64{f.MarginTop.V, f.MarginRight.V, f.MarginBottom.V, f.MarginLeft.V}
}

// borderWidths returns the border widths of a box, clockwise from the top.
func borderWidths(box boxes.Box) [4]float64 {
	f := box.Base()
	return [4]float64{f.BorderTopWidth, f.BorderRightWidth, f.BorderBottomWidth, f.BorderLeftWidth}
}

// textBoxOf returns the text box that is the only child of a box.
func textBoxOf(t *testing.T, box boxes.Box) *boxes.TextBox {
	t.Helper()
	text, ok := child(t, box).(*boxes.TextBox)
	if !ok {
		t.Fatalf("%v does not hold a text box", box)
	}
	return text
}

func TestVariablesSetTheWidthOfBoxes(t *testing.T) {
	tests := []struct {
		name, css string
		width     float64
	}{
		{"simple", "p { --var: 10px; width: var(--var) }", 10},
		{"not computed", "p { --var: 1rem; width: var(--var) }", 16},
		{"inherit", "html { --var: 10px } p { width: var(--var) }", 10},
		{"inherit override", "html { --var: 20px } p { width: var(--var); --var: 10px }", 10},
		{"default unknown", "p { width: var(--x, 10px) }", 10},
		{"default var", "p { --var: 10px; width: var(--x, var(--var)) }", 10},
		{"case sensitive", "html { --var: 20px } body { --VAR: 10px } p { width: var(--VAR) }", 10},
		{"chain", "html { --foo: 10px } body { --var: var(--foo) } p { width: var(--var) }", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, body := renderedBody(t, "<style>"+tt.css+"</style><p></p>")
			if got := child(t, body).Base().Width.V; got != tt.width {
				t.Errorf("width = %v, want %v", got, tt.width)
			}
		})
	}
}

func TestVariablesChainInTheRootElement(t *testing.T) {
	html, _ := renderedBody(t, "<style>html { --var2: 10px; --var1: var(--var2); width: var(--var1) }</style>")
	if got := html.Base().Width.V; got != 10 {
		t.Errorf("html width = %v, want 10", got)
	}
}

func TestVariablesWithCyclesAndMissingValuesLayOut(t *testing.T) {
	for _, css := range []string{
		"html { --var1: var(--var1) }",
		"html { --var1: var(--var1); width: var(--var1) }",
		"html { --var1: var(--var2); --var2: var(--var1); padding: var(--var1) }",
		"html { --var1: var(--var-missing); width: var(--var1) }",
		"html { --var1: var(--var-missing); font: var(--var1) }",
	} {
		t.Run(css, func(t *testing.T) {
			testutil.CaptureLogs(t)
			pages := boxtest.RenderPages(t, "<style>"+css+"</style>a")
			onlyPage(t, pages)
		})
	}
}

func TestVariablesInShorthandsSetBoxEdges(t *testing.T) {
	tests := []struct {
		name, css string
		margins   [4]float64
		borders   [4]float64
	}{
		{"margin", "html { --var: 10px } div { margin: 0 0 0 var(--var) }", [4]float64{0, 0, 0, 10}, [4]float64{}},
		{"margin multiple", "html { --var1: 10px; --var2: 20px } div { margin: var(--var2) 0 0 var(--var1) }", [4]float64{20, 0, 0, 10}, [4]float64{}},
		{"border", "html { --var: 1px solid blue } div { border: var(--var) }", [4]float64{}, [4]float64{1, 1, 1, 1}},
		{"border side", "html { --var: 1px solid blue } div { border-top: var(--var) }", [4]float64{}, [4]float64{1, 0, 0, 0}},
		{"border mixed", "html { --var: 1px solid } div { border: blue var(--var) }", [4]float64{}, [4]float64{1, 1, 1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, body := renderedBody(t, "<style>"+tt.css+"</style><div></div>")
			div := child(t, body)
			if got := margins(div); got != tt.margins {
				t.Errorf("margins = %v, want %v", got, tt.margins)
			}
			if got := borderWidths(div); got != tt.borders {
				t.Errorf("border widths = %v, want %v", got, tt.borders)
			}
		})
	}
}

func TestInvalidVariablesInShorthandsAreIgnored(t *testing.T) {
	tests := []struct {
		name, css string
		message   string
		single    bool
	}{
		{"margin", "html { --var: blue } div { margin: 0 0 0 var(--var) }", "invalid value", true},
		{"border mixed", "html { --var: 1px solid blue } div { border: blue var(--var) }", "multiple color values", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, logs := renderedBodyLogging(t, "<style>"+tt.css+"</style><div></div>")
			div := child(t, body)
			if got := margins(div); got != [4]float64{} {
				t.Errorf("margins = %v, want none", got)
			}
			if got := borderWidths(div); got != [4]float64{} {
				t.Errorf("border widths = %v, want none", got)
			}
			lines := logs.Lines()
			if len(lines) == 0 || !strings.Contains(lines[0], tt.message) || tt.single && len(lines) > 1 {
				t.Errorf("logs = %q, want a warning about %q (only one: %v)", lines, tt.message, tt.single)
			}
		})
	}
}

func TestVariablesInBackgroundShorthand(t *testing.T) {
	tests := []struct{ value, background string }{
		{"blue", "var(--v)"},
		{"padding-box url(pattern.png)", "var(--v)"},
		{"padding-box url(pattern.png)", "white var(--v) center"},
		{"100%", "url(pattern.png) var(--v) var(--v) / var(--v) var(--v)"},
		{"left / 100%", "url(pattern.png) top var(--v) 100%"},
	}
	for _, tt := range tests {
		t.Run(tt.background, func(t *testing.T) {
			renderedBody(t, `<style>html { --v: `+tt.value+` } div { background: `+tt.background+` }</style><div></div>`)
		})
	}
}

func TestInvalidVariablesInBackgroundShorthandAreIgnored(t *testing.T) {
	tests := []struct{ value, background string }{
		{"invalid", "var(--v)"},
		{"blue", "var(--v) var(--v)"},
		{"100%", "url(pattern.png) var(--v) var(--v) var(--v)"},
	}
	for _, tt := range tests {
		t.Run(tt.background, func(t *testing.T) {
			_, logs := renderedBodyLogging(t, `<style>html { --v: `+tt.value+` } div { background: `+tt.background+` }</style><div></div>`)
			lines := logs.Lines()
			if len(lines) != 1 || !strings.Contains(lines[0], "invalid value") {
				t.Errorf("logs = %q, want one invalid value warning", lines)
			}
		})
	}
}

func TestInitialVariablesLeaveTheWidthAuto(t *testing.T) {
	tests := []struct{ name, css string }{
		{"initial", "html { --var: initial } p { width: var(--var) }"},
		{"initial default", "p { --var: initial; width: var(--var, 10px) }"},
		{"initial default var", "p { --var: initial; width: var(--var, var(--var)) }"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, body := renderedBody(t, "<style>"+tt.css+"</style><p></p>")
			if p := child(t, body); p.Base().Width.V != body.Base().Width.V {
				t.Errorf("paragraph width = %v, body width = %v", p.Base().Width.V, body.Base().Width.V)
			}
		})
	}
}

func TestVariableFallbackForEveryProperty(t *testing.T) {
	for id := 0; id < props.NumProps(); id++ {
		name := props.PropID(id).Name()
		t.Run(name, func(t *testing.T) {
			testutil.CaptureLogs(t)
			boxtest.RenderPages(t, `<style>div { --var: improperValue; `+name+`: var(--var) }</style><div></div>`)
		})
	}
}

func TestVariableListInContent(t *testing.T) {
	_, body := renderedBody(t, `<style>
        :root { --var: "Page " counter(page) "/" counter(pages) }
        div::before { content: var(--var) }
      </style><div></div>`)
	line := child(t, child(t, body))
	if got := textBoxOf(t, child(t, line)).Text; got != "Page 1/1" {
		t.Errorf("text = %q, want %q", got, "Page 1/1")
	}
}

func TestVariableListInDisplay(t *testing.T) {
	tests := []struct{ value, display string }{
		{"inline", "var(--var)"},
		{"inline-block", "var(--var)"},
		{"inline flow", "var(--var)"},
		{"inline", "var(--var) flow"},
		{"flow", "inline var(--var)"},
	}
	for _, tt := range tests {
		t.Run(tt.value+" "+tt.display, func(t *testing.T) {
			_, body := renderedBody(t, `<style>html { --var: `+tt.value+` } div { display: `+tt.display+` }</style>
              <section><div></div></section>`)
			if got := child(t, child(t, body)); !isLine(got) {
				t.Errorf("child of section is %T, want *boxes.LineBox", got)
			}
		})
	}
}

func TestVariableListInFontFamily(t *testing.T) {
	tests := []struct{ value, font string }{
		{"testfont", "var(--var)"},
		{`"testfont"`, "var(--var)"},
		{"testfont", "var(--var), monospace"},
		{"testfont, monospace", "var(--var)"},
		{"monospace", "testfont, var(--var)"},
	}
	for _, tt := range tests {
		t.Run(tt.value+" "+tt.font, func(t *testing.T) {
			_, body := renderedBody(t, `<style>html { font-size: 2px; --var: `+tt.value+` } div { font-family: `+tt.font+` }</style>
              <div>aa</div>`)
			line := child(t, child(t, body))
			if got := textBoxOf(t, line).Width.V; got != 4 {
				t.Errorf("text width = %v, want 4", got)
			}
		})
	}
}

func isLine(box boxes.Box) bool {
	_, ok := box.(*boxes.LineBox)
	return ok
}

// counterTexts returns the texts that the ::before pseudo-element of the
// divisions of a section generates. The section must have n children.
func counterTexts(t *testing.T, section boxes.Box, n int) []string {
	t.Helper()
	var texts []string
	for _, c := range children(t, section, n) {
		if tagOf(c) != "div" {
			continue
		}
		if len(boxes.ChildrenOf(c)) == 0 {
			texts = append(texts, "")
			continue
		}
		line := child(t, c)
		texts = append(texts, textBoxOf(t, child(t, line)).Text)
	}
	return texts
}

func TestVariablesInCounterFunctions(t *testing.T) {
	tests := []struct {
		name, css, extra string
		want             []string
	}{
		{
			"variable as name",
			`html { --var: title } h1 { counter-increment: var(--var) } div::before { content: counter(var(--var)) }`,
			"", []string{"1", "2"},
		},
		{
			"variable with several values",
			`html { --name: title; --counter: title, upper-roman } h1 { counter-increment: var(--name) } div::before { content: counter(var(--counter)) }`,
			` style="--counter: var(--name), lower-roman"`, []string{"I", "II", "iii"},
		},
		{
			"variable in variable",
			`html { --name: title; --counter: var(--name), upper-roman } h1 { counter-increment: var(--name) } div::before { content: counter(var(--counter)) }`,
			` style="--counter: var(--name), lower-roman"`, []string{"I", "II", "iii"},
		},
		{
			"function in variable",
			`html { --name: title; --counter: counter(var(--name), upper-roman) } h1 { counter-increment: var(--name) } div::before { content: var(--counter) }`,
			` style="--counter: counter(var(--name), lower-roman)"`, []string{"I", "II", "iii"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := `<h1></h1><div></div><h1></h1><div></div>`
			if len(tt.want) == 3 {
				section += `<h1></h1><div` + tt.extra + `></div>`
			}
			_, body := renderedBody(t, "<style>"+tt.css+"</style><section>"+section+"</section>")
			got := counterTexts(t, child(t, body), 2*len(tt.want))
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("counters = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVariablesRepeatedInGradientFunctionsKeepTheirColor(t *testing.T) {
	tests := []struct {
		name, css string
		stops     int
	}{
		{"double chain", `html { --foo: red } body { --var: var(--foo), var(--foo) } div { background-image: linear-gradient(var(--var)) }`, 2},
		{"same variable", `body { --var: red } div { background-image: linear-gradient(var(--var), var(--var)) }`, 2},
		{"nested function", `body { --var: 255 0 0 } div { background-image: linear-gradient(rgba(var(--var))) }`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, body := renderedBody(t, "<style>"+tt.css+"</style><div></div>")
			images := child(t, body).Base().Style.BackgroundImage()
			if len(images) != 1 || images[0].Linear == nil {
				t.Fatalf("background-image = %+v", images)
			}
			if len(images[0].Linear.ColorStops) < tt.stops {
				t.Fatalf("color stops = %+v, want at least %d", images[0].Linear.ColorStops, tt.stops)
			}
			for i, stop := range images[0].Linear.ColorStops[:tt.stops] {
				c := stop.Color
				if got := [4]float64{c.Coordinates[0], c.Coordinates[1], c.Coordinates[2], c.Alpha}; got != [4]float64{1, 0, 0, 1} {
					t.Errorf("color %d = %v, want red", i, got)
				}
			}
		})
	}
}

func TestVariableInCounterFunctionMissing(t *testing.T) {
	body, logs := renderedBodyLogging(t, `<style>
        h1 { counter-increment: var(--var) }
        div::before { content: counter(var(--var)) }
      </style><section><h1></h1><div></div><h1></h1><div></div></section>`)
	lines := logs.Lines()
	if len(lines) != 2 || !strings.Contains(lines[0], "no value") || !strings.Contains(lines[1], "invalid value") {
		t.Errorf("logs = %q", lines)
	}
	for i, text := range counterTexts(t, child(t, body), 4) {
		if text != "" {
			t.Errorf("division %d has the text %q, want no content", i+1, text)
		}
	}
}

func TestVariableAndFunctionInTargetCounter(t *testing.T) {
	_, body := renderedBody(t, `<style>
        html { --counter-name: page }
        a::after { content: target-counter(attr(href), var(--counter-name)) }
      </style><a id="link" href="#link"></a>
    `)
	line := child(t, body)
	a := children(t, line, 2)[0]
	after := child(t, a)
	if got := textBoxOf(t, after).Text; got != "1" {
		t.Errorf("text = %q, want 1", got)
	}
}
