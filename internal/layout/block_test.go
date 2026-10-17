package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestBlockWidths(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page { margin: 0; size: 120px 2000px }
        body { margin: 0 }
        div { margin: 10px }
        p { padding: 2px; border-width: 1px; border-style: solid }
      </style>
      <div>
        <p></p>
        <p style="width: 50px"></p>
      </div>
      <div style="direction: rtl">
        <p style="width: 50px; direction: rtl"></p>
      </div>
      <div>
        <p style="margin: 0 10px 0 20px"></p>
        <p style="width: 50px; margin-left: 20px; margin-right: auto"></p>
        <p style="width: 50px; margin-left: auto; margin-right: 20px"></p>
        <p style="width: 50px; margin: auto"></p>

        <p style="margin-left: 20px; margin-right: auto"></p>
        <p style="margin-left: auto; margin-right: 20px"></p>
        <p style="margin: auto"></p>

        <p style="width: 200px; margin: auto"></p>

        <p style="min-width: 200px; margin: auto"></p>
        <p style="max-width: 50px; margin: auto"></p>
        <p style="min-width: 50px; margin: auto"></p>

        <p style="width: 70%"></p>
      </div>
    `))
	_, body := pageBody(t, page)
	if body.Base().ElementTag != "body" || body.Base().Width.V != 120 {
		t.Fatalf("body = %v width %v", body, body.Base().Width)
	}

	var paragraphs []boxes.Box
	for _, div := range boxes.ChildrenOf(body) {
		if div.Kind() != boxes.KindBlock || div.Base().ElementTag != "div" || div.Base().Width.V != 100 {
			t.Fatalf("div = %v width %v", div, div.Base().Width)
		}
		for _, p := range boxes.ChildrenOf(div) {
			f := p.Base()
			if p.Kind() != boxes.KindBlock || f.ElementTag != "p" ||
				f.PaddingLeft != 2 || f.PaddingRight != 2 || f.BorderLeftWidth != 1 || f.BorderRightWidth != 1 {
				t.Fatalf("paragraph = %v", p)
			}
			paragraphs = append(paragraphs, p)
		}
	}
	if len(paragraphs) != 15 {
		t.Fatalf("got %d paragraphs, want 15", len(paragraphs))
	}

	tests := []struct {
		name           string
		width, marginL float64
		marginR        float64
		checkMarginR   bool
	}{
		{"auto width", 94, 0, 0, false},
		{"over-constrained ltr ignores the initial margin-right", 50, 0, 0, false},
		{"over-constrained rtl ignores the initial margin-left", 50, 0, 0, true},
		{"auto width with margins", 64, 20, 0, false},
		{"auto margin-right", 50, 20, 0, false},
		{"auto margin-left", 50, 24, 0, false},
		{"both margins auto split the space", 50, 22, 0, false},
		{"auto width, other autos are zero (left)", 74, 20, 0, false},
		{"auto width, other autos are zero (right)", 74, 0, 0, false},
		{"auto width, both autos zero", 94, 0, 0, false},
		{"too wide, autos are zero", 200, 0, 0, false},
		{"min-width wider than the box", 200, 0, 0, false},
		{"max-width", 50, 22, 0, false},
		{"min-width not reached", 94, 0, 0, false},
		{"percentage", 70, 0, 0, false},
	}
	if paragraphs[0].Base().MarginRight.V != 0 {
		t.Errorf("paragraph 0 margin-right = %g, want 0", paragraphs[0].Base().MarginRight.V)
	}
	for i, tt := range tests {
		f := paragraphs[i].Base()
		got := f.MarginLeft.V
		want := tt.marginL
		if tt.checkMarginR {
			got, want = f.MarginRight.V, tt.marginR
		}
		if f.Width.V != tt.width || got != want {
			t.Errorf("paragraph %d (%s): width %g, margin %g; want %g, %g", i, tt.name, f.Width.V, got, tt.width, want)
		}
	}
}

func TestBlockPercentageHeightsNoHTMLHeight(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        html, body { margin: 0 }
        body { height: 50% }
      </style>
    `))
	html, body := pageBody(t, page)
	if html.Base().ElementTag != "html" || body.Base().ElementTag != "body" {
		t.Fatalf("boxes = %v, %v", html, body)
	}
	// The height of html depends on the one of body, so 50% means auto.
	if got := body.Base().Height.V; got != 0 {
		t.Fatalf("body height = %g, want 0", got)
	}
}

func TestBlockPercentageHeights(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        html, body { margin: 0 }
        html { height: 300px }
        body { height: 50% }
      </style>
    `))
	_, body := pageBody(t, page)
	if got := body.Base().Height.V; got != 150 {
		t.Fatalf("body height = %g, want 150", got)
	}
}

func TestBoxSizing(t *testing.T) {
	for _, size := range []string{
		"width: 10%; height: 1000px",
		"max-width: 10%; max-height: 1000px; height: 2000px",
		"width: 5%; min-width: 10%; min-height: 1000px",
		"width: 10%; height: 1000px; min-width: auto; max-height: none",
	} {
		t.Run(size, func(t *testing.T) {
			noLogs(t)
			page := onlyPage(t, paginate(t, fmt.Sprintf(`
      <style>
        @page { size: 100000px }
        body { width: 10000px; margin: 0 }
        div { %s; margin: 100px; padding: 10px; border: 1px solid }
      </style>
      <div></div>

      <div style="box-sizing: content-box"></div>
      <div style="box-sizing: padding-box"></div>
      <div style="box-sizing: border-box"></div>
    `, size)))
			_, body := pageBody(t, page)
			divs := expectChildren(t, body, 4)
			type sizes struct {
				boxSizing                                     string
				width, height                                 float64
				paddingW, paddingH, borderW, borderH, marginH float64
			}
			want := []sizes{
				{"content-box", 1000, 1000, 1020, 1020, 1022, 1022, 1222},
				{"content-box", 1000, 1000, 1020, 1020, 1022, 1022, 1222},
				{"padding-box", 980, 980, 1000, 1000, 1002, 1002, 1202},
				{"border-box", 978, 978, 998, 998, 1000, 1000, 1200},
			}
			for i, div := range divs {
				got := sizes{
					string(div.Base().Style.BoxSizing()), div.Base().Width.V, div.Base().Height.V,
					div.PaddingWidth(), div.PaddingHeight(), div.BorderWidth(), div.BorderHeight(), div.MarginHeight(),
				}
				if got != want[i] {
					t.Errorf("div %d = %+v, want %+v", i+1, got, want[i])
				}
			}
		})
	}
}

func TestBoxSizingZero(t *testing.T) {
	for _, size := range []string{
		"width: 0; height: 0",
		"max-width: 0; max-height: 0",
		"min-width: 0; min-height: 0; width: 0; height: 0",
	} {
		t.Run(size, func(t *testing.T) {
			noLogs(t)
			page := onlyPage(t, paginate(t, fmt.Sprintf(`
      <style>
        @page { size: 100000px }
        body { width: 10000px; margin: 0 }
        div { %s; margin: 100px; padding: 10px; border: 1px solid }
      </style>
      <div></div>

      <div style="box-sizing: content-box"></div>
      <div style="box-sizing: padding-box"></div>
      <div style="box-sizing: border-box"></div>
    `, size)))
			_, body := pageBody(t, page)
			for i, div := range boxes.ChildrenOf(body) {
				got := [7]float64{
					div.Base().Width.V, div.Base().Height.V, div.PaddingWidth(), div.PaddingHeight(),
					div.BorderWidth(), div.BorderHeight(), div.MarginHeight(),
				}
				if want := [7]float64{0, 0, 20, 20, 22, 22, 222}; got != want {
					t.Errorf("div %d = %v, want %v", i+1, got, want)
				}
			}
		})
	}
}

func TestBoxDecorationBreakBlockSlice(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 100px }
        p { padding: 2px; border: 3px solid; margin: 5px }
        img { display: block; height: 40px }
      </style>
      <p>
        <img src=pattern.png>
        <img src=pattern.png>
        <img src=pattern.png>
        <img src=pattern.png>`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}

	_, body := pageBody(t, pages[0])
	paragraph := expectChildren(t, body, 1)[0]
	img1, img2 := expectChildren(t, paragraph, 2)[0], boxes.ChildrenOf(paragraph)[1]
	p := paragraph.Base()
	checks := []struct {
		name      string
		got, want float64
	}{
		{"position_y", p.PositionY, 0},
		{"margin_top", p.MarginTop.V, 5},
		{"border_top_width", p.BorderTopWidth, 3},
		{"padding_top", p.PaddingTop, 2},
		{"content_box_y", paragraph.ContentBoxY(), 10},
		{"img 1 position_y", img1.Base().PositionY, 10},
		{"img 2 position_y", img2.Base().PositionY, 50},
		{"height", p.Height.V, 90},
		{"margin_bottom", p.MarginBottom.V, 0},
		{"border_bottom_width", p.BorderBottomWidth, 0},
		{"padding_bottom", p.PaddingBottom, 0},
		{"margin_height", paragraph.MarginHeight(), 100},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("page 1 %s = %g, want %g", c.name, c.got, c.want)
		}
	}

	_, body = pageBody(t, pages[1])
	paragraph = expectChildren(t, body, 1)[0]
	img1, img2 = expectChildren(t, paragraph, 2)[0], boxes.ChildrenOf(paragraph)[1]
	p = paragraph.Base()
	checks = []struct {
		name      string
		got, want float64
	}{
		{"position_y", p.PositionY, 0},
		{"margin_top", p.MarginTop.V, 0},
		{"border_top_width", p.BorderTopWidth, 0},
		{"padding_top", p.PaddingTop, 0},
		{"content_box_y", paragraph.ContentBoxY(), 0},
		{"img 1 position_y", img1.Base().PositionY, 0},
		{"img 2 position_y", img2.Base().PositionY, 40},
		{"height", p.Height.V, 80},
		{"padding_bottom", p.PaddingBottom, 2},
		{"border_bottom_width", p.BorderBottomWidth, 3},
		{"margin_bottom", p.MarginBottom.V, 5},
		{"margin_height", paragraph.MarginHeight(), 90},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("page 2 %s = %g, want %g", c.name, c.got, c.want)
		}
	}
}

func TestBoxDecorationBreakBlockClone(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 100px }
        p { padding: 2px; border: 3px solid; margin: 5px;
            box-decoration-break: clone }
        img { display: block; height: 40px }
      </style>
      <p>
        <img src=pattern.png>
        <img src=pattern.png>
        <img src=pattern.png>
        <img src=pattern.png>`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}

	_, body := pageBody(t, pages[0])
	paragraph := expectChildren(t, body, 1)[0]
	img1, img2 := expectChildren(t, paragraph, 2)[0], boxes.ChildrenOf(paragraph)[1]
	p := paragraph.Base()
	checks := []struct {
		name      string
		got, want float64
	}{
		{"position_y", p.PositionY, 0},
		{"margin_top", p.MarginTop.V, 5},
		{"border_top_width", p.BorderTopWidth, 3},
		{"padding_top", p.PaddingTop, 2},
		{"content_box_y", paragraph.ContentBoxY(), 10},
		{"img 1 position_y", img1.Base().PositionY, 10},
		{"img 2 position_y", img2.Base().PositionY, 50},
		{"height", p.Height.V, 80},
		// Cloned margins should be truncated on block-level boxes, so the
		// bottom margin of this fragment should be 0 (CSS Fragmentation 3
		// §5.4); it is kept.
		{"margin_bottom", p.MarginBottom.V, 5},
		{"border_bottom_width", p.BorderBottomWidth, 3},
		{"padding_bottom", p.PaddingBottom, 2},
		{"margin_height", paragraph.MarginHeight(), 100},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("page 1 %s = %g, want %g", c.name, c.got, c.want)
		}
	}

	_, body = pageBody(t, pages[1])
	paragraph = expectChildren(t, body, 1)[0]
	img1, img2 = expectChildren(t, paragraph, 2)[0], boxes.ChildrenOf(paragraph)[1]
	p = paragraph.Base()
	checks = []struct {
		name      string
		got, want float64
	}{
		{"position_y", p.PositionY, 0},
		{"margin_top", p.MarginTop.V, 0},
		{"border_top_width", p.BorderTopWidth, 3},
		{"padding_top", p.PaddingTop, 2},
		{"content_box_y", paragraph.ContentBoxY(), 5},
		{"img 1 position_y", img1.Base().PositionY, 5},
		{"img 2 position_y", img2.Base().PositionY, 45},
		{"height", p.Height.V, 80},
		{"padding_bottom", p.PaddingBottom, 2},
		{"border_bottom_width", p.BorderBottomWidth, 3},
		{"margin_bottom", p.MarginBottom.V, 5},
		{"margin_height", paragraph.MarginHeight(), 95},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("page 2 %s = %g, want %g", c.name, c.got, c.want)
		}
	}
}

func TestPageBreaksPushDivThatDoesNotFit(t *testing.T) {
	tests := []struct {
		name    string
		large   int
		small   string
		pages   int
		wantTop [][]float64
	}{
		{"last div does not fit", 60, "", 2, [][]float64{{10, 70}, {10}}},
		{"center div must not be last", 60, "page-break-after: avoid;", 2, [][]float64{{10}, {10, 30}}},
		{"avoid is ignored when nothing else fits", 80, "page-break-after: avoid;", 3, [][]float64{{10}, {10}, {10}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			pages := paginate(t, fmt.Sprintf(`
      <style>
        @page{
          size: 110px;
          margin: 10px;
          padding: 0;
        }
        .large {
          width: 10px;
          height: %dpx;
        }
        .small {
          width: 10px;
          height: 20px;
          %s
        }
      </style>
      <body>
        <div class="large"></div>
        <div class="small"></div>
        <div class="large"></div>
    `, tt.large, tt.small))
			if len(pages) != tt.pages {
				t.Fatalf("got %d pages, want %d", len(pages), tt.pages)
			}
			var got [][]float64
			for _, page := range pages {
				var positions []float64
				for box := range boxes.Descendants(page, false) {
					if box.Base().ElementTag == "div" {
						positions = append(positions, box.Base().PositionY)
					}
				}
				got = append(got, positions)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.wantTop) {
				t.Fatalf("div positions = %v, want %v", got, tt.wantTop)
			}
		})
	}
}

func TestMinMaxRTL(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page{ size: 10px }
      </style>
      <body style="direction: rtl">
        <div style="height: 5px; width: 1px; max-height: 4px; min-width: 3px"></div>
        <div style="height: 1px; width: 5px; min-height: 4px; max-width: 3px"></div>
      </body>
    `))
	_, body := pageBody(t, page)
	divs := expectChildren(t, body, 2)
	want := [][4]float64{{7, 0, 3, 4}, {7, 4, 3, 4}}
	for i, div := range divs {
		f := div.Base()
		if got := [4]float64{f.PositionX, f.PositionY, f.Width.V, f.Height.V}; got != want[i] {
			t.Errorf("div %d (x, y, width, height) = %v, want %v", i+1, got, want[i])
		}
	}
}

func TestBlockHeightsOfParagraphs(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page { margin: 0; size: 100px 20000px }
        html, body { margin: 0 }
        div { margin: 4px; border: 2px solid; padding: 4px }
        /* Use top margins so that margin collapsing doesn't change result */
        p { margin: 16px 0 0; border: 4px solid; padding: 8px; height: 50px }
      </style>
      <div>
        <p></p>
        <!-- Not in normal flow: don't contribute to the parent’s height -->
        <p style="position: absolute"></p>
        <p style="float: left"></p>
      </div>
      <div> <p></p> <p></p> <p></p> </div>
      <div style="height: 20px"> <p></p> </div>
      <div style="height: 120px"> <p></p> </div>
      <div style="max-height: 20px"> <p></p> </div>
      <div style="min-height: 120px"> <p></p> </div>
      <div style="min-height: 20px"> <p></p> </div>
      <div style="max-height: 120px"> <p></p> </div>
    `))
	_, body := pageBody(t, page)
	var heights []float64
	for _, div := range boxes.ChildrenOf(body) {
		heights = append(heights, div.Base().Height.V)
	}
	if want := []float64{90, 90 * 3, 20, 120, 20, 120, 90, 90}; fmt.Sprint(heights) != fmt.Sprint(want) {
		t.Fatalf("heights = %v, want %v", heights, want)
	}
}

func TestBlockHeightsOfImages(t *testing.T) {
	tests := []struct {
		name   string
		style  string
		want   []float64
		bodyCh string
	}{
		{
			name:   "body with a height",
			style:  "body { height: 200px; font-size: 0 }",
			want:   []float64{40, 20, 20, 20, 20, 20},
			bodyCh: "",
		},
		{
			// Percentages of the height are ignored when body has none.
			name:  "body without a height",
			style: "body { font-size: 0 }",
			want:  []float64{40, 40, 20, 40, 20, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			page := onlyPage(t, paginate(t, fmt.Sprintf(`
      <style>
        %s
      </style>
      <div>
        <img src=pattern.png style="height: 40px">
      </div>
      <div style="height: 10%%">
        <img src=pattern.png style="height: 40px">
      </div>
      <div style="max-height: 20px">
        <img src=pattern.png style="height: 40px">
      </div>
      <div style="max-height: 10%%">
        <img src=pattern.png style="height: 40px">
      </div>
      <div style="min-height: 20px"></div>
      <div style="min-height: 10%%"></div>
    `, tt.style)))
			_, body := pageBody(t, page)
			var heights []float64
			for _, div := range boxes.ChildrenOf(body) {
				heights = append(heights, div.Base().Height.V)
			}
			if fmt.Sprint(heights) != fmt.Sprint(tt.want) {
				t.Fatalf("heights = %v, want %v", heights, tt.want)
			}
		})
	}
}

var (
	collapsingMargins = []struct {
		first, second string
		want          float64
	}{
		{"10px", "15px", 15}, // not 25
		// The maximum of the absolute values of the negative adjoining
		// margins is deducted from the maximum of the positive ones.
		{"-10px", "15px", 5},
		{"10px", "-15px", -5},
		{"-10px", "-15px", -15},
		{"10px", "auto", 10}, // auto is 0
	}
	notCollapsingMargins = []struct {
		first, second string
		want          float64
	}{
		{"10px", "15px", 25},
		{"-10px", "15px", 5},
		{"10px", "-15px", -5},
		{"-10px", "-15px", -25},
		{"10px", "auto", 10}, // auto is 0
	}
)

// verticalSpace lays out a document where two paragraphs of 20px have the
// given margins and returns the boxes of its body.
func verticalSpace(t *testing.T, style, markup string) []boxes.Box {
	t.Helper()
	page := onlyPage(t, paginate(t, `
      <style>
        p { font: 20px/1 serif } /* block height == 20px */
        `+style+`
      </style>
      `+markup))
	_, body := pageBody(t, page)
	return boxes.ChildrenOf(body)
}

// gapBetween returns the distance from the bottom of the content of a box to
// the top of the content of another.
func gapBetween(first, second boxes.Box) float64 {
	return second.ContentBoxY() - (first.ContentBoxY() + first.Base().Height.V)
}

func TestVerticalSpaceBetweenSiblings(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#p1 { margin-bottom: %s } #p2 { margin-top: %s }", tt.first, tt.second),
				"<p id=p1>Lorem ipsum\n<p id=p2>dolor sit amet")
			p1, p2 := children[0], children[1]
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceWhenTheFirstIsNested(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#p1 { margin-bottom: %s } #p2 { margin-top: %s }", tt.first, tt.second),
				"<div>\n<p id=p1>Lorem ipsum\n</div>\n<p id=p2>dolor sit amet")
			div, p2 := children[0], children[1]
			p1 := expectChildren(t, div, 1)[0]
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceWhenTheSecondIsNested(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#p1 { margin-bottom: %s } #p2 { margin-top: %s }", tt.first, tt.second),
				"<p id=p1>Lorem ipsum\n<div>\n<p id=p2>dolor sit amet\n</div>")
			p1, div := children[0], children[1]
			p2 := expectChildren(t, div, 1)[0]
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceWhenTheSecondIsDoublyNested(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#p1 { margin-bottom: %s } #p2 { margin-top: %s }", tt.first, tt.second),
				"<p id=p1>Lorem ipsum\n<div>\n<div>\n<p id=p2>dolor sit amet\n</div>\n</div>")
			p1, div1 := children[0], children[1]
			p2 := expectChildren(t, expectChildren(t, div1, 1)[0], 1)[0]
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceCollapsesWithChildren(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#div1 { margin-top: %s } #div2 { margin-top: %s }", tt.first, tt.second),
				"<p>Lorem ipsum\n<div id=div1>\n<div id=div2>\n<p id=p2>dolor sit amet\n</div>\n</div>")
			p1, div1 := children[0], children[1]
			div2 := expectChildren(t, div1, 1)[0]
			p2 := expectChildren(t, div2, 1)[0]
			// The parent and the child share their edge.
			if div1.BorderBoxY() != p2.BorderBoxY() || div2.BorderBoxY() != p2.BorderBoxY() {
				t.Errorf("border box tops: div1 %g, div2 %g, p2 %g", div1.BorderBoxY(), div2.BorderBoxY(), p2.BorderBoxY())
			}
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceDoesNotCollapseWithChildrenOfFormattingContext(t *testing.T) {
	for _, tt := range notCollapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#div1 { margin-top: %s; overflow: hidden } #div2 { margin-top: %s }", tt.first, tt.second),
				"<p>Lorem ipsum\n<div id=div1>\n<div id=div2>\n<p id=p2>dolor sit amet\n</div>\n</div>")
			p1, div1 := children[0], children[1]
			p2 := expectChildren(t, expectChildren(t, div1, 1)[0], 1)[0]
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceCollapsesThroughAnEmptyDiv(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("#p1 { margin-bottom: %[1]s } #p2 { margin-top: %[2]s } div { margin-bottom: %[1]s; margin-top: %[2]s }", tt.first, tt.second),
				"<p id=p1>Lorem ipsum\n<div></div>\n<p id=p2>dolor sit amet")
			p1, p2 := children[0], children[2]
			if got := gapBetween(p1, p2); got != tt.want {
				t.Fatalf("gap = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceOfTheRootElementDoesNotCollapse(t *testing.T) {
	for _, tt := range notCollapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("html { margin-top: %s } body { margin-top: %s }", tt.first, tt.second),
				"<p>Lorem ipsum")
			// The space is counted from the top of the page.
			if got := children[0].ContentBoxY(); got != tt.want {
				t.Fatalf("top of the paragraph = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestVerticalSpaceOfBodyCollapses(t *testing.T) {
	for _, tt := range collapsingMargins {
		t.Run(tt.first+"/"+tt.second, func(t *testing.T) {
			children := verticalSpace(t,
				fmt.Sprintf("body { margin-top: %s } div { margin-top: %s }", tt.first, tt.second),
				"<div>\n<p>Lorem ipsum")
			p1 := expectChildren(t, children[0], 1)[0]
			// The space is counted from the top of the page.
			if got := p1.ContentBoxY(); got != tt.want {
				t.Fatalf("top of the paragraph = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestBoxDecorationBreakCloneBottomPadding(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 80px; margin: 0 }
        div { height: 20px }
        article { padding: 12px; box-decoration-break: clone }
      </style>
      <article>
        <div>a</div>
        <div>b</div>
        <div>c</div>
      </article>`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	article := expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 80-2*12 {
		t.Errorf("page 1 article height = %g, want %d", got, 80-2*12)
	}
	divs := expectChildren(t, article, 2)
	if divs[0].Base().PositionY != 12 || divs[1].Base().PositionY != 12+20 {
		t.Errorf("page 1 divs at y = %g, %g; want 12, 32", divs[0].Base().PositionY, divs[1].Base().PositionY)
	}

	_, body = pageBody(t, pages[1])
	article = expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 20 {
		t.Errorf("page 2 article height = %g, want 20", got)
	}
	if div := expectChildren(t, article, 1)[0]; div.Base().PositionY != 12 {
		t.Errorf("page 2 div at y = %g, want 12", div.Base().PositionY)
	}
}

func TestBoxDecorationBreakSliceBottomPadding(t *testing.T) {
	t.Skip("not supported: a block whose bottom padding crosses the page is not laid out again with room for it")
	// The last div fits in the first page, but not the padding of the
	// article. As it is impossible to break between a parent and its last
	// child, the last child goes to the next page.
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 80px; margin: 0 }
        div { height: 20px }
        article { padding: 12px; box-decoration-break: slice }
      </style>
      <article>
        <div>a</div>
        <div>b</div>
        <div>c</div>
      </article>`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	article := expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 80-12 {
		t.Errorf("page 1 article height = %g, want %d", got, 80-12)
	}
	divs := expectChildren(t, article, 2)
	if divs[0].Base().PositionY != 12 || divs[1].Base().PositionY != 12+20 {
		t.Errorf("page 1 divs at y = %g, %g; want 12, 32", divs[0].Base().PositionY, divs[1].Base().PositionY)
	}
	_, body = pageBody(t, pages[1])
	article = expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 20 {
		t.Errorf("page 2 article height = %g, want 20", got)
	}
	if div := expectChildren(t, article, 1)[0]; div.Base().PositionY != 0 {
		t.Errorf("page 2 div at y = %g, want 0", div.Base().PositionY)
	}
}

func TestNestedBlocksPaddingBorder(t *testing.T) {
	t.Skip("not supported: nested blocks whose bottom padding and border cross the page")
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 4px 12px; margin: 0 }
        html { font: 2px/1 testfont }
        article, section, div {
          border-top: 1px solid; border-bottom: 1px solid;
          padding-top: 1px; padding-bottom: 1px;
        }
      </style>
      <article>
        <section>
          <div>
            aaa
            bbb
          </div>
        </section>
      </article>`)
	for _, page := range pages {
		_, body := pageBody(t, page)
		article := expectChildren(t, body, 1)[0]
		section := expectChildren(t, article, 1)[0]
		expectChildren(t, section, 1)
	}
}

func TestOverflowNonCollapsingParent(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 6px 10px; margin: 0 }
        html { font: 2px/1 testfont }
        div { border-bottom: 3px solid }
        p { margin-bottom: 2px }
      </style>
      abc
      def
      <div>
        <p>
          aaa
        </p>
      </div>
      ghi`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	lines := expectChildren(t, body, 1)[0]
	expectChildren(t, lines, 2)

	_, body = pageBody(t, pages[1])
	expectChildren(t, body, 2)
}

func TestOverflowAutoContainsItsFloats(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <article style="overflow: auto">
        <div style="float: left; height: 50px; margin: 10px">bla bla bla</div>
          toto toto`))
	_, body := pageBody(t, page)
	article := expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 50+10+10 {
		t.Fatalf("article height = %g, want 70", got)
	}
}

func TestOverflowHiddenInFlowLayout(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <div style="overflow: hidden; height: 3px;">
        <div>abc</div>
        <div style="height: 100px; margin: 50px;">def</div>
      </div>
    `))
	_, body := pageBody(t, page)
	parent := expectChildren(t, body, 1)[0]
	if got := parent.Base().Height.V; got != 3 {
		t.Fatalf("height = %g, want 3", got)
	}
}

func TestOverflowHiddenOutOfFlowLayout(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <div style="overflow: hidden; height: 3px;">
        <div style="float: left;">abc</div>
        <div style="float: right; height: 100px; margin: 50px;">def</div>
      </div>
    `))
	_, body := pageBody(t, page)
	parent := expectChildren(t, body, 1)[0]
	if got := parent.Base().Height.V; got != 3 {
		t.Fatalf("height = %g, want 3", got)
	}
}

func TestContinueDiscard(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page { size: 80px; margin: 0 }
        div { display: inline-block; width: 100%; height: 25px }
        article { continue: discard; border: 1px solid; line-height: 1 }
      </style>
      <article>
        <div>a</div>
        <div>b</div>
        <div>c</div>
        <div>d</div>
        <div>e</div>
        <div>f</div>
      </article>`))
	_, body := pageBody(t, page)
	article := expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 3*25 {
		t.Errorf("article height = %g, want 75", got)
	}
	lines := expectChildren(t, article, 3)
	for i, line := range lines {
		if got, want := line.Base().PositionY, float64(1+25*i); got != want {
			t.Errorf("line %d at y = %g, want %g", i+1, got, want)
		}
	}
	if got := article.Base().BorderBottomWidth; got != 1 {
		t.Errorf("border-bottom-width = %g, want 1", got)
	}
}

func TestContinueDiscardChildren(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page { size: 80px; margin: 0 }
        div { display: inline-block; width: 100%; height: 25px }
        section { border: 1px solid }
        article { continue: discard; border: 1px solid; line-height: 1 }
      </style>
      <article>
        <section>
          <div>a</div>
          <div>b</div>
          <div>c</div>
          <div>d</div>
          <div>e</div>
          <div>f</div>
        </section>
      </article>`))
	_, body := pageBody(t, page)
	article := expectChildren(t, body, 1)[0]
	if got := article.Base().Height.V; got != 2+3*25 {
		t.Errorf("article height = %g, want 77", got)
	}
	section := expectChildren(t, article, 1)[0]
	if got := section.Base().Height.V; got != 3*25 {
		t.Errorf("section height = %g, want 75", got)
	}
	lines := expectChildren(t, section, 3)
	for i, line := range lines {
		if got, want := line.Base().PositionY, float64(2+25*i); got != want {
			t.Errorf("line %d at y = %g, want %g", i+1, got, want)
		}
	}
	if got := article.Base().BorderBottomWidth; got != 1 {
		t.Errorf("border-bottom-width = %g, want 1", got)
	}
}

// lineText returns the text of the first text box of a line box.
func lineText(t *testing.T, line boxes.Box) string {
	t.Helper()
	text, ok := expectChildren(t, line, 1)[0].(*boxes.TextBox)
	if !ok {
		t.Fatalf("first child of %v is not text", line)
	}
	return text.Text
}

func TestBlockInBlockWithBottomPadding(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 8em 3.5em }
        body { line-height: 1; font-family: testfont }
        div { padding-bottom: 1em }
      </style>
      abc def
      <div>
        <p>
          ghi jkl
          mno pqr
        </p>
      </div>
      stu vwx`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}

	_, body := pageBody(t, pages[0])
	children := expectChildren(t, body, 2)
	anon, div := children[0], children[1]
	line := expectChildren(t, anon, 1)[0]
	if line.Base().Height.V != 16 || lineText(t, line) != "abc def" {
		t.Errorf("page 1 anonymous line = %g high, %q", line.Base().Height.V, lineText(t, line))
	}
	line = expectChildren(t, expectChildren(t, div, 1)[0], 1)[0]
	if line.Base().Height.V != 16 || lineText(t, line) != "ghi jkl" {
		t.Errorf("page 1 paragraph line = %g high, %q", line.Base().Height.V, lineText(t, line))
	}

	_, body = pageBody(t, pages[1])
	children = expectChildren(t, body, 2)
	div, anon = children[0], children[1]
	line = expectChildren(t, expectChildren(t, div, 1)[0], 1)[0]
	if line.Base().Height.V != 16 || lineText(t, line) != "mno pqr" {
		t.Errorf("page 2 paragraph line = %g high, %q", line.Base().Height.V, lineText(t, line))
	}
	line = expectChildren(t, anon, 1)[0]
	if line.Base().Height.V != 16 || line.ContentBoxY() != 16+16 || lineText(t, line) != "stu vwx" {
		t.Errorf("page 2 anonymous line = %g high at %g, %q; want 16 high at 32",
			line.Base().Height.V, line.ContentBoxY(), lineText(t, line))
	}
}

func TestUnsupportedLayoutModesAreFatalErrors(t *testing.T) {
	tests := []struct{ name, markup string }{
		{"flex container", `<div style="display: flex"></div>`},
		{"grid container", `<div style="display: grid"></div>`},
		{"table", `<table><tr><td>a</td></tr></table>`},
		{"columns", `<div style="column-count: 2">a</div>`},
		{"floated flex container", `<div style="float: left; display: flex"></div>`},
		{"absolute grid container", `<div style="position: absolute; display: grid"></div>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages, err := paginateWithError(t, tt.markup)
			if err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("error = %v, want one about a layout mode that is not supported", err)
			}
			if pages != nil {
				t.Fatalf("got %d pages besides the error", len(pages))
			}
		})
	}
}
