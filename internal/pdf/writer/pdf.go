package writer

import (
	"fmt"
)

// firstUserObject is the number of the first object a caller can add: object
// 0 is the free-list head, 1 the page tree root and 2 the catalog.
const firstUserObject = 3

// PDF is a document under construction. Objects are numbered in creation
// order; Info, when not empty, is appended as the last object at write time.
type PDF struct {
	// objects[0] is unused; a nil entry is a reserved number not yet set.
	objects []Object
	pages   *Dict
	catalog *Dict
	info    *Dict
}

// New returns a document holding an empty page tree (object 1) and a catalog
// pointing at it (object 2).
func New() *PDF {
	p := &PDF{
		pages: NewDict("Type", Name("Pages"), "Kids", Array{}, "Count", 0),
		info:  NewDict(),
	}
	p.catalog = NewDict("Type", Name("Catalog"), "Pages", Ref{1})
	p.objects = []Object{nil, p.pages, p.catalog}
	return p
}

// Add appends o to the document and returns its reference.
func (p *PDF) Add(o Object) Ref {
	if o == nil {
		o = Null{}
	}
	p.objects = append(p.objects, o)
	return Ref{Num: len(p.objects) - 1}
}

// Reserve allocates the next object number without a value, so that
// references to it can be created before the object exists. [PDF.Set] must
// fill it before writing.
func (p *PDF) Reserve() Ref {
	p.objects = append(p.objects, nil)
	return Ref{Num: len(p.objects) - 1}
}

// Set stores o as the value of the object r, which must have been returned
// by [PDF.Add] or [PDF.Reserve].
func (p *PDF) Set(r Ref, o Object) error {
	if r.Num < firstUserObject || r.Num >= len(p.objects) {
		return fmt.Errorf("writer: cannot set object %d", r.Num)
	}
	if o == nil {
		o = Null{}
	}
	p.objects[r.Num] = o
	return nil
}

// Object returns the value of the object r. It is nil for a reserved object
// that has not been set.
func (p *PDF) Object(r Ref) (Object, bool) {
	if r.Num < 1 || r.Num >= len(p.objects) {
		return nil, false
	}
	return p.objects[r.Num], true
}

// AddPage adds page as a leaf of the page tree: it is appended to /Kids,
// /Count is incremented and /Parent points to the tree root.
func (p *PDF) AddPage(page *Dict) Ref {
	page.Set("Parent", p.PagesRef())
	r := p.Add(page)
	count, _ := p.pages.Get("Count")
	n, _ := count.(Int)
	p.pages.Set("Count", n+1)
	kids, _ := p.pages.Get("Kids")
	a, _ := kids.(Array)
	p.pages.Set("Kids", append(a, r))
	return r
}

// PageRefs returns the references in the page tree's /Kids, in order.
func (p *PDF) PageRefs() []Ref {
	kids, _ := p.pages.Get("Kids")
	a, _ := kids.(Array)
	refs := make([]Ref, 0, len(a))
	for _, k := range a {
		if r, ok := k.(Ref); ok {
			refs = append(refs, r)
		}
	}
	return refs
}

// Catalog returns the document catalog dictionary.
func (p *PDF) Catalog() *Dict { return p.catalog }

// CatalogRef returns the reference to the catalog.
func (p *PDF) CatalogRef() Ref { return Ref{Num: 2} }

// Pages returns the page tree root dictionary.
func (p *PDF) Pages() *Dict { return p.pages }

// PagesRef returns the reference to the page tree root.
func (p *PDF) PagesRef() Ref { return Ref{Num: 1} }

// Info returns the document information dictionary. It is written, as the
// last object, only when it has entries.
func (p *PDF) Info() *Dict { return p.info }
