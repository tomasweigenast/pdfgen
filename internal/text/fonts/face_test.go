package fonts

import (
	"bytes"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func TestNewFaceFromOpenType(t *testing.T) {
	f, err := NewFace(readResource(t, "testfont.otf"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsCFF {
		t.Error("IsCFF = false, want true for an OpenType font with CFF outlines")
	}
	if f.Family == "" {
		t.Error("Family is empty")
	}
	if f.IsVariable() {
		t.Error("IsVariable = true for a static font")
	}
	if got := f.GoText.Upem(); got != 1024 {
		t.Errorf("units per em = %d, want 1024", got)
	}
}

func TestWOFFDecodesToSameFont(t *testing.T) {
	otf, err := NewFace(readResource(t, "testfont.otf"), 0)
	if err != nil {
		t.Fatal(err)
	}
	woff, err := NewFace(readResource(t, "testfont.woff"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(woff.Raw, []byte("OTTO")) && !bytes.HasPrefix(woff.Raw, []byte{0, 1, 0, 0}) {
		t.Errorf("Raw starts with % x, want an sfnt header", woff.Raw[:4])
	}
	for _, text := range []string{"abc", "kk", "liga", "woff font"} {
		if a, b := textWidth(t, otf, 16, text), textWidth(t, woff, 16, text); a != b {
			t.Errorf("width of %q: otf %v, woff %v", text, a, b)
		}
	}
}

func TestDecompressLeavesSfntUntouched(t *testing.T) {
	data := readResource(t, "testfont.otf")
	got, err := Decompress(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("sfnt data was modified")
	}
}

func TestNewFaceRejectsBadData(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":          nil,
		"garbage":        []byte("this is not a font"),
		"truncated woff": append([]byte("wOFF"), make([]byte, 8)...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewFace(data, 0); err == nil {
				t.Error("expected an error")
			}
		})
	}
	if _, err := NewFace(readResource(t, "testfont.otf"), 3); err == nil {
		t.Error("expected an error for a face index out of range")
	}
}

func TestFaceIDIsStable(t *testing.T) {
	data := readResource(t, "testfont.otf")
	a, _ := NewFace(data, 0)
	b, _ := NewFace(append([]byte(nil), data...), 0)
	other, _ := NewFace(readResource(t, "testfont.woff"), 0)
	if a.ID != b.ID {
		t.Errorf("same data gives IDs %q and %q", a.ID, b.ID)
	}
	if a.ID == "" || a.ID == other.ID && !bytes.Equal(a.Raw, other.Raw) {
		t.Errorf("IDs of different data must differ: %q, %q", a.ID, other.ID)
	}
}

func TestInstanceOfStaticFontIsSelf(t *testing.T) {
	f, _ := NewFace(readResource(t, "testfont.otf"), 0)
	if f.Instance([]font.Variation{{Tag: opentype.MustNewTag("wght"), Value: 700}}) != f {
		t.Error("static font produced a distinct instance")
	}
}

func staticVariableFace(axes ...Axis) *Face {
	return &Face{ID: "x-0", axes: axes}
}

func TestVariationsForVariableFace(t *testing.T) {
	wght, opsz := opentype.MustNewTag("wght"), opentype.MustNewTag("opsz")
	axes := []Axis{{Tag: wght, Min: 100, Default: 400, Max: 900}, {Tag: opsz, Min: 14, Default: 14, Max: 32}}
	settings := func(items ...props.FontSetting) props.FontSettings { return props.FontSettings{Items: items} }

	tests := []struct {
		name     string
		weight   int
		size     float64
		settings props.FontSettings
		want     []font.Variation
	}{
		{"weight and size", 700, 16, props.FontSettings{Normal: true},
			[]font.Variation{{Tag: wght, Value: 700}, {Tag: opsz, Value: 16}}},
		{"default weight", 0, 12, props.FontSettings{Normal: true},
			[]font.Variation{{Tag: wght, Value: 400}, {Tag: opsz, Value: 12}}},
		{"size is truncated to 1/1024 px", 400, 13.333, props.FontSettings{Normal: true},
			[]font.Variation{{Tag: wght, Value: 400}, {Tag: opsz, Value: 13.33203125}}},
		{"settings override weight", 400, 16, settings(props.FontSetting{Tag: "wght", Value: 650}),
			[]font.Variation{{Tag: wght, Value: 650}, {Tag: opsz, Value: 16}}},
		{"last setting wins", 400, 16, settings(props.FontSetting{Tag: "opsz", Value: 20}, props.FontSetting{Tag: "opsz", Value: 24}),
			[]font.Variation{{Tag: wght, Value: 400}, {Tag: opsz, Value: 24}}},
		{"other axes follow", 400, 16, settings(props.FontSetting{Tag: "wdth", Value: 80}),
			[]font.Variation{{Tag: wght, Value: 400}, {Tag: opsz, Value: 16}, {Tag: opentype.MustNewTag("wdth"), Value: 80}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := staticVariableFace(axes...).VariationsFor(tc.weight, tc.size, tc.settings)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("variation %d = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestVariationsForSkipsAxesTheFontLacks(t *testing.T) {
	f := staticVariableFace(Axis{Tag: opentype.MustNewTag("wdth"), Min: 75, Default: 100, Max: 125})
	got := f.VariationsFor(700, 16, props.FontSettings{Normal: true})
	if len(got) != 0 {
		t.Errorf("got %v, want no variations", got)
	}
}

func TestVariationKeyDistinguishesInstances(t *testing.T) {
	wght := opentype.MustNewTag("wght")
	a := variationsKey([]font.Variation{{Tag: wght, Value: 400}})
	b := variationsKey([]font.Variation{{Tag: wght, Value: 700}})
	if a == b || variationsKey(nil) != "" {
		t.Errorf("keys %q and %q", a, b)
	}
}
