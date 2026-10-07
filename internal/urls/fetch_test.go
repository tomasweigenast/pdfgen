package urls

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

var _ Fetcher = (*DefaultFetcher)(nil)

func newTestServer(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/gzip", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			io.WriteString(w, "<html test=accept-encoding-header-fail")
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
		zw := gzip.NewWriter(w)
		io.WriteString(zw, "<html test=ok>")
		zw.Close()
	})
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "auth" {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		io.WriteString(w, "secret")
	})
	mux.HandleFunc("/echo-ua", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Header.Get("User-Agent")+"|"+r.Header.Get("Accept")+"|"+r.Header.Get("X-Test"))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/gzip", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/redirect-loop", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect-loop-2", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/redirect-loop-2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect-loop", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/empty-gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func fetchBody(t *testing.T, f Fetcher, url string, headers http.Header) (*Response, string) {
	t.Helper()
	resp, err := f.Fetch(context.Background(), url, headers)
	if err != nil {
		t.Fatalf("Fetch(%q): %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %q: %v", url, err)
	}
	return resp, string(body)
}

func TestFetchHTTPDecodesGzip(t *testing.T) {
	root := newTestServer(t)
	resp, body := fetchBody(t, NewDefaultFetcher(), root+"/gzip", nil)
	if body != "<html test=ok>" {
		t.Errorf("body = %q", body)
	}
	if resp.Header.Get("Content-Encoding") != "" {
		t.Errorf("Content-Encoding kept: %q", resp.Header.Get("Content-Encoding"))
	}
	if resp.Status != 200 || resp.MIMEType != "text/html" || resp.Encoding != "ISO-8859-1" {
		t.Errorf("status %d, type %q, encoding %q", resp.Status, resp.MIMEType, resp.Encoding)
	}
	if resp.RedirectedTo != "" {
		t.Errorf("RedirectedTo = %q without a redirect", resp.RedirectedTo)
	}
}

func TestFetchHTTPEmptyGzipBodyFailsOnRead(t *testing.T) {
	root := newTestServer(t)
	resp, err := NewDefaultFetcher().Fetch(context.Background(), root+"/empty-gzip", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Error("reading an empty gzip body succeeded")
	}
}

func TestFetchHTTPFollowsRedirects(t *testing.T) {
	root := newTestServer(t)
	resp, body := fetchBody(t, NewDefaultFetcher(), root+"/redirect", nil)
	if body != "<html test=ok>" {
		t.Errorf("body = %q", body)
	}
	if resp.URL != root+"/gzip" || resp.RedirectedTo != root+"/gzip" {
		t.Errorf("URL = %q, RedirectedTo = %q", resp.URL, resp.RedirectedTo)
	}
}

func TestFetchHTTPErrors(t *testing.T) {
	root := newTestServer(t)
	noRedirects := &DefaultFetcher{Timeout: 10 * time.Second}
	tests := []struct {
		name    string
		fetcher *DefaultFetcher
		path    string
		want    string
	}{
		{"missing", NewDefaultFetcher(), "/bad", "404"},
		{"unauthorized", NewDefaultFetcher(), "/auth", "401"},
		{"redirect not followed", noRedirects, "/redirect", "301"},
		{"redirect loop", NewDefaultFetcher(), "/redirect-loop", "infinite loop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fetcher.Fetch(context.Background(), root+tt.path, nil)
			var fe *FetchError
			if !errors.As(err, &fe) || IsFatal(err) {
				t.Fatalf("err = %v (%T), want a non-fatal *FetchError", err, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %q, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestFetchHTTPStatusError(t *testing.T) {
	root := newTestServer(t)
	_, err := NewDefaultFetcher().Fetch(context.Background(), root+"/bad", nil)
	var he *HTTPError
	if !errors.As(err, &he) || he.Status != http.StatusNotFound {
		t.Fatalf("err = %v, want *HTTPError with status 404", err)
	}
}

func TestFetchFailOnErrors(t *testing.T) {
	root := newTestServer(t)
	f := NewDefaultFetcher()
	f.FailOnErrors = true
	fetchBody(t, f, root+"/redirect", nil)

	for _, url := range []string{root + "/bad", "ftp://example.org/a", "relative.png"} {
		_, err := f.Fetch(context.Background(), url, nil)
		if !IsFatal(err) || !strings.Contains(err.Error(), "Error fetching") {
			t.Errorf("Fetch(%q) error = %v, want a fatal error", url, err)
		}
	}

	f = &DefaultFetcher{FailOnErrors: true}
	_, err := f.Fetch(context.Background(), root+"/redirect", nil)
	if !IsFatal(err) {
		t.Errorf("redirect without AllowRedirects: err = %v, want fatal", err)
	}
	if !strings.Contains(errors.Unwrap(err).Error(), "301") {
		t.Errorf("cause = %v, want the status", errors.Unwrap(err))
	}
}

func TestFetchHTTPHeaders(t *testing.T) {
	root := newTestServer(t)
	tests := []struct {
		name    string
		header  http.Header
		perCall http.Header
		ok      bool
	}{
		{"fetcher header", http.Header{"Authorization": {"auth"}}, nil, true},
		{"per-call header", nil, http.Header{"Authorization": {"auth"}}, true},
		{"per-call overrides", http.Header{"Authorization": {"bad"}}, http.Header{"Authorization": {"auth"}}, true},
		{"lower-case key", http.Header{"authorization": {"auth"}}, nil, true},
		{"none", nil, nil, false},
		{"wrong value", http.Header{"Authorization": {"bad"}}, nil, false},
		{"wrong name", http.Header{"Test": {"auth"}}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDefaultFetcher()
			f.Header = tt.header
			_, err := f.Fetch(context.Background(), root+"/auth", tt.perCall)
			if (err == nil) != tt.ok {
				t.Errorf("err = %v, want success %v", err, tt.ok)
			}
		})
	}
}

func TestFetchHTTPDefaultHeaders(t *testing.T) {
	root := newTestServer(t)
	f := NewDefaultFetcher()
	f.Header = http.Header{"X-Test": {"1"}, "Accept": {"text/css"}}
	_, body := fetchBody(t, f, root+"/echo-ua", nil)
	if want := UserAgent + "|text/css|1"; body != want {
		t.Errorf("got %q, want %q", body, want)
	}
}

func TestFetchHTTPTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	f := NewDefaultFetcher()
	f.Timeout = 50 * time.Millisecond
	start := time.Now()
	if _, err := f.Fetch(context.Background(), srv.URL, nil); err == nil {
		t.Fatal("no error for a server that never answers")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v to give up", elapsed)
	}
}

func TestFetchHTTPContextCancelled(t *testing.T) {
	root := newTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewDefaultFetcher().Fetch(ctx, root+"/gzip", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestFetchAllowedProtocols(t *testing.T) {
	path := testutil.ResourcePath(t, "pattern.png")
	fileURL := PathToURL(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dataURL := "data:image/png;base64," + encodeBase64(data)

	tests := []struct {
		name    string
		allowed []string
		url     string
		ok      bool
	}{
		{"unrestricted file", nil, fileURL, true},
		{"file allowed", []string{"file"}, fileURL, true},
		{"several allowed", []string{"file", "http"}, fileURL, true},
		{"repeated", []string{"file", "file"}, fileURL, true},
		{"case-insensitive", []string{"Http", "File"}, fileURL, true},
		{"unrestricted data", nil, dataURL, true},
		{"data allowed", []string{"data"}, dataURL, true},
		{"data allowed upper case", []string{"File", "Data"}, dataURL, true},
		{"http only", []string{"http"}, fileURL, false},
		{"http and https", []string{"http", "https"}, fileURL, false},
		{"similar name", []string{"filetest"}, fileURL, false},
		{"empty entry", []string{""}, fileURL, false},
		{"empty list", []string{}, fileURL, false},
		{"data not allowed", []string{"file"}, dataURL, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewDefaultFetcher()
			f.AllowedProtocols = tt.allowed
			resp, err := f.Fetch(context.Background(), tt.url, nil)
			if tt.ok {
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
				return
			}
			if err == nil || !strings.Contains(err.Error(), "URI uses disallowed protocol") {
				t.Errorf("err = %v, want a disallowed protocol error", err)
			}
		})
	}
}

func TestFetchUnsupportedSchemes(t *testing.T) {
	f := NewDefaultFetcher()
	tests := []struct{ url, want string }{
		{"ftp://example.org/file.css", "FTP is not supported"},
		{"gopher://example.org/", "unknown url type: gopher"},
		{"relative/path.css", "not an absolute URI"},
	}
	for _, tt := range tests {
		_, err := f.Fetch(context.Background(), tt.url, nil)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Fetch(%q) error = %v, want %q", tt.url, err, tt.want)
		}
	}
}

func TestFetchData(t *testing.T) {
	tests := []struct {
		name, url, body, mime, charset string
	}{
		{"default type", "data:,Hello%2C%20World", "Hello, World", "text/plain", "US-ASCII"},
		{"percent escapes", "data:text/css;charset=ASCII,a%7Bcolor%3AcurrentColor%7D", "a{color:currentColor}", "text/css", "ASCII"},
		{"raw space", "data:,ul {border-width: 1000px !important}", "ul {border-width: 1000px !important}", "text/plain", "US-ASCII"},
		{"base64", "data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==", "Hello, World!", "text/plain", ""},
		{"base64 upper-case marker", "data:text/plain;BASE64,SGk=", "Hi", "text/plain", ""},
		{"base64 without padding", "data:text/plain;base64,SGk", "Hi", "text/plain", ""},
		{"base64 with whitespace", "data:text/plain;base64,SGVs\n   bG8s IFdv\tcmxkIQ==", "Hello, World!", "text/plain", ""},
		{"base64 with escapes", "data:text/plain;base64,SGk%3D", "Hi", "text/plain", ""},
		{"utf-16 payload", "data:text/css;charset=utf-16le;base64,bABpAHsAYwBvAGwAbwByADoAcgBlAGQAfQA=", "l\x00i\x00{\x00c\x00o\x00l\x00o\x00r\x00:\x00r\x00e\x00d\x00}\x00", "text/css", "utf-16le"},
		{"svg with hash", "data:image/svg+xml,<svg fill=\"#fff\"></svg>", "<svg fill=\"#fff\"></svg>", "image/svg+xml", ""},
		{"non-ASCII text", "data:text/plain;charset=utf-8,é", "é", "text/plain", "utf-8"},
		{"empty payload", "data:text/plain,", "", "text/plain", ""},
		{"scheme in upper case", "DATA:text/plain,x", "x", "text/plain", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := fetchBody(t, NewDefaultFetcher(), tt.url, nil)
			if body != tt.body {
				t.Errorf("body = %q, want %q", body, tt.body)
			}
			if resp.MIMEType != tt.mime || resp.Encoding != tt.charset {
				t.Errorf("type %q, charset %q; want %q, %q", resp.MIMEType, resp.Encoding, tt.mime, tt.charset)
			}
		})
	}
}

func TestFetchDataMalformed(t *testing.T) {
	for _, url := range []string{"data:text/plain", "data:text/plain;base64,A"} {
		if _, err := NewDefaultFetcher().Fetch(context.Background(), url, nil); err == nil {
			t.Errorf("Fetch(%q) succeeded", url)
		}
	}
}

func TestFetchFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("body { color: red }")
	if err := os.WriteFile(filepath.Join(dir, "sheet.css"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "é b.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "noext"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := PathToURL(dir) + "/"

	tests := []struct {
		name, url, body, mime string
	}{
		{"plain", base + "sheet.css", string(content), "text/css"},
		{"query ignored", base + "sheet.css?v=2", string(content), "text/css"},
		{"fragment ignored", base + "sheet.css#top", string(content), "text/css"},
		{"escaped name", PathToURL(filepath.Join(dir, "é b.txt")), "text", "text/plain"},
		{"localhost host", "file://localhost" + strings.TrimPrefix(base, "file://") + "sheet.css", string(content), "text/css"},
		{"unknown extension", base + "noext", "x", "text/plain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := fetchBody(t, NewDefaultFetcher(), tt.url, nil)
			if body != tt.body || resp.MIMEType != tt.mime || resp.Status != 200 {
				t.Errorf("got %q (%s, %d), want %q (%s)", body, resp.MIMEType, resp.Status, tt.body, tt.mime)
			}
			if resp.Header.Get("Content-Length") == "" || resp.Header.Get("Last-Modified") == "" {
				t.Errorf("headers: %v", resp.Header)
			}
		})
	}
}

func TestFetchFileErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct{ name, url string }{
		{"missing", PathToURL(filepath.Join(dir, "missing.css"))},
		{"directory", PathToURL(dir)},
		{"remote host", "file://example.org/etc/passwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewDefaultFetcher().Fetch(context.Background(), tt.url, nil); err == nil {
				t.Error("no error")
			}
		})
	}
}

func TestParseContentType(t *testing.T) {
	tests := []struct{ in, mime, charset string }{
		{"text/css;charset=iso-8859-15", "text/css", "iso-8859-15"},
		{"TEXT/HTML; Charset=\"UTF-8\"", "text/html", "UTF-8"},
		{"image/png", "image/png", ""},
		{"", "text/plain", ""},
		{"garbage", "text/plain", ""},
	}
	for _, tt := range tests {
		mt, cs := parseContentType(tt.in)
		if mt != tt.mime || cs != tt.charset {
			t.Errorf("parseContentType(%q) = %q, %q; want %q, %q", tt.in, mt, cs, tt.mime, tt.charset)
		}
	}
}

func TestDecodeBodyDeflate(t *testing.T) {
	var buf bytes.Buffer
	fw, _ := flateWriter(&buf)
	fw.Write([]byte("squeezed"))
	fw.Close()
	body := decodeBody(io.NopCloser(&buf), "deflate")
	got, err := io.ReadAll(body)
	if err != nil || string(got) != "squeezed" {
		t.Errorf("got %q, %v", got, err)
	}
}
