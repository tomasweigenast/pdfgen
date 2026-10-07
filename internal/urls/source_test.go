package urls

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func readAll(t *testing.T, s *Source) string {
	t.Helper()
	defer s.Body.Close()
	b, err := io.ReadAll(s.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestOpenPath(t *testing.T) {
	path := testutil.ResourcePath(t, "doc1.html")
	src, err := OpenPath(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := PathToURL(path); src.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", src.BaseURL, want)
	}
	if src.ProtocolEncoding != "" || src.MIMEType != "" {
		t.Errorf("a file carries no protocol information, got %q and %q", src.ProtocolEncoding, src.MIMEType)
	}
	if !strings.Contains(readAll(t, src), "<html>") {
		t.Error("document content not read")
	}
}

func TestOpenPathExplicitBase(t *testing.T) {
	path := testutil.ResourcePath(t, "doc1.html")
	tests := []struct{ base, want string }{
		{"http://example.org/dir/", "http://example.org/dir/"},
		{"sub/dir", PathToURL("sub/dir")},
	}
	for _, tt := range tests {
		src, err := OpenPath(path, tt.base)
		if err != nil {
			t.Fatal(err)
		}
		src.Body.Close()
		if src.BaseURL != tt.want {
			t.Errorf("base %q gave %q, want %q", tt.base, src.BaseURL, tt.want)
		}
	}
}

func TestOpenPathMissing(t *testing.T) {
	if _, err := OpenPath(filepath.Join(t.TempDir(), "nope.html"), ""); !os.IsNotExist(err) {
		t.Errorf("err = %v, want a not-exist error", err)
	}
}

func TestOpenURL(t *testing.T) {
	root := newTestServer(t)
	src, err := OpenURL(context.Background(), nil, root+"/redirect", "")
	if err != nil {
		t.Fatal(err)
	}
	if src.BaseURL != root+"/gzip" {
		t.Errorf("BaseURL = %q, want the URL after the redirect", src.BaseURL)
	}
	if src.ProtocolEncoding != "ISO-8859-1" || src.MIMEType != "text/html" {
		t.Errorf("encoding %q, type %q", src.ProtocolEncoding, src.MIMEType)
	}
	if got := readAll(t, src); got != "<html test=ok>" {
		t.Errorf("body = %q", got)
	}

	src, err = OpenURL(context.Background(), nil, root+"/gzip", "http://elsewhere.test/")
	if err != nil {
		t.Fatal(err)
	}
	src.Body.Close()
	if src.BaseURL != "http://elsewhere.test/" {
		t.Errorf("explicit BaseURL = %q", src.BaseURL)
	}
}

func TestOpenURLError(t *testing.T) {
	root := newTestServer(t)
	if _, err := OpenURL(context.Background(), nil, root+"/bad", ""); err == nil {
		t.Error("no error for a missing document")
	}
}

func TestFromReader(t *testing.T) {
	path := testutil.ResourcePath(t, "doc1.html")
	tests := []struct{ name, base, want string }{
		{"", "", ""},
		{"<stdin>", "", ""},
		{path, "", PathToURL(path)},
		{path, "http://example.org/", "http://example.org/"},
		{"", "http://example.org/", "http://example.org/"},
	}
	for _, tt := range tests {
		src := FromReader(strings.NewReader("x"), tt.name, tt.base)
		if src.BaseURL != tt.want {
			t.Errorf("FromReader(name %q, base %q).BaseURL = %q, want %q", tt.name, tt.base, src.BaseURL, tt.want)
		}
		if got := readAll(t, src); got != "x" {
			t.Errorf("body = %q", got)
		}
	}
}

func TestFromBytes(t *testing.T) {
	src := FromBytes([]byte("abc"), "http://example.org/a/")
	if src.BaseURL != "http://example.org/a/" || readAll(t, src) != "abc" {
		t.Errorf("unexpected source %+v", src)
	}
	if src := FromBytes(nil, ""); src.BaseURL != "" {
		t.Errorf("BaseURL = %q without a base", src.BaseURL)
	}
}

func TestGuess(t *testing.T) {
	root := newTestServer(t)
	path := testutil.ResourcePath(t, "doc1.html")

	src, err := Guess(context.Background(), nil, root+"/gzip", "")
	if err != nil {
		t.Fatal(err)
	}
	src.Body.Close()
	if src.BaseURL != root+"/gzip" {
		t.Errorf("URL guessed wrong: base %q", src.BaseURL)
	}

	src, err = Guess(context.Background(), nil, path, "")
	if err != nil {
		t.Fatal(err)
	}
	src.Body.Close()
	if src.BaseURL != PathToURL(path) {
		t.Errorf("path guessed wrong: base %q", src.BaseURL)
	}

	src, err = Guess(context.Background(), nil, PathToURL(path), "")
	if err != nil {
		t.Fatal(err)
	}
	src.Body.Close()
	if src.ProtocolEncoding != "" || src.MIMEType != "text/html" {
		t.Errorf("file URL: encoding %q, type %q", src.ProtocolEncoding, src.MIMEType)
	}

	if _, err := Guess(context.Background(), nil, filepath.Join(t.TempDir(), "x.html"), ""); err == nil {
		t.Error("no error for a missing path")
	}
}
