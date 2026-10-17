package layout

import (
	"fmt"
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
