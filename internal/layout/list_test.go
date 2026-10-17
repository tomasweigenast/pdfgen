package layout

import (
	"fmt"
	"testing"
)

func TestListsStyle(t *testing.T) {
	for _, inside := range []string{"inside", ""} {
		for _, tt := range []struct{ style, character string }{
			{"circle", "◦ "},
			{"disc", "• "},
			{"square", "▪ "},
		} {
			t.Run(fmt.Sprintf("%s %s", inside, tt.style), func(t *testing.T) {
				d := renderPage(t, fmt.Sprintf(`
      <style>
        body { margin: 0 }
        ul { margin-left: 50px; list-style: %s %s }
      </style>
      <ul>
        <li>abc</li>
      </ul>
    `, inside, tt.style))
				listItem := at(t, d.body(t), 0, 0)
				var markerText, content string
				if inside != "" {
					children := kids(t, kids(t, listItem, 1)[0], 2)
					markerText = textAt(t, children[0], 0)
					content = textOf(t, children[1])
				} else {
					children := kids(t, listItem, 2)
					marker, lineContainer := children[0], children[1]
					if got, want := marker.Base().PositionX, listItem.Base().PositionX-marker.Base().Width.V; got != want {
						t.Errorf("marker x = %g, want %g", got, want)
					}
					if got, want := marker.Base().PositionY, listItem.Base().PositionY; got != want {
						t.Errorf("marker y = %g, want %g", got, want)
					}
					content = textAt(t, lineContainer, 0, 0)
					markerText = textAt(t, marker, 0, 0)
				}
				if markerText != tt.character {
					t.Errorf("marker = %q, want %q", markerText, tt.character)
				}
				if content != "abc" {
					t.Errorf("content = %q, want abc", content)
				}
			})
		}
	}
}

func TestListsEmptyItem(t *testing.T) {
	d := renderPage(t, `
      <ul>
        <li>a</li>
        <li></li>
        <li>a</li>
      </ul>
    `)
	items := kids(t, kids(t, d.body(t), 1)[0], 3)
	y1, y2, y3 := items[0].Base().PositionY, items[1].Base().PositionY, items[2].Base().PositionY
	if y1 == y2 || y2 == y3 {
		t.Fatalf("items at y = %g, %g, %g, want each below the previous", y1, y2, y3)
	}
}

func TestListsWhitespaceItem(t *testing.T) {
	t.Skip("not supported: an item that only has white space takes no height")
	d := renderPage(t, `
      <ul>
        <li>a</li>
        <li> </li>
        <li>a</li>
      </ul>
    `)
	items := kids(t, kids(t, d.body(t), 1)[0], 3)
	y1, y2, y3 := items[0].Base().PositionY, items[1].Base().PositionY, items[2].Base().PositionY
	if y1 == y2 || y2 == y3 {
		t.Fatalf("items at y = %g, %g, %g, want each below the previous", y1, y2, y3)
	}
}

func TestListsMarkerContent(t *testing.T) {
	d := renderPage(t, `
      <style>
        ::marker { content: 'a' }
      </style>
      <ul>
        <li>b</li>
      </ul>
    `)
	children := kids(t, at(t, d.body(t), 0, 0), 2)
	if got := textAt(t, children[0], 0, 0); got != "a" {
		t.Errorf("marker = %q, want a", got)
	}
	if got := textAt(t, children[1], 0, 0); got != "b" {
		t.Errorf("content = %q, want b", got)
	}
}
