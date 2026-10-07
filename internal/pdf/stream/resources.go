package stream

import (
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// namedMap is a resource category: names mapped to resources, in
// registration order.
type namedMap struct {
	keys []string
	vals map[string]any
}

func (m *namedMap) len() int { return len(m.keys) }

func (m *namedMap) has(name string) bool {
	_, ok := m.vals[name]
	return ok
}

func (m *namedMap) set(name string, v any) {
	if m.vals == nil {
		m.vals = make(map[string]any)
	}
	if _, ok := m.vals[name]; !ok {
		m.keys = append(m.keys, name)
	}
	m.vals[name] = v
}

// extGState is a graphics state parameter dictionary. A soft mask group is
// kept as a stream until the document is assembled, because the dictionary
// must refer to it indirectly.
type extGState struct {
	dict      *writer.Dict
	maskGroup *Stream
	maskMode  string
}

// Resources is the resource dictionary (ISO 32000-1 §7.8.3) shared by a page
// stream and the form XObjects and patterns created from it. Names are
// numbered per category in registration order.
type Resources struct {
	extGState, xObject, pattern, shading namedMap
	images                               *imageRegistry

	// ColorSpace is the object stored under /ColorSpace, normally a
	// reference to a dictionary of the named color spaces. Nil omits it.
	ColorSpace writer.Object
	// Font is the object stored under /Font, normally a reference to the
	// dictionary of the fonts of the document. Nil omits it.
	Font writer.Object
	// Fonts collects the fonts the streams draw with. Set it before drawing
	// to choose how fonts are embedded; otherwise the first font used
	// creates a registry that subsets. It is shared with child resources.
	Fonts *fonts.Registry
}

// NewResources returns empty resources that refer to colorSpace.
func NewResources(colorSpace writer.Object) *Resources {
	return &Resources{ColorSpace: colorSpace}
}

// child returns the resources of a form XObject or pattern: empty
// categories, with the color spaces of r.
func (r *Resources) child() *Resources {
	return &Resources{ColorSpace: r.ColorSpace, Fonts: r.registry(), images: r.imageRegistry()}
}

func (r *Resources) registry() *fonts.Registry {
	if r.Fonts == nil {
		r.Fonts = fonts.NewRegistry(false)
	}
	return r.Fonts
}

// Embed adds to pdf every stream and dictionary the resources refer to, then
// stores the resource dictionary at ref, which the caller reserved. Call it
// once, after all pages sharing r have been drawn.
func (r *Resources) Embed(pdf *writer.PDF, ref writer.Ref) error {
	e := &embedder{pdf: pdf, refs: make(map[*Stream]writer.Ref), images: make(map[string]writer.Ref), root: r}
	dict := e.resources(r, r.Font)
	if e.err != nil {
		return e.err
	}
	return pdf.Set(ref, dict)
}

// embedder numbers the objects of a resource tree in the order the
// categories are visited: XObjects, patterns, shadings.
type embedder struct {
	pdf    *writer.PDF
	refs   map[*Stream]writer.Ref
	images map[string]writer.Ref
	root   *Resources
	err    error
}

func (e *embedder) resources(r *Resources, fonts writer.Object) *writer.Dict {
	xObjects := writer.NewDict()
	for _, name := range r.xObject.keys {
		xObjects.Set(writer.Name(name), e.object(r.xObject.vals[name], fonts))
	}
	patterns := writer.NewDict()
	for _, name := range r.pattern.keys {
		patterns.Set(writer.Name(name), e.object(r.pattern.vals[name], fonts))
	}
	shadings := writer.NewDict()
	for _, name := range r.shading.keys {
		shadings.Set(writer.Name(name), e.object(r.shading.vals[name], fonts))
	}
	states := writer.NewDict()
	for _, name := range r.extGState.keys {
		states.Set(writer.Name(name), e.state(r.extGState.vals[name].(*extGState)))
	}

	d := writer.NewDict("ExtGState", states, "XObject", xObjects, "Pattern", patterns, "Shading", shadings)
	if r.ColorSpace != nil {
		d.Set("ColorSpace", r.ColorSpace)
	}
	if fonts != nil {
		d.Set("Font", fonts)
	}
	return d
}

// object returns what a resource dictionary entry holds: streams and
// dictionaries become indirect objects, anything else is kept as is.
func (e *embedder) object(v any, fonts writer.Object) writer.Object {
	switch v := v.(type) {
	case pendingImage:
		return e.image(v, e.root)
	case *Stream:
		return e.stream(v, fonts)
	case *writer.Dict:
		return e.pdf.Add(v)
	case writer.Object:
		return v
	}
	return writer.Null{}
}

func (e *embedder) stream(s *Stream, fonts writer.Object) writer.Ref {
	if ref, ok := e.refs[s]; ok {
		return ref
	}
	ref := e.pdf.Reserve()
	e.refs[s] = ref

	dict := s.extra.Clone()
	if s.res != nil {
		resRef := e.pdf.Add(e.resources(s.res, fonts))
		dict.Set("Resources", resRef)
	}
	compress := s.compress
	_ = e.pdf.Set(ref, &writer.Stream{Dict: dict, Data: s.Bytes(), Compress: &compress})
	return ref
}

func (e *embedder) state(g *extGState) *writer.Dict {
	if g.maskGroup == nil {
		return g.dict
	}
	mask := writer.NewDict(
		"Type", writer.Name("Mask"),
		"S", writer.Name(g.maskMode),
		"G", e.refs[g.maskGroup],
	)
	d := g.dict.Clone()
	d.Set("SMask", mask)
	return d
}

// LabColorSpaces returns the named Lab color spaces that [Stream.SetColor]
// selects for Lab-like colors, to be merged into the document's /ColorSpace
// dictionary: "lab-d50" and "lab-d65", with the matching white point
// (ISO 32000-1 §8.6.5.4).
func LabColorSpaces() *writer.Dict {
	lab := func(white [3]float64) writer.Object {
		return writer.Array{writer.Name("Lab"), writer.NewDict(
			"WhitePoint", writer.Array{writer.Real(white[0]), writer.Real(white[1]), writer.Real(white[2])},
			"Range", writer.Array{writer.Int(-125), writer.Int(125), writer.Int(-125), writer.Int(125)},
		)}
	}
	return writer.NewDict("lab-d50", lab(syntax.D50), "lab-d65", lab(syntax.D65))
}
