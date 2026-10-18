package subset

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"sort"

	tdfont "github.com/tdewolff/font"
)

// ErrUnsupported is wrapped by the errors for fonts that cannot be subset
// yet. Callers fall back to embedding the whole font.
var ErrUnsupported = errors.New("unsupported font")

// Options select what the subset keeps.
type Options struct {
	// KeepNotdef keeps the outline of glyph 0. Without it glyph 0 is empty,
	// which is what a document wants unless it displays missing glyphs.
	KeepNotdef bool
	// Hinting keeps the hinting programs (cvt, fpgm, prep) of TrueType
	// fonts.
	Hinting bool
}

// Font is a parsed font. Subsetting does not change it, so a Font can be
// subset several times.
type Font struct {
	sfnt *tdfont.SFNT
}

// Parse reads the face at index of an sfnt or collection. WOFF data must be
// decompressed by the caller.
func Parse(data []byte, index int) (*Font, error) {
	tables, version, err := readTables(data, index)
	if err != nil {
		return nil, err
	}
	if _, ok := tables["CFF2"]; ok {
		return nil, fmt.Errorf("%w: CFF2 outlines", ErrUnsupported)
	}

	// A single face parses faster and without the collection directory.
	standalone := writeSFNT(version, tables)
	sf, err := parseSFNT(standalone)
	if err != nil {
		// The cmap is not needed to subset by glyph index, and some
		// collections ship cmap formats the parser rejects.
		delete(tables, "cmap")
		var err2 error
		sf, err2 = parseSFNT(writeSFNT(version, tables))
		if err2 != nil {
			if _, cff := tables["CFF "]; cff {
				return nil, fmt.Errorf("%w: %v", ErrUnsupported, err2)
			}
			return nil, fmt.Errorf("parsing font: %w", err)
		}
	}
	if !sf.IsCFF && !sf.IsTrueType {
		return nil, fmt.Errorf("%w: no outlines", ErrUnsupported)
	}
	return &Font{sfnt: sf}, nil
}

func parseSFNT(data []byte) (sf *tdfont.SFNT, err error) {
	defer func() {
		if value := recover(); value != nil {
			sf = nil
			err = fmt.Errorf("font parser panic: %v", value)
		}
	}()
	return tdfont.ParseSFNT(data, 0)
}

// IsCFF reports whether the outlines are PostScript (CFF) outlines.
func (f *Font) IsCFF() bool { return f.sfnt.IsCFF }

// NumGlyphs returns the number of glyphs of the font.
func (f *Font) NumGlyphs() int { return int(f.sfnt.NumGlyphs()) }

// Full returns the face at index as a standalone sfnt with all its glyphs.
// Data that already is a standalone sfnt is returned unchanged.
func Full(data []byte, index int) ([]byte, error) {
	if len(data) >= 4 && string(data[:4]) != "ttcf" {
		return data, nil
	}
	tables, version, err := readTables(data, index)
	if err != nil {
		return nil, err
	}
	return writeSFNT(version, tables), nil
}

// Subset returns an sfnt that contains only glyphs. Glyph i of the result is
// glyphs[i] of the font, and glyphs[0] must be 0 (.notdef). Glyph 0 may be
// repeated, which gives several .notdef glyphs; any other glyph may appear
// once. TrueType composite glyphs pull in the glyphs they use, which are
// appended after the requested ones.
func (f *Font) Subset(glyphs []uint16, opts Options) ([]byte, error) {
	if len(glyphs) == 0 || glyphs[0] != 0 {
		return nil, errors.New("subset: the first glyph must be .notdef")
	}
	seen := make(map[uint16]bool, len(glyphs))
	for _, g := range glyphs {
		if int(g) >= f.NumGlyphs() {
			return nil, fmt.Errorf("subset: glyph %d out of range (%d glyphs)", g, f.NumGlyphs())
		}
		if g != 0 && seen[g] {
			return nil, fmt.Errorf("subset: glyph %d listed twice", g)
		}
		seen[g] = true
	}

	tags := []string{"head", "hhea", "hmtx", "maxp"}
	if f.sfnt.IsCFF {
		tags = append(tags, "CFF ")
	} else {
		tags = append(tags, "glyf", "loca")
		if opts.Hinting {
			for _, t := range []string{"cvt ", "fpgm", "prep"} {
				if _, ok := f.sfnt.Tables[t]; ok {
					tags = append(tags, t)
				}
			}
		}
	}
	// Subset gives up on glyphs it cannot find, and modifies the slice it is
	// given.
	sub, err := f.sfnt.Subset(append([]uint16(nil), glyphs...), tdfont.SubsetOptions{Tables: tags})
	if err != nil {
		return nil, fmt.Errorf("subsetting font: %w", err)
	}

	tables := sub.Tables
	if opts.KeepNotdef {
		if err := f.restoreNotdef(tables, glyphs); err != nil {
			return nil, fmt.Errorf("restoring .notdef: %w", err)
		}
	}
	if sub.IsTrueType {
		if err := f.rewriteOutlines(tables, glyphs, opts); err != nil {
			return nil, fmt.Errorf("rewriting outlines: %w", err)
		}
	}
	version := uint32(0x00010000)
	if sub.IsCFF {
		version = tagOTTO
	}
	return writeSFNT(version, tables), nil
}

const tagOTTO = 0x4F54544F

// readTables returns the tables of the face at index and its sfnt version.
func readTables(b []byte, index int) (map[string][]byte, uint32, error) {
	if len(b) < 12 {
		return nil, 0, errors.New("font data too short")
	}
	dir := 0
	if string(b[:4]) == "ttcf" {
		n := int(binary.BigEndian.Uint32(b[8:]))
		if index < 0 || index >= n || 12+4*n > len(b) {
			return nil, 0, fmt.Errorf("font index %d out of range", index)
		}
		dir = int(binary.BigEndian.Uint32(b[12+4*index:]))
	} else if index != 0 {
		return nil, 0, fmt.Errorf("font index %d out of range", index)
	}
	if dir+12 > len(b) {
		return nil, 0, errors.New("truncated font directory")
	}
	version := binary.BigEndian.Uint32(b[dir:])
	n := int(binary.BigEndian.Uint16(b[dir+4:]))
	if dir+12+16*n > len(b) {
		return nil, 0, errors.New("truncated font directory")
	}
	tables := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		rec := b[dir+12+16*i:]
		off := int(binary.BigEndian.Uint32(rec[8:]))
		length := int(binary.BigEndian.Uint32(rec[12:]))
		if off < 0 || length < 0 || off+length > len(b) {
			return nil, 0, fmt.Errorf("table %q outside the font data", rec[:4])
		}
		tables[string(rec[:4])] = b[off : off+length : off+length]
	}
	return tables, version, nil
}

// writeSFNT serializes tables in tag order. The head table is written as it
// is, apart from checkSumAdjustment: its created and modified dates come from
// the source font, which keeps the output reproducible.
func writeSFNT(version uint32, tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for t := range tables {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	n := len(tags)
	entrySelector, searchRange := 0, 0
	if n > 0 {
		entrySelector = bits.Len(uint(n)) - 1
		searchRange = 16 << entrySelector
	}

	size := 12 + 16*n
	for _, t := range tags {
		size += (len(tables[t]) + 3) &^ 3
	}
	out := make([]byte, 12+16*n, size)
	binary.BigEndian.PutUint32(out, version)
	binary.BigEndian.PutUint16(out[4:], uint16(n))
	binary.BigEndian.PutUint16(out[6:], uint16(searchRange))
	binary.BigEndian.PutUint16(out[8:], uint16(entrySelector))
	binary.BigEndian.PutUint16(out[10:], uint16(16*n-searchRange))

	adjustment := -1
	for i, t := range tags {
		data := tables[t]
		off := len(out)
		if t == "head" && len(data) >= 12 {
			data = bytes.Clone(data)
			binary.BigEndian.PutUint32(data[8:], 0)
			adjustment = off + 8
		}
		rec := out[12+16*i:]
		copy(rec, t)
		binary.BigEndian.PutUint32(rec[4:], checksum(data))
		binary.BigEndian.PutUint32(rec[8:], uint32(off))
		binary.BigEndian.PutUint32(rec[12:], uint32(len(data)))
		out = append(out, data...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	if adjustment >= 0 {
		binary.BigEndian.PutUint32(out[adjustment:], 0xB1B0AFBA-checksum(out))
	}
	return out
}

func checksum(b []byte) uint32 {
	var sum uint32
	for ; len(b) >= 4; b = b[4:] {
		sum += binary.BigEndian.Uint32(b)
	}
	if len(b) > 0 {
		var last [4]byte
		copy(last[:], b)
		sum += binary.BigEndian.Uint32(last[:])
	}
	return sum
}

// GlyphCount returns the number of glyphs of the face at index, from its maxp
// table. It works for any font, including those Parse does not support.
func GlyphCount(data []byte, index int) (int, error) {
	tables, _, err := readTables(data, index)
	if err != nil {
		return 0, err
	}
	maxp := tables["maxp"]
	if len(maxp) < 6 {
		return 0, errors.New("missing maxp table")
	}
	return int(binary.BigEndian.Uint16(maxp[4:])), nil
}

// rewriteOutlines applies the TrueType-only options to the glyphs of a
// subset: it restores the outline of .notdef and drops hinting programs.
func (f *Font) rewriteOutlines(tables map[string][]byte, glyphs []uint16, opts Options) error {
	if !opts.KeepNotdef && opts.Hinting {
		return nil
	}
	var notdef []byte
	if opts.KeepNotdef && f.sfnt.Glyf != nil && !f.sfnt.Glyf.IsComposite(0) {
		notdef = f.sfnt.Glyf.Get(0)
	}
	return rewriteGlyf(tables, func(i int, g []byte) []byte {
		if i < len(glyphs) && glyphs[i] == 0 && notdef != nil {
			g = notdef
		}
		if !opts.Hinting {
			g = stripInstructions(g)
		}
		return g
	})
}
