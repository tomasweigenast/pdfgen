package layout_test

import (
	"math"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

func TestImportantDeclarationsOfUserStylesheetsAndAuthor(t *testing.T) {
	testutil.AssertNoLogs(t)
	doc := boxtest.FakeHTML(t, `
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
      <p></p>
    `, boxtest.WithStylesheet(`
      body p:nth-child(1) { color: red }
      p:nth-child(2) { color: lime !important }

      p:nth-child(4) { color: lime !important }
      body p:nth-child(4) { color: red }
    `))
	_, body := pageBody(t, onlyPage(t, doc.RenderPages(t)))
	for i, paragraph := range children(t, body, 6) {
		c := paragraph.Base().Style.Color()
		if got := [4]float64{c.Coordinates[0], c.Coordinates[1], c.Coordinates[2], c.Alpha}; got != [4]float64{0, 1, 0, 1} {
			t.Errorf("paragraph %d: color = %v, want lime", i+1, got)
		}
	}
}

func TestUnitsInMarginsOfLaidOutBlocks(t *testing.T) {
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
			pages := renderPages(t, `
              <html style="font: 16px / 1.5 testfont">
              <style>@page { size: 300px 200px; margin: 50px }</style>
              <body style="font: 10px / 1.2 testfont">
              <p style="margin-left: `+tt.value+`"></p>
              <p style="margin-left: `+strings.ToUpper(tt.value)+`"></p>`)
			_, body := pageBody(t, onlyPage(t, pages))
			ps := children(t, body, 2)
			m1, m2 := ps[0].Base().MarginLeft.V, ps[1].Base().MarginLeft.V
			if m1 != m2 {
				t.Errorf("margins differ: %v and %v", m1, m2)
			}
			if math.Abs(m1-tt.width) > 0.01*tt.width {
				t.Errorf("margin-left: %s = %v, want %v", tt.value, m1, tt.width)
			}
		})
	}
}

func TestRecursiveLineHeightUnitsLayOut(t *testing.T) {
	for _, property := range []string{"line-height", "font-size"} {
		for _, units := range [][2]string{{"1lh", "1rlh"}, {"1rlh", "1lh"}} {
			t.Run(property+" "+units[0]+" "+units[1], func(t *testing.T) {
				renderPages(t, `<html style="`+property+`: `+units[0]+`"><body style="`+property+`: `+units[1]+`">a`)
			})
		}
	}
}

func TestMediaQueriesChangeThePageSize(t *testing.T) {
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
		t.Run(tt.media, func(t *testing.T) {
			logs := testutil.CaptureLogs(t)
			doc := boxtest.FakeHTML(t, "<p>a<span>b", boxtest.WithStylesheet("@page{size:20px}"+tt.media))
			html, _ := pageBody(t, onlyPage(t, doc.RenderPages(t)))
			if got := html.Base().Width.V; got != tt.width {
				t.Errorf("html width = %v, want %v", got, tt.width)
			}
			if warned := len(logs.Lines()) > 0; warned != tt.warning {
				t.Errorf("logs = %q, want warning: %v", logs.Lines(), tt.warning)
			}
		})
	}
}
