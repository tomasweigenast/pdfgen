package layout

import (
	"context"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/images"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/units"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// parseBoxes builds the box tree of a document and returns the boxes made
// by the elements called tag, in tree order.
func parseBoxes(t *testing.T, source, tag string) []boxes.Box {
	t.Helper()
	doc := csstest.Parse(t, source)
	env := csstest.Environment()
	collector := targets.NewCollector()
	styles := csstest.StyleFor(t, doc, cascade.Options{Environment: env, Targets: collector})
	builder := boxes.Builder{
		StyleFor:     styles,
		Targets:      collector,
		CounterStyle: env.CounterStyle,
		GetImage:     images.NewGetter(context.Background(), urls.NewDefaultFetcher(), nil, images.Options{}),
	}
	root, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	var found []boxes.Box
	for b := range boxes.Descendants(root, false) {
		if b.Base().ElementTag == tag {
			found = append(found, b)
		}
	}
	if len(found) == 0 {
		t.Fatalf("no %s box in %q", tag, source)
	}
	return found
}

// parseBox returns the first box made by an element called tag.
func parseBox(t *testing.T, source, tag string) boxes.Box {
	t.Helper()
	return parseBoxes(t, source, tag)[0]
}

// size is a containing block with a known width and height.
func size(w, h float64) containingBlock {
	return containingBlock{Width: units.F(w), Height: units.F(h)}
}

// autoHeight is a containing block whose height depends on its content.
func autoHeight(w float64) containingBlock {
	return containingBlock{Width: units.F(w), Height: units.Auto}
}

// sameMaybe compares dimensions that may be auto.
func sameMaybe(a, b units.MaybeFloat) bool {
	if a.Auto || b.Auto {
		return a.Auto == b.Auto
	}
	return a.V == b.V
}
