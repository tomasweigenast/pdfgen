package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/html"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

// styled is a document with the computed styles of its elements.
type styled struct {
	t   *testing.T
	doc *html.Document
	sf  *cascade.StyleFor
}

func newStyled(t *testing.T, source string) *styled {
	t.Helper()
	doc := csstest.Parse(t, source)
	return &styled{t: t, doc: doc, sf: csstest.StyleFor(t, doc, cascade.Options{})}
}

func (s *styled) el(name string) *html.Element {
	s.t.Helper()
	return csstest.Element(s.t, s.doc, name)
}

func (s *styled) style(name string) *computed.Style {
	s.t.Helper()
	return s.sf.Get(s.el(name), "")
}

// block returns a block box for the first element called name.
func (s *styled) block(name string, children ...boxes.Box) *boxes.BlockBox {
	s.t.Helper()
	return boxes.NewBlockBox(name, s.style(name), s.el(name), children)
}

// plainStyle is a style without declarations, for boxes that do not need
// an element.
func plainStyle() *computed.Style { return computed.New(nil) }

func pageType() cascade.PageType { return cascade.PageType{Side: "right"} }
