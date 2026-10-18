package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestOverflowClipsImage(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________
		________
		__rBBB__
		__BBBB__
		________
		________
		________
		________
	`, `
      <style>
        @page { size: 8px }
        body { margin: 2px 0 0 2px; font-size: 0 }
        div { height: 2px; overflow: hidden }
      </style>
      <div><img src="pattern.png"></div>`)
}

func TestOverflowPropagatesToViewport(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________
		________
		__rBBB__
		__BBBB__
		__BBBB__
		________
		________
		________
	`, `
      <style>
        @page { size: 8px; margin: 2px 2px 3px 2px }
        body { height: 1px; overflow: hidden; font-size: 0 }
      </style>
      <div><img src="pattern.png"></div>`)
}

func TestOverflowKeepsBorder(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		________
		________
		__BBBB__
		__B__B__
		__B__B__
		__BBBB__
		________
		________
	`, `
      <style>
        @page { size: 8px; margin: 2px; }
        div { width: 2px; height: 2px; overflow: hidden;
              border: 1px solid blue; }
      </style>
      <div></div>`)
}

func TestOverflowKeepsPageMargins(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		rr______
		rr______
		__BBBB__
		__BBBB__
		__BBBB__
		__BBBB__
		________
		________
	`, `
      <style>
        @page {
          size: 8px;
          margin: 2px;
          background:#fff;
          @top-left-corner { content: ''; background:#f00; } }
        body { overflow: auto; background:#00f; }
      </style>
      `)
}

func TestOverflowClipsMultilineText(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		BBBBBB__
		BBBBBB__
		BBBB____
		BBBB____
		BBBB____
		________
		________
		________
	`, `
      <style>
        @page { size: 8px }
        body { font-family: testfont; line-height: 1; font-size: 2px }
        p { color: blue }
      </style>
      <p>abc</p>
      <p style="height: 3px; overflow: hidden">ab<br>ab<br>ab<br>ab</p>
      `)
}

func TestClipProperty(t *testing.T) {
	testutil.AssertNoLogs(t)
	tests := []struct {
		name, clip, pixels string
	}{
		{`5px, 5px, 9px, auto`, `5px, 5px, 9px, auto`, `
		______________
		______________
		______________
		______________
		______________
		______________
		______rBBBrBg_
		______BBBBBBg_
		______BBBBBBg_
		______BBBBBBg_
		______________
		______________
		______________
		______________
		______________
		______________
		`},
		{`5px, 5px, auto, 10px`, `5px, 5px, auto, 10px`, `
		______________
		______________
		______________
		______________
		______________
		______________
		______rBBBr___
		______BBBBB___
		______BBBBB___
		______BBBBB___
		______rBBBr___
		______BBBBB___
		______ggggg___
		______________
		______________
		______________
		`},
		{`5px, auto, 9px, 10px`, `5px, auto, 9px, 10px`, `
		______________
		______________
		______________
		______________
		______________
		______________
		_grBBBrBBBr___
		_gBBBBBBBBB___
		_gBBBBBBBBB___
		_gBBBBBBBBB___
		______________
		______________
		______________
		______________
		______________
		______________
		`},
		{`auto, 5px, 9px, 10px`, `auto, 5px, 9px, 10px`, `
		______________
		______ggggg___
		______rBBBr___
		______BBBBB___
		______BBBBB___
		______BBBBB___
		______rBBBr___
		______BBBBB___
		______BBBBB___
		______BBBBB___
		______________
		______________
		______________
		______________
		______________
		______________
		`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertPixels(t, tt.pixels, `
			<style>
			  @page { size: 14px 16px }
			  div { margin: 1px; border: 1px green solid;
			        background: url(pattern.png);
			        position: absolute; /* clip only applies on abspos */
			        top: 0; bottom: 2px; left: 0; right: 0;
			        clip: rect(`+tt.clip+`); }
			</style>
			<div>`)
		})
	}
}
