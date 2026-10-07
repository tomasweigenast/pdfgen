package urls

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// Source is a document to read together with what is known about where it
// came from. The caller closes Body.
type Source struct {
	Body io.ReadCloser
	// BaseURL resolves the relative references of the document. It is ""
	// when the origin does not give one.
	BaseURL string
	// ProtocolEncoding is the charset announced by the transport, if any.
	ProtocolEncoding string
	// MIMEType is the media type announced by the transport, if any.
	MIMEType string
}

// OpenPath reads a file. Without an explicit baseURL, the base is the file
// itself.
func OpenPath(path, baseURL string) (*Source, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		baseURL = PathToURL(path)
	} else {
		baseURL = EnsureURL(baseURL)
	}
	return &Source{Body: file, BaseURL: baseURL}, nil
}

// OpenURL fetches a URL. Without an explicit baseURL, the base is the final
// URL of the response, after redirects.
func OpenURL(ctx context.Context, f Fetcher, url, baseURL string) (*Source, error) {
	if f == nil {
		f = NewDefaultFetcher()
	}
	resp, err := f.Fetch(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		baseURL = resp.URL
	} else {
		baseURL = EnsureURL(baseURL)
	}
	return &Source{Body: resp.Body, BaseURL: baseURL, ProtocolEncoding: resp.Encoding, MIMEType: resp.MIMEType}, nil
}

// FromReader wraps a reader. The name stands for the file the reader comes
// from, if any, and gives the base URL when none is passed. Names written
// like "<stdin>" are not file names and are ignored.
func FromReader(r io.Reader, name, baseURL string) *Source {
	switch {
	case baseURL != "":
		baseURL = EnsureURL(baseURL)
	case name != "" && !strings.HasPrefix(name, "<"):
		baseURL = EnsureURL(name)
	}
	rc, ok := r.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(r)
	}
	return &Source{Body: rc, BaseURL: baseURL}
}

// FromBytes wraps in-memory content.
func FromBytes(b []byte, baseURL string) *Source {
	if baseURL != "" {
		baseURL = EnsureURL(baseURL)
	}
	return &Source{Body: io.NopCloser(bytes.NewReader(b)), BaseURL: baseURL}
}

// Guess opens s as a URL when it starts with a scheme and as a file path
// otherwise.
func Guess(ctx context.Context, f Fetcher, s, baseURL string) (*Source, error) {
	if IsAbsolute(s) {
		return OpenURL(ctx, f, s, baseURL)
	}
	src, err := OpenPath(s, baseURL)
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	return src, nil
}
