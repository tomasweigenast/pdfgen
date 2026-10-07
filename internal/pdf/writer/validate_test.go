package writer

import (
	"bytes"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validate(t *testing.T, pdf []byte) *model.Context {
	t.Helper()
	api.DisableConfigDir()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	ctx, err := api.ReadValidateAndOptimize(bytes.NewReader(pdf), conf)
	if err != nil {
		t.Fatalf("pdfcpu rejects the output: %v", err)
	}
	return ctx
}

func TestOutputIsAcceptedByPDFParser(t *testing.T) {
	for _, c := range goldenCases {
		if !c.valid {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			ctx := validate(t, render(t, c))
			p, _ := c.build()
			if ctx.PageCount != len(p.PageRefs()) {
				t.Errorf("page count = %d, want %d", ctx.PageCount, len(p.PageRefs()))
			}
		})
	}
}

func TestCompressedStreamsAreAcceptedByPDFParser(t *testing.T) {
	p := New()
	var data bytes.Buffer
	for range 200 {
		data.WriteString("1 2 3 4 re\nf\n")
	}
	draw := p.Add(&Stream{Data: data.Bytes()})
	p.AddPage(pageDict(p.PagesRef(), draw))
	p.Info().Set("Title", Text("compressed"))

	for _, version := range []string{"1.4", "1.7", "2.0"} {
		var buf bytes.Buffer
		if err := p.Write(&buf, WriteOptions{Version: version, Compress: true}); err != nil {
			t.Fatal(err)
		}
		if ctx := validate(t, buf.Bytes()); ctx.PageCount != 1 {
			t.Errorf("version %s: page count = %d", version, ctx.PageCount)
		}
	}
}
