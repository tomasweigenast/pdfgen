package images

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

// patternSamples is pattern.png: a red pixel at the top left of a blue square.
func patternSamples() []byte {
	samples := make([]byte, 0, 4*4*3)
	for i := 0; i < 16; i++ {
		if i == 0 {
			samples = append(samples, 255, 0, 0)
		} else {
			samples = append(samples, 0, 0, 255)
		}
	}
	return samples
}

func TestRasterDecodesFormats(t *testing.T) {
	tests := []struct {
		file          string
		width, height int
		components    int
		jpeg, alpha   bool
	}{
		{"pattern.png", 4, 4, 3, false, false},
		{"pattern.palette.png", 4, 4, 3, false, false},
		{"pattern.gif", 4, 4, 3, false, false},
		{"icon.png", 16, 16, 3, false, false},
		{"logo_small.png", 100, 41, 3, false, true},
		{"blue.jpg", 4, 4, 3, true, false},
		{"not-optimized.jpg", 10, 10, 3, true, false},
		{"cmyk_with_icc.jpg", 2, 2, 4, true, false},
		{"cmyk_without_icc.jpg", 2, 2, 4, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			r := loadResource(t, tt.file, Orientation{}, Options{})
			if r.width != tt.width || r.height != tt.height || r.components != tt.components ||
				(r.jpeg != nil) != tt.jpeg || (r.alpha != nil) != tt.alpha {
				t.Errorf("got %d×%d, %d components, jpeg=%v alpha=%v; want %d×%d, %d, %v, %v",
					r.width, r.height, r.components, r.jpeg != nil, r.alpha != nil,
					tt.width, tt.height, tt.components, tt.jpeg, tt.alpha)
			}
		})
	}
}

func TestRasterStoresIndexedAndAnimatedImagesAsRGB(t *testing.T) {
	want := patternSamples()
	for _, file := range []string{"pattern.png", "pattern.palette.png", "pattern.gif"} {
		e := embed(t, loadResource(t, file, Orientation{}, Options{}), false, 1)
		if got := unpredict(t, e.obj); !bytes.Equal(got, want) {
			t.Errorf("%s samples = %v, want %v", file, got, want)
		}
		if cs := e.get(t, "ColorSpace"); cs != writer.Name("DeviceRGB") {
			t.Errorf("%s color space = %v", file, cs)
		}
	}
}

func TestRasterXObjectDictionary(t *testing.T) {
	e := embed(t, loadResource(t, "pattern.png", Orientation{}, Options{}), true, 1)
	d := e.obj.Dict
	if dictName(d, "Type") != "XObject" || dictName(d, "Subtype") != "Image" ||
		dictName(d, "Filter") != "FlateDecode" || dictName(d, "ColorSpace") != "DeviceRGB" {
		t.Errorf("unexpected dictionary %v", d)
	}
	if dictInt(t, d, "Width") != 4 || dictInt(t, d, "Height") != 4 || dictInt(t, d, "BitsPerComponent") != 8 {
		t.Errorf("unexpected geometry in %v", d)
	}
	if v, _ := d.Get("Interpolate"); v != writer.Bool(true) {
		t.Errorf("Interpolate = %v, want true", v)
	}
	parms, _ := d.Get("DecodeParms")
	p := parms.(*writer.Dict)
	if dictInt(t, p, "Predictor") != 15 || dictInt(t, p, "Columns") != 4 || dictInt(t, p, "Colors") != 3 {
		t.Errorf("DecodeParms = %v", p)
	}
	if _, ok := d.Get("SMask"); ok {
		t.Error("opaque image has a soft mask")
	}
	if e.obj.Compress == nil || *e.obj.Compress {
		t.Error("encoded data would be compressed again")
	}

	e = embed(t, loadResource(t, "pattern.png", Orientation{}, Options{}), false, 1)
	if v, _ := e.obj.Dict.Get("Interpolate"); v != writer.Bool(false) {
		t.Errorf("Interpolate = %v, want false", v)
	}
}

func TestRasterGrayImageUsesDeviceGray(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 3, 2))
	copy(img.Pix, []byte{0, 50, 100, 150, 200, 250})
	r, err := NewRaster("gray", rgbaPNG(t, img), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := embed(t, r, false, 1)
	if cs := e.get(t, "ColorSpace"); cs != writer.Name("DeviceGray") {
		t.Errorf("color space = %v", cs)
	}
	parms, _ := e.get(t, "DecodeParms").(*writer.Dict)
	if _, ok := parms.Get("Colors"); ok {
		t.Error("gray image sets Colors")
	}
	if got := unpredict(t, e.obj); !bytes.Equal(got, img.Pix) {
		t.Errorf("samples = %v, want %v", got, img.Pix)
	}
}

func TestRasterAlphaBecomesSoftMask(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{200, 100, 50, 255})
	img.SetNRGBA(1, 0, color.NRGBA{10, 20, 30, 64})
	r, err := NewRaster("alpha", rgbaPNG(t, img), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := embed(t, r, true, 1)
	if got, want := unpredict(t, e.obj), []byte{200, 100, 50, 10, 20, 30}; !bytes.Equal(got, want) {
		t.Errorf("color samples = %v, want %v", got, want)
	}
	mask := e.sub(t, "SMask")
	if dictName(mask.Dict, "ColorSpace") != "DeviceGray" || dictName(mask.Dict, "Subtype") != "Image" {
		t.Errorf("unexpected mask dictionary %v", mask.Dict)
	}
	if v, _ := mask.Dict.Get("Interpolate"); v != writer.Bool(true) {
		t.Errorf("mask Interpolate = %v", v)
	}
	if got, want := unpredict(t, mask), []byte{255, 64}; !bytes.Equal(got, want) {
		t.Errorf("mask samples = %v, want %v", got, want)
	}
}

func TestRasterGrayWithAlphaUsesDeviceGray(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{90, 90, 90, 255})
	img.SetNRGBA(1, 0, color.NRGBA{30, 30, 30, 128})
	r, err := NewRaster("ga", rgbaPNG(t, img), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := embed(t, r, false, 1)
	if cs := e.get(t, "ColorSpace"); cs != writer.Name("DeviceGray") {
		t.Errorf("color space = %v", cs)
	}
	if got := unpredict(t, e.obj); !bytes.Equal(got, []byte{90, 30}) {
		t.Errorf("samples = %v", got)
	}
}

func TestRasterTransparentGIFColorBecomesSoftMask(t *testing.T) {
	data := readResource(t, "pattern.gif")
	r, err := NewRaster("gif", data, Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.alpha != nil {
		t.Error("opaque GIF has alpha")
	}
}

func TestRasterReducesSixteenBitPNGToEightBits(t *testing.T) {
	img := image.NewRGBA64(image.Rect(0, 0, 1, 1))
	img.SetRGBA64(0, 0, color.RGBA64{0xff00, 0x8000, 0x0100, 0xffff})
	r, err := NewRaster("deep", rgbaPNG(t, img), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := embed(t, r, false, 1)
	if got := unpredict(t, e.obj); !bytes.Equal(got, []byte{255, 128, 1}) {
		t.Errorf("samples = %v", got)
	}
	if dictInt(t, e.obj.Dict, "BitsPerComponent") != 8 {
		t.Error("not 8 bits per component")
	}

	gray := image.NewGray16(image.Rect(0, 0, 2, 1))
	gray.SetGray16(0, 0, color.Gray16{0x1200})
	gray.SetGray16(1, 0, color.Gray16{0xff80})
	r, err = NewRaster("deepgray", rgbaPNG(t, gray), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := unpredict(t, embed(t, r, false, 1).obj); !bytes.Equal(got, []byte{0x12, 0xff}) {
		t.Errorf("gray samples = %v", got)
	}
}

func TestJPEGIsEmbeddedUnchanged(t *testing.T) {
	for _, file := range []string{"blue.jpg", "not-optimized.jpg", "cmyk_without_icc.jpg"} {
		data := readResource(t, file)
		e := embed(t, loadResource(t, file, Orientation{}, Options{}), true, 1)
		if !bytes.Equal(e.obj.Data, data) {
			t.Errorf("%s was changed", file)
		}
		if dictName(e.obj.Dict, "Filter") != "DCTDecode" {
			t.Errorf("%s filter = %v", file, e.get(t, "Filter"))
		}
		if _, ok := e.obj.Dict.Get("DecodeParms"); ok {
			t.Errorf("%s has DecodeParms", file)
		}
	}
}

func TestJPEGIsEmbeddedUnchangedWhenExifSaysUpright(t *testing.T) {
	data := readResource(t, "not-optimized.jpg")
	r, err := NewRaster("plain", data, OrientationFromImage, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.jpeg, data) {
		t.Error("JPEG without orientation was re-encoded")
	}
}

func TestAdobeCMYKJPEGIsInverted(t *testing.T) {
	e := embed(t, loadResource(t, "cmyk_without_icc.jpg", Orientation{}, Options{}), false, 1)
	if cs := e.get(t, "ColorSpace"); cs != writer.Name("DeviceCMYK") {
		t.Errorf("color space = %v", cs)
	}
	decode, _ := e.get(t, "Decode").(writer.Array)
	want := writer.Array{writer.Int(1), writer.Int(0), writer.Int(1), writer.Int(0), writer.Int(1), writer.Int(0), writer.Int(1), writer.Int(0)}
	if len(decode) != 8 {
		t.Fatalf("Decode = %v, want %v", decode, want)
	}
	for i := range want {
		if decode[i] != want[i] {
			t.Fatalf("Decode = %v, want %v", decode, want)
		}
	}

	e = embed(t, loadResource(t, "blue.jpg", Orientation{}, Options{}), false, 1)
	if _, ok := e.obj.Dict.Get("Decode"); ok {
		t.Error("RGB JPEG has a Decode array")
	}
}

func TestEmbeddedProfileBecomesICCBasedColorSpace(t *testing.T) {
	data := readResource(t, "cmyk_with_icc.jpg")
	e := embed(t, loadResource(t, "cmyk_with_icc.jpg", Orientation{}, Options{}), false, 1)
	cs, ok := e.get(t, "ColorSpace").(writer.Array)
	if !ok || len(cs) != 2 || cs[0] != writer.Name("ICCBased") {
		t.Fatalf("color space = %v", e.get(t, "ColorSpace"))
	}
	o, _ := e.pdf.Object(cs[1].(writer.Ref))
	profile := o.(*writer.Stream)
	if dictInt(t, profile.Dict, "N") != 4 || dictName(profile.Dict, "Alternate") != "DeviceCMYK" {
		t.Errorf("profile dictionary = %v", profile.Dict)
	}
	if !bytes.Contains(data, profile.Data[:128]) || len(profile.Data) < 128 {
		t.Error("profile is not the one in the file")
	}
	if profile.Compress != nil {
		t.Error("profile stream overrides compression")
	}

	e = embed(t, loadResource(t, "cmyk_without_icc.jpg", Orientation{}, Options{}), false, 1)
	if cs := e.get(t, "ColorSpace"); cs != writer.Name("DeviceCMYK") {
		t.Errorf("color space without profile = %v", cs)
	}
}

func TestPNGProfileBecomesICCBasedColorSpace(t *testing.T) {
	profile := make([]byte, 160)
	copy(profile[16:], "RGB ")
	copy(profile[36:], "acsp")
	iccp := append([]byte("test\x00\x00"), zlibBytes(profile)...)
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{1, 2, 3, 255})

	r, err := NewRaster("icc", rgbaPNG(t, img, [2][]byte{[]byte("iCCP"), iccp}), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := embed(t, r, false, 1)
	cs, ok := e.get(t, "ColorSpace").(writer.Array)
	if !ok {
		t.Fatalf("color space = %v", e.get(t, "ColorSpace"))
	}
	o, _ := e.pdf.Object(cs[1].(writer.Ref))
	if got := o.(*writer.Stream).Data; !bytes.Equal(got, profile) {
		t.Error("profile data differs")
	}
}

func TestProfileOfAnotherColorSpaceIsIgnored(t *testing.T) {
	profile := make([]byte, 160)
	copy(profile[16:], "CMYK")
	copy(profile[36:], "acsp")
	iccp := append([]byte("test\x00\x00"), zlibBytes(profile)...)
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{1, 2, 3, 255})
	r, err := NewRaster("icc", rgbaPNG(t, img, [2][]byte{[]byte("iCCP"), iccp}), Orientation{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if cs := embed(t, r, false, 1).get(t, "ColorSpace"); cs != writer.Name("DeviceRGB") {
		t.Errorf("color space = %v", cs)
	}
}

func TestJPEGRecompression(t *testing.T) {
	original := readResource(t, "not-optimized.jpg")
	tests := []struct {
		name string
		opts Options
	}{
		{"quality", Options{JPEGQuality: 30}},
		{"optimize", Options{OptimizeImages: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := embed(t, loadResource(t, "not-optimized.jpg", Orientation{}, tt.opts), false, 1)
			if dictName(e.obj.Dict, "Filter") != "DCTDecode" {
				t.Fatalf("filter = %v", e.get(t, "Filter"))
			}
			if bytes.Equal(e.obj.Data, original) {
				t.Error("JPEG was not re-encoded")
			}
			img, err := jpeg.Decode(bytes.NewReader(e.obj.Data))
			if err != nil || img.Bounds().Dx() != 10 {
				t.Errorf("re-encoded data is not a 10 pixel wide JPEG: %v", err)
			}
		})
	}

	low := embed(t, loadResource(t, "not-optimized.jpg", Orientation{}, Options{JPEGQuality: 5}), false, 1)
	high := embed(t, loadResource(t, "not-optimized.jpg", Orientation{}, Options{JPEGQuality: 95}), false, 1)
	if len(low.obj.Data) >= len(high.obj.Data) {
		t.Errorf("quality 5 gives %d bytes, quality 95 gives %d", len(low.obj.Data), len(high.obj.Data))
	}
}

func TestOptimizeShrinksFlateData(t *testing.T) {
	plain := embed(t, loadResource(t, "logo_small.png", Orientation{}, Options{}), false, 1)
	best := embed(t, loadResource(t, "logo_small.png", Orientation{}, Options{OptimizeImages: true}), false, 1)
	if len(best.obj.Data) > len(plain.obj.Data) {
		t.Errorf("optimized data is %d bytes, plain is %d", len(best.obj.Data), len(plain.obj.Data))
	}
	if !bytes.Equal(unpredict(t, best.obj), unpredict(t, plain.obj)) {
		t.Error("optimizing changed the pixels")
	}
}

func TestJPEGThatCannotBeDecodedIsEmbeddedUnrotated(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	r := loadResource(t, "cmyk_without_icc.jpg", Orientation{Angle: 90}, Options{})
	e := embed(t, r, false, 1)
	if !bytes.Equal(e.obj.Data, readResource(t, "cmyk_without_icc.jpg")) {
		t.Error("the file was changed")
	}
	if got := logs.Warnings(); len(got) != 1 {
		t.Errorf("warnings = %q, want one", got)
	}
}

func TestIntrinsicSize(t *testing.T) {
	r := loadResource(t, "logo_small.png", Orientation{}, Options{})
	w, h, ratio := r.IntrinsicSize(2, 16)
	if w.V != 50 || h.V != 20.5 || math.Abs(ratio.V-100.0/41) > 1e-12 || w.Auto || h.Auto || ratio.Auto {
		t.Errorf("IntrinsicSize(2) = %v, %v, %v", w, h, ratio)
	}
}

func TestRasterRejectsInvalidData(t *testing.T) {
	tests := map[string][]byte{
		"empty":           nil,
		"text":            []byte("not an image"),
		"truncated PNG":   readResource(t, "pattern.png")[:30],
		"truncated JPEG":  readResource(t, "blue.jpg")[:12],
		"JPEG scan only":  {0xff, 0xd8, 0xff, 0xda, 0, 2},
		"JPEG 12 bit":     {0xff, 0xd8, 0xff, 0xc0, 0, 11, 12, 0, 1, 0, 1, 1, 1, 0x11, 0, 0xff, 0xd9},
		"JPEG arithmetic": {0xff, 0xd8, 0xff, 0xc9, 0, 11, 8, 0, 1, 0, 1, 1, 1, 0x11, 0, 0xff, 0xd9},
	}
	for name, data := range tests {
		if _, err := NewRaster("bad", data, Orientation{}, Options{}); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestEmbedIsDeterministic(t *testing.T) {
	for _, file := range []string{"logo_small.png", "blue.jpg", "pattern.gif"} {
		var first []byte
		for i := 0; i < 3; i++ {
			r := loadResource(t, file, Orientation{}, Options{OptimizeImages: true})
			e := embed(t, r, true, 0.5)
			var buf bytes.Buffer
			if err := e.pdf.Write(&buf, writer.WriteOptions{}); err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = buf.Bytes()
			} else if !bytes.Equal(first, buf.Bytes()) {
				t.Fatalf("%s: output differs between runs", file)
			}
		}
	}
}

func TestPredictorFilterRoundTrips(t *testing.T) {
	const width, height, bpp = 7, 9, 3
	samples := make([]byte, width*height*bpp)
	for i := range samples {
		samples[i] = byte(i*i + 3*i)
	}
	e := embedded{obj: &writer.Stream{
		Data: zlibBytes(predictorFilter(samples, width, height, bpp)),
		Dict: writer.NewDict("Width", width, "DecodeParms", writer.NewDict("Colors", bpp)),
	}}
	if got := unpredict(t, e.obj); !bytes.Equal(got, samples) {
		t.Error("samples differ after filtering and unfiltering")
	}
}
