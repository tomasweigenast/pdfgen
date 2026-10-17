package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestInlineBlockSizes(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { margin: 0; size: 200px 2000px }
        body { margin: 0 }
        div { display: inline-block; }
      </style>
      <div> </div>
      <div>a</div>
      <div style="margin: 10px; height: 100px"></div>
      <div style="margin-left: 10px; margin-top: -50px;
                  padding-right: 20px;"></div>
      <div>
        Ipsum dolor sit amet,
        consectetur adipiscing elit.
        Sed sollicitudin nibh
        et turpis molestie tristique.
      </div>
      <div style="width: 100px; height: 100px;
                  padding-left: 10px; margin-right: 10px;
                  margin-top: -10px; margin-bottom: 50px"></div>
      <div style="font-size: 0">
        <div style="min-width: 10px; height: 10px"></div>
        <div style="width: 10%">
          <div style="width: 10px; height: 10px"></div>
        </div>
      </div>
      <div style="min-width: 150px">foo</div>
      <div style="max-width: 10px
        ">Supercalifragilisticexpialidocious</div>`)
	if d.html.Base().ElementTag != "html" {
		t.Fatalf("root tag = %q", d.html.Base().ElementTag)
	}
	body := d.body(t)
	if body.Base().ElementTag != "body" || body.Base().Width.V != 200 {
		t.Fatalf("body = %s, width %g", body.Base().ElementTag, body.Base().Width.V)
	}

	lines := kids(t, body, 4)

	// First line: white space between divs ends up preserved in text boxes.
	line1 := kids(t, lines[0], 8)
	div1, div2, div3, div4 := line1[0], line1[2], line1[4], line1[6]

	// First div, one ignored space collapsing with the next space.
	if div1.Base().ElementTag != "div" || div1.Base().Width.V != 0 {
		t.Errorf("div 1: %s of width %g", div1.Base().ElementTag, div1.Base().Width.V)
	}
	// Second div, one letter.
	if w := div2.Base().Width.V; div2.Base().ElementTag != "div" || !(0 < w && w < 20) {
		t.Errorf("div 2: %s of width %g, want between 0 and 20", div2.Base().ElementTag, w)
	}
	// Third div, empty with margin.
	if div3.Base().Width.V != 0 || div3.MarginWidth() != 20 || div3.Base().Height.V != 100 {
		t.Errorf("div 3: width %g, margin width %g, height %g", div3.Base().Width.V, div3.MarginWidth(), div3.Base().Height.V)
	}
	// Fourth div, empty with margin and padding.
	if div4.Base().Width.V != 0 || div4.MarginWidth() != 30 {
		t.Errorf("div 4: width %g, margin width %g", div4.Base().Width.V, div4.MarginWidth())
	}

	// Second line: fifth div, long text, full-width.
	line2 := kids(t, lines[1], 2)
	div5 := line2[0]
	if len(boxes.ChildrenOf(div5)) <= 1 || div5.Base().Width.V != 200 {
		t.Errorf("div 5: %d children, width %g", len(boxes.ChildrenOf(div5)), div5.Base().Width.V)
	}

	// Third line.
	line3 := kids(t, lines[2], 4)
	div6, div7 := line3[0], line3[2]
	// Sixth div, empty with a fixed width and height.
	if div6.Base().Width.V != 100 || div6.MarginWidth() != 120 ||
		div6.Base().Height.V != 100 || div6.MarginHeight() != 140 {
		t.Errorf("div 6: width %g, margin width %g, height %g, margin height %g",
			div6.Base().Width.V, div6.MarginWidth(), div6.Base().Height.V, div6.MarginHeight())
	}
	// Seventh div. Spaces have font-size: 0, they get removed.
	if div7.Base().Width.V != 20 {
		t.Errorf("div 7 width = %g, want 20", div7.Base().Width.V)
	}
	childLine := kids(t, div7, 1)[0]
	childDivs := kids(t, childLine, 2)
	if childDivs[0].Base().Width.V != 10 || childDivs[1].Base().Width.V != 2 {
		t.Errorf("child widths = %g, %g, want 10, 2", childDivs[0].Base().Width.V, childDivs[1].Base().Width.V)
	}
	if grandchild := kids(t, childDivs[1], 1)[0]; grandchild.Base().Width.V != 10 {
		t.Errorf("grandchild width = %g, want 10", grandchild.Base().Width.V)
	}

	line4 := kids(t, lines[3], 3)
	if line4[0].Base().Width.V != 150 || line4[2].Base().Width.V != 10 {
		t.Errorf("last widths = %g, %g, want 150, 10", line4[0].Base().Width.V, line4[2].Base().Width.V)
	}
}

func TestInlineBlockWithMargin(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 100px }
        span { font-family: testfont; display: inline-block; margin: 0 30px }
      </style>
      <span>a b c d e f g h i j k l</span>`)
	span := at(t, d.body(t), 0, 0)
	if w := span.Base().Width.V; w != 40 { // 100 - 2 * 30
		t.Errorf("span width = %g, want 40", w)
	}
}
