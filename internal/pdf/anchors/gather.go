package anchors

import (
	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Link is a hyperlink of a page. Rect is the bounding box (x1, y1, x2, y2) of
// the clickable area in CSS pixels from the top-left corner of the page.
type Link struct {
	Kind   urls.LinkKind
	Target string
	Rect   [4]float64
	Box    boxes.Box
}

// Bookmark is an entry of the outline of a page: a point in CSS pixels from
// the top-left corner of the page.
type Bookmark struct {
	Level int
	Label string
	X, Y  float64
	State string
}

// Data is what a page offers to the document structures that point at
// positions of the page.
type Data struct {
	// Anchors maps each anchor name to the box (x1, y1, x2, y2) it names.
	Anchors   map[string][4]float64
	Links     []Link
	Bookmarks []Bookmark
}

// Gather walks the box tree rooted at root, which should have its
// transformation matrices set, and collects its anchors, links and
// bookmarks. The first of several anchors of the same name wins.
func Gather(root boxes.Box) Data {
	d := Data{Anchors: map[string][4]float64{}}
	d.gather(root, nil, nil)
	return d
}

func (d *Data) gather(box boxes.Box, parentMatrix *matrix.Matrix, parentLink *urls.Link) {
	if ph, ok := box.(*boxes.AbsolutePlaceholder); ok {
		box = ph.Box
	}
	f := box.Base()

	// The matrix of a box does not include those of its ancestors.
	m := parentMatrix
	if f.TransformationMatrix != nil {
		own := *f.TransformationMatrix
		if parentMatrix != nil {
			own = own.Mul(*parentMatrix)
		}
		m = &own
	}

	link := f.Link
	if link == nil {
		link = parentLink
	}
	label := f.BookmarkLabel
	level := f.Style.BookmarkLevel()
	bookmarkLevel := level.N
	if level.Keyword == "none" {
		bookmarkLevel = 0
	}
	name := string(f.Style.Anchor())
	_, isText := box.(*boxes.TextBox)
	_, isLine := box.(*boxes.LineBox)
	hasBookmark := label != "" && bookmarkLevel != 0
	// The link is inherited by the text and lines inside the box that has
	// it, which would only repeat the area of that box.
	hasLink := link != nil && !isText && !isLine
	_, taken := d.Anchors[name]
	hasAnchor := name != "" && !taken

	if hasBookmark || hasLink || hasAnchor {
		hit := f.HitArea()
		if hasLink {
			d.Links = append(d.Links, Link{
				Kind:   link.Kind,
				Target: link.Target,
				Rect:   rectangleAABB(m, hit.X, hit.Y, hit.Width, hit.Height),
				Box:    box,
			})
		}
		if hasBookmark {
			x, y := transformPoint(m, hit.X, hit.Y)
			d.Bookmarks = append(d.Bookmarks, Bookmark{
				Level: bookmarkLevel, Label: label, X: x, Y: y, State: string(f.Style.BookmarkState()),
			})
		}
		if hasAnchor {
			x1, y1 := transformPoint(m, hit.X, hit.Y)
			x2, y2 := transformPoint(m, hit.X+hit.Width, hit.Y+hit.Height)
			d.Anchors[name] = [4]float64{x1, y1, x2, y2}
		}
	}

	for _, child := range boxes.AllChildren(box) {
		d.gather(child, m, link)
	}
}

func transformPoint(m *matrix.Matrix, x, y float64) (float64, float64) {
	if m == nil {
		return x, y
	}
	return m.TransformPoint(x, y)
}

// rectangleAABB returns the axis-aligned bounding box (x1, y1, x2, y2) of the
// rectangle after the transformation m, which may be nil.
func rectangleAABB(m *matrix.Matrix, x, y, width, height float64) [4]float64 {
	if m == nil {
		return [4]float64{x, y, x + width, y + height}
	}
	x1, y1 := m.TransformPoint(x, y)
	x2, y2 := m.TransformPoint(x+width, y)
	x3, y3 := m.TransformPoint(x, y+height)
	x4, y4 := m.TransformPoint(x+width, y+height)
	return [4]float64{
		min(x1, x2, x3, x4), min(y1, y2, y3, y4),
		max(x1, x2, x3, x4), max(y1, y2, y3, y4),
	}
}
