package draw_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

const borderStyleSource = `
	<style>
		@page { size: 140px 110px }
		body { width: 100px; height: 70px; margin: 10px; border: 10PX %s blue }
	</style>
	<body>`

var borderStyles = []string{
	"none", "solid", "dashed", "dotted", "double",
	"inset", "outset", "groove", "ridge",
}

func TestBordersLookDifferentPerStyle(t *testing.T) {
	testutil.AssertNoLogs(t)
	docs := make([]string, len(borderStyles))
	for i, style := range borderStyles {
		docs[i] = fmt.Sprintf(borderStyleSource, style)
	}
	testutil.AssertDifferentRenderings(t, docs...)
}

func TestSolidBorderPixels(t *testing.T) {
	testutil.AssertNoLogs(t)
	const width, height, margin, border = 140, 110, 10, 10
	var art strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			inBox := x >= margin && x < width-margin && y >= margin && y < height-margin
			inside := x >= margin+border && x < width-margin-border &&
				y >= margin+border && y < height-margin-border
			if inBox && !inside {
				art.WriteByte('B')
			} else {
				art.WriteByte('_')
			}
		}
		art.WriteByte('\n')
	}
	testutil.AssertPixels(t, art.String(), fmt.Sprintf(borderStyleSource, "solid"))
}

func TestSideBordersMeetOnTheCornerDiagonals(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		zRRRRRRz
		KzRRRRzG
		KKzRRzGG
		KKK__GGG
		KKK__GGG
		KKzBBzGG
		KzBBBBzG
		zBBBBBBz
	`, `
		<style>
			@page { size: 8px }
			div {
				border: 3px solid;
				border-color: red lime blue black;
				height: 2px;
				width: 2px;
			}
		</style>
		<div></div>`)
}

func TestCollapsedTableBordersLookDifferentPerStyle(t *testing.T) {
	t.Skip("not supported: table layout")
	const source = `
		<style>
			@page { size: 140px 110px }
			table { width: 100px; height: 70px; margin: 10px;
			        border-collapse: collapse; border: 10px %s blue }
		</style>
		<table><td>abc</td>`
	var docs []string
	for _, style := range []string{"none", "solid", "dashed", "dotted", "double", "inset", "groove"} {
		docs = append(docs, fmt.Sprintf(source, style))
	}
	testutil.AssertDifferentRenderings(t, docs...)
}

func TestCollapsedTableBordersEquivalentStyles(t *testing.T) {
	t.Skip("not supported: table layout")
	const source = `
		<style>
			@page { size: 140px 110px }
			table { width: 100px; height: 70px; margin: 10px;
			        border-collapse: collapse; border: 10pX %s blue }
		</style>
		<table><td>abc</td>`
	for _, styles := range [][2]string{{"groove", "outset"}, {"ridge", "inset"}} {
		t.Run(styles[0]+" "+styles[1], func(t *testing.T) {
			testutil.AssertSameRenderings(t, 0, fmt.Sprintf(source, styles[0]), fmt.Sprintf(source, styles[1]))
		})
	}
}

func TestSmallBordersWithoutHeight(t *testing.T) {
	for _, style := range []string{"none", "solid", "dashed", "dotted"} {
		t.Run(style, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			rendertest.PDF(t, fmt.Sprintf(`
				<style>
					@page { size: 50px 50px }
					body { margin: 5px; height: 0; border: 10px %s blue }
				</style>
				<body>`, style))
		})
	}
}

func TestSmallBordersWithoutArea(t *testing.T) {
	for _, style := range []string{"none", "solid", "dashed", "dotted"} {
		t.Run(style, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			rendertest.PDF(t, fmt.Sprintf(`
				<style>
					@page { size: 50Px 50px }
					body { height: 0; width: 0; border-width: 1px 0; border-style: %s }
				</style>
				<body>`, style))
		})
	}
}

func TestBorderWidthInEm(t *testing.T) {
	testutil.AssertNoLogs(t)
	rendertest.PDF(t, `<body style="border: 1em solid">`)
}

func TestBorderBoxSizingBorders(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________
		_RRRRRR_
		_R____R_
		_RRRRRR_
		________
	`, `
		<style>
			@page { size: 8px 5px }
			div {
				border: 1px solid red;
				box-sizing: border-box;
				height: 3px;
				margin: 1px;
				min-height: auto;
				min-width: auto;
				width: 6px;
			}
		</style>
		<div></div>`)
}

func TestMarginBoxBackgrounds(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		_______________
		_GGG______BBBB_
		_GGG______BBBB_
		_______________
		_____RRRR______
		_____RRRR______
		_____RRRR______
		_____RRRR______
		_______________
		_bbb______gggg_
		_bbb______gggg_
		_bbb______gggg_
		_bbb______gggg_
		_bbb______gggg_
		_______________
	`, `
		<style>
			html { height: 100% }
			body { background: #f00; height: 100% }
			@page {
				size: 15px;
				margin: 4px 6Px 7px 5px;

				@top-left-corner {
					margin: 1px;
					content: " ";
					background: #0f0;
				}
				@top-right-corner {
					margin: 1px;
					content: " ";
					background: #00f;
				}
				@bottom-right-corner {
					margin: 1px;
					content: " ";
					background: #008000;
				}
				@bottom-left-corner {
					margin: 1px;
					content: " ";
					background: #000080;
				}
			}
		</style>
		<body>`)
}

func TestInlineBlockRendersTwiceIdentically(t *testing.T) {
	testutil.AssertNoLogs(t)
	const html = `<div style="background: red; display: inline-block">`
	pages := rendertest.Pages(t, html)
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	first := rendertest.PDF(t, html)
	second := rendertest.PDF(t, html)
	if string(first) != string(second) {
		t.Error("two renderings of the same document differ")
	}
}

func TestBorderRadiusClipsBackground(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		___zzzzz
		__zzzzzz
		_zzzzzzz
		zzzzzzzz
		zzzzzzzz
		zzzzzzzR
		zzzzzzRR
		zzzzzRRR
	`, `
		<style>
			@page { size: 8px 8px }
			div {
				background: Red;
				border-radius: 50% 0 0 0;
				height: 16px;
				width: 16PX;
			}
		</style>
		<div></div>`)
}

func TestSplitBorderRadius(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		___zzzzz
		__zzzzzz
		_zzzzzzz
		zzzzzzzz
		zzzzzzzz
		zzzzzzzz
		zzzzzzRR
		zzzzzRRR

		RRRRRRRR
		RRRRRRRR
		RRRRRRRR
		RRRRRRRR
		RRRRRRRR
		RRRRRRRR
		RRRRRRRR
		RRRRRRRR

		zzzzzzRR
		zzzzzzzR
		zzzzzzzz
		zzzzzzzz
		zzzzzzzz
		zzzzzzzz
		_zzzzzzz
		__zzzzzz
	`, `
		<style>
			@page { size: 8px 8px }
			div {
				background: red;
				color: transparent;
				border-radius: 8px;
				line-height: 9px;
				width: 16px;
			}
		</style>
		<div>a b c</div>`)
}

func TestInfiniteBorderRadius(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		__________
		__zzzzzz__
		_zzzzzzzz_
		_zzzRRzzz_
		_zzRRRRzz_
		_zzRRRRzz_
		_zzzRRzzz_
		_zzzzzzzz_
		__zzzzzz__
		__________
	`, `
		<style>
			@page { size: 10px }
			div {
				background-color: red;
				border-radius: calc(infinity * 1px);
				height: 8px;
				margin: 1px;
				width: 8px;
			}
		</style>
		<div></div>`)
}
