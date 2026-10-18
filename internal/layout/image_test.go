package layout_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// imgOfPage returns the body of the only page of a document and the image in
// its only line, nil when the body is empty.
func imgOfPage(t *testing.T, pages []*boxes.PageBox) (body, img boxes.Box) {
	t.Helper()
	_, body = pageBody(t, onlyPage(t, pages))
	if len(boxes.ChildrenOf(body)) != 0 {
		img = child(t, child(t, body))
	}
	return body, img
}

func getImg(t *testing.T, source string) (body, img boxes.Box) {
	t.Helper()
	return imgOfPage(t, renderPages(t, source))
}

func TestImagesHaveTheirIntrinsicSize(t *testing.T) {
	tests := []struct {
		name, html string
	}{
		{"png", `<img src="pattern.png">`},
		{"gif", `<img src="pattern.gif">`},
		{"jpeg", `<img src="blue.jpg">`},
		{"embed png", `<embed src=pattern.png>`},
		{"embed png named svg", `<embed src=really-a-png.svg type=image/png>`},
		{"object png", `<object data=pattern.png>`},
		{"object png named svg", `<object data=really-a-png.svg type=image/png>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, img := getImg(t, tt.html)
			if img.Base().Width.V != 4 || img.Base().Height.V != 4 {
				t.Errorf("image is %g × %g, want 4 × 4", img.Base().Width.V, img.Base().Height.V)
			}
		})
	}
}

func TestSVGImagesHaveTheirIntrinsicSize(t *testing.T) {
	t.Skip("not supported: SVG images")
	for _, source := range []string{
		`<img src="pattern.svg">`,
		`<img src="data:image/svg+xml,<svg width='4' height='4'></svg>">`,
		`<img src="DatA:image/svg+xml,<svg width='4px' height='4px'></svg>">`,
		`<embed src=pattern.svg>`,
		`<embed src=really-a-svg.png type=image/svg+xml>`,
		`<object data=pattern.svg>`,
		`<object data=really-a-svg.png type=image/svg+xml>`,
	} {
		_, img := getImg(t, source)
		if img.Base().Width.V != 4 || img.Base().Height.V != 4 {
			t.Errorf("%s is %g × %g, want 4 × 4", source, img.Base().Width.V, img.Base().Height.V)
		}
	}
}

func TestSVGImageInPhysicalUnits(t *testing.T) {
	t.Skip("not supported: SVG images")
	_, img := getImg(t, `<img src="data:image/svg+xml,<svg width='2.54cm' height='0.5in'></svg>">`)
	if img.Base().Width.V != 96 || img.Base().Height.V != 48 {
		t.Errorf("image is %g × %g, want 96 × 48", img.Base().Width.V, img.Base().Height.V)
	}
}

func TestInvalidImagesFallBackToTheirAlternativeText(t *testing.T) {
	for _, url := range []string{
		"nonexistent.png",
		"unknownprotocol://example.org/foo.png",
		"data:image/unknowntype,Not an image",
		// Invalid protocol.
		`datå:image/svg+xml,<svg width="4" height="4"></svg>`,
		// Zero-byte images.
		"data:image/png,",
		"data:image/jpeg,",
		"data:image/svg+xml,",
		// Incorrect format.
		"data:image/png,Not a PNG",
		"data:image/jpeg,Not a JPEG",
		"data:image/svg+xml,<svg>invalid xml",
	} {
		t.Run(url, func(t *testing.T) {
			logs := testutil.CaptureLogs(t)
			pages := boxtest.RenderPages(t, fmt.Sprintf("<img src='%s' alt='invalid image'>", url))
			if lines := logs.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "ERROR: Failed to load image") {
				t.Errorf("logged %q, want one error about the image", lines)
			}
			_, img := imgOfPage(t, pages)
			// Not a replaced box.
			if img.Kind() != boxes.KindInline {
				t.Fatalf("%v is not an inline box", img)
			}
			if text := textOf(child(t, img)); text != "invalid image" {
				t.Errorf("alternative text = %q, want %q", text, "invalid image")
			}
		})
	}
}

func TestImagesAreSniffedWhenTheirTypeIsWrong(t *testing.T) {
	// No logs, and the image is found.
	for name, url := range map[string]string{
		"gif as jpeg": "data:image/jpeg;base64," +
			"R0lGODlhAQABAIABAP///wAAACwAAAAAAQABAAACAkQBADs=",
		"gif as png": "data:image/png;base64," +
			"R0lGODlhAQABAIABAP///wAAACwAAAAAAQABAAACAkQBADs=",
		"png as jpeg": "data:image/jpeg;base64," +
			"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC" +
			"0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
		"png named svg": "really-a-png.svg",
	} {
		t.Run(name, func(t *testing.T) {
			getImg(t, fmt.Sprintf("<img src='%s'>", url))
		})
	}
}

func TestSVGImagesAreSniffedWhenTheirTypeIsWrong(t *testing.T) {
	t.Skip("not supported: SVG images")
	for _, url := range []string{
		`data:image/png,<svg width="1" height="1"></svg>`,
		"really-a-svg.png",
	} {
		getImg(t, fmt.Sprintf("<img src='%s'>", url))
	}
}

func TestImageFailuresAreCached(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	boxtest.RenderPages(t, "<img src=nonexistent.png><img src=nonexistent.png>")
	// Failures are cached too: only one error.
	if lines := logs.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "ERROR: Failed to load image") {
		t.Errorf("logged %q, want one error about the image", lines)
	}
}

func TestImageSizeFollowsTheRatio(t *testing.T) {
	tests := []struct {
		name, html string
	}{
		// Layout rules try to preserve the ratio, so the height is 40px too.
		{"width", `<body style="font-size: 0">
        <img src="pattern.png" style="width: 40px">`},
		{"height", `<body style="font-size: 0">
        <img src="pattern.png" style="height: 40px">`},
		{"percentage width", `<body style="font-size: 0"><p style="width: 200px">
        <img src="pattern.png" style="width: 20%">`},
		{"min-width", `<body style="font-size: 0">
        <img src="pattern.png" style="min-width: 40px">`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, img := getImg(t, tt.html)
			b := img.Base()
			if body.Base().Height.V != 40 || b.PositionY != 0 || b.Width.V != 40 || b.Height.V != 40 {
				t.Errorf("body height %g; image at y = %g, %g × %g; want 40; 0, 40 × 40",
					body.Base().Height.V, b.PositionY, b.Width.V, b.Height.V)
			}
		})
	}
}

func TestImageMaxWidthKeepsTheRatio(t *testing.T) {
	_, img := getImg(t, `<img src="pattern.png" style="max-width: 2px">`)
	if img.Base().Width.V != 2 || img.Base().Height.V != 2 {
		t.Errorf("image is %g × %g, want 2 × 2", img.Base().Width.V, img.Base().Height.V)
	}
}

func TestImageIgnoresDisplayTableCell(t *testing.T) {
	page := onlyPage(t, renderPages(t, `<body style="font-size: 0">
        <img src="pattern.png" style="width: 40px">
        <img src="pattern.png" style="width: 60px; display: table-cell">
    `))
	_, body := pageBody(t, page)
	line := child(t, body)
	images := children(t, line, 2)
	img1, img2 := images[0].Base(), images[1].Base()
	if body.Base().Height.V != 60 {
		t.Errorf("body height = %g, want 60", body.Base().Height.V)
	}
	if img1.Width.V != 40 || img1.Height.V != 40 || img2.Width.V != 60 || img2.Height.V != 60 {
		t.Errorf("images are %g × %g and %g × %g, want 40 × 40 and 60 × 60",
			img1.Width.V, img1.Height.V, img2.Width.V, img2.Height.V)
	}
	if img1.PositionY != 20 || img2.PositionY != 0 {
		t.Errorf("images at y = %g and %g, want 20 and 0", img1.PositionY, img2.PositionY)
	}
}

func TestBlockImages(t *testing.T) {
	tests := []struct {
		name, css               string
		width, height, contentX float64
	}{
		{"width", "img { width: 40px; margin: 10px auto; display: block }", 40, 40, 30},
		{"min-width in percentage", "img { min-width: 40%; margin: 10px auto; display: block }", 40, 40, 30},
		{"min-width", "img { min-width: 40px; margin: 10px auto; display: block }", 40, 40, 30},
		{"min-height and max-width", `img { min-height: 30px; max-width: 2px;
                  margin: 10px auto; display: block }`, 2, 30, 49},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf(`
        <style>
            @page { size: 100px }
            %s
        </style>
        <body>
            <img src="pattern.png">
    `, tt.css)))
			_, body := pageBody(t, page)
			img := child(t, body)
			b := img.Base()
			if b.ElementTag != "img" || b.PositionX != 0 || b.PositionY != 0 ||
				b.Width.V != tt.width || b.Height.V != tt.height {
				t.Errorf("%s at (%g, %g), %g × %g; want img at (0, 0), %g × %g",
					b.ElementTag, b.PositionX, b.PositionY, b.Width.V, b.Height.V, tt.width, tt.height)
			}
			if img.ContentBoxX() != tt.contentX || img.ContentBoxY() != 10 {
				t.Errorf("content box at (%g, %g), want (%g, 10)", img.ContentBoxX(), img.ContentBoxY(), tt.contentX)
			}
		})
	}
}

func TestSVGImageLayout(t *testing.T) {
	t.Skip("not supported: SVG images")
	page := onlyPage(t, renderPages(t, `
        <body style="float: left">
        <img style="height: 200px; margin: 10px; display: block" src="
            data:image/svg+xml,
            <svg width='150' height='100'></svg>
        ">
    `))
	_, body := pageBody(t, page)
	img := child(t, body)
	if body.Base().Width.V != 320 || body.Base().Height.V != 220 ||
		img.Base().Width.V != 300 || img.Base().Height.V != 200 {
		t.Errorf("body %g × %g, image %g × %g; want 320 × 220 and 300 × 200",
			body.Base().Width.V, body.Base().Height.V, img.Base().Width.V, img.Base().Height.V)
	}

	page = onlyPage(t, renderPages(t, `
        <div style="width: 300px; height: 300px">
        <img src="
            data:image/svg+xml,
            <svg viewBox='0 0 20 10'></svg>
        ">`))
	_, body = pageBody(t, page)
	div := child(t, body)
	img = child(t, child(t, div))
	if div.Base().Width.V != 300 || div.Base().Height.V != 300 ||
		img.Base().Width.V != 300 || img.Base().Height.V != 150 {
		t.Errorf("div %g × %g, image %g × %g; want 300 × 300 and 300 × 150",
			div.Base().Width.V, div.Base().Height.V, img.Base().Width.V, img.Base().Height.V)
	}

	renderPages(t, `
        <img style="position: absolute" src="
            data:image/svg+xml,
            <svg viewBox='0 0 20 10'></svg>
        ">`)
}

func TestEmbeddedObjectsWithoutContentFallBackToTheirChildren(t *testing.T) {
	tests := []struct {
		html     string
		children []boxes.Kind
	}{
		{`<embed>`, nil},
		{`<embed src="unknown">`, nil},
		{`<object></object>`, nil},
		{`<object data="unknown"></object>`, nil},
		{`<object>abc</object>`, []boxes.Kind{boxes.KindText}},
		{`<object data="unknown">abc</object>`, []boxes.Kind{boxes.KindText}},
	}
	for _, tt := range tests {
		t.Run(tt.html, func(t *testing.T) {
			// Failing to load an object is not an error that this test
			// checks.
			testutil.CaptureLogs(t)
			_, img := imgOfPage(t, boxtest.RenderPages(t, tt.html))
			var kinds []boxes.Kind
			for _, c := range boxes.ChildrenOf(img) {
				kinds = append(kinds, c.Kind())
			}
			if fmt.Sprint(kinds) != fmt.Sprint(tt.children) {
				t.Errorf("children = %v, want %v", kinds, tt.children)
			}
		})
	}
}

// imageMinMaxWidthCases are the limits on the size of two images in an
// inline block and the width that the inline block gets.
var imageMinMaxWidthCases = []struct {
	props    map[string]string
	divWidth float64
}{
	{nil, 4},
	{map[string]string{"min-width": "10px"}, 10},
	{map[string]string{"max-width": "1px"}, 1},
	{map[string]string{"width": "10px"}, 10},
	{map[string]string{"width": "1px"}, 1},
	{map[string]string{"min-height": "10px"}, 10},
	{map[string]string{"max-height": "1px"}, 1},
	{map[string]string{"height": "10px"}, 10},
	{map[string]string{"height": "1px"}, 1},
	{map[string]string{"min-width": "10px", "min-height": "1px"}, 10},
	{map[string]string{"min-width": "1px", "min-height": "10px"}, 10},
	{map[string]string{"max-width": "10px", "max-height": "1px"}, 1},
	{map[string]string{"max-width": "1px", "max-height": "10px"}, 1},
}

func imageMinMaxWidth(t *testing.T, images string) {
	t.Helper()
	for _, tt := range imageMinMaxWidthCases {
		css := make([]string, 0, 6)
		for _, key := range []string{"min-width", "max-width", "width", "min-height", "max-height", "height"} {
			value, ok := tt.props[key]
			if !ok {
				value = "auto"
				if strings.HasPrefix(key, "max-") {
					value = "none"
				}
			}
			css = append(css, key+": "+value)
		}
		t.Run(strings.Join(css, "; "), func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf(`
      <style> img { display: block; %s } </style>
      <div style="display: inline-block">
        %s
      </div>`, strings.Join(css, "; "), images)))
			_, body := pageBody(t, page)
			div := child(t, child(t, body))
			if got := div.Base().Width.V; got != tt.divWidth {
				t.Errorf("inline block width = %g, want %g", got, tt.divWidth)
			}
		})
	}
}

func TestImageMinMaxWidthWithRasterAndSVG(t *testing.T) {
	t.Skip("not supported: SVG images")
	imageMinMaxWidth(t, `<img src="pattern.png"><img src="pattern.svg">`)
}

func TestImageMinMaxWidthWithRasterImages(t *testing.T) {
	imageMinMaxWidth(t, `<img src="pattern.png"><img src="pattern.png">`)
}

func TestSVGWithoutSizeTakesTheSizeOfItsRatio(t *testing.T) {
	t.Skip("not supported: SVG images")
	// The size is undefined when neither the width nor the height is set
	// (CSS 2.1 §10.3.2).
	for _, tt := range []struct {
		css   string
		width float64
	}{
		{"width: 10px", 10},
		{"width: 1px", 1},
		{"height: 10px", 20},
		{"height: 1px", 2},
	} {
		t.Run(tt.css, func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf(`
      <style> svg { %s } </style>
      <div style="display: inline-block">
        <svg viewBox="0 0 8 4"></svg>
      </div>`, tt.css)))
			_, body := pageBody(t, page)
			div := child(t, child(t, body))
			if div.Base().Width.V != tt.width {
				t.Errorf("inline block width = %g, want %g", div.Base().Width.V, tt.width)
			}
		})
	}
}

func TestSVGWithoutSizeInTableCell(t *testing.T) {
	t.Skip("not supported: SVG images and table layout")
	for _, tt := range []struct {
		css   string
		width float64
	}{
		{"width: 10px", 10},
		{"width: 1px", 1},
		{"height: 10px", 20},
		{"height: 1px", 2},
	} {
		t.Run(tt.css, func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf(`
      <style> svg { %s } </style>
      <table>
        <tr>
          <td><svg viewBox="0 0 8 4"></svg></td>
          <td style="width: 1000mm"></td>
        </tr>
      </table>`, tt.css)))
			_, body := pageBody(t, page)
			wrapper := child(t, body)
			table := child(t, wrapper)
			rowGroup := child(t, table)
			row := child(t, rowGroup)
			cells := children(t, row, 2)
			if cells[0].Base().Width.V != tt.width {
				t.Errorf("cell width = %g, want %g", cells[0].Base().Width.V, tt.width)
			}
		})
	}
}
