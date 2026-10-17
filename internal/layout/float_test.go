package layout

import (
	"fmt"
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
