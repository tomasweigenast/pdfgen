package stream

import (
	"strconv"

	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// Image is a raster image that can be stored as an image XObject
// (ISO 32000-1 §8.9.5).
type Image interface {
	// ID identifies the image across pages and streams: two images with
	// the same ID are the same XObject.
	ID() string
	// Embed adds the image, with any soft mask or color profile it needs,
	// to pdf and returns the reference of the image XObject. dpiRatio, in
	// (0, 1], is the largest reduction any use of the image asks for.
	Embed(pdf *writer.PDF, interpolate bool, dpiRatio float64) (writer.Ref, error)
}

// imageUse is an image XObject that has been drawn but not written yet.
type imageUse struct {
	image       Image
	interpolate bool
	dpiRatio    float64
}

// imageRegistry holds the images drawn through a set of resources, so that
// each one is embedded once however many pages and patterns use it.
type imageRegistry struct {
	uses map[string]*imageUse
}

// pendingImage stands in an XObject dictionary for an image that
// [Resources.Embed] writes.
type pendingImage struct{ name string }

func (r *Resources) imageRegistry() *imageRegistry {
	if r.images == nil {
		r.images = &imageRegistry{uses: make(map[string]*imageUse)}
	}
	return r.images
}

// AddImage registers img as an XObject and returns its resource name. The
// name is derived from the image ID and interpolation, so repeated calls and
// other streams of the same resources reuse one object. dpiRatio is the
// reduction this use allows; the largest one wins.
func (s *Stream) AddImage(img Image, interpolate bool, dpiRatio float64) string {
	flag := 0
	if interpolate {
		flag = 1
	}
	name := "i" + img.ID() + strconv.Itoa(flag)
	s.res.xObject.set(name, pendingImage{name})

	reg := s.res.imageRegistry()
	if use, ok := reg.uses[name]; ok {
		use.dpiRatio = max(use.dpiRatio, dpiRatio)
		return name
	}
	reg.uses[name] = &imageUse{image: img, interpolate: interpolate, dpiRatio: dpiRatio}
	return name
}

func (e *embedder) image(p pendingImage, r *Resources) writer.Object {
	if ref, ok := e.images[p.name]; ok {
		return ref
	}
	use, ok := r.imageRegistry().uses[p.name]
	if !ok {
		return writer.Null{}
	}
	ref, err := use.image.Embed(e.pdf, use.interpolate, use.dpiRatio)
	if err != nil {
		e.err = err
		return writer.Null{}
	}
	e.images[p.name] = ref
	return ref
}
