package urls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func TestIsAbsolute(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://example.org/", true},
		{"file:///tmp/a", true},
		{"data:,x", true},
		{"git+ssh://host/repo", true},
		{"custom:foo/bar", true},
		{"C:\\dir\\file.html", false},
		{"C:/dir/file.html", false},
		{"relative/path.html", false},
		{"//host/path", false},
		{"#fragment", false},
		{"", false},
		{"1http://x", false},
		{"a b:c", false},
	}
	for _, tt := range tests {
		if got := IsAbsolute(tt.url); got != tt.want {
			t.Errorf("IsAbsolute(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestJoin(t *testing.T) {
	const base = "http://a/b/c/d;p?q"
	tests := []struct{ base, ref, want string }{
		{base, "g:h", "g:h"},
		{base, "g", "http://a/b/c/g"},
		{base, "./g", "http://a/b/c/g"},
		{base, "g/", "http://a/b/c/g/"},
		{base, "/g", "http://a/g"},
		{base, "//g", "http://g"},
		{base, "?y", "http://a/b/c/d;p?y"},
		{base, "g?y", "http://a/b/c/g?y"},
		{base, "#s", "http://a/b/c/d;p?q#s"},
		{base, "g#s", "http://a/b/c/g#s"},
		{base, ";x", "http://a/b/c/;x"},
		{base, "", base},
		{base, ".", "http://a/b/c/"},
		{base, "./", "http://a/b/c/"},
		{base, "..", "http://a/b/"},
		{base, "../", "http://a/b/"},
		{base, "../g", "http://a/b/g"},
		{base, "../..", "http://a/"},
		{base, "../../g", "http://a/g"},
		{base, "../../../g", "http://a/g"},
		{base, "/./g", "http://a/g"},
		{base, "g.", "http://a/b/c/g."},
		{base, "./g/.", "http://a/b/c/g/"},
		{base, "g/./h", "http://a/b/c/g/h"},
		{base, "g/../h", "http://a/b/c/h"},
		{base, "g//h", "http://a/b/c/g/h"},
		{base, "https://other/x", "https://other/x"},
		{"", "rel.png", "rel.png"},
		{"file:///dir/doc.html", "img/a.png", "file:///dir/img/a.png"},
		{"file:///dir/", "a.png", "file:///dir/a.png"},
		{"file:///dir/doc.html", "../a.png", "file:///a.png"},
		{"data:text/css,a", "b.png", "b.png"},
		{"http://host", "a", "http://host/a"},
	}
	for _, tt := range tests {
		if got := Join(tt.base, tt.ref); got != tt.want {
			t.Errorf("Join(%q, %q) = %q, want %q", tt.base, tt.ref, got, tt.want)
		}
	}
}

func TestJoinCustomScheme(t *testing.T) {
	const base = "pdfgen-test:foo/bar.css"
	if got := Join(base, "baz.png"); got != "baz.png" {
		t.Fatalf("before registration: got %q", got)
	}
	RegisterHierarchicalScheme("pdfgen-test")
	if got := Join(base, "baz.png"); got != "pdfgen-test:foo/baz.png" {
		t.Fatalf("after registration: got %q", got)
	}
}

func TestIRIToURI(t *testing.T) {
	tests := []struct{ in, want string }{
		{"http://example.org/a b", "http://example.org/a%20b"},
		{"http://example.org/é_%e9", "http://example.org/%C3%A9_%e9"},
		{"http://example.org/a?b=c d&e=ü#frag", "http://example.org/a?b=c%20d&e=%C3%BC#frag"},
		{"http://example.org/[x]~'()*", "http://example.org/[x]~'()*"},
		{"http://example.org/<>\"{}|\\^`", "http://example.org/%3C%3E%22%7B%7D%7C%5C%5E%60"},
		{"data:text/plain,a b é", "data:text/plain,a b é"},
		{"file:///tmp/ü", "file:///tmp/%C3%BC"},
	}
	for _, tt := range tests {
		got := IRIToURI(tt.in)
		if got != tt.want {
			t.Errorf("IRIToURI(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if again := IRIToURI(got); again != got {
			t.Errorf("IRIToURI is not idempotent on %q: %q", got, again)
		}
	}
}

func TestPathToURL(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a b.html"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	slash := filepath.ToSlash(dir)
	if !strings.HasPrefix(slash, "/") {
		t.Skip("platform paths without a leading slash")
	}
	enc := quotePath(slash)
	tests := []struct{ path, want string }{
		{filepath.Join(dir, "a b.html"), "file://" + enc + "/a%20b.html"},
		{dir, "file://" + enc + "/"},
		{dir + "/", "file://" + enc + "/"},
		{filepath.Join(dir, "missing") + "/", "file://" + enc + "/missing/"},
		{filepath.Join(dir, "x", "..", "ü.css"), "file://" + enc + "/%C3%BC.css"},
	}
	for _, tt := range tests {
		if got := PathToURL(tt.path); got != tt.want {
			t.Errorf("PathToURL(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestPathToURLRelative(t *testing.T) {
	got := PathToURL("resources/doc.html")
	if !strings.HasPrefix(got, "file://") || !strings.HasSuffix(got, "/resources/doc.html") {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureURL(t *testing.T) {
	if got := EnsureURL("http://example.org/x"); got != "http://example.org/x" {
		t.Errorf("URL changed: %q", got)
	}
	if got := EnsureURL("some/file.html"); !strings.HasPrefix(got, "file://") {
		t.Errorf("path not converted: %q", got)
	}
}

func TestUnquote(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"a%20b", "a b"},
		{"%C3%A9", "é"},
		{"100%", "100%"},
		{"%zz%4", "%zz%4"},
		{"%FF", "\uFFFD"},
	}
	for _, tt := range tests {
		if got := Unquote(tt.in); got != tt.want {
			t.Errorf("Unquote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGetLink(t *testing.T) {
	const base = "http://example.org/dir/doc.html"
	tests := []struct {
		name, url, base string
		want            *Link
	}{
		{"fragment without base", "#sec%201", "", &Link{LinkInternal, "sec 1"}},
		{"fragment with base", "#top", base, &Link{LinkInternal, "top"}},
		{"absolute", "http://other.org/a b", base, &Link{LinkExternal, "http://other.org/a%20b"}},
		{"relative", "x.html#y", base, &Link{LinkExternal, "http://example.org/dir/x.html#y"}},
		{"same document", "doc.html#sec", base, &Link{LinkInternal, "sec"}},
		{"same document absolute", base + "#sec", base, &Link{LinkInternal, "sec"}},
		{"same document without fragment", "doc.html", base, &Link{LinkExternal, base}},
		{"same document with query", "doc.html?a=1#sec", base, &Link{LinkExternal, base + "?a=1#sec"}},
		{"relative without base", "x.html", "", &Link{LinkExternal, "x.html"}},
		{"lone hash", "#", base, &Link{LinkExternal, base}},
		{"empty", "", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLink(tt.url, tt.base)
			switch {
			case tt.want == nil && got != nil, tt.want != nil && got == nil, got != nil && *got != *tt.want:
				t.Errorf("GetLink(%q, %q) = %v, want %v", tt.url, tt.base, got, tt.want)
			}
		})
	}
}

func TestGetLinkMalformed(t *testing.T) {
	baseURL := testutil.ResourceURL(t, "dummy.html")
	tests := []struct{ url, base, malformed string }{
		{"https://pdfgen.test]", baseURL, "Malformed URL"},
		{"https://pdfgen.test]", "https://pdfgen.test]", "Malformed URL"},
		{"https://pdfgen.test/", "https://pdfgen.test]", "Malformed base URL"},
	}
	for _, tt := range tests {
		logs := testutil.CaptureLogs(t)
		link := GetLink(tt.url, tt.base)
		lines := logs.Lines()
		if len(lines) != 1 || !strings.Contains(lines[0], tt.malformed) || strings.Contains(tt.malformed, "base") != strings.Contains(lines[0], "base") {
			t.Errorf("GetLink(%q, %q) logged %q, want one %q warning", tt.url, tt.base, lines, tt.malformed)
		}
		if link == nil || link.Kind != LinkExternal || link.Target != IRIToURI(tt.url) {
			t.Errorf("GetLink(%q, %q) = %v, want the URL kept as an external link", tt.url, tt.base, link)
		}
	}
}

func TestJoinCheckedWithoutBase(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	if uri, ok := JoinChecked("", "img.png", false, "img.png"); ok || uri != "" {
		t.Errorf("got %q, %v", uri, ok)
	}
	if lines := logs.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "without a base URI") {
		t.Errorf("logs: %q", lines)
	}
	if uri, ok := JoinChecked("", "img.png", true, ""); !ok || uri != "img.png" {
		t.Errorf("allowRelative: got %q, %v", uri, ok)
	}
	if uri, ok := JoinChecked("http://h/a/", "é.png", false, ""); !ok || uri != "http://h/a/%C3%A9.png" {
		t.Errorf("with base: got %q, %v", uri, ok)
	}
}

func TestGetURLTuple(t *testing.T) {
	tests := []struct {
		url, base string
		want      Link
		ok        bool
	}{
		{"#a%20b", "", Link{LinkInternal, "a b"}, true},
		{"http://x/é", "", Link{LinkExternal, "http://x/%C3%A9"}, true},
		{"y", "http://x/", Link{LinkExternal, "http://x/y"}, true},
		{"y", "", Link{}, false},
	}
	for _, tt := range tests {
		got, ok := GetURLTuple(tt.url, tt.base)
		if got != tt.want || ok != tt.ok {
			t.Errorf("GetURLTuple(%q, %q) = %v, %v", tt.url, tt.base, got, ok)
		}
	}
}
