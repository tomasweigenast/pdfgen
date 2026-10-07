package stream_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

// testImage is a 2×1 gray image that counts how often it is embedded.
type testImage struct {
	id     string
	embeds int
	ratio  float64
	fail   bool
}

func (i *testImage) ID() string { return i.id }

func (i *testImage) Embed(pdf *writer.PDF, interpolate bool, dpiRatio float64) (writer.Ref, error) {
	if i.fail {
		return writer.Ref{}, errors.New("broken image")
	}
	i.embeds++
	i.ratio = dpiRatio
	off := false
	return pdf.Add(&writer.Stream{
		Dict: writer.NewDict(
			"Type", writer.Name("XObject"), "Subtype", writer.Name("Image"),
			"Width", 2, "Height", 1, "ColorSpace", writer.Name("DeviceGray"),
			"BitsPerComponent", 8, "Interpolate", interpolate),
		Data:     []byte{0, 255},
		Compress: &off,
	}), nil
}

func TestAddImageNamesDependOnIDAndInterpolation(t *testing.T) {
	s := stream.New(stream.NewResources(nil), stream.Options{})
	img := &testImage{id: "abc"}
	tests := []struct {
		interpolate bool
		want        string
	}{
		{true, "iabc1"},
		{false, "iabc0"},
		{true, "iabc1"},
	}
	for _, tt := range tests {
		if got := s.AddImage(img, tt.interpolate, 1); got != tt.want {
			t.Errorf("AddImage(interpolate=%v) = %q, want %q", tt.interpolate, got, tt.want)
		}
	}
}

func TestEmbedWritesSharedImageOnce(t *testing.T) {
	pdf := writer.New()
	res := stream.NewResources(nil)
	img := &testImage{id: "abc"}

	page1 := stream.New(res, stream.Options{})
	page1.AddImage(img, true, 0.5)
	page2 := stream.New(res, stream.Options{})
	page2.AddImage(img, true, 0.75)
	group := page2.AddGroup(0, 0, 1, 1)
	group.AddImage(img, true, 0.25)

	if err := res.Embed(pdf, pdf.Reserve()); err != nil {
		t.Fatal(err)
	}
	if img.embeds != 1 {
		t.Errorf("image embedded %d times, want 1", img.embeds)
	}
	if img.ratio != 0.75 {
		t.Errorf("dpi ratio = %v, want the largest, 0.75", img.ratio)
	}
	var out bytes.Buffer
	if err := pdf.Write(&out, writer.WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(out.Bytes(), []byte("/Subtype /Image")); n != 1 {
		t.Errorf("%d image objects in the file, want 1", n)
	}
}

func TestEmbedReportsImageErrors(t *testing.T) {
	pdf := writer.New()
	res := stream.NewResources(nil)
	stream.New(res, stream.Options{}).AddImage(&testImage{id: "x", fail: true}, false, 1)
	if err := res.Embed(pdf, pdf.Reserve()); err == nil {
		t.Fatal("Embed succeeded with a failing image")
	}
}
