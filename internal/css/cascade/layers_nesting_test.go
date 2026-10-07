package cascade_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func TestLayers(t *testing.T) {
	styles := []string{
		"@layer { div { width: 100px } }",
		"@layer a { div { width: 100px } }",
		"div { width: 100px } @layer a { div { width: 200px } }",
		"@layer { div { width: 200px } } @layer a { div { width: 100px } }",
		"@layer a { div { width: 200px } } @layer { div { width: 100px } }",
		"@layer a { div { width: 200px } } @layer a { div { width: 100px } }",
		"@layer a { div { width: 100px } } @layer a.b { div { width: 200px } }",
		"@layer a.b { div { width: 200px } } @layer a { div { width: 100px } }",
		"@layer a { div { width: 200px } } @layer b { div { width: 100px } }",
		"@layer b, a; @layer a { div { width: 100px } } @layer b { div { width: 200px } }",
		"@layer b; @layer a { div { width: 100px } } @layer b { div { width: 200px } }",
		"@import url(data:text/css,div{width:100px}); @layer a { div { width: 200px } }",
		"@import url(data:text/css,div{width:200px}) layer; @layer a { div { width: 100px } }",
		"@import url(data:text/css,div{width:200px}) layer(b); @layer a { div { width: 100px } }",
		"@import url(data:text/css,div{width:100px}) layer(a); @layer a.b { div { width: 200px } }",
		"@import url(data:text/css,div{width:200px}) layer(a.b); @layer a { div { width: 100px } }",
		"@layer a { div { width: 100px } } @import url(data:text/css,div{width:200px}) layer(a.b);",
	}
	for _, css := range styles {
		testutil.AssertNoLogs(t)
		if got := widthOf(t, "<style>"+css+"</style><div>abc</div>", "div"); got != props.Px(100) {
			t.Errorf("%s: width = %v, want 100px", css, got)
		}
	}
}

func TestNestedRules(t *testing.T) {
	styles := []string{
		"div { p { width: 10px } }",
		"p { div & { width: 10px } }",
		"p { width: 20px; div & { width: 10px } }",
		"p { div & { width: 10px } width: 20px }",
		"div { & { & { p { & { width: 10px } } } } }",
		"@media print { div { p { width: 10px } } }",
		"div { em, p { width: 10px } }",
		"p { a, div & { width: 10px } }",
	}
	for _, css := range styles {
		testutil.AssertNoLogs(t)
		style, _ := styled(t, "<style>"+css+"</style><div><p></p></div><p></p>")
		if got := style("p", 0, "").Width(); got != props.Px(10) {
			t.Errorf("%s: nested width = %v, want 10px", css, got)
		}
		if got := style("p", 1, "").Width(); got == props.Px(10) {
			t.Errorf("%s: other width = %v, want something else", css, got)
		}
	}
}
