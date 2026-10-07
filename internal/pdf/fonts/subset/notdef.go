package subset

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// restoreNotdef puts the outline of the source glyph 0 back in the subset
// tables, which the subsetter leaves empty. A .notdef that cannot be moved
// as is (a TrueType composite, or a charstring that calls subroutines or
// declares hints) stays empty.
func (f *Font) restoreNotdef(tables map[string][]byte) error {
	if f.sfnt.IsCFF {
		notdef, ok := cffNotdef(f.sfnt.Tables["CFF "])
		if !ok {
			return nil
		}
		patched, err := replaceCFFNotdef(tables["CFF "], notdef)
		if err != nil {
			return fmt.Errorf("restoring .notdef: %w", err)
		}
		tables["CFF "] = patched
		return nil
	}

	glyf := f.sfnt.Glyf
	if glyf == nil || glyf.IsComposite(0) {
		return nil
	}
	outline := glyf.Get(0)
	if len(outline) == 0 {
		return nil
	}
	if err := prependGlyph(tables, outline); err != nil {
		return fmt.Errorf("restoring .notdef: %w", err)
	}
	return nil
}

// prependGlyph stores outline as glyph 0 of the glyf table. Glyph 0 is empty
// in the subset, so it sits at offset 0 and every other glyph moves by the
// length of the outline.
func prependGlyph(tables map[string][]byte, outline []byte) error {
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
	if len(offsets) < 2 || offsets[1] != 0 {
		return errors.New("glyph 0 is not empty")
	}

	padded := len(outline) + len(outline)%2
	shift := uint32(padded)
	for i := 1; i < len(offsets); i++ {
		offsets[i] += shift
	}

	newGlyf := make([]byte, 0, padded+len(glyf))
	newGlyf = append(newGlyf, outline...)
	newGlyf = append(newGlyf, make([]byte, padded-len(outline))...)
	newGlyf = append(newGlyf, glyf...)

	long = offsets[len(offsets)-1] > 2*0xFFFF
	newLoca := make([]byte, 0, 4*len(offsets))
	for _, o := range offsets {
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
