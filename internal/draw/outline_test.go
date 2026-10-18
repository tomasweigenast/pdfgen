package draw_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

const outlineStyleSource = `
	<style>
		@page { size: 140px 110px }
		body { width: 100px; height: 70px; margin: 20px; outline: 10PX %s blue }
	</style>
	<body>`

func TestOutlinesLookDifferentPerStyle(t *testing.T) {
	testutil.AssertNoLogs(t)
	docs := make([]string, len(borderStyles))
	for i, style := range borderStyles {
		docs[i] = fmt.Sprintf(outlineStyleSource, style)
	}
	testutil.AssertDifferentRenderings(t, docs...)
}

func TestSolidOutlinePixels(t *testing.T) {
	testutil.AssertNoLogs(t)
	const width, height, margin, outline = 140, 110, 10, 10
	var art strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			inBox := x >= margin && x < width-margin && y >= margin && y < height-margin
			inside := x >= margin+outline && x < width-margin-outline &&
				y >= margin+outline && y < height-margin-outline
			if inBox && !inside {
				art.WriteByte('B')
			} else {
				art.WriteByte('_')
			}
		}
		art.WriteByte('\n')
	}
	testutil.AssertPixels(t, art.String(), fmt.Sprintf(outlineStyleSource, "solid"))
}

func TestOutlineAndBorder(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		__________
		_BBBBBBBB_
		_BRRRRRRB_
		_BR____RB_
		_BRRRRRRB_
		_BBBBBBBB_
		__________
	`, `
		<style>
			@page { size: 10px 7px }
			div {
				border: 1px solid red;
				outline: 1px solid blue;
				height: 1px;
				margin: 2px;
				min-height: auto;
				min-width: auto;
				width: 4px;
			}
		</style>
		<div></div>
	`)
}

func TestOutlineOffset(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____________
		_BBBBBBBBBB_
		_B________B_
		_B_RRRRRR_B_
		_B_R____R_B_
		_B_RRRRRR_B_
		_B________B_
		_BBBBBBBBBB_
		____________
	`, `
		<style>
			@page { size: 12px 9px }
			div {
				border: 1px solid red;
				outline: 1px solid blue;
				outline-offset: 1px;
				height: 1px;
				margin: 3px;
				min-height: auto;
				min-width: auto;
				width: 4px;
			}
		</style>
		<div></div>
	`)
}

func TestOutlineDrawnForDescendants(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		_BBBB_
		_B__B_
		_B__B_
		_BBBB_
		______
	`, `
		<style>
			@page { size: 6px }
			body { margin: 0 }
			section { margin: 2px }
			div { width: 2px; height: 2px; outline: 1px solid blue }
		</style>
		<section><div></div></section>
	`)
}
