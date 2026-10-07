package linebreak

import "testing"

func TestBreakOpportunities(t *testing.T) {
	tests := []struct {
		text     string
		canBreak bool
		// next is the character index of the first break, -1 for none.
		next int
	}{
		{"ab", false, -1},
		{"a b", true, 2},
		{"a", false, -1},
		{"", false, -1},
		{" a", true, 1},
		{"a-b", true, 2},
		{"can't stop", true, 6},
		{"3.14 is pi", true, 5},
		{"foo bar baz", true, 4},
		{"x\u202ay z", true, 2},
		{"日本語のテキスト", true, 1},
		{"word, another", true, 6},
		{"  ", false, -1},
		{"a\u00adb", true, 2},
		{"e.g. this", true, 5},
		{"non\u00a0breaking", false, -1},
	}
	for _, tt := range tests {
		if got := CanBreakText(tt.text, "en"); got != tt.canBreak {
			t.Errorf("CanBreakText(%q) = %v, want %v", tt.text, got, tt.canBreak)
		}
		offset, ok := NextBreakPoint(tt.text, "en")
		want := -1
		if tt.next >= 0 {
			want = byteOffset(tt.text, tt.next)
		}
		if !ok {
			offset = -1
		}
		if offset != want {
			t.Errorf("NextBreakPoint(%q) = %d, want %d", tt.text, offset, want)
		}
	}
}

func TestNextWordBoundaries(t *testing.T) {
	tests := []struct {
		text       string
		start, end int
		ok         bool
	}{
		{"ab", 0, 2, true},
		{"a b", 0, 1, true},
		{"a", 0, 0, false},
		{" a", 1, 2, true},
		{"a-b", 0, 1, true},
		{"can't stop", 0, 3, true},
		{"3.14 is pi", 0, 1, true},
		{"word, another", 0, 4, true},
		{"  ", 0, 0, false},
		{"a\u00adb", 0, 4, true},
		{"日本語のテキスト", 6, 9, true},
	}
	for _, tt := range tests {
		start, end, ok := NextWordBoundaries(tt.text, "en")
		if start != tt.start || end != tt.end || ok != tt.ok {
			t.Errorf("NextWordBoundaries(%q) = %d, %d, %v, want %d, %d, %v",
				tt.text, start, end, ok, tt.start, tt.end, tt.ok)
		}
	}
}

func TestLastWordEnd(t *testing.T) {
	tests := []struct {
		text string
		end  int
		ok   bool
	}{
		{"a b", 1, true},
		{"foo bar baz", 7, true},
		{"3.14 is pi", 7, true},
		{"ab", 0, false},
		{" a", 0, false},
		{"e.g. this", 3, true},
	}
	for _, tt := range tests {
		end, ok := LastWordEnd(tt.text, "en")
		if end != tt.end || ok != tt.ok {
			t.Errorf("LastWordEnd(%q) = %d, %v, want %d, %v", tt.text, end, ok, tt.end, tt.ok)
		}
	}
}

func TestCatalanMiddleDotJoinsWords(t *testing.T) {
	text := "col·lecció"
	if _, end, _ := NextWordBoundaries(text, "en"); end != 3 {
		t.Errorf("without Catalan, the word ends at %d, want 3", end)
	}
	if _, end, _ := NextWordBoundaries(text, "ca-ES"); end != len(text) {
		t.Errorf("in Catalan, the word ends at %d, want %d", end, len(text))
	}
}

func TestHyphensShownAtBreaks(t *testing.T) {
	tests := []struct {
		text string
		// positions where breaking shows a hyphen
		want []int
	}{
		{"word", []int{1, 2, 3}},
		{"a b", nil},
		{"a-b", nil},
		{"a\u00adb", []int{1, 2}},
		{"12.5", nil},
		{"日本", nil},
	}
	for _, tt := range tests {
		attrs := computeLogAttrs([]rune(tt.text))
		var got []int
		for i, a := range attrs {
			if a.insertsHyphen {
				got = append(got, i)
			}
		}
		if len(got) != len(tt.want) {
			t.Errorf("%q: hyphens at %v, want %v", tt.text, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%q: hyphens at %v, want %v", tt.text, got, tt.want)
				break
			}
		}
	}
}

func TestDisablingHyphensKeepsHyphenationPoints(t *testing.T) {
	attrs := computeLogAttrs([]rune("a\u2027b"))
	disableHyphenInsertion(attrs)
	if !attrs[2].insertsHyphen || !attrs[2].removesPreceding {
		t.Errorf("a hyphenation point must still show a hyphen: %+v", attrs[2])
	}
	if attrs[1].insertsHyphen {
		t.Errorf("breaks inside words must not show hyphens: %+v", attrs[1])
	}
}

func TestWordBoundariesKeepContractionsAndNumbers(t *testing.T) {
	tests := []struct {
		text       string
		boundaries []int
	}{
		{"can't", []int{0, 5}},
		{"3.14", []int{0, 4}},
		{"a.b c", []int{0, 3, 4, 5}},
		{"a  b", []int{0, 1, 3, 4}},
	}
	for _, tt := range tests {
		attrs := computeLogAttrs([]rune(tt.text))
		var got []int
		for i, a := range attrs {
			if a.wordBoundary {
				got = append(got, i)
			}
		}
		if len(got) != len(tt.boundaries) {
			t.Errorf("%q: boundaries %v, want %v", tt.text, got, tt.boundaries)
			continue
		}
		for i := range got {
			if got[i] != tt.boundaries[i] {
				t.Errorf("%q: boundaries %v, want %v", tt.text, got, tt.boundaries)
				break
			}
		}
	}
}
