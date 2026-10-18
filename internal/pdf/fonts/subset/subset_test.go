package subset

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func testFont(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(testutil.ResourcePath(t, "testfont.otf"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// systemTrueType returns a TrueType font from the system, or skips.
func systemTrueType(t *testing.T) []byte {
	t.Helper()
	for _, p := range []string{
		"/System/Library/Fonts/Supplemental/Andale Mono.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"C:\\Windows\\Fonts\\arial.ttf",
	} {
		if b, err := os.ReadFile(p); err == nil {
			return b
		}
	}
	t.Skip("no TrueType font found on this system")
	return nil
}

func numGlyphs(t *testing.T, sfnt []byte) int {
	t.Helper()
	tables, _, err := readTables(sfnt, 0)
	if err != nil {
		t.Fatal(err)
	}
	return int(binary.BigEndian.Uint16(tables["maxp"][4:]))
}

func TestSubsetCFFRenumbersGlyphsInGivenOrder(t *testing.T) {
	f, err := Parse(testFont(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsCFF() {
		t.Fatal("test font should be CFF")
	}
	out, err := f.Subset([]uint16{0, 5, 3}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if string(out[:4]) != "OTTO" {
		t.Errorf("version = %q, want OTTO", out[:4])
	}
	if n := numGlyphs(t, out); n != 3 {
		t.Errorf("glyphs = %d, want 3", n)
	}
	tables, _, _ := readTables(out, 0)
	l, err := parseCFFLayout(tables["CFF "])
	if err != nil {
		t.Fatal(err)
	}
	origTables, _, _ := readTables(testFont(t), 0)
	ol, _ := parseCFFLayout(origTables["CFF "])
	if !bytes.Equal(l.items[1], ol.items[5]) || !bytes.Equal(l.items[2], ol.items[3]) {
		t.Error("charstrings are not in the requested order")
	}
}

func TestSubsetIsDeterministic(t *testing.T) {
	for name, data := range map[string][]byte{"cff": testFont(t), "truetype": systemTrueType(t)} {
		t.Run(name, func(t *testing.T) {
			var first []byte
			for i := 0; i < 2; i++ {
				f, err := Parse(data, 0)
				if err != nil {
					t.Fatal(err)
				}
				out, err := f.Subset([]uint16{0, 4, 6, 9}, Options{})
				if err != nil {
					t.Fatal(err)
				}
				if first == nil {
					first = out
				} else if !bytes.Equal(first, out) {
					t.Error("subsets differ between runs")
				}
			}
		})
	}
}

func TestSubsetHeadChecksumIsConsistent(t *testing.T) {
	for name, data := range map[string][]byte{"cff": testFont(t), "truetype": systemTrueType(t)} {
		t.Run(name, func(t *testing.T) {
			f, err := Parse(data, 0)
			if err != nil {
				t.Fatal(err)
			}
			out, err := f.Subset([]uint16{0, 4, 6}, Options{KeepNotdef: true})
			if err != nil {
				t.Fatal(err)
			}
			if sum := checksum(out); sum != 0xB1B0AFBA {
				t.Errorf("whole-file checksum = %#x, want 0xB1B0AFBA", sum)
			}
		})
	}
}

func TestSubsetNotdefIsEmptyUnlessKept(t *testing.T) {
	for name, data := range map[string][]byte{"cff": testFont(t), "truetype": systemTrueType(t)} {
		t.Run(name, func(t *testing.T) {
			f, err := Parse(data, 0)
			if err != nil {
				t.Fatal(err)
			}
			notdefSize := func(opts Options) int {
				out, err := f.Subset([]uint16{0, 4}, opts)
				if err != nil {
					t.Fatal(err)
				}
				tables, _, _ := readTables(out, 0)
				if f.IsCFF() {
					l, err := parseCFFLayout(tables["CFF "])
					if err != nil {
						t.Fatal(err)
					}
					if len(l.items) != 2 {
						t.Fatalf("charstrings = %d, want 2", len(l.items))
					}
					return len(l.items[0])
				}
				loca := tables["loca"]
				if binary.BigEndian.Uint16(tables["head"][50:]) == 0 {
					return int(binary.BigEndian.Uint16(loca[2:])) * 2
				}
				return int(binary.BigEndian.Uint32(loca[4:]))
			}
			empty := notdefSize(Options{})
			kept := notdefSize(Options{KeepNotdef: true})
			if f.IsCFF() {
				if empty != 1 {
					t.Errorf("empty .notdef charstring has %d bytes, want 1", empty)
				}
			} else if empty != 0 {
				t.Errorf("empty .notdef has %d bytes, want 0", empty)
			}
			if kept <= empty {
				t.Errorf("kept .notdef has %d bytes, want more than %d", kept, empty)
			}
		})
	}
}

func TestSubsetAllowsRepeatedNotdef(t *testing.T) {
	f, err := Parse(testFont(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.Subset([]uint16{0, 4, 0, 0}, Options{KeepNotdef: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := numGlyphs(t, out); n != 4 {
		t.Errorf("glyphs = %d, want 4", n)
	}
	tables, _, _ := readTables(out, 0)
	l, err := parseCFFLayout(tables["CFF "])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(l.items[0], l.items[2]) || !bytes.Equal(l.items[0], l.items[3]) || len(l.items[0]) <= 1 {
		t.Error("every copy of .notdef should keep the outline")
	}
}

func TestSubsetRejectsInvalidGlyphLists(t *testing.T) {
	f, err := Parse(testFont(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	for name, glyphs := range map[string][]uint16{
		"empty":            nil,
		"no notdef":        {3, 4},
		"duplicate":        {0, 4, 4},
		"out of range":     {0, 60000},
		"notdef not first": {4, 0},
	} {
		if _, err := f.Subset(glyphs, Options{}); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestSubsetTrueTypeHintingTables(t *testing.T) {
	f, err := Parse(systemTrueType(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	has := func(out []byte, tag string) bool {
		tables, _, _ := readTables(out, 0)
		_, ok := tables[tag]
		return ok
	}
	plain, err := f.Subset([]uint16{0, 4}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	hinted, err := f.Subset([]uint16{0, 4}, Options{Hinting: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"cvt ", "fpgm", "prep"} {
		if has(plain, tag) {
			t.Errorf("%q kept without hinting", tag)
		}
	}
	if _, ok := f.sfnt.Tables["fpgm"]; ok && !has(hinted, "fpgm") {
		t.Error("fpgm dropped with hinting")
	}
}

func TestParseReportsUnsupportedOutlines(t *testing.T) {
	tables := map[string][]byte{"CFF2": {0}, "head": make([]byte, 54)}
	_, err := Parse(writeSFNT(tagOTTO, tables), 0)
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("err = %v, want ErrUnsupported", err)
	}
}

func TestParseEmptyTableDirectoryReturnsAnError(t *testing.T) {
	if _, err := Parse([]byte("0000\x00\x00000000"), 0); err == nil {
		t.Fatal("Parse succeeded for a font with no tables")
	}
}

func TestFullExtractsFaceOfCollection(t *testing.T) {
	data := testFont(t)
	same, err := Full(data, 0)
	if err != nil || !bytes.Equal(same, data) {
		t.Fatalf("a standalone font should be returned unchanged (err %v)", err)
	}

	tables, version, _ := readTables(data, 0)
	standalone := writeSFNT(version, tables)

	// A one-face collection pointing at the same tables.
	var ttc []byte
	ttc = append(ttc, "ttcf"...)
	ttc = binary.BigEndian.AppendUint32(ttc, 0x00010000)
	ttc = binary.BigEndian.AppendUint32(ttc, 1)
	ttc = binary.BigEndian.AppendUint32(ttc, 16)
	shift := uint32(16)
	face := bytes.Clone(standalone)
	n := int(binary.BigEndian.Uint16(face[4:]))
	for i := 0; i < n; i++ {
		rec := face[12+16*i+8:]
		binary.BigEndian.PutUint32(rec, binary.BigEndian.Uint32(rec)+shift)
	}
	ttc = append(ttc, face...)

	got, err := Full(ttc, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(got[:4]) != "OTTO" {
		t.Errorf("extracted face starts with %q", got[:4])
	}
	if _, err := Parse(ttc, 0); err != nil {
		t.Errorf("parsing a collection: %v", err)
	}
	if _, err := Parse(ttc, 1); err == nil {
		t.Error("face 1 of a one-face collection should fail")
	}
}

func TestSubsetTrueTypeDropsGlyphInstructions(t *testing.T) {
	f, err := Parse(systemTrueType(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	instructions := func(opts Options) int {
		out, err := f.Subset([]uint16{0, 4, 5}, opts)
		if err != nil {
			t.Fatal(err)
		}
		tables, _, _ := readTables(out, 0)
		glyf, loca := tables["glyf"], tables["loca"]
		total := 0
		for g := 1; g <= 2; g++ {
			var start, end uint32
			if binary.BigEndian.Uint16(tables["head"][50:]) == 0 {
				start = uint32(binary.BigEndian.Uint16(loca[2*g:])) * 2
				end = uint32(binary.BigEndian.Uint16(loca[2*g+2:])) * 2
			} else {
				start, end = binary.BigEndian.Uint32(loca[4*g:]), binary.BigEndian.Uint32(loca[4*g+4:])
			}
			data := glyf[start:end]
			if len(data) < 12 || int16(binary.BigEndian.Uint16(data)) < 0 {
				continue
			}
			pos := 10 + 2*int(int16(binary.BigEndian.Uint16(data)))
			total += int(binary.BigEndian.Uint16(data[pos:]))
		}
		return total
	}
	if n := instructions(Options{}); n != 0 {
		t.Errorf("%d instruction bytes left without hinting", n)
	}
}

func TestStripInstructionsOfCompositeGlyph(t *testing.T) {
	g := []byte{
		0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, // header, 2 components
		0x01, 0x23, 0, 7, 0, 1, 0, 2, // words, more, instructions flag
		0x01, 0x03, 0, 8, 0, 3, 0, 4, // last, with instructions
		0, 2, 0xaa, 0xbb, // instruction length and bytes
	}
	got := stripInstructions(g)
	if len(got) != 26 {
		t.Fatalf("length = %d, want 26", len(got))
	}
	if flags := binary.BigEndian.Uint16(got[18:]); flags&0x100 != 0 {
		t.Errorf("last component flags = %#x, instructions bit still set", flags)
	}
}
