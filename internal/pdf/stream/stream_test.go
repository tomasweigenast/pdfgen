package stream

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/matrix"
	"github.com/tomasweigenast/pdfgen/internal/pdf/writer"
)

func newTestStream() *Stream {
	return New(NewResources(writer.Ref{Num: 9}), Options{})
}

func rgba(r, g, b, a float64) syntax.Color {
	return syntax.NewColor(syntax.SRGB, []float64{r, g, b}, a)
}

func assertContent(t *testing.T, s *Stream, want string) {
	t.Helper()
	if got := string(s.Bytes()); got != want {
		t.Errorf("content:\n%q\nwant:\n%q", got, want)
	}
}

func TestPopStateRemovesEmptyPair(t *testing.T) {
	s := newTestStream()
	s.PushState()
	s.PopState()
	assertContent(t, s, "")

	s.PushState()
	s.PushState()
	s.Rectangle(0, 0, 1, 1)
	s.PopState()
	s.PopState()
	assertContent(t, s, "q\nq\n0 0 1 1 re\nQ\nQ")
	if got := s.CTM(); got != matrix.Identity {
		t.Errorf("CTM = %v after balanced states", got)
	}
}

func TestTransformAccumulatesAndRestores(t *testing.T) {
	s := newTestStream()
	s.Transform(1, 0, 0, -1, 0, 10)
	s.PushState()
	s.Transform(2, 0, 0, 2, 5, 0)
	x, y := s.CTM().TransformPoint(1, 1)
	if x != 7 || y != 8 {
		t.Errorf("point (1,1) maps to (%v,%v), want (7,8)", x, y)
	}
	s.Rectangle(0, 0, 1, 1)
	s.PopState()
	x, y = s.CTM().TransformPoint(1, 1)
	if x != 1 || y != 9 {
		t.Errorf("after PopState (1,1) maps to (%v,%v), want (1,9)", x, y)
	}
	assertContent(t, s, "1 0 0 -1 0 10 cm\nq\n2 0 0 2 5 0 cm\n0 0 1 1 re\nQ")
}

func TestSetColorSkipsRepeatedColors(t *testing.T) {
	s := newTestStream()
	red := rgba(1, 0, 0, 1)
	s.SetColor(red, false)
	s.SetColor(red, false)
	s.SetColor(red, true)
	s.SetColor(rgba(1, 0, 0, 0.5), false)
	assertContent(t, s, "/a1.0 gs\n1 0 0 rg\n/A1.0 gs\n1 0 0 RG\n/a0.5 gs")

	s.PushState()
	s.PopState()
	s.SetColor(red, false)
	assertContent(t, s, "/a1.0 gs\n1 0 0 rg\n/A1.0 gs\n1 0 0 RG\n/a0.5 gs\n/a1.0 gs\n1 0 0 rg")
}

func TestSetColorSpaces(t *testing.T) {
	tests := []struct {
		name  string
		color syntax.Color
		want  string
	}{
		{"hsl", syntax.NewColor(syntax.HSL, []float64{120, 100, 50}, 1), "/a1.0 gs\n0 1 0 rg"},
		{"cmyk", syntax.NewColor(syntax.DeviceCMYK, []float64{0, 1, 0, 0}, 1), "/a1.0 gs\n/DeviceCMYK cs\n0 1 0 0 scn"},
		{"profile", syntax.NewColor("--brand", []float64{0.1, 0.2, 0.3}, 1), "/a1.0 gs\n/--brand cs\n0.1 0.2 0.3 scn"},
		{"lab", syntax.NewColor(syntax.Lab, []float64{50, 20, -30}, 1), "/a1.0 gs\n/lab-d50 cs\n50 20 -30 scn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(NewResources(nil), Options{ColorProfiles: map[string]bool{"--brand": true}})
			s.SetColor(tt.color, false)
			assertContent(t, s, tt.want)
		})
	}
}

func TestOklabColorsUseTheD65LabSpace(t *testing.T) {
	s := newTestStream()
	s.SetColor(syntax.NewColor(syntax.Oklab, []float64{1, 0, 0}, 1), false)
	got := string(s.Bytes())
	if !strings.HasPrefix(got, "/a1.0 gs\n/lab-d65 cs\n") || !strings.HasSuffix(got, " scn") {
		t.Errorf("content = %q", got)
	}
}

func TestSetColorUnsupportedSpaceFallsBackToRGB(t *testing.T) {
	capture := logging.StartCapture()
	defer capture.Close()

	s := newTestStream()
	s.SetColor(syntax.NewColor(syntax.DisplayP3, []float64{0.25, 0.5, 0.75}, 1), false)
	assertContent(t, s, "/a1.0 gs\n0.25 0.5 0.75 rg")
	if w := capture.Warnings(); len(w) != 1 {
		t.Errorf("warnings = %v, want one", w)
	}

	s = newTestStream()
	s.SetColor(syntax.NewColor("--unknown", []float64{0.5}, 1), true)
	assertContent(t, s, "/A1.0 gs\n0.5 0 0 RG")
}

func TestSetAlphaRegistersSharedStates(t *testing.T) {
	s := newTestStream()
	both := true
	s.SetAlpha(0.5, true, &both)
	s.SetAlpha(0.5, true, &both)
	s.PushState()
	s.PopState()
	s.SetAlpha(0.5, false, nil)
	s.SetAlpha(0.25, true, nil)
	assertContent(t, s, "/A0.5 gs\n/a0.5 gs\n/a0.5 gs\n/A0.25 gs")
	if n := s.res.extGState.len(); n != 3 {
		t.Errorf("%d ExtGState entries, want 3 (A0.5, a0.5, A0.25)", n)
	}
}

func TestPyFloat(t *testing.T) {
	tests := map[float64]string{0: "0.0", 1: "1.0", 0.5: "0.5", 0.25: "0.25", 1e-5: "1e-05", 0.0001: "0.0001", 0.3: "0.3"}
	for in, want := range tests {
		if got := pyFloat(in); got != want {
			t.Errorf("pyFloat(%v) = %q, want %q", in, got, want)
		}
	}
	if got := pyFloat(math.Nextafter(0.1, 1)); got != "0.10000000000000002" {
		t.Errorf("pyFloat shortest = %q", got)
	}
}

func TestTextObjectsAreMergedAndKeepTheirFont(t *testing.T) {
	s := newTestStream()
	s.BeginText()
	s.SetFontSize("F1", 12)
	s.SetFontSize("F1", 12)
	s.ShowTextString("a")
	s.EndText()
	s.BeginText()
	s.SetFontSize("F1", 12)
	s.ShowTextString("b")
	s.SetFontSize("F1", 14)
	s.EndText()
	s.BeginText()
	s.SetFontSize("F1", 14)
	s.EndText()
	assertContent(t, s, "BT\n/F1 12 Tf\n(a) Tj\n(b) Tj\n/F1 14 Tf\nET")
}

func TestFontSelectionIsForgottenAfterPopState(t *testing.T) {
	s := newTestStream()
	s.BeginText()
	s.PushState()
	s.SetFontSize("F1", 12)
	s.PopState()
	s.SetFontSize("F1", 12)
	assertContent(t, s, "BT\nq\n/F1 12 Tf\nQ\n/F1 12 Tf")
}

func TestGroupsPatternsAndShadingsAreNumbered(t *testing.T) {
	s := newTestStream()
	g0 := s.AddGroup(0, 0, 10, 10)
	g1 := s.AddGroup(0, 0, 10, 10)
	p0 := s.AddPattern(0, 0, 4, 4, 4, 4, matrix.Identity)
	fn := InterpolationFunction([]float64{0, 1}, []float64{1, 0, 0}, []float64{0, 0, 1}, 1)
	sh0 := s.AddShading(2, []float64{0, 1}, []float64{0, 0, 10, 0}, true, fn, "")
	sh1 := s.AddShading(3, []float64{0, 1}, []float64{0, 0, 0, 0, 0, 5}, false, fn, "DeviceGray")
	if g0.ID() != "x0" || g1.ID() != "x1" || p0.ID() != "p0" || sh0 != "s0" || sh1 != "s1" {
		t.Errorf("names = %s %s %s %s %s", g0.ID(), g1.ID(), p0.ID(), sh0, sh1)
	}

	// A nested group numbers its own resources.
	inner := g0.AddGroup(0, 0, 1, 1)
	if inner.ID() != "x0" {
		t.Errorf("nested group id = %s, want x0", inner.ID())
	}
	if g0.res.ColorSpace != s.res.ColorSpace {
		t.Error("groups must share the color space resource")
	}
}

func TestSoftMaskStateUsesAGroup(t *testing.T) {
	s := newTestStream()
	mask := s.SetAlphaState(0, 0, 8, 8, "luminosity")
	if mask.ID() != "x0" {
		t.Errorf("mask group id = %s", mask.ID())
	}
	s.SetBlendMode("Multiply")
	assertContent(t, s, "/s0 gs\n/s1 gs")
}

func TestFunctions(t *testing.T) {
	f0 := InterpolationFunction([]float64{0, 1}, []float64{1, 0.5}, []float64{0, 0}, 1)
	f1 := InterpolationFunction([]float64{0, 1}, []float64{0, 0}, []float64{1, 1}, 1)
	st := StitchingFunction([]float64{0, 1}, []float64{0, 1, 0, 1}, []float64{0.5}, []Function{f0, f1})
	got := string(writer.Append(nil, st))
	want := "<</FunctionType 3/Domain [0 1]/Encode [0 1 0 1]/Bounds [0.5]/Functions [" +
		"<</FunctionType 2/Domain [0 1]/C0 [1 0.5]/C1 [0 0]/N 1>> " +
		"<</FunctionType 2/Domain [0 1]/C0 [0 0]/C1 [1 1]/N 1>>]>>"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestEmbedWritesResourceTree(t *testing.T) {
	pdf := writer.New()
	resRef := pdf.Reserve()
	res := NewResources(writer.Name("CS"))
	res.Font = writer.Ref{Num: 7}
	s := New(res, Options{})
	mask := s.SetAlphaState(0, 0, 4, 4, "alpha")
	mask.SetColor(rgba(1, 1, 1, 0.5), false)
	if err := res.Embed(pdf, resRef); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := pdf.Write(&out, writer.WriteOptions{OmitID: true}); err != nil {
		t.Fatal(err)
	}
	file := out.String()
	for _, want := range []string{
		"/SMask <</Type /Mask/S /Alpha/G 4 0 R>>",
		"/Subtype /Form",
		"/Group <</Type /Group/S /Transparency/I true/CS /DeviceRGB>>",
		"/ColorSpace /CS/Font 7 0 R",
		"/a0.5 <</ca 0.5>>",
	} {
		if !strings.Contains(file, want) {
			t.Errorf("output lacks %q:\n%s", want, file)
		}
	}
}

func TestLabColorSpaces(t *testing.T) {
	got := string(writer.Append(nil, LabColorSpaces()))
	want := "<</lab-d50 [/Lab <</WhitePoint [0.964296 1 0.825105]/Range [-125 125 -125 125]>>]" +
		"/lab-d65 [/Lab <</WhitePoint [0.950456 1 1.089058]/Range [-125 125 -125 125]>>]>>"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
