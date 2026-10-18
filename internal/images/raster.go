package images

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"math"

	xdraw "golang.org/x/image/draw"

	"github.com/tomasweigenast/pdfgen/internal/images/exif"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/units"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	_ "image/gif"
	_ "image/png"
)

// defaultJPEGQuality is the quality of JPEG data that has to be re-encoded
// when no quality was asked for.
const defaultJPEGQuality = 75

// pixmap is an image in the form it is written to the PDF: either JPEG data
// or 8-bit samples that are compressed on embedding.
type pixmap struct {
	width, height int
	// components is the number of color components per pixel: 1 (gray),
	// 3 (RGB) or 4 (CMYK).
	components int
	// jpeg, when set, holds the DCT-encoded image and samples is unused.
	jpeg []byte
	// samples holds the interleaved color components, row by row.
	samples []byte
	// alpha holds one opacity byte per pixel, or is nil for opaque images.
	alpha []byte
	// invert marks CMYK JPEG data stored with 255 meaning no ink.
	invert bool
}

// Raster is a bitmap image: JPEG, PNG, GIF, WebP, TIFF or BMP.
type Raster struct {
	pixmap
	id    string
	ratio float64
	opts  Options
}

// NewRaster decodes an image file. id names the image's XObject, so it must
// be unique per distinct image. A JPEG file is kept as it is unless the
// orientation (including the presence of Exif data when it comes from the
// image) or the options ask for a change; anything else is decoded.
func NewRaster(id string, data []byte, orientation Orientation, opts Options) (*Raster, error) {
	r := &Raster{id: id, opts: opts}

	isJPEG := bytes.HasPrefix(data, []byte{0xff, 0xd8})

	angle, flip := orientation.Angle, orientation.Flip
	// Applying the orientation of a JPEG that has an Exif block always
	// produces a new image, even when the block says it is upright, so the
	// file is recompressed instead of embedded as it is.
	transformed := false
	if orientation.FromImage {
		angle, flip = 0, false
		if tiff, ok := exif.Find(data); ok {
			if v, ok := exif.Orientation(tiff); ok {
				angle, flip = exifTransform(v)
			}
			transformed = isJPEG
		}
	}
	transformed = transformed || angle != 0 || flip

	var passthrough pixmap
	if isJPEG {
		info, err := scanJPEG(data)
		if err != nil {
			return nil, err
		}
		if info.components != 1 && info.components != 3 && info.components != 4 {
			return nil, fmt.Errorf("unsupported JPEG with %d components", info.components)
		}
		passthrough = pixmap{
			width: info.width, height: info.height, components: info.components,
			jpeg: data, invert: info.components == 4 && info.adobe,
		}
		if !transformed && !opts.OptimizeImages && opts.JPEGQuality == 0 {
			r.pixmap = passthrough
			return r.finish()
		}
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		if !isJPEG {
			return nil, err
		}
		// The decoder covers fewer JPEG variants than the PDF does, so the
		// file is still embedded as it is.
		logging.Warnf("Cannot decode JPEG image %s to rewrite it, embedding it unchanged: %s", id, err)
		r.pixmap = passthrough
		return r.finish()
	}
	pm := extract(img, format == "png")
	pm.samples, pm.width, pm.height = orient(pm.samples, pm.width, pm.height, pm.components, angle, flip)
	if pm.alpha != nil {
		pm.alpha, _, _ = orient(pm.alpha, img.Bounds().Dx(), img.Bounds().Dy(), 1, angle, flip)
	}
	r.pixmap = pm
	if isJPEG && pm.components != 4 {
		quality := cmpOr(opts.JPEGQuality, defaultJPEGQuality)
		encoded, err := encodeJPEG(pm, quality)
		if err != nil {
			return nil, err
		}
		r.pixmap = pixmap{width: pm.width, height: pm.height, components: pm.components, jpeg: encoded}
	}
	return r.finish()
}

func (r *Raster) finish() (*Raster, error) {
	if r.width <= 0 || r.height <= 0 {
		return nil, fmt.Errorf("image has no pixels")
	}
	r.ratio = math.Inf(1)
	if r.height != 0 {
		r.ratio = float64(r.width) / float64(r.height)
	}
	return r, nil
}

func cmpOr(v, def int) int {
	if v != 0 {
		return v
	}
	return def
}

// extract converts a decoded image to 8-bit samples. Gray and CMYK images
// keep their color model; everything else becomes RGB, with the opacity in a
// separate plane when the image is not opaque. PNG images with an alpha
// channel keep it even if every pixel is opaque.
func extract(img image.Image, isPNG bool) pixmap {
	b := img.Bounds()
	pm := pixmap{width: b.Dx(), height: b.Dy()}
	n := pm.width * pm.height

	switch img := img.(type) {
	case *image.Gray:
		pm.components = 1
		pm.samples = make([]byte, 0, n)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			i := img.PixOffset(b.Min.X, y)
			pm.samples = append(pm.samples, img.Pix[i:i+pm.width]...)
		}
		return pm
	case *image.Gray16:
		pm.components = 1
		pm.samples = make([]byte, 0, n)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				pm.samples = append(pm.samples, img.Pix[img.PixOffset(x, y)])
			}
		}
		return pm
	case *image.CMYK:
		pm.components = 4
		pm.samples = make([]byte, 0, 4*n)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			i := img.PixOffset(b.Min.X, y)
			pm.samples = append(pm.samples, img.Pix[i:i+4*pm.width]...)
		}
		return pm
	}

	hasAlpha := !isOpaque(img)
	if isPNG {
		switch img.(type) {
		case *image.NRGBA, *image.NRGBA64:
			hasAlpha = true
		}
	}
	nrgba := toNRGBA(img)
	pm.components = 3
	pm.samples = make([]byte, 0, 3*n)
	if hasAlpha {
		pm.alpha = make([]byte, 0, n)
	}
	for y := 0; y < pm.height; y++ {
		row := nrgba.Pix[y*nrgba.Stride : y*nrgba.Stride+4*pm.width]
		for x := 0; x < len(row); x += 4 {
			pm.samples = append(pm.samples, row[x], row[x+1], row[x+2])
			if hasAlpha {
				pm.alpha = append(pm.alpha, row[x+3])
			}
		}
	}
	if isPNG && hasAlpha && isGrayscale(pm.samples) {
		gray := make([]byte, n)
		for i := range gray {
			gray[i] = pm.samples[3*i]
		}
		pm.components, pm.samples = 1, gray
	}
	return pm
}

func isOpaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return false
}

func isGrayscale(rgb []byte) bool {
	for i := 0; i < len(rgb); i += 3 {
		if rgb[i] != rgb[i+1] || rgb[i] != rgb[i+2] {
			return false
		}
	}
	return true
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

// encodeJPEG compresses gray or RGB samples. CMYK is not supported by the
// encoder.
func encodeJPEG(pm pixmap, quality int) ([]byte, error) {
	var img image.Image
	switch pm.components {
	case 1:
		img = &image.Gray{Pix: pm.samples, Stride: pm.width, Rect: image.Rect(0, 0, pm.width, pm.height)}
	case 3:
		rgba := image.NewRGBA(image.Rect(0, 0, pm.width, pm.height))
		for i, j := 0, 0; i < len(pm.samples); i, j = i+3, j+4 {
			rgba.Pix[j], rgba.Pix[j+1], rgba.Pix[j+2], rgba.Pix[j+3] = pm.samples[i], pm.samples[i+1], pm.samples[i+2], 255
		}
		img = rgba
	default:
		return nil, fmt.Errorf("cannot encode %d-component JPEG", pm.components)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ID returns the name that identifies the image's XObject.
func (r *Raster) ID() string { return r.id }

// IntrinsicSize returns the pixel dimensions divided by the image
// resolution, in CSS pixels per dppx, and the aspect ratio.
func (r *Raster) IntrinsicSize(resolution, _ float64) (w, h, ratio units.MaybeFloat) {
	return units.F(float64(r.width) / resolution), units.F(float64(r.height) / resolution), units.F(r.ratio)
}

// Draw paints the image into a w×h box. When a maximum DPI is set and the
// box would show the image at a higher resolution, the XObject is built from
// a reduced copy.
func (r *Raster) Draw(s *stream.Stream, w, h float64, rendering string) {
	if r.width <= 0 || r.height <= 0 {
		return
	}
	ratio := 1.0
	if r.opts.DPI > 0 {
		const ptToIn = 4.0 / 3 / 96
		ctm := s.CTM()
		widthInches := math.Abs(w * ctm[0][0] * ptToIn)
		heightInches := math.Abs(h * ctm[1][1] * ptToIn)
		if widthInches > 0 && heightInches > 0 {
			dpi := max(float64(r.width)/widthInches, float64(r.height)/heightInches)
			if dpi > float64(r.opts.DPI) {
				ratio = float64(r.opts.DPI) / dpi
			}
		}
	}
	name := s.AddImage(r, rendering == "auto", ratio)
	s.Transform(w, 0, 0, -h, 0, h)
	s.DrawXObject(name)
}

// Embed writes the image XObject, with its soft mask.
func (r *Raster) Embed(pdf *writer.PDF, interpolate bool, dpiRatio float64) (writer.Ref, error) {
	pm := r.pixmap
	if dpiRatio < 1 {
		pm = r.reduced(dpiRatio)
	}

	ref := pdf.Reserve()
	dict := writer.NewDict(
		"Type", writer.Name("XObject"),
		"Subtype", writer.Name("Image"),
		"Width", pm.width,
		"Height", pm.height,
		"ColorSpace", colorSpace(pm.components),
		"BitsPerComponent", 8,
		"Interpolate", interpolate,
	)
	var data []byte
	if pm.jpeg != nil {
		if pm.invert {
			dict.Set("Decode", writer.Array{
				writer.Int(1), writer.Int(0), writer.Int(1), writer.Int(0),
				writer.Int(1), writer.Int(0), writer.Int(1), writer.Int(0),
			})
		}
		dict.Set("Filter", writer.Name("DCTDecode"))
		data = pm.jpeg
	} else {
		dict.Set("Filter", writer.Name("FlateDecode"))
		parms := writer.NewDict("Predictor", 15, "Columns", pm.width)
		if pm.components > 1 {
			parms.Set("Colors", writer.Int(pm.components))
		}
		dict.Set("DecodeParms", parms)
		data = deflate(predictorFilter(pm.samples, pm.width, pm.height, pm.components), r.opts.OptimizeImages)
	}

	if pm.alpha != nil {
		mask := writer.NewDict(
			"Filter", writer.Name("FlateDecode"),
			"Type", writer.Name("XObject"),
			"Subtype", writer.Name("Image"),
			"DecodeParms", writer.NewDict("Predictor", 15, "Columns", pm.width),
			"Width", pm.width,
			"Height", pm.height,
			"ColorSpace", writer.Name("DeviceGray"),
			"BitsPerComponent", 8,
			"Interpolate", interpolate,
		)
		maskData := deflate(predictorFilter(pm.alpha, pm.width, pm.height, 1), r.opts.OptimizeImages)
		dict.Set("SMask", pdf.Add(rawStream(mask, maskData)))
	}
	if err := pdf.Set(ref, rawStream(dict, data)); err != nil {
		return writer.Ref{}, err
	}
	return ref, nil
}

// rawStream returns a stream whose data is already encoded as its Filter
// says, so the writer must not compress it again.
func rawStream(dict *writer.Dict, data []byte) *writer.Stream {
	off := false
	return &writer.Stream{Dict: dict, Data: data, Compress: &off}
}

// colorSpace returns the device color space for n components. Embedded
// profiles are not used: they only apply through @color-profile.
func colorSpace(n int) writer.Name {
	switch n {
	case 1:
		return "DeviceGray"
	case 4:
		return "DeviceCMYK"
	}
	return "DeviceRGB"
}

// reduced returns the image scaled by ratio, or the image itself when it
// cannot be scaled.
func (r *Raster) reduced(ratio float64) pixmap {
	width := max(1, int(math.Round(float64(r.width)*ratio)))
	height := max(1, int(math.Round(float64(r.height)*ratio)))
	if width >= r.width && height >= r.height {
		return r.pixmap
	}

	src := r.pixmap
	if src.jpeg != nil {
		img, _, err := image.Decode(bytes.NewReader(src.jpeg))
		if err != nil {
			logging.Warnf("Cannot reduce the resolution of image %s: %s", r.id, err)
			return r.pixmap
		}
		src = extract(img, false)
	}
	out := pixmap{width: width, height: height, components: src.components}
	if src.alpha != nil {
		out.samples, out.alpha = scaleWithAlpha(src, width, height)
	} else {
		out.samples = scalePlanes(src.samples, src.width, src.height, src.components, width, height)
	}
	if r.jpeg != nil && out.components != 4 {
		encoded, err := encodeJPEG(out, defaultJPEGQuality)
		if err != nil {
			logging.Warnf("Cannot reduce the resolution of image %s: %s", r.id, err)
			return r.pixmap
		}
		out.jpeg, out.samples = encoded, nil
	}
	return out
}

// scalePlanes resamples interleaved samples one component at a time.
func scalePlanes(samples []byte, width, height, n, newWidth, newHeight int) []byte {
	out := make([]byte, newWidth*newHeight*n)
	for c := 0; c < n; c++ {
		src := image.NewGray(image.Rect(0, 0, width, height))
		for i := range src.Pix {
			src.Pix[i] = samples[i*n+c]
		}
		dst := image.NewGray(image.Rect(0, 0, newWidth, newHeight))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
		for i, v := range dst.Pix {
			out[i*n+c] = v
		}
	}
	return out
}

// scaleWithAlpha resamples with premultiplied colors, so that transparent
// pixels do not bleed their color into the edges.
func scaleWithAlpha(pm pixmap, newWidth, newHeight int) (samples, alpha []byte) {
	src := image.NewNRGBA(image.Rect(0, 0, pm.width, pm.height))
	for i := 0; i < pm.width*pm.height; i++ {
		if pm.components == 1 {
			g := pm.samples[i]
			src.Pix[4*i], src.Pix[4*i+1], src.Pix[4*i+2] = g, g, g
		} else {
			copy(src.Pix[4*i:4*i+3], pm.samples[3*i:3*i+3])
		}
		src.Pix[4*i+3] = pm.alpha[i]
	}
	dst := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	samples = make([]byte, 0, newWidth*newHeight*pm.components)
	alpha = make([]byte, 0, newWidth*newHeight)
	for i := 0; i < len(dst.Pix); i += 4 {
		if pm.components == 1 {
			samples = append(samples, dst.Pix[i])
		} else {
			samples = append(samples, dst.Pix[i:i+3]...)
		}
		alpha = append(alpha, dst.Pix[i+3])
	}
	return samples, alpha
}
