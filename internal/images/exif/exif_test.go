package exif

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// tiffWith builds a TIFF structure whose first directory holds an unrelated
// tag followed by an Orientation tag of the given type and count.
func tiffWith(order binary.ByteOrder, orientation, kind uint16, count uint32) []byte {
	b := make([]byte, 8+2+2*12+4)
	if order == binary.LittleEndian {
		copy(b, "II*\x00")
	} else {
		copy(b, "MM\x00*")
	}
	order.PutUint32(b[4:], 8)
	order.PutUint16(b[8:], 2)
	entry := b[10:]
	order.PutUint16(entry, 0x010f) // Make
	order.PutUint16(entry[2:], 2)
	order.PutUint32(entry[4:], 1)
	entry = entry[12:]
	order.PutUint16(entry, tagOrientation)
	order.PutUint16(entry[2:], kind)
	order.PutUint32(entry[4:], count)
	order.PutUint16(entry[8:], orientation)
	return b
}

func jpegWith(app1 []byte) []byte {
	b := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 4, 'J', 'F', 0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(b[10:], uint16(2+len(exifHeader)+len(app1)))
	b = append(b, exifHeader...)
	b = append(b, app1...)
	return append(b, 0xff, 0xda, 0, 2)
}

func pngWith(tiff []byte) []byte {
	b := []byte("\x89PNG\r\n\x1a\n")
	chunk := func(kind string, data []byte) {
		b = binary.BigEndian.AppendUint32(b, uint32(len(data)))
		b = append(b, kind...)
		b = append(b, data...)
		b = append(b, 0, 0, 0, 0)
	}
	chunk("IHDR", make([]byte, 13))
	chunk("eXIf", tiff)
	chunk("IEND", nil)
	return b
}

func webpWith(tiff []byte, prefix bool) []byte {
	payload := tiff
	if prefix {
		payload = append(append([]byte{}, exifHeader...), tiff...)
	}
	b := []byte("RIFF\x00\x00\x00\x00WEBPVP8X\x0a\x00\x00\x00")
	b = append(b, make([]byte, 10)...)
	b = append(b, "EXIF"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(payload)))
	b = append(b, payload...)
	if len(payload)%2 == 1 {
		b = append(b, 0)
	}
	return b
}

func TestOrientation(t *testing.T) {
	tests := []struct {
		name string
		tiff []byte
		want int
		ok   bool
	}{
		{"little endian", tiffWith(binary.LittleEndian, 6, 3, 1), 6, true},
		{"big endian", tiffWith(binary.BigEndian, 8, 3, 1), 8, true},
		{"identity", tiffWith(binary.BigEndian, 1, 3, 1), 1, true},
		{"zero", tiffWith(binary.BigEndian, 0, 3, 1), 0, false},
		{"above range", tiffWith(binary.BigEndian, 9, 3, 1), 0, false},
		{"wrong type", tiffWith(binary.BigEndian, 3, 4, 1), 0, false},
		{"wrong count", tiffWith(binary.BigEndian, 3, 3, 2), 0, false},
		{"no byte order", []byte("XX*\x00\x00\x00\x00\x08"), 0, false},
		{"truncated header", []byte("II*\x00"), 0, false},
		{"directory out of range", []byte("II*\x00\xff\x00\x00\x00\x00\x00"), 0, false},
		{"truncated entries", tiffWith(binary.BigEndian, 3, 3, 1)[:30], 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Orientation(tt.tiff)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Orientation() = %d, %v, want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestOrientationOfContainers(t *testing.T) {
	tiff := tiffWith(binary.BigEndian, 6, 3, 1)
	tests := []struct {
		name string
		data []byte
		ok   bool
	}{
		{"jpeg", jpegWith(tiff), true},
		{"png", pngWith(tiff), true},
		{"webp with prefix", webpWith(tiff, true), true},
		{"webp without prefix", webpWith(tiff, false), true},
		{"bare tiff", tiff, true},
		{"jpeg without exif", []byte{0xff, 0xd8, 0xff, 0xda, 0, 2}, false},
		{"jpeg cut inside a segment", jpegWith(tiff)[:14], false},
		{"png without exif", pngWith(nil)[:8+25], false},
		{"unknown", []byte("GIF89a"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := OrientationOf(tt.data)
			if ok != tt.ok || (ok && got != 6) {
				t.Errorf("OrientationOf() = %d, %v, want 6, %v", got, ok, tt.ok)
			}
		})
	}
}

func TestOrientationOfFile(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "testdata", "resources")
	tests := []struct {
		file string
		want int
		ok   bool
	}{
		{"not-optimized-exif.jpg", 3, true},
		{"not-optimized.jpg", 0, false},
	}
	for _, tt := range tests {
		data, err := os.ReadFile(filepath.Join(dir, tt.file))
		if err != nil {
			t.Fatal(err)
		}
		got, ok := OrientationOf(data)
		if got != tt.want || ok != tt.ok {
			t.Errorf("OrientationOf(%s) = %d, %v, want %d, %v", tt.file, got, ok, tt.want, tt.ok)
		}
	}
}

func FuzzFind(f *testing.F) {
	f.Add(jpegWith(tiffWith(binary.BigEndian, 6, 3, 1)))
	f.Add(pngWith(tiffWith(binary.LittleEndian, 2, 3, 1)))
	f.Add(webpWith(tiffWith(binary.BigEndian, 5, 3, 1), true))
	f.Fuzz(func(t *testing.T, data []byte) {
		OrientationOf(data)
	})
}
