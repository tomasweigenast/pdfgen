package layout_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// logicalDocuments are groups of documents that must look the same: each
// one is written with physical properties or with logical ones.
var logicalDocuments = [][]string{
	{
		`<div style="border-top: 1px solid">`,
		`<div style="border-block-start: 1px solid">`,
		`<div style="--size: 1px; border-block-start: var(--size) solid">`,
	}, {
		`<div style="border-left: 1px solid">`,
		`<div style="border-inline-start: 1px solid">`,
		`<div style="--size: 1px; border-inline-start: var(--size) solid">`,
	}, {
		`<div style="border-inline-start: 1px solid">`,
		`<article style="direction: rtl">` +
			`  <div style="border-inline-end: 1px solid">`,
	}, {
		`<div style="border-style: solid; border-width: 1px 2px">`,
		`<div style="border-style: solid; border-width: logical 1px 2px">`,
		// A custom property in a logical shorthand is not handled:
		// `<div style="border-style: solid; --size: 1px; border-width: logical var(--size) 2px">`
	}, {
		`<div style="padding-top: 1px">`,
		`<div style="padding-block-start: 1px">`,
	}, {
		`<div style="padding: 1px 2px 3px 4px">`,
		`<div style="padding: logical 1px 4px 3px 2px">`,
		`<div style="padding: 1px 2px;` +
			`            padding-inline-start: 4px; padding-block-end: 3px">`,
	}, {
		`<div style="margin-right: 1px">`,
		`<div style="margin-inline-end: 1px">`,
		`<article style="direction: rtl">` +
			`  <div style="margin-inline-start: 1px">`,
		`<article style="direction: rtl">` +
			`  <div style="margin: logical 0 1px 0 0">`,
	}, {
		`<div style="border-radius: 0 0 0 4px">`,
		`<div style="border-bottom-left-radius: 4px">`,
		`<div style="border-end-start-radius: 4px">`,
		`<article style="direction: rtl">` +
			`  <div style="border-end-end-radius: 4px">`,
	}, {
		`<div style="width: 5px">`,
		`<div style="inline-size: 5px">`,
		`<div style="max-inline-size: 5px">`,
		`<div style="max-inline-size: 5px; min-block-size: 2px">`,
	}, {
		`<div style="height: 6px">`,
		`<div style="block-size: 6px">`,
		`<div style="min-block-size: 6px">`,
		`<div style="min-block-size: 6px; max-inline-size: 10px">`,
	}, {
		`<div style="position: absolute; width: 5px; inset: auto 1px auto auto">`,
		`<div style="position: absolute; width: 5px; right: 1px">`,
		`<div style="position: absolute; width: 5px; inset-inline-end: 1px">`,
		`<div style="position: absolute; width: 5px; inset-inline: auto 1px">`,
		`<div style="position: absolute; width: 5px;` +
			`  --size: 1px; inset-inline: auto var(--size)">`,
	}, {
		`<div style="float: left; width: 5px">`,
		`<div style="float: inline-start; width: 5px">`,
		`<article style="direction: rtl">` +
			`  <div style="float: inline-end; width: 5px">`,
	}, {
		`<div style="float: left; width: 5px"></div>` +
			`<div style="float: left; width: 3px"></div>`,
		`<div style="float: left; width: 5px"></div>` +
			`<div style="float: left; width: 3px; clear: right"></div>`,
		`<div style="float: inline-start; width: 5px"></div>` +
			`<div style="float: left; width: 3px; clear: inline-end"></div>`,
		`<div style="float: left; width: 5px"></div>` +
			`<div style="float: inline-start; width: 3px; clear: inline-end"></div>`,
		`<article style="direction: rtl">` +
			`  <div style="float: left; width: 5px"></div>` +
			`  <div style="float: inline-end; width: 3px; clear: inline-start"></div>`,
	}, {
		`<div style="float: left; width: 5px"></div>` +
			`<div style="float: left; width: 3px; clear: left"></div>`,
		`<div style="float: inline-start; width: 5px"></div>` +
			`<div style="float: left; width: 3px; clear: inline-start"></div>`,
		`<div style="float: inline-start; width: 5px"></div>` +
			`<div style="float: inline-start; width: 3px; clear: left"></div>`,
		`<div style="float: inline-start; width: 5px"></div>` +
			`<div style="float: inline-start; width: 3px; clear: both"></div>`,
		`<article style="direction: rtl">` +
			`  <div style="float: inline-end; width: 5px"></div>` +
			`  <div style="float: inline-end; width: 3px; clear: left"></div>`,
		`<article style="direction: rtl">` +
			`  <div style="float: left; width: 5px"></div>` +
			`  <div style="float: inline-end; width: 3px; clear: inline-end"></div>`,
	},
}

const logicalBaseStyle = `
    <style>
      @page { size: 10px }
      div { background: pink; background-clip: padding-box; height: 5px }
    </style>
    `

func TestLogicalPropertiesRenderTheSame(t *testing.T) {
	for i, documents := range logicalDocuments {
		t.Run(fmt.Sprint(i+1), func(t *testing.T) {
			testutil.AssertNoLogs(t)
			htmls := make([]string, len(documents))
			for j, document := range documents {
				htmls[j] = logicalBaseStyle + document
			}
			testutil.AssertSameRenderings(t, 0, htmls...)
		})
	}
}

// divGeometry describes the divs of a page: where they are and how big,
// with their margins, paddings, borders and the radii of their corners.
func divGeometry(page *boxes.PageBox) string {
	var out strings.Builder
	for box := range boxes.Descendants(page, true) {
		b := box.Base()
		if b.ElementTag != "div" {
			continue
		}
		fmt.Fprintf(&out, "div at (%g, %g) %g × %g margins %g %g %g %g paddings %g %g %g %g borders %g %g %g %g radii %v %v %v %v\n",
			b.PositionX, b.PositionY, b.Width.V, b.Height.V,
			b.MarginTop.V, b.MarginRight.V, b.MarginBottom.V, b.MarginLeft.V,
			b.PaddingTop, b.PaddingRight, b.PaddingBottom, b.PaddingLeft,
			b.BorderTopWidth, b.BorderRightWidth, b.BorderBottomWidth, b.BorderLeftWidth,
			b.BorderTopLeftRadius, b.BorderTopRightRadius, b.BorderBottomRightRadius, b.BorderBottomLeftRadius)
	}
	return out.String()
}

func TestLogicalPropertiesGiveTheSameBoxGeometry(t *testing.T) {
	for i, documents := range logicalDocuments {
		t.Run(fmt.Sprint(i+1), func(t *testing.T) {
			var want string
			for j, document := range documents {
				got := divGeometry(onlyPage(t, renderPages(t, logicalBaseStyle+document)))
				if j == 0 {
					want = got
					continue
				}
				if got != want {
					t.Errorf("document %d differs from the first:\n%s\nwant:\n%s\n%s", j+1, got, want, document)
				}
			}
		})
	}
}
