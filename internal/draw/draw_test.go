package draw_test

import (
	"fmt"
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

const opacitySource = `
	<style>
		@page { size: 60px 60px }
		div { background: #000; width: 20px; height: 20px }
	</style>
	%s`

func opacityDocuments(bodies ...string) []string {
	docs := make([]string, len(bodies))
	for i, body := range bodies {
		docs[i] = fmt.Sprintf(opacitySource, body)
	}
	return docs
}

func TestOpacityZeroPaintsNothing(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`<div></div>`,
		`<div></div><div style="opacity: 0"></div>`,
		`<div></div><div style="opacity: 0%"></div>`,
	)...)
}

func TestOpacityBlendsWithBackdrop(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`<div style="background: rgb(102, 102, 102)"></div>`,
		`<div style="opacity: 0.6"></div>`,
		`<div style="opacity: 60%"></div>`,
		`<div style="opacity: 60.0%"></div>`,
	)...)
}

func TestNestedOpacityMultiplies(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0, opacityDocuments(
		`<div style="background: rgb(102, 102, 102)"></div>`,
		`<div style="opacity: 0.6"></div>`,
		`<div style="background: none; opacity: 0.666666">
			<div style="opacity: 0.9"></div>
		</div>`,
	)...)
}

func TestOpacityIsClampedToUnitRange(t *testing.T) {
	testutil.AssertNoLogs(t)
	t.Run("above one", func(t *testing.T) {
		testutil.AssertSameRenderings(t, 0, opacityDocuments(
			`<div></div>`,
			`<div style="opacity: 1.2"></div>`,
			`<div style="opacity: 120%"></div>`,
		)...)
	})
	t.Run("below zero", func(t *testing.T) {
		testutil.AssertSameRenderings(t, 0, opacityDocuments(
			`<div></div>`,
			`<div></div><div style="opacity: -0.2"></div>`,
			`<div></div><div style="opacity: -20%"></div>`,
		)...)
	})
}
