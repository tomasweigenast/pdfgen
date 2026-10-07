package html

import (
	"bytes"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

// metaPrescanLimit is the number of bytes searched for an encoding
// declaration (HTML Living Standard §13.2.3.2).
const metaPrescanLimit = 1024

var bomEncodings = []struct {
	mark string
	name string
	enc  encoding.Encoding
}{
	{"\xef\xbb\xbf", "utf-8", unicode.UTF8},
	{"\xff\xfe\x00\x00", "utf-32le", utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)},
	{"\x00\x00\xfe\xff", "utf-32be", utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)},
	{"\xff\xfe", "utf-16le", unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)},
	{"\xfe\xff", "utf-16be", unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)},
}

// DetectEncoding picks the character encoding of an HTML byte stream and
// returns it with its canonical name, along with the number of leading
// bytes taken by a byte order mark.
//
// In order of precedence: a byte order mark, the override, the encoding the
// transport announced, an encoding declared by a meta element in the first
// 1024 bytes, and finally windows-1252. Labels that are not known encodings
// are skipped. The tentative guesses of the HTML Living Standard that need a
// second parse, such as a declaration found after the prescan window, are
// not followed.
func DetectEncoding(data []byte, override, transport string) (enc encoding.Encoding, name string, bomLen int) {
	for _, b := range bomEncodings {
		if bytes.HasPrefix(data, []byte(b.mark)) {
			return b.enc, b.name, len(b.mark)
		}
	}
	for _, label := range []string{override, transport} {
		if label == "" {
			continue
		}
		if enc, name := charset.Lookup(label); enc != nil {
			return enc, name, 0
		}
	}
	enc, name, _ = charset.DetermineEncoding(data[:min(len(data), metaPrescanLimit)], "")
	return enc, name, 0
}

// decode converts an HTML byte stream to UTF-8. Bytes that are invalid in
// the detected encoding become U+FFFD.
func decode(data []byte, override, transport string) (text, name string, err error) {
	enc, name, bomLen := DetectEncoding(data, override, transport)
	out, err := enc.NewDecoder().Bytes(data[bomLen:])
	if err != nil {
		return "", name, err
	}
	return string(out), name, nil
}
