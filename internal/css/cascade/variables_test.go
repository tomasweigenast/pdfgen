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

func TestPresentationalAttributesCascadeToStyles(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc := csstest.Parse(t, `<body marginwidth="12" marginheight="8" text="#123456" bgcolor="#abcdef"><div align="right"><font size="5" color="red" face="serif">x</font><hr width="30" size="4"><ol><li value="3">item</li></ol></div></body>`)
	sf := csstest.StyleFor(t, doc, cascade.Options{PresentationalHints: true})
	style := func(tag string, index int) *computed.Style {
		t.Helper()
		return sf.Get(csstest.Elements(doc, tag)[index], "")
	}
	body := style("body", 0)
	if body.MarginLeft() != props.Px(12) || body.MarginRight() != props.Px(12) || body.MarginTop() != props.Px(8) || body.MarginBottom() != props.Px(8) {
		t.Errorf("body margins = %v %v %v %v", body.MarginTop(), body.MarginRight(), body.MarginBottom(), body.MarginLeft())
	}
	if got := body.Color(); !slices.Equal(got.Coordinates, []float64{0x12 / 255.0, 0x34 / 255.0, 0x56 / 255.0}) {
		t.Errorf("body color = %v", got)
	}
	if got := body.BackgroundColor(); !slices.Equal(got.Coordinates, []float64{0xab / 255.0, 0xcd / 255.0, 0xef / 255.0}) {
		t.Errorf("body background-color = %v", got)
	}
	if got := style("div", 0).TextAlignAll(); got != "right" {
		t.Errorf("div text-align = %q, want right", got)
	}
	font := style("font", 0)
	if font.FontSize() != 24 || !slices.Equal(font.Color().Coordinates, []float64{1, 0, 0}) || font.FontFamily()[0] != "serif" {
		t.Errorf("font attributes gave size %v, color %v, family %v", font.FontSize(), font.Color(), font.FontFamily())
	}
	hr := style("hr", 0)
	if hr.Width() != props.Px(30) || hr.Height() != props.Px(2) {
		t.Errorf("hr dimensions = %v by %v", hr.Width(), hr.Height())
	}
	if got := style("li", 0).CounterReset(); len(got.Items) == 0 {
		t.Error("li value did not reset its list counter")
	}
}

func TestTableAndReplacedElementAttributesCascade(t *testing.T) {
	doc := csstest.Parse(t, `<table width="200" height="40" cellspacing="3" cellpadding="4" border="2" bordercolor="blue"><col width="25%"><tbody><tr align="center" height="30"><td width="50" height="20">cell</td></tr></tbody></table><img align="middle" hspace="5" vspace="6" width="40" height="30" border="1"><input type="text" width="10"><input type="image" width="12" height="8">`)
	sf := csstest.StyleFor(t, doc, cascade.Options{PresentationalHints: true})
	style := func(tag string, index int) *computed.Style {
		t.Helper()
		return sf.Get(csstest.Elements(doc, tag)[index], "")
	}
	table := style("table", 0)
	if table.Width() != props.Px(200) || table.Height() != props.Px(40) || table.BorderSpacing().X != 3 || table.BorderTopWidth() != 2 {
		t.Errorf("table dimensions or border = %v %v %v %v", table.Width(), table.Height(), table.BorderSpacing(), table.BorderTopWidth())
	}
	if got := style("td", 0); got.PaddingTop() != props.Px(4) || got.Width() != props.Px(50) || got.Height() != props.Px(20) || got.TextAlignAll() != "center" {
		t.Errorf("cell hints = padding %v, size %v by %v, align %v", got.PaddingTop(), got.Width(), got.Height(), got.TextAlignAll())
	}
	if got := style("col", 0).Width(); got.Dimension.Unit != "%" || got.Dimension.Value != 25 {
		t.Errorf("column width = %v", got)
	}
	img := style("img", 0)
	if img.VerticalAlign() != props.LengthKeyword("middle") || img.MarginLeft() != props.Px(5) || img.MarginTop() != props.Px(6) || img.Width() != props.Px(40) || img.BorderTopStyle() != "solid" {
		t.Errorf("image hints = align %v, margins %v/%v, width %v, border %v", img.VerticalAlign(), img.MarginLeft(), img.MarginTop(), img.Width(), img.BorderTopStyle())
	}
	if style("input", 0).Width() == props.Px(10) {
		t.Error("non-image input unexpectedly accepted a width hint")
	}
	if got := style("input", 1); got.Width() != props.Px(12) || got.Height() != props.Px(8) {
		t.Errorf("image input dimensions = %v by %v", got.Width(), got.Height())
	}
}

func TestImportanceLayersAndGlobalValues(t *testing.T) {
	tests := []struct {
		name, css, tag string
		wantWidth      props.Length
		wantColor      string
	}{
		{"important specificity", `div { color: red !important } #target { color: blue !important }`, "div", props.Length{}, "blue"},
		{"important layer order", `@layer first { p { width: 20px !important } } @layer second { p { width: 10px !important } }`, "p", props.Px(10), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style, _ := styled(t, `<style>`+tt.css+`</style><div id="target"></div><p></p>`)
			got := style(tt.tag, 0, "")
			if tt.wantWidth != (props.Length{}) && got.Width() != tt.wantWidth {
				t.Errorf("width = %v, want %v", got.Width(), tt.wantWidth)
			}
			if tt.wantColor != "" && got.Color().Coordinates[2] != 1 {
				t.Errorf("color = %v, want blue", got.Color())
			}
		})
	}
	parent, child := familyStyles(t)
	if child.Color().Coordinates[0] != parent.Color().Coordinates[0] || child.Width() != props.Px(10) {
		t.Errorf("inherited values = color %v, width %v", child.Color(), child.Width())
	}
}

func TestStylesheetOriginsAreAvailable(t *testing.T) {
	ua := cascade.UAStylesheets(false)
	forms := cascade.UAStylesheets(true)
	if len(ua) == 0 || len(forms) < len(ua) {
		t.Fatalf("user agent stylesheets: without forms %d, with forms %d", len(ua), len(forms))
	}
	doc := csstest.Parse(t, `<style>p { color: red }</style><p>text</p>`)
	sf := csstest.StyleFor(t, doc, cascade.Options{})
	sheets := sf.Sheets()
	if len(sheets) < 2 {
		t.Fatalf("got %d stylesheet entries, want user agent and author stylesheets", len(sheets))
	}
	if sheets[len(sheets)-1].Origin != cascade.Author {
		t.Errorf("last stylesheet origin = %v, want author", sheets[len(sheets)-1].Origin)
	}
}

func TestInternalTableStylesResetInapplicableSpacing(t *testing.T) {
	doc := csstest.Parse(t, `<style>table { border-collapse: collapse; padding: 8px } .cell { display: table-cell; margin: 9px }</style><table><tr><td class="cell">x</td></tr></table>`)
	sf := csstest.StyleFor(t, doc, cascade.Options{})
	table := sf.Get(csstest.Elements(doc, "table")[0], "")
	if table.PaddingTop() != props.Px(0) || table.PaddingLeft() != props.Px(0) {
		t.Errorf("collapsed table padding = %v %v", table.PaddingTop(), table.PaddingLeft())
	}
	cell := sf.Get(csstest.Elements(doc, "td")[0], "")
	if cell.MarginTop() != props.Px(0) || cell.MarginLeft() != props.Px(0) {
		t.Errorf("table-cell margins = %v %v", cell.MarginTop(), cell.MarginLeft())
	}
	if got := sf.Get(csstest.Elements(doc, "td")[0], "before"); got != nil {
		t.Errorf("undeclared pseudo-element style = %v, want nil", got)
	}
}

func TestColorProfileRulesWithoutSourcesAreIgnored(t *testing.T) {
	logs := logging.StartCapture()
	defer logs.Close()
	profiles := map[string]*cascade.ColorProfile{}
	cascade.ParseStylesheet(`@color-profile invalid { src: url(profile.icc) } @color-profile --display { components: red green blue; rendering-intent: saturation; unknown: value }`, "https://example.com/style.css", &cascade.Environment{ColorProfiles: profiles})
	if len(profiles) != 0 {
		t.Fatalf("profiles = %v, want none without a valid source", profiles)
	}
	lines := logs.Lines()
	if len(lines) != 3 || !strings.Contains(strings.Join(lines, "\n"), "Invalid color profile name") || !strings.Contains(strings.Join(lines, "\n"), "descriptor not supported") || !strings.Contains(strings.Join(lines, "\n"), "No source for profile") {
		t.Errorf("color profile warnings = %q", lines)
	}
}

func familyStyles(t *testing.T) (*computed.Style, *computed.Style) {
	t.Helper()
	style, _ := styled(t, `<style>p { color: red; width: 10px } span { color: inherit; width: inherit }</style><p><span></span></p>`)
	return style("p", 0, ""), style("span", 0, "")
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
