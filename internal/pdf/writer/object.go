package writer

import (
	"bytes"
	"compress/zlib"
	"math"
	"math/big"
	"strconv"
	"unicode/utf16"
)

// Object is a PDF object that can be placed in a dictionary, an array or the
// object table of a [PDF].
type Object interface {
	// appendTo serializes the object. compress is the document-wide default
	// for streams that do not carry their own setting.
	appendTo(dst []byte, compress bool) []byte
}

type (
	// Null is the PDF null object.
	Null struct{}
	// Bool is a PDF boolean.
	Bool bool
	// Int is a PDF integer.
	Int int64
	// Real is a PDF real number. Integral values are written without a
	// fractional part; others with at most six decimals and no trailing zeros.
	Real float64
	// Name is a PDF name without the leading slash. Bytes that are not
	// regular characters are written as #xx (ISO 32000-1 §7.3.5).
	Name string
	// Ref is an indirect reference to the object with the given number. The
	// generation number is always zero.
	Ref struct{ Num int }
	// Array is a PDF array.
	Array []Object
)

func (Null) appendTo(dst []byte, _ bool) []byte { return append(dst, "null"...) }

func (b Bool) appendTo(dst []byte, _ bool) []byte {
	return strconv.AppendBool(dst, bool(b))
}

func (i Int) appendTo(dst []byte, _ bool) []byte { return strconv.AppendInt(dst, int64(i), 10) }

func (r Real) appendTo(dst []byte, _ bool) []byte { return appendReal(dst, float64(r)) }

func (n Name) appendTo(dst []byte, _ bool) []byte { return appendName(dst, string(n)) }

func (r Ref) appendTo(dst []byte, _ bool) []byte {
	dst = strconv.AppendInt(dst, int64(r.Num), 10)
	return append(dst, " 0 R"...)
}

func (a Array) appendTo(dst []byte, compress bool) []byte {
	dst = append(dst, '[')
	for i, o := range a {
		if i > 0 {
			dst = append(dst, ' ')
		}
		dst = appendObject(dst, o, compress)
	}
	return append(dst, ']')
}

// appendObject serializes o, writing a nil Object as null.
func appendObject(dst []byte, o Object, compress bool) []byte {
	if o == nil {
		return append(dst, "null"...)
	}
	return o.appendTo(dst, compress)
}

// appendReal writes f with up to six decimals and trailing zeros stripped.
// Values that round to zero keep the sign and a bare decimal point ("0.",
// "-0."), which readers accept as zero.
func appendReal(dst []byte, f float64) []byte {
	switch {
	case math.IsNaN(f):
		return append(dst, "nan"...)
	case math.IsInf(f, 1):
		return append(dst, "inf"...)
	case math.IsInf(f, -1):
		return append(dst, "-inf"...)
	case f == math.Trunc(f):
		if math.Abs(f) < 1<<63 {
			return strconv.AppendInt(dst, int64(f), 10)
		}
		return append(dst, new(big.Float).SetFloat64(f).Text('f', 0)...)
	}
	s := strconv.AppendFloat(nil, f, 'f', 6, 64)
	return append(dst, bytes.TrimRight(s, "0")...)
}

func appendName(dst []byte, s string) []byte {
	dst = append(dst, '/')
	const hex = "0123456789ABCDEF"
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= ' ' || c >= 0x7f || isDelimiter(c) {
			dst = append(dst, '#', hex[c>>4], hex[c&15])
			continue
		}
		dst = append(dst, c)
	}
	return dst
}

func isDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%', '#':
		return true
	}
	return false
}

// String is a PDF string, written either as a literal "(...)" or as a
// hexadecimal "<...>" string.
type String struct {
	b   []byte
	hex bool
}

// Literal returns a literal string holding b. Only backslash and
// parentheses are escaped (ISO 32000-1 §7.3.4.2).
func Literal(b []byte) String { return String{b: b} }

// Hex returns a hexadecimal string holding b.
func Hex(b []byte) String { return String{b: b, hex: true} }

// Text returns a text string (ISO 32000-1 §7.9.2.2): a literal string when s
// is pure ASCII, otherwise a hexadecimal string with a UTF-16BE byte order
// mark and encoding.
func Text(s string) String {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return String{b: []byte(s)}
	}
	units := utf16.Encode([]rune(s))
	b := make([]byte, 0, 2+2*len(units))
	b = append(b, 0xfe, 0xff)
	for _, u := range units {
		b = append(b, byte(u>>8), byte(u))
	}
	return String{b: b, hex: true}
}

// Bytes returns the raw bytes held by the string.
func (s String) Bytes() []byte { return s.b }

func (s String) appendTo(dst []byte, _ bool) []byte {
	if s.hex {
		const hex = "0123456789abcdef"
		dst = append(dst, '<')
		for _, c := range s.b {
			dst = append(dst, hex[c>>4], hex[c&15])
		}
		return append(dst, '>')
	}
	dst = append(dst, '(')
	for _, c := range s.b {
		if c == '\\' || c == '(' || c == ')' {
			dst = append(dst, '\\')
		}
		dst = append(dst, c)
	}
	return append(dst, ')')
}

// Stream is a PDF stream object. Length is always computed, and Filter is
// set to FlateDecode when the stream is compressed.
type Stream struct {
	// Dict holds the extra stream dictionary entries; it may be nil.
	Dict *Dict
	// Data is the stream content before any compression.
	Data []byte
	// Compress overrides [WriteOptions].Compress for this stream when non-nil.
	Compress *bool
}

func (s *Stream) appendTo(dst []byte, compress bool) []byte {
	if s.Compress != nil {
		compress = *s.Compress
	}
	data := s.Data
	d := s.Dict.clone()
	if compress {
		d.Set("Filter", Name("FlateDecode"))
		data = deflate(data)
	}
	d.Set("Length", Int(len(data)))
	dst = d.appendTo(dst, compress)
	dst = append(dst, "\nstream\n"...)
	dst = append(dst, data...)
	return append(dst, "\nendstream"...)
}

// deflate compresses b with a fixed level so the output does not depend on
// the caller.
func deflate(b []byte) []byte {
	var buf bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

// AppendReal appends f formatted as a PDF real number: integral values
// without a fractional part, others with at most six decimals and no
// trailing zeros. Content stream operands use the same format.
func AppendReal(dst []byte, f float64) []byte { return appendReal(dst, f) }

// Append appends the serialization of o to dst, with streams left
// uncompressed. A nil o is written as null.
func Append(dst []byte, o Object) []byte { return appendObject(dst, o, false) }
