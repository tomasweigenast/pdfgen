package layout

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestBlocksFillPagesInOrder(t *testing.T) {
	for _, markup := range []string{"<div>1</div>", "<div></div>", "<img src=pattern.png>"} {
		t.Run(markup, func(t *testing.T) {
			noLogs(t)
			var content string
			for range 5 {
				content += markup
			}
			pages := paginate(t, `
      <style>
        @page { size: 100px; margin: 10px }
        body { margin: 0 }
        div { height: 30px; font-size: 20px }
        img { height: 30px; display: block }
      </style>`+content)
			var got [][]float64
			for _, page := range pages {
				_, body := pageBody(t, page)
				var tops []float64
				for _, child := range boxes.ChildrenOf(body) {
					if child.Base().PositionX != 10 {
						t.Errorf("%v at x = %g, want 10", child, child.Base().PositionX)
					}
					tops = append(tops, child.Base().PositionY)
				}
				got = append(got, tops)
			}
			if fmt.Sprint(got) != "[[10 40] [10 40] [10]]" {
				t.Fatalf("tops = %v, want [[10 40] [10 40] [10]]", got)
			}
		})
	}
}

func TestBoxFilledToTheEndOfThePageStillFits(t *testing.T) {
	noLogs(t)
	// A block that is filled to the end of a page when it breaks must not
	// be forced onto the next page because rounding made it slightly
	// taller than what fits, when it would have fitted without the filling.
	pages := paginate(t, `
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
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	children := expectChildren(t, body, 3)
	expectChildren(t, children[2], 2)
	_, body = pageBody(t, pages[1])
	expectChildren(t, expectChildren(t, body, 1)[0], 2)
}

func TestBreakWithoutAvoidFollowsTheLastFittingBlock(t *testing.T) {
	noLogs(t)
	// Without any avoid, the page breaks after the div.
	pages := paginate(t, `
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
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	_, body := pageBody(t, pages[0])
	children := expectChildren(t, body, 2)
	first, div := children[0], children[1]
	if first.Base().PositionY != 0 || first.Base().Height.V != 25 {
		t.Errorf("first block at y = %g, height %g; want 0, 25", first.Base().PositionY, first.Base().Height.V)
	}
	if div.Base().PositionY != 25 || div.Base().Height.V != 100 {
		t.Errorf("div at y = %g, height %g; want 25, 100", div.Base().PositionY, div.Base().Height.V)
	}
	_, body = pageBody(t, pages[1])
	last := expectChildren(t, body, 1)[0]
	if last.Base().PositionY != 0 || last.Base().Height.V != 25 {
		t.Errorf("last block at y = %g, height %g; want 0, 25", last.Base().PositionY, last.Base().Height.V)
	}
}

func TestAvoidedBreaksLeaveOnlyTheBreakBeforeTheDiv(t *testing.T) {
	tests := []struct {
		name   string
		markup string
		nested bool
	}{
		{"div", `
      <img src=pattern.png><!-- page break here -->
      <div>
        <p style="page-break-inside: avoid">
          <img src=pattern.png><br/><img src=pattern.png></p>
        <p style="page-break-before: avoid; page-break-after: avoid; widows: 2"
          ><img src=pattern.png><br/><img src=pattern.png></p>
      </div>
      <img src=pattern.png>
    `, false},
		{"nested div", `
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
    `, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			pages := paginate(t, `
      <style>
        @page { size: 140px; margin: 0 }
        img { height: 25px; vertical-align: top }
      </style>`+tt.markup)
			if len(pages) != 2 {
				t.Fatalf("got %d pages, want 2", len(pages))
			}
			_, body := pageBody(t, pages[0])
			first := expectChildren(t, body, 1)[0]
			if first.Base().PositionY != 0 || first.Base().Height.V != 25 {
				t.Errorf("first block at y = %g, height %g; want 0, 25", first.Base().PositionY, first.Base().Height.V)
			}

			_, body = pageBody(t, pages[1])
			container := body
			if tt.nested {
				container = expectChildren(t, body, 1)[0]
			}
			children := expectChildren(t, container, 2)
			div, last := children[0], children[1]
			if div.Base().PositionY != 0 || div.Base().Height.V != 100 {
				t.Errorf("div at y = %g, height %g; want 0, 100", div.Base().PositionY, div.Base().Height.V)
			}
			if last.Base().PositionY != 100 || last.Base().Height.V != 25 {
				t.Errorf("last block at y = %g, height %g; want 100, 25", last.Base().PositionY, last.Base().Height.V)
			}
		})
	}
}

func TestBreaksBetweenBlocksAtDifferentDepths(t *testing.T) {
	noLogs(t)
	style := `
      <style>
        @page { size: 100px; margin: 0 }
        img { height: 30px; display: block; }
      </style>`
	t.Run("without avoid", func(t *testing.T) {
		pages := paginate(t, style+`
      <div>
        <img src=pattern.png style="page-break-after: always">
        <section>
          <img src=pattern.png>
          <img src=pattern.png>
        </section>
      </div>
      <img src=pattern.png><!-- page break here -->
      <img src=pattern.png>
    `)
		if len(pages) != 3 {
			t.Fatalf("got %d pages, want 3", len(pages))
		}
		_, body := pageBody(t, pages[0])
		if got := expectChildren(t, body, 1)[0].Base().Height.V; got != 100 {
			t.Errorf("page 1 div height = %g, want 100", got)
		}
		_, body = pageBody(t, pages[1])
		children := expectChildren(t, body, 2)
		if children[0].Base().Height.V != 60 || children[1].Base().Height.V != 30 {
			t.Errorf("page 2 heights = %g, %g; want 60, 30", children[0].Base().Height.V, children[1].Base().Height.V)
		}
		_, body = pageBody(t, pages[2])
		if got := expectChildren(t, body, 1)[0].Base().Height.V; got != 30 {
			t.Errorf("page 3 image height = %g, want 30", got)
		}
	})

	t.Run("with avoid", func(t *testing.T) {
		pages := paginate(t, style+`
      <div>
        <img src=pattern.png style="page-break-after: always">
        <section>
          <img src=pattern.png><!-- page break here -->
          <img src=pattern.png style="page-break-after: avoid">
        </section>
      </div>
      <img src=pattern.png style="page-break-after: avoid">
      <img src=pattern.png>
    `)
		if len(pages) != 3 {
			t.Fatalf("got %d pages, want 3", len(pages))
		}
		_, body := pageBody(t, pages[0])
		if got := expectChildren(t, body, 1)[0].Base().Height.V; got != 100 {
			t.Errorf("page 1 div height = %g, want 100", got)
		}
		_, body = pageBody(t, pages[1])
		section := expectChildren(t, expectChildren(t, body, 1)[0], 1)[0]
		if got := expectChildren(t, section, 1)[0].Base().Height.V; got != 30 {
			t.Errorf("page 2 image height = %g, want 30", got)
		}
		// The used height of a block is not increased to fill the blank
		// space at the end of the page when children are removed from it
		// because of break-*: avoid.
		_, body = pageBody(t, pages[2])
		for i, child := range expectChildren(t, body, 3) {
			if got := child.Base().Height.V; got != 30 {
				t.Errorf("page 3 child %d height = %g, want 30", i+1, got)
			}
		}
	})
}

func TestBreakAfterAvoidWithOutOfFlowChildren(t *testing.T) {
	noLogs(t)
	pages := paginate(t, `
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
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	html := expectChildren(t, pages[0], 1)[0]
	items := expectChildren(t, html, 2)
	children := expectChildren(t, items[0], 2)
	div1, section := children[0], children[1]
	div2 := expectChildren(t, section, 1)[0]
	if div1.Base().PositionY != 0 || div2.Base().PositionY != 20 ||
		div1.Base().Height.V != 20 || div2.Base().Height.V != 20 {
		t.Errorf("page 1: divs at y = %g, %g with heights %g, %g", div1.Base().PositionY, div2.Base().PositionY,
			div1.Base().Height.V, div2.Base().Height.V)
	}

	_, body := pageBody(t, pages[1])
	children = expectChildren(t, body, 2)
	section, div4 := children[0], children[1]
	div3 := expectChildren(t, section, 1)[0]
	expectChildren(t, div3, 2)
	if div3.Base().PositionY != 0 || div4.Base().PositionY != 20 ||
		div3.Base().Height.V != 20 || div4.Base().Height.V != 20 {
		t.Errorf("page 2: divs at y = %g, %g with heights %g, %g", div3.Base().PositionY, div4.Base().PositionY,
			div3.Base().Height.V, div4.Base().Height.V)
	}
}

func TestMarginBreakAtUnforcedAndForcedBreaks(t *testing.T) {
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
			noLogs(t)
			pages := paginate(t, fmt.Sprintf(`
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
    `, tt.breakAfter, tt.marginBreak))
			if len(pages) != 2 {
				t.Fatalf("got %d pages, want 2", len(pages))
			}
			_, body := pageBody(t, pages[0])
			div := expectChildren(t, expectChildren(t, body, 1)[0], 1)[0]
			if got := div.Base().MarginTop.V; got != 5 {
				t.Errorf("page 1 margin-top = %g, want 5", got)
			}
			_, body = pageBody(t, pages[1])
			div = expectChildren(t, expectChildren(t, body, 1)[0], 1)[0]
			if got := div.Base().MarginTop.V; got != tt.marginTop {
				t.Errorf("page 2 margin-top = %g, want %g", got, tt.marginTop)
			}
		})
	}
}

func TestOrphansAndWidowsWithBreakInsideAvoid(t *testing.T) {
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
			noLogs(t)
			pages := paginate(t, fmt.Sprintf(`
      <style>
        @page { size: 200px }
        h1 { height: 120px }
        p { line-height: 20px;
            width: 1px; /* line break at each word */
            %s }
      </style>
      <h1>Tasty test</h1>
      <!-- There is room for 4 lines after h1 on the fist page -->
      <p>one two three four five six seven</p>
    `, tt.style))
			var got []int
			for i, page := range pages {
				_, body := pageBody(t, page)
				children := boxes.ChildrenOf(body)
				if i == 0 {
					children = children[1:] // skip h1
				}
				count := 0
				if len(children) != 0 {
					count = len(boxes.ChildrenOf(children[0]))
				}
				got = append(got, count)
			}
			if fmt.Sprint(got) != fmt.Sprint(tt.lineCounts) {
				t.Fatalf("lines per page = %v, want %v", got, tt.lineCounts)
			}
		})
	}
}

func TestPageAndLineBoxBreaking(t *testing.T) {
	noLogs(t)
	// The empty span checks a corner case when the leading white space of a
	// line is skipped.
	pages := paginate(t, `
      <style>
        @page { size: 100px; margin: 2px; border: 1px solid }
        body { margin: 0 }
        div { font-family: testfont; font-size: 20px }
      </style>
      <div><span/>1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15</div>
    `)
	var text string
	for _, page := range pages {
		_, body := pageBody(t, page)
		div := expectChildren(t, body, 1)[0]
		for _, line := range boxes.ChildrenOf(div) {
			for descendant := range boxes.Descendants(line, false) {
				if t, ok := descendant.(*boxes.TextBox); ok {
					text += t.Text
				}
			}
		}
	}
	if len(pages) != 4 || text != "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15" {
		t.Fatalf("got %d pages with %q", len(pages), text)
	}
}
