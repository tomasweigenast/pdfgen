package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestCanvasBackgroundPropagation(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`body_propagates`,
			`
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			BBBBBBBBBB
			`,
			`
			  <style>
			    @page { size: 10px }
			    /* body’s background propagates to the whole canvas */
			    body { margin: 2px; background: #00f; height: 5Px }
			  </style>
			<body>
			`,
		},
		{
			`html_blocks_body`,
			`
			rrrrrrrrrr
			rrrrrrrrrr
			rrBBBBBBrr
			rrBBBBBBrr
			rrBBBBBBrr
			rrBBBBBBrr
			rrBBBBBBrr
			rrrrrrrrrr
			rrrrrrrrrr
			rrrrrrrrrr
			`,
			`
			 <style>
			   @page { size: 10px }
			   /* html’s background propagates to the whole canvas */
			   html { padding: 1px; background: #f00 }
			   /* html has a background, so body’s does not propagate */
			   body { margin: 1px; background: #00f; height: 5px }
			</style>
			<body>
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestCanvasBackgroundSize(t *testing.T) {
	testutil.AssertPixels(t, `
		__________
		__________
		__RRRRRR__
		__RGGGGR__
		__zzzzzz__
		__zzzzzz__
		__BBBBBB__
		__BBBBBB__
		__________
		__________
	`, `
		<style>
		   @page { size: 10px; margin: 2px }
		   /* html’s background propagates to the whole canvas */
		   html { background: linear-gradient(
		     red 0, red 50%, blue 50%, blue 100%); }
		   /* html has a background, so body’s does not propagate */
		   body { margin: 1px; background: lime; height: 1px }
		</style>
		<body>
	`)
}

func TestBackgroundImageLayers(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`url(pattern.png)`,
			`
			______________
			______________
			__rBBBrBBBrB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			__rBBBrBBBrB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			__rBBBrBBBrB__
			__BBBBBBBBBB__
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) repeat-x`,
			`
			______________
			______________
			__rBBBrBBBrB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) repeat-x }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) repeat-y`,
			`
			______________
			______________
			__rBBB________
			__BBBB________
			__BBBB________
			__BBBB________
			__rBBB________
			__BBBB________
			__BBBB________
			__BBBB________
			__rBBB________
			__BBBB________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) repeat-y }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 0 0%`,
			`
			______________
			______________
			__rBBB________
			__BBBB________
			__BBBB________
			__BBBB________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 0 0% }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 50% 0px`,
			`
			______________
			______________
			_____rBBB_____
			_____BBBB_____
			_____BBBB_____
			_____BBBB_____
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 50% 0px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 6px top`,
			`
			______________
			______________
			________rBBB__
			________BBBB__
			________BBBB__
			________BBBB__
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 6px top }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat bottom 6PX right 0`,
			`
			______________
			______________
			________rBBB__
			________BBBB__
			________BBBB__
			________BBBB__
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat bottom 6PX right 0 }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat left center`,
			`
			______________
			______________
			______________
			______________
			______________
			__rBBB________
			__BBBB________
			__BBBB________
			__BBBB________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat left center }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat center left`,
			`
			______________
			______________
			______________
			______________
			______________
			__rBBB________
			__BBBB________
			__BBBB________
			__BBBB________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat center left }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 3px 3px`,
			`
			______________
			______________
			______________
			______________
			______________
			_____rBBB_____
			_____BBBB_____
			_____BBBB_____
			_____BBBB_____
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 3px 3px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 100% 50%`,
			`
			______________
			______________
			______________
			______________
			______________
			________rBBB__
			________BBBB__
			________BBBB__
			________BBBB__
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 100% 50% }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 0% bottom`,
			`
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			__rBBB________
			__BBBB________
			__BBBB________
			__BBBB________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 0% bottom }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat center 6px`,
			`
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			_____rBBB_____
			_____BBBB_____
			_____BBBB_____
			_____BBBB_____
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat center 6px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat bottom center`,
			`
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			_____rBBB_____
			_____BBBB_____
			_____BBBB_____
			_____BBBB_____
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat bottom center }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 6px 100%`,
			`
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			________rBBB__
			________BBBB__
			________BBBB__
			________BBBB__
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 6px 100% }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) repeat-x 1px 2px`,
			`
			______________
			______________
			______________
			______________
			__BrBBBrBBBr__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			__BBBBBBBBBB__
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) repeat-x 1px 2px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) repeat-y local 2pX 1px`,
			`
			______________
			______________
			____BBBB______
			____rBBB______
			____BBBB______
			____BBBB______
			____BBBB______
			____rBBB______
			____BBBB______
			____BBBB______
			____BBBB______
			____rBBB______
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) repeat-y local 2pX 1px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat fixed`,
			`
			# The image is actually here:
			#######
			______________
			______________
			__BB__________
			__BB__________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat fixed }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat fixed right 3px`,
			`
			#                   x x x x
			______________
			______________
			______________
			__________rB__   #
			__________BB__   #
			__________BB__   #
			__________BB__   #
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat fixed right 3px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png)no-repeat fixed 50%center`,
			`
			______________
			______________
			______________
			______________
			______________
			______________
			_____rBBB_____
			_____BBBB_____
			_____BBBB_____
			_____BBBB_____
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png)no-repeat fixed 50%center }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat, url(pattern.png) no-repeat 2px 1px`,
			`
			______________
			______________
			__rBBB________
			__BBBBBB______
			__BBBBBB______
			__BBBBBB______
			____BBBB______
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat, url(pattern.png) no-repeat 2px 1px }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
		{
			`url(pattern.png) no-repeat 2px 1px, url(pattern.png) no-repeat`,
			`
			______________
			______________
			__rBBB________
			__BBrBBB______
			__BBBBBB______
			__BBBBBB______
			____BBBB______
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 16px }
			  html { background: #fff }
			  body { margin: 2px; height: 10px; background: url(pattern.png) no-repeat 2px 1px, url(pattern.png) no-repeat }
			  p { background: none }
			</style>
			<body>
			<p>&nbsp;
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestZeroSizeBackgroundImage(t *testing.T) {
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
		  html { background: #fff }
		  body { background: url(pattern.png);
		         background-size: cover;
		         display: inline-block }
		</style>
		<body>
	`)
}

func TestBackgroundOrigin(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`border-box`,
			`
			____________
			____________
			____________
			____________
			____________
			____________
			____________
			_______rBBB_
			_______BBBB_
			_______BBBB_
			_______BBBB_
			____________
			`,
			`
			<style>
			  @page { size: 12px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 6px;
			         border: 1px solid transparent;
			         background: url(pattern.png) bottom right no-repeat;
			         background-origin: border-box }
			</style>
			<body>
			`,
		},
		{
			`padding-box`,
			`
			____________
			____________
			____________
			____________
			____________
			____________
			______rBBB__
			______BBBB__
			______BBBB__
			______BBBB__
			____________
			____________
			`,
			`
			<style>
			  @page { size: 12px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 6px;
			         border: 1px solid transparent;
			         background: url(pattern.png) bottom right no-repeat;
			         background-origin: padding-box }
			</style>
			<body>
			`,
		},
		{
			`content-box`,
			`
			____________
			____________
			____________
			____________
			____________
			_____rBBB___
			_____BBBB___
			_____BBBB___
			_____BBBB___
			____________
			____________
			____________
			`,
			`
			<style>
			  @page { size: 12px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 6px;
			         border: 1px solid transparent;
			         background: url(pattern.png) bottom right no-repeat;
			         background-origin: content-box }
			</style>
			<body>
			`,
		},
		{
			`border-box; background-clip: content-box`,
			`
			____________
			____________
			____________
			____________
			____________
			____________
			____________
			_______rB___
			_______BB___
			____________
			____________
			____________
			`,
			`
			<style>
			  @page { size: 12px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 6px;
			         border: 1px solid transparent;
			         background: url(pattern.png) bottom right no-repeat;
			         background-origin: border-box; background-clip: content-box }
			</style>
			<body>
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestBackgroundOfTransformedBox(t *testing.T) {
	t.Skip("not supported: transform matrices are not computed for pages yet")
	testutil.AssertPixels(t, `
		_______
		_RRRRR_
		_RRRRR_
		_RRRRR_
		_RRRRR_
		_RRRRR_
		_______
	`, `
		<style>
		  @page { size: 7px 7px }
		  html { background: #fff }
		  body { position: absolute;
		         width: 5px; height: 5px;
		         top: -5px; left: -5px;
		         transform: translate(6px,6px);
		         background: red }
		</style>
		<body>
	`)
}

func TestBackgroundRepeatSpace(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`repeat_space_1`,
			`
			____________
			_rBBB__rBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			____________
			_rBBB__rBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			____________
			_rBBB__rBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			____________
			`,
			`
			<style>
			  @page { size: 12px 16px }
			  html { background: #fff }
			  body { margin: 1px; height: 14px;
			         background: url(pattern.png) space }
			</style>
			<body>
			`,
		},
		{
			`repeat_space_2`,
			`
			____________
			_rBBB__rBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_rBBB__rBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_rBBB__rBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			_BBBB__BBBB_
			____________
			`,
			`
			<style>
			  @page { size: 12px 14px }
			  html { background: #fff }
			  body { margin: 1px; height: 12px;
			         background: url(pattern.png) space }
			</style>
			<body>
			`,
		},
		{
			`repeat_space_3`,
			`
			____________
			_rBBBrBBBrB_
			_BBBBBBBBBB_
			_BBBBBBBBBB_
			_BBBBBBBBBB_
			____________
			____________
			____________
			_rBBBrBBBrB_
			_BBBBBBBBBB_
			_BBBBBBBBBB_
			_BBBBBBBBBB_
			____________
			`,
			`
			<style>
			  @page { size: 12px 13px }
			  html { background: #fff }
			  body { margin: 1px; height: 11px;
			         background: url(pattern.png) repeat space }
			</style>
			<body>
			`,
		},
		{
			`repeat_space_4`,
			`
			________
			_rBBBGG_
			_BBBBGG_
			_BBBBGG_
			_BBBBGG_
			_GGGGGG_
			_GGGGGG_
			________
			`,
			`
			<style>
			  @page { size: 8px }
			  html { background: #fff }
			  body { margin: 1px; height: 6Px;
			         background: url(pattern.png) space lime }
			</style>
			<body>
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestBackgroundRepeatRound(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`repeat_round_1`,
			`
			__________
			_rrBBBBBB_
			_rrBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_rrBBBBBB_
			_rrBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			__________
			`,
			`
			<style>
			  @page { size: 10px 14px }
			  html { background: #fff }
			  body { margin: 1px; height: 12px;
			         image-rendering: pixelated;
			         background: url(pattern.png) top/6px round repeat; }
			</style>
			<body>
			`,
		},
		{
			`repeat_round_2`,
			`
			__________
			_rrBBBBBB_
			_rrBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_BBBBBBBB_
			_rrBBBBBB_
			_rrBBBBBB_
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
			  @page { size: 10px 18px }
			  html { background: #fff }
			  body { margin: 1px; height: 16px;
			         image-rendering: pixelated;
			         background: url(pattern.png) center/auto 8px repeat round; }
			</style>
			<body>
			`,
		},
		{
			`repeat_round_3`,
			`
			__________
			_rrBBBBBB_
			_rrBBBBBB_
			_rrBBBBBB_
			_BBBBBBBB_
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
			  @page { size: 10px 14px }
			  html { background: #fff }
			  body { margin: 1px; height: 12px;
			         image-rendering: pixelated;
			         background: url(pattern.png) center/6px 9px round; }
			</style>
			<body>
			`,
		},
		{
			`repeat_round_4`,
			`
			__________
			_rBBBrBBB_
			_rBBBrBBB_
			_rBBBrBBB_
			_BBBBBBBB_
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
			  @page { size: 10px 14px }
			  html { background: #fff }
			  body { margin: 1px; height: 12px;
			         image-rendering: pixelated;
			         background: url(pattern.png) center/5px 9px round; }
			</style>
			<body>
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestBackgroundClip(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`#00f border-box`,
			`
			________
			_BBBBBB_
			_BBBBBB_
			_BBBBBB_
			_BBBBBB_
			_BBBBBB_
			_BBBBBB_
			________
			`,
			`
			<style>
			  @page { size: 8px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 2px;
			         border: 1px solid transparent;
			         background: #00f border-box }
			</style>
			<body>
			`,
		},
		{
			`#00f padding-box`,
			`
			________
			________
			__BBBB__
			__BBBB__
			__BBBB__
			__BBBB__
			________
			________
			`,
			`
			<style>
			  @page { size: 8px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 2px;
			         border: 1px solid transparent;
			         background: #00f padding-box }
			</style>
			<body>
			`,
		},
		{
			`#00f content-box`,
			`
			________
			________
			________
			___BB___
			___BB___
			________
			________
			________
			`,
			`
			<style>
			  @page { size: 8px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 2px;
			         border: 1px solid transparent;
			         background: #00f content-box }
			</style>
			<body>
			`,
		},
		{
			`url(pattern.png) padding-box, #0f0`,
			`
			________
			_GGGGGG_
			_GrBBBG_
			_GBBBBG_
			_GBBBBG_
			_GBBBBG_
			_GGGGGG_
			________
			`,
			`
			<style>
			  @page { size: 8px }
			  html { background: #fff }
			  body { margin: 1px; padding: 1px; height: 2px;
			         border: 1px solid transparent;
			         background: url(pattern.png) padding-box, #0f0 }
			</style>
			<body>
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestBackgroundSize(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			`row_1`,
			`
			____________
			____________
			____________
			___rrBBBBBB_
			___rrBBBBBB_
			___BBBBBBBB_
			___BBBBBBBB_
			___BBBBBBBB_
			___BBBBBBBB_
			___BBBBBBBB_
			___BBBBBBBB_
			____________
			`,
			`
			<style>
			  @page { size: 12px }
			  html { background: #fff }
			  body { margin: 1px; height: 10px;
			         /* Use nearest neighbor algorithm for image resizing: */
			         image-rendering: pixelated;
			         background: url(pattern.png) no-repeat
			                     bottom right / 80% 8px; }
			</style>
			<body>
			`,
		},
		{
			`row_2`,
			`
			____________
			____________
			____________
			____________
			____________
			____________
			____________
			_______rBBB_
			_______BBBB_
			_______BBBB_
			_______BBBB_
			____________
			`,
			`
			<style>
			  @page { size: 12px }
			  html { background: #fff }
			  body { margin: 1px; height: 10px;
			         /* Use nearest neighbor algorithm for image resizing: */
			         image-rendering: pixelated;
			         background: url(pattern.png) bottom right/auto no-repeat }
			</style>
			<body>
			`,
		},
		{
			`row_3`,
			`
			______________
			_rrBBBBBB_____
			_rrBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			______________
			`,
			`
			<style>
			  @page { size: 14px 10px }
			  html { background: #fff }
			  body { margin: 1px; height: 8px;
			         /* Use nearest neighbor algorithm for image resizing: */
			         image-rendering: pixelated;
			         background: url(pattern.png) no-repeat;
			         background-size: contain }
			</style>
			<body>
			`,
		},
		{
			`row_4`,
			`
			______________
			_rrBBBBBB_____
			_rrBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			______________
			`,
			`
			<style>
			  @page { size: 14px 10px }
			  html { background: #fff }
			  body { margin: 1px; height: 8px;
			         /* Use nearest neighbor algorithm for image resizing: */
			         image-rendering: pixelated;
			         background: url(pattern.png) no-repeat left / auto 8px;
			         clip: auto; /* no-op to cover more validation */ }
			</style>
			<body>
			`,
		},
		{
			`row_5`,
			`
			______________
			_rrBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			_BBBBBBBB_____
			______________
			______________
			______________
			______________
			______________
			`,
			`
			<style>
			  @page { size: 14px 10px }
			  html { background: #fff }
			  body { margin: 1px; height: 8px;
			         /* Use nearest neighbor algorithm for image resizing: */
			         image-rendering: pixelated;
			         background: url(pattern.png) no-repeat 0 0 / 8px 4px;
			         clip: auto; /* no-op to cover more validation */ }
			</style>
			<body>
			`,
		},
		{
			`row_6`,
			`
			______________
			_rrrBBBBBBBBB_
			_rrrBBBBBBBBB_
			_rrrBBBBBBBBB_
			_BBBBBBBBBBBB_
			_BBBBBBBBBBBB_
			_BBBBBBBBBBBB_
			_BBBBBBBBBBBB_
			_BBBBBBBBBBBB_
			______________
			`,
			`
			<style>
			  @page { size: 14px 10px }
			  html { background: #fff }
			  body { margin: 1px; height: 8px;
			         /* Use nearest neighbor algorithm for image resizing: */
			         image-rendering: pixelated;
			         background: url(pattern.png) no-repeat right 0/cover }
			</style>
			<body>
			`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertPixels(t, tc.pixels, tc.html)
		})
	}
}

func TestBleedBackgroundSize(t *testing.T) {
	testutil.AssertPixels(t, `
		RRRR
		RRRR
		RRRR
		RRRR
	`, `
		<style>
		   @page { size: 2px; bleed: 1px; background: red }
		</style>
		<body>
	`)
}

func TestBackgroundSizeClip(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBB
		BRBB
		BBBB
		BBBB
	`, `
		<style>
		   @page { size: 4px; margin: 1px;
		           background: url(pattern.png) red;
		           background-clip: content-box }
		</style>
		<body>
	`)
}

func TestPageBackgroundFixed(t *testing.T) {
	testutil.AssertPixels(t, `
		RBBB
		BBBB
		BBBB
		BBBB
	`, `
		<style>
		   @page { size: 4px; margin: 1px;
		           background: url(pattern.png) red;
		           background-attachment: fixed; }
		</style>
		<body>
	`)
}

func TestPageBackgroundFixedBleed(t *testing.T) {
	testutil.AssertPixels(t, `
		RRRRRR
		RRBBBR
		RBBBBR
		RBBBBR
		RBBBBR
		RRRRRR
	`, `
		<style>
		   @page { size: 4px; margin: 1px; bleed: 1px;
		           background: url(pattern.png) no-repeat red;
		           background-attachment: fixed; }
		</style>
		<body>
	`)
}

func TestBleedBackgroundSizeClip(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBBBB
		BBBBBB
		BBRBBB
		BBBBBB
		BBBBBB
		BBBBBB
	`, `
		<style>
		   @page { size: 4px; bleed: 1px; margin: 1px;
		           background: url(pattern.png) red;
		           background-clip: content-box }
		</style>
		<body>
	`)
}

func TestCropMarks(t *testing.T) {
	testutil.AssertPixels(t, `
		KK__KK
		K____K
		______
		______
		K____K
		KK__KK
	`, `
		<style>
		   @page { size: 4px; bleed: 1px; margin: 1px; marks: crop }
		</style>
		<body>
	`)
}

func TestCrossMarks(t *testing.T) {
	testutil.AssertPixels(t, `
		__KK__
		______
		K____K
		K____K
		______
		__KK__
	`, `
		<style>
		   @page { size: 4px; bleed: 1px; margin: 1px; marks: cross }
		</style>
		<body>
	`)
}

func TestCropAndCrossMarks(t *testing.T) {
	testutil.AssertPixels(t, `
		KKKKKK
		K____K
		K____K
		K____K
		K____K
		KKKKKK
	`, `
		<style>
		   @page { size: 4px; bleed: 1px; margin: 1px; marks: crop cross }
		</style>
		<body>
	`)
}
