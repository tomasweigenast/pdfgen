package draw_test

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestListStyleImage(t *testing.T) {
	// The list item has horizontal margins of 7px and 2px and so is 3px
	// wide. The marker image is 4px wide, with a margin of 0.5em = 2px when
	// outside.
	for _, tc := range []struct{ position, pixels string }{
		{"outside", `
			____________
			____________
			___rBBB_____
			___BBBB_____
			___BBBB_____
			___BBBB_____
			____________
			____________
			____________
			____________
		`},
		{"inside", `
			____________
			____________
			_______rBBB_
			_______BBBB_
			_______BBBB_
			_______BBBB_
			____________
			____________
			____________
			____________
		`},
	} {
		t.Run(tc.position, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, tc.pixels, fmt.Sprintf(`
				<style>
					@page { size: 12px 10px }
					body { margin: 0; font-family: testfont }
					ul { margin: 2px 2px 0 7px; list-style: url(pattern.png) %s; font-size: 2px }
				</style>
				<ul><li></li></ul>`, tc.position))
		})
	}
}

func TestListStyleImageNone(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		__________
		__________
		__________
		__________
		__________
		__________
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px }
			body { margin: 0; font-family: testfont }
			ul { margin: 0 0 0 5px; list-style: none; font-size: 2px }
		</style>
		<ul><li>`)
}
