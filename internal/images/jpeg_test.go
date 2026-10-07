package images

import (
	"bytes"
	"testing"
)

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
