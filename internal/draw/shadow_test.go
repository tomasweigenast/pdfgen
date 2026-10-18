package draw_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestBoxShadowPixels(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			"outset",
			`
			__________
			_BBBBB____
			_BBBBB____
			_BBBBB____
			_BBBBBrrr_
			_BBBBBrrr_
			____rrrrr_
			____rrrrr_
			____rrrrr_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 5px; height: 5px; background: blue;
			        box-shadow: 3px 3px red }
			</style>
			<body><div>
			`,
		},
		{
			"outset spread",
			`
			__________
			_BBB______
			_BBBrrrrr_
			_BBBrrrrr_
			__rrrrrrr_
			__rrrrrrr_
			__rrrrrrr_
			__rrrrrrr_
			__rrrrrrr_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 3px; height: 3px; background: blue;
			        box-shadow: 3px 3px 0 2px red }
			</style>
			<body><div>
			`,
		},
		{
			"outset negative spread",
			`
			__________
			_BBB______
			_BBB______
			_BBB______
			__________
			_____r____
			__________
			__________
			__________
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 3px; height: 3px; background: blue;
			        box-shadow: 3px 3px 0 -1px red }
			</style>
			<body><div>
			`,
		},
		{
			"outset big negative spread",
			`
			__________
			_BBB______
			_BBB______
			_BBB______
			__________
			__________
			__________
			__________
			__________
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 3px; height: 3px; background: blue;
			        box-shadow: 3px 3px 0 -2px red }
			</style>
			<body><div>
			`,
		},
		{
			"inset",
			`
			__________
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrBBBBB_
			_rrrBBBBB_
			_rrrBBBBB_
			_rrrBBBBB_
			_rrrBBBBB_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: 3px 3px red inset }
			</style>
			<body><div>
			`,
		},
		{
			"inset spread",
			`
			__________
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrrBBBB_
			_rrrrBBBB_
			_rrrrBBBB_
			_rrrrBBBB_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: 3px 3px 0 1px red inset }
			</style>
			<body><div>
			`,
		},
		{
			"inset negative spread",
			`
			__________
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrrrrrr_
			_BBBBBBBr_
			_BBBBBBBr_
			_BBBBBBBr_
			_BBBBBBBr_
			_BBBBBBBr_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: -2px 4px 0 -1px red inset }
			</style>
			<body><div>
			`,
		},
		{
			"inset big negative spread",
			`
			__________
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: -2px 4px 0 -5px red inset }
			</style>
			<body><div>
			`,
		},
		{
			"transparent background",
			`
			__________
			__________
			__________
			__________
			______rrr_
			______rrr_
			____rrrrr_
			____rrrrr_
			____rrrrr_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 5px; height: 5px; box-shadow: 3px 3px red }
			</style>
			<body><div>
			`,
		},
		{
			"inset outset",
			`
			rrrrrrrr__
			rgggggggg_
			rgBBBBBBB_
			rgBBBBBBB_
			rgBBBBBBB_
			rgBBBBBBB_
			rgBBBBBBB_
			rgBBBBBBB_
			_gBBBBBBB_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: -1px -1px red, 1px 1px green inset }
			</style>
			<body><div>
			`,
		},
		{
			"multiple outset",
			`
			__________
			_BBBBB____
			_BBBBBr___
			_BBBBBrg__
			_BBBBBrg__
			_BBBBBrg__
			__rrrrrg__
			___ggggg__
			__________
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 5px; height: 5px; background: blue;
			        box-shadow: 1px 1px red, 2px 2px green }
			</style>
			<body><div>
			`,
		},
		{
			"multiple inset",
			`
			__________
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrrrrrr_
			_rrrBBBBg_
			_rrrBBBBg_
			_rrrBBBBg_
			_rrrBBBBg_
			_rrrggggg_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: 3px 3px red inset, -1px -1px green inset }
			</style>
			<body><div>
			`,
		},
		{
			"outset radius",
			`
			__________
			__zzBB____
			_zzzBB____
			_zzBzz____
			_BBzzzzrr_
			_BBzzzzrr_
			____zzrzz_
			____rrzzz_
			____rrzz__
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 5px; height: 5px; background: blue;
			        box-shadow: 3px 3px red; border-radius: 4px 0 4px 0 }
			</style>
			<body><div>
			`,
		},
		{
			"inset radius",
			`
			__________
			__zzzrrrr_
			_zzzzrrrr_
			_zzzzrrrr_
			_zzzrzzzB_
			_rrrzzzzB_
			_rrrzzzzB_
			_rrrzzzBB_
			_rrrBBBBB_
			__________
			`,
			`
			<style>
			  @page { size: 10px; margin: 1px }
			  div { width: 8px; height: 8px; background: blue;
			        box-shadow: 3px 3px red inset; border-radius: 4px 0 0 0 }
			</style>
			<body><div>
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, tt.pixels, tt.html)
		})
	}
}

func TestBoxShadowBlurUsesShadings(t *testing.T) {
	testutil.AssertNoLogs(t)
	const page = `<style>@page { size: 30px; margin: 10px } div { width: 10px; height: 10px; box-shadow: %s }</style><div></div>`
	sharp := string(rendertest.PDF(t, fmt.Sprintf(page, "0 0 0 red")))
	blurred := string(rendertest.PDF(t, fmt.Sprintf(page, "0 0 4px red")))
	inset := string(rendertest.PDF(t, fmt.Sprintf(page, "inset 0 0 4px red")))

	if n := strings.Count(sharp, "/ShadingType"); n != 0 {
		t.Errorf("shadow without blur has %d shadings, want none", n)
	}
	// One gradient for each of the four sides and four corners.
	for name, doc := range map[string]string{"outer": blurred, "inset": inset} {
		if n := strings.Count(doc, "/ShadingType"); n < 8 {
			t.Errorf("%s blurred shadow has %d shadings, want at least 8", name, n)
		}
	}
}
