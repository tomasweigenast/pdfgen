package layout_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func TestLineHeightInheritanceOfPercentagesAndNumbers(t *testing.T) {
	_, body := pageBody(t, onlyPage(t, renderPages(t, `
      <style>
        html { font-size: 10px; line-height: 140% }
        section { font-size: 10px; line-height: 1.4 }
        div, p { font-size: 20px; vertical-align: 50% }
      </style>
      <body><div><section><p></p></section></div></body>
    `)))
	div := child(t, body)
	section := child(t, div)
	paragraph := child(t, section)

	if got := float64(div.Base().Style.FontSize()); got != 20 {
		t.Errorf("div font size = %g, want 20", got)
	}
	// 140% of 10px is computed on html and inherited as 14px.
	if got, _ := div.Base().Style.Strut(); got != 14 {
		t.Errorf("div strut height = %g, want 14", got)
	}
	if got := div.Base().Style.VerticalAlign(); got != props.Px(7) {
		t.Errorf("div vertical-align = %v, want 7px", got)
	}

	if got := float64(paragraph.Base().Style.FontSize()); got != 20 {
		t.Errorf("paragraph font size = %g, want 20", got)
	}
	// The number 1.4 is inherited from section and applied to 20px.
	if got, _ := paragraph.Base().Style.Strut(); got != 28 {
		t.Errorf("paragraph strut height = %g, want 28", got)
	}
	if got := paragraph.Base().Style.VerticalAlign(); got != props.Px(14) {
		t.Errorf("paragraph vertical-align = %v, want 14px", got)
	}
}
