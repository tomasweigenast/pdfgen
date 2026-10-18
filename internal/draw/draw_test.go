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

const centeredImage = `
	________
	________
	__rBBB__
	__BBBB__
	__BBBB__
	__BBBB__
	________
	________
`

const blueImage = `
	________
	________
	__aaaa__
	__aaaa__
	__aaaa__
	__aaaa__
	________
	________
`

func TestImagesArePainted(t *testing.T) {
	tests := []struct {
		filename, pixels, skip string
	}{
		{"pattern.svg", centeredImage, "not supported: SVG images"},
		{"pattern.png", centeredImage, ""},
		{"pattern.palette.png", centeredImage, ""},
		{"pattern.gif", centeredImage, ""},
		{"blue.jpg", blueImage, ""},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, tt.pixels, `
				<style>
					@page { size: 8px }
					body { margin: 2px 0 0 2px; font-size: 0 }
					img { overflow: hidden }
				</style>
				<div><img src="`+tt.filename+`"></div>`)
		})
	}
}

func TestOverflowHiddenClipsDescendants(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		GGG_
		GRR_
		GRR_
		____
	`, `
		<style>@page { size: 4px }</style>
		<div style="overflow: hidden; background: lime; width: 3px; height: 3px">
			<div style="background: red; margin: 1px 0 0 1px; width: 5px; height: 5px"></div>
		</div>`)
}
