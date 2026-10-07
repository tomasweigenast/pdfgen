package writer

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
)

// WriteOptions control the serialization of a document.
type WriteOptions struct {
	// Version is the header version, such as "1.7". Empty means "1.7".
	Version string
	// Identifier is the first element of the trailer /ID. Empty uses the
	// digest of the document, which is always the second element.
	Identifier []byte
	// OmitID leaves the /ID entry out.
	OmitID bool
	// Compress is the default for streams whose Compress field is nil. With
	// versions 1.5 and later it also packs every non-stream object into an
	// object stream and replaces the cross-reference table with a
	// cross-reference stream.
	Compress bool
}

const defaultVersion = "1.7"

// Write serializes the document. Every added object is written, in number
// order, whether or not it is reachable from the catalog. The document is
// not modified, so writing twice gives identical bytes.
func (p *PDF) Write(w io.Writer, o WriteOptions) error {
	if o.Version == "" {
		o.Version = defaultVersion
	}
	objs := append([]Object(nil), p.objects...)
	for i, obj := range objs[1:] {
		if obj == nil {
			return fmt.Errorf("writer: object %d was reserved but never set", i+1)
		}
	}
	info := 0
	if p.info.Len() > 0 {
		objs = append(objs, p.info)
		info = len(objs) - 1
	}

	e := &encoder{w: bufio.NewWriter(w), opts: o, objs: objs, info: info}
	if o.Version >= "1.5" && o.Compress {
		e.writeCompressed()
	} else {
		e.writeClassic()
	}
	if e.err != nil {
		return fmt.Errorf("writer: %w", e.err)
	}
	if err := e.w.Flush(); err != nil {
		return fmt.Errorf("writer: %w", err)
	}
	return nil
}

type encoder struct {
	w    *bufio.Writer
	opts WriteOptions
	objs []Object // objs[0] is unused
	info int      // number of the information dictionary, 0 if absent
	pos  int64
	err  error
}

func (e *encoder) write(b []byte) {
	if e.err != nil {
		return
	}
	n, err := e.w.Write(b)
	e.pos += int64(n)
	e.err = err
}

func (e *encoder) line(s string) { e.write([]byte(s + "\n")) }

func (e *encoder) writeHeader() {
	e.line("%PDF-" + e.opts.Version)
	e.write([]byte("%\xf0\x9f\x96\xa4\n"))
}

// writeIndirect writes "N 0 obj", the serialized data and "endobj".
func (e *encoder) writeIndirect(num int, data []byte) {
	e.write(strconv.AppendInt(nil, int64(num), 10))
	e.write([]byte(" 0 obj\n"))
	e.write(data)
	e.write([]byte("\nendobj\n"))
}

func (e *encoder) data(num int) []byte {
	return e.objs[num].appendTo(nil, e.opts.Compress)
}

func (e *encoder) wantID() bool { return !e.opts.OmitID }

// idEntry returns the /ID array for the given digest of all object data.
func (e *encoder) idEntry(sum [md5.Size]byte) []byte {
	digest := []byte(hex.EncodeToString(sum[:]))
	first := e.opts.Identifier
	if len(first) == 0 {
		first = digest
	}
	var b []byte
	b = append(b, '[')
	b = Literal(first).appendTo(b, false)
	b = append(b, ' ')
	b = Literal(digest).appendTo(b, false)
	return append(b, ']')
}

func (e *encoder) writeClassic() {
	e.writeHeader()
	n := len(e.objs)
	offsets := make([]int64, n)
	h := md5.New()
	for num := 1; num < n; num++ {
		data := e.data(num)
		offsets[num] = e.pos
		e.writeIndirect(num, data)
		if e.wantID() {
			h.Write(data)
		}
	}

	xref := e.pos
	e.line("xref")
	e.line("0 " + strconv.Itoa(n))
	e.line("0000000000 65535 f ")
	for num := 1; num < n; num++ {
		e.line(fmt.Sprintf("%010d 00000 n ", offsets[num]))
	}
	e.line("trailer")
	e.line("<<")
	e.line("/Size " + strconv.Itoa(n))
	e.line("/Root " + string(Ref{Num: 2}.appendTo(nil, false)))
	if e.info != 0 {
		e.line("/Info " + string(Ref{Num: e.info}.appendTo(nil, false)))
	}
	if e.wantID() {
		var sum [md5.Size]byte
		copy(sum[:], h.Sum(nil))
		e.line("/ID " + string(e.idEntry(sum)))
	}
	e.line(">>")
	e.line("startxref")
	e.line(strconv.FormatInt(xref, 10))
	e.line("%%EOF")
}

// writeCompressed writes every non-stream object into one object stream and
// indexes the file with a cross-reference stream (ISO 32000-1 §7.5.7,
// §7.5.8).
func (e *encoder) writeCompressed() {
	e.writeHeader()
	n := len(e.objs)
	compress := true

	datas := make([][]byte, n+1) // retained for the /ID digest
	offsets := make([]int64, n+2)
	var packed []int
	for num := 1; num < n; num++ {
		datas[num] = e.data(num)
		if _, isStream := e.objs[num].(*Stream); !isStream {
			packed = append(packed, num)
			continue
		}
		offsets[num] = e.pos
		e.writeIndirect(num, datas[num])
	}

	var index []byte
	parts := [][]byte{nil}
	slot := make(map[int]int, len(packed))
	off := 0
	for i, num := range packed {
		if i > 0 {
			index = append(index, ' ')
		}
		index = strconv.AppendInt(index, int64(num), 10)
		index = append(index, ' ')
		index = strconv.AppendInt(index, int64(off), 10)
		off += len(datas[num]) + 1
		parts = append(parts, datas[num])
		slot[num] = i
	}
	parts[0] = index
	objStm := &Stream{
		Dict:     NewDict("Type", Name("ObjStm"), "N", len(packed), "First", len(index)+1),
		Data:     bytes.Join(parts, []byte("\n")),
		Compress: &compress,
	}
	objStmNum := n
	datas[objStmNum] = objStm.appendTo(nil, compress)
	offsets[objStmNum] = e.pos
	e.writeIndirect(objStmNum, datas[objStmNum])

	xrefNum := n + 1
	xrefPos := e.pos
	offsets[xrefNum] = xrefPos
	// Field widths: the largest offset is the xref stream's own, and the
	// third field holds generation 65535 of the free head or an index.
	w2 := byteWidth(xrefPos)
	w3 := byteWidth(int64(max(65535, len(packed))))
	var rows []byte
	row := func(typ byte, f2, f3 int64) {
		rows = append(rows, typ)
		rows = appendBigEndian(rows, f2, w2)
		rows = appendBigEndian(rows, f3, w3)
	}
	row(0, 0, 65535)
	for num := 1; num <= xrefNum; num++ {
		if i, ok := slot[num]; ok {
			row(2, int64(objStmNum), int64(i))
			continue
		}
		row(1, offsets[num], 0)
	}

	d := NewDict(
		"Type", Name("XRef"),
		"Index", Array{Int(0), Int(xrefNum + 1)},
		"W", Array{Int(1), Int(w2), Int(w3)},
		"Size", xrefNum+1,
		"Root", Ref{Num: 2},
	)
	if e.info != 0 {
		d.Set("Info", Ref{Num: e.info})
	}
	if e.wantID() {
		h := md5.New()
		for num := 1; num <= objStmNum; num++ {
			h.Write(datas[num])
		}
		var sum [md5.Size]byte
		copy(sum[:], h.Sum(nil))
		d.Set("ID", rawObject(e.idEntry(sum)))
	}
	xref := &Stream{Dict: d, Data: rows, Compress: &compress}
	e.writeIndirect(xrefNum, xref.appendTo(nil, compress))

	e.line("startxref")
	e.line(strconv.FormatInt(xrefPos, 10))
	e.line("%%EOF")
}

// rawObject is pre-serialized PDF syntax.
type rawObject []byte

func (r rawObject) appendTo(dst []byte, _ bool) []byte { return append(dst, r...) }

// byteWidth returns the number of bytes needed to hold v, at least one.
func byteWidth(v int64) int {
	w := 1
	for ; v > 0xff; v >>= 8 {
		w++
	}
	return w
}

func appendBigEndian(dst []byte, v int64, width int) []byte {
	for i := width - 1; i >= 0; i-- {
		dst = append(dst, byte(v>>(8*i)))
	}
	return dst
}
