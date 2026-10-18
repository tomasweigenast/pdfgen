package fonts_test

import (
	"bytes"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/tomasweigenast/pdfgen/internal/pdf/fonts"
	"github.com/tomasweigenast/pdfgen/internal/pdf/stream"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	textfonts "github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

func validate(t *testing.T, pdf []byte) {
	t.Helper()
	api.DisableConfigDir()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	if _, err := api.ReadValidateAndOptimize(bytes.NewReader(pdf), conf); err != nil {
		t.Fatalf("pdfcpu rejects the output: %v", err)
	}
}

func inkPixels(img *image.NRGBA) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if c := img.NRGBAAt(x, y); c.R < 128 {
				n++
			}
		}
	}
	return n
}

func TestCFFTextIsDrawnFromTheSubset(t *testing.T) {
	face := testFace(t)
	pdf := buildPDF(t, 9, 6, fonts.BuildOptions{}, fonts.NewRegistry(false), func(s *stream.Stream) {
		drawText(s, face, 3, 0, 4, "A A")
	})
	testutil.AssertPixelsPDF(t, `
		_________
		_________
		KKK___KKK
		KKK___KKK
		KKK___KKK
		_________
	`, pdf)
	validate(t, pdf)
}

func TestFullAndSubsetFontsDrawTheSameText(t *testing.T) {
	for name, face := range map[string]func(testing.TB) *textfonts.Face{
		"cff":      testFace,
		"truetype": systemTrueType,
	} {
		t.Run(name, func(t *testing.T) {
			f := face(t)
			render := func(full bool) []byte {
				// Hinting programs stay in both, so that they render alike.
				return buildPDF(t, 40, 14, fonts.BuildOptions{Hinting: true}, fonts.NewRegistry(full), func(s *stream.Stream) {
					drawText(s, f, 12, 1, 11, "Ag Bh")
				})
			}
			sub, full := render(false), render(true)
			validate(t, sub)
			validate(t, full)
			if len(sub) >= len(full) && name == "truetype" {
				t.Errorf("subset (%d bytes) is not smaller than the full font (%d bytes)", len(sub), len(full))
			}
			imgSub := testutil.RasterizePDF(t, sub, testutil.PixelZoom)
			imgFull := testutil.RasterizePDF(t, full, testutil.PixelZoom)
			if inkPixels(imgSub) == 0 {
				t.Fatal("the subset drew nothing")
			}
			if !bytes.Equal(imgSub.Pix, imgFull.Pix) {
				t.Errorf("subset and full font render differently (%d and %d ink pixels)", inkPixels(imgSub), inkPixels(imgFull))
			}
		})
	}
}

func TestMissingGlyphsShowNotdefOutline(t *testing.T) {
	face := testFace(t)
	for _, full := range []bool{false, true} {
		pdf := buildPDF(t, 6, 6, fonts.BuildOptions{}, fonts.NewRegistry(full), func(s *stream.Stream) {
			font := s.AddFont(face)
			cid := font.MissingCID('☃', 1000)
			font.SetText(cid, "☃")
			s.BeginText()
			s.SetTextMatrix(1, 0, 0, -1, 0, 4)
			s.SetFont(font, 4)
			s.ShowGlyphs([]fonts.TJElement{{CID: cid}})
			s.EndText()
		})
		validate(t, pdf)
		if n := inkPixels(testutil.RasterizePDF(t, pdf, testutil.PixelZoom)); n == 0 {
			t.Errorf("full=%v: .notdef was drawn empty", full)
		}
	}
}

func TestAdjustmentsMoveFollowingGlyphs(t *testing.T) {
	face := testFace(t)
	pdf := buildPDF(t, 9, 6, fonts.BuildOptions{}, fonts.NewRegistry(false), func(s *stream.Stream) {
		font := s.AddFont(face)
		a, _ := face.GoText.NominalGlyph('A')
		cid := font.CID(uint16(a))
		s.BeginText()
		s.SetTextMatrix(1, 0, 0, -1, 0, 4)
		s.SetFont(font, 3)
		// A thousand units is a whole em: the second glyph moves 3 px right.
		s.ShowGlyphs([]fonts.TJElement{{CID: cid, Adjust: -1000}, {CID: cid}})
		s.EndText()
	})
	testutil.AssertPixelsPDF(t, `
		_________
		_________
		KKK___KKK
		KKK___KKK
		KKK___KKK
		_________
	`, pdf)
}

func TestFontObjectsAreWrittenAsType0(t *testing.T) {
	for name, tc := range map[string]struct {
		face func(testing.TB) *textfonts.Face
		want []string
	}{
		"cff":      {testFace, []string{"/CIDFontType0", "/FontFile3", "/OpenType", "/Identity-H", "/CIDToGIDMap /Identity", "/ToUnicode"}},
		"truetype": {systemTrueType, []string{"/CIDFontType2", "/FontFile2", "/Length1", "/Identity-H", "/CIDToGIDMap /Identity"}},
	} {
		t.Run(name, func(t *testing.T) {
			f := tc.face(t)
			pdf := string(buildPDF(t, 10, 10, fonts.BuildOptions{}, fonts.NewRegistry(false), func(s *stream.Stream) {
				drawText(s, f, 5, 0, 5, "AB")
			}))
			for _, want := range tc.want {
				if !strings.Contains(pdf, want) {
					t.Errorf("PDF lacks %q", want)
				}
			}
			if !strings.Contains(pdf, "/BaseFont /") {
				t.Error("PDF lacks BaseFont")
			}
		})
	}
}

func TestCIDSetIsWrittenForOldVersions(t *testing.T) {
	face := testFace(t)
	draw := func(s *stream.Stream) { drawText(s, face, 3, 0, 4, "AB") }
	old := string(buildPDF(t, 6, 6, fonts.BuildOptions{PDFVersion: "1.4"}, fonts.NewRegistry(false), draw))
	current := string(buildPDF(t, 6, 6, fonts.BuildOptions{PDFVersion: "1.7"}, fonts.NewRegistry(false), draw))
	if !strings.Contains(old, "/CIDSet") {
		t.Error("PDF 1.4 font descriptor lacks CIDSet")
	}
	if strings.Contains(current, "/CIDSet") {
		t.Error("PDF 1.7 font descriptor has a CIDSet")
	}
}

func TestOutputIsDeterministic(t *testing.T) {
	for name, face := range map[string]func(testing.TB) *textfonts.Face{
		"cff":      testFace,
		"truetype": systemTrueType,
	} {
		t.Run(name, func(t *testing.T) {
			f := face(t)
			var first []byte
			for range 2 {
				pdf := buildPDF(t, 30, 10, fonts.BuildOptions{Compress: true}, fonts.NewRegistry(false), func(s *stream.Stream) {
					drawText(s, f, 8, 0, 8, "BAC")
					font := s.AddFont(f)
					font.MissingCID('☃', 800)
				})
				if first == nil {
					first = pdf
				} else if !bytes.Equal(first, pdf) {
					t.Error("two builds of the same document differ")
				}
			}
		})
	}
}

func TestTextCanBeExtractedWithClusters(t *testing.T) {
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext not found in PATH")
	}
	face := testFace(t)
	pdf := buildPDF(t, 40, 10, fonts.BuildOptions{Compress: true}, fonts.NewRegistry(false), func(s *stream.Stream) {
		font := s.AddFont(face)
		var els []fonts.TJElement
		// One glyph standing for two characters, as a ligature does.
		for _, c := range []struct {
			r    rune
			text string
		}{{'A', "fi"}, {'B', "x"}, {'C', "y"}} {
			g, _ := face.GoText.NominalGlyph(c.r)
			cid := font.CID(uint16(g))
			font.SetText(cid, c.text)
			els = append(els, fonts.TJElement{CID: cid})
		}
		s.BeginText()
		s.SetTextMatrix(1, 0, 0, -1, 0, 8)
		s.SetFont(font, 8)
		s.ShowGlyphs(els)
		s.EndText()
	})
	file := filepath.Join(t.TempDir(), "text.pdf")
	if err := os.WriteFile(file, pdf, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(path, file, "-").CombinedOutput()
	if err != nil {
		t.Fatalf("pdftotext: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "fixy" {
		t.Errorf("extracted text = %q, want %q", got, "fixy")
	}
}

func TestUnhintedTrueTypeSubsetKeepsTheShapes(t *testing.T) {
	face := systemTrueType(t)
	render := func(opts fonts.BuildOptions) *image.NRGBA {
		pdf := buildPDF(t, 60, 30, opts, fonts.NewRegistry(false), func(s *stream.Stream) {
			drawText(s, face, 24, 1, 22, "Rg")
		})
		validate(t, pdf)
		return testutil.RasterizePDF(t, pdf, testutil.PixelZoom)
	}
	plain, hinted := inkPixels(render(fonts.BuildOptions{})), inkPixels(render(fonts.BuildOptions{Hinting: true}))
	if plain == 0 {
		t.Fatal("the unhinted subset drew nothing")
	}
	if diff := plain - hinted; diff > hinted*15/100 || -diff > hinted*15/100 {
		t.Errorf("ink pixels: %d unhinted, %d hinted", plain, hinted)
	}
}
