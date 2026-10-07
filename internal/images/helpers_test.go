package images

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

func readResource(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testutil.ResourcePath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func loadResource(t testing.TB, name string, o Orientation, opts Options) *Raster {
	t.Helper()
	r, err := NewRaster("test", readResource(t, name), o, opts)
	if err != nil {
		t.Fatalf("NewRaster(%s): %v", name, err)
	}
	return r
}

// embedded is an image XObject written to a PDF.
type embedded struct {
	pdf *writer.PDF
	ref writer.Ref
	obj *writer.Stream
}

func embed(t testing.TB, r *Raster, interpolate bool, ratio float64) embedded {
	t.Helper()
	pdf := writer.New()
	ref, err := r.Embed(pdf, interpolate, ratio)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := pdf.Object(ref)
	return embedded{pdf, ref, o.(*writer.Stream)}
}

func (e embedded) get(t testing.TB, key writer.Name) writer.Object {
	t.Helper()
	v, ok := e.obj.Dict.Get(key)
	if !ok {
		return nil
	}
	return v
}

// sub returns the stream a reference-valued entry points to.
func (e embedded) sub(t testing.TB, key writer.Name) *writer.Stream {
	t.Helper()
	ref, ok := e.get(t, key).(writer.Ref)
	if !ok {
		t.Fatalf("/%s is not a reference", key)
	}
	o, _ := e.pdf.Object(ref)
	return o.(*writer.Stream)
}

func dictInt(t testing.TB, d *writer.Dict, key writer.Name) int {
	t.Helper()
	v, _ := d.Get(key)
	i, ok := v.(writer.Int)
	if !ok {
		t.Fatalf("/%s = %v, want an integer", key, v)
	}
	return int(i)
}

func dictName(d *writer.Dict, key writer.Name) writer.Name {
	v, _ := d.Get(key)
	n, _ := v.(writer.Name)
	return n
}

// unpredict inflates a Flate stream and undoes its PNG predictor
// (PNG 1.5 §9.2), returning the raw samples.
func unpredict(t testing.TB, s *writer.Stream) []byte {
	t.Helper()
	zr, err := zlib.NewReader(bytes.NewReader(s.Data))
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	width, _ := s.Dict.Get("Width")
	colors := 1
	if parms, ok := s.Dict.Get("DecodeParms"); ok {
		if c, ok := parms.(*writer.Dict).Get("Colors"); ok {
			colors = int(c.(writer.Int))
		}
	}
	stride := int(width.(writer.Int)) * colors
	if len(filtered)%(stride+1) != 0 {
		t.Fatalf("filtered data has %d bytes, not a multiple of %d", len(filtered), stride+1)
	}
	var out []byte
	for pos := 0; pos < len(filtered); pos += stride + 1 {
		kind, row := filtered[pos], filtered[pos+1:pos+1+stride]
		cur := make([]byte, stride)
		prior := make([]byte, stride)
		if len(out) > 0 {
			copy(prior, out[len(out)-stride:])
		}
		for x := range row {
			var left, upLeft byte
			if x >= colors {
				left, upLeft = cur[x-colors], prior[x-colors]
			}
			cur[x] = row[x] + predict(int(kind), left, prior[x], upLeft)
		}
		out = append(out, cur...)
	}
	return out
}

func rgbaPNG(t testing.TB, img image.Image, extra ...[2][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	for _, c := range extra {
		data = insertChunk(data, string(c[0]), c[1])
	}
	return data
}

// insertChunk adds a chunk right after IHDR.
func insertChunk(data []byte, kind string, body []byte) []byte {
	const afterIHDR = 8 + 8 + 13 + 4
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(body)))
	chunk = append(chunk, kind...)
	chunk = append(chunk, body...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
	out := append([]byte{}, data[:afterIHDR]...)
	out = append(out, chunk...)
	return append(out, data[afterIHDR:]...)
}

func zlibBytes(b []byte) []byte {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

// gradient3x2 is an image whose six pixels are all different, to tell
// orientations apart.
func gradient3x2() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(10 * (y*3 + x + 1)), 0, 0, 255})
		}
	}
	return img
}

// renderPDF draws on a page of width×height CSS pixels that rasterizes at one
// pixel per CSS pixel.
func renderPDF(t testing.TB, width, height float64, draw func(s *stream.Stream)) []byte {
	t.Helper()
	scale := testutil.PixelZoom * 0.75

	pdf := writer.New()
	colorSpaces := pdf.Add(writer.NewDict())
	resRef := pdf.Reserve()
	res := stream.NewResources(colorSpaces)

	s := stream.New(res, stream.Options{})
	s.Transform(1, 0, 0, -1, 0, height*scale)
	s.Transform(scale, 0, 0, scale, 0, 0)
	draw(s)

	contents := pdf.Add(s.Object())
	if err := res.Embed(pdf, resRef); err != nil {
		t.Fatal(err)
	}
	pdf.AddPage(writer.NewDict(
		"Type", writer.Name("Page"),
		"MediaBox", writer.Array{writer.Int(0), writer.Int(0), writer.Real(width * scale), writer.Real(height * scale)},
		"Contents", contents,
		"Resources", resRef,
	))
	var out bytes.Buffer
	if err := pdf.Write(&out, writer.WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// rasterImage returns the pixels of an opaque RGB raster.
func rasterImage(t testing.TB, r *Raster) image.Image {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, r.width, r.height))
	for i := 0; i < r.width*r.height; i++ {
		copy(img.Pix[4*i:], r.samples[3*i:3*i+3])
		img.Pix[4*i+3] = 255
	}
	return img
}
