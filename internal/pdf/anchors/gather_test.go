package anchors_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/pdf/anchors"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

func gather(t *testing.T, body string) anchors.Data {
	t.Helper()
	pages := boxtest.RenderPages(t, `<style>@page { size: 500px; margin: 0 } body { margin: 0 }</style>`+body)
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	boxes.SetTransformationMatrices(pages[0])
	return anchors.Gather(pages[0])
}

func TestLinkRectangleIsTheBoundingBoxOfTheTransformedBox(t *testing.T) {
	tests := []struct {
		name, html string
		want       [4]float64
	}{
		{
			name: "no transform",
			html: `<a href="https://example.org/" style="display: block; width: 100px; height: 50px"></a>`,
			want: [4]float64{0, 0, 100, 50},
		},
		{
			name: "translate",
			html: `<a href="https://example.org/" style="display: block; width: 100px; height: 50px;
				transform: translate(10px, 20px)"></a>`,
			want: [4]float64{10, 20, 110, 70},
		},
		{
			name: "rotate",
			html: `<a href="https://example.org/" style="display: block; width: 100px; height: 50px;
				transform-origin: 0 0; transform: rotate(90deg)"></a>`,
			want: [4]float64{-50, 0, 0, 100},
		},
		{
			name: "transforms of the ancestors apply",
			html: `<div style="width: 200px; transform-origin: 0 0; transform: translate(10px, 0)">
				<a href="https://example.org/" style="display: block; width: 100px; height: 50px;
					transform: translate(0, 5px)"></a></div>`,
			want: [4]float64{10, 5, 110, 55},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := gather(t, tt.html)
			if len(d.Links) != 1 {
				t.Fatalf("got %d links, want 1", len(d.Links))
			}
			link := d.Links[0]
			if link.Kind != urls.LinkExternal || link.Target != "https://example.org/" {
				t.Errorf("link = %v %q", link.Kind, link.Target)
			}
			for i, v := range link.Rect {
				if diff := v - tt.want[i]; diff > 1e-9 || diff < -1e-9 {
					t.Fatalf("rectangle = %v, want %v", link.Rect, tt.want)
				}
			}
		})
	}
}

func TestLinkIsNotRepeatedOnTextAndLines(t *testing.T) {
	d := gather(t, `<p><a href="#top">some <b>bold</b> text</a></p>`)
	// The anchor element, the bold element inside it and nothing else.
	if len(d.Links) != 2 {
		t.Errorf("got %d links, want 2", len(d.Links))
	}
	for _, link := range d.Links {
		switch link.Box.(type) {
		case *boxes.TextBox, *boxes.LineBox:
			t.Errorf("link on a %T", link.Box)
		}
	}
}

func TestFirstAnchorOfAName(t *testing.T) {
	d := gather(t, `<p id="a" style="margin: 0; height: 10px"></p><p id="a" style="margin: 0; height: 20px"></p>`)
	got, ok := d.Anchors["a"]
	if !ok {
		t.Fatal("no anchor a")
	}
	if want := [4]float64{0, 0, 500, 10}; got != want {
		t.Errorf("anchor = %v, want %v", got, want)
	}
}

func TestBookmarksFollowTheDocumentOrder(t *testing.T) {
	d := gather(t, `<h1 style="margin: 0">a</h1><h2 style="margin: 0">b</h2><h2 style="bookmark-state: closed; margin: 0">c</h2>`)
	var labels []string
	for _, b := range d.Bookmarks {
		labels = append(labels, b.Label)
	}
	if len(labels) != 3 || labels[0] != "a" || labels[1] != "b" || labels[2] != "c" {
		t.Fatalf("labels = %q", labels)
	}
	if d.Bookmarks[0].Level != 1 || d.Bookmarks[1].Level != 2 {
		t.Errorf("levels = %d, %d", d.Bookmarks[0].Level, d.Bookmarks[1].Level)
	}
	if d.Bookmarks[2].State != "closed" {
		t.Errorf("state = %q, want closed", d.Bookmarks[2].State)
	}
	if d.Bookmarks[0].Y != 0 || d.Bookmarks[1].Y <= 0 {
		t.Errorf("positions = %v, %v", d.Bookmarks[0].Y, d.Bookmarks[1].Y)
	}
}
