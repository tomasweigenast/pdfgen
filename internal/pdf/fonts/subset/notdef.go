package subset

import (
	"encoding/binary"
	"errors"
)

// restoreNotdef puts the outline of the source glyph 0 back in every glyph of
// the subset that stands for it, which the subsetter leaves empty. A .notdef
// that cannot be moved as is (a TrueType composite, or a charstring that
// calls subroutines or declares hints) stays empty.
func (f *Font) restoreNotdef(tables map[string][]byte, glyphs []uint16) error {
	var indices []int
	for i, g := range glyphs {
		if g == 0 {
			indices = append(indices, i)
		}
	}
	if !f.sfnt.IsCFF {
		return nil // handled with the rest of the glyphs by rewriteGlyf
	}
	notdef, ok := cffNotdef(f.sfnt.Tables["CFF "])
	if !ok {
		return nil
	}
	patched, err := replaceCFFGlyphs(tables["CFF "], indices, notdef)
	if err != nil {
		return err
	}
	tables["CFF "] = patched
	return nil
}

// rewriteGlyf rebuilds the glyf and loca tables of a subset with each glyph
// passed through fn, which gets the index of the glyph in the subset.
func rewriteGlyf(tables map[string][]byte, fn func(i int, glyph []byte) []byte) error {
	head, loca, glyf := tables["head"], tables["loca"], tables["glyf"]
	if len(head) < 54 {
		return errors.New("bad head table")
	}
	long := binary.BigEndian.Uint16(head[50:]) != 0

	var offsets []uint32
	if long {
		for i := 0; i+4 <= len(loca); i += 4 {
			offsets = append(offsets, binary.BigEndian.Uint32(loca[i:]))
		}
	} else {
		for i := 0; i+2 <= len(loca); i += 2 {
			offsets = append(offsets, uint32(binary.BigEndian.Uint16(loca[i:]))*2)
		}
	}
	if len(offsets) == 0 {
		return errors.New("empty loca table")
	}

	newGlyf := make([]byte, 0, len(glyf))
	newOffsets := make([]uint32, 0, len(offsets))
	newOffsets = append(newOffsets, 0)
	for i := 0; i+1 < len(offsets); i++ {
		start, end := offsets[i], offsets[i+1]
		if start > end || int(end) > len(glyf) {
			return errors.New("bad loca table")
		}
		newGlyf = append(newGlyf, fn(i, glyf[start:end])...)
		if len(newGlyf)%2 == 1 {
			newGlyf = append(newGlyf, 0)
		}
		newOffsets = append(newOffsets, uint32(len(newGlyf)))
	}

	long = newOffsets[len(newOffsets)-1] > 2*0xFFFF
	newLoca := make([]byte, 0, 4*len(newOffsets))
	for _, o := range newOffsets {
		if long {
			newLoca = binary.BigEndian.AppendUint32(newLoca, o)
		} else {
			newLoca = binary.BigEndian.AppendUint16(newLoca, uint16(o/2))
		}
	}
	newHead := append([]byte(nil), head...)
	format := uint16(0)
	if long {
		format = 1
	}
	binary.BigEndian.PutUint16(newHead[50:], format)

	tables["glyf"], tables["loca"], tables["head"] = newGlyf, newLoca, newHead
	return nil
}

// stripInstructions removes the hinting program of a glyph. Subsets drop the
// shared programs (fpgm, prep), which glyph programs may depend on.
func stripInstructions(g []byte) []byte {
	if len(g) < 10 {
		return g
	}
	contours := int16(binary.BigEndian.Uint16(g))
	if contours >= 0 {
		pos := 10 + 2*int(contours)
		if pos+2 > len(g) {
			return g
		}
		length := int(binary.BigEndian.Uint16(g[pos:]))
		if pos+2+length > len(g) {
			return g
		}
		out := append([]byte(nil), g[:pos]...)
		out = append(out, 0, 0)
		return append(out, g[pos+2+length:]...)
	}

	const (
		arg1And2AreWords   = 0x0001
		weHaveAScale       = 0x0008
		moreComponents     = 0x0020
		weHaveAnXAndYScale = 0x0040
		weHaveATwoByTwo    = 0x0080
		weHaveInstructions = 0x0100
	)
	out := append([]byte(nil), g...)
	off := 10
	for {
		if off+4 > len(out) {
			return g
		}
		flags := binary.BigEndian.Uint16(out[off:])
		size := 4
		if flags&arg1And2AreWords != 0 {
			size += 4
		} else {
			size += 2
		}
		switch {
		case flags&weHaveAScale != 0:
			size += 2
		case flags&weHaveAnXAndYScale != 0:
			size += 4
		case flags&weHaveATwoByTwo != 0:
			size += 8
		}
		binary.BigEndian.PutUint16(out[off:], flags&^weHaveInstructions)
		off += size
		if flags&moreComponents == 0 {
			break
		}
	}
	if off > len(out) {
		return g
	}
	return out[:off]
}
