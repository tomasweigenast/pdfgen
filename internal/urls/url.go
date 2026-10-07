package urls

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// splitURL is the result of splitting a URL reference into the five
// components of RFC 3986 §3.
type splitURL struct {
	scheme, netloc, path, query, fragment string
}

// errInvalidIPv6 is reported for an authority holding only one of the two
// brackets of an IP literal.
var errInvalidIPv6 = errors.New("invalid IPv6 URL")

// schemeLength returns the length of the scheme prefix of s including the
// colon, or 0 when s does not start with a scheme of at least minLen
// characters (RFC 3986 §3.1). Absolute-URL detection asks for two so that
// Windows drive letters are not mistaken for schemes.
func schemeLength(s string, minLen int) int {
	if len(s) == 0 || !isASCIILetter(s[0]) {
		return 0
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case isASCIILetter(c) || '0' <= c && c <= '9' || c == '.' || c == '+' || c == '-':
		case c == ':' && i >= minLen:
			return i + 1
		default:
			return 0
		}
	}
	return 0
}

func isASCIILetter(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }

// IsAbsolute reports whether url starts with a scheme.
func IsAbsolute(url string) bool { return schemeLength(url, 2) > 0 }

// Scheme returns the lower-cased scheme of url, or "" when url is relative.
func Scheme(url string) string {
	n := schemeLength(url, 2)
	if n == 0 {
		return ""
	}
	return strings.ToLower(url[:n-1])
}

// split breaks a URL reference into its components. Leading control
// characters and spaces are dropped and tabs and newlines removed, as
// browsers do (URL Standard, "basic URL parser"). An error is returned when
// the authority is not well formed.
func split(url string) (splitURL, error) {
	url = strings.TrimLeft(url, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	if strings.ContainsAny(url, "\t\r\n") {
		url = strings.NewReplacer("\t", "", "\r", "", "\n", "").Replace(url)
	}
	var u splitURL
	if n := schemeLength(url, 1); n > 0 {
		u.scheme = strings.ToLower(url[:n-1])
		url = url[n:]
	}
	if strings.HasPrefix(url, "//") {
		end := strings.IndexAny(url[2:], "/?#")
		if end < 0 {
			end = len(url) - 2
		}
		u.netloc = url[2 : 2+end]
		url = url[2+end:]
		if strings.Contains(u.netloc, "[") != strings.Contains(u.netloc, "]") {
			return u, errInvalidIPv6
		}
	}
	if i := strings.IndexByte(url, '#'); i >= 0 {
		u.fragment = url[i+1:]
		url = url[:i]
	}
	if i := strings.IndexByte(url, '?'); i >= 0 {
		u.query = url[i+1:]
		url = url[:i]
	}
	u.path = url
	return u, nil
}

func (u splitURL) String() string {
	s := u.path
	if u.netloc != "" || (usesNetloc(u.scheme) && !strings.HasPrefix(s, "//")) {
		if s != "" && s[0] != '/' {
			s = "/" + s
		}
		s = "//" + u.netloc + s
	}
	if u.scheme != "" {
		s = u.scheme + ":" + s
	}
	if u.query != "" {
		s += "?" + u.query
	}
	if u.fragment != "" {
		s += "#" + u.fragment
	}
	return s
}

var (
	schemesMu       sync.RWMutex
	relativeSchemes = map[string]bool{
		"": true, "ftp": true, "http": true, "gopher": true, "nntp": true,
		"imap": true, "wais": true, "file": true, "https": true, "shttp": true,
		"mms": true, "prospero": true, "rtsp": true, "rtspu": true,
		"sftp": true, "svn": true, "svn+ssh": true, "ws": true, "wss": true,
	}
	netlocSchemes = map[string]bool{
		"": true, "ftp": true, "http": true, "gopher": true, "nntp": true,
		"telnet": true, "imap": true, "wais": true, "file": true, "mms": true,
		"https": true, "shttp": true, "snews": true, "prospero": true,
		"rtsp": true, "rtspu": true, "rsync": true, "svn": true,
		"svn+ssh": true, "sftp": true, "nfs": true, "git": true,
		"git+ssh": true, "ws": true, "wss": true,
	}
)

func usesNetloc(scheme string) bool {
	schemesMu.RLock()
	defer schemesMu.RUnlock()
	return scheme != "" && netlocSchemes[scheme]
}

func usesRelative(scheme string) bool {
	schemesMu.RLock()
	defer schemesMu.RUnlock()
	return relativeSchemes[scheme]
}

// RegisterHierarchicalScheme declares that URLs of a custom scheme have
// slash-separated paths, so that relative references are resolved against them
// by Join. Only the well-known network and file schemes resolve by default;
// for any other scheme a relative reference is returned unchanged. Fetchers
// serving a custom scheme call this once at start-up.
func RegisterHierarchicalScheme(scheme string) {
	scheme = strings.ToLower(scheme)
	schemesMu.Lock()
	defer schemesMu.Unlock()
	relativeSchemes[scheme] = true
}

// resolve joins a URL reference to a base URL following RFC 3986 §5.2, with
// two deliberate departures that browsers share: empty path segments in the
// middle of a relative path are dropped, and only the schemes known to be
// hierarchical are resolved.
func resolve(base, ref string) (string, error) {
	if base == "" {
		return ref, nil
	}
	if ref == "" {
		return base, nil
	}
	b, err := split(base)
	if err != nil {
		return "", err
	}
	r, err := split(ref)
	if err != nil {
		return "", err
	}
	if r.scheme == "" {
		r.scheme = b.scheme
	}
	if r.scheme != b.scheme || !usesRelative(r.scheme) {
		return ref, nil
	}
	if usesNetloc(r.scheme) {
		if r.netloc != "" {
			return r.String(), nil
		}
		r.netloc = b.netloc
	}
	if r.path == "" {
		r.path = b.path
		if r.query == "" {
			r.query = b.query
		}
		return r.String(), nil
	}

	baseParts := strings.Split(b.path, "/")
	if baseParts[len(baseParts)-1] != "" {
		baseParts = baseParts[:len(baseParts)-1]
	}
	var segments []string
	if strings.HasPrefix(r.path, "/") {
		segments = strings.Split(r.path, "/")
	} else {
		segments = append(baseParts, strings.Split(r.path, "/")...)
		if len(segments) > 2 {
			kept := segments[:1]
			for _, s := range segments[1 : len(segments)-1] {
				if s != "" {
					kept = append(kept, s)
				}
			}
			segments = append(kept, segments[len(segments)-1])
		}
	}
	var out []string
	for _, seg := range segments {
		switch seg {
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		case ".":
		default:
			out = append(out, seg)
		}
	}
	if last := segments[len(segments)-1]; last == "." || last == ".." {
		out = append(out, "")
	}
	r.path = strings.Join(out, "/")
	if r.path == "" {
		r.path = "/"
	}
	return r.String(), nil
}

// Join resolves ref against base (RFC 3986 §5). A malformed URL yields
// ref unchanged.
func Join(base, ref string) string {
	s, err := resolve(base, ref)
	if err != nil {
		return ref
	}
	return s
}

// IRIToURI turns an IRI into an ASCII-only URI by percent-encoding every
// byte that RFC 3986 does not allow, as UTF-8 (RFC 3987 §3.1). Reserved
// characters and existing percent escapes are left alone, so the result of
// a second call is unchanged. data: URLs are returned as they are.
func IRIToURI(url string) string {
	if strings.HasPrefix(url, "data:") {
		return url
	}
	var b strings.Builder
	for i := 0; i < len(url); i++ {
		c := url[i]
		if isURISafe(c) {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func isURISafe(c byte) bool {
	switch {
	case isASCIILetter(c), '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("_.-~/:?#[]@!$&'()*+,;=%", c) >= 0
}

// JoinChecked resolves ref against base and converts the result to a URI.
// A relative reference with no base is kept as it is when allowRelative is
// set, and otherwise reported as an error mentioning context; ok is false
// then.
func JoinChecked(base, ref string, allowRelative bool, context string) (uri string, ok bool) {
	switch {
	case IsAbsolute(ref):
		return IRIToURI(ref), true
	case base != "":
		return IRIToURI(Join(base, ref)), true
	case allowRelative:
		return IRIToURI(ref), true
	}
	logging.Errorf("Relative URI reference without a base URI: %s", context)
	return "", false
}

// PathToURL returns the file URL of a filesystem path. A relative path is
// made absolute first, and a path that is a directory, or that ends with a
// separator, gets a trailing slash so that relative references resolve
// inside it.
func PathToURL(path string) string {
	trailing := strings.HasSuffix(path, "/") || strings.HasSuffix(path, string(os.PathSeparator))
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if !trailing {
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			trailing = true
		}
	}
	p := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if trailing && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return "file://" + quotePath(p)
}

func quotePath(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		if isASCIILetter(c) || '0' <= c && c <= '9' || c == '/' || c == '_' || c == '.' || c == '-' || c == '~' {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

// EnsureURL returns s when it is already a URL, and otherwise the file URL
// of the path it names.
func EnsureURL(s string) string {
	if IsAbsolute(s) {
		return s
	}
	return PathToURL(s)
}

// Unquote decodes the percent escapes of s as UTF-8. Malformed escapes are
// kept, and invalid sequences become U+FFFD.
func Unquote(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	b := unquoteBytes(s)
	return strings.ToValidUTF8(string(b), "�")
}

func unquoteBytes(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, ok1 := unhex(s[i+1])
			lo, ok2 := unhex(s[i+2])
			if ok1 && ok2 {
				out = append(out, hi<<4|lo)
				i += 2
				continue
			}
		}
		out = append(out, s[i])
	}
	return out
}

func unhex(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// LinkKind tells whether a Link points inside the document or outside it.
type LinkKind int

const (
	// LinkExternal is a link to a resource identified by a URI.
	LinkExternal LinkKind = iota
	// LinkInternal is a link to an anchor of the same document.
	LinkInternal
)

// Link is the destination of a hyperlink.
type Link struct {
	Kind LinkKind
	// Target is the absolute URI for an external link, and the unquoted
	// fragment identifier for an internal one.
	Target string
}

// GetURLTuple classifies a reference that is either a fragment or something
// resolvable against baseURL. ok is false for a relative reference without
// a base.
func GetURLTuple(url, baseURL string) (link Link, ok bool) {
	switch {
	case strings.HasPrefix(url, "#"):
		return Link{LinkInternal, Unquote(url[1:])}, true
	case IsAbsolute(url):
		return Link{LinkExternal, IRIToURI(url)}, true
	case baseURL != "":
		return Link{LinkExternal, IRIToURI(Join(baseURL, url))}, true
	}
	return Link{}, false
}

// GetLink returns the destination of a link attribute value. A bare
// fragment never needs a base URL, and a URL that differs from baseURL only
// by its fragment is an internal link. The result is nil when the value
// cannot be resolved.
func GetLink(url, baseURL string) *Link {
	if strings.HasPrefix(url, "#") && len(url) > 1 {
		return &Link{LinkInternal, Unquote(url[1:])}
	}
	uri, ok := JoinChecked(baseURL, url, true, url)
	if !ok || uri == "" {
		return nil
	}
	if baseURL != "" {
		parsed, err := split(uri)
		if err != nil {
			logging.Warnf("Malformed URL: %s", uri)
		} else if parsedBase, err := split(baseURL); err != nil {
			logging.Warnf("Malformed base URL: %s", baseURL)
		} else if parsed.fragment != "" && parsed.withoutFragment() == parsedBase.withoutFragment() {
			return &Link{LinkInternal, Unquote(parsed.fragment)}
		}
	}
	return &Link{LinkExternal, uri}
}

func (u splitURL) withoutFragment() splitURL {
	u.fragment = ""
	return u
}
