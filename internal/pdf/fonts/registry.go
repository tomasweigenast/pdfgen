package fonts

import (
	textfonts "github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

// Registry holds the fonts of a document, in the order they were first used,
// and writes them out when the document is assembled.
type Registry struct {
	full  bool
	fonts []*Font
	byID  map[string]*Font
}

// NewRegistry returns an empty registry. With full set, fonts are embedded
// whole and their CIDs are glyph indices, which makes every glyph of the
// font available to later editing and form filling.
func NewRegistry(full bool) *Registry {
	return &Registry{full: full, byID: map[string]*Font{}}
}

// Add returns the font that embeds face, creating it on first use. Faces with
// the same ID share one font.
func (r *Registry) Add(face *textfonts.Face) *Font {
	if f, ok := r.byID[face.ID]; ok {
		return f
	}
	f := newFont(face, r.full)
	r.byID[face.ID] = f
	r.fonts = append(r.fonts, f)
	return f
}

// Fonts returns the fonts in the order they were added.
func (r *Registry) Fonts() []*Font { return r.fonts }
