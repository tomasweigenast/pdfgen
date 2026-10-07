package testutil

import "testing"

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
