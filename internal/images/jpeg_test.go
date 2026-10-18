package images

import (
	"bytes"
	"image"
	"testing"
)

func FuzzNewRaster(f *testing.F) {
	for _, name := range []string{"pattern.png", "pattern.gif", "blue.jpg", "cmyk_with_icc.jpg", "not-optimized-exif.jpg"} {
		f.Add(readResource(f, name))
	}
	f.Add(bytes.Repeat([]byte{0xff}, 40))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4<<20 {
			t.Skip()
		}
		if config, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
			width, height := int64(config.Width), int64(config.Height)
			if width < 0 || height < 0 || height != 0 && width > (4<<20)/height {
				t.Skip()
			}
		}
		r, err := NewRaster("fuzz", data, OrientationFromImage, Options{})
		if err != nil {
			return
		}
		embed(t, r, true, 0.5)
	})
}
