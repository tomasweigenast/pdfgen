package fonts

import (
	"testing"
)

func TestToUnitsTruncates(t *testing.T) {
	tests := []struct {
		px   float64
		want int32
	}{
		{16, 16384},
		{13.333, 13652},
		{2, 2048},
		{1000, 1024000},
		{0.0009, 0},
		{1.0 / 3, 341},
	}
	for _, tc := range tests {
		if got := ToUnits(tc.px); got != tc.want {
			t.Errorf("ToUnits(%v) = %d, want %d", tc.px, got, tc.want)
		}
	}
	if got := FromUnits(13652); got != 13.33203125 {
		t.Errorf("FromUnits(13652) = %v", got)
	}
}

func TestMetricsOfTestFont(t *testing.T) {
	f, err := NewFace(readResource(t, "testfont.otf"), 0)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		size                float64
		ascent, descent     int32
		underline, ulThick  int32
		strike, strikeThick int32
		height, baseline    float64
	}{
		{16, 13104, 3280, -2496, 320, 4240, 816, 16, 12.796875},
		{13.333, 10919, 2733, -2080, 267, 3533, 680, 13.33203125, 10.6630859375},
		{2, 1638, 410, -312, 40, 530, 102, 2, 1.599609375},
		{1000, 819000, 205000, -156000, 20000, 265000, 51000, 1000, 799.8046875},
	}
	for _, tc := range tests {
		m := f.Metrics(tc.size)
		got := [6]int32{m.Ascent, m.Descent, m.UnderlinePosition, m.UnderlineThickness, m.StrikethroughPosition, m.StrikethroughThickness}
		want := [6]int32{tc.ascent, tc.descent, tc.underline, tc.ulThick, tc.strike, tc.strikeThick}
		if got != want {
			t.Errorf("size %v: metrics %v, want %v", tc.size, got, want)
		}
		if m.UnitsPerEm != 1024 || m.Size != ToUnits(tc.size) {
			t.Errorf("size %v: UnitsPerEm = %d, Size = %d", tc.size, m.UnitsPerEm, m.Size)
		}
		h, b := m.Strut(nil)
		if h != tc.height || b != tc.baseline || m.Height() != tc.height || m.Baseline() != tc.baseline {
			t.Errorf("size %v: strut (%v, %v), want (%v, %v)", tc.size, h, b, tc.height, tc.baseline)
		}
	}
}

func TestStrutWithLineHeight(t *testing.T) {
	f, _ := NewFace(readResource(t, "testfont.otf"), 0)
	m := f.Metrics(16)
	lh := 40.0
	h, b := m.Strut(&lh)
	if h != 40 || b != 12.796875+(40-16)/2 {
		t.Errorf("Strut = (%v, %v)", h, b)
	}
}

// OS/2 fsSelection bit 7 (USE_TYPO_METRICS) selects the typographic
// ascender and descender over the hhea ones, and the line gap never adds to
// the height of a line.
func TestMetricsSource(t *testing.T) {
	data := readResource(t, "testfont.otf")
	be16 := func(v int16) []byte { return []byte{byte(v >> 8), byte(v)} }

	t.Run("line gap is ignored", func(t *testing.T) {
		patched := patchedFont(t, data, "hhea", 8, be16(200))
		f, err := NewFace(patched, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := f.Metrics(16), (func() Metrics { g, _ := NewFace(data, 0); return g.Metrics(16) })(); got != want {
			t.Errorf("metrics changed with the line gap: %+v, want %+v", got, want)
		}
	})

	t.Run("hhea values by default", func(t *testing.T) {
		patched := patchedFont(t, data, "OS/2", 68, append(be16(500), be16(-100)...))
		f, err := NewFace(patched, 0)
		if err != nil {
			t.Fatal(err)
		}
		m := f.Metrics(16)
		if m.Ascent != 13104 || m.Descent != 3280 {
			t.Errorf("ascent, descent = %d, %d, want hhea values 13104, 3280", m.Ascent, m.Descent)
		}
	})

	t.Run("typo values with USE_TYPO_METRICS", func(t *testing.T) {
		patched := patchedFont(t, data, "OS/2", 68, append(be16(512), be16(-256)...))
		patched = patchedFont(t, patched, "OS/2", 62, []byte{0, 0x80})
		f, err := NewFace(patched, 0)
		if err != nil {
			t.Fatal(err)
		}
		m := f.Metrics(16)
		if m.Ascent != 8192 || m.Descent != 4096 {
			t.Errorf("ascent, descent = %d, %d, want typo values 8192, 4096", m.Ascent, m.Descent)
		}
	})
}

func TestCharacterRatioOfTestFont(t *testing.T) {
	f, _ := NewFace(readResource(t, "testfont.otf"), 0)
	tests := []struct {
		unit string
		want float64
	}{
		{"ex", 0.7998},
		{"ch", 1},
		{"cap", 1},
		{"ic", 2},
	}
	for _, tc := range tests {
		if got := f.CharacterRatio(tc.unit, "", nil); got != tc.want {
			t.Errorf("CharacterRatio(%q) = %v, want %v", tc.unit, got, tc.want)
		}
	}
	if got := f.CharacterRatio("em", "", nil); got != 0 {
		t.Errorf("CharacterRatio(em) = %v, want 0", got)
	}
}

// The cap ratio measures the advance of "O", not the height of capitals,
// and ex measures the top of the ink of "x", not the OS/2 x-height.
func TestCharacterRatioMeasures(t *testing.T) {
	f, _ := NewFace(readResource(t, "testfont.otf"), 0)
	ink, advance := f.measureRune('x', ToUnits(1000), "", nil)
	if advance != 1024000 {
		t.Errorf("advance of x = %d", advance)
	}
	if ink != 819000 {
		t.Errorf("ink top of x = %d", ink)
	}
	if got, want := ratio(FromUnits(advance)), 1.0; got != want {
		t.Errorf("ratio = %v, want %v", got, want)
	}
}

func TestRatioRounding(t *testing.T) {
	tests := []struct {
		measure float64
		want    float64
	}{
		// 0.515625 is exactly representable, so the tie goes to the even digit.
		{515.625, 0.51562},
		{515.626, 0.51563},
		{546.875, 0.54688},
		{0, 0.5},
		{0.001, 0.5},
		{1000, 1},
	}
	for _, tc := range tests {
		if got := ratio(tc.measure); got != tc.want {
			t.Errorf("ratio(%v) = %v, want %v", tc.measure, got, tc.want)
		}
	}
}

func TestRatioCharacter(t *testing.T) {
	for unit, want := range map[string]rune{"ex": 'x', "cap": 'O', "ic": '水', "ch": '0'} {
		if got, ok := RatioCharacter(unit); !ok || got != want {
			t.Errorf("RatioCharacter(%q) = %q, %v", unit, got, ok)
		}
	}
	if _, ok := RatioCharacter("lh"); ok {
		t.Error("lh is not a character unit")
	}
}

func TestWOFFMetricsMatchOpenType(t *testing.T) {
	otf, _ := NewFace(readResource(t, "testfont.otf"), 0)
	woff, err := NewFace(readResource(t, "testfont.woff"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := otf.Metrics(16), woff.Metrics(16); a != b {
		t.Errorf("metrics differ: otf %+v, woff %+v", a, b)
	}
}

// At 7.5 px the descender of the test font scales to 1537.5 units, which
// table values round down to 1537: the line is exactly 7.5 px high.
func TestAscentAndDescentRoundLikeTableValues(t *testing.T) {
	f, err := NewFace(readResource(t, "testfont.otf"), 0)
	if err != nil {
		t.Fatal(err)
	}
	m := f.Metrics(7.5)
	if m.Ascent != 6143 || m.Descent != 1537 {
		t.Errorf("ascent %d, descent %d; want 6143 and 1537", m.Ascent, m.Descent)
	}
}
