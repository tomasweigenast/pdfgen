package cascade_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestPageStyles(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc := parseFile(t, "doc1.html")
	user := cascade.ParseStylesheet(`
      html { color: red }
      @page { margin: 10px }
      @page :right {
        color: blue;
        margin-bottom: 12pt;
        font-size: 20px;
        @top-left { width: 10em }
        @top-right { font-size: 10px}
      }`, "", nil)
	sf := csstest.StyleFor(t, doc, cascade.Options{UserStylesheets: []*cascade.Stylesheet{user}})

	tests := []struct {
		page              cascade.PageType
		top, left, bottom float64
		color             [4]float64
	}{
		{cascade.PageType{Side: "left", Index: 0}, 5, 10, 10, [4]float64{1, 0, 0, 1}},
		{cascade.PageType{Side: "right", Index: 0}, 5, 10, 16, [4]float64{0, 0, 1, 1}},
		{cascade.PageType{Side: "left", Index: 1}, 10, 10, 10, [4]float64{1, 0, 0, 1}},
		{cascade.PageType{Side: "right", Index: 1}, 10, 10, 16, [4]float64{0, 0, 1, 1}},
	}
	for _, tt := range tests {
		sf.SetPageTypeComputedStyles(tt.page)
		style := sf.Page(tt.page, "")
		if got := style.MarginTop(); got != props.Px(tt.top) {
			t.Errorf("%+v: margin-top = %v, want %v", tt.page, got, tt.top)
		}
		if got := style.MarginLeft(); got != props.Px(tt.left) {
			t.Errorf("%+v: margin-left = %v, want %v", tt.page, got, tt.left)
		}
		if got := style.MarginBottom(); got != props.Px(tt.bottom) {
			t.Errorf("%+v: margin-bottom = %v, want %v", tt.page, got, tt.bottom)
		}
		if got := rgba(style.Color()); got != tt.color {
			t.Errorf("%+v: color = %v, want %v", tt.page, got, tt.color)
		}
	}

	left := cascade.PageType{Side: "left"}
	sf.SetPageTypeComputedStyles(left)
	if style := sf.Page(left, "@top-left"); style != nil {
		t.Error("@top-left has a style on left pages")
	}
	right := cascade.PageType{Side: "right"}
	sf.SetPageTypeComputedStyles(right)
	topLeft := sf.Page(right, "@top-left")
	if got := topLeft.FontSize(); got != 20 {
		t.Errorf("@top-left font-size = %v, want 20 from the page", got)
	}
	if got := topLeft.Width(); got != props.Px(200) {
		t.Errorf("@top-left width = %v, want 200px", got)
	}
	if got := sf.Page(right, "@top-right").FontSize(); got != 10 {
		t.Errorf("@top-right font-size = %v, want 10", got)
	}
}

func TestNamedPages(t *testing.T) {
	testutil.AssertNoLogs(t)
	style, _ := styled(t, `<style>
        @page NARRow { size: landscape }
        div { page: AUTO }
        p { page: NARRow }
      </style>
      <div><p><span>a</span></p></div>`)
	tests := []struct {
		tag  string
		want props.Keyword
	}{
		{"html", ""}, {"body", ""}, {"div", ""}, {"p", "NARRow"}, {"span", "NARRow"},
	}
	for _, tt := range tests {
		if got := style(tt.tag, 0, "").Page(); got != tt.want {
			t.Errorf("%s: page = %q, want %q", tt.tag, got, tt.want)
		}
	}
}
