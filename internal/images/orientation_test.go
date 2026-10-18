package images

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// reds returns the first sample of every pixel.
func reds(t testing.TB, r *Raster) []byte {
	t.Helper()
	samples := unpredict(t, embed(t, r, false, 1).obj)
	var out []byte
	for i := 0; i < len(samples); i += 3 {
		out = append(out, samples[i])
	}
	return out
}

func exifTIFF(orientation uint16) []byte {
	b := []byte("MM\x00*\x00\x00\x00\x08\x00\x01")
	b = binary.BigEndian.AppendUint16(b, 0x0112)
	b = binary.BigEndian.AppendUint16(b, 3)
	b = binary.BigEndian.AppendUint32(b, 1)
	b = binary.BigEndian.AppendUint16(b, orientation)
	return append(b, 0, 0, 0, 0, 0, 0)
}

// The source image is
//
//	10 20 30
//	40 50 60
func TestOrientationRotatesThenMirrors(t *testing.T) {
	tests := []struct {
		name          string
		o             Orientation
		width, height int
		want          []byte
	}{
		{"none", Orientation{}, 3, 2, []byte{10, 20, 30, 40, 50, 60}},
		{"flip", Orientation{Flip: true}, 3, 2, []byte{30, 20, 10, 60, 50, 40}},
		{"90", Orientation{Angle: 90}, 2, 3, []byte{30, 60, 20, 50, 10, 40}},
		{"180", Orientation{Angle: 180}, 3, 2, []byte{60, 50, 40, 30, 20, 10}},
		{"270", Orientation{Angle: 270}, 2, 3, []byte{40, 10, 50, 20, 60, 30}},
		{"90 flip", Orientation{Angle: 90, Flip: true}, 2, 3, []byte{60, 30, 50, 20, 40, 10}},
		{"180 flip", Orientation{Angle: 180, Flip: true}, 3, 2, []byte{40, 50, 60, 10, 20, 30}},
		{"270 flip", Orientation{Angle: 270, Flip: true}, 2, 3, []byte{10, 40, 20, 50, 30, 60}},
	}
	data := rgbaPNG(t, gradient3x2())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRaster("o", data, tt.o, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if r.width != tt.width || r.height != tt.height {
				t.Errorf("size %d×%d, want %d×%d", r.width, r.height, tt.width, tt.height)
			}
			if got := reds(t, r); !bytes.Equal(got, tt.want) {
				t.Errorf("pixels %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrientationOfAlphaFollowsTheColor(t *testing.T) {
	img := gradient3x2()
	for i := 0; i < 6; i++ {
		img.Pix[4*i+3] = uint8(100 + i)
	}
	r, err := NewRaster("o", rgbaPNG(t, img), Orientation{Angle: 90}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := embed(t, r, false, 1)
	if got, want := unpredict(t, e.sub(t, "SMask")), []byte{102, 105, 101, 104, 100, 103}; !bytes.Equal(got, want) {
		t.Errorf("alpha %v, want %v", got, want)
	}
	if dictInt(t, e.sub(t, "SMask").Dict, "Width") != 2 || dictInt(t, e.obj.Dict, "Width") != 2 {
		t.Error("mask and image sizes differ")
	}
}

func TestExifOrientationOfPNG(t *testing.T) {
	// Each value is the rotation that makes the image upright, applied to
	// the pixel order above.
	want := map[uint16][]byte{
		1: {10, 20, 30, 40, 50, 60},
		2: {30, 20, 10, 60, 50, 40},
		3: {60, 50, 40, 30, 20, 10},
		4: {40, 50, 60, 10, 20, 30},
		5: {10, 40, 20, 50, 30, 60},
		6: {40, 10, 50, 20, 60, 30},
		7: {60, 30, 50, 20, 40, 10},
		8: {30, 60, 20, 50, 10, 40},
	}
	for orientation, pixels := range want {
		data := rgbaPNG(t, gradient3x2(), [2][]byte{[]byte("eXIf"), exifTIFF(orientation)})
		r, err := NewRaster("exif", data, OrientationFromImage, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if got := reds(t, r); !bytes.Equal(got, pixels) {
			t.Errorf("orientation %d: pixels %v, want %v", orientation, got, pixels)
		}
	}
}

func TestExplicitOrientationIgnoresExif(t *testing.T) {
	data := rgbaPNG(t, gradient3x2(), [2][]byte{[]byte("eXIf"), exifTIFF(6)})
	for name, o := range map[string]Orientation{"none": {}, "flip": {Flip: true}} {
		r, err := NewRaster("exif", data, o, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if r.width != 3 {
			t.Errorf("%s: width %d, the Exif rotation was applied", name, r.width)
		}
	}
}

func TestRotatedJPEGStaysJPEG(t *testing.T) {
	r := loadResource(t, "not-optimized-exif.jpg", OrientationFromImage, Options{})
	e := embed(t, r, false, 1)
	if dictName(e.obj.Dict, "Filter") != "DCTDecode" {
		t.Errorf("filter = %v", e.get(t, "Filter"))
	}
	if bytes.Equal(e.obj.Data, readResource(t, "not-optimized-exif.jpg")) {
		t.Error("the JPEG was not rewritten")
	}
}

func TestRotatingAnImageTwiceGivesTheOriginal(t *testing.T) {
	data := rgbaPNG(t, gradient3x2())
	r, _ := NewRaster("a", data, Orientation{Angle: 90}, Options{})
	again, err := NewRaster("b", rgbaPNG(t, toNRGBA(rasterImage(t, r))), Orientation{Angle: 270}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := reds(t, again); !bytes.Equal(got, []byte{10, 20, 30, 40, 50, 60}) {
		t.Errorf("pixels %v", got)
	}
}

// withExif inserts an Exif segment right after the start of image marker.
func withExif(t testing.TB, jpegData, tiff []byte) []byte {
	t.Helper()
	body := append([]byte("Exif\x00\x00"), tiff...)
	segment := []byte{0xff, 0xe1, byte((len(body) + 2) >> 8), byte(len(body) + 2)}
	segment = append(segment, body...)
	return append(append(append([]byte{}, jpegData[:2]...), segment...), jpegData[2:]...)
}

func TestJPEGWithUprightExifIsRecompressed(t *testing.T) {
	plain := readResource(t, "not-optimized.jpg")
	data := withExif(t, plain, exifTIFF(1))

	explicit, err := NewRaster("explicit", data, Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(explicit.jpeg, data) {
		t.Error("an explicit orientation must keep the file")
	}

	fromImage, err := NewRaster("exif", data, OrientationFromImage, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(fromImage.jpeg, data) {
		t.Fatal("the JPEG was embedded unchanged")
	}
	if fromImage.width != explicit.width || fromImage.height != explicit.height {
		t.Errorf("size = %dx%d, want %dx%d", fromImage.width, fromImage.height, explicit.width, explicit.height)
	}
	if got := dictName(embed(t, fromImage, false, 1).obj.Dict, "Filter"); got != "DCTDecode" {
		t.Errorf("filter = %v", got)
	}
}
