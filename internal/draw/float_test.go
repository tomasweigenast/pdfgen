package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestFloat(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB__aaaa
		BBBB__aaaa
		BBBB__aaaa
		BBBB__aaaa
		__________
	`, `
      <style>
        @page { size: 10px 5px }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
      </div>`)
}

func TestFloatRtl(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB__aaaa
		BBBB__aaaa
		BBBB__aaaa
		BBBB__aaaa
		__________
	`, `
      <style>
        @page { size: 10px 5Px }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
      </div>`)
}

func TestFloatInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBBGG_____aaaa
		BBBBGG_____aaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <span>a</span>
      </div>`)
}

func TestFloatInlineRtl(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB_____GGaaaa
		BBBB_____GGaaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <span>a</span>
      </div>`)
}

func TestFloatInlineBlock(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBBGG_____aaaa
		BBBBGG_____aaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <span style="display: inline-block">a</span>
      </div>`)
}

func TestFloatInlineBlockRtl(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB_____GGaaaa
		BBBB_____GGaaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15Px 5px }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <span style="display: inline-block">a</span>
      </div>`)
}

func TestFloatTable(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBBGG_____aaaa
		BBBBGG_____aaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <table><tbody><tr><td>a</td></tr></tbody></table>
      </div>`)
}

func TestFloatTableRtl(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB_____GGaaaa
		BBBB_____GGaaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <table><tbody><tr><td>a</td></tr></tbody></table>
      </div>`)
}

func TestFloatInlineTable(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBBGG_____aaaa
		BBBBGG_____aaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15PX 5px }
        table { display: inline-table }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <table><tbody><tr><td>a</td></tr></tbody></table>
      </div>`)
}

func TestFloatInlineTableRtl(t *testing.T) {
	t.Skip("not supported: table layout")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB_____GGaaaa
		BBBB_____GGaaaa
		BBBB_______aaaa
		BBBB_______aaaa
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        table { display: inline-table }
        body { font-family: testfont; font-size: 2px; line-height: 1;
               color: lime }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="blue.jpg">
        <table><tbody><tr><td>a</td></tr></tbody></table>
      </div>`)
}

func TestFloatReplacedBlock(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBBaaaa___rBBB
		BBBBaaaa___BBBB
		BBBBaaaa___BBBB
		BBBBaaaa___BBBB
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="pattern.png">
        <img style="display: block" src="blue.jpg">
      </div>`)
}

func TestFloatReplacedBlockRtl(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB___aaaarBBB
		BBBB___aaaaBBBB
		BBBB___aaaaBBBB
		BBBB___aaaaBBBB
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="pattern.png">
        <img style="display: block" src="blue.jpg">
      </div>`)
}

func TestFloatReplacedInline(t *testing.T) {
	t.Skip("not supported: an inline image between two floats moves to the following page")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBBaaaa___rBBB
		BBBBaaaa___BBBB
		BBBBaaaa___BBBB
		BBBBaaaa___BBBB
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { line-height: 1px }
      </style>
      <div>
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="pattern.png">
        <img src="blue.jpg">
      </div>`)
}

func TestFloatReplacedInlineRtl(t *testing.T) {
	t.Skip("not supported: an inline image between two floats moves to the following page")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rBBB___aaaarBBB
		BBBB___aaaaBBBB
		BBBB___aaaaBBBB
		BBBB___aaaaBBBB
		_______________
	`, `
      <style>
        @page { size: 15px 5px }
        body { line-height: 1px }
      </style>
      <div style="direction: rtl">
        <img style="float: left" src="pattern.png">
        <img style="float: right" src="pattern.png">
        <img src="blue.jpg">
      </div>`)
}

func TestFloatMargin(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRRR__
		BBBBRRRRRRRRRR__
		__RRRRRRRRRR____
		__RRRRRRRRRR____
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                width: 4px;
            }
            div.pushed {
                margin-left: 2px;
            }
        </style>
        <div class="split">aa</div>
        <div class="pushed">bbbbb bbbbb</div>`)
}

func TestFloatSplit1(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRRRRR
		BBBBRRRRRRRRRRRR
		BBBBRRRR________
		BBBBRRRR________
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                width: 4px;
            }
        </style>
        <div class="split">aa aa</div>
        <div>bbbbbb bb</div>`)
}

func TestFloatSplit2(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRRRRRRBBBB
		RRRRRRRRRRRRBBBB
		RRRR________BBBB
		RRRR________BBBB
	`, `
        <style>
          @page {
            size: 16px 2px;
          }
          body {
            color: red;
            font-family: testfont;
            font-size: 2px;
            line-height: 1;
          }
          div.split {
            color: blue;
            float: right;
            width: 4px;
          }
        </style>
        <div class="split">aa aa</div>
        <div>bbbbbb bb</div>`)
}

func TestFloatSplit3(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRRRRR
		BBBBRRRRRRRRRRRR
		RRRRRRRRRR______
		RRRRRRRRRR______
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                width: 4px;
            }
        </style>
        <div class="split">aa</div>
        <div>bbbbbb bbbbb</div>`)
}

func TestFloatSplit4(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRRRRRRBBBB
		RRRRRRRRRRRRBBBB
		RRRRRRRRRR______
		RRRRRRRRRR______
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: right;
                width: 4px;
            }
        </style>
        <div class="split">aa</div>
        <div>bbbbbb bbbbb</div>`)
}

func TestFloatSplit5(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRgggg
		BBBBRRRRRRRRgggg
		BBBBRRRR____gggg
		BBBBRRRR____gggg
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                width: 4px;
            }
            div.split2 {
                color: green;
                float: right;
                width: 4px;
        </style>
        <div class="split">aa aa</div>
        <div class="split2">cc cc</div>
        <div>bbbb bb</div>`)
}

func TestFloatSplit6(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRgggg
		BBBBRRRRRRRRgggg
		BBBBRRRR________
		BBBBRRRR________
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                width: 4px;
            }
            div.split2 {
                color: green;
                float: right;
                width: 4px;
        </style>
        <div class="split">aa aa</div>
        <div class="split2">cc</div>
        <div>bbbb bb</div>`)
}

func TestFloatSplit7(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRgggg
		BBBBRRRRRRRRgggg
		RRRR________gggg
		RRRR________gggg
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                width: 4px;
            }
            div.split2 {
                color: green;
                float: right;
                width: 4px;
        </style>
        <div class="split">aa</div>
        <div class="split2">cc cc</div>
        <div>bbbb bb</div>`)
}

func TestFloatSplit8(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBB__RRRRRRRRRR
		BBBB__RRRRRRRRRR
		BBBB__RRRR______
		BBBB__RRRR______
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: left;
                margin-right: 2px;
                width: 4px;
            }
        </style>
        <div class="split">aa aa</div>
        <div>bbbbb bb</div>`)
}

func TestFloatSplit9(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRRRRBBBB__
		RRRRRRRRRRBBBB__
		RRRR______BBBB__
		RRRR______BBBB__
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: right;
                margin-right: 2px;
                width: 4px;
            }
        </style>
        <div class="split">aa aa</div>
        <div>bbbbb bb</div>`)
}

func TestFloatSplit10(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRRRR______
		RRRRRRRRRR______
		RRRRRRRRRR______
		RRRRRRRRRR______
		________________
		RRRRRR____BBBB__
		RRRRRR____BBBB__
		RRRRRR____BBBB__
		RRRRRR____BBBB__
		________________
	`, `
        <style>
            @page {
                size: 16px 5px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: right;
                margin-right: 2px;
                width: 4px;
            }
        </style>
        <div>bbbbb bbbbb</div>
        <div class="split">aa aa</div>
        <div>bbb bbb</div>`)
}

func TestFloatSplit11(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________________
		_BBBBBBBBBB_____
		_BBBBBBBBBB_____
		_BBBBBBBBBB_____
		_BBBBBBBBBB_____
		________________
		________________
		________________
		_BBBB___________
		_BBBB___________
		_rrrrrrrrrrrrrr_
		_rrrrrrrrrrrrrr_
		________________
		________________
	`, `
        <style>
            @page {
                margin: 1px;
                size: 16px 7px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                float: right;
            }
        </style>
        <div class="split">aaaaa aaaaa aa</div>
        bbbbbbb`)
}

func TestFloatSplit12(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBGG______BBBB
		BBBBGG______BBBB
		BBBB________BBBB
		BBBB________BBBB
		________________
	`, `
        <style>
            @page {
                size: 16px 5px;
            }
            body {
                color: lime;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            article {
                background: blue;
                height: 5px;
            }
            div {
                background: red;
                color: blue;
            }
        </style>
        <article></article>
        <section>
          a
          <div style="float: left"><p>aa<p>aa</div>
          <div style="float: right"><p>bb<p>bb</div>`)
}

func TestFloatSplit13(t *testing.T) {
	t.Skip("not supported: a float split across pages next to a line of text is laid out differently")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBGG______BBBB
		BBBBGG______BBBB
		BBBB________BBBB
		BBBB________BBBB
		________________
	`, `
        <style>
            @page {
                size: 16px 5px;
            }
            body {
                color: lime;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            article {
                background: blue;
                height: 5px;
            }
            div {
                background: red;
                color: blue;
            }
        </style>
        <article></article>
        <section>
          <div style="float: left"><p>a<p>aa</div>
          a
          <div style="float: right"><p>bb<p>bb</div>`)
}

func TestFloatSplit14(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBBBBBBBBBBBBB
		BBBBGG______BBBB
		BBBBGG______BBBB
		BBBB________BBBB
		BBBB________BBBB
		________________
	`, `
        <style>
            @page {
                size: 16px 5px;
            }
            body {
                color: lime;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            article {
                background: blue;
                height: 5px;
            }
            div {
                background: red;
                color: blue;
            }
        </style>
        <article></article>
        a
        <div style="float: left"><p>aa<p>aa</div>
        <div style="float: right"><p>bb<p>bb</div>`)
}

func TestFloatSplit15(t *testing.T) {
	t.Skip("not supported: a float split across pages with a nested float is laid out differently")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BB__RRRRRRRRRR__
		BB__RRRRRRRRRR__
		BB__RRRRRRRRRR__
		BB__RRRRRRRRRR__
		GGBBRRRRRRRRRR__
		GGBBRRRRRRRRRR__
		GGBBRRRRRRRRRR__
		GGBBRRRRRRRRRR__
		RRRRRRRRRR______
		RRRRRRRRRR______
	`, `
        <style>
            @page {
                size: 16px 2px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
        </style>
        <div style="float: left; position: relative; color: blue; width: 4px">
          a a a
          <div style="float: left; color: lime; width: 2px">
            a a
          </div>
          a a
        </div>
        <div>bbbbb bbbbb bbbbb bbbbb bbbbb</div>`)
}

func TestFloatSplitClear(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		rrrr
		rrrr
	`, `
        <style>
            @page {
                size: 4px;
            }
            body {
                color: red;
                font: 2px / 1 testfont;
            }
            div {
                color: blue;
                float: left;
            }
            article {
                clear: left;
            }
        </style>
        <div>bb bb bb bb bb</div>
        <article>aa</article>`)
}

func TestFloatSplitClearFloat(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		rrrr
		rrrr
	`, `
        <style>
            @page {
                size: 4px;
            }
            body {
                color: red;
                font: 2px / 1 testfont;
            }
            div {
                color: blue;
                float: left;
            }
            article {
                clear: left;
                float: left;
                text-decoration: underline;
            }
        </style>
        <div>bb bb bb bb bb</div>
        <article>aa</article>`)
}

func TestFloatSplitClearEmpty(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		BBBB
		____
		____
	`, `
        <style>
            @page {
                size: 4px;
            }
            body {
                color: red;
                font: 2px / 1 testfont;
            }
            div {
                color: blue;
                float: left;
            }
            article {
                clear: left;
            }
        </style>
        <div>bb bb bb bb bb</div>
        <article></article>`)
}

func TestFloatSplitLargeParent(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBB
		BBBB
		BBBB
		BBBB
		GGGG
		
		BBBB
		BBBB
		BBBB
		BBBB
		GGGG
		
		BBBB
		BBBB
		GGGG
		GGGG
		GGGG
		
		rrrr
		rrrr
		GGGG
		GGGG
		GGGG
	`, `
        <style>
            @page {
                size: 4px 5px;
            }
            body {
                color: red;
                font: 2px / 1 testfont;
            }
            section {
                background: lime;
                height: 100px;
            }
            div {
                color: blue;
                float: left;
            }
            article {
                clear: left;
            }
        </style>
        <section>
        <div>bb bb bb bb bb</div>
        <article>aa</article>`)
}

func TestFloatSplitInStackingContext(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRR
		RRRR
		BBBB
		BBBB
		____
		
		BBBB
		BBBB
		RRRR
		RRRR
		____
		
		RRRR
		RRRR
		____
		____
		____
	`, `
        <style>
            @page {
                size: 4px 5px;
            }
            body {
                color: red;
                font: 2px / 1 testfont;
            }
            section {
                display: flow-root;
            }
            div {
                color: blue;
                float: left;
            }
            article {
                clear: left;
            }
        </style>
        <section>
        aa
        <div>bb bb</div>
        cc cc`)
}
