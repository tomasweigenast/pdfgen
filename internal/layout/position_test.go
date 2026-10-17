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

func TestAbsolutePositioning1(t *testing.T) {
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

func TestAbsolutePositioning2(t *testing.T) {
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

func TestAbsolutePositioning6(t *testing.T) {
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

func TestAbsolutePositioning8(t *testing.T) {
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

func TestFixedPositioningRegression2(t *testing.T) {
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
