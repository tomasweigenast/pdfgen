package writer

import (
	"bytes"
	"compress/zlib"
	"crypto/md5"
	"encoding/hex"
	"io"
	"regexp"
	"strings"
	"testing"
)

func write(t *testing.T, p *PDF, o WriteOptions) string {
	t.Helper()
	var buf bytes.Buffer
	if err := p.Write(&buf, o); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return buf.String()
}

func TestNoIdentifier(t *testing.T) {
	out := write(t, New(), WriteOptions{OmitID: true})
	if strings.Contains(out, "/ID") {
		t.Errorf("unexpected /ID in %q", out)
	}
}

func TestDefaultIdentifierIsDocumentDigest(t *testing.T) {
	out := write(t, New(), WriteOptions{})
	m := regexp.MustCompile(`/ID \[\(([0-9a-f]{32})\) \(([0-9a-f]{32})\)\]`).FindStringSubmatch(out)
	if m == nil || m[1] != m[2] {
		t.Fatalf("/ID not found or halves differ in %q", out)
	}
	objects := out[strings.Index(out, "1 0 obj"):strings.Index(out, "xref")]
	var data []string
	for _, chunk := range strings.Split(objects, "endobj\n") {
		if i := strings.Index(chunk, "obj\n"); i >= 0 {
			data = append(data, strings.TrimSuffix(chunk[i+4:], "\n"))
		}
	}
	sum := md5.Sum([]byte(strings.Join(data, "")))
	if got := hex.EncodeToString(sum[:]); got != m[1] {
		t.Errorf("digest = %s, want %s", m[1], got)
	}
}

func TestCustomIdentifier(t *testing.T) {
	out := write(t, New(), WriteOptions{Identifier: []byte("abc")})
	if !regexp.MustCompile(`/ID \[\(abc\) \([0-9a-f]{32}\)\]`).MatchString(out) {
		t.Errorf("custom identifier not found in %q", out)
	}
}

func TestHeaderVersion(t *testing.T) {
	tests := []struct{ version, header string }{
		{"", "%PDF-1.7\n"},
		{"2.0", "%PDF-2.0\n"},
		{"1.4", "%PDF-1.4\n"},
	}
	for _, tt := range tests {
		out := write(t, New(), WriteOptions{Version: tt.version})
		if !strings.HasPrefix(out, tt.header+"%\xf0\x9f\x96\xa4\n") {
			t.Errorf("version %q: header = %q", tt.version, out[:20])
		}
	}
}

func TestObjectsAreNumberedInCreationOrder(t *testing.T) {
	p := New()
	a := p.Add(Int(10))
	b := p.Add(Int(20))
	if a.Num != 3 || b.Num != 4 {
		t.Fatalf("refs = %v, %v; want 3 and 4", a, b)
	}
	out := write(t, p, WriteOptions{OmitID: true})
	if !strings.Contains(out, "3 0 obj\n10\nendobj\n4 0 obj\n20\nendobj\n") {
		t.Errorf("objects not in order: %q", out)
	}
}

func TestUnreachableObjectsAreWritten(t *testing.T) {
	p := New()
	p.Add(Text("orphan"))
	if out := write(t, p, WriteOptions{}); !strings.Contains(out, "(orphan)") {
		t.Error("unreferenced object missing from output")
	}
}

func TestReserveAndSet(t *testing.T) {
	p := New()
	r := p.Reserve()
	p.Add(NewDict("Refers", r))
	if err := p.Write(io.Discard, WriteOptions{}); err == nil {
		t.Fatal("writing with an unset reserved object should fail")
	}
	if err := p.Set(r, Int(5)); err != nil {
		t.Fatal(err)
	}
	out := write(t, p, WriteOptions{OmitID: true})
	if !strings.Contains(out, "3 0 obj\n5\nendobj\n4 0 obj\n<</Refers 3 0 R>>\nendobj\n") {
		t.Errorf("unexpected output %q", out)
	}
}

func TestSetRejectsUnknownObjects(t *testing.T) {
	p := New()
	for _, num := range []int{-1, 0, 1, 2, 3, 99} {
		if err := p.Set(Ref{Num: num}, Null{}); err == nil {
			t.Errorf("Set(%d) succeeded", num)
		}
	}
}

func TestAddPageUpdatesTree(t *testing.T) {
	p := New()
	page := NewDict("Type", Name("Page"), "MediaBox", Array{Int(0), Int(0), Int(1), Int(1)})
	r1 := p.AddPage(page)
	r2 := p.AddPage(NewDict("Type", Name("Page"), "Parent", Ref{Num: 99}))

	refs := p.PageRefs()
	if len(refs) != 2 || refs[0] != r1 || refs[1] != r2 {
		t.Errorf("PageRefs = %v, want [%v %v]", refs, r1, r2)
	}
	if got := serialize(p.Pages()); got != "<</Type /Pages/Kids [3 0 R 4 0 R]/Count 2>>" {
		t.Errorf("pages = %s", got)
	}
	if got := serialize(page); got != "<</Type /Page/MediaBox [0 0 1 1]/Parent 1 0 R>>" {
		t.Errorf("page = %s", got)
	}
	if v, _ := p.Object(r2); !strings.HasSuffix(serialize(v), "/Parent 1 0 R>>") {
		t.Errorf("Parent not normalized: %s", serialize(v))
	}
}

func TestInitialObjects(t *testing.T) {
	out := write(t, New(), WriteOptions{OmitID: true})
	want := "1 0 obj\n<</Type /Pages/Kids []/Count 0>>\nendobj\n" +
		"2 0 obj\n<</Type /Catalog/Pages 1 0 R>>\nendobj\n"
	if !strings.Contains(out, want) {
		t.Errorf("got %q", out)
	}
}

func TestInfoIsWrittenLastAndOnlyWhenNotEmpty(t *testing.T) {
	p := New()
	if out := write(t, p, WriteOptions{}); strings.Contains(out, "/Info") {
		t.Error("empty info dictionary was written")
	}
	p.Info().Set("Title", Text("T"))
	p.Add(Int(1))
	out := write(t, p, WriteOptions{OmitID: true})
	if !strings.Contains(out, "4 0 obj\n<</Title (T)>>\nendobj\nxref\n0 5\n") ||
		!strings.Contains(out, "/Info 4 0 R\n") {
		t.Errorf("info not last: %q", out)
	}
}

func TestClassicXrefOffsetsPointToObjects(t *testing.T) {
	p := New()
	p.Add(&Stream{Data: []byte("abc")})
	out := write(t, p, WriteOptions{})
	table := out[strings.Index(out, "xref\n"):]
	rows := regexp.MustCompile(`(\d{10}) (\d{5}) ([nf]) \n`).FindAllStringSubmatch(table, -1)
	if len(rows) != 4 || rows[0][3] != "f" || rows[0][2] != "65535" {
		t.Fatalf("rows = %v", rows)
	}
	for i, row := range rows[1:] {
		var off int
		for _, c := range row[1] {
			off = off*10 + int(c-'0')
		}
		if !strings.HasPrefix(out[off:], strings.Join([]string{string(rune('1' + i)), "0 obj"}, " ")) {
			t.Errorf("object %d offset %d points at %q", i+1, off, out[off:off+10])
		}
	}
	if !strings.HasSuffix(out, "startxref\n"+itoa(strings.Index(out, "xref\n"))+"\n%%EOF\n") {
		t.Errorf("bad tail %q", out[len(out)-30:])
	}
}

func itoa(n int) string { return string(appendObject(nil, Int(n), false)) }

func TestCompressedLayout(t *testing.T) {
	p := New()
	p.Add(&Stream{Data: []byte("abc"), Compress: ptr(false)})
	p.Info().Set("Title", Text("T"))
	out := write(t, p, WriteOptions{Compress: true, OmitID: true})

	if strings.Contains(out, "\nxref\n") || strings.Contains(out, "\ntrailer\n") {
		t.Error("classic xref written in compressed mode")
	}
	m := regexp.MustCompile(`(?s)6 0 obj\n<</Type /XRef/Index \[0 7\]/W \[1 (\d) (\d)\]/Size 7/Root 2 0 R/Info 4 0 R/Filter /FlateDecode/Length (\d+)>>\nstream\n`).
		FindStringSubmatchIndex(out)
	if m == nil {
		t.Fatalf("xref stream dictionary not found in %q", out)
	}
	objStm := regexp.MustCompile(`(?s)5 0 obj\n<</Type /ObjStm/N 3/First (\d+)/Filter /FlateDecode/Length (\d+)>>\nstream\n`).
		FindStringSubmatchIndex(out)
	if objStm == nil {
		t.Fatalf("object stream not found in %q", out)
	}
	n := atoi(out[objStm[4]:objStm[5]])
	zr, err := zlib.NewReader(strings.NewReader(out[objStm[1] : objStm[1]+n]))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	want := "1 0 2 33 4 64\n<</Type /Pages/Kids []/Count 0>>\n<</Type /Catalog/Pages 1 0 R>>\n<</Title (T)>>"
	if string(body) != want {
		t.Errorf("object stream = %q, want %q", body, want)
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
