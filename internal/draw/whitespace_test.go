package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestWhitespaceInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR__RRRR____
		RRRR__RRRR____
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
		</style>
		<span>aa </span><span> aa</span>`)
}

func TestWhitespaceNestedInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR__RRRR____
		RRRR__RRRR____
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
		</style>
		<span><span>aa </span></span><span><span> aa</span></span>`)
}

func TestWhitespaceInlineSpaceBetween(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR__RRRR____
		RRRR__RRRR____
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
		</style>
		<span>aa </span> <span> aa</span>`)
}

func TestWhitespaceFloatBetween(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR__RRRR__BB
		RRRR__RRRR__BB
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
			div { float: right; color: blue }
		</style>
		<span>aa </span><div>a</div><span> aa</span>`)
}

func TestWhitespaceInFloat(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRR____BB
		RRRRRRRR____BB
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
			div { color: blue; float: right }
		</style>
		<span>aa</span><div> a </div><span>aa</span>`)
}

func TestWhitespaceAbsoluteBetween(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR__RRRR__BB
		RRRR__RRRR__BB
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
			div { color: blue; position: absolute; right: 0; top: 0 }
		</style>
		<span>aa </span><div>a</div><span> aa</span>`)
}

func TestWhitespaceInAbsolute(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRR____BB
		RRRRRRRR____BB
		______________
		______________
	`, `
		<style>
			@page { size: 14px 4px }
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
			div { color: blue; position: absolute; right: 0; top: 0 }
		</style>
		<span>aa</span><div> a </div><span>aa</span>`)
}

func TestWhitespaceRunningBetween(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR__RRRR____
		RRRR__RRRR____
		______BB______
		______BB______
	`, `
		<style>
			@page {
				size: 14px 4px; margin: 0 0 2px;
				@bottom-center { content: element(test) }
			}
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
			div { background: green; color: blue; position: running(test) }
		</style>
		<span>aa </span><div>a</div><span> aa</span>`)
}

func TestWhitespaceInRunning(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRR______
		RRRRRRRR______
		______BB______
		______BB______
	`, `
		<style>
			@page {
				size: 14px 4px; margin: 0 0 2px;
				@bottom-center { content: element(test) }
			}
			body { color: red; font-family: testfont; font-size: 2px; line-height: 1 }
			div { background: green; color: blue; position: running(test) }
		</style>
		<span>aa</span><div> a </div><span>aa</span>`)
}
