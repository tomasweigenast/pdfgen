package anchors

import (
	"sort"

	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// AddLinks adds a link annotation to file for each link and returns the
// annotations' references, to be set as the /Annots of the page. m maps CSS
// pixels of the page to PDF user space.
func AddLinks(file *writer.PDF, links []Link, m matrix.Matrix) writer.Array {
	var annots writer.Array
	for _, link := range links {
		x1, y1 := m.TransformPoint(link.Rect[0], link.Rect[1])
		x2, y2 := m.TransformPoint(link.Rect[2], link.Rect[3])
		annot := writer.NewDict(
			"Type", writer.Name("Annot"),
			"Subtype", writer.Name("Link"),
			"Rect", writer.Array{writer.Real(x1), writer.Real(y1), writer.Real(x2), writer.Real(y2)},
			"BS", writer.NewDict("W", 0),
		)
		if link.Kind == urls.LinkInternal {
			annot.Set("Dest", writer.Text(link.Target))
		} else {
			annot.Set("A", writer.NewDict(
				"Type", writer.Name("Action"),
				"S", writer.Name("URI"),
				"URI", writer.Text(link.Target),
			))
		}
		ref := file.Add(annot)
		if link.Box != nil {
			link.Box.Base().LinkAnnotation = annot
		}
		annots = append(annots, ref)
	}
	return annots
}

// NamedDestination is an entry of the name tree of destinations.
type NamedDestination struct {
	Name string
	Dest writer.Array
}

// Destinations returns the entries for the destinations of a page: an
// explicit destination at the point of each, with the page at reference page.
// m maps CSS pixels of the page to PDF user space.
func Destinations(dests []Destination, page writer.Ref, m matrix.Matrix) []NamedDestination {
	out := make([]NamedDestination, 0, len(dests))
	for _, d := range dests {
		x, y := m.TransformPoint(d.X, d.Y)
		out = append(out, NamedDestination{
			Name: d.Name,
			Dest: writer.Array{page, writer.Name("XYZ"), writer.Real(x), writer.Real(y), writer.Int(0)},
		})
	}
	return out
}

// SetNameTree sets the /Dests entry of the catalog's /Names to a name tree of
// dests, which is sorted by name as the format requires. It does nothing
// when dests is empty.
func SetNameTree(catalog *writer.Dict, dests []NamedDestination) {
	if len(dests) == 0 {
		return
	}
	sort.SliceStable(dests, func(i, j int) bool { return dests[i].Name < dests[j].Name })
	names := make(writer.Array, 0, 2*len(dests))
	for _, d := range dests {
		names = append(names, writer.Text(d.Name), d.Dest)
	}
	existing, _ := catalog.Get("Names")
	tree, ok := existing.(*writer.Dict)
	if !ok {
		tree = writer.NewDict()
		catalog.Set("Names", tree)
	}
	tree.Set("Dests", writer.NewDict("Names", names))
}

// AddOutlines adds the outline tree to file and sets /Outlines of its
// catalog. pages holds the references of the pages, which the nodes index.
// Nothing is added when nodes is empty.
func AddOutlines(file *writer.PDF, nodes []Node, pages []writer.Ref) {
	if len(nodes) == 0 {
		return
	}
	outlines, count := addOutlineLevel(file, nodes, pages, nil)
	root := writer.NewDict(
		"Count", count,
		"First", outlines[0].ref,
		"Last", outlines[len(outlines)-1].ref,
	)
	rootRef := file.Add(root)
	for _, o := range outlines {
		o.dict.Set("Parent", rootRef)
	}
	file.Catalog().Set("Outlines", rootRef)
}

type outline struct {
	dict *writer.Dict
	ref  writer.Ref
}

// addOutlineLevel adds the outline items for nodes, children of parent, and
// returns them with the number of open items among them and their
// descendants (ISO 32000-1 §12.3.3).
func addOutlineLevel(file *writer.PDF, nodes []Node, pages []writer.Ref, parent *outline) ([]outline, int) {
	count := len(nodes)
	var outlines []outline
	for _, n := range nodes {
		var dest writer.Object = writer.Null{}
		if n.Page >= 0 && n.Page < len(pages) {
			dest = writer.Array{pages[n.Page], writer.Name("XYZ"), writer.Real(n.X), writer.Real(n.Y), writer.Int(0)}
		}
		dict := writer.NewDict("Title", writer.Text(n.Label), "Dest", dest)
		cur := outline{dict: dict, ref: file.Add(dict)}

		children, childCount := addOutlineLevel(file, n.Children, pages, &cur)
		dict.Set("Count", writer.Int(childCount))
		if n.State == "closed" {
			dict.Set("Count", writer.Int(-childCount))
		} else {
			count += childCount
		}
		if len(outlines) > 0 {
			prev := outlines[len(outlines)-1]
			dict.Set("Prev", prev.ref)
			prev.dict.Set("Next", cur.ref)
		}
		if len(children) > 0 {
			dict.Set("First", children[0].ref)
			dict.Set("Last", children[len(children)-1].ref)
		}
		if parent != nil {
			dict.Set("Parent", parent.ref)
		}
		outlines = append(outlines, cur)
	}
	return outlines, count
}
