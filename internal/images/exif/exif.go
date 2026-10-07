package exif

import (
	"bytes"
	"encoding/binary"
)

const tagOrientation = 0x0112

var exifHeader = []byte("Exif\x00\x00")

// Orientation returns the Orientation tag (1 to 8) of the first image file
// directory of a TIFF structure, as found in an Exif block (Exif 2.32 §4.6.4,
// tag 274). The boolean is false when the data is malformed or the tag is
// missing or out of range.
func Orientation(tiff []byte) (int, bool) {
	var order binary.ByteOrder
	switch {
	case bytes.HasPrefix(tiff, []byte("II*\x00")):
		order = binary.LittleEndian
	case bytes.HasPrefix(tiff, []byte("MM\x00*")):
		order = binary.BigEndian
	default:
		return 0, false
	}
	if len(tiff) < 8 {
		return 0, false
	}
	ifd := int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd > len(tiff)-2 {
		return 0, false
	}
	count := int(order.Uint16(tiff[ifd:]))
	entries := tiff[ifd+2:]
	for i := 0; i < count; i++ {
		if (i+1)*12 > len(entries) {
			return 0, false
		}
		e := entries[i*12 : i*12+12]
		if order.Uint16(e) != tagOrientation {
			continue
		}
		const short = 3
		if order.Uint16(e[2:]) != short || order.Uint32(e[4:]) != 1 {
			return 0, false
		}
		v := int(order.Uint16(e[8:]))
		if v < 1 || v > 8 {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

// Find returns the TIFF structure of the Exif block embedded in a JPEG, PNG
// or WebP file, or data itself when it already is a bare TIFF structure. The
// boolean is false when there is none.
func Find(data []byte) ([]byte, bool) {
	switch {
	case bytes.HasPrefix(data, []byte{0xff, 0xd8}):
		return fromJPEG(data)
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return fromPNG(data)
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return fromWebP(data)
	case bytes.HasPrefix(data, []byte("II*\x00")), bytes.HasPrefix(data, []byte("MM\x00*")):
		return data, true
	}
	return nil, false
}

// OrientationOf returns the Exif orientation of an image file. The boolean is
// false when the file carries none.
func OrientationOf(data []byte) (int, bool) {
	tiff, ok := Find(data)
	if !ok {
		return 0, false
	}
	return Orientation(tiff)
}

// fromJPEG walks the marker segments up to the start of scan; Exif lives in
// an APP1 segment (Exif 2.32 §4.5.4).
func fromJPEG(data []byte) ([]byte, bool) {
	for pos := 2; pos+4 <= len(data); {
		if data[pos] != 0xff {
			return nil, false
		}
		marker := data[pos+1]
		if marker == 0xff {
			pos++
			continue
		}
		if marker == 0xda || marker == 0xd9 {
			return nil, false
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd8) {
			pos += 2
			continue
		}
		size := int(binary.BigEndian.Uint16(data[pos+2:]))
		if size < 2 || pos+2+size > len(data) {
			return nil, false
		}
		segment := data[pos+4 : pos+2+size]
		if marker == 0xe1 && bytes.HasPrefix(segment, exifHeader) {
			return segment[len(exifHeader):], true
		}
		pos += 2 + size
	}
	return nil, false
}

// fromPNG looks for the eXIf chunk (PNG 1.5 §11.3.6.1).
func fromPNG(data []byte) ([]byte, bool) {
	for pos := 8; pos+8 <= len(data); {
		size := int(binary.BigEndian.Uint32(data[pos:]))
		kind := string(data[pos+4 : pos+8])
		end := pos + 8 + size
		if size < 0 || end+4 > len(data) {
			return nil, false
		}
		switch kind {
		case "eXIf":
			return data[pos+8 : end], true
		case "IDAT", "IEND":
			return nil, false
		}
		pos = end + 4
	}
	return nil, false
}

// fromWebP looks for the EXIF chunk of the RIFF container, which may or may
// not repeat the "Exif\0\0" prefix.
func fromWebP(data []byte) ([]byte, bool) {
	for pos := 12; pos+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[pos+4:]))
		end := pos + 8 + size
		if size < 0 || end > len(data) {
			return nil, false
		}
		if string(data[pos:pos+4]) == "EXIF" {
			return bytes.TrimPrefix(data[pos+8:end], exifHeader), true
		}
		pos = end + size&1
	}
	return nil, false
}
