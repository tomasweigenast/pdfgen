package writer

import (
	"bytes"
	"image/png"
	"os/exec"
	"strings"
	"testing"
)

// pixelColors maps the characters of the reference drawings to RGB values;
// 'z' matches any color.
var pixelColors = map[byte][3]uint8{
	'_': {255, 255, 255},
	'R': {255, 0, 0},
	'B': {0, 0, 255},
	'G': {0, 255, 0},
	'K': {0, 0, 0},
}

// assertPixels rasterizes the document with Ghostscript at one pixel per
// PDF point and compares it with the reference drawing.
func assertPixels(t *testing.T, p *PDF, opts WriteOptions, reference string) {
	t.Helper()
	gs, err := exec.LookPath("gs")
	if err != nil {
		t.Skip("gs not found")
	}
	var pdf bytes.Buffer
	if err := p.Write(&pdf, opts); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(gs, "-q", "-dNOPAUSE", "-dSAFER", "-sDEVICE=png16m",
		"-r576", "-dDownScaleFactor=8", "-sOutputFile=-", "-")
	cmd.Stdin = &pdf
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gs: %v\n%s", err, stderr.String())
	}
	img, err := png.Decode(&out)
	if err != nil {
		t.Fatal(err)
	}

	var rows []string
	for _, l := range strings.Split(reference, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			rows = append(rows, l)
		}
	}
	w, h := len(rows[0]), len(rows)
	if b := img.Bounds(); b.Dx() != w || b.Dy() != h {
		t.Fatalf("image is %dx%d, reference is %dx%d", b.Dx(), b.Dy(), w, h)
	}
	for y, row := range rows {
		if len(row) != w {
			t.Fatalf("reference row %d has width %d, want %d", y, len(row), w)
		}
		for x := range w {
			if row[x] == 'z' {
				continue
			}
			r, g, b, _ := img.At(x, y).RGBA()
			got := [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
			if want := pixelColors[row[x]]; got != want {
				t.Fatalf("pixel (%d, %d): got %v, want %v (%c)", x, y, got, want, row[x])
			}
		}
	}
}

func drawingDocument(ops ...string) (*PDF, Ref) {
	p := New()
	draw := p.Add(&Stream{Data: lines(ops...)})
	p.AddPage(pageDict(p.PagesRef(), draw))
	return p, draw
}

func TestRenderPaths(t *testing.T) {
	tests := []struct {
		name string
		ops  []string
		want string
	}{
		{"fill", []string{"2 2 5 6 re", "f"}, `
			__________
			__________
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__________
			__________`},
		{"stroke", []string{"2 2 5 6 re", "2 w", "S"}, `
			__________
			_KKKKKKK__
			_KKKKKKK__
			_KK___KK__
			_KK___KK__
			_KK___KK__
			_KK___KK__
			_KKKKKKK__
			_KKKKKKK__
			__________`},
		{"line", []string{"2 2 m", "2 w", "2 5 l", "S"}, `
			__________
			__________
			__________
			__________
			__________
			_KK_______
			_KK_______
			_KK_______
			__________
			__________`},
		{"stroke color", []string{"2 2 5 6 re", "2 w", "0 0 255 RG", "S"}, `
			__________
			_BBBBBBB__
			_BBBBBBB__
			_BB___BB__
			_BB___BB__
			_BB___BB__
			_BB___BB__
			_BBBBBBB__
			_BBBBBBB__
			__________`},
		{"fill color", []string{"2 2 5 6 re", "255 0 0 rg", "f"}, `
			__________
			__________
			__RRRRR___
			__RRRRR___
			__RRRRR___
			__RRRRR___
			__RRRRR___
			__RRRRR___
			__________
			__________`},
		{"dash", []string{"2 2 m", "2 w", "2 6 l", "[2 1] 0 d", "S"}, `
			__________
			__________
			__________
			__________
			_KK_______
			__________
			_KK_______
			_KK_______
			__________
			__________`},
		{"curve", []string{"2 5 m", "2 w", "2 5 3 5 5 5 c", "S"}, `
			__________
			__________
			__________
			__________
			__KKK_____
			__KKK_____
			__________
			__________
			__________
			__________`},
		{"curve from current point", []string{"2 5 m", "2 w", "3 5 5 5 v", "S"}, `
			__________
			__________
			__________
			__________
			__KKK_____
			__KKK_____
			__________
			__________
			__________
			__________`},
		{"curve to end point", []string{"2 5 m", "2 w", "3 5 5 5 y", "S"}, `
			__________
			__________
			__________
			__________
			__KKK_____
			__KKK_____
			__________
			__________
			__________
			__________`},
		{"matrix", []string{"1 0 0 1 1 1 cm", "2 2 m", "2 w", "2 5 l", "S"}, `
			__________
			__________
			__________
			__________
			__KK______
			__KK______
			__KK______
			__________
			__________
			__________`},
		{"fill and stroke", []string{"2 2 5 6 re", "2 w", "0 0 255 RG", "B"}, `
			__________
			_BBBBBBB__
			_BBBBBBB__
			_BBKKKBB__
			_BBKKKBB__
			_BBKKKBB__
			_BBKKKBB__
			_BBBBBBB__
			_BBBBBBB__
			__________`},
		{"clip", []string{"3 3 5 6 re", "4 3 2 6 re", "W", "n", "0 5 m", "10 5 l",
			"255 0 0 RG", "2 w", "S"}, `
			__________
			__________
			__________
			__________
			___RRRRR__
			___RRRRR__
			__________
			__________
			__________
			__________`},
		{"clip even-odd", []string{"3 3 5 6 re", "4 3 2 6 re", "W*", "n", "0 5 m", "10 5 l",
			"255 0 0 RG", "2 w", "S"}, `
			__________
			__________
			__________
			__________
			___R__RR__
			___R__RR__
			__________
			__________
			__________
			__________`},
		{"close", []string{"2 2 m", "2 8 l", "7 8 l", "7 2 l", "h", "0 0 255 RG", "2 w", "S"}, `
			__________
			_BBBBBBB__
			_BBBBBBB__
			_BB___BB__
			_BB___BB__
			_BB___BB__
			_BB___BB__
			_BBBBBBB__
			_BBBBBBB__
			__________`},
		{"stroke and close", []string{"2 2 m", "2 8 l", "7 8 l", "7 2 l", "0 0 255 RG", "2 w", "s"}, `
			__________
			_BBBBBBB__
			_BBBBBBB__
			_BB___BB__
			_BB___BB__
			_BB___BB__
			_BB___BB__
			_BBBBBBB__
			_BBBBBBB__
			__________`},
		{"fill stroke and close", []string{"2 2 m", "2 8 l", "7 8 l", "7 2 l",
			"255 0 0 rg", "0 0 255 RG", "2 w", "b"}, `
			__________
			_BBBBBBB__
			_BBBBBBB__
			_BBRRRBB__
			_BBRRRBB__
			_BBRRRBB__
			_BBRRRBB__
			_BBBBBBB__
			_BBBBBBB__
			__________`},
		{"save and restore state", []string{"2 2 5 6 re", "q", "4 4 2 2 re", "255 0 0 rg", "Q", "f"}, `
			__________
			__________
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__KKKKK___
			__________
			__________`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := drawingDocument(tt.ops...)
			assertPixels(t, p, WriteOptions{}, tt.want)
		})
	}
}

func TestRenderExtGState(t *testing.T) {
	p := New()
	gs := p.Add(NewDict("Type", Name("ExtGState"), "LW", 2))
	draw := p.Add(&Stream{Data: lines("2 2 5 6 re", "/GS gs", "S")})
	p.AddPage(pageDict(p.PagesRef(), draw, "Resources", NewDict(
		"ExtGState", NewDict("GS", gs))))
	assertPixels(t, p, WriteOptions{}, `
		__________
		_KKKKKKK__
		_KKKKKKK__
		_KK___KK__
		_KK___KK__
		_KK___KK__
		_KK___KK__
		_KKKKKKK__
		_KKKKKKK__
		__________`)
}

func TestRenderCompressedStream(t *testing.T) {
	p := New()
	draw := p.Add(&Stream{Data: lines("2 2 5 6 re", "f"), Compress: ptr(true)})
	p.AddPage(pageDict(p.PagesRef(), draw))
	var buf bytes.Buffer
	if err := p.Write(&buf, WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("2 2 5 6 re")) {
		t.Error("stream content is not compressed")
	}
	assertPixels(t, p, WriteOptions{}, `
		__________
		__________
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__________
		__________`)
}

func TestRenderCompressedObjectStream(t *testing.T) {
	p, _ := drawingDocument("2 2 5 6 re", "f")
	p.Info().Set("Title", Text("t"))
	assertPixels(t, p, WriteOptions{Compress: true}, `
		__________
		__________
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__KKKKK___
		__________
		__________`)
}

func TestRenderText(t *testing.T) {
	p := New()
	font := p.Add(NewDict(
		"Type", Name("Font"), "Subtype", Name("Type1"), "Name", Name("F1"),
		"BaseFont", Name("Helvetica"), "Encoding", Name("MacRomanEncoding")))
	draw := p.Add(&Stream{Data: lines(
		"BT", "/F1 200 Tf", "1 0 0 1 -20 5 Tm",
		"["+string(appendObject(nil, Text("l"), false))+"] TJ",
		"["+string(appendObject(nil, Text("É"), false))+"] TJ", "ET")})
	p.AddPage(pageDict(p.PagesRef(), draw, "Resources", NewDict(
		"ProcSet", Array{Name("PDF"), Name("Text")},
		"Font", NewDict("F1", font))))
	assertPixels(t, p, WriteOptions{}, `
		KKKKKKKKKK
		KKKKKKKKKK
		KKKKKKKKKK
		KKKKKKKKKK
		KKKKKKKKKK
		zzzzzzzzzz
		__________
		__________
		__________
		__________`)
}
