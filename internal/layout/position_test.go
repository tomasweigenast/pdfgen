package layout

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

// xy returns the position of a box.
func xy(box boxes.Box) [2]float64 {
	return [2]float64{box.Base().PositionX, box.Base().PositionY}
}

// sizeOf returns the width and the height of a box.
func sizeOf(box boxes.Box) [2]float64 {
	return [2]float64{box.Base().Width.V, box.Base().Height.V}
}

func tags(boxes []boxes.Box) []string {
	var out []string
	for _, box := range boxes {
		out = append(out, box.Base().ElementTag)
	}
	return out
}

func TestAbsoluteBoxesTakeTheStaticPositionUnlessOffsetsAreGiven(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <div style="margin: 3px">
        <div style="height: 20px; width: 20px; position: absolute"></div>
        <div style="height: 20px; width: 20px; position: absolute;
                    left: 0"></div>
        <div style="height: 20px; width: 20px; position: absolute;
                    top: 0"></div>
      </div>
    `))
	_, body := pageBody(t, page)
	div1 := expectChildren(t, body, 1)[0]
	children := expectChildren(t, div1, 3)
	if div1.Base().Height.V != 0 || xy(div1) != [2]float64{0, 0} {
		t.Errorf("div 1 height %g at %v, want 0 at (0, 0)", div1.Base().Height.V, xy(div1))
	}
	want := [][2]float64{{3, 3}, {0, 3}, {3, 0}}
	for i, div := range children {
		if sizeOf(div) != [2]float64{20, 20} || xy(div) != want[i] {
			t.Errorf("div %d is %v at %v, want 20x20 at %v", i+2, sizeOf(div), xy(div), want[i])
		}
	}
}

func TestAbsoluteBoxInRelativeBlockStartsAtItsOrigin(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <div style="position: relative; width: 20px">
        <div style="height: 20px; width: 20px; position: absolute"></div>
        <div style="height: 20px; width: 20px"></div>
      </div>
    `))
	_, body := pageBody(t, page)
	div1 := expectChildren(t, body, 1)[0]
	for _, div := range append([]boxes.Box{div1}, expectChildren(t, div1, 2)...) {
		if xy(div) != [2]float64{0, 0} || sizeOf(div) != [2]float64{20, 20} {
			t.Errorf("%v is %v at %v, want 20x20 at (0, 0)", div, sizeOf(div), xy(div))
		}
	}
}

func TestAbsoluteOffsetsAreRelativeToThePaddingBox(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <div style="position: relative; width: 20px; height: 60px;
                  border: 10px solid; padding-top: 6px; top: 5px; left: 1px">
        <div style="height: 20px; width: 20px; position: absolute;
                    bottom: 50%"></div>
        <div style="height: 20px; width: 20px; position: absolute;
                    top: 13px"></div>
      </div>
    `))
	_, body := pageBody(t, page)
	div1 := expectChildren(t, body, 1)[0]
	children := expectChildren(t, div1, 2)
	if xy(div1) != [2]float64{1, 5} || sizeOf(div1) != [2]float64{20, 60} ||
		div1.BorderWidth() != 40 || div1.BorderHeight() != 86 {
		t.Errorf("div 1 at %v, %v, border %g x %g", xy(div1), sizeOf(div1), div1.BorderWidth(), div1.BorderHeight())
	}
	for i, div := range children {
		if xy(div) != [2]float64{11, 28} || sizeOf(div) != [2]float64{20, 20} {
			t.Errorf("div %d at %v, %v; want (11, 28), 20x20", i+2, xy(div), sizeOf(div))
		}
	}
}

func TestAbsoluteBoxWithAutoMarginsIsCentered(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>@page{ width: 50px; height: 50px }</style>
      <body style="font-size: 0">
        <div style="position: absolute; margin: auto;
                    left: 0; right: 10px;
                    top: 0; bottom: 10px;
                    width: 10px; height: 20px">
    `))
	_, body := pageBody(t, page)
	div := expectChildren(t, body, 1)[0]
	if got := [2]float64{div.ContentBoxX(), div.ContentBoxY()}; got != [2]float64{15, 10} {
		t.Errorf("content box at %v, want (15, 10)", got)
	}
	if sizeOf(div) != [2]float64{10, 20} {
		t.Errorf("size = %v, want 10x20", sizeOf(div))
	}
}

func TestAbsoluteImages(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page { size: 50px; }
        img { display: block; position: absolute }
      </style>
      <div style="margin: 10px">
        <img src=pattern.png />
        <img src=pattern.png style="left: 15px" />
        <img src=pattern.png style="top: 15px" />
        <img src=pattern.png style="bottom: 25px" />
      </div>
    `))
	_, body := pageBody(t, page)
	div := expectChildren(t, body, 1)[0]
	images := expectChildren(t, div, 4)
	if div.Base().Height.V != 0 || xy(div) != [2]float64{0, 0} {
		t.Errorf("div height %g at %v, want 0 at (0, 0)", div.Base().Height.V, xy(div))
	}
	want := [][2]float64{{10, 10}, {15, 10}, {10, 15}, {10, 21}}
	for i, img := range images {
		if xy(img) != want[i] || sizeOf(img) != [2]float64{4, 4} {
			t.Errorf("image %d is %v at %v, want 4x4 at %v", i+1, sizeOf(img), xy(img), want[i])
		}
	}
}

func TestFixedBoxesNestedInFixedBoxesRepeatWithThem(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 100px 100px }
        section { break-after: page }
        .fixed { position: fixed; top: 10px; left: 15px; width: 20px }
      </style>
      <div class="fixed">
        <article class="fixed" style="top: 20px">
          <header class="fixed" style="left: 5px"></header>
        </article>
      </div>
      <section></section>
      <pre></pre>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	check := func(div boxes.Box) {
		t.Helper()
		if xy(div) != [2]float64{15, 10} {
			t.Errorf("div at %v, want (15, 10)", xy(div))
		}
		article := expectChildren(t, div, 1)[0]
		if xy(article) != [2]float64{15, 20} {
			t.Errorf("article at %v, want (15, 20)", xy(article))
		}
		header := expectChildren(t, article, 1)[0]
		if xy(header) != [2]float64{5, 10} {
			t.Errorf("header at %v, want (5, 10)", xy(header))
		}
	}

	html := expectChildren(t, pages[0], 1)[0]
	body := expectChildren(t, html, 1)[0]
	if tag := body.Base().ElementTag; tag != "body" {
		t.Fatalf("child of html is %q, want body", tag)
	}
	children := expectChildren(t, body, 2)
	if tags := tags(children); fmt.Sprint(tags) != "[div section]" {
		t.Errorf("children of body = %v, want [div section]", tags)
	}
	check(children[0])

	html = expectChildren(t, pages[1], 1)[0]
	children = expectChildren(t, html, 2)
	check(children[0])
	if tag := children[1].Base().ElementTag; tag != "body" {
		t.Errorf("second child of html is %q, want body", tag)
	}
}

func TestRelativeBlocksMoveWithoutMovingTheFlow(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        p { height: 20px }
      </style>
      <p>1</p>
      <div style="position: relative; top: 10px">
        <p>2</p>
        <p style="position: relative; top: -5px; left: 5px">3</p>
        <p>4</p>
        <p style="position: relative; bottom: 5px; right: 5px">5</p>
        <p style="position: relative">6</p>
        <p>7</p>
      </div>
      <p>8</p>
    `))
	_, body := pageBody(t, page)
	top := expectChildren(t, body, 3)
	p1, div, p8 := top[0], top[1], top[2]
	inner := expectChildren(t, div, 6)
	checks := []struct {
		name string
		box  boxes.Box
		want [2]float64
	}{
		{"p1", p1, [2]float64{0, 0}},
		{"div", div, [2]float64{0, 30}},
		{"p2", inner[0], [2]float64{0, 30}},
		{"p3", inner[1], [2]float64{5, 45}}, // (0 + 5, 50 - 5)
		{"p4", inner[2], [2]float64{0, 70}},
		{"p5", inner[3], [2]float64{-5, 85}}, // (0 - 5, 90 - 5)
		{"p6", inner[4], [2]float64{0, 110}},
		{"p7", inner[5], [2]float64{0, 130}},
		{"p8", p8, [2]float64{0, 140}},
	}
	for _, c := range checks {
		if got := xy(c.box); got != c.want {
			t.Errorf("%s at %v, want %v", c.name, got, c.want)
		}
	}
	if got := div.Base().Height.V; got != 120 {
		t.Errorf("div height = %g, want 120", got)
	}
}

func TestRelativeInlineBoxesMoveWithoutMovingTheFlow(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        img { width: 20px }
        body { font-size: 0 } /* Remove spaces */
      </style>
      <body>
      <span><img src=pattern.png></span>
      <span style="position: relative; left: 10px">
        <img src=pattern.png>
        <img src=pattern.png
             style="position: relative; left: -5px; top: 5px">
        <img src=pattern.png>
        <img src=pattern.png
             style="position: relative; right: 5px; bottom: 5px">
        <img src=pattern.png style="position: relative">
        <img src=pattern.png>
      </span>
      <span><img src=pattern.png></span>
    `))
	_, body := pageBody(t, page)
	line := expectChildren(t, body, 1)[0]
	spans := expectChildren(t, line, 3)
	img1 := expectChildren(t, spans[0], 1)[0]
	imgs := expectChildren(t, spans[1], 6)
	img8 := expectChildren(t, spans[2], 1)[0]
	// The position of the span is not tested, as it depends on fonts.
	if got := xy(img1); got != [2]float64{0, 0} {
		t.Errorf("img1 at %v, want (0, 0)", got)
	}
	if got := spans[1].Base().PositionX; got != 30 {
		t.Errorf("span 2 at x = %g, want 30", got)
	}
	want := [][2]float64{{30, 0}, {45, 5}, {70, 0}, {85, -5}, {110, 0}, {130, 0}}
	for i, img := range imgs {
		if got := xy(img); got != want[i] {
			t.Errorf("img%d at %v, want %v", i+2, got, want[i])
		}
	}
	if got := xy(img8); got != [2]float64{140, 0} {
		t.Errorf("img8 at %v, want (140, 0)", got)
	}
	if got := spans[1].Base().Width.V; got != 120 {
		t.Errorf("span 2 width = %g, want 120", got)
	}
}

func TestRelativeInlineBoxPrefersLeftOverRightInLTR(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        img { width: 20px }
        body { font-size: 0 } /* Remove spaces */
      </style>
      <body>
      <span><img src=pattern.png></span>
      <span style="position: relative; left: 10px; right: 5px
        "><img src=pattern.png></span>
      <span><img src=pattern.png></span>
    `))
	_, body := pageBody(t, page)
	line := expectChildren(t, body, 1)[0]
	spans := expectChildren(t, line, 3)
	if got := spans[1].Base().PositionX; got != 20+10 {
		t.Fatalf("span 2 at x = %g, want 30", got)
	}
}

func TestRelativeInlineBoxPrefersRightOverLeftInRTL(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        img { width: 20px }
        body { direction: rtl; width: 100px;
               font-size: 0 } /* Remove spaces */
      </style>
      <body>
      <span><img src=pattern.png></span>
      <span style="position: relative; left: 10px; right: 5px
        "><img src=pattern.png></span>
      <span><img src=pattern.png></span>
    `))
	_, body := pageBody(t, page)
	line := expectChildren(t, body, 1)[0]
	spans := expectChildren(t, line, 3)
	if got := spans[1].Base().PositionX; got != 100-20-5-20 {
		t.Fatalf("span 2 at x = %g, want 55", got)
	}
}

func TestAbsoluteSpansInRelativeSpanStartAtItsLeft(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <body style="font-size: 0">
        <img src=pattern.png>
        <span style="position: relative">
          <span style="position: absolute">2</span>
          <span style="position: absolute">3</span>
          <span>4</span>
        </span>
    `))
	_, body := pageBody(t, page)
	line := expectChildren(t, body, 1)[0]
	items := expectChildren(t, line, 2)
	span1 := items[1]
	spans := expectChildren(t, span1, 3)
	if span1.Base().PositionX != 4 {
		t.Errorf("span 1 at x = %g, want 4", span1.Base().PositionX)
	}
	if xy(spans[0]) != [2]float64{4, 0} || xy(spans[1]) != [2]float64{4, 0} {
		t.Errorf("absolute spans at %v and %v, want (4, 0)", xy(spans[0]), xy(spans[1]))
	}
	if spans[2].Base().PositionX != 4 {
		t.Errorf("span 4 at x = %g, want 4", spans[2].Base().PositionX)
	}
}

func TestAbsoluteInlineBoxBetweenImagesTakesTheStaticPosition(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style> img { width: 5px; height: 20px} </style>
      <body style="font-size: 0">
        <img src=pattern.png>
        <span style="position: absolute">2</span>
        <img src=pattern.png>
    `))
	_, body := pageBody(t, page)
	line := expectChildren(t, body, 1)[0]
	items := expectChildren(t, line, 3)
	want := [][2]float64{{0, 0}, {5, 0}, {5, 0}}
	for i, box := range items {
		if got := xy(box); got != want[i] {
			t.Errorf("box %d at %v, want %v", i+1, got, want[i])
		}
	}
}

func TestAbsoluteBlockInsideLineIsPlacedBelowTheLine(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style> img { width: 5px; height: 20px} </style>
      <body style="font-size: 0">
        <img src=pattern.png>
        <span style="position: absolute; display: block">2</span>
        <img src=pattern.png>
    `))
	_, body := pageBody(t, page)
	line := expectChildren(t, body, 1)[0]
	items := expectChildren(t, line, 3)
	want := [][2]float64{{0, 0}, {0, 20}, {5, 0}}
	for i, box := range items {
		if got := xy(box); got != want[i] {
			t.Errorf("box %d at %v, want %v", i+1, got, want[i])
		}
	}
}

func TestAbsoluteBoxesInNestedContainingBlocks(t *testing.T) {
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        @page { size: 1000px 2000px }
        html { font-size: 0 }
        p { height: 20px }
      </style>
      <p>1</p>
      <div style="width: 100px">
        <p>2</p>
        <p style="position: absolute; top: -5px; left: 5px">3</p>
        <p style="margin: 3px">4</p>
        <p style="position: absolute; bottom: 5px; right: 15px;
                  width: 50px; height: 10%;
                  padding: 3px; margin: 7px">5
          <span>
            <img src="pattern.png">
            <span style="position: absolute"></span>
            <span style="position: absolute; top: -10px; right: 5px;
                         width: 20px; height: 15px"></span>
          </span>
        </p>
        <p style="margin-top: 8px">6</p>
      </div>
      <p>7</p>
    `))
	_, body := pageBody(t, page)
	top := expectChildren(t, body, 3)
	p1, div, p7 := top[0], top[1], top[2]
	inner := expectChildren(t, div, 5)
	p2, p3, p4, p5, p6 := inner[0], inner[1], inner[2], inner[3], inner[4]
	line := expectChildren(t, p5, 1)[0]
	span1 := expectChildren(t, line, 1)[0]
	parts := expectChildren(t, span1, 3)
	img, span2, span3 := parts[0], parts[1], parts[2]

	// p5 x = page width - right - margin/padding/border - width
	//      = 1000 - 15 - 2 * 10 - 50 = 915
	// p5 y = page height - bottom - margin/padding/border - height
	//      = 2000 - 5 - 2 * 10 - 200 = 1775
	// span3 x = p5 right - p5 margin - span width - span right
	//         = 985 - 7 - 20 - 5 = 953
	// span3 y = p5 y + p5 margin top + span top = 1775 + 7 - 10 = 1772
	// p6 y = p4 y + p4 margin height - margin collapsing = 40 + 26 - 3 = 63
	checks := []struct {
		name string
		box  boxes.Box
		want [2]float64
	}{
		{"p1", p1, [2]float64{0, 0}},
		{"div", div, [2]float64{0, 20}},
		{"p2", p2, [2]float64{0, 20}},
		{"p3", p3, [2]float64{5, -5}},
		{"p4", p4, [2]float64{0, 40}},
		{"p5", p5, [2]float64{915, 1775}},
		{"img", img, [2]float64{925, 1785}},
		{"span2", span2, [2]float64{929, 1785}},
		{"span3", span3, [2]float64{953, 1772}},
		{"p6", p6, [2]float64{0, 63}},
		{"p7", p7, [2]float64{0, 91}},
	}
	for _, c := range checks {
		if got := xy(c.box); got != c.want {
			t.Errorf("%s at %v, want %v", c.name, got, c.want)
		}
	}
	if got := div.Base().Height.V; got != 71 { // 20*3 + 2*3 + 8 - 3
		t.Errorf("div height = %g, want 71", got)
	}
}

func TestAbsoluteInAbsoluteNextPage(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page { size: 3px 9px }
        body { font: 2px/1 testfont }
        article { height: 8px }
        div { position: absolute }
      </style>
      <article>a</article>
      <article style="position: relative"><div><div>b<br>c
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	article := expectChildren(t, body, 1)[0]
	line := expectChildren(t, article, 1)[0]
	if got := lineText(t, line); got != "a" {
		t.Errorf("page 1 text = %q, want a", got)
	}

	_, body = pageBody(t, pages[1])
	article = expectChildren(t, body, 1)[0]
	div1 := expectChildren(t, article, 1)[0]
	div2 := expectChildren(t, div1, 1)[0]
	lines := expectChildren(t, div2, 2)
	if a, b := textOfFirst(lines[0]), textOfFirst(lines[1]); a != "b" || b != "c" {
		t.Errorf("page 2 lines = %q, %q; want b, c", a, b)
	}
}

// textOfFirst returns the text of the first child of a line box.
func textOfFirst(line boxes.Box) string {
	if text, ok := boxes.ChildrenOf(unwrapPlaceholder(line))[0].(*boxes.TextBox); ok {
		return text.Text
	}
	return ""
}

func TestFixedBoxIsShownOnThePagesBeforeAndAfterIt(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      a
      <div style="page-break-before: always; page-break-after: always">
        <p style="position: fixed">b</p>
      </div>
      c
    `)
	if len(pages) != 3 {
		t.Fatalf("got %d pages, want 3", len(pages))
	}
	html := expectChildren(t, pages[0], 1)[0]
	if got := tags(boxes.ChildrenOf(html)); fmt.Sprint(got) != "[body p]" {
		t.Errorf("page 1 children of html = %v, want [body p]", got)
	}
	_, body := pageBody(t, pages[1])
	div := expectChildren(t, body, 1)[0]
	if got := tags(expectChildren(t, div, 1)); fmt.Sprint(got) != "[p]" {
		t.Errorf("page 2 children of div = %v, want [p]", got)
	}
	html = expectChildren(t, pages[2], 1)[0]
	if got := tags(boxes.ChildrenOf(html)); fmt.Sprint(got) != "[p body]" {
		t.Errorf("page 3 children of html = %v, want [p body]", got)
	}
}

func TestFixedBoxesRepeatOnEveryPage(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page:first { size: 100px 200px }
        @page { size: 200px 100px; margin: 0 }
        article { break-after: page }
        .fixed { position: fixed; top: 10px; width: 20px }
      </style>
      <ul class="fixed" style="right: 0"><li>a</li></ul>
      <img class="fixed" style="right: 20px" src="pattern.png" />
      <div class="fixed" style="right: 40px">b</div>
      <article>page1</article>
      <article>page2</article>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}

	_, body := pageBody(t, pages[0])
	items := expectChildren(t, body, 4)
	ul, img, div, article := items[0], items[1], items[2], items[3]
	marker := expectChildren(t, ul, 1)[0]
	want := map[string][2]float64{"ul": {80, 10}, "img": {60, 10}, "div": {40, 10}, "article": {0, 0}}
	for name, box := range map[string]boxes.Box{"ul": ul, "img": img, "div": div, "article": article} {
		if got := xy(box); got != want[name] {
			t.Errorf("page 1 %s at %v, want %v", name, got, want[name])
		}
	}
	if marker.Base().PositionX != ul.Base().PositionX {
		t.Errorf("page 1 marker at x = %g, want %g", marker.Base().PositionX, ul.Base().PositionX)
	}

	html := expectChildren(t, pages[1], 1)[0]
	items = expectChildren(t, html, 4)
	ul, img, div = items[0], items[1], items[2]
	marker = expectChildren(t, ul, 1)[0]
	want = map[string][2]float64{"ul": {180, 10}, "img": {160, 10}, "div": {140, 10}}
	for name, box := range map[string]boxes.Box{"ul": ul, "img": img, "div": div} {
		if got := xy(box); got != want[name] {
			t.Errorf("page 2 %s at %v, want %v", name, got, want[name])
		}
	}
	if marker.Base().PositionX != ul.Base().PositionX {
		t.Errorf("page 2 marker at x = %g, want %g", marker.Base().PositionX, ul.Base().PositionX)
	}
	// The article of the first page keeps its position.
	if got := xy(article); got != [2]float64{0, 0} {
		t.Errorf("article at %v, want (0, 0)", got)
	}
}
