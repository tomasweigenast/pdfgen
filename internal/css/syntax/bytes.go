package syntax

import (
	"bytes"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

// Encoding is a character encoding identified by its WHATWG name, such as
// "utf-8" or "windows-1252".
type Encoding struct {
	// Name is the canonical lower-case name.
	Name string
	enc  encoding.Encoding
}

// UTF8 is the UTF-8 encoding, the fallback of stylesheet decoding.
var UTF8 = LookupEncoding("utf-8")

// LookupEncoding returns the encoding for a label, or nil when the label is
// unknown. Surrounding ASCII whitespace and letter case are ignored.
func LookupEncoding(label string) *Encoding {
	label = AsciiLower(strings.Trim(label, "\t\n\f\r "))
	enc, err := htmlindex.Get(label)
	if err != nil {
		return nil
	}
	name, err := htmlindex.Name(enc)
	if err != nil {
		return nil
	}
	return &Encoding{Name: name, enc: enc}
}

// DecodeStylesheetBytes decodes a stylesheet and returns the encoding used.
// The encoding is taken, in order, from a byte order mark, the protocol
// encoding label, an @charset rule, the environment encoding, and finally
// UTF-8 (CSS Syntax 3 §3.2). Invalid sequences become U+FFFD.
func DecodeStylesheetBytes(css []byte, protocolEncoding string, environment *Encoding) (string, *Encoding) {
	if protocolEncoding != "" {
		if enc := LookupEncoding(protocolEncoding); enc != nil {
			return decodeWithBOM(css, enc)
		}
	}
	if bytes.HasPrefix(css, []byte(`@charset "`)) {
		// The label is searched in a bounded window so that a missing quote
		// does not scan the whole stylesheet.
		const start, limit = 10, 100
		if end := bytes.IndexByte(css[start:min(limit, len(css))], '"'); end != -1 {
			end += start
			if bytes.HasPrefix(css[end:], []byte(`";`)) {
				label := make([]rune, 0, end-start)
				for _, b := range css[start:end] {
					label = append(label, rune(b))
				}
				if enc := LookupEncoding(string(label)); enc != nil {
					if enc.Name == "utf-16be" || enc.Name == "utf-16le" {
						return decodeWithBOM(css, UTF8)
					}
					return decodeWithBOM(css, enc)
				}
			}
		}
	}
	if environment != nil {
		return decodeWithBOM(css, environment)
	}
	return decodeWithBOM(css, UTF8)
}

// decodeWithBOM decodes b with fallback, unless a byte order mark names
// another encoding, which then wins; the mark is removed.
func decodeWithBOM(b []byte, fallback *Encoding) (string, *Encoding) {
	enc := fallback
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		enc, b = UTF8, b[3:]
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		enc, b = LookupEncoding("utf-16be"), b[2:]
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		enc, b = LookupEncoding("utf-16le"), b[2:]
	}
	out, _ := enc.enc.NewDecoder().Bytes(b)
	return string(out), enc
}

// ParseStylesheetBytes decodes a stylesheet as browsers do (see
// DecodeStylesheetBytes) and parses it. The returned encoding is also the
// environment encoding of any stylesheet imported by an @import rule.
func ParseStylesheetBytes(css []byte, protocolEncoding string, environment *Encoding,
	skipComments, skipWhitespace bool) ([]Node, *Encoding) {
	text, enc := DecodeStylesheetBytes(css, protocolEncoding, environment)
	return ParseStylesheet(text, skipComments, skipWhitespace), enc
}
