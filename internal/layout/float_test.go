package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestFloatCollisionMovesBelowInsufficientSpace(t *testing.T) {
	containing := parseBox(t, `<body style="width: 290px">`, "body")
	resolvePercentages(containing, size(290, 500))
	containing.Base().Height = units.F(500)
	shape := parseBox(t, `<div style="float: left; width: 100px; height: 60px">`, "div")
	resolvePercentages(shape, size(290, 500))
	second := shape.Copy()
	second.Base().PositionX = 100
	ctx := &Context{}
	ctx.CreateBFC(nil, []boxes.Box{shape, second})
	x, y, width := avoidCollisions(ctx, shape, containing, true)
	if x != 0 || y != 60 || width != 290 {
		t.Fatalf("collision = (%g,%g,%g), want (0,60,290)", x, y, width)
	}
	line := boxes.NewLineBox("body", containing.Base().Style, nil, nil)
	line.Width, line.Height = units.F(60), units.F(60)
	x, y, width = avoidCollisions(ctx, line, containing, false)
	if x != 200 || y != 0 || width != 90 {
		t.Fatalf("line collision = (%g,%g,%g), want (200,0,90)", x, y, width)
	}
}

func TestFloatClearanceMatchesLogicalDirections(t *testing.T) {
	shape := parseBox(t, `<div style="float: inline-start; height: 60px">`, "div")
	resolvePercentages(shape, size(100, 500))
	block := parseBox(t, `<div style="clear: right; margin-top: 10px">`, "div")
	resolvePercentages(block, size(100, 500))
	ctx := &Context{}
	ctx.CreateBFC(nil, []boxes.Box{shape})
	if clearance := getClearance(ctx, block, props.Keyword("ltr"), 10); clearance != nil {
		t.Fatalf("unexpected clearance: %g", *clearance)
	}
	if clearance := getClearance(ctx, block, props.Keyword("rtl"), 10); clearance == nil || *clearance != 50 {
		t.Fatalf("clearance = %v, want 50", clearance)
	}
}

// outerArea returns the position and the size of the margin box.
func outerArea(box boxes.Box) [4]float64 {
	return [4]float64{box.Base().PositionX, box.Base().PositionY, box.MarginWidth(), box.MarginHeight()}
}

func TestFloatsPercentageOffsetsOfAutoHeightContainer(t *testing.T) {
	noLogs(t)
	paginate(t, `<div style="top:100%; float:left">`)
}

func TestFloatsPageBreaks(t *testing.T) {
	tests := []struct {
		name      string
		height    int
		images    int
		wantTop   [][]float64
		wantPages int
	}{
		{"floats shorter than the page", 45, 2, [][]float64{{10}, {10}}, 2},
		{"floats taller than the page", 81, 2, [][]float64{{10}, {10}}, 2},
		{"several floats on each page", 30, 5, [][]float64{{10, 40}, {10, 40}, {10}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			var tags string
			for range tt.images {
				tags += "<img src=pattern.png>\n"
			}
			pages := paginate(t, fmt.Sprintf(`
      <style>
        @page { size: 100px; margin: 10px }
        img { height: %dpx; width:70px; float: left;}
      </style>
      <body>%s`, tt.height, tags))
			if len(pages) != tt.wantPages {
				t.Fatalf("got %d pages, want %d", len(pages), tt.wantPages)
			}
			for i, page := range pages {
				var tops []float64
				for box := range boxes.Descendants(page, false) {
					if box.Base().ElementTag == "img" {
						if box.Base().PositionX != 10 {
							t.Errorf("page %d: image at x = %g, want 10", i+1, box.Base().PositionX)
						}
						tops = append(tops, box.Base().PositionY)
					}
				}
				if fmt.Sprint(tops) != fmt.Sprint(tt.wantTop[i]) {
					t.Errorf("page %d: image tops = %v, want %v", i+1, tops, tt.wantTop[i])
				}
			}
		})
	}
}

func TestFormattingContextAvoidsFloatsInRTL(t *testing.T) {
	noLogs(t)
	paginate(t, `
      <div style="direction: rtl">
        <div style="overflow: hidden"></div>
      </div>
    `)
}

func TestFloatsStackToTheirSides(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 200px }
        p { width: 70px; height: 20px; margin: 0 }
        .left { float: left }
        .right { float: right }
      </style>
      <p class="left"></p>
      <p class="left"></p>
      <p class="left"></p>
      <p class="right"></p>
      <p class="left"></p>
      <p class="right"></p>
      <p class="right"></p>
      <p class="left"></p>
      <p class="left"></p>
      <p class="left"></p>
    `))
	_, body := pageBody(t, page)
	var positions [][2]float64
	for _, p := range boxes.ChildrenOf(body) {
		positions = append(positions, [2]float64{p.Base().PositionX, p.Base().PositionY})
	}
	want := [][2]float64{
		{0, 0}, {70, 0}, {0, 20}, {130, 20}, {0, 40}, {130, 40},
		{130, 60}, {0, 60}, {0, 80}, {70, 80},
	}
	if fmt.Sprint(positions) != fmt.Sprint(want) {
		t.Fatalf("positions = %v, want %v", positions, want)
	}
}

func TestFloatShrinksToFitItsContent(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 200px }
        .float { float: left }
        .wide { width: 150px; height: 10px }
        .narrow { width: 30px; height: 10px }
      </style>
      <div class="float"><div class="narrow"></div><div class="wide"></div></div>
      <div class="float" style="max-width: 20px"><div class="narrow"></div></div>
      <div class="float" style="min-width: 60px"><div class="narrow"></div></div>
      <div class="float" style="width: 50%"><div class="narrow"></div></div>
    `))
	_, body := pageBody(t, page)
	var widths []float64
	for _, div := range boxes.ChildrenOf(body) {
		widths = append(widths, div.Base().Width.V)
	}
	want := []float64{150, 20, 60, 100}
	if fmt.Sprint(widths) != fmt.Sprint(want) {
		t.Fatalf("widths = %v, want %v", widths, want)
	}
}

func TestFloatClearancePushesBlockBelowFloats(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        div { width: 20px; height: 10px; margin: 0 }
      </style>
      <div style="float: left; height: 50px"></div>
      <div style="clear: left; margin-top: 5px"></div>
      <div></div>
    `))
	_, body := pageBody(t, page)
	divs := expectChildren(t, body, 3)
	cleared := divs[1].Base()
	if cleared.Clearance == nil || *cleared.Clearance != 45 || cleared.PositionY != 45 {
		t.Fatalf("clearance = %v, margin box at y = %g; want 45, 45", cleared.Clearance, cleared.PositionY)
	}
	if got := divs[1].BorderBoxY(); got != 50 {
		t.Errorf("top border edge of the cleared block = %g, want 50", got)
	}
	if got := divs[2].Base().PositionY; got != 60 {
		t.Errorf("the block after the cleared one is at y = %g, want 60", got)
	}
}

func TestFloatsAtTheTopOfTheNextPageWhenTheyDoNotFit(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 50px; margin: 0 }
        div { width: 10px; margin: 0 }
      </style>
      <div style="height: 40px"></div>
      <div style="float: left; height: 20px"></div>
      <div style="height: 5px"></div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	expectChildren(t, body, 1)
	_, body = pageBody(t, pages[1])
	divs := expectChildren(t, body, 2)
	if !divs[0].IsFloated() || divs[0].Base().PositionY != 0 || divs[1].Base().PositionY != 0 {
		t.Fatalf("second page: float at y = %g, block at y = %g", divs[0].Base().PositionY, divs[1].Base().PositionY)
	}
}

func TestAdjacentFloatsSitSideBySide(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        div { float: left }
        img { width: 100px; vertical-align: top }
      </style>
      <div><img src=pattern.png /></div>
      <div><img src=pattern.png /></div>`))
	_, body := pageBody(t, page)
	divs := expectChildren(t, body, 2)
	if outerArea(divs[0]) != [4]float64{0, 0, 100, 100} || outerArea(divs[1]) != [4]float64{100, 0, 100, 100} {
		t.Fatalf("areas = %v, %v", outerArea(divs[0]), outerArea(divs[1]))
	}
}

func TestFloatsThatDoNotFitBesideEachOtherMoveDown(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 290px }
        div { float: left; width: 100px;  }
        img { width: 60px; vertical-align: top }
      </style>
      <div><img src=pattern.png /><!-- 1 --></div>
      <div><img src=pattern.png /><!-- 2 --></div>
      <div><img src=pattern.png /><!-- 4 --></div>
      <img src=pattern.png /><!-- 3
      --><img src=pattern.png /><!-- 5 -->`))
	_, body := pageBody(t, page)
	children := expectChildren(t, body, 4)
	div1, div2, div4, anon := children[0], children[1], children[2], children[3]
	lines := expectChildren(t, anon, 2)
	img3 := expectChildren(t, lines[0], 1)[0]
	img5 := expectChildren(t, lines[1], 1)[0]
	want := map[string][4]float64{"div 1": {0, 0, 100, 60}, "div 2": {100, 0, 100, 60}, "img 3": {200, 0, 60, 60},
		"div 4": {0, 60, 100, 60}, "img 5": {100, 60, 60, 60}}
	for name, box := range map[string]boxes.Box{"div 1": div1, "div 2": div2, "img 3": img3, "div 4": div4, "img 5": img5} {
		if got := outerArea(box); got != want[name] {
			t.Errorf("%s area = %v, want %v", name, got, want[name])
		}
	}
}

func TestLeftAndRightFloatsFillRowsInOrder(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style type="text/css">
        body { width: 200px }
        p { width: 70px; height: 20px }
        .left { float: left }
        .right { float: right }
      </style>
      <p class="left"> ⇦ A 1 </p>
      <p class="left"> ⇦ B 2 </p>
      <p class="left"> ⇦ A 3 </p>
      <p class="right"> B 4 ⇨ </p>
      <p class="left"> ⇦ A 5 </p>
      <p class="right"> B 6 ⇨ </p>
      <p class="right"> B 8 ⇨ </p>
      <p class="left"> ⇦ A 7 </p>
      <p class="left"> ⇦ A 9 </p>
      <p class="left"> ⇦ B 10 </p>
    `))
	_, body := pageBody(t, page)
	var positions [][2]float64
	for _, p := range boxes.ChildrenOf(body) {
		positions = append(positions, xy(p))
	}
	want := [][2]float64{
		{0, 0}, {70, 0}, {0, 20}, {130, 20}, {0, 40}, {130, 40},
		{130, 60}, {0, 60}, {0, 80}, {70, 80},
	}
	if fmt.Sprint(positions) != fmt.Sprint(want) {
		t.Fatalf("positions = %v, want %v", positions, want)
	}
}

func TestLinesAvoidFloatsThatLeaveNoRoom(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 100px }
        p { float: left; height: 100px }
        img { width: 60px; vertical-align: top }
      </style>
      <p style="width: 20px"></p>
      <p style="width: 100%"></p>
      <img src=pattern.png /><img src=pattern.png />
    `))
	_, body := pageBody(t, page)
	children := expectChildren(t, body, 3)
	anon := children[2]
	lines := expectChildren(t, anon, 2)
	if anon.Base().PositionY != 0 || xy(lines[0]) != [2]float64{20, 0} || xy(lines[1]) != [2]float64{0, 200} {
		t.Fatalf("anonymous block at y = %g, lines at %v and %v", anon.Base().PositionY, xy(lines[0]), xy(lines[1]))
	}
}

func TestTextLinesAvoidFloatsThatLeaveNoRoom(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 100px; font: 60px testfont; }
        p { float: left; height: 100px }
        img { width: 60px; vertical-align: top }
      </style>
      <p style="width: 20px"></p>
      <p style="width: 100%"></p>
      A B
    `))
	_, body := pageBody(t, page)
	anon := expectChildren(t, body, 3)[2]
	lines := expectChildren(t, anon, 2)
	if anon.Base().PositionY != 0 || xy(lines[0]) != [2]float64{20, 0} || xy(lines[1]) != [2]float64{0, 200} {
		t.Fatalf("anonymous block at y = %g, lines at %v and %v", anon.Base().PositionY, xy(lines[0]), xy(lines[1]))
	}
}

func TestFloatAfterFullLineIsPlacedOnTheNextLine(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 90px; font-size: 0 }
        img { vertical-align: top }
      </style>
      <body>
      <span>
        <img src=pattern.png style="width: 50px" />
        <img src=pattern.png style="width: 50px" />
        <img src=pattern.png style="float: left; width: 30px" />
      </span>
    `))
	_, body := pageBody(t, page)
	lines := expectChildren(t, body, 2)
	span1 := expectChildren(t, lines[0], 1)[0]
	span2 := expectChildren(t, lines[1], 1)[0]
	img1 := expectChildren(t, span1, 1)[0]
	imgs := expectChildren(t, span2, 2)
	want := [][4]float64{{0, 0, 50, 50}, {30, 50, 50, 50}, {0, 50, 30, 30}}
	for i, img := range []boxes.Box{img1, imgs[0], imgs[1]} {
		if got := outerArea(img); got != want[i] {
			t.Errorf("image %d area = %v, want %v", i+1, got, want[i])
		}
	}
}

func TestFloatAfterFullLineWithoutInlineBoxIsPlacedOnTheNextLine(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 90px; font-size: 0 }
        img { vertical-align: top }
      </style>
      <body>
      <img src=pattern.png style="width: 50px" />
      <img src=pattern.png style="width: 50px" />
      <img src=pattern.png style="float: left; width: 30px" />
    `))
	_, body := pageBody(t, page)
	lines := expectChildren(t, body, 2)
	img1 := expectChildren(t, lines[0], 1)[0]
	imgs := expectChildren(t, lines[1], 2)
	want := [][4]float64{{0, 0, 50, 50}, {30, 50, 50, 50}, {0, 50, 30, 30}}
	for i, img := range []boxes.Box{img1, imgs[0], imgs[1]} {
		if got := outerArea(img); got != want[i] {
			t.Errorf("image %d area = %v, want %v", i+1, got, want[i])
		}
	}
}

func TestFloatsDoNotAffectOtherPages(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        body { width: 90px; font-size: 0 }
        img { vertical-align: top }
      </style>
      <body>
      <img src=pattern.png style="float: left; width: 30px" />
      <img src=pattern.png style="width: 50px" />
      <div style="page-break-before: always"></div>
      <img src=pattern.png style="width: 50px" />
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	children := expectChildren(t, body, 2)
	floated, anon := children[0], children[1]
	img1 := expectChildren(t, expectChildren(t, anon, 1)[0], 1)[0]
	if outerArea(floated) != [4]float64{0, 0, 30, 30} || outerArea(img1) != [4]float64{30, 0, 50, 50} {
		t.Errorf("page 1: float %v, image %v", outerArea(floated), outerArea(img1))
	}
	_, body = pageBody(t, pages[1])
	expectChildren(t, body, 2)
}

func TestFloatOverflowSplitsFormattingContext(t *testing.T) {
	noLogs(t)
	// A float that overflows is deferred, and must not make its formatting
	// context higher on the current page, or the whole block would go to the
	// next page instead of being split.
	pages := paginate(t, `
      <style>@page { size: 100px }</style>
      <div style="height: 70px"></div>
      <div style="display: flow-root">
        <p style="height: 20px">a</p>
        <div style="float: right; width: 30px; height: 40px"></div>
        <p>b</p>
      </div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	// Page 1 keeps the spacer and the first paragraph.
	_, body := pageBody(t, pages[0])
	children := expectChildren(t, body, 2)
	spacer, flow1 := children[0], children[1]
	if spacer.Base().Height.V != 70 {
		t.Errorf("spacer height = %g, want 70", spacer.Base().Height.V)
	}
	a := expectChildren(t, flow1, 1)[0]
	if a.Base().ElementTag != "p" || a.Base().PositionY != 70 {
		t.Errorf("first paragraph = %v at y = %g, want p at 70", a, a.Base().PositionY)
	}

	// Page 2 resumes the block with the deferred float and the last paragraph.
	_, body = pageBody(t, pages[1])
	flow2 := expectChildren(t, body, 1)[0]
	items := expectChildren(t, flow2, 2)
	float, b := items[0], items[1]
	if !float.IsFloated() || float.Base().PositionY != 0 {
		t.Errorf("deferred float = %v at y = %g, want a float at 0", float, float.Base().PositionY)
	}
	if b.Base().ElementTag != "p" || b.Base().PositionY != 0 {
		t.Errorf("last paragraph = %v at y = %g, want p at 0", b, b.Base().PositionY)
	}
}

func TestPreferredWidthsOfFloatedParagraph(t *testing.T) {
	noLogs(t)
	floatWidth := func(bodyWidth int) float64 {
		// The second line ends with a no-break space between its words.
		page := onlyPage(t, paginate(t, fmt.Sprintf(`
          <body style="width: %dpx; font-family: testfont">
          <p style="white-space: pre-line; float: left">
            Lorem ipsum dolor sit amet,
              consectetur elit
          </p>
        `, bodyWidth)))
		_, body := pageBody(t, page)
		return expectChildren(t, body, 1)[0].Base().Width.V
	}
	// Preferred minimum width.
	if got, want := floatWidth(10), float64(len("consectetur elit")*16); got != want {
		t.Errorf("minimum width = %g, want %g", got, want)
	}
	// Preferred width.
	if got, want := floatWidth(1000000), float64(len("Lorem ipsum dolor sit amet,")*16); got != want {
		t.Errorf("preferred width = %g, want %g", got, want)
	}
}

func TestPreferredWidthsIgnoreCollapsedSpaceBeforeInlineBoxes(t *testing.T) {
	noLogs(t)
	// Incorrect whitespace handling in the preferred width used to cause an
	// unnecessary line break.
	page := onlyPage(t, paginate(t, `
      <p style="float: left">Lorem <em>ipsum</em> dolor.</p>
    `))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	children := expectChildren(t, paragraph, 1)
	if children[0].Kind() != boxes.KindLine {
		t.Fatalf("child = %v, want a line box", children[0])
	}
}

func TestPreferredWidthsWithBreaks(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   float64
	}{
		{"images and a break", `
      <style>img { width: 20px }</style>
      <p style="float: left">
        <img src=pattern.png><img src=pattern.png><br>
        <img src=pattern.png></p>
    `, 40},
		{"text and breaks", `<style>p { font: 20px testfont }</style><p style="float: left">XX<br>XX<br>X</p>`, 40},
		{"the space at the start of a line is collapsed", `<style>p { font: 20px testfont }</style><p style="float: left">XX<br> XX<br>X</p>`, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			page := onlyPage(t, paginate(t, tt.source))
			_, body := pageBody(t, page)
			paragraph := expectChildren(t, body, 1)[0]
			if got := paragraph.Base().Width.V; got != tt.want {
				t.Fatalf("width = %g, want %g", got, tt.want)
			}
		})
	}
}

const floatInInlineStyle = `
      <style>
        body {
          font-family: testfont;
          font-size: 20px;
        }
        p {
          width: 14em;
          text-align: justify;
        }
        span {
          float: right;
        }
      </style>`

func TestRightFloatInInlineBoxGoesToTheRightEdge(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, floatInInlineStyle+`
      <p>
        aa bb <a><span>cc</span> ddd</a> ee ff
      </p>
    `))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	lines := expectChildren(t, paragraph, 2)
	line1, line2 := lines[0], lines[1]

	parts := expectChildren(t, line1, 3)
	p1, a, p2 := parts[0].(*boxes.TextBox), parts[1], parts[2].(*boxes.TextBox)
	if p1.Width.V != 6*20 || p1.Text != "aa bb " || p1.PositionX != 0 {
		t.Errorf("first text = %q, width %g at %g", p1.Text, p1.Width.V, p1.PositionX)
	}
	if p2.Width.V != 3*20 || p2.Text != " ee" || p2.PositionX != 9*20 {
		t.Errorf("last text = %q, width %g at %g", p2.Text, p2.Width.V, p2.PositionX)
	}
	inner := expectChildren(t, a, 2)
	span, aText := inner[0], inner[1].(*boxes.TextBox)
	// The leading space collapses.
	if aText.Width.V != 3*20 || aText.Text != "ddd" || aText.PositionX != 6*20 {
		t.Errorf("a text = %q, width %g at %g", aText.Text, aText.Width.V, aText.PositionX)
	}
	cc := expectChildren(t, expectChildren(t, span, 1)[0], 1)[0].(*boxes.TextBox)
	if span.Base().Width.V != 2*20 || cc.Text != "cc" || span.Base().PositionX != 12*20 {
		t.Errorf("span = %q, width %g at %g", cc.Text, span.Base().Width.V, span.Base().PositionX)
	}
	p3 := expectChildren(t, line2, 1)[0]
	if p3.Base().Width.V != 2*20 {
		t.Errorf("last line width = %g, want 40", p3.Base().Width.V)
	}
}

func floatInInlineArticle(t *testing.T, content string) (line1, line2 boxes.Box) {
	t.Helper()
	page := onlyPage(t, paginate(t, `
      <style>
        @page {
          size: 10em;
        }
        article {
          font-family: testfont;
          line-height: 1;
        }
        div {
          float: left;
          width: 50%;
        }
      </style>
      <article>`+content+`</article>`))
	_, body := pageBody(t, page)
	article := expectChildren(t, body, 1)[0]
	lines := expectChildren(t, article, 2)
	return lines[0], lines[1]
}

func trimmedText(box boxes.Box) string {
	return strings.TrimSpace(box.(*boxes.TextBox).Text)
}

func TestFloatBeforeTextInInlineBoxTakesHalfOfTheLine(t *testing.T) {
	noLogs(t)
	line1, line2 := floatInInlineArticle(t, `
        <span>
          <div>a b c</div>
          1 2 3 4 5 6
        </span>`)
	span1 := expectChildren(t, line1, 1)[0]
	items := expectChildren(t, span1, 2)
	div, text := items[0], items[1]
	if got := trimmedText(expectChildren(t, expectChildren(t, div, 1)[0], 1)[0]); got != "a b c" {
		t.Errorf("div text = %q, want a b c", got)
	}
	if got := trimmedText(text); got != "1 2 3" {
		t.Errorf("first line text = %q, want 1 2 3", got)
	}
	text2 := expectChildren(t, expectChildren(t, line2, 1)[0], 1)[0]
	if got := trimmedText(text2); got != "4 5 6" {
		t.Errorf("second line text = %q, want 4 5 6", got)
	}
}

func TestFloatAmongTextInInlineBoxTakesHalfOfTheLine(t *testing.T) {
	noLogs(t)
	line1, line2 := floatInInlineArticle(t, `
        <span>
          1 2 3 <div>a b c</div> 4 5 6
        </span>`)
	span1 := expectChildren(t, line1, 1)[0]
	items := expectChildren(t, span1, 2)
	text, div := items[0], items[1]
	if got := trimmedText(text); got != "1 2 3" {
		t.Errorf("first line text = %q, want 1 2 3", got)
	}
	if got := trimmedText(expectChildren(t, expectChildren(t, div, 1)[0], 1)[0]); got != "a b c" {
		t.Errorf("div text = %q, want a b c", got)
	}
	text2 := expectChildren(t, expectChildren(t, line2, 1)[0], 1)[0]
	if got := trimmedText(text2); got != "4 5 6" {
		t.Errorf("second line text = %q, want 4 5 6", got)
	}
}

func TestFloatThatDoesNotFitBesideTextMovesBelowTheLine(t *testing.T) {
	noLogs(t)
	line1, line2 := floatInInlineArticle(t, `
        <span>
          1 2 3 4 <div>a b c</div> 5 6
        </span>`)
	items := expectChildren(t, line1, 2)
	span1, div := items[0], items[1]
	texts := expectChildren(t, span1, 2)
	if got := trimmedText(texts[0]); got != "1 2 3 4" {
		t.Errorf("first text = %q, want 1 2 3 4", got)
	}
	if got := trimmedText(texts[1]); got != "5" {
		t.Errorf("second text = %q, want 5", got)
	}
	if div.Base().PositionY != 16 {
		t.Errorf("div at y = %g, want 16", div.Base().PositionY)
	}
	if got := trimmedText(expectChildren(t, expectChildren(t, div, 1)[0], 1)[0]); got != "a b c" {
		t.Errorf("div text = %q, want a b c", got)
	}
	text := expectChildren(t, expectChildren(t, line2, 1)[0], 1)[0]
	if got := trimmedText(text); got != "6" {
		t.Errorf("second line text = %q, want 6", got)
	}
}

func TestFloatNextLine(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body {
          font-family: testfont;
          font-size: 20px;
        }
        p {
          text-align: justify;
          width: 13em;
        }
        span {
          float: left;
        }
      </style>
      <p>pp pp pp pp <a><span>ppppp</span> aa</a> pp pp pp pp pp</p>`))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	lines := expectChildren(t, paragraph, 3)
	expectChildren(t, lines[0], 1)
	expectChildren(t, lines[2], 1)
	items := expectChildren(t, lines[1], 2)
	a, p := items[0], items[1]
	inner := expectChildren(t, a, 2)
	span, aText := inner[0], inner[1]
	if span.Base().PositionX != 0 || span.Base().Width.V != 5*20 {
		t.Errorf("span at %g, width %g; want 0, 100", span.Base().PositionX, span.Base().Width.V)
	}
	if aText.Base().PositionX != 5*20 || a.Base().PositionX != 5*20 ||
		aText.Base().Width.V != 2*20 || a.Base().Width.V != 2*20 {
		t.Errorf("a text at %g (a at %g), width %g (a %g)", aText.Base().PositionX, a.Base().PositionX,
			aText.Base().Width.V, a.Base().Width.V)
	}
	if p.Base().PositionX != 7*20 {
		t.Errorf("text after the link at %g, want 140", p.Base().PositionX)
	}
}

func floatTextIndentStyle(side string) string {
	return fmt.Sprintf(`
      <style>
        body {
          font-family: testfont;
          font-size: 20px;
        }
        p {
          text-align: justify;
          text-indent: 1em;
          width: 14em;
        }
        span {
          float: %s;
        }
      </style>`, side)
}

func TestLeftFloatInLineWithTextIndent(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, floatTextIndentStyle("left")+`<p><a>aa <span>float</span> aa</a></p>`))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	line := expectChildren(t, paragraph, 1)[0]
	a := expectChildren(t, line, 1)[0]
	parts := expectChildren(t, a, 3)
	a1, span, a2 := parts[0], parts[1], parts[2]
	spanText := expectChildren(t, span, 1)[0]
	if span.Base().PositionX != 0 || spanText.Base().PositionX != 0 ||
		span.Base().Width.V != (1+5)*20 || spanText.Base().Width.V != (1+5)*20 {
		t.Errorf("span at %g (text %g), width %g (text %g); want 0, 120",
			span.Base().PositionX, spanText.Base().PositionX, span.Base().Width.V, spanText.Base().Width.V)
	}
	if a1.Base().Width.V != 3*20 || a1.Base().PositionX != (1+5+1)*20 {
		t.Errorf("first text at %g, width %g; want 140, 60", a1.Base().PositionX, a1.Base().Width.V)
	}
	// The leading space collapses.
	if a2.Base().Width.V != 2*20 || a2.Base().PositionX != (1+5+1+3)*20 {
		t.Errorf("second text at %g, width %g; want 200, 40", a2.Base().PositionX, a2.Base().Width.V)
	}
}

func TestLeftFloatOnSecondLineWithTextIndent(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, floatTextIndentStyle("left")+`
      <p>
        oooooooooooo
        <a>aa <span>float</span> aa</a></p>`))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	lines := expectChildren(t, paragraph, 2)
	p1 := expectChildren(t, lines[0], 1)[0]
	if p1.Base().PositionX != 1*20 || p1.Base().Width.V != 12*20 {
		t.Errorf("first line text at %g, width %g; want 20, 240", p1.Base().PositionX, p1.Base().Width.V)
	}
	a := expectChildren(t, lines[1], 1)[0]
	parts := expectChildren(t, a, 3)
	a1, span, a2 := parts[0], parts[1], parts[2]
	spanText := expectChildren(t, span, 1)[0]
	if span.Base().PositionX != 0 || spanText.Base().PositionX != 0 ||
		span.Base().Width.V != (1+5)*20 || spanText.Base().Width.V != (1+5)*20 {
		t.Errorf("span at %g (text %g), width %g (text %g); want 0, 120",
			span.Base().PositionX, spanText.Base().PositionX, span.Base().Width.V, spanText.Base().Width.V)
	}
	if a1.Base().Width.V != 3*20 || a1.Base().PositionX != (1+5)*20 {
		t.Errorf("first text at %g, width %g; want 120, 60", a1.Base().PositionX, a1.Base().Width.V)
	}
	if a2.Base().Width.V != 2*20 || a2.Base().PositionX != (1+5+3)*20 {
		t.Errorf("second text at %g, width %g; want 180, 40", a2.Base().PositionX, a2.Base().Width.V)
	}
}

func TestRightFloatOnSecondLineWithTextIndent(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, floatTextIndentStyle("right")+`
      <p>
        oooooooooooo
        <a>aa <span>float</span> aa</a>
        oooooooooooo
      </p>`))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	lines := expectChildren(t, paragraph, 3)
	p1 := expectChildren(t, lines[0], 1)[0]
	if p1.Base().PositionX != 1*20 || p1.Base().Width.V != 12*20 {
		t.Errorf("first line text at %g, width %g; want 20, 240", p1.Base().PositionX, p1.Base().Width.V)
	}
	a := expectChildren(t, lines[1], 1)[0]
	parts := expectChildren(t, a, 3)
	a1, span, a2 := parts[0], parts[1], parts[2]
	spanText := expectChildren(t, span, 1)[0]
	if span.Base().PositionX != (14-5-1)*20 || spanText.Base().PositionX != (14-5-1)*20 ||
		span.Base().Width.V != (1+5)*20 || spanText.Base().Width.V != (1+5)*20 {
		t.Errorf("span at %g (text %g), width %g (text %g); want 160, 120",
			span.Base().PositionX, spanText.Base().PositionX, span.Base().Width.V, spanText.Base().Width.V)
	}
	if a1.Base().PositionX != 0 {
		t.Errorf("first text at %g, want 0", a1.Base().PositionX)
	}
	if a2.Base().Width.V != 2*20 || a2.Base().PositionX != (14-5-1-2)*20 {
		t.Errorf("second text at %g, width %g; want 120, 40", a2.Base().PositionX, a2.Base().Width.V)
	}
	p2 := expectChildren(t, lines[2], 1)[0]
	if p2.Base().PositionX != 0 || p2.Base().Width.V != 12*20 {
		t.Errorf("last line text at %g, width %g; want 0, 240", p2.Base().PositionX, p2.Base().Width.V)
	}
}

func TestFloatPreviousBreak(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page {
          size: 13px;
        }
        body {
          font-family: testfont;
          font-size: 2px;
          line-height: 1;
        }
        p {
          break-after: avoid;
        }
        section {
          break-inside: avoid;
          float: left;
        }
      </style>
      <article>
        oooooo
        oooooo
        oooooo
        oooooo
      </article>
      <p>xxxxxx</p>
      <p>yyyyyy</p>
      <section>
        <div>aaaaaa</div>
        <div>cccccc</div>
      </section>
      <article>
        dddddd
      </article>`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	expectChildren(t, body, 1)
	_, body = pageBody(t, pages[1])
	expectChildren(t, body, 4)
}

func TestFloatFail(t *testing.T) {
	t.Skip("not supported: a float inside a line moves the content that was already placed on the line")
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body {
          font-family: testfont;
          font-size: 20px;
        }
        p {
          text-align: justify;
          width: 12em;
        }
        span {
          float: left;
          background: red;
        }
        a {
          background: yellow;
        }
      </style>
      <p>bb bb pp bb pp pb <a><span>pp pp</span> apa</a> bb bb</p>`))
	_, body := pageBody(t, page)
	paragraph := expectChildren(t, body, 1)[0]
	expectChildren(t, paragraph, 3)
}

func TestNestedRightFloat(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 100px; font: 20px testfont }
        div { float: right; width: 50px }
      </style>
      <b><i><div></div></i></b>ab c`))
	_, body := pageBody(t, page)
	lines := expectChildren(t, body, 2)
	if lines[0].Base().Width.V != 40 || lines[1].Base().Width.V != 20 {
		t.Fatalf("line widths = %g, %g; want 40, 20", lines[0].Base().Width.V, lines[1].Base().Width.V)
	}
}

func TestFirstLetterFloat(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        body { width: 100px; font: 20px testfont }
        p:first-letter { float: left }
      </style>
      <p>Lor`))
	_, body := pageBody(t, page)
	p := expectChildren(t, body, 1)[0]
	line := expectChildren(t, p, 1)[0]
	items := expectChildren(t, line, 2)
	if items[0].Base().PositionX != 0 {
		t.Fatalf("first letter at x = %g, want 0", items[0].Base().PositionX)
	}
}

func floatPageText(t *testing.T, page *boxes.PageBox, depth int) (top float64, text string) {
	t.Helper()
	_, body := pageBody(t, page)
	div := expectChildren(t, body, 1)[0]
	box := div
	for range depth {
		box = expectChildren(t, box, 1)[0]
	}
	return div.Base().PositionY, box.(*boxes.TextBox).Text
}

func TestFloatBreakAfterPage(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <div style="break-after: page; float: left">float</div>
      <div>end</div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	for i, want := range []string{"float", "end"} {
		if top, text := floatPageText(t, pages[i], 2); top != 0 || text != want {
			t.Errorf("page %d: div at y = %g with %q, want 0 with %q", i+1, top, text, want)
		}
	}
}

func TestFloatInBlockBreakAfterPage(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <div>
        <div style="break-after: page; float: left">float</div>
      </div>
      <div>end</div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if top, text := floatPageText(t, pages[0], 3); top != 0 || text != "float" {
		t.Errorf("page 1: div at y = %g with %q, want 0 with float", top, text)
	}
	if top, text := floatPageText(t, pages[1], 2); top != 0 || text != "end" {
		t.Errorf("page 2: div at y = %g with %q, want 0 with end", top, text)
	}
}

func TestFloatBreakBeforePage(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <div>start</div>
      <div style="break-before: page; float: left">float</div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	for i, want := range []string{"start", "float"} {
		if top, text := floatPageText(t, pages[i], 2); top != 0 || text != want {
			t.Errorf("page %d: div at y = %g with %q, want 0 with %q", i+1, top, text, want)
		}
	}
}

func TestFloatInDivBreakBeforePage(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <div>start</div>
      <div>
        <div style="break-before: page; float: left">float</div>
      </div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if top, text := floatPageText(t, pages[0], 2); top != 0 || text != "start" {
		t.Errorf("page 1: div at y = %g with %q, want 0 with start", top, text)
	}
	if top, text := floatPageText(t, pages[1], 3); top != 0 || text != "float" {
		t.Errorf("page 2: div at y = %g with %q, want 0 with float", top, text)
	}
}

func TestFloatBreakAfterPageNoFit(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 100px }
        html { line-height: 20px }
      </style>
      <div style="height: 90px">start</div>
      <div style="break-after: right; float: left">float</div>
      <div>end</div>
    `)
	if len(pages) != 3 {
		t.Fatalf("got %d pages, want 3", len(pages))
	}
	for i, want := range []string{"start", "float", "end"} {
		if top, text := floatPageText(t, pages[i], 2); top != 0 || text != want {
			t.Errorf("page %d: div at y = %g with %q, want 0 with %q", i+1, top, text, want)
		}
	}
}

func TestLogicalFloatsFollowTheDirection(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		want      [3]float64
	}{
		{"ltr", "ltr", [3]float64{0, 70, 30}},
		{"rtl", "rtl", [3]float64{70, 0, 40}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			page := onlyPage(t, paginate(t, `
      <style>
        body { direction: `+tt.direction+`; width: 100px }
        div { width: 30px; height: 10px }
      </style>
      <div style="float: inline-start"></div>
      <div style="float: inline-end"></div>
      <div style="float: inline-start"></div>
    `))
			_, body := pageBody(t, page)
			divs := expectChildren(t, body, 3)
			got := [3]float64{divs[0].Base().PositionX, divs[1].Base().PositionX, divs[2].Base().PositionX}
			// The third float goes next to the first, on the same side.
			if got != tt.want {
				t.Fatalf("x positions = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClearanceOfTheFirstChildMovesItBelowFloatsOutsideItsParent(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>div, p { width: 20px; margin: 0 }</style>
      <div style="float: left; height: 50px"></div>
      <section><p style="clear: left; margin-top: 10px; height: 5px"></p></section>
    `))
	_, body := pageBody(t, page)
	children := expectChildren(t, body, 2)
	section := children[1]
	p := expectChildren(t, section, 1)[0]
	if got := p.BorderBoxY(); got != 50 {
		t.Errorf("paragraph border box at y = %g, want 50", got)
	}
	if section.Base().PositionY != 0 || section.Base().Height.V != 55 {
		t.Errorf("section at y = %g with height %g, want 0 and 55", section.Base().PositionY, section.Base().Height.V)
	}
}

// A4 is 793.7007874015749px wide, and 45% of it only just overflows what the
// first 55% float leaves free; the comparison has to see that last bit.
func TestFloatsOverflowByOneUnitInTheLastPlace(t *testing.T) {
	tests := []struct {
		margin string
		wantY  float64
	}{
		{"0", 10},
		{"2mm", 0},
		{"4mm", 10},
		{"6mm", 0},
	}
	for _, tt := range tests {
		t.Run(tt.margin, func(t *testing.T) {
			page := onlyPage(t, paginate(t, `
              <style>
                @page { size: A4; margin: `+tt.margin+` }
                body { margin: 0 }
                div { float: left; height: 10px }
              </style>
              <div style="width: 55%"></div>
              <div style="width: 45%"></div>
            `))
			_, body := pageBody(t, page)
			floats := boxes.ChildrenOf(body)
			if len(floats) != 2 {
				t.Fatalf("body has %d children, want 2", len(floats))
			}
			if got := floats[1].Base().PositionY - floats[0].Base().PositionY; got != tt.wantY {
				t.Errorf("second float is %g below the first, want %g", got, tt.wantY)
			}
		})
	}
}
