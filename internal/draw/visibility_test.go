package draw_test

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

const visibilitySource = `
  <style>
    @page { size: 12px 7px }
    body { font-size: 0; line-height: 0 }
    img { margin: 1px 0 0 1px }
    %s
  </style>
  <div>
    <img src="pattern.png">
    <span><img src="pattern.png"></span>
  </div>`

func TestVisibilityDefault(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____________
		_rBBB_rBBB__
		_BBBB_BBBB__
		_BBBB_BBBB__
		_BBBB_BBBB__
		____________
		____________
	`, fmt.Sprintf(visibilitySource, ``))
}

func TestVisibilityHiddenParent(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____________
		____________
		____________
		____________
		____________
		____________
		____________
	`, fmt.Sprintf(visibilitySource, `div { visibility: hidden }`))
}

func TestVisibilityVisibleChildOfHidden(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____________
		______rBBB__
		______BBBB__
		______BBBB__
		______BBBB__
		____________
		____________
	`, fmt.Sprintf(visibilitySource, `div { visibility: hidden } span { visibility: visible }`))
}

func TestVisibilityOfPageDoesNotInherit(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____________
		_rBBB_rBBB__
		_BBBB_BBBB__
		_BBBB_BBBB__
		_BBBB_BBBB__
		____________
		____________
	`, fmt.Sprintf(visibilitySource, `@page { visibility: hidden; background: red }`))
}
