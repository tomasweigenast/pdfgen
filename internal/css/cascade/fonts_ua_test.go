package cascade_test

import (
	"math"
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/counters"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/ua"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestFontSize(t *testing.T) {
	tests := []struct {
		parent     string
		parentSize float64
		child      string
		childSize  float64
	}{
		{"10px", 10, "10px", 10},
		{"x-small", 12, "xx-large", 32},
		{"x-large", 24, "2em", 48},
		{"1em", 16, "1em", 16},
		{"1em", 16, "larger", 6.0 / 5 * 16},
		{"medium", 16, "larger", 6.0 / 5 * 16},
		{"x-large", 24, "larger", 32},
		{"xx-large", 32, "larger", 1.2 * 32},
		{"1px", 1, "larger", 3.0 / 5 * 16},
		{"28px", 28, "larger", 32},
		{"100px", 100, "larger", 120},
		{"xx-small", 3.0 / 5 * 16, "larger", 12},
		{"1em", 16, "smaller", 8.0 / 9 * 16},
		{"medium", 16, "smaller", 8.0 / 9 * 16},
		{"x-large", 24, "smaller", 6.0 / 5 * 16},
		{"xx-large", 32, "smaller", 24},
		{"xx-small", 3.0 / 5 * 16, "smaller", 0.8 * 3 / 5 * 16},
		{"1px", 1, "smaller", 0.8},
		{"28px", 28, "smaller", 24},
		{"100px", 100, "smaller", 32},
	}
	isClose := func(a, b float64) bool { return math.Abs(a-b) <= 1e-9*math.Max(math.Abs(a), math.Abs(b)) }
	for _, tt := range tests {
		testutil.AssertNoLogs(t)
		doc := csstest.Parse(t, "<p>a<span>b")
		user := cascade.ParseStylesheet("p{font-size:"+tt.parent+"}span{font-size:"+tt.child+"}", "", nil)
		sf := csstest.StyleFor(t, doc, cascade.Options{UserStylesheets: []*cascade.Stylesheet{user}})
		if got := float64(sf.Get(csstest.Element(t, doc, "p"), "").FontSize()); !isClose(got, tt.parentSize) {
			t.Errorf("p { font-size: %s } gives %v, want %v", tt.parent, got, tt.parentSize)
		}
		if got := float64(sf.Get(csstest.Element(t, doc, "span"), "").FontSize()); !isClose(got, tt.childSize) {
			t.Errorf("span { font-size: %s } in %s gives %v, want %v", tt.child, tt.parent, got, tt.childSize)
		}
	}
}

func TestLineHeightInheritance(t *testing.T) {
	testutil.AssertNoLogs(t)
	style, _ := styled(t, `<style>
        html { font-size: 10px; line-height: 140% }
        section { font-size: 10px; line-height: 1.4 }
        div, p { font-size: 20px; vertical-align: 50% }
      </style>
      <body><div><section><p></p></section></div></body>`)
	html, div, p := style("html", 0, ""), style("div", 0, ""), style("p", 0, "")
	if got := html.FontSize(); got != 10 {
		t.Errorf("html font-size = %v", got)
	}
	if got := div.FontSize(); got != 20 {
		t.Errorf("div font-size = %v", got)
	}
	// 140% of 10px is inherited as 14px.
	if height, _ := div.Strut(); height != 14 {
		t.Errorf("div line height = %v, want 14", height)
	}
	if got := div.VerticalAlign(); got != props.Px(7) {
		t.Errorf("div vertical-align = %v, want 7px", got)
	}
	if got := p.FontSize(); got != 20 {
		t.Errorf("p font-size = %v", got)
	}
	// 1.4 is inherited as a number, multiplied by the font size.
	if height, _ := p.Strut(); height != 28 {
		t.Errorf("p line height = %v, want 28", height)
	}
	if got := p.VerticalAlign(); got != props.Px(14) {
		t.Errorf("p vertical-align = %v, want 14px", got)
	}
}

func TestUserAgentStylesheetsAreValid(t *testing.T) {
	for _, css := range []string{ua.HTML, ua.Form, ua.PresentationalHints} {
		logs := logging.StartCapture()
		cascade.ParseStylesheet(css, "", nil)
		logs.Close()
		if lines := logs.Lines(); len(lines) != 0 {
			t.Errorf("logs = %q", lines)
		}
	}
}

func TestFontFaceRuleWarnings(t *testing.T) {
	tests := []struct {
		css  string
		want []string
	}{
		{"@font-face{}", []string{"WARNING: Missing src descriptor in '@font-face' rule at 1:1"}},
		{"@font-face{src: url(test.woff)}", []string{"WARNING: Missing font-family descriptor in '@font-face' rule at 1:1"}},
		{"@font-face{font-family: test}", []string{"WARNING: Missing src descriptor in '@font-face' rule at 1:1"}},
		{"@font-face { font-family: test; src: wrong }", []string{
			"WARNING: Ignored `src: wrong ` at 1:33, invalid value.",
			"WARNING: Missing src descriptor in '@font-face' rule at 1:1"}},
		{"@font-face { font-family: good, bad; src: url(test.woff) }", []string{
			"WARNING: Ignored `font-family: good, bad` at 1:14, invalid value.",
			"WARNING: Missing font-family descriptor in '@font-face' rule at 1:1"}},
		{"@font-face { font-family: good, bad; src: really bad }", []string{
			"WARNING: Ignored `font-family: good, bad` at 1:14, invalid value.",
			"WARNING: Ignored `src: really bad ` at 1:38, invalid value.",
			"WARNING: Missing src descriptor in '@font-face' rule at 1:1"}},
	}
	for _, tt := range tests {
		logs := logging.StartCapture()
		cascade.ParseStylesheet(tt.css, "https://example.org/foo/", nil)
		logs.Close()
		if got := logs.Lines(); !slices.Equal(got, tt.want) {
			t.Errorf("%s: logs = %q, want %q", tt.css, got, tt.want)
		}
	}
}

func TestFontFaceThatCannotBeLoadedIsReported(t *testing.T) {
	logs := logging.StartCapture()
	env := csstest.Environment()
	cascade.ParseStylesheet("@font-face { font-family: missing; src: url(missing.woff) }", csstest.BaseURL(t), env)
	logs.Close()
	if got, want := logs.Lines(), []string{"WARNING: Font-face 'missing' cannot be loaded"}; !slices.Equal(got, want) {
		t.Errorf("logs = %q, want %q", got, want)
	}
}

func TestInvalidCounterStyles(t *testing.T) {
	for _, css := range []string{
		"@counter-style test {system: alphabetic; symbols: a}",
		"@counter-style test {system: cyclic}",
		"@counter-style test {system: additive; additive-symbols: a 1}",
		"@counter-style test {system: additive; additive-symbols: 10 x, 1 i, 5 v}",
	} {
		logs := logging.StartCapture()
		env := &cascade.Environment{CounterStyle: counters.Registry{}}
		cascade.ParseStylesheet(css, "https://example.org/foo/", env)
		logs.Close()
		if len(logs.Lines()) < 1 {
			t.Errorf("%s: no warning", css)
		}
		if _, ok := env.CounterStyle["test"]; ok {
			t.Errorf("%s: style registered", css)
		}
	}
}

func TestCounterStyleDefinitionsOverridePredefinedOnes(t *testing.T) {
	const rule = `@counter-style lower-roman { system: cyclic; symbols: "*" }`
	symbols := func(t *testing.T, env *cascade.Environment) []string {
		t.Helper()
		style, ok := env.CounterStyle["lower-roman"]
		if !ok {
			t.Fatal("lower-roman is not defined")
		}
		var out []string
		for _, s := range style.Symbols {
			out = append(out, s.Value)
		}
		return out
	}

	t.Run("user", func(t *testing.T) {
		env := &cascade.Environment{CounterStyle: counters.Registry{}}
		user := cascade.ParseStylesheet(rule, "", env)
		doc := csstest.Parse(t, `<p>`)
		csstest.StyleFor(t, doc, cascade.Options{Environment: env, UserStylesheets: []*cascade.Stylesheet{user}})
		if got := symbols(t, env); !slices.Equal(got, []string{"*"}) {
			t.Errorf("symbols = %q, want the user's", got)
		}
	})

	t.Run("author", func(t *testing.T) {
		env := &cascade.Environment{CounterStyle: counters.Registry{}}
		doc := csstest.Parse(t, `<style>`+rule+`</style><p>`)
		csstest.StyleFor(t, doc, cascade.Options{Environment: env})
		if got := symbols(t, env); !slices.Equal(got, []string{"*"}) {
			t.Errorf("symbols = %q, want the author's", got)
		}
	})

	t.Run("none", func(t *testing.T) {
		env := &cascade.Environment{CounterStyle: counters.Registry{}}
		csstest.StyleFor(t, csstest.Parse(t, `<p>`), cascade.Options{Environment: env})
		// The predefined lower-roman is an additive style without symbols.
		if got := symbols(t, env); len(got) != 0 {
			t.Errorf("symbols = %q, want the predefined style", got)
		}
	})
}
