package draw_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

const (
	skipSVG   = "not supported: SVG images"
	skipTable = "not supported: table layout"
)

const resizedImage = `
	____________
	____________
	__rrBBBBBB__
	__rrBBBBBB__
	__BBBBBBBB__
	__BBBBBBBB__
	__BBBBBBBB__
	__BBBBBBBB__
	__BBBBBBBB__
	__BBBBBBBB__
	____________
	____________
`

const noImage = `
	________
	________
	________
	________
	________
	________
	________
	________
`

const pageBreakImages = `
	________
	________
	__rBBB__
	__BBBB__
	__BBBB__
	__BBBB__
	________
	________

	________
	________
	________
	________
	________
	________
	________
	________

	________
	________
	__rBBB__
	__BBBB__
	__BBBB__
	__BBBB__
	________
	________
`

const repeatedHeaderImage = `
	________
	________
	__rBBB__
	__BBBB__
	__BBBB__
	__BBBB__
	________
	________

	__rBBB__
	__BBBB__
	__BBBB__
	__BBBB__
	________
	________
	________
	________
`

const coverImage = `
	________
	________
	__BB____
	__BB____
	__BB____
	__BB____
	________
	________
`

const borderedImage = `
	________
	_GGGGGG_
	_GrBBBG_
	_GBBBBG_
	_GBBBBG_
	_GBBBBG_
	_GGGGGG_
	________
`

func TestResizedImages(t *testing.T) {
	for _, filename := range []string{"pattern.svg", "pattern.png", "pattern.palette.png", "pattern.gif"} {
		t.Run(filename, func(t *testing.T) {
			if strings.HasSuffix(filename, ".svg") {
				t.Skip(skipSVG)
			}
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, resizedImage, `
				<style>
					@page { size: 12px }
					body { margin: 2px 0 0 2px; font-size: 0 }
					img { display: block; width: 8px; image-rendering: pixelated;
						overflow: hidden }
				</style>
				<div><img src="`+filename+`"></div>`)
		})
	}
}

func TestImageOverflow(t *testing.T) {
	t.Skip(skipSVG)
}

func TestSVGSizing(t *testing.T) {
	t.Skip(skipSVG)
}

func TestSVGSizingPercentage(t *testing.T) {
	t.Skip(skipSVG)
}

func TestSVGResizing(t *testing.T) {
	t.Skip(skipSVG)
}

func TestImagesInBlock(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, centeredImage, `
		<style>
			@page { size: 8px }
			body { margin: 0; font-size: 0 }
			img { display: block; margin: 2px auto 0 }
		</style>
		<div><img src="pattern.png"></div>`)
}

func TestImagesThatFailToLoad(t *testing.T) {
	for _, src := range []string{"inexistent1.png", "dangerous.eps"} {
		t.Run(src, func(t *testing.T) {
			logs := testutil.CaptureLogs(t)
			testutil.AssertPixels(t, noImage, `
				<style>
					@page { size: 8px }
					body { margin: 0; font-size: 0 }
					img { display: block; margin: 2px auto 0 }
				</style>
				<div><img src="`+src+`" alt=""></div>`)
			lines := logs.Lines()
			if len(lines) != 1 {
				t.Fatalf("got %d log lines, want 1: %q", len(lines), lines)
			}
			if !strings.Contains(lines[0], "ERROR: Failed to load image") || !strings.Contains(lines[0], src) {
				t.Errorf("unexpected log line %q", lines[0])
			}
		})
	}
}

func TestImageWithoutSource(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, noImage, `
		<style>
			@page { size: 8px }
			body { margin: 0; font-size: 0 }
			img { display: block; margin: 2px auto 0 }
		</style>
		<div><img alt=""></div>`)
}

func TestAlternativeTextRendersLikeText(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	var documents []string
	for _, content := range []string{
		`Hello`,
		`<img src="inexistent2.png" alt="Hello">`,
		`<img alt="Hello">`,
	} {
		documents = append(documents, `
			<style>
				@page { size: 200px 30px }
				body { margin: 0; font-size: 0 }
			</style>
			<div>`+content+`</div>`)
	}
	testutil.AssertSameRenderings(t, 0, documents...)
	lines := logs.Lines()
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "ERROR: Failed to load image") || !strings.Contains(lines[0], "inexistent2.png") {
		t.Errorf("unexpected log line %q", lines[0])
	}
}

func TestAlternativeTextForEmptySVG(t *testing.T) {
	t.Skip(skipSVG)
}

func TestTransparentBackgroundImageRepeats(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, "_\n_\n_", `
		<style>
			@page { size: 1px }
			div { height: 100px; width: 100px; background: url(logo_small.png) }
		</style>
		<div></div><div></div><div></div>`)
}

func TestImagesWithZeroSize(t *testing.T) {
	tests := []struct{ name, style string }{
		{"width", "width: 0; height: 1px"},
		{"height", "width: 1px; height: 0"},
		{"width and height", "width: 0; height: 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, noImage, `
				<style>
					@page { size: 8px }
					body { margin: 2px; font-size: 0 }
				</style>
				<div><img src="pattern.png" alt="not shown" style="`+tt.style+`"></div>`)
		})
	}
}

func TestImagesAcrossPageBreak(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, pageBreakImages, `
		<style>
			@page { size: 8px; margin: 2px }
			body { font-size: 0 }
		</style>
		<div><img src="pattern.png"></div>
		<div style="page-break-before: right"><img src="pattern.png"></div>`)
}

func TestInlineImageInRepeatedTableHeader(t *testing.T) {
	t.Skip(skipTable)
	testutil.AssertPixels(t, repeatedHeaderImage, `
		<style>
			@page { size: 8px; margin: 0 }
			table { border-collapse: collapse; margin: 2px }
			th, td { border: none; padding: 0 }
			th { height: 4px; line-height: 4px }
			td { height: 2px }
			img { vertical-align: top }
		</style>
		<table>
			<thead>
				<tr><th><img src="pattern.png"></th></tr>
			</thead>
			<tbody>
				<tr><td></td></tr>
				<tr><td></td></tr>
			</tbody>
		</table>`)
}

func TestBlockImageInRepeatedTableHeader(t *testing.T) {
	t.Skip(skipTable)
	testutil.AssertPixels(t, repeatedHeaderImage, `
		<style>
			@page { size: 8px; margin: 0 }
			table { border-collapse: collapse; margin: 2px }
			th, td { border: none; padding: 0 }
			th { height: 4px }
			td { height: 2px }
			img { display: block }
		</style>
		<table>
			<thead>
				<tr><th><img src="pattern.png"></th></tr>
			</thead>
			<tbody>
				<tr><td></td></tr>
				<tr><td></td></tr>
			</tbody>
		</table>`)
}

func TestImageWithPadding(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, centeredImage, `
		<style>
			@page { size: 8px }
			body { font-size: 0 }
		</style>
		<div style="line-height: 1px">
			<img src=pattern.png style="padding: 2px 0 0 2px">
		</div>`)
}

func TestImageInInlineBlock(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, centeredImage, `
		<style>
			@page { size: 8px }
			body { margin: 2px 0 0 2px; font-size: 0 }
		</style>
		<div style="display: inline-block">
			<p><img src=pattern.png></p>
		</div>`)
}

func TestImageAfterTransparentText(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, centeredImage, `<style>
			@page { size: 8px }
			body { margin: 2px 0 0 2px; font-size: 2px; line-height: 0 }
		</style>
		<div style="color: #0001">123</div>
		<img src=pattern.png>
	`)
}

func TestImageIsNotRepeatedAfterRepeatingBackground(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertPixels(t, `
		____________
		____________
		__aaaaaaaa__
		__aaaaaaaa__
		____________
		__aaaa______
		__aaaa______
		__aaaa______
		__aaaa______
		____________
		____________
		____________
	`, `
		<style>
			@page { size: 12px }
			body { margin: 2px; font-size: 0 }
		</style>
		<div style="background: url(blue.jpg);
			height: 2px; margin-bottom: 1px"></div>
		<img src=blue.jpg>
	`)
}

func TestImageResolution(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 0,
		`
			<style>@page { size: 20px; margin: 2px }</style>
			<div style="font-size: 0">
				<img src="pattern.png" style="width: 8px"></div>
		`,
		`
			<style>@page { size: 20px; margin: 2px }</style>
			<div style="image-resolution: .5dppx; font-size: 0">
				<img src="pattern.png"></div>
		`,
		`
			<style>@page { size: 20px; margin: 2px }
				div::before { content: url(pattern.png) }
			</style>
			<div style="image-resolution: .5x; font-size: 0"></div>
		`,
		`
			<style>@page { size: 20px; margin: 2px }
			</style>
			<div style="height: 16px; image-resolution: .5dppx;
				background: url(pattern.png) no-repeat"></div>
		`,
	)
}

func TestObjectFit(t *testing.T) {
	tests := []struct {
		name, pixels, margin, style string
	}{
		{"cover", coverImage, "2px 0 0 2px", "object-fit: cover; height: 4px; width: 2px; overflow: hidden"},
		{"contain", centeredImage, "1px 0 0 2px", "object-fit: contain; height: 6px; width: 4px; overflow: hidden"},
		{"none", centeredImage, "1px 0 0 1px", "object-fit: none; height: 6px; width: 6px"},
		{"scale-down", centeredImage, "1px 0 0 1px", "object-fit: scale-down; height: 6px; width: 6px"},
		{"none with position", centeredImage, "1px 0 0 1px",
			"object-fit: none; height: 6px; width: 6px; object-position: bottom 50% right 50%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, tt.pixels, `
				<style>
					@page { size: 8px }
					body { margin: `+tt.margin+`; font-size: 0 }
					img { `+tt.style+` }
				</style>
				<img src="pattern.png">`)
		})
	}
}

func TestImageWithBorder(t *testing.T) {
	for _, position := range []string{"", "position: absolute"} {
		name := "in flow"
		if position != "" {
			name = "absolute"
		}
		t.Run(name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, borderedImage, `
				<style>
					@page { size: 8px }
					body { margin: 0; font-size: 0 }
					img { margin: 1px; border: 1px solid lime; `+position+` }
				</style>
				<div><img src="pattern.png"></div>`)
		})
	}
}

func TestImageExifOrientationIsApplied(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 25,
		`
			<style>@page { size: 10px }</style>
			<img style="display: block" src="not-optimized.jpg">
		`,
		`
			<style>@page { size: 10px }</style>
			<img style="display: block" src="not-optimized-exif.jpg">
		`,
	)
}

func TestImageOrientationOverridesExif(t *testing.T) {
	testutil.AssertNoLogs(t)
	testutil.AssertSameRenderings(t, 25,
		`
			<style>@page { size: 10px }</style>
			<img style="display: block; image-orientation: 180deg"
				src="not-optimized-exif.jpg">
		`,
		`
			<style>@page { size: 10px }</style>
			<img style="display: block" src="not-optimized-exif.jpg">
		`,
	)
}

func TestImageOrientationKeepsJPEGFormat(t *testing.T) {
	testutil.AssertNoLogs(t)
	pdf := rendertest.PDF(t, `
		<style>@page { size: 10px }</style>
		<img style="display: block; image-orientation: 180deg"
			src="not-optimized-exif.jpg">`)
	if !bytes.Contains(pdf, []byte("DCTDecode")) {
		t.Error("rotated JPEG is not embedded as DCTDecode")
	}
}

func TestBackgroundImageAfterTranslucentBorder(t *testing.T) {
	for _, filename := range []string{"pattern.svg", "pattern.png"} {
		for _, border := range []string{"1px solid #1111", "1px dotted #1111"} {
			t.Run(filename+"/"+border, func(t *testing.T) {
				if strings.HasSuffix(filename, ".svg") {
					t.Skip(skipSVG)
				}
				testutil.AssertNoLogs(t)
				testutil.AssertPixels(t, centeredImage, `
					<style>
						@page { size: 8px; margin: 2px }
						div { width: 4px; height: 4px; box-sizing: border-box }
						div + div { margin-top: -4px }
					</style>
					<div style="border: `+border+`"></div>
					<div style="background: url(`+filename+`)"></div>`)
			})
		}
	}
}
