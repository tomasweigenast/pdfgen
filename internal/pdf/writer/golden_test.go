package writer

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func lines(items ...string) []byte { return []byte(strings.Join(items, "\n")) }

func ptr[T any](v T) *T { return &v }

func pageDict(pages Ref, contents Ref, extra ...any) *Dict {
	d := NewDict(
		"Type", Name("Page"),
		"Parent", pages,
		"Contents", contents,
		"MediaBox", Array{Int(0), Int(0), Int(10), Int(10)},
	)
	for i := 0; i < len(extra); i += 2 {
		d.Set(Name(extra[i].(string)), extra[i+1].(Object))
	}
	return d
}

type goldenCase struct {
	name string
	// exact is false when the output contains Flate data, whose bytes
	// depend on the deflate implementation.
	exact bool
	// valid is false for documents that are intentionally degenerate.
	valid bool
	build func() (*PDF, WriteOptions)
}

var goldenCases = []goldenCase{
	{"empty", true, false, func() (*PDF, WriteOptions) {
		return New(), WriteOptions{}
	}},
	{"single_page", true, true, func() (*PDF, WriteOptions) {
		p := New()
		draw := p.Add(&Stream{Data: lines("2 2 5 6 re", "f")})
		p.AddPage(pageDict(p.PagesRef(), draw))
		return p, WriteOptions{OmitID: true}
	}},
	{"metadata_text", true, true, func() (*PDF, WriteOptions) {
		p := New()
		info := p.Info()
		info.Set("Title", Text("Héllo (wörld)"))
		info.Set("Author", Text(`a\b)`))
		info.Set("Producer", Text("gen"))
		info.Set("Keywords", Text("😀 ♡"))
		font := p.Add(NewDict(
			"Type", Name("Font"), "Subtype", Name("Type1"), "Name", Name("F1"),
			"BaseFont", Name("Helvetica"), "Encoding", Name("MacRomanEncoding")))
		draw := p.Add(&Stream{Data: lines(
			"BT", "/F1 12.5 Tf", "1 0 0 1 20.25 5 Tm", "(Hello \\(there\\)) Tj", "ET")})
		for range 2 {
			p.AddPage(pageDict(p.PagesRef(), draw, "Resources", NewDict(
				"ProcSet", Array{Name("PDF"), Name("Text")},
				"Font", NewDict("F1", font))))
		}
		return p, WriteOptions{Version: "1.4", Identifier: []byte("custom id")}
	}},
	{"numbers", true, false, func() (*PDF, WriteOptions) {
		p := New()
		values := Array{Int(0), Int(-7), Real(0.5), Real(100), Real(595.2756), Real(-0.0001),
			Real(1e-7), Real(-1e-7), Real(3.14159265), Real(123456.7890123), Real(1e20),
			Real(0.1 + 0.2), Real(0), Real(2)}
		nested := p.Add(NewDict(
			"Numbers", values,
			"Inner", NewDict(
				"A", Array{Array{}, Array{Int(1), Real(2.5)}},
				"B", Name("Name"), "C", NewDict())))
		p.Add(Array{nested, Int(1), Real(2.25)})
		p.Add(Text("x"))
		p.Add(NewDict("Ref", p.PagesRef()))
		return p, WriteOptions{}
	}},
	{"streams", true, true, func() (*PDF, WriteOptions) {
		p := New()
		form := p.Add(&Stream{
			Dict: NewDict("Type", Name("XObject"), "Subtype", Name("Form"),
				"BBox", Array{Int(0), Int(0), Int(1), Real(1.5)}),
			Data: lines("q", "0.75 0 0 -0.75 0 841.8898 cm", "1.5 2 m",
				"1 2 3 4.5 5 6 c", "[2 1.5] 0 d", "S", "Q"),
		})
		image := p.Add(&Stream{
			Dict: NewDict("Type", Name("XObject"), "Subtype", Name("Image"),
				"Width", 5, "Height", 1, "ColorSpace", Name("DeviceGray"),
				"BitsPerComponent", 8, "Filter", Name("DCTDecode")),
			Data: []byte("\x00\xff\x10\n\x80"),
		})
		p.Add(&Stream{})
		p.AddPage(pageDict(p.PagesRef(), form, "Resources", NewDict(
			"XObject", NewDict("Im1", image))))
		return p, WriteOptions{Version: "1.7", OmitID: true}
	}},
	{"forward_refs", true, false, func() (*PDF, WriteOptions) {
		p := New()
		first := p.Reserve()
		second := p.Add(NewDict("Back", first))
		_ = p.Set(first, NewDict("Next", second, "Self", first))
		p.Info().Set("Title", Text("refs"))
		return p, WriteOptions{}
	}},
	{"big_classic", true, true, func() (*PDF, WriteOptions) {
		p := New()
		big := p.Add(&Stream{Data: bytes.Repeat(allBytes(), 300)})
		p.AddPage(pageDict(p.PagesRef(), big))
		return p, WriteOptions{Version: "2.0", Identifier: []byte("(id)")}
	}},
	{"compressed_objstm", false, true, func() (*PDF, WriteOptions) {
		p := New()
		var ops []string
		for i := range 20 {
			ops = append(ops, strconv.Itoa(i)+" "+strconv.Itoa(i)+" 5 6 re", "f")
		}
		draw := p.Add(&Stream{Data: lines(ops...), Compress: ptr(false)})
		raw := p.Add(&Stream{Data: []byte("plain data"), Compress: ptr(false)})
		p.Info().Set("Title", Text("Compressed é"))
		p.AddPage(pageDict(p.PagesRef(), draw))
		p.AddPage(pageDict(p.PagesRef(), raw))
		return p, WriteOptions{Version: "1.7", Compress: true}
	}},
	{"compressed_wide", false, true, func() (*PDF, WriteOptions) {
		p := New()
		big := p.Add(&Stream{Data: bytes.Repeat(allBytes(), 300), Compress: ptr(false)})
		p.AddPage(pageDict(p.PagesRef(), big))
		p.Info().Set("Title", Text("wide"))
		return p, WriteOptions{Version: "1.5", Identifier: []byte("wide-id"), Compress: true}
	}},
	{"compress_flag_old_version", false, true, func() (*PDF, WriteOptions) {
		p := New()
		draw := p.Add(&Stream{Data: lines("1 2 3 4 re", "f"), Compress: ptr(false)})
		p.AddPage(pageDict(p.PagesRef(), draw))
		return p, WriteOptions{Version: "1.4", OmitID: true, Compress: true}
	}},
}

func allBytes() []byte {
	b := make([]byte, 256)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func render(t *testing.T, c goldenCase) []byte {
	t.Helper()
	p, opts := c.build()
	var buf bytes.Buffer
	if err := p.Write(&buf, opts); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return buf.Bytes()
}

func TestOutputMatchesGolden(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", c.name+".pdf"))
			if err != nil {
				t.Fatal(err)
			}
			got := render(t, c)
			if c.exact {
				if !bytes.Equal(got, want) {
					t.Errorf("output differs from golden at byte %d\n got: %q\nwant: %q",
						firstDiff(got, want), excerpt(got, firstDiff(got, want)), excerpt(want, firstDiff(got, want)))
				}
				return
			}
			if g, w := normalize(t, got), normalize(t, want); !bytes.Equal(g, w) {
				t.Errorf("normalized output differs from golden at byte %d\n got: %q\nwant: %q",
					firstDiff(g, w), excerpt(g, firstDiff(g, w)), excerpt(w, firstDiff(g, w)))
			}
		})
	}
}

func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func excerpt(b []byte, at int) []byte {
	return b[max(0, at-40):min(len(b), at+40)]
}

var (
	flateStream = regexp.MustCompile(`(?s)(<<[^>]*?)/Length (\d+)>>\nstream\n`)
	idEntry     = regexp.MustCompile(`/ID \[\([^\n]*?\) \([0-9a-f]{32}\)\]`)
	startxref   = regexp.MustCompile(`startxref\n\d+\n`)
	widthEntry  = regexp.MustCompile(`/W \[1 (\d+) (\d+)\]`)
)

// normalize replaces every Flate stream by its inflated content and masks
// what legitimately depends on the deflate output: stream lengths, the
// trailing cross-reference entry and the file digest. The remaining bytes,
// including every offset that precedes the object stream, must match.
func normalize(t *testing.T, pdf []byte) []byte {
	t.Helper()
	var out []byte
	for {
		loc := flateStream.FindSubmatchIndex(pdf)
		if loc == nil {
			break
		}
		dict := string(pdf[loc[2]:loc[3]])
		n, _ := strconv.Atoi(string(pdf[loc[4]:loc[5]]))
		body := pdf[loc[1] : loc[1]+n]
		out = append(out, pdf[:loc[2]]...)
		out = append(out, dict...)
		if strings.Contains(dict, "/FlateDecode") {
			zr, err := zlib.NewReader(bytes.NewReader(body))
			if err != nil {
				t.Fatalf("inflate: %v", err)
			}
			raw, err := io.ReadAll(zr)
			if err != nil {
				t.Fatalf("inflate: %v", err)
			}
			if m := widthEntry.FindStringSubmatch(dict); m != nil && strings.Contains(dict, "/XRef") {
				w2, _ := strconv.Atoi(m[1])
				w3, _ := strconv.Atoi(m[2])
				raw = raw[:len(raw)-(1+w2+w3)]
			}
			out = append(out, "/Length *>>\nstream\n"...)
			out = append(out, raw...)
		} else {
			out = append(out, "/Length *>>\nstream\n"...)
			out = append(out, body...)
		}
		pdf = pdf[loc[1]+n:]
	}
	out = append(out, pdf...)
	out = idEntry.ReplaceAll(out, []byte("/ID *"))
	return startxref.ReplaceAll(out, []byte("startxref\n*\n"))
}

func TestWriteIsDeterministic(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.name, func(t *testing.T) {
			if a, b := render(t, c), render(t, c); !bytes.Equal(a, b) {
				t.Error("two builds of the same document differ")
			}
		})
	}
}

func TestWriteTwiceFromSameDocument(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.name, func(t *testing.T) {
			p, opts := c.build()
			var a, b bytes.Buffer
			if err := p.Write(&a, opts); err != nil {
				t.Fatal(err)
			}
			if err := p.Write(&b, opts); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a.Bytes(), b.Bytes()) {
				t.Error("writing the same document twice gives different bytes")
			}
		})
	}
}
