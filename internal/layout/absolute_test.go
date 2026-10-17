package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

// absoluteImage lays out an image positioned absolutely in a box of 21px by
// 11px, and returns the image.
func absoluteImage(t *testing.T, container, image string) boxes.Box {
	t.Helper()
	noLogs(t)
	page := onlyPage(t, paginate(t, `
      <style>
        div { position: relative; width: 21px; height: 11px; `+container+` }
        img { display: block; position: absolute; `+image+` }
      </style>
      <div><img src=pattern.png></div>`))
	_, body := pageBody(t, page)
	return expectChildren(t, expectChildren(t, body, 1)[0], 1)[0]
}

func TestAbsoluteReplacedAutoMarginsRoundDown(t *testing.T) {
	// The room left for the margins is 21 - 4 = 17, shared as 8 and 8.
	img := absoluteImage(t, "", "left: 0; right: 0; margin: auto")
	f := img.Base()
	if f.MarginLeft.V != 8 || f.MarginRight.V != 8 || img.ContentBoxX() != 8 {
		t.Fatalf("margins %g and %g, content at %g; want 8, 8, 8", f.MarginLeft.V, f.MarginRight.V, img.ContentBoxX())
	}
	// And 11 - 4 = 7 vertically, shared as 3 and 3.
	img = absoluteImage(t, "", "top: 0; bottom: 0; margin: auto")
	f = img.Base()
	if f.MarginTop.V != 3 || f.MarginBottom.V != 3 || img.ContentBoxY() != 3 {
		t.Fatalf("margins %g and %g, content at %g; want 3, 3, 3", f.MarginTop.V, f.MarginBottom.V, img.ContentBoxY())
	}
}

func TestAbsoluteReplacedWiderThanItsContainingBlock(t *testing.T) {
	tests := []struct {
		name      string
		container string
		left      float64
		right     float64
	}{
		{"ltr", "", 0, -9},
		{"rtl", "direction: rtl", -9, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := absoluteImage(t, tt.container, "left: 0; right: 0; margin: auto; width: 30px")
			f := img.Base()
			if f.MarginLeft.V != tt.left || f.MarginRight.V != tt.right {
				t.Fatalf("margins %g and %g, want %g and %g", f.MarginLeft.V, f.MarginRight.V, tt.left, tt.right)
			}
		})
	}
}

func TestAbsoluteReplacedStaticPositionAndOverConstraint(t *testing.T) {
	tests := []struct {
		name      string
		container string
		image     string
		x, y      float64
	}{
		{"static position", "", "", 0, 0},
		{"left only", "", "left: 5px; top: 2px", 5, 2},
		{"right and bottom", "", "right: 3px; bottom: 1px", 21 - 3 - 4, 11 - 1 - 4},
		{"over-constrained keeps left in ltr", "", "left: 5px; right: 1px", 5, 0},
		{"over-constrained keeps right in rtl", "direction: rtl", "left: 5px; right: 1px", 21 - 1 - 4, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := absoluteImage(t, tt.container, tt.image)
			if got := xy(img); got != [2]float64{tt.x, tt.y} {
				t.Fatalf("image at %v, want (%g, %g)", got, tt.x, tt.y)
			}
		})
	}
}

func TestAbsoluteBlockAnchoredToTheRight(t *testing.T) {
	tests := []struct {
		name      string
		container string
		style     string
		x, w      float64
	}{
		{"right with the width of its content", "", "right: 0", 21 - 6, 6},
		{"left and right set the width", "", "left: 2px; right: 3px", 2, 16},
		{"over-constrained in ltr moves the right margin", "", "left: 2px; right: 3px; width: 5px", 2, 5},
		{"auto offsets and width at the right in rtl", "direction: rtl", "", 21 - 6, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noLogs(t)
			page := onlyPage(t, paginate(t, `
      <style>
        article { position: relative; width: 21px; height: 11px; `+tt.container+` }
        div { position: absolute; `+tt.style+` }
        span { display: block; width: 6px; height: 3px }
      </style>
      <article><div><span></span></div></article>`))
			_, body := pageBody(t, page)
			div := expectChildren(t, expectChildren(t, body, 1)[0], 1)[0]
			if got := [2]float64{div.Base().PositionX, div.Base().Width.V}; got != [2]float64{tt.x, tt.w} {
				t.Fatalf("(x, width) = %v, want (%g, %g)", got, tt.x, tt.w)
			}
		})
	}
}
