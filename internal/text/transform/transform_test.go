package transform

import "testing"

func TestTransformKeywords(t *testing.T) {
	tests := []struct {
		function, want string
		ok             bool
	}{
		{"capitalize", "Hé LO1", true},
		{"uppercase", "HÉ LO1", true},
		{"lowercase", "hé lo1", true},
		{"full-width", "ｈé　ｌＯ１", true},
		{"none", "hé lO1", false},
	}
	for _, tt := range tests {
		got, ok := Transform(tt.function, "hé lO1", "")
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: got %q, %v; want %q, %v", tt.function, got, ok, tt.want, tt.ok)
		}
	}
}

func TestUppercaseByLanguage(t *testing.T) {
	tests := []struct{ lang, in, want string }{
		{"az", "ı i", "I İ"},
		{"tr", "ı i", "I İ"},
		{"el", "καλημέρα αύριο θεϊκό ευφυΐα Νεράιδα", "ΚΑΛΗΜΕΡΑ ΑΥΡΙΟ ΘΕΪΚΟ ΕΥΦΥΪΑ ΝΕΡΑΪΔΑ"},
		{"", "straße", "STRASSE"},
	}
	for _, tt := range tests {
		if got := Uppercase(tt.in, tt.lang); got != tt.want {
			t.Errorf("Uppercase(%q, %q) = %q, want %q", tt.in, tt.lang, got, tt.want)
		}
	}
}

func TestLowercaseByLanguage(t *testing.T) {
	tests := []struct{ lang, in, want string }{
		{"az", "İ I", "i ı"},
		{"tr", "İ I", "i ı"},
		{"lt", "Ì Í Ĩ", "i̇̀ i̇́ i̇̃"},
		{"", "ΟΔΟΣ", "οδος"},
	}
	for _, tt := range tests {
		if got := Lowercase(tt.in, tt.lang); got != tt.want {
			t.Errorf("Lowercase(%q, %q) = %q, want %q", tt.in, tt.lang, got, tt.want)
		}
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct{ in, want, lang string }{
		{"abc def ghi", "Abc Def Ghi", ""},
		{"AbC def ghi", "AbC Def Ghi", ""},
		{"I’m SO cool", "I’m SO Cool", ""},
		{"Wow.wow!wow", "Wow.wow!wow", ""},
		{"!now not tomorrow", "!Now Not Tomorrow", ""},
		{"SUPER cool", "SUPER Cool", ""},
		{"i 😻 non‑breaking characters", "I 😻 Non‑breaking Characters", ""},
		{"3lite 3lite", "3lite 3lite", ""},
		{"one/two/three", "One/two/three", ""},
		{"supernatural,super", "Supernatural,super", ""},
		{"éternel αιώνια", "Éternel Αιώνια", ""},
		{"great ijland", "Great Ijland", "fr"},
		{"great ijland", "Great IJland", "nl"},
		{"great ijland", "Great İjland", "az"},
	}
	for _, tt := range tests {
		if got := Capitalize(tt.in, tt.lang); got != tt.want {
			t.Errorf("Capitalize(%q, %q) = %q, want %q", tt.in, tt.lang, got, tt.want)
		}
	}
}

func TestFirstLetter(t *testing.T) {
	tests := []struct{ in, want string }{
		{"abc", "a"},
		{`"abc`, `"a`},
		{"(«x", "(«x"},
		{"", ""},
		{"...", "..."},
		{"1st", "1"},
	}
	for _, tt := range tests {
		if got := FirstLetter(tt.in); got != tt.want {
			t.Errorf("FirstLetter(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
