package cascade_test

import (
	"math"
	"os"
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

func parseFile(t *testing.T, name string) *html.Document {
	t.Helper()
	data, err := os.ReadFile(testutil.ResourcePath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(string(data)), "", testutil.ResourceURL(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func loadFile(t *testing.T, name string, env *cascade.Environment) *cascade.Stylesheet {
	t.Helper()
	sheet, err := cascade.LoadStylesheet(testutil.ResourceURL(t, name), env, false)
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func rgba(c props.Color) [4]float64 {
	return [4]float64{c.Coordinates[0], c.Coordinates[1], c.Coordinates[2], c.Alpha}
}

func TestFindStylesheetsInTreeOrder(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc := parseFile(t, "doc1.html")
	sheets := cascade.FindStylesheets(doc, nil)
	if len(sheets) != 2 {
		t.Fatalf("found %d stylesheets, want 2", len(sheets))
	}
	var names []string
	for _, s := range sheets {
		name := s.BaseURL[strings.LastIndex(s.BaseURL, "/")+1:]
		names = append(names, name[strings.LastIndex(name, ",")+1:])
	}
	if want := []string{"a%7Bcolor%3AcurrentColor%7D", "doc1.html"}; !slices.Equal(names, want) {
		t.Errorf("stylesheets = %q, want %q", names, want)
	}
	rules := 0
	for _, s := range sheets {
		rules += cascade.SelectorCount(s) + len(s.PageRules())
	}
	if rules != 10 {
		t.Errorf("found %d rules, want 10", rules)
	}
}

func TestAnnotateDocument(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc := parseFile(t, "doc1.html")
	env := csstest.Environment()
	sf, err := cascade.ComputeAll(doc, cascade.Options{
		Environment:     env,
		UAStylesheets:   []*cascade.Stylesheet{loadFile(t, "mini_ua.css", env)},
		UserStylesheets: []*cascade.Stylesheet{loadFile(t, "user.css", env)},
	})
	if err != nil {
		t.Fatal(err)
	}
	get := func(name string, index int) *html.Element { return csstest.Elements(doc, name)[index] }
	h1 := sf.Get(get("h1", 0), "")
	p := sf.Get(get("p", 0), "")
	ul := sf.Get(get("ul", 0), "")
	li0 := sf.Get(get("li", 0), "")
	div := sf.Get(get("div", 0), "")
	after := sf.Get(get("a", 0), "after")
	a := sf.Get(get("a", 0), "")
	span1 := sf.Get(get("span", 0), "")
	span2 := sf.Get(get("span", 1), "")

	if got := h1.BackgroundImage(); len(got) != 1 || got[0].Type != "url" || got[0].URL != testutil.ResourceURL(t, "logo_small.png") {
		t.Errorf("h1 background-image = %+v", got)
	}
	if got := h1.FontWeight(); got != 700 {
		t.Errorf("h1 font-weight = %v", got)
	}
	if got := h1.FontSize(); got != 40 {
		t.Errorf("h1 font-size = %v", got)
	}

	checkLengths := func(name string, got []props.Length, want ...float64) {
		t.Helper()
		for i := range want {
			if got[i] != props.Px(want[i]) {
				t.Errorf("%s = %v, want %v px", name, got, want)
				return
			}
		}
	}
	checkLengths("p margins", []props.Length{p.MarginTop(), p.MarginRight(), p.MarginBottom(), p.MarginLeft()}, 24, 0, 24, 0)
	if got := p.BackgroundColor(); !got.IsCurrentColor() {
		t.Errorf("p background-color = %v, want currentcolor", got)
	}

	if got := ul.FontWeight(); got != 400 {
		t.Errorf("ul font-weight = %v", got)
	}
	if got := []props.Pixels{ul.BorderTopWidth(), ul.BorderRightWidth(), ul.BorderBottomWidth(), ul.BorderLeftWidth()}; !slices.Equal(got, []props.Pixels{0, 5, 0, 24}) {
		t.Errorf("ul border widths = %v", got)
	}

	if got := li0.FontWeight(); got != 700 {
		t.Errorf("li font-weight = %v", got)
	}
	if got := li0.FontSize(); got != 8 {
		t.Errorf("li font-size = %v", got)
	}
	checkLengths("li margins", []props.Length{li0.MarginTop(), li0.MarginRight(), li0.MarginBottom(), li0.MarginLeft()}, 16, 0, 16, 32)

	if got := a.TextDecorationLine(); !slices.Equal(got, props.Keywords{"underline"}) {
		t.Errorf("a text-decoration-line = %v", got)
	}
	if got := a.FontWeight(); got != 900 {
		t.Errorf("a font-weight = %v", got)
	}
	if got := a.FontSize(); got != 24 {
		t.Errorf("a font-size = %v", got)
	}
	checkLengths("a padding", []props.Length{a.PaddingTop(), a.PaddingRight(), a.PaddingBottom(), a.PaddingLeft()}, 1, 2, 3, 4)
	if a.BorderTopWidth() != 42 || a.BorderBottomWidth() != 42 {
		t.Errorf("a border widths = %v, %v", a.BorderTopWidth(), a.BorderBottomWidth())
	}
	if got := rgba(a.Color()); got != [4]float64{1, 0, 0, 1} {
		t.Errorf("a color = %v", got)
	}
	if got := a.BorderTopColor(); !got.IsCurrentColor() {
		t.Errorf("a border-top-color = %v", got)
	}

	if got := div.FontSize(); got != 40 {
		t.Errorf("div font-size = %v", got)
	}
	if got := span1.Width(); got != props.Px(160) {
		t.Errorf("span width = %v", got)
	}
	if got := span1.Height(); got != props.Px(400) {
		t.Errorf("span height = %v", got)
	}
	if got := span2.FontSize(); got != 32 {
		t.Errorf("inner span font-size = %v", got)
	}

	content := after.Content()
	want := []props.ContentItem{
		{Type: "string", Args: []string{" ["}},
		{Type: "attr()", Args: []string{"href", "string", ""}},
		{Type: "string", Args: []string{"]"}},
	}
	if len(content.Items) != len(want) {
		t.Fatalf("a::after content = %+v", content)
	}
	for i := range want {
		if content.Items[i].Type != want[i].Type || !slices.Equal(content.Items[i].Args, want[i].Args) {
			t.Errorf("a::after content = %+v", content)
		}
	}
	if got := rgba(after.BackgroundColor()); got != [4]float64{1, 0, 0, 1} {
		t.Errorf("a::after background-color = %v", got)
	}
	if after.BorderTopWidth() != 42 || after.BorderBottomWidth() != 3 {
		t.Errorf("a::after border widths = %v, %v", after.BorderTopWidth(), after.BorderBottomWidth())
	}
}

func TestImportantDeclarations(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc := csstest.Parse(t, `
      <style>
        p:nth-child(1) { color: lime }
        body p:nth-child(2) { color: red }

        p:nth-child(3) { color: lime !important }
        body p:nth-child(3) { color: red }

        body p:nth-child(5) { color: lime }
        p:nth-child(5) { color: red }

        p:nth-child(6) { color: red }
        p:nth-child(6) { color: lime }
      </style>
      <p></p>
      <p></p>
      <p></p>
      <p></p>
      <p></p>
      <p></p>`)
	user := cascade.ParseStylesheet(`
      body p:nth-child(1) { color: red }
      p:nth-child(2) { color: lime !important }

      p:nth-child(4) { color: lime !important }
      body p:nth-child(4) { color: red }`, "", nil)
	sf := csstest.StyleFor(t, doc, cascade.Options{UserStylesheets: []*cascade.Stylesheet{user}})
	for i, p := range csstest.Elements(doc, "p") {
		if got := rgba(sf.Get(p, "").Color()); got != [4]float64{0, 1, 0, 1} {
			t.Errorf("paragraph %d: color = %v, want lime", i+1, got)
		}
	}
}

func TestUnits(t *testing.T) {
	tests := []struct {
		value string
		width float64
	}{
		{"96px", 96}, {"1in", 96}, {"72pt", 96}, {"6pc", 96}, {"2.54cm", 96},
		{"25.4mm", 96}, {"101.6q", 96},
		{"1.1em", 11}, {"1.1rem", 17.6}, {"1.1ch", 11}, {"1.1rch", 17.6},
		{"1.1cap", 11}, {"1.1rcap", 17.6}, {"1.5ex", 12}, {"2rex", 25.6},
		{"1ic", 20}, {"1ric", 32}, {"1.1lh", 13.2}, {"1.1rlh", 26.4},
		{"50vh", 50}, {"50vw", 100}, {"25vb", 25}, {"1.25vi", 2.5}, {"10vmin", 10}, {"20vmax", 40},
		{"50lvh", 50}, {"50lvw", 100}, {"25lvb", 25}, {"1.25lvi", 2.5}, {"10lvmin", 10}, {"20lvmax", 40},
		{"50svh", 50}, {"50svw", 100}, {"25svb", 25}, {"1.25svi", 2.5}, {"10svmin", 10}, {"20svmax", 40},
		{"50dvh", 50}, {"50dvw", 100}, {"25dvb", 25}, {"1.25dvi", 2.5}, {"10dvmin", 10}, {"20dvmax", 40},
		{"50pvh", 100}, {"50pvw", 150}, {"25pvb", 50}, {"1.5pvi", 4.5}, {"10pvmin", 20}, {"20pvmax", 60},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			doc := csstest.Parse(t, `
              <html style="font: 16px / 1.5 testfont">
              <style>@page { size: 300px 200px; margin: 50px }</style>
              <body style="font: 10px / 1.2 testfont">
              <p style="margin-left: `+tt.value+`"></p>
              <p style="margin-left: `+strings.ToUpper(tt.value)+`"></p>`)
			sf := csstest.StyleFor(t, doc, cascade.Options{})
			// The layout sets the sizes of the first page before the
			// margins are used.
			sf.Context().PageBox = &computed.Size{Width: 300, Height: 200}
			sf.Context().PageArea = &computed.Size{Width: 200, Height: 100}
			ps := csstest.Elements(doc, "p")
			m1, m2 := sf.Get(ps[0], "").MarginLeft(), sf.Get(ps[1], "").MarginLeft()
			if m1 != m2 {
				t.Errorf("margins differ: %v and %v", m1, m2)
			}
			if got := m1.Dimension.Value; math.Abs(got-tt.width) > 0.01*tt.width {
				t.Errorf("margin-left: %s = %v, want %v", tt.value, got, tt.width)
			}
		})
	}
}

func TestRecursiveLineHeightUnits(t *testing.T) {
	for _, property := range []string{"line-height", "font-size"} {
		for _, units := range [][2]string{{"1lh", "1rlh"}, {"1rlh", "1lh"}} {
			testutil.AssertNoLogs(t)
			doc := csstest.Parse(t, `<html style="`+property+`: `+units[0]+`"><body style="`+property+`: `+units[1]+`">a`)
			sf := csstest.StyleFor(t, doc, cascade.Options{})
			for _, el := range doc.Elements() {
				style := sf.Get(el, "")
				style.FontSize()
				style.LineHeight()
				style.Strut()
			}
		}
	}
}

func TestMediaQueries(t *testing.T) {
	tests := []struct {
		media   string
		width   float64
		warning bool
	}{
		{"@media screen { @page { size: 10px } }", 20, false},
		{"@media print { @page { size: 10px } }", 10, false},
		{`@media ("unknown content") { @page { size: 10px } }`, 20, true},
	}
	for _, tt := range tests {
		logs := logging.StartCapture()
		doc := csstest.Parse(t, "<p>a<span>b")
		user := cascade.ParseStylesheet("@page{size:20px}"+tt.media, "", nil)
		sf := csstest.StyleFor(t, doc, cascade.Options{UserStylesheets: []*cascade.Stylesheet{user}})
		page := cascade.PageType{Side: "right"}
		sf.SetPageTypeComputedStyles(page)
		size := sf.Page(page, "").Size()
		logs.Close()
		if float64(size.X) != tt.width {
			t.Errorf("%s: page width = %v, want %v", tt.media, size.X, tt.width)
		}
		if warned := len(logs.Lines()) > 0; warned != tt.warning {
			t.Errorf("%s: logs = %q", tt.media, logs.Lines())
		}
	}
}

func TestStringAndURLToCSS(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a", `"a"`},
		{"a\nb", `"a\Ab"`},
		{"a\rb", `"a\Db"`},
		{"\n\na", `"\A\Aa"`},
		{`a"b`, `"a\"b"`},
	}
	for _, tt := range tests {
		if got := cascade.StringToCSS(tt.in); got != tt.want {
			t.Errorf("StringToCSS(%q) = %s, want %s", tt.in, got, tt.want)
		}
		if got, want := cascade.URLToCSS(tt.in), "url("+tt.want+")"; got != want {
			t.Errorf("URLToCSS(%q) = %s, want %s", tt.in, got, want)
		}
	}
}
