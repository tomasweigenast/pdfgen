package images

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// jpegInfo is what the PDF needs to know about a JPEG file, read from its
// marker segments without decoding the scan (ITU T.81 §B.1).
type jpegInfo struct {
	width, height int
	components    int
	precision     int
	// adobe reports an APP14 segment. Adobe writes CMYK data inverted, so the
	// PDF has to invert it back.
	adobe bool
}

func scanJPEG(data []byte) (jpegInfo, error) {
	var info jpegInfo
	if !bytes.HasPrefix(data, []byte{0xff, 0xd8}) {
		return info, errors.New("missing JPEG start of image")
	}
	for pos := 2; pos+2 <= len(data); {
		if data[pos] != 0xff {
			return info, errors.New("invalid JPEG marker")
		}
		marker := data[pos+1]
		switch {
		case marker == 0xff:
			pos++
			continue
		case marker == 0x01 || (marker >= 0xd0 && marker <= 0xd8):
			pos += 2
			continue
		case marker == 0xd9:
			pos = len(data)
			continue
		}
		if pos+4 > len(data) {
			break
		}
		size := int(binary.BigEndian.Uint16(data[pos+2:]))
		if size < 2 || pos+2+size > len(data) {
			break
		}
		segment := data[pos+4 : pos+2+size]
		switch {
		case isSOF(marker):
			if marker != 0xc0 && marker != 0xc1 && marker != 0xc2 {
				return info, fmt.Errorf("unsupported JPEG process %#x", marker)
			}
			if len(segment) < 6 {
				return info, errors.New("truncated JPEG frame header")
			}
			info.precision = int(segment[0])
			info.height = int(binary.BigEndian.Uint16(segment[1:]))
			info.width = int(binary.BigEndian.Uint16(segment[3:]))
			info.components = int(segment[5])
		case marker == 0xee:
			info.adobe = true
		case marker == 0xda:
			pos = len(data)
			continue
		}
		pos += 2 + size
	}
	if info.components == 0 {
		return info, errors.New("missing JPEG frame header")
	}
	if info.precision != 8 {
		return info, fmt.Errorf("unsupported JPEG precision %d", info.precision)
	}
	if info.width == 0 || info.height == 0 {
		return info, errors.New("JPEG without dimensions")
	}
	return info, nil
}

// isSOF reports whether marker starts a frame header: SOF0 to SOF15 except
// DHT, JPG and DAC.
func isSOF(marker byte) bool {
	return marker >= 0xc0 && marker <= 0xcf && marker != 0xc4 && marker != 0xc8 && marker != 0xcc
}
