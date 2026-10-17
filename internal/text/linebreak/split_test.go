package linebreak

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/text/fonts"
)

var (
	sansFonts = []string{"DejaVu Sans", "sans"}
	monoFonts = []string{"DejaVu Sans Mono", "monospace"}
)

// newSystemContext returns a context with the installed fonts, and skips
// the test when there are none.
func newSystemContext(t *testing.T) *Context {
	t.Helper()
	fc := fonts.NewFontConfiguration()
	if fc.Query(fonts.Description{Families: monoFonts, Size: 16}, "", 'a') == nil {
		t.Skip("no installed fonts")
	}
	return NewContext(fc)
}

// makeText splits text with the initial style in a monospace font, changed
// by set.
func makeText(ctx *Context, text string, maxWidth *float64, set func(*Style)) SplitResult {
	style := NewStyle(16, monoFonts...)
	if set != nil {
		set(style)
	}
	return SplitFirstLine(ctx, text, style, maxWidth, 0, true, false)
}

func sans(size float64) func(*Style) {
	return func(s *Style) {
		s.Font.Families = sansFonts
		s.Font.Size = size
	}
}

func size(px float64) func(*Style) {
	return func(s *Style) { s.Font.Size = px }
}

func TestLineContent(t *testing.T) {
	ctx := newSystemContext(t)
	for _, tt := range []struct {
		width     float64
		remaining string
	}{
		{100, "text for test"},
		{45, "is a text for test"},
	} {
		text := "This is a text for test"
		r := makeText(ctx, text, width(tt.width), sans(19))
		if got := text[r.ResumeIndex:]; got != tt.remaining {
			t.Errorf("width %v: remaining %q, want %q", tt.width, got, tt.remaining)
		}
		// The space between the lines is removed.
		if r.Length+1 != r.ResumeIndex {
			t.Errorf("width %v: length %d, resume at %d", tt.width, r.Length, r.ResumeIndex)
		}
	}
}

func TestLineWithAnyWidth(t *testing.T) {
	ctx := newSystemContext(t)
	w1 := makeText(ctx, "some text", nil, nil).Width
	w2 := makeText(ctx, "some text some text", nil, nil).Width
	if w1 >= w2 {
		t.Errorf("widths %v and %v, want the first one smaller", w1, w2)
	}
}

func TestLineBreaking(t *testing.T) {
	ctx := newSystemContext(t)
	text := "Thïs is a text for test"
	if r := makeText(ctx, text, width(90), size(1)); r.ResumeIndex != -1 {
		t.Errorf("1px text is split at %d", r.ResumeIndex)
	}
	if r := makeText(ctx, text, width(90), size(100)); text[r.ResumeIndex:] != "is a text for test" {
		t.Errorf("100px text: remaining %q", text[r.ResumeIndex:])
	}
	if r := makeText(ctx, text, width(100), sans(19)); text[r.ResumeIndex:] != "text for test" {
		t.Errorf("19px text: remaining %q", text[r.ResumeIndex:])
	}
}

func TestLineBreakingRightToLeft(t *testing.T) {
	ctx := newSystemContext(t)
	text := "لوريم ايبسوم دولا"
	if r := makeText(ctx, text, width(90), size(1)); r.ResumeIndex != -1 {
		t.Errorf("1px text is split at %d", r.ResumeIndex)
	}
	want := "ايبسوم دولا"
	if r := makeText(ctx, text, width(90), size(100)); text[r.ResumeIndex:] != want {
		t.Errorf("100px text: remaining %q, want %q", text[r.ResumeIndex:], want)
	}
}

func TestTextDimension(t *testing.T) {
	ctx := newSystemContext(t)
	text := "This is a text for test. This is a test for text.py"
	r1 := makeText(ctx, text, width(200), size(12))
	r2 := makeText(ctx, text, width(200), size(20))
	if r1.Width*r1.Height >= r2.Width*r2.Height {
		t.Errorf("12px line %vx%v is not smaller than 20px line %vx%v", r1.Width, r1.Height, r2.Width, r2.Height)
	}
}

func TestWholeTextFits(t *testing.T) {
	ctx := newTestContext(t)
	r := SplitFirstLine(ctx, "abc def", testStyle(), width(200), 0, true, false)
	if r.Length != 7 || r.ResumeIndex != -1 || r.Width != 112 || r.Height != 16 || r.Baseline != 12.796875 {
		t.Errorf("got %+v", r)
	}
	r = SplitFirstLine(ctx, "abc def", testStyle(), nil, 0, true, false)
	if r.Length != 7 || r.ResumeIndex != -1 || r.Width != 112 {
		t.Errorf("without width: got %+v", r)
	}
}

func TestTrailingSpacesAreLeftOutWhenSpacesCollapse(t *testing.T) {
	ctx := newTestContext(t)
	r := SplitFirstLine(ctx, "abc def", testStyle(), width(100), 0, true, false)
	if r.Layout.Text != "abc" || r.Length != 3 || r.ResumeIndex != 4 || r.Width != 48 {
		t.Errorf("got text %q, length %d, resume %d, width %v", r.Layout.Text, r.Length, r.ResumeIndex, r.Width)
	}
	style := testStyle()
	style.WhiteSpace = "pre-wrap"
	r = SplitFirstLine(ctx, "abc   def", style, width(100), 0, true, false)
	if r.Layout.Text != "abc   " || r.ResumeIndex != 6 || r.Width != 96 {
		t.Errorf("pre-wrap: got text %q, resume %d, width %v", r.Layout.Text, r.ResumeIndex, r.Width)
	}
}

func TestNextWordIsPulledOnTheLineWithoutItsSpace(t *testing.T) {
	ctx := newTestContext(t)
	// "aaa bbb" fits in 112 px without the space after it.
	r := SplitFirstLine(ctx, "aaa bbb ccc", testStyle(), width(112), 0, true, false)
	if r.Layout.Text != "aaa bbb" || r.ResumeIndex != 8 || r.Width != 112 {
		t.Errorf("got text %q, resume %d, width %v", r.Layout.Text, r.ResumeIndex, r.Width)
	}
}

func TestPreservedLineBreakEndsTheLine(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.WhiteSpace = "pre"
	r := SplitFirstLine(ctx, "ab\ncd", style, width(1), 0, true, false)
	if r.Length != 2 || r.ResumeIndex != 3 {
		t.Errorf("got length %d, resume %d; want 2 and 3", r.Length, r.ResumeIndex)
	}
}

func TestWhiteSpaceModes(t *testing.T) {
	ctx := newTestContext(t)
	// The text as it is after white space processing of each mode.
	collapsed := map[string]string{
		"normal":   "This + is text",
		"pre":      "This +    \n    is text",
		"nowrap":   "This + is text",
		"pre-wrap": "This +    \n    is text",
		"pre-line": "This +\nis text",
	}
	tests := []struct {
		width float64
		mode  string
		lines []string
	}{
		{1, "normal", []string{"This", "+", "is", "text"}},
		{1, "pre", []string{"This +    ", "    is text"}},
		{1, "nowrap", []string{"This + is text"}},
		{1, "pre-wrap", []string{"This ", "+    ", "    ", "is ", "text"}},
		{1, "pre-line", []string{"This", "+", "is", "text"}},
		{1000000, "normal", []string{"This + is text"}},
		{1000000, "pre", []string{"This +    ", "    is text"}},
		{1000000, "nowrap", []string{"This + is text"}},
		{1000000, "pre-wrap", []string{"This +    ", "    is text"}},
		{1000000, "pre-line", []string{"This +", "is text"}},
	}
	for _, tt := range tests {
		style := NewStyle(100, "testfont")
		style.WhiteSpace = tt.mode
		got := splitLines(t, ctx, collapsed[tt.mode], style, tt.width)
		if strings.Join(got, "|") != strings.Join(tt.lines, "|") {
			t.Errorf("%s at %v px: lines %q, want %q", tt.mode, tt.width, got, tt.lines)
		}
	}
}

func hyphenStyle(lang, hyphens string) *Style {
	s := testStyle()
	s.Lang = lang
	s.Hyphens = hyphens
	return s
}

func TestHyphenateCharacter(t *testing.T) {
	ctx := newTestContext(t)
	for _, char := range []string{"!", "à", "ù ù", "", "\u2014\u2014\u2014"} {
		style := hyphenStyle("fr", "auto")
		style.HyphenateCharacter = char
		lines := splitLines(t, ctx, "hyphénation", style, 80)
		if len(lines) < 2 {
			t.Errorf("%q: lines %q, want several", char, lines)
			continue
		}
		if char != "" && !strings.HasSuffix(lines[0], char) {
			t.Errorf("%q: first line %q does not end with it", char, lines[0])
		}
		full := strings.Join(lines, "")
		if char != "" {
			full = strings.ReplaceAll(full, char, "")
		}
		if full != "hyphénation" {
			t.Errorf("%q: lines %q", char, lines)
		}
	}
}

func TestManualHyphenationAtEverySoftHyphen(t *testing.T) {
	ctx := newTestContext(t)
	word := []rune("hyphénation")
	for i := 1; i < len(word); i++ {
		for _, char := range []string{"!", "ù ù"} {
			text := string(word[:i]) + "\u00ad" + string(word[i:])
			style := hyphenStyle("fr", "manual")
			style.HyphenateCharacter = char
			lines := splitLines(t, ctx, text, style, 80)
			if len(lines) != 2 {
				t.Errorf("%q: lines %q, want 2", text, lines)
				continue
			}
			if !strings.HasSuffix(lines[0], char) {
				t.Errorf("%q: first line %q does not end with %q", text, lines[0], char)
			}
			if full := strings.ReplaceAll(strings.Join(lines, ""), char, ""); full != text {
				t.Errorf("%q: lines %q", text, lines)
			}
		}
	}
}

func TestManualHyphenationAfterASpace(t *testing.T) {
	ctx := newTestContext(t)
	word := []rune("hy phénation")
	for i := 1; i < len(word); i++ {
		for _, char := range []string{"!", "ù ù"} {
			text := string(word[:i]) + "\u00ad" + string(word[i:])
			style := hyphenStyle("fr", "manual")
			style.HyphenateCharacter = char
			lines := splitLines(t, ctx, text, style, 80)
			if len(lines) != 2 && len(lines) != 3 {
				t.Errorf("%q: lines %q, want 2 or 3", text, lines)
				continue
			}
			if strings.HasSuffix(lines[0], char) {
				full := strings.ReplaceAll(strings.Join(lines, ""), char, "")
				if full != text {
					t.Errorf("%q: lines %q", text, lines)
				}
				continue
			}
			if !strings.HasSuffix(strings.TrimRight(lines[0], "\u00ad"), "y") {
				t.Errorf("%q: first line %q", text, lines[0])
			}
			if len(lines) == 3 && !strings.HasSuffix(strings.TrimRight(lines[1], "\u00ad"), char) {
				t.Errorf("%q: second line %q", text, lines[1])
			}
		}
	}
}

func TestSoftHyphensWinOverAutomaticHyphenation(t *testing.T) {
	ctx := newTestContext(t)
	style := hyphenStyle("en", "auto")
	lines := splitLines(t, ctx, "in\u00adlighten\u00adlighten\u00adin", style, 1.6)
	want := []string{"in\u00ad\u2010", "lighten\u00ad\u2010", "lighten\u00ad\u2010", "in"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines %q, want %q", lines, want)
	}
}

func TestHyphenatedLineKeepsItsSoftHyphen(t *testing.T) {
	ctx := newTestContext(t)
	lines := splitLines(t, ctx, "super\u00adaa bb", hyphenStyle("", "manual"), 96)
	if len(lines) == 0 || lines[0] != "super\u00ad\u2010" {
		t.Errorf("lines %q", lines)
	}
}

func TestHyphenateLimitZone(t *testing.T) {
	ctx := newTestContext(t)
	tests := []struct {
		zone       LengthPercentage
		hyphenated bool
	}{
		{LengthPercentage{}, true},
		{LengthPercentage{Value: 144}, false},
		{LengthPercentage{Value: 5, Percent: true}, true},
		{LengthPercentage{Value: 95, Percent: true}, false},
		{LengthPercentage{Calc: func(base float64) float64 { return base * 5 / 100 }}, true},
		{LengthPercentage{Calc: func(base float64) float64 { return base - 48 }}, false},
	}
	for _, tt := range tests {
		style := hyphenStyle("fr", "auto")
		style.HyphenateLimitZone = tt.zone
		lines := splitLines(t, ctx, "mmmmm hyphénation", style, 192)
		if tt.hyphenated {
			if len(lines) != 2 || !strings.HasSuffix(lines[0], "\u2010") {
				t.Errorf("zone %+v: lines %q, want a hyphenated first line", tt.zone, lines)
			}
			continue
		}
		if len(lines) < 2 || !strings.HasSuffix(lines[0], "mm") || strings.Join(lines, "") != "mmmmmhyphénation" {
			t.Errorf("zone %+v: lines %q, want no hyphenation", tt.zone, lines)
		}
	}
}

func TestHyphenationAfterNoBreakSpace(t *testing.T) {
	ctx := newTestContext(t)
	style := NewStyle(2, "testfont")
	style.Lang, style.Hyphens = "en", "auto"
	lines := splitLines(t, ctx, "this\u00a0hyphenation", style, 20)
	want := []string{"this\u00a0hy\u2010", "phenation"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines %q, want %q", lines, want)
	}
}

func TestHyphenateLimitChars(t *testing.T) {
	ctx := newTestContext(t)
	tests := []struct {
		limits HyphenateLimitChars
		lines  int
	}{
		{HyphenateLimitChars{5, 2, 2}, 2},
		{HyphenateLimitChars{5, 2, 0}, 2},
		{HyphenateLimitChars{0, 0, 0}, 2},
		{HyphenateLimitChars{4, 4, 2}, 1},
		{HyphenateLimitChars{6, 2, 4}, 2},
		{HyphenateLimitChars{5, 1, 2}, 2},
		{HyphenateLimitChars{7, 2, 2}, 1},
		{HyphenateLimitChars{6, 2, 2}, 2},
		{HyphenateLimitChars{3, 2, 2}, 2},
		{HyphenateLimitChars{2, 4, 6}, 1},
		{HyphenateLimitChars{5, 4, 4}, 1},
	}
	for _, tt := range tests {
		style := hyphenStyle("en", "auto")
		style.HyphenateLimitChars = tt.limits
		if lines := splitLines(t, ctx, "hyphen", style, 16); len(lines) != tt.lines {
			t.Errorf("%+v: lines %q, want %d", tt.limits, lines, tt.lines)
		}
	}
}

func TestHyphenateLimitCharsAroundPunctuation(t *testing.T) {
	ctx := newTestContext(t)
	for _, limits := range []HyphenateLimitChars{{3, 3, 3}, {3, 6, 2}, {8, 2, 2}} {
		style := hyphenStyle("en", "auto")
		style.HyphenateLimitChars = limits
		if lines := splitLines(t, ctx, "..lighten..", style, 16); len(lines) != 1 {
			t.Errorf("%+v: lines %q, want 1", limits, lines)
		}
	}
}

func TestOverflowWrap(t *testing.T) {
	ctx := newTestContext(t)
	tests := []struct {
		wrap     string
		text     string
		minLines int
		maxLines int
		full     string
	}{
		{"anywhere", "aaaaaaaa", 2, 100, "aaaaaaaa"},
		{"break-word", "aaaaaaaa", 2, 100, "aaaaaaaa"},
		{"normal", "aaaaaaaa", 1, 1, "aaaaaaaa"},
		{"break-word", "hyphenations", 4, 100, "hy\u2010phen\u2010ations"},
		{"break-word", "A splitted word. An hyphenated word.", 9, 100, "Asplittedword.Anhy\u2010phen\u2010atedword."},
	}
	for _, tt := range tests {
		style := hyphenStyle("en", "auto")
		style.OverflowWrap = tt.wrap
		lines := splitLines(t, ctx, tt.text, style, 80)
		if len(lines) < tt.minLines || len(lines) > tt.maxLines || strings.Join(lines, "") != tt.full {
			t.Errorf("%s %q: lines %q", tt.wrap, tt.text, lines)
		}
	}
}

func TestBreakWordIsIgnoredForMinContent(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.OverflowWrap = "break-word"
	r := SplitFirstLine(ctx, "aaaaaa", style, width(0), 0, true, true)
	if r.ResumeIndex != -1 || r.Width != 96 {
		t.Errorf("min-content: resume %d, width %v; want the whole word", r.ResumeIndex, r.Width)
	}
	style.OverflowWrap = "anywhere"
	r = SplitFirstLine(ctx, "aaaaaa", style, width(0), 0, true, true)
	if r.ResumeIndex != 1 || r.Width != 16 {
		t.Errorf("anywhere: resume %d, width %v; want one character", r.ResumeIndex, r.Width)
	}
}

func TestWordsAreOnlyBrokenAtTheStartOfALine(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	style.OverflowWrap = "anywhere"
	r := SplitFirstLine(ctx, "aaaaaa", style, width(40), 0, false, false)
	if r.ResumeIndex != -1 {
		t.Errorf("resume %d, want the word kept whole", r.ResumeIndex)
	}
	style.OverflowWrap = "normal"
	style.WordBreak = "break-all"
	// Breaking inside the word shows a hyphen, so only one character and
	// the hyphen fit.
	r = SplitFirstLine(ctx, "aaaaaa", style, width(40), 0, false, false)
	if r.ResumeIndex != 1 {
		t.Errorf("break-all: resume %d, want 1", r.ResumeIndex)
	}
}

func TestJustificationSpacingIsAddedWhenReactivated(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	r := SplitFirstLine(ctx, "a b c", style, nil, 3, true, false)
	if r.Width != 80 {
		t.Fatalf("width %v, want 80 before justification", r.Width)
	}
	r.Layout.Reactivate(style)
	if w, _ := LineSize(r.Layout.FirstLine(), style); w != 86 {
		t.Errorf("justified width %v, want 86", w)
	}
}

func TestNewLayoutKeepsTextAndJustificationSpacing(t *testing.T) {
	ctx := newTestContext(t)
	style := testStyle()
	l := NewLayout(ctx, "a b c", style, nil, 3)
	if l.Text != "a b c" || l.JustificationSpacing != 3 {
		t.Fatalf("layout = %q with spacing %v", l.Text, l.JustificationSpacing)
	}
	if w, _ := LineSize(l.FirstLine(), style); w != 80 {
		t.Errorf("width %v, want 80 before justification", w)
	}
	l.Reactivate(style)
	if w, _ := LineSize(l.FirstLine(), style); w != 86 {
		t.Errorf("justified width %v, want 86", w)
	}
}

// splitLines breaks text into lines of at most maxWidth px, starting a
// new line where the previous one ends, and returns the text of each line.
func splitLines(t testing.TB, ctx *Context, text string, style *Style, maxWidth float64) []string {
	t.Helper()
	var lines []string
	for skip := 0; skip < len(text) && len(lines) < 100; {
		r := SplitFirstLine(ctx, text[skip:], style, &maxWidth, 0, true, false)
		if r.ResumeIndex == 0 {
			t.Fatalf("no progress at %d in %q", skip, text)
		}
		if r.Length > 0 {
			lines = append(lines, r.Layout.Text)
		}
		if r.ResumeIndex < 0 {
			break
		}
		skip += r.ResumeIndex
	}
	return lines
}
