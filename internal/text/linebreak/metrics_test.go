package linebreak

import "testing"

func TestStrut(t *testing.T) {
	ctx := newTestContext(t)
	const baseline = 12.796875
	tests := []struct {
		lineHeight         LineHeight
		height, baselineAt float64
	}{
		{LineHeight{Normal: true}, 16, baseline},
		{LineHeight{Number: true, Value: 1.5}, 24, baseline + 4},
		{LineHeight{Value: 20}, 20, baseline + 2},
		{LineHeight{Value: 10}, 10, baseline - 3},
	}
	for _, tt := range tests {
		style := testStyle()
		style.LineHeight = tt.lineHeight
		h, b := Strut(ctx, style)
		if h != tt.height || b != tt.baselineAt {
			t.Errorf("%+v: strut %v, %v; want %v, %v", tt.lineHeight, h, b, tt.height, tt.baselineAt)
		}
	}
	style := testStyle()
	style.Font.Size = 0
	if h, b := Strut(ctx, style); h != 0 || b != 0 {
		t.Errorf("zero font size: strut %v, %v", h, b)
	}
}

func TestStrutIgnoresSpacing(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.WordSpacing = 10
	style.LetterSpacing = 3
	if h, b := Strut(ctx, style); h != 16 || b != 12.796875 {
		t.Errorf("strut %v, %v; want 16, 12.796875", h, b)
	}
}

func TestCharacterRatioOfStyle(t *testing.T) {
	ctx := newTestContext(t)
	if r := CharacterRatio(ctx, testStyle(), "ex"); r != 0.7998 {
		t.Errorf("ex = %v, want 0.7998", r)
	}
	if r := CharacterRatio(ctx, testStyle(), "ch"); r != 1 {
		t.Errorf("ch = %v, want 1", r)
	}
}
