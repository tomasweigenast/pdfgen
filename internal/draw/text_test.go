package draw_test

import (
	"fmt"
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

func TestTextAlignRight(t *testing.T) {
	testutil.AssertPixels(t, `
		_________
		__RR__RR_
		__RR__RR_
		______RR_
		______RR_
		_________
	`, `
		<style>
			@page { size: 9px 6px }
			body { color: red; font-family: testfont; font-size: 2px }
			div { line-height: 1; margin: 1px; text-align: right }
		</style>
		<div>a c e</div>`)
}

func TestTextAlignJustify(t *testing.T) {
	testutil.AssertPixels(t, `
		_________
		_RR___RR_
		_RR___RR_
		_RR______
		_RR______
		_________
	`, `
		<style>
			@page { size: 9px 6px }
			body { color: red; font-family: testfont; font-size: 2px }
			div { line-height: 1; margin: 1px; text-align: justify }
		</style>
		<div>a c e</div>`)
}

func TestTextAlignJustifyNbsp(t *testing.T) {
	testutil.AssertPixels(t, `
		___________________
		_RR___RR___RR___RR_
		_RR___RR___RR___RR_
		___________________
	`, `
		<style>
			@page { size: 19px 4px }
			body { color: red; font-family: testfont; font-size: 2px }
			div { line-height: 1; margin: 1px; text-align: justify-all }
		</style>
		<div>a b&nbsp;c&nbsp;d</div>`)
}

func TestTextWordSpacing(t *testing.T) {
	testutil.AssertPixels(t, `
		___________________
		_RR____RR____RR____
		_RR____RR____RR____
		___________________
	`, `
		<style>
			@page { size: 19px 4px }
			body { color: red; font-family: testfont; font-size: 2px }
			div { line-height: 1; margin: 1px; word-spacing: 1em }
		</style>
		<div>a c e</div>`)
}

func TestTextLetterSpacing(t *testing.T) {
	testutil.AssertPixels(t, `
		___________________
		_RR____RR____RR____
		_RR____RR____RR____
		___________________
	`, `
		<style>
			@page { size: 19px 4px }
			body { color: red; font-family: testfont; font-size: 2px }
			div { line-height: 1; margin: 1px; letter-spacing: 2em }
		</style>
		<div>ace</div>`)
}

// decorated renders abc in a 13px wide page whose margin is 2px, with the
// body style given.
func decorated(height int, bodyCSS string, extraCSS ...string) string {
	extra := ""
	for _, c := range extraCSS {
		extra += c
	}
	return fmt.Sprintf(`
		<style>
			@page { size: 13px %dpx; margin: 2px }
			body { font-family: testfont; font-size: 3px; %s }
			%s
		</style>
		<div>abc</div>`, height, bodyCSS, extra)
}

func TestTextUnderline(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zsssssssssz_
		_zsssssssssz_
		_zuuuuuuuuuz_
		_zzzzzzzzzzz_
		_____________
	`, decorated(7, "color: rgba(255, 0, 0, 0.5); text-decoration: underline blue auto"))
}

func TestTextUnderlineOffset(t *testing.T) {
	for _, offset := range []string{"2px", "70%", "calc(0.5em + 20%)"} {
		t.Run(offset, func(t *testing.T) {
			testutil.AssertPixels(t, `
				_____________
				_zzzzzzzzzzz_
				_zRRRRRRRRRz_
				_zRRRRRRRRRz_
				_zzzzzzzzzzz_
				_zzzzzzzzzzz_
				_zBBBBBBBBBz_
				_zzzzzzzzzzz_
				_____________
			`, decorated(9, "color: red; text-decoration: underline blue; text-underline-offset: "+offset))
		})
	}
}

func TestTextUnderlineThickness(t *testing.T) {
	for _, thickness := range []string{"3px", "100%", "calc(0.5em + 50%)"} {
		t.Run(thickness, func(t *testing.T) {
			testutil.AssertPixels(t, `
				_____________
				_zzzzzzzzzzz_
				_zRRRRRRRRRz_
				_zRRRRRRRRRz_
				_zzzzzzzzzzz_
				_zzzzzzzzzzz_
				_zBBBBBBBBBz_
				_zBBBBBBBBBz_
				_zzzzzzzzzzz_
			`, decorated(9, "color: red; text-decoration: underline blue "+thickness+"; text-underline-offset: 2px"))
		})
	}
}

func TestTextUnderlineThicknessCalcInherit(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zRRRRRRRRRz_
		_zRRRRRRRRRz_
		_zzzzzzzzzzz_
		_zzzzzzzzzzz_
		_zBBBBBBBBBz_
		_zBBBBBBBBBz_
		_zzzzzzzzzzz_
	`, `
		<style>
			@page { size: 13px 9px; margin: 2px }
			body {
				color: red; font-family: testfont; font-size: 3px;
				text-decoration: underline blue max(1rem, 1px);
				text-underline-offset: 2px
			}
			div::after { content: "abc" }
		</style>
		<div></div>`)
}

func TestTextOverline(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zzzzzzzzzzz_
		_zsssssssssz_
		_zsssssssssz_
		_zzzzzzzzzzz_
		_____________
	`, decorated(7, "color: rgba(255, 0, 0, 0.5); text-decoration: overline blue"))
}

func TestTextLineThrough(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zBBBBBBBBBz_
		_zuuuuuuuuuz_
		_zBBBBBBBBBz_
		_zzzzzzzzzzz_
		_____________
	`, decorated(7, "color: blue; text-decoration: line-through rgba(255, 0, 0, 0.5)"))
}

func TestTextMultipleTextDecoration(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zsssssssssz_
		_zBBBBBBBBBz_
		_zuuuuuuuuuz_
		_zzzzzzzzzzz_
		_____________
	`, decorated(7, "color: rgba(255, 0, 0, 0.5); text-decoration: underline line-through blue"))
}

func TestTextNestedTextDecoration(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zsssssssssz_
		_zsssBBBsssz_
		_zuuuuuuuuuz_
		_zzzzzzzzzzz_
		_____________
	`, `
		<style>
			@page { size: 13px 7px; margin: 2px }
			body {
				color: rgba(255, 0, 0, 0.5); font-family: testfont; font-size: 3px;
				text-decoration: underline blue
			}
			span { text-decoration: line-through blue }
		</style>
		<div>a<span>b</span>c</div>`)
}

func TestTextNestedTextDecorationColor(t *testing.T) {
	t.Skip("not supported: a nested decoration keeps the colour of the outer one")
}

func TestTextNestedBlockTextDecoration(t *testing.T) {
	t.Skip("not supported: decorations are not propagated to nested blocks")
}

func TestTextFloatTextDecoration(t *testing.T) {
	t.Skip("not supported: decorations are not propagated to floats")
}

func TestTextDecorationVar(t *testing.T) {
	testutil.AssertPixels(t, `
		_____________
		_zzzzzzzzzzz_
		_zRRRRRRRRRz_
		_zBBBBBBBBBz_
		_zRRRRRRRRRz_
		_zzzzzzzzzzz_
		_____________
	`, decorated(7, `--blue: blue; color: red;
		text-decoration-color: var(--blue); text-decoration-line: line-through`))
}
