package images

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func segment(marker byte, body []byte) []byte {
	b := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(b[2:], uint16(2+len(body)))
	return append(b, body...)
}

func iccSegment(seq, count byte, body string) []byte {
	return segment(0xe2, append(append([]byte{}, iccTag...), append([]byte{seq, count}, body...)...))
}

func TestScanJPEGJoinsProfileChunks(t *testing.T) {
	frame := []byte{8, 0, 3, 0, 5, 3, 1, 0x22, 0, 2, 0x11, 1, 3, 0x11, 1}
	build := func(chunks ...[]byte) []byte {
		data := []byte{0xff, 0xd8}
		for _, c := range chunks {
			data = append(data, c...)
		}
		data = append(data, segment(0xc0, frame)...)
		return append(data, 0xff, 0xda, 0, 2)
	}
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"one chunk", build(iccSegment(1, 1, "profile")), "profile"},
		{"out of order", build(iccSegment(2, 2, "file"), iccSegment(1, 2, "pro")), "profile"},
		{"missing chunk", build(iccSegment(1, 2, "pro")), ""},
		{"wrong numbering", build(iccSegment(1, 2, "pro"), iccSegment(3, 2, "file")), ""},
		{"none", build(), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := scanJPEG(tt.data)
			if err != nil {
				t.Fatal(err)
			}
			if string(info.icc) != tt.want {
				t.Errorf("profile %q, want %q", info.icc, tt.want)
			}
			if info.width != 5 || info.height != 3 || info.components != 3 {
				t.Errorf("frame %d×%d with %d components", info.width, info.height, info.components)
			}
		})
	}
}

func FuzzNewRaster(f *testing.F) {
	for _, name := range []string{"pattern.png", "pattern.gif", "blue.jpg", "cmyk_with_icc.jpg", "not-optimized-exif.jpg"} {
		f.Add(readResource(f, name))
	}
	f.Add(bytes.Repeat([]byte{0xff}, 40))
	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := NewRaster("fuzz", data, OrientationFromImage, Options{})
		if err != nil {
			return
		}
		embed(t, r, true, 0.5)
	})
}
