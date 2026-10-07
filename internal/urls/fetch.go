package urls

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// Fetcher retrieves the resources a document refers to.
//
// An implementation returns an error for a resource it cannot provide.
// Callers treat the error as non-fatal, warning and carrying on, unless
// IsFatal reports that it must stop the rendering. Implementations backing a
// custom scheme may call RegisterHierarchicalScheme so that relative
// references resolve against its URLs.
type Fetcher interface {
	Fetch(ctx context.Context, url string, headers http.Header) (*Response, error)
}

// Response is a fetched resource. The caller closes Body.
type Response struct {
	// URL is the address the content came from, after any redirects.
	URL string
	// Status is the HTTP status code, or 200 for other schemes.
	Status int
	Header http.Header
	// MIMEType is the media type without parameters, in lower case. It is
	// "text/plain" when the source does not say.
	MIMEType string
	// Encoding is the charset parameter of the content type, if any.
	Encoding string
	// RedirectedTo is set when at least one redirect was followed; it
	// holds the final URL, the same as URL.
	RedirectedTo string
	Body         io.ReadCloser
}

// FetchError reports that a resource could not be fetched. The document is
// rendered without it.
type FetchError struct {
	URL string
	Err error
}

func (e *FetchError) Error() string { return e.Err.Error() }
func (e *FetchError) Unwrap() error { return e.Err }

// FatalError reports a fetch failure that must stop the rendering, as
// requested with DefaultFetcher.FailOnErrors.
type FatalError struct {
	URL string
	Err error
}

func (e *FatalError) Error() string { return fmt.Sprintf("Error fetching %q", e.URL) }
func (e *FatalError) Unwrap() error { return e.Err }

// IsFatal reports whether err, or an error it wraps, stops the rendering.
func IsFatal(err error) bool {
	var fatal *FatalError
	return errors.As(err, &fatal)
}

// HTTPError is returned for an HTTP response whose status is not 2xx.
type HTTPError struct {
	Status int
	URL    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP Error %d: %s", e.Status, http.StatusText(e.Status))
}

// UserAgent is the default value of the User-Agent request header.
const UserAgent = "pdfgen"

// DefaultFetcher fetches file:, http:, https: and data: URLs. Other schemes,
// FTP included, are rejected with an error.
type DefaultFetcher struct {
	// Timeout bounds connecting and waiting for the response headers. Zero
	// or less means no limit.
	Timeout time.Duration
	// TLSConfig configures HTTPS connections; nil uses the defaults.
	TLSConfig *tls.Config
	// Header holds request headers added to the defaults (User-Agent,
	// Accept and Accept-Encoding).
	Header http.Header
	// AllowedProtocols lists the schemes that may be fetched, compared
	// case-insensitively. Nil allows all of them; an empty slice, none.
	AllowedProtocols []string
	// AllowRedirects makes HTTP redirects be followed. Otherwise a
	// redirect is an error.
	AllowRedirects bool
	// FailOnErrors makes every error a *FatalError.
	FailOnErrors bool

	once   sync.Once
	client *http.Client
}

// NewDefaultFetcher returns a fetcher with a 10 second timeout that follows
// redirects.
func NewDefaultFetcher() *DefaultFetcher {
	return &DefaultFetcher{Timeout: 10 * time.Second, AllowRedirects: true}
}

// maxRedirects and maxRepeats bound a redirect chain and the number of times
// it may come back to the same URL.
const (
	maxRedirects = 10
	maxRepeats   = 4
)

func (f *DefaultFetcher) httpClient() *http.Client {
	f.once.Do(func() {
		transport := &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: f.Timeout}).DialContext,
			TLSClientConfig:       f.TLSConfig,
			TLSHandshakeTimeout:   f.Timeout,
			ResponseHeaderTimeout: f.Timeout,
			ForceAttemptHTTP2:     true,
		}
		f.client = &http.Client{Transport: transport, CheckRedirect: f.checkRedirect}
	})
	return f.client
}

func (f *DefaultFetcher) checkRedirect(req *http.Request, via []*http.Request) error {
	if !f.AllowRedirects {
		return http.ErrUseLastResponse
	}
	target := req.URL.String()
	seen := map[string]int{target: 1}
	for _, r := range via[1:] {
		seen[r.URL.String()]++
	}
	if seen[target] >= maxRepeats || len(seen) >= maxRedirects {
		return errors.New("redirect error: infinite loop detected")
	}
	return nil
}

// Fetch implements Fetcher.
func (f *DefaultFetcher) Fetch(ctx context.Context, url string, headers http.Header) (*Response, error) {
	resp, err := f.fetch(ctx, url, headers)
	if err != nil {
		if f.FailOnErrors {
			return nil, &FatalError{URL: url, Err: err}
		}
		return nil, &FetchError{URL: url, Err: err}
	}
	return resp, nil
}

func (f *DefaultFetcher) fetch(ctx context.Context, url string, headers http.Header) (*Response, error) {
	scheme := Scheme(url)
	if scheme == "" {
		return nil, fmt.Errorf("not an absolute URI: %s", url)
	}
	if f.AllowedProtocols != nil && !slices.ContainsFunc(f.AllowedProtocols, func(p string) bool {
		return strings.EqualFold(p, scheme)
	}) {
		return nil, fmt.Errorf("URI uses disallowed protocol: %s", url)
	}
	switch scheme {
	case "file":
		return fetchFile(url)
	case "data":
		return fetchData(url)
	case "http", "https":
		return f.fetchHTTP(ctx, IRIToURI(url), headers)
	case "ftp":
		return nil, fmt.Errorf("FTP is not supported: %s", url)
	}
	return nil, fmt.Errorf("unknown url type: %s", scheme)
}

func (f *DefaultFetcher) fetchHTTP(ctx context.Context, url string, headers http.Header) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip")
	for _, h := range []http.Header{f.Header, headers} {
		for k, v := range h {
			req.Header[http.CanonicalHeaderKey(k)] = slices.Clone(v)
		}
	}

	resp, err := f.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, &HTTPError{Status: resp.StatusCode, URL: url}
	}

	body := decodeBody(resp.Body, resp.Header.Get("Content-Encoding"))
	if body != io.ReadCloser(resp.Body) {
		resp.Header.Del("Content-Encoding")
	}
	out := &Response{
		URL:    resp.Request.URL.String(),
		Status: resp.StatusCode,
		Header: resp.Header,
		Body:   body,
	}
	out.MIMEType, out.Encoding = parseContentType(resp.Header.Get("Content-Type"))
	if out.URL != url {
		out.RedirectedTo = out.URL
	}
	return out, nil
}

// parseContentType splits a Content-Type value into its lower-cased media
// type, "text/plain" when absent or malformed, and its charset parameter.
func parseContentType(value string) (mimeType, charset string) {
	mt, params, err := mime.ParseMediaType(value)
	if err != nil || !strings.Contains(mt, "/") {
		return "text/plain", ""
	}
	return mt, params["charset"]
}

// decodeBody undoes a gzip or deflate content coding. The decoder starts
// reading on first use, so an empty body is not an error until it is read.
func decodeBody(body io.ReadCloser, coding string) io.ReadCloser {
	switch strings.ToLower(strings.TrimSpace(coding)) {
	case "gzip", "x-gzip":
		return &decodedBody{src: body, open: func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) }}
	case "deflate":
		return &decodedBody{src: body, open: func(r io.Reader) (io.Reader, error) { return flate.NewReader(r), nil }}
	}
	return body
}

type decodedBody struct {
	src  io.ReadCloser
	open func(io.Reader) (io.Reader, error)
	r    io.Reader
	err  error
}

func (d *decodedBody) Read(p []byte) (int, error) {
	if d.r == nil && d.err == nil {
		d.r, d.err = d.open(d.src)
		if d.err == io.EOF {
			d.err = io.ErrUnexpectedEOF
		}
	}
	if d.err != nil {
		return 0, d.err
	}
	return d.r.Read(p)
}

func (d *decodedBody) Close() error { return d.src.Close() }

func fetchFile(url string) (*Response, error) {
	u, err := split(url)
	if err != nil {
		return nil, err
	}
	if u.netloc != "" && !strings.EqualFold(u.netloc, "localhost") {
		return nil, fmt.Errorf("file URL has a remote host: %s", url)
	}
	path := Unquote(u.path)
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	path = filepath.FromSlash(path)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err == nil && info.IsDir() {
		err = fmt.Errorf("%s is a directory", path)
	}
	if err != nil {
		file.Close()
		return nil, err
	}

	mimeType := "text/plain"
	if t := mime.TypeByExtension(filepath.Ext(path)); t != "" {
		mimeType, _, _ = strings.Cut(t, ";")
	}
	header := http.Header{}
	header.Set("Content-Type", mimeType)
	header.Set("Content-Length", fmt.Sprint(info.Size()))
	header.Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	return &Response{URL: url, Status: http.StatusOK, Header: header, MIMEType: mimeType, Body: file}, nil
}

// fetchData decodes a data: URL (RFC 2397). Percent escapes are decoded even
// in base64 payloads, and characters outside the base64 alphabet, such as
// line breaks, are ignored.
func fetchData(url string) (*Response, error) {
	rest := url[len("data:"):]
	mediaType, payload, ok := strings.Cut(rest, ",")
	if !ok {
		return nil, fmt.Errorf("malformed data URL: missing comma")
	}
	data := unquoteBytes(payload)
	if n := len(mediaType) - len(";base64"); n >= 0 && strings.EqualFold(mediaType[n:], ";base64") {
		mediaType = mediaType[:n]
		var err error
		if data, err = decodeBase64(data); err != nil {
			return nil, err
		}
	}
	if mediaType == "" {
		mediaType = "text/plain;charset=US-ASCII"
	}
	header := http.Header{}
	header.Set("Content-Type", mediaType)
	header.Set("Content-Length", fmt.Sprint(len(data)))
	mimeType, charset := parseContentType(mediaType)
	return &Response{
		URL:      url,
		Status:   http.StatusOK,
		Header:   header,
		MIMEType: mimeType,
		Encoding: charset,
		Body:     io.NopCloser(strings.NewReader(string(data))),
	}, nil
}

func decodeBase64(data []byte) ([]byte, error) {
	clean := make([]byte, 0, len(data))
	for _, c := range data {
		if isASCIILetter(c) || '0' <= c && c <= '9' || c == '+' || c == '/' {
			clean = append(clean, c)
		}
	}
	out := make([]byte, base64.RawStdEncoding.DecodedLen(len(clean)))
	n, err := base64.RawStdEncoding.Decode(out, clean)
	if err != nil {
		return nil, fmt.Errorf("malformed base64 data: %w", err)
	}
	return out[:n], nil
}
