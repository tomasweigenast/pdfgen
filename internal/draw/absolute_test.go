package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestAbsoluteSplit1(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRR____
		BBBBRRRRRRRR____
		BBBBRR__________
		BBBBRR__________
	`, `
        <style>
            @page {
                size: 16px 2Px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                left: 0;
                position: absolute;
                top: 0;
                width: 4px;
            }
        </style>
        <div class="split">aa aa</div>
        <div>bbbbbb bbb</div>`)
}

func TestAbsoluteSplit2(t *testing.T) {
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
                position: absolute;
                top: 0;
                right: 0;
                width: 4px;
            }
        </style>
        <div class="split">aa aa</div>
        <div>bbbbbb bb</div>`)
}

func TestAbsoluteSplit3(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRR____
		BBBBRRRRRRRR____
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
                font-size: 2PX;
                line-height: 1;
            }
            div.split {
                color: blue;
                position: absolute;
                top: 0;
                left: 0;
                width: 4px;
            }
        </style>
        <div class="split">aa</div>
        <div>bbbbbb bbbbb</div>`)
}

func TestAbsoluteSplit4(t *testing.T) {
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
                position: absolute;
                top: 0;
                right: 0;
                width: 4px;
            }
        </style>
        <div class="split">aa</div>
        <div>bbbbbb bbbbb</div>`)
}

func TestAbsoluteSplit5(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRR____gggg
		BBBBRRRR____gggg
		BBBBRRRRRR__gggg
		BBBBRRRRRR__gggg
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
                position: absolute;
                top: 0;
                left: 0;
                width: 4px;
            }
            div.split2 {
                color: green;
                position: absolute;
                top: 0;
                right: 0;
                width: 4px;
        </style>
        <div class="split">aa aa</div>
        <div class="split2">cc cc</div>
        <div>bbbb bbbbb</div>`)
}

func TestAbsoluteSplit6(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRR____gggg
		BBBBRRRR____gggg
		BBBBRRRRRR______
		BBBBRRRRRR______
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
                position: absolute;
                width: 4px;
            }
            div.split2 {
                color: green;
                position: absolute;
                top: 0;
                right: 0;
                width: 4px;
        </style>
        <div class="split">aa aa</div>
        <div class="split2">cc</div>
        <div>bbbb bbbbb</div>`)
}

func TestAbsoluteSplit7(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBRRRRRRRRgggg
		BBBBRRRRRRRRgggg
		____RRRR____gggg
		____RRRR____gggg
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
                position: absolute;
                width: 4px;
            }
            div.split2 {
                color: green;
                position: absolute;
                top: 0;
                right: 0;
                width: 4px;
            }
            div.push {
                margin-left: 4px;
            }
        </style>
        <div class="split">aa</div>
        <div class="split2">cc cc</div>
        <div class="push">bbbb bb</div>`)
}

func TestAbsoluteSplit8(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		______
		______
		______
		__RR__
		__RR__
		______
		______
	`, `
        <style>
            @page {
                margin: 2px 0;
                size: 6px 8px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div {
                position: absolute;
                left: 2px;
                top: 2px;
                width: 2px;
            }
        </style>
        <div>a a a a</div>`)
}

func TestAbsoluteSplit9(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		______
		BBRRBB
		BBRRBB
		BBRR__
		BBRR__
		______
		______
	`, `
        <style>
            @page {
                margin: 2px 0;
                size: 6px 8px;
            }
            body {
                color: blue;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div {
                color: red;
                position: absolute;
                left: 2px;
                top: 0;
                width: 2px;
            }
        </style>
        aaa a<div>a a a a</div>`)
}

func TestAbsoluteSplit10(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BB____
		BB____
		__RR__
		__RR__
		__RR__
		__RR__
		
		BBRR__
		BBRR__
		__RR__
		__RR__
		______
		______
	`, `
        <style>
            @page {
                size: 6px;
            }
            body {
                color: blue;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div {
                color: red;
                position: absolute;
                left: 2px;
                top: 2px;
                width: 2px;
            }
            div + article {
                break-before: page;
            }
        </style>
        <article>a</article>
        <div>a a a a</div>
        <article>a</article>`)
}

func TestAbsoluteSplit11(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBBB
		BBBBBB
		BBRRBB
		BBRRBB
		__RR__
		__RR__
	`, `
        <style>
            @page {
                size: 6px;
            }
            body {
                color: blue;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div {
                bottom: 0;
                color: red;
                position: absolute;
                left: 2px;
                width: 2px;
            }
        </style>
        aaa aaa<div>a a</div>`)
}

func TestAbsoluteSplit12(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBBB__
		BBBBBB__
		________
		________
		________
		________
		________
		________
		BB______
		BB______
		BBRR____
		BBRR____
		BBRRRR__
		BBRRRR__
		BBRRRRRR
		BBRRRRRR
	`, `
        <style>
            @page {
                size: 8px;
            }
            body {
                color: blue;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div {
                break-inside: avoid;
            }
            section {
                left: 2px;
                position: absolute;
                color: red;
            }
        </style>
        aaa
        <div>
          a
          <section>x<br>xx<br>xxx</section>
          <br>
          a<br>
          a<br>
          a
        </div>`)
}

func TestAbsoluteNextPage(t *testing.T) {
	t.Skip("not supported: an absolute box that does not fit the page is still rendered on it")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRRRRRR______
		RRRRRRRRRR______
		RRRRRRRRRR______
		RRRRRRRRRR______
		BBBBBBRRRR______
		BBBBBBRRRR______
		BBBBBB__________
		________________
	`, `
        <style>
            @page {
                size: 16px 4px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div.split {
                color: blue;
                position: absolute;
                font-size: 3px;
            }
        </style>
        aaaaa aaaaa
        <div class="split">bb</div>
        aaaaa`)
}

func TestAbsoluteRtl1(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		__________RRRRRR
		__________RRRRRR
		________________
	`, `
        <style>
            @page {
                size: 16px 3px;
            }
            body {
                direction: rtl;
            }
            div {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
                position: absolute;
            }
        </style>
        <div>bbb</div>`)
}

func TestAbsoluteRtl2(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________________
		_________RRRRRR_
		_________RRRRRR_
	`, `
        <style>
            @page {
                size: 16px 3px;
            }
            body {
                direction: rtl;
            }
            div {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
                padding: 1px;
                position: absolute;
            }
        </style>
        <div>bbb</div>`)
}

func TestAbsoluteRtl3(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________________
		RRRRRR__________
		RRRRRR__________
	`, `
        <style>
            @page {
                size: 16px 3px;
            }
            body {
                direction: rtl;
            }
            div {
                bottom: 0;
                color: red;
                font-family: testfont;
                font-size: 2px;
                left: 0;
                line-height: 1;
                position: absolute;
            }
        </style>
        <div>bbb</div>`)
}

func TestAbsoluteRtl4(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________________
		_________RRRRRR_
		_________RRRRRR_
	`, `
        <style>
            @page {
                size: 16px 3px;
            }
            body {
                direction: rtl;
            }
            div {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
                position: absolute;
                right: 1px;
                top: 1px;
            }
        </style>
        <div>bbb</div>`)
}

func TestAbsoluteRtl5(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		RRRRRR__________
		RRRRRR__________
		________________
	`, `
        <style>
            @page {
                size: 16px 3px;
            }
            div {
                color: red;
                direction: rtl;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
                position: absolute;
            }
        </style>
        <div>bbb</div>`)
}

func TestAbsolutePagesCounter(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		_RR___
		_RR___
		_RR___
		_RR___
		_____B
		______
		_RR___
		_RR___
		_BB___
		_BB___
		_____B
	`, `
        <style>
            @page {
                font-family: testfont;
                margin: 1px;
                size: 6px 6px;
                @bottom-right-corner {
                    color: blue;
                    content: counter(pages);
                    font-size: 1px;
                }
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
            }
            div {
                color: blue;
                position: absolute;
            }
        </style>
        a a a <div>a a</div>`)
}

func TestAbsolutePagesCounterOrphans(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		_RR___
		_RR___
		_RR___
		_RR___
		______
		______
		______
		_____B
		______
		_RR___
		_RR___
		_BB___
		_BB___
		_GG___
		_GG___
		______
		_____B
	`, `
        <style>
            @page {
                font-family: testfont;
                margin: 1px;
                size: 6px 9px;
                @bottom-right-corner {
                    color: blue;
                    content: counter(pages);
                    font-size: 1PX;
                }
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
                orphans: 2;
                widows: 2;
            }
            div {
                color: blue;
                position: absolute;
            }
            div ~ div {
                color: lime;
            }
        </style>
        a a a <div>a a a</div> a <div>a a a</div>`)
}

func TestAbsoluteInInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		_GG___
		_GG___
		_GG___
		_GG___
		______
		______
		______
		______
		
		______
		_RR___
		_RR___
		_RR___
		_RR___
		_BB___
		_BB___
		______
		______
	`, `
        <style>
            @page {
                margin: 1px;
                size: 6px 9px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2px;
                line-height: 1;
                orphans: 2;
                widows: 2;
            }
            p {
                color: lime;
            }
            div {
                color: blue;
                position: absolute;
            }
        </style>
        <p>a a</p> a a <div>a</div>`)
}

func TestFixedInInline(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		______
		_GG___
		_GG___
		_GG___
		_GG___
		_BB___
		_BB___
		______
		______
		
		______
		_RR___
		_RR___
		_RR___
		_RR___
		_BB___
		_BB___
		______
		______
	`, `
        <style>
            @page {
                margin: 1px;
                size: 6px 9px;
            }
            body {
                color: red;
                font-family: testfont;
                font-size: 2PX;
                line-height: 1;
                orphans: 2;
                widows: 2;
            }
            p {
                color: lime;
            }
            div {
                color: blue;
                position: fixed;
            }
        </style>
        <p>a a</p> a a <div>a</div>`)
}

func TestAbsoluteImageBackground(t *testing.T) {
	t.Skip("not supported: SVG images")
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____
		_RBB
		_BBB
		_BBB
	`, `
        <style>
          @page {
            size: 4px;
          }
          img {
            background: blue;
            position: absolute;
            top: 1px;
            left: 1px;
          }
        </style>
        <img src="pattern-transparent.svg" />`)
}

func TestAbsoluteInAbsoluteBreak(t *testing.T) {
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
		
		BBBB
		BBBB
		RRRR
		RRRR
		RRRR
		
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
            font-size: 2px;
            line-height: 1;
          }
          div {
            position: absolute;
            width: 100%;
          }
        </style>
        <div style="background: blue">
          <br><br><br><br><br>
          <div style="background: red">
            <br><br>
          </div>
        </div>
        <br><br><br><br><br><br><br>`)
}

func TestAbsoluteAlternativeLayout(t *testing.T) {
	tests := []struct {
		position, display, skip string
	}{
		{"absolute", "grid", "not supported: grid layout"},
		{"absolute", "flex", "not supported: flex layout"},
		{"relative", "grid", "not supported: grid layout"},
		{"relative", "flex", "not supported: flex layout"},
	}
	for _, tt := range tests {
		t.Run(tt.position+"/"+tt.display, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, `
				______
				_RRRR_
				_RBBR_
				_RBBR_
				_RRRR_
				______
			`, `
				<style>
					@page { size: 6px }
					div {
						background: blue;
						position: `+tt.position+`;
						display: `+tt.display+`;
						top: 1px;
						left: 1px;
						border: 1px solid red;
						font: 2px testfont;
						color: transparent;
						width: 2px;
					}
				</style>
				<div>a</div>`)
		})
	}
}
