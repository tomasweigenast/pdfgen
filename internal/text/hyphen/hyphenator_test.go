package hyphen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func mustNew(t *testing.T, lang string, left, right int) *Hyphenator {
	t.Helper()
	h, err := New(lang, left, right)
	if err != nil {
		t.Fatalf("New(%q): %v", lang, err)
	}
	return h
}

func TestInserted(t *testing.T) {
	h := mustNew(t, "nl_NL", 2, 2)
	if got := h.Inserted("lettergrepen", "-"); got != "let-ter-gre-pen" {
		t.Errorf("got %q", got)
	}
}

func TestWrap(t *testing.T) {
	h := mustNew(t, "nl_NL", 2, 2)
	head, tail, ok := h.Wrap("autobandventieldopje", 11, "-")
	if !ok || head != "autoband-" || tail != "ventieldopje" {
		t.Errorf("got %q %q %v", head, tail, ok)
	}
	if _, _, ok := h.Wrap("autobandventieldopje", 2, "-"); ok {
		t.Error("expected no fitting split")
	}
}

func TestIterateLongestFirst(t *testing.T) {
	want := []Split{{"Amster", "dam"}, {"Am", "sterdam"}}
	for _, lang := range []string{"nl_NL", "nl_NL-variant"} {
		if got := mustNew(t, lang, 2, 2).Iterate("Amsterdam"); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v", lang, got)
		}
	}
}

func TestMissingDictionary(t *testing.T) {
	if _, err := New("mi_SS", 2, 2); err == nil {
		t.Fatal("expected an error")
	}
}

func TestAliasedLanguage(t *testing.T) {
	const word = "autobandventieldopje"
	if got := mustNew(t, "fr", 2, 2).Inserted(word, "-"); got == "au-to-band-ven-tiel-dop-je" {
		t.Fatalf("fr unexpectedly hyphenates like nl_NL: %q", got)
	}
	registry.Lock()
	orig := registry.languages["fr"]
	registry.Unlock()
	t.Cleanup(func() {
		registry.Lock()
		registry.languages["fr"] = orig
		registry.Unlock()
	})
	if err := Alias("fr", "nl_NL"); err != nil {
		t.Fatal(err)
	}
	if got := mustNew(t, "fr", 2, 2).Inserted(word, "-"); got != "au-to-band-ven-tiel-dop-je" {
		t.Errorf("got %q", got)
	}
}

func TestDictionaryFromFile(t *testing.T) {
	h, err := NewFromFile(filepath.Join("dictionaries", "hyph_fr.dic"), 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := h.Inserted("bonjour", "-"); got != "bon-jour" {
		t.Errorf("got %q", got)
	}
}

func TestDictionaryWithMicrosoftCP1251Encoding(t *testing.T) {
	data := []byte("microsoft-cp1251\n\xf7" + "1\xed\n")
	h, err := NewFromData(data, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := h.Inserted("речник", "-"); got != "реч-ник" {
		t.Errorf("got %q", got)
	}
}

func TestMissingFile(t *testing.T) {
	if _, err := NewFromFile(filepath.Join(t.TempDir(), "none.dic"), 2, 2); !os.IsNotExist(unwrap(err)) {
		t.Errorf("got %v", err)
	}
}

func unwrap(err error) error {
	for {
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return err
		}
		err = u.Unwrap()
	}
}

func TestLeftRightLimits(t *testing.T) {
	tests := []struct {
		left, right int
		want        string
	}{
		{2, 2, "let-ter-gre-pen"},
		{4, 2, "letter-gre-pen"},
		{2, 4, "let-ter-grepen"},
		{4, 4, "letter-grepen"},
	}
	for _, tt := range tests {
		if got := mustNew(t, "nl_NL", tt.left, tt.right).Inserted("lettergrepen", "-"); got != tt.want {
			t.Errorf("left=%d right=%d: got %q, want %q", tt.left, tt.right, got, tt.want)
		}
	}
}

func TestNonstandardHyphenation(t *testing.T) {
	h := mustNew(t, "hu", 1, 1)
	want := []Split{{"kulisz", "sza"}, {"ku", "lissza"}}
	if got := h.Iterate("kulissza"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if got := h.Inserted("kulissza", "-"); got != "ku-lisz-sza" {
		t.Errorf("got %q", got)
	}
}

func TestUppercaseWords(t *testing.T) {
	if got := mustNew(t, "nl_NL", 2, 2).Inserted("LETTERGREPEN", "-"); got != "LET-TER-GRE-PEN" {
		t.Errorf("got %q", got)
	}
	h := mustNew(t, "hu", 1, 1)
	want := []Split{{"KULISZ", "SZA"}, {"KU", "LISSZA"}}
	if got := h.Iterate("KULISSZA"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if got := h.Inserted("KULISSZA", "-"); got != "KU-LISZ-SZA" {
		t.Errorf("got %q", got)
	}
}

func TestAllBundledDictionariesParse(t *testing.T) {
	for _, lang := range Languages() {
		if _, err := New(lang, 2, 2); err != nil {
			t.Errorf("%s: %v", lang, err)
		}
	}
}

func TestLanguageFallback(t *testing.T) {
	tests := map[string]string{
		"en":               "en",
		"en_GB":            "en_GB",
		"en_FR":            "en",
		"sr-Latn":          "sr_Latn",
		"SR-LATN":          "sr_Latn",
		"sr-Cyrl":          "sr",
		"fr-Latn-FR":       "fr",
		"en-GB_variant1-x": "en_GB",
		"xx":               "",
	}
	for in, want := range tests {
		if got := LanguageFallback(in); got != want {
			t.Errorf("LanguageFallback(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFinalSigmaLowercasing(t *testing.T) {
	if got := lower("ΟΔΟΣ"); got != "οδος" {
		t.Errorf("got %q", got)
	}
	if got := lower("ΣΟΦΟΣ"); got != "σοφος" {
		t.Errorf("got %q", got)
	}
}

func TestAmericanEnglish(t *testing.T) {
	h := mustNew(t, "en_US", 2, 2)
	if got := h.Inserted("hyphenation", "-"); got != "hy-phen-ation" {
		t.Errorf("Inserted(hyphenation) = %q", got)
	}
	if got := LanguageFallback("en-US"); got != "en_US" {
		t.Errorf("LanguageFallback(en-US) = %q, want en_US", got)
	}
}
