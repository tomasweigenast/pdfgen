package anchors

import (
	"sort"

	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Destination is a named position of a page, in CSS pixels from its
// top-left corner.
type Destination struct {
	Name string
	X, Y float64
}

// Resolved holds the links of a page that can be written, and the
// destinations that the page defines.
type Resolved struct {
	Links        []Link
	Destinations []Destination
}

// ResolveLinks decides, for every page, which links are kept and which
// anchors it defines. Internal links to a missing anchor are dropped with an
// error. When several anchors share a name the first one in document order
// is the destination.
func ResolveLinks(pages []*Data) []Resolved {
	known := map[string]bool{}
	out := make([]Resolved, len(pages))
	for i, page := range pages {
		names := make([]string, 0, len(page.Anchors))
		for name := range page.Anchors {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if known[name] {
				continue
			}
			known[name] = true
			a := page.Anchors[name]
			out[i].Destinations = append(out[i].Destinations, Destination{Name: name, X: a[0], Y: a[1]})
		}
	}
	for i, page := range pages {
		for _, link := range page.Links {
			if link.Kind == urls.LinkInternal && !known[link.Target] {
				logging.Errorf("No anchor #%s for internal URI reference", link.Target)
				continue
			}
			out[i].Links = append(out[i].Links, link)
		}
	}
	return out
}
