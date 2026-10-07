package boxtest

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

// Parse builds the box for the first element of an HTML fragment.
func Parse(t testing.TB, html string) boxes.Box {
	t.Helper()
	t.Skip("box tree not implemented")
	return nil
}

// ParseAll builds the complete box tree of a document and checks that it
// follows the rules about what boxes can contain.
func ParseAll(t testing.TB, html string) *boxes.BlockBox {
	t.Helper()
	t.Skip("box tree not implemented")
	return nil
}

// RenderPages lays out a document and returns its page boxes.
func RenderPages(t testing.TB, html string) []*boxes.PageBox {
	t.Helper()
	t.Skip("rendering pipeline not implemented")
	return nil
}
