package testutil

import "testing"

// Box and PageBox stand in for the box tree types until the boxes package
// defines them.
type (
	Box     = any
	PageBox = any
)

// Document is the result of FakeHTML until the rendering pipeline exists.
type Document struct{}

// Option configures FakeHTML.
type Option func(*Document)

// FakeHTML builds a document with the test user agent stylesheet and test
// font, and uncompressed PDF output.
func FakeHTML(t testing.TB, html string, opts ...Option) *Document {
	t.Helper()
	t.Skip("rendering pipeline not implemented")
	return nil
}

// RenderPages lays out a document and returns its page boxes.
func RenderPages(t testing.TB, html string) []PageBox {
	t.Helper()
	t.Skip("rendering pipeline not implemented")
	return nil
}

// Parse builds the box for the first element of an HTML fragment.
func Parse(t testing.TB, html string) Box {
	t.Helper()
	t.Skip("box tree not implemented")
	return nil
}

// ParseAll builds the complete box tree of a document.
func ParseAll(t testing.TB, html string) Box {
	t.Helper()
	t.Skip("box tree not implemented")
	return nil
}

// Serialize converts a box into nested lists comparable with AssertTree.
func Serialize(box Box) any {
	return nil
}

// AssertTree checks that the children of <body> match expected.
func AssertTree(t testing.TB, box Box, expected any) {
	t.Helper()
	t.Skip("box tree not implemented")
}
