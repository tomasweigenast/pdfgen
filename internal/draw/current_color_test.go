package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestCurrentColorBackground(t *testing.T) {
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  html, body { height: 100%; margin: 0 }
		  html { color: red; background: currentColor }
		  body { color: lime; background: inherit }
		</style>
		<body>
	`)
}

func TestCurrentColorInheritedBorder(t *testing.T) {
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  html { color: red; border-color: currentColor }
		  body { color: lime; border: 1Px solid; border-color: inherit;
		         margin: 0 }
		</style>
		<body>
	`)
}

func TestCurrentColorInheritedOutline(t *testing.T) {
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  html { color: red; outline-color: currentColor }
		  body { color: lime; outline: 1px solid; outline-color: inherit;
		         margin: 1px }
		</style>
		<body>
	`)
}

func TestCurrentColorCollapsedBorder(t *testing.T) {
	t.Skip("not supported: tables")
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  html { color: red; border-color: currentColor; }
		  body { margin: 0 }
		  table { border-collapse: collapse;
		          color: lime; border: 1PX solid; border-color: inherit }
		</style>
		<table><td>
	`)
}

func TestCurrentColorSVGFill(t *testing.T) {
	t.Skip("not supported: SVG images")
	testutil.AssertPixels(t, `
		KK
		KK
	`, `
		<style>
		  @page { size: 2px }
		  svg { display: block }
		</style>
		<svg xmlns="http://www.w3.org/2000/svg"
		     width="2" height="2" fill="currentColor">
		  <rect width="2" height="2"></rect>
		</svg>
	`)
}

func TestCurrentColorSVGInherited(t *testing.T) {
	t.Skip("not supported: currentColor inherited into SVG fill")
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  svg { display: block }
		  body { color: lime }
		</style>
		<svg xmlns="http://www.w3.org/2000/svg"
		     width="2" height="2">
		  <rect width="2" height="2" fill="currentColor"></rect>
		</svg>
	`)
}

func TestCurrentColorVariable(t *testing.T) {
	t.Skip("not supported: text is not drawn yet")
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  html { color: lime; font-family: testfont; --var: currentColor }
		  div { color: var(--var); font-size: 2px; line-height: 1 }
		</style>
		<div>aa
	`)
}

func TestCurrentColorVariableBorder(t *testing.T) {
	testutil.AssertPixels(t, `
		GG
		GG
	`, `
		<style>
		  @page { size: 2px }
		  html { color: lime; --var: currentColor }
		  div { color: var(--var); width: 0; height: 0; border: 1px solid }
		</style>
		<div>
	`)
}
