package draw_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	_ "github.com/tomasweigenast/pdfgen/internal/testutil/rendertest"
)

// pngDataURL encodes the rows of colors as a PNG data URL. SVG sources are
// not supported yet, so the border images of the tests are drawn from
// rasters with the same pixels.
func pngDataURL(t testing.TB, rows [][]color.NRGBA) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, len(rows[0]), len(rows)))
	for y, row := range rows {
		for x, c := range row {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

var (
	red     = color.NRGBA{255, 0, 0, 255}
	yellow  = color.NRGBA{255, 255, 0, 255}
	magenta = color.NRGBA{255, 0, 255, 255}
	lime    = color.NRGBA{0, 255, 0, 255}
	cyan    = color.NRGBA{0, 255, 255, 255}
	navy    = color.NRGBA{0, 0, 128, 255}
	green   = color.NRGBA{0, 128, 0, 255}
	blue    = color.NRGBA{0, 0, 255, 255}
	black   = color.NRGBA{0, 0, 0, 255}
)

func borderPNG(t testing.TB) string {
	return pngDataURL(t, [][]color.NRGBA{
		{red, yellow, magenta, lime},
		{magenta, navy, green, cyan},
		{yellow, green, navy, yellow},
		{blue, yellow, cyan, black},
	})
}

func border2PNG(t testing.TB) string {
	return pngDataURL(t, [][]color.NRGBA{
		{red, yellow, magenta, cyan, lime},
		{magenta, navy, green, navy, cyan},
		{yellow, green, navy, green, yellow},
		{cyan, navy, green, navy, magenta},
		{blue, yellow, cyan, magenta, black},
	})
}

// maskPNG has an opaque ring with notches, a hole in the middle and one
// half-transparent pixel on the left.
func maskPNG(t testing.TB) string {
	o := black
	n := color.NRGBA{}
	h := color.NRGBA{0, 0, 0, 128}
	return pngDataURL(t, [][]color.NRGBA{
		{n, o, n, o, o},
		{o, o, o, o, o},
		{h, o, n, o, o},
		{o, o, o, o, o},
		{n, o, o, o, o},
	})
}

type imageSources struct{ border, border2, mask string }

func newImageSources(t testing.TB) imageSources {
	return imageSources{borderPNG(t), border2PNG(t), maskPNG(t)}
}

// withSource replaces the SVG file of html by the raster with the same
// pixels and keeps the pixels of the raster sharp.
func (src imageSources) withSource(html string) string {
	html = strings.ReplaceAll(html, "border2.svg", src.border2)
	html = strings.ReplaceAll(html, "border.svg", src.border)
	html = strings.ReplaceAll(html, "mask.svg", src.mask)
	return strings.Replace(html, "<style>", "<style> div { image-rendering: pixelated }", 1)
}

func TestBorderImages(t *testing.T) {
	tests := []struct {
		name, pixels, html string
	}{
		{
			"stretch",
			`
			__________
			_RYYYMMMG_
			_M______C_
			_M______C_
			_Y______Y_
			_Y______Y_
			_BYYYCCCK_
			__________
			`,
			`
			<style>
				@page { size: 10px 8px }
				div {
					border: 1px solid black;
					border-image-source: url(border.svg);
					border-image-slice: 25%;
					height: 4px;
					margin: 1px;
					width: 6px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"fill",
			`
			__________
			_RYYYMMMG_
			_MbbbgggC_
			_MbbbgggC_
			_YgggbbbY_
			_YgggbbbY_
			_BYYYCCCK_
			__________
			`,
			`
			<style>
				@page { size: 10px 8px }
				div {
					border: 1px solid black;
					border-image-source: url(border.svg);
					border-image-slice: 25% fill;
					height: 4px;
					margin: 1px;
					width: 6px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"default slice",
			`
			_____________
			_RYMG___RYMG_
			_MbgC___MbgC_
			_YgbY___YgbY_
			_BYCK___BYCK_
			_____________
			_____________
			_RYMG___RYMG_
			_MbgC___MbgC_
			_YgbY___YgbY_
			_BYCK___BYCK_
			_____________
			`,
			`
			<style>
				@page { size: 13px 12px }
				div {
					border: 4px solid black;
					border-image-source: url(border.svg);
					height: 2px;
					margin: 1px;
					width: 3px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"uneven width",
			`
			____________
			_RRRYYYMMMG_
			_MMM______C_
			_MMM______C_
			_YYY______Y_
			_YYY______Y_
			_BBBYYYCCCK_
			____________
			`,
			`
			<style>
				@page { size: 12px 8px }
				div {
					border: 1px solid black;
					border-left-width: 3px;
					border-image-source: url(border.svg);
					border-image-slice: 25%;
					height: 4px;
					margin: 1px;
					width: 6px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"slice in pixels",
			`
			__________
			_RYYYMMMG_
			_M______C_
			_M______C_
			_Y______Y_
			_Y______Y_
			_BYYYCCCK_
			__________
			`,
			`
			<style>
				@page { size: 10px 8px }
				div {
					border: 1px solid black;
					border-image-source: url(border.svg);
					border-image-slice: 1;
					height: 4px;
					margin: 1px;
					width: 6px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"repeat",
			`
			___________
			_RYMYMYMYG_
			_M_______C_
			_Y_______Y_
			_M_______C_
			_Y_______Y_
			_BYCYCYCYK_
			___________
			`,
			`
			<style>
				@page { size: 11px 8px }
				div {
					border: 1px solid black;
					border-image-source: url(border.svg);
					border-image-slice: 25%;
					border-image-repeat: repeat;
					height: 4px;
					margin: 1px;
					width: 7px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"space",
			`
			_________
			_R_YMC_G_
			_________
			_M_____C_
			_Y_____Y_
			_C_____M_
			_________
			_B_YCM_K_
			_________
			`,
			`
			<style>
				@page { size: 9px 9px }
				div {
					border: 1px solid black;
					border-image-source: url(border2.svg);
					border-image-slice: 20%;
					border-image-repeat: space;
					height: 5px;
					margin: 1px;
					width: 5px;
				}
			</style>
			<div></div>
			`,
		},
		{
			"outset",
			`
			____________
			_RYYYYMMMMG_
			_M________C_
			_M_bbbbbb_C_
			_M_bbbbbb_C_
			_Y_bbbbbb_Y_
			_Y_bbbbbb_Y_
			_Y________Y_
			_BYYYYCCCCK_
			____________
			`,
			`
			<style>
				@page { size: 12px 10px }
				div {
					border: 1px solid black;
					border-image-source: url(border.svg);
					border-image-slice: 25%;
					border-image-outset: 2px;
					height: 2px;
					margin: 3px;
					width: 4px;
					background: #000080
				}
			</style>
			<div></div>
			`,
		},
		{
			"width",
			`
			__________
			_RRYYMMGG_
			_RRYYMMGG_
			_MM____CC_
			_YY____YY_
			_BBYYCCKK_
			_BBYYCCKK_
			__________
			`,
			`
			<style>
				@page { size: 10px 8px }
				div {
					border: 1px solid black;
					border-image-source: url(border.svg);
					border-image-slice: 25%;
					border-image-width: 2;
					height: 4px;
					margin: 1PX;
					width: 6px;
				}
			</style>
			<div></div>
			`,
		},
	}
	src := newImageSources(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			t.Run("svg", func(t *testing.T) {
				t.Skip("not supported: SVG images")
			})
			t.Run("raster", func(t *testing.T) {
				testutil.AssertPixels(t, tt.pixels, src.withSource(tt.html))
			})
		})
	}
}

func TestBorderImageGradients(t *testing.T) {
	expected := `
		__________
		_RRRRRRRR_
		_RRRRRRRR_
		_RR____RR_
		_BB____BB_
		_BBBBBBBB_
		_BBBBBBBB_
		__________
	`
	tests := []struct {
		name, source, extra string
	}{
		{"keyword positions", "linear-gradient(to bottom, red, red 50%, blue 50%, blue)", ""},
		{"font relative length", "linear-gradient(to bottom, red 0, red 50%, blue 0.5em, blue)", "font-size: 6px;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			testutil.AssertPixels(t, expected, `
				<style>
					@page { size: 10px 8px }
					div {
						border: 1px solid black;
						border-image-source: `+tt.source+`;
						border-image-slice: 25%;
						border-image-width: 2;
						`+tt.extra+`
						height: 4px;
						margin: 1px;
						width: 6px;
					}
				</style>
				<div></div>
			`)
		})
	}
}

func TestMaskBorder(t *testing.T) {
	tests := []struct {
		name, pixels, slice string
	}{
		{
			"ring",
			`
			__________
			__RR__RRR_
			_R______R_
			_R______R_
			_s______R_
			_s______R_
			_R______R_
			_R______R_
			__RRRRRRR_
			__________
			`,
			"20%",
		},
		{
			"fill",
			`
			__________
			__RR__RRR_
			_RRRRRRRR_
			_RRRRRRRR_
			_sRR__RRR_
			_sRR__RRR_
			_RRRRRRRR_
			_RRRRRRRR_
			__RRRRRRR_
			__________
			`,
			"20% fill",
		},
	}
	src := newImageSources(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			html := `
				<style>
					@page { size: 10px 10px }
					div {
						background: red;
						mask-border-source: url(mask.svg);
						mask-border-slice: ` + tt.slice + `;
						height: 8px;
						width: 8px;
						margin: 1px;
					}
				</style>
				<div></div>
			`
			t.Run("svg", func(t *testing.T) {
				t.Skip("not supported: SVG images")
			})
			t.Run("raster", func(t *testing.T) {
				testutil.AssertPixels(t, tt.pixels, src.withSource(html))
			})
		})
	}
}
