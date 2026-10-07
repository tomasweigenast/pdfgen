package images

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
)

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// pngProfile returns the embedded ICC profile of a PNG file, from the iCCP
// chunk (PNG 1.5 §11.3.3.3), or nil.
func pngProfile(data []byte) []byte {
	if !bytes.HasPrefix(data, pngSignature) {
		return nil
	}
	for pos := len(pngSignature); pos+8 <= len(data); {
		size := int(binary.BigEndian.Uint32(data[pos:]))
		kind := string(data[pos+4 : pos+8])
		end := pos + 8 + size
		if size < 0 || end+4 > len(data) {
			return nil
		}
		switch kind {
		case "iCCP":
			return inflateProfile(data[pos+8 : end])
		case "IDAT", "IEND":
			return nil
		}
		pos = end + 4
	}
	return nil
}

// inflateProfile decodes the body of an iCCP chunk: a name, a null, the
// compression method (0 is deflate) and the compressed profile.
func inflateProfile(chunk []byte) []byte {
	name := bytes.IndexByte(chunk, 0)
	if name < 0 || name > 79 || len(chunk) < name+2 || chunk[name+1] != 0 {
		return nil
	}
	zr, err := zlib.NewReader(bytes.NewReader(chunk[name+2:]))
	if err != nil {
		return nil
	}
	defer zr.Close()
	profile, err := io.ReadAll(io.LimitReader(zr, 16<<20))
	if err != nil {
		return nil
	}
	return profile
}

// profileSpace returns the number of color components of the data color
// space of an ICC profile (ICC.1:2010 §7.2.6), or 0 for a malformed header or
// a space the PDF color spaces do not cover.
func profileSpace(profile []byte) int {
	if len(profile) < 128 || string(profile[36:40]) != "acsp" {
		return 0
	}
	switch string(profile[16:20]) {
	case "GRAY":
		return 1
	case "RGB ":
		return 3
	case "CMYK":
		return 4
	}
	return 0
}

// predictorFilter filters every row of a bpp-bytes-per-pixel image the way PNG
// does (PNG 1.5 §9) and prepends the filter type to each row. The filter of
// a row is the one that minimizes the sum of absolute values of its bytes,
// read as signed. It is the format that /Predictor 15 expects
// (ISO 32000-1 §7.4.4.4).
func predictorFilter(samples []byte, width, height, bpp int) []byte {
	stride := width * bpp
	out := make([]byte, 0, height*(stride+1))
	zero := make([]byte, stride)
	var candidates [5][]byte
	for i := range candidates {
		candidates[i] = make([]byte, stride)
	}
	for y := 0; y < height; y++ {
		row := samples[y*stride : (y+1)*stride]
		prior := zero
		if y > 0 {
			prior = samples[(y-1)*stride : y*stride]
		}
		best, bestSum := 0, -1
		for kind := range candidates {
			cand := candidates[kind]
			for x := range row {
				var left, up, upLeft byte
				if x >= bpp {
					left, upLeft = row[x-bpp], prior[x-bpp]
				}
				up = prior[x]
				cand[x] = row[x] - predict(kind, left, up, upLeft)
			}
			sum := 0
			for _, b := range cand {
				if b < 128 {
					sum += int(b)
				} else {
					sum += 256 - int(b)
				}
			}
			if bestSum < 0 || sum < bestSum {
				best, bestSum = kind, sum
			}
		}
		out = append(out, byte(best))
		out = append(out, candidates[best]...)
	}
	return out
}

func predict(kind int, left, up, upLeft byte) byte {
	switch kind {
	case 1:
		return left
	case 2:
		return up
	case 3:
		return byte((int(left) + int(up)) / 2)
	case 4:
		return paeth(left, up, upLeft)
	}
	return 0
}

func paeth(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := abs(p-int(a)), abs(p-int(b)), abs(p-int(c))
	switch {
	case pa <= pb && pa <= pc:
		return a
	case pb <= pc:
		return b
	}
	return c
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func deflate(data []byte, best bool) []byte {
	level := zlib.DefaultCompression
	if best {
		level = zlib.BestCompression
	}
	var buf bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&buf, level)
	_, _ = zw.Write(data)
	_ = zw.Close()
	return buf.Bytes()
}
