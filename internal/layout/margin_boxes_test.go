package layout_test

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func marginBoxes(t testing.TB, page *boxes.PageBox, n int) []boxes.Box {
	t.Helper()
	list := children(t, page, 1+n)
	if tagOf(list[0]) != "html" {
		t.Fatalf("the first child of the page is %q, want html", tagOf(list[0]))
	}
	return list[1:]
}

func TestMarginBoxesCornersFillTheMargins(t *testing.T) {
	page := onlyPage(t, renderPages(t, `
      <style>
        @page {
          @top-left-corner {
            content: 'top_left';
            padding: 10px;
          }
          @top-right-corner {
            content: 'top_right';
            padding: 10px;
          }
          @bottom-left-corner {
            content: 'bottom_left';
            padding: 10px;
          }
          @bottom-right-corner {
            content: 'bottom_right';
            padding: 10px;
          }
          size: 1000px;
          margin-top: 10%;
          margin-bottom: 40%;
          margin-left: 20%;
          margin-right: 30%;
        }
      </style>
    `))
	corners := marginBoxes(t, page, 4)
	for i, want := range []string{"top_left", "top_right", "bottom_left", "bottom_right"} {
		line := child(t, corners[i])
		if text := child(t, line).(*boxes.TextBox).Text; text != want {
			t.Errorf("corner %d shows %q, want %q", i, text, want)
		}
	}

	tests := []struct {
		name                string
		x, y, width, height float64
	}{
		{"top left", 0, 0, 200, 100},
		{"top right", 700, 0, 300, 100},
		{"bottom left", 0, 600, 200, 400},
		{"bottom right", 700, 600, 300, 400},
	}
	for i, tt := range tests {
		box := corners[i]
		if box.Base().PositionX != tt.x || box.Base().PositionY != tt.y ||
			box.MarginWidth() != tt.width || box.MarginHeight() != tt.height {
			t.Errorf("%s corner is %g × %g at (%g, %g), want %g × %g at (%g, %g)", tt.name,
				box.MarginWidth(), box.MarginHeight(), box.Base().PositionX, box.Base().PositionY,
				tt.width, tt.height, tt.x, tt.y)
		}
	}
}

// fixedDimension renders a page with margins of 100px, or the given ones,
// and one margin box.
func fixedDimension(t *testing.T, pageMargin, rule string) boxes.Box {
	t.Helper()
	page := onlyPage(t, renderPages(t, fmt.Sprintf(`
      <style>
        @page {
          margin: %s;
          %s
        }
      </style>
    `, pageMargin, rule)))
	return marginBoxes(t, page, 1)[0]
}

func TestMarginBoxFixedDimensionRules(t *testing.T) {
	tests := []struct {
		name string
		rule string
		// The axis that is checked is the height when vertical, else the
		// width.
		vertical               bool
		outer                  float64
		marginStart, marginEnd float64
		inner                  float64
	}{
		{"a specified size is kept and a margin gives way", `@left-middle { content: ""; margin: 10px; width: 130px }`, false, 100, -40, 10, 130},
		{"an auto margin takes what is left", `@left-bottom {
            content: "";
            margin-left: 10px;
            margin-right: auto;
            width: 70px;
          }`, false, 100, 10, 20, 70},
		{"an auto margin is zero when the box is too big", `@right-top {
            content: "";
            margin-right: 10px;
            margin-left: auto;
            width: 130px;
          }`, false, 100, 0, -30, 130},
		{"an auto size takes what is left", `@top-left { content: ""; margin-top: 10px; margin-bottom: auto }`, true, 100, 10, 0, 90},
		{"auto margins and size", `@top-center { content: ""; margin: auto 0 }`, true, 100, 0, 0, 100},
		{"auto margins are shared", `@bottom-right { content: ""; margin: auto; height: 70px }`, true, 100, 15, 15, 70},
		{"a box that is too big keeps its auto margin at zero", `@bottom-center { content: ""; margin: auto 0; height: 150px }`, true, 100, 0, -50, 150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := fixedDimension(t, "100px", tt.rule)
			b := box.Base()
			var outer, start, end, inner float64
			if tt.vertical {
				outer, start, end, inner = box.MarginHeight(), b.MarginTop.V, b.MarginBottom.V, b.Height.V
			} else {
				outer, start, end, inner = box.MarginWidth(), b.MarginLeft.V, b.MarginRight.V, b.Width.V
			}
			if outer != tt.outer || start != tt.marginStart || end != tt.marginEnd || inner != tt.inner {
				t.Errorf("outer %g, margins %g and %g, inner %g; want %g, %g and %g, %g",
					outer, start, end, inner, tt.outer, tt.marginStart, tt.marginEnd, tt.inner)
			}
		})
	}
}

func TestMarginBoxFixedDimensionOverConstrainedMargins(t *testing.T) {
	box := fixedDimension(t, "100px 200px", `@bottom-left-corner { content: ""; margin: 60px }`)
	b := box.Base()
	if box.MarginWidth() != 200 || b.MarginLeft.V != 60 || b.MarginRight.V != 60 || b.Width.V != 80 {
		t.Errorf("width: outer %g, margins %g and %g, inner %g; want 200, 60 and 60, 80",
			box.MarginWidth(), b.MarginLeft.V, b.MarginRight.V, b.Width.V)
	}
	// The total was too big, so the outside margin was ignored.
	if box.MarginHeight() != 100 || b.MarginTop.V != 60 || b.MarginBottom.V != 40 || b.Height.V != 0 {
		t.Errorf("height: outer %g, margins %g and %g, inner %g; want 100, 60 and 40, 0",
			box.MarginHeight(), b.MarginTop.V, b.MarginBottom.V, b.Height.V)
	}
}

// svgImages returns a content value with an SVG image of each width and a
// height of 10 pixels.
func svgImages(widths ...int) string {
	items := make([]string, len(widths))
	for i, width := range widths {
		items[i] = fmt.Sprintf(`url('data:image/svg+xml,<svg width="%d" height="10"></svg>')`, width)
	}
	return strings.Join(items, " ")
}

// pngImages returns a content value with a PNG image of each width and a
// height of 10 pixels.
func pngImages(t testing.TB, widths ...int) string {
	t.Helper()
	items := make([]string, len(widths))
	for i, width := range widths {
		var data strings.Builder
		encoder := base64.NewEncoder(base64.StdEncoding, &data)
		if err := png.Encode(encoder, image.NewNRGBA(image.Rect(0, 0, width, 10))); err != nil {
			t.Fatal(err)
		}
		if err := encoder.Close(); err != nil {
			t.Fatal(err)
		}
		items[i] = fmt.Sprintf("url('data:image/png;base64,%s')", data.String())
	}
	return strings.Join(items, " ")
}

// topMarginsCase is a set of margin boxes along the top of a page whose
// content is made of images, and the widths that they get.
type topMarginsCase struct {
	name   string
	css    func(img func(...int) string) string
	widths []float64
	// images tells whether the content has images.
	images bool
}

var topMarginsCases = []topMarginsCase{
	{"preferred widths when they fit", func(img func(...int) string) string {
		return `@top-left { content: ` + img(50, 50) + ` }
        @top-center { content: ` + img(50, 50) + ` }
        @top-right { content: ` + img(50, 50) + ` }`
	}, []float64{100, 100, 100}, true},
	{"auto margins are zero", func(img func(...int) string) string {
		return `@top-left { content: ` + img(50, 50) + `; margin: auto }
        @top-center { content: ` + img(50, 50) + ` }
        @top-right { content: ` + img(50, 50) + ` }`
	}, []float64{100, 100, 100}, true},
	{"at least the minimum widths, even if boxes overlap", func(img func(...int) string) string {
		return `@top-left { content: ` + img(100, 50) + ` }
        @top-center { content: ` + img(300, 150) + ` }
        @top-right { content: 'foo'; width: 200px }`
	}, []float64{150, 300, 200}, true},
	{"remaining space is distributed proportionally", func(img func(...int) string) string {
		return `@top-left { content: ` + img(150, 150) + ` }
        @top-center { content: ` + img(150, 150) + ` }
        @top-right { content: ` + img(150, 150) + ` }`
	}, []float64{200, 200, 200}, true},
	{"outer boxes share what the middle one leaves", func(img func(...int) string) string {
		return `@top-left { content: ` + img(100, 100, 100) + ` }
        @top-center { content: ` + img(100, 100) + ` }
        @top-right { content: ` + img(10) + ` }`
	}, []float64{220, 160, 10}, true},
	{"a specified width", func(img func(...int) string) string {
		return `@top-left { content: ` + img(100, 100, 100) + `; width: 205px }
        @top-center { content: ` + img(100, 100) + ` }
        @top-right { content: ` + img(10) + ` }`
	}, []float64{205, 190, 10}, true},
	{"width and the others have no effect without content", func(img func(...int) string) string {
		return `@top-left { width: 1000px; margin: 1000px; padding: 1000px;
                    border: 1000px solid }
        @top-center { content: ` + img(100, 100) + ` }
        @top-right { content: ` + img(10) + ` }`
	}, []float64{200, 10}, true},
	{"shrink-to-fit with 150px left", func(img func(...int) string) string {
		return `@top-left { content: ''; width: 200px }
        @top-center { content: ''; width: 300px }
        @top-right { content: ` + img(50, 50) + ` }`
	}, []float64{200, 300, 100}, true},
	{"shrink-to-fit takes the space that is left", func(img func(...int) string) string {
		return `@top-left { content: ''; width: 200px }
        @top-center { content: ''; width: 300px }
        @top-right { content: ` + img(100, 100, 100) + ` }`
	}, []float64{200, 300, 150}, true},
	{"shrink-to-fit beyond the space that is left", func(img func(...int) string) string {
		return `@top-left { content: ''; width: 200px }
        @top-center { content: ''; width: 300px }
        @top-right { content: ` + img(170, 175) + ` }`
	}, []float64{200, 300, 175}, true},
	{"two boxes with a width", func(img func(...int) string) string {
		return `@top-left { content: ''; width: 200px }
        @top-right { content: ''; width: 500px }`
	}, []float64{200, 500}, false},
	{"one box with a width, one that grows", func(img func(...int) string) string {
		return `@top-left { content: ''; width: 200px }
        @top-right { content: ` + img(150, 50, 150) + ` }`
	}, []float64{200, 350}, true},
	{"one box with a width, one that is wider than the rest", func(img func(...int) string) string {
		return `@top-left { content: ''; width: 200px }
        @top-right { content: ` + img(150, 50, 150, 200) + ` }`
	}, []float64{200, 400}, true},
	{"the left box wider than the page", func(img func(...int) string) string {
		return `@top-left { content: ` + img(150, 50, 450) + ` }
        @top-right { content: ''; width: 200px }`
	}, []float64{450, 200}, true},
	{"two outer boxes share the side", func(img func(...int) string) string {
		return `@top-left { content: ` + img(150, 100) + ` }
        @top-right { content: ` + img(10, 120) + ` }`
	}, []float64{250, 130}, true},
	{"two outer boxes, one wider than the page", func(img func(...int) string) string {
		return `@top-left { content: ` + img(550, 100) + ` }
        @top-right { content: ` + img(10, 120) + ` }`
	}, []float64{550, 120}, true},
	{"the space left goes in proportion to the content", func(img func(...int) string) string {
		// 250 + (100 * 1 / 4), 250 + (100 * 3 / 4).
		return `@top-left { content: ` + img(250, 60) + ` }
        @top-right { content: ` + img(250, 180) + ` }`
	}, []float64{275, 325}, true},
}

// checkTopMargins renders margin boxes along the top of a page of 800px
// with margins of 100px, 42px of padding and a border of 7px, which leaves
// 600px for them to share, and checks their widths and positions.
func checkTopMargins(t *testing.T, css string, widths []float64) {
	t.Helper()
	var atKeywords []string
	for _, keyword := range []string{"@top-left", "@top-center", "@top-right"} {
		if strings.Contains(css, keyword+" { content: ") {
			atKeywords = append(atKeywords, keyword)
		}
	}
	page := onlyPage(t, renderPages(t, fmt.Sprintf(`
      <style>
        @page {
          size: 800px;
          margin: 100px;
          padding: 42px;
          border: 7px solid;
          %s
        }
      </style>
    `, css)))
	list := boxes.ChildrenOf(page)
	if len(list) == 0 || tagOf(list[0]) != "html" {
		t.Fatalf("page children = %v, want the html first", list)
	}
	var gotKeywords []string
	var gotWidths []float64
	offsets := map[string]float64{"@top-left": 0, "@top-center": 0.5, "@top-right": 1}
	for _, box := range list[1:] {
		margin := box.(*boxes.MarginBox)
		gotKeywords = append(gotKeywords, margin.AtKeyword)
		gotWidths = append(gotWidths, box.MarginWidth())
		if want := 100 + offsets[margin.AtKeyword]*(600-box.MarginWidth()); box.Base().PositionX != want {
			t.Errorf("%s at x = %g, want %g", margin.AtKeyword, box.Base().PositionX, want)
		}
	}
	if fmt.Sprint(gotKeywords) != fmt.Sprint(atKeywords) {
		t.Errorf("margin boxes = %v, want %v", gotKeywords, atKeywords)
	}
	if fmt.Sprint(gotWidths) != fmt.Sprint(widths) {
		t.Errorf("widths = %v, want %v", gotWidths, widths)
	}
}

func TestMarginBoxWidthsFromSVGContent(t *testing.T) {
	for _, tt := range topMarginsCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.images {
				t.Skip("not supported: SVG images")
			}
			checkTopMargins(t, tt.css(svgImages), tt.widths)
		})
	}
}

func TestMarginBoxWidthsFromRasterContent(t *testing.T) {
	for _, tt := range topMarginsCases {
		t.Run(tt.name, func(t *testing.T) {
			checkTopMargins(t, tt.css(func(widths ...int) string { return pngImages(t, widths...) }), tt.widths)
		})
	}
}

func TestMarginBoxesVerticalAlign(t *testing.T) {
	// The content of the margin boxes is 100px high:
	//
	//   3px ->    +-----+
	//             |  1  |
	//             +-----+
	//
	//       43px ->   +-----+
	//       53px ->   |  2  |
	//                 +-----+
	//
	//             83px ->   +-----+
	//                       |  3  |
	//             103px ->  +-----+
	page := onlyPage(t, renderPages(t, `
      <style>
        @page {
          size: 800px;
          margin: 106px;

          @top-left {
            content: "foo"; line-height: 20px; border: 3px solid;
            vertical-align: top;
          }
          @top-center {
            content: "foo"; line-height: 20px; border: 3px solid;
            vertical-align: middle;
          }
          @top-right {
            content: "foo"; line-height: 20px; border: 3px solid;
            vertical-align: bottom;
          }
        }
      </style>
    `))
	for i, want := range []float64{3, 43, 83} {
		line := child(t, marginBoxes(t, page, 3)[i])
		if got := line.Base().PositionY; got != want {
			t.Errorf("margin box %d: line at y = %g, want %g", i+1, got, want)
		}
	}
}

// textOfMargin returns the text of a margin box, the box after the content
// of the page.
func textOfMargin(page *boxes.PageBox, index int) string {
	return textOf(page.Children[index])
}

func TestMarginBoxesCountersOfPagesAndElements(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page {
          counter-increment: count;
          counter-reset: page pages;
          margin: 50px;
          size: 200px;
          @bottom-center {
            content: counter(page) ' of ' counter(pages)
                     ' (' counter(count) ')';
          }
        }
        h1 {
          height: 40px;
        }
      </style>
      <h1>test1</h1>
      <h1>test2</h1>
      <h1>test3</h1>
      <h1>test4</h1>
      <h1>test5</h1>
      <h1>test6</h1>
    `), 3)
	for i, want := range []string{"0 of 3 (1)", "0 of 3 (2)", "0 of 3 (3)"} {
		if got := textOfMargin(pages[i], 1); got != want {
			t.Errorf("page %d footer = %q, want %q", i+1, got, want)
		}
	}
}

func TestMarginBoxesShowRunningElements(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        footer {
          position: running(footer);
        }
        @page {
          margin: 50px;
          size: 200px;
          @bottom-center {
            content: element(footer);
          }
        }
        body {
          font-size: 1px
        }
        h1 {
          height: 40px;
        }
        .pages:before {
          content: counter(page);
        }
        .pages:after {
          content: counter(pages);
        }
      </style>
      <footer class="pages"> of </footer>
      <h1>test1</h1>
      <h1>test2</h1>
      <h1>test3</h1>
      <h1>test4</h1>
      <h1>test5</h1>
      <h1>test6</h1>
      <footer>Static</footer>
    `), 3)
	for i, want := range []string{"1 of 3", "2 of 3", "Static"} {
		if got := textOfMargin(pages[i], 1); got != want {
			t.Errorf("page %d footer = %q, want %q", i+1, got, want)
		}
	}
}

func TestRunningElementsByKeyword(t *testing.T) {
	// Start does not work: running elements are removed from the original
	// tree, which is how it knows whether it is at the start of the page.
	tests := []struct {
		argument string
		texts    []string
	}{
		{"first", []string{"", "2-first", "3-first", "3-last", "5"}},
		{"last", []string{"", "2-last", "3-last", "3-last", "5"}},
		{"first-except", []string{"", "", "", "3-last", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.argument, func(t *testing.T) {
			pages := pagesOf(t, renderPages(t, fmt.Sprintf(`
      <style>
        @page {
          margin: 50px;
          size: 200px;
          @bottom-center { content: element(title, %s) }
        }
        article { break-after: page }
        h1 { position: running(title) }
      </style>
      <article>
        <div>1</div>
      </article>
      <article>
        <h1>2-first</h1>
        <h1>2-last</h1>
      </article>
      <article>
        <p>3</p>
        <h1>3-first</h1>
        <h1>3-last</h1>
      </article>
      <article>
      </article>
      <article>
        <h1>5</h1>
      </article>
    `, tt.argument)), 5)
			for i, page := range pages {
				margin := marginBoxes(t, page, 1)[0]
				if len(boxes.ChildrenOf(margin)) == 0 {
					if tt.texts[i] != "" {
						t.Errorf("page %d has no running element, want %q", i+1, tt.texts[i])
					}
					continue
				}
				h1 := child(t, margin)
				text := child(t, child(t, h1)).(*boxes.TextBox)
				if text.Text != tt.texts[i] {
					t.Errorf("page %d shows %q, want %q", i+1, text.Text, tt.texts[i])
				}
			}
		})
	}
}

func TestRunningElementsKeepTheirDisplay(t *testing.T) {
	t.Skip("not supported: table layout")
	page := onlyPage(t, renderPages(t, `
      <style>
        @page {
          margin: 50px;
          size: 200px;
          @bottom-left { content: element(inline) }
          @bottom-center { content: element(block) }
          @bottom-right { content: element(table) }
        }
        table { position: running(table) }
        div { position: running(block) }
        span { position: running(inline) }
      </style>
      text
      <table><tr><td>table</td></tr></table>
      <div>block</div>
      <span>inline</span>
    `))
	list := marginBoxes(t, page, 3)
	for i, want := range []string{"inline", "block", "table"} {
		if got := textOf(list[i]); got != want {
			t.Errorf("margin box %d shows %q, want %q", i+1, got, want)
		}
	}
}

func TestRunningImage(t *testing.T) {
	renderPages(t, `
      <style>
        img {
          position: running(img);
        }
        @page {
          @bottom-center {
            content: element(img);
          }
        }
      </style>
      <img src="pattern.png" />
    `)
}

func TestRunningElementWithAbsoluteChild(t *testing.T) {
	renderPages(t, `
      <style>
        footer {
          position: running(footer);
        }
        p {
          position: absolute;
        }
        @page {
          @bottom-center {
            content: element(footer);
          }
        }
      </style>
      <footer>Hello!<p>Bonjour!</p></footer>
    `)
}

func TestRunningElementWithAbsoluteChildOverflowingTheMargin(t *testing.T) {
	renderPages(t, `
      <style>
        header {
          position: running(header);
        }
        p {
          font: 2px/1 testfont;
          position: absolute;
        }
        div {
          break-before: page;
        }
        @page {
          size: 10px;
          margin: 2px;
          @top-center {
            content: element(header);
          }
        }
      </style>
      <header>
        <p>
          abc<br>
          abc<br>
          abc<br>
          abc<br>
          abc<br>
        </p>
      </header>
      <div>a</div>
      <div>a</div>
      <div>a</div>
      <div>a</div>
    `)
}

func TestRunningFlexContainer(t *testing.T) {
	t.Skip("not supported: flex layout")
	renderPages(t, `
      <style>
        footer {
          display: flex;
          position: running(footer);
        }
        @page {
          @bottom-center {
            content: element(footer);
          }
        }
      </style>
      <footer>
        Hello!
      </footer>
    `)
}

func TestRunningFloat(t *testing.T) {
	renderPages(t, `
      <style>
        footer {
          float: left;
          position: running(footer);
        }
        @page {
          @bottom-center {
            content: element(footer);
          }
        }
      </style>
      <footer>
        Hello!
      </footer>
    `)
}
