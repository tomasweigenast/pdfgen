package html

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"
)

func utf16Bytes(s string, bigEndian bool, bom bool) []byte {
	var out []byte
	put := func(u uint16) {
		if bigEndian {
			out = append(out, byte(u>>8), byte(u))
		} else {
			out = append(out, byte(u), byte(u>>8))
		}
	}
	if bom {
		put(0xFEFF)
	}
	for _, u := range utf16.Encode([]rune(s)) {
		put(u)
	}
	return out
}

func latin1(s string) []byte {
	b, err := charmap.ISO8859_1.NewEncoder().String(s)
	if err != nil {
		panic(err)
	}
	return []byte(b)
}

func TestDetectEncoding(t *testing.T) {
	const page = "<title>é</title>"
	tests := []struct {
		name                string
		data                []byte
		override, transport string
		wantName            string
		wantTitle           string
	}{
		{"UTF-8 BOM", append([]byte("\xef\xbb\xbf"), page...), "", "", "utf-8", "é"},
		{"UTF-16LE BOM", utf16Bytes(page, false, true), "", "", "utf-16le", "é"},
		{"UTF-16BE BOM", utf16Bytes(page, true, true), "", "", "utf-16be", "é"},
		{"BOM beats override and transport", utf16Bytes(page, true, true), "latin1", "latin1", "utf-16be", "é"},
		{"override", latin1(page), "iso-8859-1", "utf-8", "windows-1252", "é"},
		{"override without BOM", utf16Bytes(page, true, false), "UTF-16BE", "", "utf-16be", "é"},
		{"override beats meta", latin1(`<meta charset=utf-8>` + page), "latin1", "", "windows-1252", "é"},
		{"transport", latin1(page), "", "ISO-8859-1", "windows-1252", "é"},
		{"transport beats meta", []byte(`<meta charset=latin1>` + page), "", "utf-8", "utf-8", "é"},
		{"unknown override falls through to transport", latin1(page), "no-such-encoding", "latin1", "windows-1252", "é"},
		{"unknown transport falls through to meta", []byte(`<meta charset=utf-8>` + page), "", "bogus", "utf-8", "é"},
		{"meta charset", []byte(`<meta charset=utf-8>` + page), "", "", "utf-8", "é"},
		{"meta http-equiv", latin1(`<meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1">` + page), "", "", "windows-1252", "é"},
		{"meta utf-16 means utf-8", []byte(`<meta charset=utf-16>` + page), "", "", "utf-8", "é"},
		{"default", latin1(page), "", "", "windows-1252", "é"},
		{"default decodes windows-1252 extensions", []byte("<title>\x80</title>"), "", "", "windows-1252", "€"},
		{"meta beyond the prescan window is ignored", append(append(latin1(strings.Repeat(" ", 1100)), `<meta charset=utf-8>`...), latin1(page)...), "", "", "windows-1252", "é"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(bytes.NewReader(tt.data), tt.override, "", WithTransportEncoding(tt.transport))
			if err != nil {
				t.Fatal(err)
			}
			if doc.Encoding != tt.wantName {
				t.Errorf("encoding = %q, want %q", doc.Encoding, tt.wantName)
			}
			var title string
			for _, e := range doc.Elements() {
				if e.LocalName() == "title" {
					title = e.ChildText()
				}
			}
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
		})
	}
}

func TestDetectEncodingBOMLength(t *testing.T) {
	tests := []struct {
		data string
		name string
		bom  int
	}{
		{"\xef\xbb\xbfx", "utf-8", 3},
		{"\xff\xfe\x00\x00x\x00\x00\x00", "utf-32le", 4},
		{"\x00\x00\xfe\xff\x00\x00\x00x", "utf-32be", 4},
		{"\xff\xfex\x00", "utf-16le", 2},
		{"\xfe\xff\x00x", "utf-16be", 2},
		{"x", "windows-1252", 0},
	}
	for _, tt := range tests {
		_, name, bom := DetectEncoding([]byte(tt.data), "", "")
		if name != tt.name || bom != tt.bom {
			t.Errorf("%q: %s, %d; want %s, %d", tt.data, name, bom, tt.name, tt.bom)
		}
	}
}

func TestParseUTF32(t *testing.T) {
	data := []byte("\xff\xfe\x00\x00")
	for _, r := range "<title>ü</title>" {
		data = append(data, byte(r), byte(r>>8), byte(r>>16), byte(r>>24))
	}
	doc, err := Parse(bytes.NewReader(data), "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range doc.Elements() {
		if e.LocalName() == "title" && e.ChildText() != "ü" {
			t.Errorf("title = %q", e.ChildText())
		}
	}
}

func TestParseInvalidBytesBecomeReplacementCharacters(t *testing.T) {
	doc, err := Parse(strings.NewReader("<title>a\xffb</title>"), "utf-8", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range doc.Elements() {
		if e.LocalName() == "title" && e.ChildText() != "a�b" {
			t.Errorf("title = %q", e.ChildText())
		}
	}
}

func TestParseStringHasNoEncoding(t *testing.T) {
	doc, err := ParseString("<meta charset=latin1><title>é</title>", "")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Encoding != "" {
		t.Errorf("Encoding = %q", doc.Encoding)
	}
	for _, e := range doc.Elements() {
		if e.LocalName() == "title" && e.ChildText() != "é" {
			t.Errorf("title = %q", e.ChildText())
		}
	}
}
