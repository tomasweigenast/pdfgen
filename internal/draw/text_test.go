package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestTextOverflowClip(t *testing.T) {
	testutil.AssertPixels(t, `
		_________
		_RRRRRRR_
		_RRRRRRR_
		_________
		_RR__RRR_
		_RR__RRR_
		_________
	`, `
		<style>
			@page { size: 9px 7px }
			body { color: red; font-family: testfont; font-size: 2px }
			div { line-height: 1; margin: 1px; overflow: hidden; width: 3.5em }
		</style>
		<div>abcde</div>
		<div style="white-space: nowrap">a bcde</div>`)
}

func TestTextOverflowEllipsis(t *testing.T) {
	testutil.AssertPixels(t, `
		_________
		_RRRRRR__
		_RRRRRR__
		_________
		_RR__RR__
		_RR__RR__
		_________
		_RRRRRR__
		_RRRRRR__
		_________
		_RRRRRRR_
		_RRRRRRR_
		_________
		_RRRRRRR_
		_RRRRRRR_
		_________
	`, `
		<style>
			@page { size: 9px 16px }
			body { color: red; font-family: testfont; font-size: 2px }
			div {
				line-height: 1; margin: 1px; overflow: hidden;
				text-overflow: ellipsis; width: 3.5em
			}
			div div { margin: 0 }
		</style>
		<div>abcde</div>
		<div style="white-space: nowrap">a bcde</div>
		<div><span>a<span>b</span>cd</span>e</div>
		<div><div style="text-overflow: clip">abcde</div></div>
		<div><div style="overflow: visible">abcde</div></div>`)
}

func TestTextAlignRTLTrailingWhitespace(t *testing.T) {
	testutil.AssertPixels(t, `
		_________
		_rrrrBBB_
		_________
		_rrrrBBB_
		_________
		_BBBrrrr_
		_________
		_BBBrrrr_
		_________
	`, `
		<style>
			@page { size: 9px }
			body { font-family: testfont; color: blue; font-size: 1px }
			p { background: red; line-height: 1; width: 7em; margin: 1em }
		</style>
		<p style="direction: rtl"> abc </p>
		<p style="direction: rtl"> &#8207;abc </p>
		<p style="direction: ltr"> abc </p>
		<p style="direction: ltr"> &#8207;abc </p>`)
}

func TestRTLDefaultDirection(t *testing.T) {
	testutil.AssertPixels(t, `
		_____BBBBB_____
		_____BBBBB_____
		_____BBBBB_____
		_____BBBBB_____
		BBBBBBBBBB_____
	`, `
		<style>
			@page { size: 15px 5px }
			body { font-family: testfont; color: blue; font-size: 5px; line-height: 1 }
		</style>
		اب`)
}

func TestRTLForcedDirection(t *testing.T) {
	testutil.AssertPixels(t, `
		__________BBBBB
		__________BBBBB
		__________BBBBB
		__________BBBBB
		_____BBBBBBBBBB
	`, `
		<style>
			@page { size: 15px 5px }
			body { font-family: testfont; color: blue; font-size: 5px; line-height: 1 }
		</style>
		<div style="direction: rtl">اب</div>`)
}

func TestRTLNestedInline(t *testing.T) {
	testutil.AssertPixels(t, `
		RRRRR________________BBBBB___________RRRRR___________BBBBB
		RRRRR________________BBBBB___________RRRRR___________BBBBB
		RRRRR________________BBBBB___________RRRRR___________BBBBB
		RRRRR________________BBBBB___________RRRRR___________BBBBB
		RRRRRRRRRR______BBBBBBBBBB______RRRRRRRRRR______BBBBBBBBBB
		______________________________________BBBBB__________RRRRR
		______________________________________BBBBB__________RRRRR
		______________________________________BBBBB__________RRRRR
		______________________________________BBBBB__________RRRRR
		_________________________________BBBBBBBBBB_____RRRRRRRRRR
	`, `
		<style>
			@page { size: 58px 10px }
			body { font-family: testfont; color: blue; font-size: 5px; line-height: 1 }
			span { color: red }
		</style>
		<div style="direction: rtl; text-align: justify">
			اب <span>اب</span> اب <span>با اب</span> اب
		</div>`)
}

func TestMaxLinesEllipsis(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBBBBBB__
		BBBBBBBB__
		BBBBBBBBBB
		BBBBBBBBBB
		__________
		__________
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px 10px }
			p { block-ellipsis: auto; color: blue; font-family: testfont; font-size: 2px; max-lines: 2 }
		</style>
		<p>
			abcd efgh ijkl
		</p>`)
}

func TestMaxLinesNested(t *testing.T) {
	t.Skip("not supported: nested max-lines with continue: discard")
}

func TestLineClamp(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBBBBBBBB
		BBBBBBBBBB
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px 10px }
			p { color: blue; font-family: testfont; font-size: 2px; line-clamp: 3 "(…)" }
		</style>
		<p>
			aa a
			bb b
			cc c
			dddd
			eeee
			ffff
			gggg
			hhhh
		</p>`)
}

func TestLineClampNone(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px 10px }
			p {
				color: blue; font-family: testfont; font-size: 2px;
				max-lines: 1; continue: discard; block-ellipsis: "…"; line-clamp: none
			}
		</style>
		<p>
			aa a
			bb b
			cc c
		</p>`)
}

func TestLineClampNumber(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BBBB
		BBBB__BBBB
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px 10px }
			p { color: blue; font-family: testfont; font-size: 2px; line-clamp: 3 }
		</style>
		<p>
			aa a
			bb b
			cc c
			dddd
			eeee
		</p>`)
}

func TestLineClampNested(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBBBBBBBB
		BBBBBBBBBB
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px 10px }
			div { color: blue; font-family: testfont; font-size: 2px; line-clamp: 3 "(…)" }
		</style>
		<div>
			aa a
			<p>
				bb b
				cc c
				dddd
				eeee
				ffff
				gggg
				hhhh
			</p>
		</div>`)
}

func TestLineClampNestedAfter(t *testing.T) {
	testutil.AssertPixels(t, `
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBB__BB__
		BBBBBBBBBB
		BBBBBBBBBB
		__________
		__________
		__________
		__________
	`, `
		<style>
			@page { size: 10px 10px }
			div { color: blue; font-family: testfont; font-size: 2px; line-clamp: 3 "(…)" }
		</style>
		<div>
			aa a
			<p>
				bb b
			</p>
			cc c
			dddd
			eeee
			ffff
			gggg
			hhhh
		</div>`)
}

func TestEllipsisNested(t *testing.T) {
	t.Skip("not supported: block-ellipsis across nested blocks")
}
