package images

import (
	"bytes"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

const centered = `
    ________
    ________
    __rBBB__
    __BBBB__
    __BBBB__
    __BBBB__
    ________
    ________
`

const resized = `
    ____________
    ____________
    __rrBBBBBB__
    __rrBBBBBB__
    __BBBBBBBB__
    __BBBBBBBB__
    __BBBBBBBB__
    __BBBBBBBB__
    __BBBBBBBB__
    __BBBBBBBB__
    ____________
    ____________
`

const blueSquare = `
    ________
    ________
    __aaaa__
    __aaaa__
    __aaaa__
    __aaaa__
    ________
    ________
`

// drawAt paints img into a w×h box whose top left corner is at (x, y).
func drawAt(s *stream.Stream, img Image, x, y, w, h float64, rendering string) {
	s.PushState()
	s.Transform(1, 0, 0, 1, x, y)
	img.Draw(s, w, h, rendering)
	s.PopState()
}

func TestDrawPaintsImagesAtPixelScale(t *testing.T) {
	tests := []struct {
		file, want string
	}{
		{"pattern.png", centered},
		{"pattern.palette.png", centered},
		{"pattern.gif", centered},
		{"blue.jpg", blueSquare},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			r := loadResource(t, tt.file, Orientation{}, Options{})
			pdf := renderPDF(t, 8, 8, func(s *stream.Stream) { drawAt(s, r, 2, 2, 4, 4, "auto") })
			testutil.AssertPixelsPDF(t, tt.want, pdf)
		})
	}
}

func TestDrawResizesPixelatedImages(t *testing.T) {
	for _, file := range []string{"pattern.png", "pattern.palette.png", "pattern.gif"} {
		t.Run(file, func(t *testing.T) {
			r := loadResource(t, file, Orientation{}, Options{})
			pdf := renderPDF(t, 12, 12, func(s *stream.Stream) { drawAt(s, r, 2, 2, 8, 8, "pixelated") })
			testutil.AssertPixelsPDF(t, resized, pdf)
		})
	}
}

func TestDrawInterpolatesOnlyForAutoRendering(t *testing.T) {
	r := loadResource(t, "pattern.png", Orientation{}, Options{})
	for rendering, want := range map[string]string{"auto": "/Interpolate true", "pixelated": "/Interpolate false", "crisp-edges": "/Interpolate false"} {
		pdf := renderPDF(t, 12, 12, func(s *stream.Stream) { drawAt(s, r, 2, 2, 8, 8, rendering) })
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("image-rendering %s: %q not found", rendering, want)
		}
	}
}

func TestDrawPaintsTransparentImagesOverTheBackground(t *testing.T) {
	r := loadResource(t, "logo_small.png", Orientation{}, Options{})
	pdf := renderPDF(t, 100, 41, func(s *stream.Stream) { drawAt(s, r, 0, 0, 100, 41, "auto") })
	pixels := testutil.RasterizePDF(t, pdf, testutil.PixelZoom)
	w, h, got := testutil.ImagePixels(pixels)
	if w != 100 || h != 41 {
		t.Fatalf("size %d×%d", w, h)
	}
	white := 0
	for _, p := range got {
		if p.R == 255 && p.G == 255 && p.B == 255 {
			white++
		}
	}
	if white == 0 || white == len(got) {
		t.Errorf("%d of %d pixels are white, want the transparent part only", white, len(got))
	}
}

func TestDrawPaintsCMYKImagesWithDeviceColors(t *testing.T) {
	// The embedded profile does not change the rendering.
	for _, file := range []string{"cmyk_without_icc.jpg", "cmyk_with_icc.jpg"} {
		t.Run(file, func(t *testing.T) {
			r := loadResource(t, file, Orientation{}, Options{})
			pdf := renderPDF(t, 2, 2, func(s *stream.Stream) { drawAt(s, r, 0, 0, 2, 2, "auto") })
			testutil.AssertPixelsPDF(t, "QQ\nQQ", pdf)
		})
	}
}

func TestDrawAppliesExifOrientation(t *testing.T) {
	const page = 10
	render := func(file string, o Orientation) []byte {
		r := loadResource(t, file, o, Options{})
		return renderPDF(t, page, page, func(s *stream.Stream) { drawAt(s, r, 0, 0, page, page, "auto") })
	}
	plain := testutil.RasterizePDF(t, render("not-optimized.jpg", Orientation{}), testutil.PixelZoom)
	for name, pdf := range map[string][]byte{
		"from the file":   render("not-optimized-exif.jpg", OrientationFromImage),
		"explicit 180deg": render("not-optimized-exif.jpg", Orientation{Angle: 180}),
	} {
		_, _, want := testutil.ImagePixels(plain)
		w, h, got := testutil.ImagePixels(testutil.RasterizePDF(t, pdf, testutil.PixelZoom))
		testutil.AssertPixelsEqual(t, strings.ReplaceAll(t.Name()+"_"+name, " ", "_"), w, h, got, want, 25)
	}
	unrotated := testutil.RasterizePDF(t, render("not-optimized-exif.jpg", Orientation{}), testutil.PixelZoom)
	_, _, want := testutil.ImagePixels(unrotated)
	_, _, rotated := testutil.ImagePixels(testutil.RasterizePDF(t, render("not-optimized-exif.jpg", OrientationFromImage), testutil.PixelZoom))
	same := true
	for i := range want {
		if want[i] != rotated[i] {
			same = false
		}
	}
	if same {
		t.Error("the Exif orientation had no effect")
	}
}

func TestDrawKeepsJPEGFormatAfterRotation(t *testing.T) {
	r := loadResource(t, "not-optimized-exif.jpg", Orientation{Angle: 180}, Options{})
	pdf := renderPDF(t, 10, 10, func(s *stream.Stream) { drawAt(s, r, 0, 0, 10, 10, "auto") })
	if !bytes.Contains(pdf, []byte("/Filter /DCTDecode")) {
		t.Error("rotated JPEG is not DCT encoded")
	}
}

// pageStream returns a stream on the CSS pixel scale of a PDF page, in
// which one CSS pixel is 0.75 point.
func pageStream() (*stream.Stream, *stream.Resources) {
	res := stream.NewResources(nil)
	s := stream.New(res, stream.Options{})
	s.Transform(0.75, 0, 0, 0.75, 0, 0)
	return s, res
}

// embeddedImages returns the image XObjects of a document in object order.
func embeddedImages(t *testing.T, res *stream.Resources) []*writer.Stream {
	t.Helper()
	pdf := writer.New()
	if err := res.Embed(pdf, pdf.Reserve()); err != nil {
		t.Fatal(err)
	}
	var images []*writer.Stream
	for n := 1; ; n++ {
		o, ok := pdf.Object(writer.Ref{Num: n})
		if !ok {
			return images
		}
		if st, ok := o.(*writer.Stream); ok && dictName(st.Dict, "Subtype") == "Image" && dictName(st.Dict, "ColorSpace") != "DeviceGray" {
			images = append(images, st)
		}
	}
}

func TestDrawReducesImagesAboveTheMaximumDPI(t *testing.T) {
	tests := []struct {
		name  string
		dpi   int
		sizes []float64
		width int
	}{
		{"no limit", 0, []float64{1}, 4},
		{"below the limit", 96, []float64{4}, 4},
		{"half the resolution", 48, []float64{4}, 2},
		{"the largest use decides", 48, []float64{4, 2, 1}, 2},
		{"a big use keeps the full image", 48, []float64{4, 8}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := loadResource(t, "pattern.png", Orientation{}, Options{DPI: tt.dpi})
			s, res := pageStream()
			for _, size := range tt.sizes {
				drawAt(s, r, 0, 0, size, size, "auto")
			}
			imgs := embeddedImages(t, res)
			if len(imgs) != 1 {
				t.Fatalf("%d images embedded, want 1", len(imgs))
			}
			if got := dictInt(t, imgs[0].Dict, "Width"); got != tt.width {
				t.Errorf("width = %d, want %d", got, tt.width)
			}
		})
	}
}

func TestReducedImagesKeepTheirKind(t *testing.T) {
	t.Run("jpeg", func(t *testing.T) {
		e := embed(t, loadResource(t, "not-optimized.jpg", Orientation{}, Options{}), false, 0.5)
		img, err := jpeg.Decode(bytes.NewReader(e.obj.Data))
		if err != nil || img.Bounds().Dx() != 5 || dictInt(t, e.obj.Dict, "Width") != 5 {
			t.Errorf("not a 5 pixel wide JPEG: %v", err)
		}
	})
	t.Run("alpha", func(t *testing.T) {
		e := embed(t, loadResource(t, "logo_small.png", Orientation{}, Options{}), false, 0.5)
		mask := e.sub(t, "SMask")
		if dictInt(t, e.obj.Dict, "Width") != 50 || dictInt(t, mask.Dict, "Width") != 50 || dictInt(t, mask.Dict, "Height") != 21 {
			t.Errorf("sizes %v, mask %v", e.obj.Dict, mask.Dict)
		}
		if got := len(unpredict(t, mask)); got != 50*21 {
			t.Errorf("mask has %d samples", got)
		}
	})
	t.Run("never below one pixel", func(t *testing.T) {
		e := embed(t, loadResource(t, "pattern.png", Orientation{}, Options{}), false, 0.01)
		if dictInt(t, e.obj.Dict, "Width") != 1 || dictInt(t, e.obj.Dict, "Height") != 1 {
			t.Errorf("size %v", e.obj.Dict)
		}
	})
}

func TestDrawSharesImagesBetweenStreams(t *testing.T) {
	r := loadResource(t, "blue.jpg", Orientation{}, Options{})
	s, res := pageStream()
	drawAt(s, r, 0, 0, 4, 4, "auto")
	other := stream.New(res, stream.Options{})
	drawAt(other, r, 0, 0, 4, 4, "auto")
	drawAt(other, r, 0, 0, 4, 4, "pixelated")
	if got := len(embeddedImages(t, res)); got != 2 {
		t.Errorf("%d image objects, want one per interpolation setting", got)
	}
	if !strings.Contains(string(s.Bytes()), "/itest1 Do") || !strings.Contains(string(other.Bytes()), "/itest0 Do") {
		t.Errorf("contents %q, %q", s.Bytes(), other.Bytes())
	}
}
