package layout_test

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestPageSizeBasic(t *testing.T) {
	tests := []struct {
		size          string
		width, height int
	}{
		{"auto", 793, 1122},
		{"2in 10in", 192, 960},
		{"242px", 242, 242},
		{"letter", 816, 1056},
		{"letter portrait", 816, 1056},
		{"letter landscape", 1056, 816},
		{"portrait", 793, 1122},
		{"landscape", 1122, 793},
	}
	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf("<style>@page { size: %s; }</style>", tt.size)))
			if got := int(page.MarginWidth()); got != tt.width {
				t.Errorf("margin width = %d, want %d", got, tt.width)
			}
			if got := int(page.MarginHeight()); got != tt.height {
				t.Errorf("margin height = %d, want %d", got, tt.height)
			}
		})
	}
}

func TestPageSizeWithMargin(t *testing.T) {
	page := onlyPage(t, renderPages(t, `<style>
      @page { size: 200px 300px; margin: 10px 10% 20% 1in }
      body { margin: 8px }
    </style>
    <p style="margin: 0">`))
	if page.MarginWidth() != 200 || page.MarginHeight() != 300 {
		t.Errorf("margin box = %g × %g, want 200 × 300", page.MarginWidth(), page.MarginHeight())
	}
	if page.PositionX != 0 || page.PositionY != 0 {
		t.Errorf("page at (%g, %g), want the origin", page.PositionX, page.PositionY)
	}
	// 200px - 10% - 1 inch, and 300px - 10px - 20%.
	if size := pageSize(page); size != [2]float64{84, 230} {
		t.Errorf("page content = %v, want [84 230]", size)
	}

	html, body := pageBody(t, page)
	if tagOf(html) != "html" {
		t.Errorf("tag = %q, want html", tagOf(html))
	}
	// The margins of the root element do not collapse.
	if html.Base().PositionX != 96 || html.Base().PositionY != 10 || html.Base().Width.V != 84 {
		t.Errorf("html at (%g, %g) width %g, want (96, 10) width 84",
			html.Base().PositionX, html.Base().PositionY, html.Base().Width.V)
	}

	b := body.Base()
	if b.ElementTag != "body" || b.PositionX != 96 || b.PositionY != 10 {
		t.Errorf("body %q at (%g, %g), want body at (96, 10)", b.ElementTag, b.PositionX, b.PositionY)
	}
	// The user agent stylesheet gives body its margins.
	if b.MarginLeft.V != 8 || b.MarginRight.V != 8 || b.MarginTop.V != 8 || b.MarginBottom.V != 8 {
		t.Errorf("body margins = %v %v %v %v, want 8", b.MarginTop, b.MarginRight, b.MarginBottom, b.MarginLeft)
	}
	if b.Width.V != 68 {
		t.Errorf("body width = %g, want 68", b.Width.V)
	}

	p := child(t, body).Base()
	if p.ElementTag != "p" || p.PositionX != 104 || p.PositionY != 18 || p.Width.V != 68 {
		t.Errorf("paragraph %q at (%g, %g) width %g, want p at (104, 18) width 68",
			p.ElementTag, p.PositionX, p.PositionY, p.Width.V)
	}
}

func TestPageSizeWithMarginBorderPadding(t *testing.T) {
	page := onlyPage(t, renderPages(t, `<style> @page {
      size: 100px; margin: 1px 2px; padding: 4px 8px;
      border-width: 16px 32px; border-style: solid;
    }</style>`))
	if size := pageSize(page); size != [2]float64{16, 58} {
		t.Errorf("page content = %v, want [16 58]", size)
	}
	html := child(t, page).Base()
	if html.ElementTag != "html" || html.PositionX != 42 || html.PositionY != 21 {
		t.Errorf("html %q at (%g, %g), want html at (42, 21)", html.ElementTag, html.PositionX, html.PositionY)
	}
}

func TestPageSizeMargins(t *testing.T) {
	tests := []struct {
		margin                   string
		top, right, bottom, left float64
	}{
		{"auto", 15, 10, 15, 10},
		{"5px 5px auto auto", 5, 5, 25, 15},
	}
	for _, tt := range tests {
		t.Run(tt.margin, func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf(`<style>@page {
      size: 106px 206px; width: 80px; height: 170px;
      padding: 1px; border: 2px solid; margin: %s }</style>`, tt.margin)))
			b := page.Base()
			got := [4]float64{b.MarginTop.V, b.MarginRight.V, b.MarginBottom.V, b.MarginLeft.V}
			if want := [4]float64{tt.top, tt.right, tt.bottom, tt.left}; got != want {
				t.Errorf("margins = %v, want %v", got, want)
			}
		})
	}
}

func TestPageSizeOverConstrained(t *testing.T) {
	tests := []struct {
		style         string
		width, height float64
	}{
		{"size: 4px 10000px; width: 100px; height: 100px;" +
			"padding: 1px; border: 2px solid; margin: 3px", 112, 112},
		{"size: 1000px; margin: 100px; max-width: 500px; min-height: 1500px", 700, 1700},
		{"size: 1000px; margin: 100px; min-width: 1500px; max-height: 500px", 1700, 700},
	}
	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			page := onlyPage(t, renderPages(t, fmt.Sprintf("<style>@page { %s }</style>", tt.style)))
			if page.MarginWidth() != tt.width || page.MarginHeight() != tt.height {
				t.Errorf("margin box = %g × %g, want %g × %g",
					page.MarginWidth(), page.MarginHeight(), tt.width, tt.height)
			}
		})
	}
}

func TestPageBreaks(t *testing.T) {
	for _, content := range []string{"<div>1</div>", "<div></div>", "<img src=pattern.png>"} {
		t.Run(content, func(t *testing.T) {
			source := `
      <style>
        @page { size: 100px; margin: 10px }
        body { margin: 0 }
        div { height: 30px; font-size: 20px }
        img { height: 30px; display: block }
      </style>`
			for range 5 {
				source += content
			}
			var got [][]float64
			for _, page := range renderPages(t, source) {
				_, body := pageBody(t, page)
				var ys []float64
				for _, c := range boxes.ChildrenOf(body) {
					if tag := tagOf(c); tag != "div" && tag != "img" {
						t.Errorf("unexpected child %v", c)
					}
					if c.Base().PositionX != 10 {
						t.Errorf("%v at x = %g, want 10", c, c.Base().PositionX)
					}
					ys = append(ys, c.Base().PositionY)
				}
				got = append(got, ys)
			}
			if want := fmt.Sprint([][]float64{{10, 40}, {10, 40}, {10}}); fmt.Sprint(got) != want {
				t.Errorf("positions = %v, want %v", got, want)
			}
		})
	}
}

func TestPageBreaksBoxSplit(t *testing.T) {
	// If floats round the wrong way, a block that gets filled to the end of
	// a page by breaking over it can be forced onto the next page because it
	// is slightly taller than what fits, even though it would not have been
	// without being filled. These numbers trigger the issue.
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 982.4146981627297px; margin: 0 }
        div { font-size: 5px; height: 200.0123456789px; margin: 0; padding: 0 }
        figure { margin: 0; padding: 0 }
      </style>
      <div>text</div>
      <div>text</div><!-- no page break here -->
      <section>
        <div>line1</div>
        <div>line2</div><!-- page break here -->
        <div>line3</div>
        <div>line4</div>
      </section>
    `), 2)
	_, body := pageBody(t, pages[0])
	list := children(t, body, 3)
	children(t, list[2], 2)

	_, body = pageBody(t, pages[1])
	children(t, child(t, body), 2)
}

func TestForcedBreaksToASideLeaveBlankPages(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { margin: 10px }
        @page :left { margin-left: 50px }
        @page :right { margin-right: 50px }
        html { page-break-before: left }
        div { page-break-after: left }
        ul { page-break-before: always }
      </style>
      <div>1</div>
      <p>2</p>
      <p>3</p>
      <article>
        <section>
          <ul><li>4</li></ul>
        </section>
      </article>
    `), 4)

	// The first page would be a right page, but the root element asks for a
	// left one.
	checkMargins := func(page *boxes.PageBox, left, right float64) {
		t.Helper()
		if page.MarginLeft.V != left || page.MarginRight.V != right {
			t.Errorf("margins = %g and %g, want %g and %g", page.MarginLeft.V, page.MarginRight.V, left, right)
		}
	}
	checkMargins(pages[0], 50, 10)
	_, body := pageBody(t, pages[0])
	div := child(t, body)
	text := child(t, child(t, div))
	if tagOf(div) != "div" || text.(*boxes.TextBox).Text != "1" {
		t.Errorf("first page has %v with %q", div, textOf(div))
	}

	// An empty page makes the next one a left page.
	checkMargins(pages[1], 10, 50)
	children(t, child(t, pages[1]), 0)

	checkMargins(pages[2], 50, 10)
	_, body = pageBody(t, pages[2])
	list := children(t, body, 2)
	if tagOf(list[0]) != "p" || tagOf(list[1]) != "p" {
		t.Errorf("third page has %v", list)
	}

	checkMargins(pages[3], 10, 50)
	_, body = pageBody(t, pages[3])
	article := child(t, body)
	section := child(t, article)
	if ulist := child(t, section); tagOf(ulist) != "ul" {
		t.Errorf("last page has %v, want ul", ulist)
	}
}

func TestBreakBetweenImagesWithoutAvoid(t *testing.T) {
	// Reference for the tests that avoid breaks: this one breaks after the
	// div.
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 140px; margin: 0 }
        img { height: 25px; vertical-align: top }
      </style>
      <img src=pattern.png>
      <div>
        <p><img src=pattern.png><br/><img src=pattern.png><p>
        <p><img src=pattern.png><br/><img src=pattern.png><p>
      </div><!-- page break here -->
      <img src=pattern.png>
    `), 2)
	_, body := pageBody(t, pages[0])
	list := children(t, body, 2)
	img1, div := list[0], list[1]
	if img1.Base().PositionY != 0 || img1.Base().Height.V != 25 {
		t.Errorf("first image at y = %g, height %g", img1.Base().PositionY, img1.Base().Height.V)
	}
	if div.Base().PositionY != 25 || div.Base().Height.V != 100 {
		t.Errorf("div at y = %g, height %g", div.Base().PositionY, div.Base().Height.V)
	}

	_, body = pageBody(t, pages[1])
	img2 := child(t, body)
	if img2.Base().PositionY != 0 || img2.Base().Height.V != 25 {
		t.Errorf("second image at y = %g, height %g", img2.Base().PositionY, img2.Base().Height.V)
	}
}

func TestAvoidingBreaksLeavesOneLegalBreakBeforeTheDiv(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 140px; margin: 0 }
        img { height: 25px; vertical-align: top }
      </style>
      <img src=pattern.png><!-- page break here -->
      <div>
        <p style="page-break-inside: avoid">
          <img src=pattern.png><br/><img src=pattern.png></p>
        <p style="page-break-before: avoid; page-break-after: avoid; widows: 2"
          ><img src=pattern.png><br/><img src=pattern.png></p>
      </div>
      <img src=pattern.png>
    `), 2)
	_, body := pageBody(t, pages[0])
	img1 := child(t, body)
	if img1.Base().PositionY != 0 || img1.Base().Height.V != 25 {
		t.Errorf("first image at y = %g, height %g", img1.Base().PositionY, img1.Base().Height.V)
	}

	_, body = pageBody(t, pages[1])
	list := children(t, body, 2)
	div, img2 := list[0], list[1]
	if div.Base().PositionY != 0 || div.Base().Height.V != 100 {
		t.Errorf("div at y = %g, height %g", div.Base().PositionY, div.Base().Height.V)
	}
	if img2.Base().PositionY != 100 || img2.Base().Height.V != 25 {
		t.Errorf("second image at y = %g, height %g", img2.Base().PositionY, img2.Base().Height.V)
	}
}

func TestAvoidingBreaksInsideNestedBlocks(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 140px; margin: 0 }
        img { height: 25px; vertical-align: top }
      </style>
      <img src=pattern.png><!-- page break here -->
      <div>
        <div>
          <p style="page-break-inside: avoid">
            <img src=pattern.png><br/><img src=pattern.png></p>
          <p style="page-break-before:avoid; page-break-after:avoid; widows:2"
            ><img src=pattern.png><br/><img src=pattern.png></p>
        </div>
        <img src=pattern.png>
      </div>
    `), 2)
	_, body := pageBody(t, pages[0])
	img1 := child(t, body)
	if img1.Base().PositionY != 0 || img1.Base().Height.V != 25 {
		t.Errorf("first image at y = %g, height %g", img1.Base().PositionY, img1.Base().Height.V)
	}

	_, body = pageBody(t, pages[1])
	outer := child(t, body)
	list := children(t, outer, 2)
	inner, img2 := list[0], list[1]
	if inner.Base().PositionY != 0 || inner.Base().Height.V != 100 {
		t.Errorf("inner div at y = %g, height %g", inner.Base().PositionY, inner.Base().Height.V)
	}
	if img2.Base().PositionY != 100 || img2.Base().Height.V != 25 {
		t.Errorf("image at y = %g, height %g", img2.Base().PositionY, img2.Base().Height.V)
	}
}

func TestForcedBreakInsideNestedBlocksFillsPages(t *testing.T) {
	// Reference for the next test.
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 100px; margin: 0 }
        img { height: 30px; display: block; }
      </style>
      <div>
        <img src=pattern.png style="page-break-after: always">
        <section>
          <img src=pattern.png>
          <img src=pattern.png>
        </section>
      </div>
      <img src=pattern.png><!-- page break here -->
      <img src=pattern.png>
    `), 3)
	_, body := pageBody(t, pages[0])
	if div := child(t, body); div.Base().Height.V != 100 {
		t.Errorf("first div height = %g, want 100", div.Base().Height.V)
	}
	_, body = pageBody(t, pages[1])
	list := children(t, body, 2)
	if list[0].Base().Height.V != 60 || list[1].Base().Height.V != 30 {
		t.Errorf("heights = %g and %g, want 60 and 30", list[0].Base().Height.V, list[1].Base().Height.V)
	}
	_, body = pageBody(t, pages[2])
	if img := child(t, body); img.Base().Height.V != 30 {
		t.Errorf("last image height = %g, want 30", img.Base().Height.V)
	}
}

func TestAvoidedBreakInsideNestedBlocksMovesTheBreakUp(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 100px; margin: 0 }
        img { height: 30px; display: block; }
      </style>
      <div>
        <img src=pattern.png style="page-break-after: always">
        <section>
          <img src=pattern.png><!-- page break here -->
          <img src=pattern.png style="page-break-after: avoid">
        </section>
      </div>
      <img src=pattern.png style="page-break-after: avoid">
      <img src=pattern.png>
    `), 3)
	_, body := pageBody(t, pages[0])
	if div := child(t, body); div.Base().Height.V != 100 {
		t.Errorf("first div height = %g, want 100", div.Base().Height.V)
	}
	_, body = pageBody(t, pages[1])
	img2 := child(t, child(t, child(t, body)))
	if img2.Base().Height.V != 30 {
		t.Errorf("image height = %g, want 30", img2.Base().Height.V)
	}
	// The used height of the div is not increased to fill the page when
	// children are removed from it for a break that is avoided.

	_, body = pageBody(t, pages[2])
	list := children(t, body, 3)
	for i, box := range list {
		if box.Base().Height.V != 30 {
			t.Errorf("child %d height = %g, want 30", i, box.Base().Height.V)
		}
	}
}

func TestBlankPagesHaveNoMarginBoxesOfTheirOwn(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { @bottom-center { content: counter(page) } }
        @page:blank { @bottom-center { content: none } }
      </style>
      <p style="page-break-after: right">foo</p>
      <p>bar</p>
    `), 3)
	// The content and @bottom-center, or the content only.
	for i, want := range []int{2, 1, 2} {
		if got := len(pages[i].Children); got != want {
			t.Errorf("page %d has %d children, want %d", i+1, got, want)
		}
	}
}

func TestOutOfFlowBoxesStayWithTheirBlockAtBreaks(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 75px; margin: 0 }
        div { height: 20px }
      </style>
      <div></div>
      <section>
        <div></div>
        <div style="page-break-after: avoid">
          <div style="position: absolute"></div>
          <div style="position: fixed"></div>
        </div>
      </section>
      <div></div>
    `), 2)
	// The fixed box of the second page is repeated after the content.
	html := child(t, pages[0])
	body := children(t, html, 2)[0]
	items := children(t, body, 2)
	div1, section := items[0], items[1]
	div2 := child(t, section)
	if div1.Base().PositionY != 0 || div2.Base().PositionY != 20 {
		t.Errorf("positions = %g and %g, want 0 and 20", div1.Base().PositionY, div2.Base().PositionY)
	}
	if div1.Base().Height.V != 20 || div2.Base().Height.V != 20 {
		t.Errorf("heights = %g and %g, want 20", div1.Base().Height.V, div2.Base().Height.V)
	}

	_, body = pageBody(t, pages[1])
	items = children(t, body, 2)
	section, div4 := items[0], items[1]
	div3 := child(t, section)
	if tags := children(t, div3, 2); tagOf(tags[0]) != "div" || tagOf(tags[1]) != "div" {
		t.Errorf("div3 has %v, want the absolute and the fixed boxes", tags)
	}
	if div3.Base().PositionY != 0 || div4.Base().PositionY != 20 {
		t.Errorf("positions = %g and %g, want 0 and 20", div3.Base().PositionY, div4.Base().PositionY)
	}
	if div3.Base().Height.V != 20 || div4.Base().Height.V != 20 {
		t.Errorf("heights = %g and %g, want 20", div3.Base().Height.V, div4.Base().Height.V)
	}
}

func TestUnforcedBreakTruncatesMarginsButForcedBreakKeepsThem(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 75px; margin: 0 }
        div { height: 20px; margin: 10px }
      </style>
      <div style="height: 40px"></div>
      <div></div>
      <div style="break-before: left"></div>
      <div style="break-before: right"></div>
    `), 5)
	contentBox := func(page *boxes.PageBox) (x, y float64) {
		t.Helper()
		_, body := pageBody(t, page)
		div := child(t, body)
		return div.ContentBoxX(), div.ContentBoxY()
	}
	for i, want := range [][2]float64{{10, 10}, {10, 0}, {}, {10, 10}, {10, 10}} {
		if i == 2 {
			// An empty page, to get to a left page.
			children(t, child(t, pages[i]), 0)
			continue
		}
		// The break before the second div is not forced.
		if x, y := contentBox(pages[i]); [2]float64{x, y} != want {
			t.Errorf("page %d: content box at (%g, %g), want %v", i+1, x, y, want)
		}
	}
}

func TestMarginBreak(t *testing.T) {
	tests := []struct {
		breakAfter, marginBreak string
		marginTop               float64
	}{
		{"page", "auto", 5},
		{"auto", "auto", 0},
		{"page", "keep", 5},
		{"auto", "keep", 5},
		{"page", "discard", 0},
		{"auto", "discard", 0},
	}
	for _, tt := range tests {
		t.Run(tt.breakAfter+"/"+tt.marginBreak, func(t *testing.T) {
			pages := pagesOf(t, renderPages(t, fmt.Sprintf(`
      <style>
        @page { size: 70px; margin: 0 }
        div { height: 63px; margin: 5px 0 8px;
              break-after: %s; margin-break: %s }
      </style>
      <section>
        <div></div>
      </section>
      <section>
        <div></div>
      </section>
    `, tt.breakAfter, tt.marginBreak)), 2)
			for i, want := range []float64{5, tt.marginTop} {
				_, body := pageBody(t, pages[i])
				div := child(t, child(t, body))
				if got := div.Base().MarginTop.V; got != want {
					t.Errorf("page %d: div margin-top = %g, want %g", i+1, got, want)
				}
			}
		})
	}
}

func TestMarginBreakClearance(t *testing.T) {
	t.Skip("not supported: the margin of a box after a break is not truncated when a border separates it from its first child")
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 70px; margin: 0 }
        div { height: 63px; margin: 5px 0 8px; break-after: page }
      </style>
      <section>
        <div></div>
      </section>
      <section>
        <div style="border-top: 1px solid black">
          <div></div>
        </div>
      </section>
    `), 2)
	_, body := pageBody(t, pages[0])
	if div := child(t, child(t, body)); div.Base().MarginTop.V != 5 {
		t.Errorf("div margin-top = %g, want 5", div.Base().MarginTop.V)
	}
	_, body = pageBody(t, pages[1])
	div1 := child(t, child(t, body))
	if div1.Base().MarginTop.V != 0 {
		t.Errorf("outer div margin-top = %g, want 0", div1.Base().MarginTop.V)
	}
	div2 := child(t, div1)
	if div2.Base().MarginTop.V != 5 || div2.ContentBoxY() != 5 {
		t.Errorf("inner div margin-top = %g, content box y = %g, want 5 and 5",
			div2.Base().MarginTop.V, div2.ContentBoxY())
	}
}

func TestRectoVersoBreak(t *testing.T) {
	tests := []struct {
		direction, pageBreak string
		pages                int
	}{
		{"ltr", "recto", 3},
		{"ltr", "verso", 2},
		{"rtl", "recto", 3},
		{"rtl", "verso", 2},
		{"ltr", "right", 3},
		{"ltr", "left", 2},
		{"rtl", "right", 2},
		{"rtl", "left", 3},
	}
	for _, tt := range tests {
		t.Run(tt.direction+"/"+tt.pageBreak, func(t *testing.T) {
			pagesOf(t, renderPages(t, fmt.Sprintf(`
      <style>
        html { direction: %s }
        p { break-before: %s }
      </style>
      abc
      <p>def</p>
    `, tt.direction, tt.pageBreak)), tt.pages)
		})
	}
}

func TestRectoVersoBreakOnTheRoot(t *testing.T) {
	tests := []struct {
		direction, pageBreak, firstPage string
	}{
		{"ltr", "recto", "right"},
		{"ltr", "verso", "left"},
		{"rtl", "recto", "left"},
		{"rtl", "verso", "right"},
		{"ltr", "right", "right"},
		{"ltr", "left", "left"},
		{"rtl", "right", "right"},
		{"rtl", "left", "left"},
	}
	for _, tt := range tests {
		t.Run(tt.direction+"/"+tt.pageBreak, func(t *testing.T) {
			// The left page is 4px wide and the right one 5px, like the
			// length of their names.
			page := onlyPage(t, renderPages(t, fmt.Sprintf(`
      <style>
        @page:left { size: 4px }
        @page:right { size: 5px }
        html { direction: %s; break-before: %s }
      </style>
      abc
    `, tt.direction, tt.pageBreak)))
			if got := page.Base().Width.V; got != float64(len(tt.firstPage)) {
				t.Errorf("page width = %g, want %d: the first page is a %s page", got, len(tt.firstPage), tt.firstPage)
			}
		})
	}
}

func TestNamedPages(t *testing.T) {
	type check func(t *testing.T, pages []*boxes.PageBox)
	size := func(page *boxes.PageBox, width, height float64) {
		t.Helper()
		if got := pageSize(page); got != [2]float64{width, height} {
			t.Errorf("page size = %v, want %g × %g", got, width, height)
		}
	}
	tests := []struct {
		name   string
		source string
		pages  int
		check  check
	}{
		{"unused name leaves the default size", `
      <style>
        @page { size: 100px 100px }
        section { page: small }
      </style>
      <div>
        <section>large</section>
      </div>`, 1, func(t *testing.T, pages []*boxes.PageBox) { size(pages[0], 100, 100) }},
		{"unused page rule", `
      <style>
        @page { size: 100px 100px }
        @page narrow { margin: 1px }
        section { page: small }
      </style>
      <div>
        <section>large</section>
      </div>`, 1, func(t *testing.T, pages []*boxes.PageBox) { size(pages[0], 100, 100) }},
		{"a page for each name", `
      <style>
        @page { margin: 0 }
        @page narrow { size: 100px 200px }
        @page large { size: 200px 100px }
        div { page: narrow }
        section { page: large }
      </style>
      <div>
        <section>large</section>
        <section>large</section>
        <p>narrow</p>
      </div>`, 2, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 200, 100)
			_, body := pageBody(t, pages[0])
			sections := children(t, child(t, body), 2)
			if tagOf(sections[0]) != "section" || tagOf(sections[1]) != "section" {
				t.Errorf("first page has %v", sections)
			}
			size(pages[1], 100, 200)
			_, body = pageBody(t, pages[1])
			if p := child(t, child(t, body)); tagOf(p) != "p" {
				t.Errorf("second page has %v", p)
			}
		}},
		{"a named element breaks the page", `
      <style>
        @page { size: 200px 200px; margin: 0 }
        @page small { size: 100px 100px }
        p { page: small }
      </style>
      <section>normal</section>
      <section>normal</section>
      <p>small</p>
      <section>small</section>`, 2, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 200, 200)
			_, body := pageBody(t, pages[0])
			sections := children(t, body, 2)
			if tagOf(sections[0]) != "section" || tagOf(sections[1]) != "section" {
				t.Errorf("first page has %v", sections)
			}
			size(pages[1], 100, 100)
			_, body = pageBody(t, pages[1])
			list := children(t, body, 2)
			if tagOf(list[0]) != "p" || tagOf(list[1]) != "section" {
				t.Errorf("second page has %v", list)
			}
		}},
		{"a named block inside a block", `
      <style>
        @page { size: 200px 200px; margin: 0 }
        @page small { size: 100px 100px }
        div { page: small }
      </style>
      <section><p>a</p>b</section>
      <section>c<div>d</div></section>`, 2, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 200, 200)
			_, body := pageBody(t, pages[0])
			sections := children(t, body, 2)
			children(t, sections[0], 2)
			children(t, sections[1], 1)
			size(pages[1], 100, 100)
			_, body = pageBody(t, pages[1])
			child(t, child(t, body))
		}},
		{"names nested in names", `
      <style>
        @page { margin: 0 }
        @page large { size: 200px 200px }
        @page small { size: 100px 100px }
        section { page: large }
        div { page: small }
      </style>
      <section>a<p>b</p>c</section>
      <section>d<div>e</div>f</section>`, 3, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 200, 200)
			_, body := pageBody(t, pages[0])
			sections := children(t, body, 2)
			children(t, sections[0], 3)
			children(t, sections[1], 1)
			size(pages[1], 100, 100)
			_, body = pageBody(t, pages[1])
			child(t, child(t, body))
			size(pages[2], 200, 200)
			_, body = pageBody(t, pages[2])
			children(t, child(t, body), 1)
		}},
		{"a named page after a forced break to the right", `
      <style>
        @page { size: 200px 200px; margin: 0 }
        @page small { size: 100px 100px }
        p { page: small; break-before: right }
      </style>
      <section>normal</section>
      <section>normal</section>
      <p>small</p>
      <section>small</section>`, 3, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 200, 200)
			_, body := pageBody(t, pages[0])
			children(t, body, 2)
			size(pages[1], 200, 200)
			children(t, child(t, pages[1]), 0)
			size(pages[2], 100, 100)
			_, body = pageBody(t, pages[2])
			list := children(t, body, 2)
			if tagOf(list[0]) != "p" || tagOf(list[1]) != "section" {
				t.Errorf("last page has %v", list)
			}
		}},
		{"a named page splits at the end of the page", `
      <style>
        @page small { size: 100px 100px }
        section { page: small }
        p { line-height: 80px }
      </style>
      <section>
        <p>small</p>
        <p>small</p>
      </section>`, 2, func(t *testing.T, pages []*boxes.PageBox) {
			for _, page := range pages {
				size(page, 100, 100)
				_, body := pageBody(t, page)
				section := child(t, body)
				if p := child(t, section); tagOf(section) != "section" || tagOf(p) != "p" {
					t.Errorf("page has %v with %v", section, p)
				}
			}
		}},
		{"a forced break between pages of the same name", `
      <style>
        @page { size: 200px 200px }
        @page small { size: 100px 100px }
        section { break-after: page; page: small }
        article { page: small }
      </style>
      <section>
        <div>big</div>
        <div>big</div>
      </section>
      <article>
        <div>small</div>
        <div>small</div>
      </article>`, 2, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 100, 100)
			_, body := pageBody(t, pages[0])
			if section := child(t, body); tagOf(section) != "section" {
				t.Errorf("first page has %v", section)
			}
			size(pages[1], 100, 100)
			_, body = pageBody(t, pages[1])
			if article := child(t, body); tagOf(article) != "article" {
				t.Errorf("second page has %v", article)
			}
		}},
		{"fixed and running boxes", `
      <style>
        #running { position: running(running); }
        #fixed { position: fixed; }
        @page { size: 200px 200px; @top-center { content: element(header); }}
        section { page: small; }
        @page small { size: 100px 100px; }
        .pagebreak { break-after: page; }
      </style>
      <div id="running">running</div>
      <div id="fixed">fixed</div>
      <section>
        <h1>text</h1>
        <div class="pagebreak"></div>
        <article>text</article>
      </section>`, 2, func(t *testing.T, pages []*boxes.PageBox) {
			size(pages[0], 100, 100)
			if got := len(pages[0].Children); got != 2 {
				t.Fatalf("first page has %d children, want the html and a margin box", got)
			}
			_, body := pageBody2(t, pages[0])
			list := children(t, body, 2)
			if tagOf(list[0]) != "div" || tagOf(list[1]) != "section" {
				t.Errorf("first page body has %v, want the fixed div and the section", list)
			}
			section := list[1]
			items := children(t, section, 2)
			if tagOf(items[0]) != "h1" {
				t.Errorf("section has %v", items)
			}

			size(pages[1], 100, 100)
			html := pages[1].Children[0]
			htmlChildren := children(t, html, 2)
			if tagOf(htmlChildren[0]) != "div" {
				t.Errorf("second page html has %v, want the repeated fixed div first", htmlChildren)
			}
			article := child(t, child(t, htmlChildren[1]))
			if tagOf(article) != "article" {
				t.Errorf("second page has %v", article)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := pagesOf(t, renderPages(t, tt.source), tt.pages)
			tt.check(t, pages)
		})
	}
}

// pageBody2 returns the html and body boxes of a page that also has margin
// boxes after its content.
func pageBody2(t testing.TB, page *boxes.PageBox) (html, body boxes.Box) {
	t.Helper()
	html = page.Children[0]
	if tag := tagOf(html); tag != "html" {
		t.Fatalf("the first child of the page is %q, want html", tag)
	}
	var found boxes.Box
	for _, c := range boxes.ChildrenOf(html) {
		if tagOf(c) == "body" {
			found = c
		}
	}
	if found == nil {
		t.Fatalf("html has no body: %v", boxes.ChildrenOf(html))
	}
	return html, found
}

func TestPageGroups(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 200px 200px }
        @page small { size: 100px 100px }
        @page :nth(1 of small) { size: 50px 50px }
        section { page: small }
        div, div section { break-after: page }
      </style>
      <div></div>
      <article></article>
      <section>
        <div></div>
        <div></div>
      </section>
      <section>
      </section>
      <div></div>
      <div></div>
      <section>
        <div></div>
      </section>
      <div>
        <section></section>
        <section></section>
      </div>
    `), 9)

	// The content of each page below its body: the tag of the first box and
	// the tags of its children.
	tests := []struct {
		size   [2]float64
		top    []string
		nested []string
	}{
		{[2]float64{200, 200}, []string{"div"}, nil},
		{[2]float64{200, 200}, []string{"article"}, nil},
		{[2]float64{50, 50}, []string{"section"}, []string{"div"}},
		{[2]float64{100, 100}, []string{"section"}, []string{"div"}},
		{[2]float64{50, 50}, []string{"section", "div"}, nil},
		{[2]float64{200, 200}, []string{"div"}, nil},
		{[2]float64{50, 50}, []string{"section"}, []string{"div"}},
		{[2]float64{50, 50}, []string{"div"}, []string{"section"}},
		{[2]float64{50, 50}, []string{"div"}, []string{"section"}},
	}
	for i, tt := range tests {
		page := pages[i]
		if got := pageSize(page); got != tt.size {
			t.Errorf("page %d: size = %v, want %v", i+1, got, tt.size)
		}
		_, body := pageBody(t, page)
		list := children(t, body, len(tt.top))
		for j, tag := range tt.top {
			if tagOf(list[j]) != tag {
				t.Errorf("page %d: box %d is %v, want %s", i+1, j, list[j], tag)
			}
		}
		if tt.nested != nil {
			nested := children(t, list[0], len(tt.nested))
			for j, tag := range tt.nested {
				if tagOf(nested[j]) != tag {
					t.Errorf("page %d: nested box %d is %v, want %s", i+1, j, nested[j], tag)
				}
			}
		}
	}
}

func TestPageGroupsWithBlankPageInside(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 100px }
        @page div { size: 50px }
        div { page: div }
        p { break-before: right }
      </style>
      <div>
        <p>1</p>
        <p>2</p>
      </div>
    `), 3)
	for i, page := range pages {
		if got := pageSize(page); got != [2]float64{50, 50} {
			t.Errorf("page %d: size = %v, want 50 × 50", i+1, got)
		}
	}
}

func TestPageGroupsWithBlankPageOutside(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 100px }
        @page p { size: 50px }
        p { page: p; break-before: right }
      </style>
      <div>
        <p>1</p>
        <p>2</p>
      </div>
    `), 3)
	for i, want := range []float64{50, 100, 50} {
		if got := pageSize(pages[i]); got != [2]float64{want, want} {
			t.Errorf("page %d: size = %v, want %g × %g", i+1, got, want, want)
		}
	}
}

func TestPageGroupsFirstAndNth(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 100px }
        @page div { size: 50px }
        @page :nth(2n+1 of div) { size: 30px }
        div { page: div; break-before: right }
        p { break-before: page }
      </style>
      <div>
        <p>1</p>
        <p>2</p>
        <p>3</p>
      </div>
      <div>
        <p>4</p>
        <p>5</p>
      </div>
    `), 6)
	for i, want := range []float64{30, 50, 30, 100, 30, 50} {
		if got := pageSize(pages[i]); got != [2]float64{want, want} {
			t.Errorf("page %d: size = %v, want %g × %g", i+1, got, want, want)
		}
	}
}

func TestPageGroupsAndCounters(t *testing.T) {
	pages := pagesOf(t, renderPages(t, `
      <style>
        html { counter-reset: h1-counter }
        h1 { page-break-before: always; counter-increment: h1-counter; }
        div.main-content { page: main-content-page-group; break-before: page; }
        @page { size: 1000px }
        @page :nth(1 of main-content-page-group) { counter-reset: page 1; size: 500px; }
        a::after {
          content: target-counter(attr(href), h1-counter) '.'
          target-counter(attr(href), page);
        }
      </style>
      <html>
      <body>
        INTRO PAGE
        <div class="main-content">
        <h1 id="t2">First title</h1>
        <h1>Title3</h1>
        <a href="#t2">Second title</a>
        </div>
      </body>
      </html>
    `), 3)
	for i, want := range []float64{1000, 500, 1000} {
		if got := pageSize(pages[i]); got != [2]float64{want, want} {
			t.Errorf("page %d: size = %v, want %g × %g", i+1, got, want, want)
		}
	}

	pages = pagesOf(t, renderPages(t, `
      <!DOCTYPE html>
      <html>
        <head>
          <style>
            @page { size: 1000px }
            div { page: group }
            h1 { break-before: page }
            p.counter::after { content: counter(pages) }
            @page :nth(1 of group) { background-color: blue; size: 500px; }
          </style>
        </head>
        <body>
          INTRO PAGE
          <div>
            <h1>Title</h1>
            <p>content</p>
            <h1>Title</h1>
            <p class="counter">content</p>
          </div>
          <div>
            <h1>Title</h1>
            <p>content</p>
            <h1>Title</h1>
            <p class="counter">content</p>
          </div>
        </body>
      </html>
    `), 5)
	for i, want := range []float64{1000, 500, 1000, 500, 1000} {
		if got := pageSize(pages[i]); got != [2]float64{want, want} {
			t.Errorf("page %d: size = %v, want %g × %g", i+1, got, want, want)
		}
	}
}

func TestOrphansWidowsAndAvoid(t *testing.T) {
	tests := []struct {
		style      string
		lineCounts []int
	}{
		{"orphans: 2; widows: 2", []int{4, 3}},
		{"orphans: 5; widows: 2", []int{0, 7}},
		{"orphans: 2; widows: 4", []int{3, 4}},
		{"orphans: 4; widows: 4", []int{0, 7}},
		{"orphans: 2; widows: 2; page-break-inside: avoid", []int{0, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			// There is room for 4 lines after the h1 on the first page.
			pages := renderPages(t, fmt.Sprintf(`
      <style>
        @page { size: 200px }
        h1 { height: 120px }
        p { line-height: 20px;
            width: 1px; /* line break at each word */
            %s }
      </style>
      <h1>Tasty test</h1>
      <p>one two three four five six seven</p>
    `, tt.style))
			if len(pages) != len(tt.lineCounts) {
				t.Fatalf("got %d pages, want %d", len(pages), len(tt.lineCounts))
			}
			for i, page := range pages {
				_, body := pageBody(t, page)
				list := boxes.ChildrenOf(body)
				if i == 0 {
					// Skip the h1.
					list = list[1:]
				}
				count := 0
				if len(list) != 0 {
					count = len(boxes.ChildrenOf(list[0]))
				}
				if count != tt.lineCounts[i] {
					t.Errorf("page %d has %d lines, want %d", i+1, count, tt.lineCounts[i])
				}
			}
		})
	}
}

func TestPageAndLineboxBreaking(t *testing.T) {
	// The empty span tests a corner case of skipping the first white space.
	pages := pagesOf(t, renderPages(t, `
      <style>
        @page { size: 100px; margin: 2px; border: 1px solid }
        body { margin: 0 }
        div { font-family: testfont; font-size: 20px }
      </style>
      <div><span/>1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15</div>
    `), 4)
	var all string
	for _, page := range pages {
		_, body := pageBody(t, page)
		for _, line := range boxes.ChildrenOf(child(t, body)) {
			all += textOf(line)
		}
	}
	if want := "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15"; all != want {
		t.Errorf("text = %q, want %q", all, want)
	}
}
