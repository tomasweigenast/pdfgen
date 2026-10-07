package subset

import (
	"encoding/binary"
	"errors"
)

// The CFF code here reads and rewrites just enough of the table to swap one
// charstring (Adobe Technical Note #5176).

const (
	cffOpCharset     = 15
	cffOpCharStrings = 17
	cffOpPrivate     = 18
)

// cffIndex parses the INDEX at off and returns its items and the offset
// after it.
func cffIndex(b []byte, off int) ([][]byte, int, error) {
	if off < 0 || off+2 > len(b) {
		return nil, 0, errors.New("INDEX outside the table")
	}
	count := int(binary.BigEndian.Uint16(b[off:]))
	if count == 0 {
		return nil, off + 2, nil
	}
	if off+3 > len(b) {
		return nil, 0, errors.New("truncated INDEX")
	}
	offSize := int(b[off+2])
	if offSize < 1 || offSize > 4 {
		return nil, 0, errors.New("bad INDEX offset size")
	}
	offsetsStart := off + 3
	dataStart := offsetsStart + (count+1)*offSize - 1
	if dataStart > len(b) {
		return nil, 0, errors.New("truncated INDEX")
	}
	at := func(i int) int {
		v := 0
		for _, c := range b[offsetsStart+i*offSize : offsetsStart+(i+1)*offSize] {
			v = v<<8 | int(c)
		}
		return v
	}
	items := make([][]byte, count)
	prev := at(0)
	for i := 0; i < count; i++ {
		next := at(i + 1)
		if next < prev || dataStart+next > len(b) {
			return nil, 0, errors.New("bad INDEX offsets")
		}
		items[i] = b[dataStart+prev : dataStart+next]
		prev = next
	}
	return items, dataStart + prev, nil
}

func writeCFFIndex(items [][]byte) []byte {
	if len(items) == 0 {
		return []byte{0, 0}
	}
	total := 1
	for _, it := range items {
		total += len(it)
	}
	offSize := 1
	for total >= 1<<(8*offSize) {
		offSize++
	}
	out := binary.BigEndian.AppendUint16(nil, uint16(len(items)))
	out = append(out, byte(offSize))
	put := func(v int) {
		for i := offSize - 1; i >= 0; i-- {
			out = append(out, byte(v>>(8*i)))
		}
	}
	pos := 1
	put(pos)
	for _, it := range items {
		pos += len(it)
		put(pos)
	}
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}

type dictEntry struct {
	op       int
	operands []byte
	values   []int
}

// parseDict splits a DICT into its entries. Real operands are kept but
// yield no value.
func parseDict(b []byte) ([]dictEntry, error) {
	var entries []dictEntry
	start := 0
	var values []int
	for i := 0; i < len(b); {
		b0 := int(b[i])
		switch {
		case b0 <= 21:
			op, n := b0, 1
			if b0 == 12 {
				if i+1 >= len(b) {
					return nil, errors.New("truncated DICT")
				}
				op, n = 1200+int(b[i+1]), 2
			}
			entries = append(entries, dictEntry{op: op, operands: b[start:i], values: values})
			i += n
			start, values = i, nil
		case b0 == 28:
			if i+3 > len(b) {
				return nil, errors.New("truncated DICT")
			}
			values = append(values, int(int16(binary.BigEndian.Uint16(b[i+1:]))))
			i += 3
		case b0 == 29:
			if i+5 > len(b) {
				return nil, errors.New("truncated DICT")
			}
			values = append(values, int(int32(binary.BigEndian.Uint32(b[i+1:]))))
			i += 5
		case b0 == 30:
			i++
			for i < len(b) && b[i]&0x0F != 0x0F && b[i]>>4 != 0x0F {
				i++
			}
			i++
			values = append(values, 0)
		case b0 >= 32 && b0 <= 246:
			values = append(values, b0-139)
			i++
		case b0 >= 247 && b0 <= 250:
			if i+2 > len(b) {
				return nil, errors.New("truncated DICT")
			}
			values = append(values, (b0-247)*256+int(b[i+1])+108)
			i += 2
		case b0 >= 251 && b0 <= 254:
			if i+2 > len(b) {
				return nil, errors.New("truncated DICT")
			}
			values = append(values, -(b0-251)*256-int(b[i+1])-108)
			i += 2
		default:
			return nil, errors.New("bad DICT operand")
		}
	}
	return entries, nil
}

type cffLayout struct {
	topStart, topEnd int // Top DICT INDEX
	gsubrEnd         int
	top              []dictEntry
	charset          int // offset, or -1 for a predefined charset
	charStrings      int
	csEnd            int
	private          int // offset, or -1
	items            [][]byte
}

func parseCFFLayout(b []byte) (*cffLayout, error) {
	if len(b) < 4 {
		return nil, errors.New("CFF table too short")
	}
	l := &cffLayout{charset: -1, private: -1}
	_, off, err := cffIndex(b, int(b[2]))
	if err != nil {
		return nil, err
	}
	l.topStart = off
	tops, off, err := cffIndex(b, off)
	if err != nil || len(tops) != 1 {
		return nil, errors.New("bad Top DICT INDEX")
	}
	l.topEnd = off
	if _, off, err = cffIndex(b, off); err != nil { // strings
		return nil, err
	}
	if _, l.gsubrEnd, err = cffIndex(b, off); err != nil {
		return nil, err
	}
	if l.top, err = parseDict(tops[0]); err != nil {
		return nil, err
	}
	for _, e := range l.top {
		switch {
		case e.op == cffOpCharset && len(e.values) == 1 && e.values[0] > 2:
			l.charset = e.values[0]
		case e.op == cffOpCharStrings && len(e.values) == 1:
			l.charStrings = e.values[0]
		case e.op == cffOpPrivate && len(e.values) == 2:
			l.private = e.values[1]
		case e.op == 1230: // ROS: CID-keyed
			return nil, errors.New("CID-keyed CFF")
		}
	}
	if l.charStrings == 0 {
		return nil, errors.New("no CharStrings")
	}
	if l.items, l.csEnd, err = cffIndex(b, l.charStrings); err != nil {
		return nil, err
	}
	return l, nil
}

// cffNotdef returns the charstring of glyph 0 when it can be copied into
// another font: no subroutine calls (their numbers would not survive the
// subset) and no hint masks (their length depends on the stem count).
func cffNotdef(table []byte) ([]byte, bool) {
	l, err := parseCFFLayout(table)
	if err != nil || len(l.items) == 0 {
		return nil, false
	}
	cs := l.items[0]
	if len(cs) == 0 {
		return nil, false
	}
	for i := 0; i < len(cs); {
		b0 := cs[i]
		switch {
		case b0 == 28:
			i += 3
		case b0 >= 32 && b0 <= 246:
			i++
		case b0 >= 247 && b0 <= 254:
			i += 2
		case b0 == 255:
			i += 5
		case b0 == 12:
			i += 2
		case b0 == 10 || b0 == 29 || b0 == 19 || b0 == 20:
			return nil, false
		default:
			i++
		}
	}
	return cs, true
}

// replaceCFFNotdef returns table with the first charstring replaced. The
// Top DICT offsets are rewritten with fixed-width operands so that its size
// does not depend on them.
func replaceCFFNotdef(table, notdef []byte) ([]byte, error) {
	l, err := parseCFFLayout(table)
	if err != nil {
		return nil, err
	}
	if l.charset >= 0 && (l.charset < l.gsubrEnd || l.charset > l.charStrings) {
		return nil, errors.New("unexpected charset position")
	}
	if l.private >= 0 && l.private < l.csEnd {
		return nil, errors.New("unexpected Private DICT position")
	}

	items := append([][]byte{notdef}, l.items[1:]...)
	charStrings := writeCFFIndex(items)
	delta := len(charStrings) - (l.csEnd - l.charStrings)

	build := func(topShift int) []byte {
		var dict []byte
		for _, e := range l.top {
			operands := e.operands
			switch e.op {
			case cffOpCharset:
				if l.charset >= 0 {
					operands = fixedInt(nil, l.charset+topShift)
				}
			case cffOpCharStrings:
				operands = fixedInt(nil, l.charStrings+topShift)
			case cffOpPrivate:
				operands = fixedInt(fixedInt(nil, e.values[0]), l.private+topShift+delta)
			}
			dict = append(dict, operands...)
			if e.op >= 1200 {
				dict = append(dict, 12, byte(e.op-1200))
			} else {
				dict = append(dict, byte(e.op))
			}
		}
		return writeCFFIndex([][]byte{dict})
	}
	topShift := len(build(0)) - (l.topEnd - l.topStart)
	top := build(topShift)

	out := append([]byte(nil), table[:l.topStart]...)
	out = append(out, top...)
	out = append(out, table[l.topEnd:l.charStrings]...)
	out = append(out, charStrings...)
	out = append(out, table[l.csEnd:]...)
	return out, nil
}

func fixedInt(dst []byte, v int) []byte {
	return binary.BigEndian.AppendUint32(append(dst, 29), uint32(int32(v)))
}
