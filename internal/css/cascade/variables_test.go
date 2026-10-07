package cascade_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

// styled computes the styles of a document and returns a function giving
// the style of the n-th element with a tag name.
func styled(t *testing.T, source string) (func(tag string, n int, pseudo string) *computed.Style, *html.Document) {
	t.Helper()
	doc := csstest.Parse(t, source)
	sf := csstest.StyleFor(t, doc, cascade.Options{})
	return func(tag string, n int, pseudo string) *computed.Style {
		t.Helper()
		elements := csstest.Elements(doc, tag)
		if n >= len(elements) {
			t.Fatalf("no %s element number %d", tag, n)
		}
		return sf.Get(elements[n], pseudo)
	}, doc
}

func widthOf(t *testing.T, source, tag string) props.Length {
	t.Helper()
	style, _ := styled(t, source)
	return style(tag, 0, "").Width()
}

func TestVariableWidths(t *testing.T) {
	tests := []struct {
		name, css string
		tag       string
		want      props.Length
	}{
		{"simple", "p { --var: 10px; width: var(--var) }", "p", props.Px(10)},
		{"not computed", "p { --var: 1rem; width: var(--var) }", "p", props.Px(16)},
		{"inherit", "html { --var: 10px } p { width: var(--var) }", "p", props.Px(10)},
		{"inherit override", "html { --var: 20px } p { width: var(--var); --var: 10px }", "p", props.Px(10)},
		{"default unknown", "p { width: var(--x, 10px) }", "p", props.Px(10)},
		{"default var", "p { --var: 10px; width: var(--x, var(--var)) }", "p", props.Px(10)},
		{"case sensitive", "html { --var: 20px } body { --VAR: 10px } p { width: var(--VAR) }", "p", props.Px(10)},
		{"chain", "html { --foo: 10px } body { --var: var(--foo) } p { width: var(--var) }", "p", props.Px(10)},
		{"chain root", "html { --var2: 10px; --var1: var(--var2); width: var(--var1) }", "html", props.Px(10)},
		{"initial", "html { --var: initial } p { width: var(--var) }", "p", props.LengthKeyword("auto")},
		{"initial default", "p { --var: initial; width: var(--var, 10px) }", "p", props.LengthKeyword("auto")},
		{"initial default var", "p { --var: initial; width: var(--var, var(--var)) }", "p", props.LengthKeyword("auto")},
		{"math", "div { --one: 1; width: calc(100px * var(--one)) }", "div", props.Px(100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			got := widthOf(t, "<style>"+tt.css+"</style><div></div><p></p>", tt.tag)
			if got != tt.want {
				t.Errorf("width = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVariableDoubleChainInGradient(t *testing.T) {
	testutil.AssertNoLogs(t)
	for _, css := range []string{
		"html { --foo: red } body { --var: var(--foo), var(--foo) } div { background-image: linear-gradient(var(--var)) }",
		"body { --var: red } div { background-image: linear-gradient(var(--var), var(--var)) }",
		"body { --var: 255 0 0 } div { background-image: linear-gradient(rgba(var(--var)), red) }",
	} {
		style, _ := styled(t, "<style>"+css+"</style><div></div>")
		images := style("div", 0, "").BackgroundImage()
		if len(images) != 1 || images[0].Linear == nil {
			t.Fatalf("%s: background-image = %+v", css, images)
		}
		for i, stop := range images[0].Linear.ColorStops[:2] {
			if got := rgba(stop.Color); got != [4]float64{1, 0, 0, 1} {
				t.Errorf("%s: color %d = %v, want red", css, i, got)
			}
		}
	}
}

func TestVariableCyclesAndMissingValuesDoNotFail(t *testing.T) {
	for _, css := range []string{
		"html { --var1: var(--var1) }",
		"html { --var1: var(--var1); width: var(--var1) }",
		"html { --var1: var(--var2); --var2: var(--var1); padding: var(--var1) }",
		"html { --var1: var(--var-missing); width: var(--var1) }",
		"html { --var1: var(--var-missing); font: var(--var1) }",
	} {
		logs := logging.StartCapture()
		style, _ := styled(t, "<style>"+css+"</style>a")
		html := style("html", 0, "")
		html.Width()
		html.PaddingTop()
		html.FontSize()
		logs.Close()
		if got := html.Width(); got != props.LengthKeyword("auto") {
			t.Errorf("%s: width = %v, want auto", css, got)
		}
	}
}

func TestVariableInShorthands(t *testing.T) {
	testutil.AssertNoLogs(t)
	tests := []struct {
		css     string
		margins [4]float64
		borders [4]props.Pixels
	}{
		{"html { --var: 10px } div { margin: 0 0 0 var(--var) }", [4]float64{0, 0, 0, 10}, [4]props.Pixels{}},
		{"html { --var1: 10px; --var2: 20px } div { margin: var(--var2) 0 0 var(--var1) }", [4]float64{20, 0, 0, 10}, [4]props.Pixels{}},
		{"html { --var: 1px solid blue } div { border: var(--var) }", [4]float64{}, [4]props.Pixels{1, 1, 1, 1}},
		{"html { --var: 1px solid blue } div { border-top: var(--var) }", [4]float64{}, [4]props.Pixels{1, 0, 0, 0}},
		{"html { --var: 1px solid } div { border: blue var(--var) }", [4]float64{}, [4]props.Pixels{1, 1, 1, 1}},
	}
	for _, tt := range tests {
		style, _ := styled(t, "<style>"+tt.css+"</style><div></div>")
		div := style("div", 0, "")
		margins := [4]float64{div.MarginTop().Dimension.Value, div.MarginRight().Dimension.Value, div.MarginBottom().Dimension.Value, div.MarginLeft().Dimension.Value}
		borders := [4]props.Pixels{div.BorderTopWidth(), div.BorderRightWidth(), div.BorderBottomWidth(), div.BorderLeftWidth()}
		if margins != tt.margins || borders != tt.borders {
			t.Errorf("%s: margins %v, borders %v; want %v, %v", tt.css, margins, borders, tt.margins, tt.borders)
		}
	}
}

func TestInvalidVariableInShorthands(t *testing.T) {
	tests := []struct {
		css     string
		message string
	}{
		{"html { --var: blue } div { margin: 0 0 0 var(--var) }", "invalid value"},
		{"html { --var: 1px solid blue } div { border: blue var(--var) }", "multiple color values"},
	}
	for _, tt := range tests {
		logs := logging.StartCapture()
		style, _ := styled(t, "<style>"+tt.css+"</style><div></div>")
		div := style("div", 0, "")
		values := []float64{
			div.MarginTop().Dimension.Value, div.MarginLeft().Dimension.Value,
			float64(div.BorderTopWidth()), float64(div.BorderLeftWidth()),
		}
		logs.Close()
		for _, v := range values {
			if v != 0 {
				t.Errorf("%s: values %v, want zeros", tt.css, values)
				break
			}
		}
		if lines := logs.Lines(); len(lines) == 0 || !strings.Contains(lines[0], tt.message) {
			t.Errorf("%s: logs = %q", tt.css, lines)
		}
	}
}

func TestVariableInBackgroundShorthand(t *testing.T) {
	tests := []struct {
		variable, background string
		valid                bool
	}{
		{"blue", "var(--v)", true},
		{"padding-box url(pattern.png)", "var(--v)", true},
		{"padding-box url(pattern.png)", "white var(--v) center", true},
		{"100%", "url(pattern.png) var(--v) var(--v) / var(--v) var(--v)", true},
		{"left / 100%", "url(pattern.png) top var(--v) 100%", true},
		{"invalid", "var(--v)", false},
		{"blue", "var(--v) var(--v)", false},
		{"100%", "url(pattern.png) var(--v) var(--v) var(--v)", false},
	}
	for _, tt := range tests {
		logs := logging.StartCapture()
		style, _ := styled(t, "<style>html { --v: "+tt.variable+" } div { background: "+tt.background+" }</style><div></div>")
		div := style("div", 0, "")
		for id := 0; id < props.NumProps(); id++ {
			if name := props.PropID(id).Name(); strings.HasPrefix(name, "background-") {
				div.Get(props.PropID(id))
			}
		}
		logs.Close()
		lines := logs.Lines()
		if tt.valid && len(lines) != 0 {
			t.Errorf("%s / %s: logs = %q", tt.variable, tt.background, lines)
		}
		if !tt.valid && (len(lines) != 1 || !strings.Contains(lines[0], "invalid value")) {
			t.Errorf("%s / %s: logs = %q, want one invalid value", tt.variable, tt.background, lines)
		}
	}
}

func TestVariableFallbackForEveryProperty(t *testing.T) {
	for id := 0; id < props.NumProps(); id++ {
		name := props.PropID(id).Name()
		logs := logging.StartCapture()
		style, _ := styled(t, "<style>div { --var: improperValue; "+name+": var(--var) }</style><div></div>")
		div := style("div", 0, "")
		for other := 0; other < props.NumProps(); other++ {
			div.Get(props.PropID(other))
		}
		logs.Close()
	}
}

func TestVariableInContentList(t *testing.T) {
	testutil.AssertNoLogs(t)
	style, _ := styled(t, `<style>
        :root { --var: "Page " counter(page) "/" counter(pages) }
        div::before { content: var(--var) }
      </style><div></div>`)
	content := style("div", 0, "before").Content()
	var types []string
	for _, item := range content.Items {
		types = append(types, item.Type)
	}
	if want := []string{"string", "counter()", "string", "counter()"}; !slices.Equal(types, want) {
		t.Errorf("content = %+v", content)
	}
}

func TestVariableInDisplay(t *testing.T) {
	tests := [][2]string{
		{"inline", "var(--var)"},
		{"inline-block", "var(--var)"},
		{"inline flow", "var(--var)"},
		{"inline", "var(--var) flow"},
		{"flow", "inline var(--var)"},
	}
	for _, tt := range tests {
		testutil.AssertNoLogs(t)
		style, _ := styled(t, "<style>html { --var: "+tt[0]+" } div { display: "+tt[1]+" }</style><section><div></div></section>")
		if got := style("div", 0, "").Display(); got[0] != "inline" {
			t.Errorf("%v: display = %v, want inline", tt, got)
		}
	}
}

func TestVariableInFontFamily(t *testing.T) {
	tests := []struct {
		variable, font string
		want           props.Strings
	}{
		{"testfont", "var(--var)", props.Strings{"testfont"}},
		{`"testfont"`, "var(--var)", props.Strings{"testfont"}},
		{"testfont", "var(--var), monospace", props.Strings{"testfont", "monospace"}},
		{"testfont, monospace", "var(--var)", props.Strings{"testfont", "monospace"}},
		{"monospace", "testfont, var(--var)", props.Strings{"testfont", "monospace"}},
	}
	for _, tt := range tests {
		testutil.AssertNoLogs(t)
		style, _ := styled(t, "<style>html { font-size: 2px; --var: "+tt.variable+" } div { font-family: "+tt.font+" }</style><div>aa</div>")
		if got := style("div", 0, "").FontFamily(); !slices.Equal(got, tt.want) {
			t.Errorf("%s / %s: font-family = %q", tt.variable, tt.font, got)
		}
	}
}

func counterOf(t *testing.T, item props.ContentItem) (string, string) {
	t.Helper()
	if item.Type != "counter()" || item.Style == nil {
		t.Fatalf("item = %+v, want counter()", item)
	}
	return item.Args[0], item.Style.Name
}

func TestVariableInFunctions(t *testing.T) {
	sources := []string{
		`html { --var: title } h1 { counter-increment: var(--var) } div::before { content: counter(var(--var)) }`,
		`html { --name: title; --counter: title, upper-roman } h1 { counter-increment: var(--name) } div::before { content: counter(var(--counter)) }`,
		`html { --name: title; --counter: var(--name), upper-roman } h1 { counter-increment: var(--name) } div::before { content: counter(var(--counter)) }`,
		`html { --name: title; --counter: counter(var(--name), upper-roman) } h1 { counter-increment: var(--name) } div::before { content: var(--counter) }`,
	}
	wantStyles := []string{"decimal", "upper-roman", "upper-roman", "upper-roman"}
	for i, css := range sources {
		testutil.AssertNoLogs(t)
		style, _ := styled(t, "<style>"+css+"</style><section><h1></h1><div></div></section>")
		if got := style("h1", 0, "").CounterIncrement(); len(got.Items) != 1 || got.Items[0] != (props.CounterOp{Name: "title", Value: 1}) {
			t.Errorf("%s: counter-increment = %+v", css, got)
		}
		content := style("div", 0, "before").Content()
		if len(content.Items) != 1 {
			t.Fatalf("%s: content = %+v", css, content)
		}
		if name, counterStyle := counterOf(t, content.Items[0]); name != "title" || counterStyle != wantStyles[i] {
			t.Errorf("%s: counter(%s, %s)", css, name, counterStyle)
		}
	}

	testutil.AssertNoLogs(t)
	style, _ := styled(t, `<style>html { --name: title; --counter: var(--name), upper-roman }
        div::before { content: counter(var(--counter)) }</style>
        <div style="--counter: var(--name), lower-roman"></div>`)
	if _, counterStyle := counterOf(t, style("div", 0, "before").Content().Items[0]); counterStyle != "lower-roman" {
		t.Errorf("overridden counter style = %s", counterStyle)
	}
}

func TestVariableInFunctionMissing(t *testing.T) {
	logs := logging.StartCapture()
	style, _ := styled(t, `<style>
        h1 { counter-increment: var(--var) }
        div::before { content: counter(var(--var)) }
      </style><section><h1></h1><div></div><h1></h1><div></div></section>`)
	for i := 0; i < 2; i++ {
		style("h1", i, "").CounterIncrement()
		style("div", i, "before").Content()
	}
	logs.Close()
	lines := logs.Lines()
	if len(lines) != 2 || !strings.Contains(lines[0], "no value") || !strings.Contains(lines[1], "invalid value") {
		t.Errorf("logs = %q", lines)
	}
	if got := style("div", 0, "before").Content(); got.Keyword != "normal" {
		t.Errorf("content = %+v, want the initial normal", got)
	}
}

func TestVariableAndFunctionInTargetCounter(t *testing.T) {
	testutil.AssertNoLogs(t)
	style, _ := styled(t, `<style>
        html { --counter-name: page }
        a::after { content: target-counter(attr(href), var(--counter-name)) }
      </style><a id="link" href="#link"></a>`)
	content := style("a", 0, "after").Content()
	if len(content.Items) != 1 || content.Items[0].Type != "target-counter()" {
		t.Errorf("content = %+v", content)
	}
}
