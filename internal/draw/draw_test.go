package draw_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

func TestBlankPageIsWhite(t *testing.T) {
	testutil.AssertPixels(t, `
		____
		____
		____
		____
	`, `<style>@page { size: 4px }</style>`)
}

func TestColoredBlockIsPainted(t *testing.T) {
	testutil.AssertPixels(t, `
		RR__
		RR__
		____
		____
	`, `
		<style>@page { size: 4px }</style>
		<div style="background: red; width: 2px; height: 2px"></div>`)
}
