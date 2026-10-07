package hyphen

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

//go:embed dictionaries/hyph_*.dic
var dictionaryFS embed.FS

// ErrNoDictionary is returned when no dictionary matches a language.
var ErrNoDictionary = errors.New("hyphen: no dictionary for language")

// Default minimum lengths of the first and last syllables.
const (
	DefaultLeft  = 2
	DefaultRight = 2
)

var registry = struct {
	sync.Mutex
	// languages maps a language code ("en_US", "en") to a dictionary file.
	languages map[string]string
	// lowercase maps lowercased codes back to the spelling in languages.
	lowercase map[string]string
	// loaded caches parsed dictionaries by file name.
	loaded map[string]*dictionary
}{}

func init() {
	entries, err := dictionaryFS.ReadDir("dictionaries")
	if err != nil {
		panic(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	registry.languages = map[string]string{}
	registry.loaded = map[string]*dictionary{}
	for _, file := range names {
		name := strings.TrimSuffix(strings.TrimPrefix(file, "hyph_"), ".dic")
		registry.languages[name] = file
		short, _, _ := strings.Cut(name, "_")
		if _, ok := registry.languages[short]; !ok {
			registry.languages[short] = file
		}
	}
	registry.lowercase = map[string]string{}
	for name := range registry.languages {
		registry.lowercase[strings.ToLower(name)] = name
	}
}

// Languages returns the language codes with a bundled dictionary, sorted.
func Languages() []string {
	registry.Lock()
	defer registry.Unlock()
	codes := make([]string, 0, len(registry.languages))
	for code := range registry.languages {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// Alias makes code resolve to the dictionary of target, which must be a code
// returned by Languages. It affects subsequent calls to New.
func Alias(code, target string) error {
	registry.Lock()
	defer registry.Unlock()
	file, ok := registry.languages[target]
	if !ok {
		return fmt.Errorf("%w: %q", ErrNoDictionary, target)
	}
	registry.languages[code] = file
	return nil
}

// LanguageFallback returns the code of the bundled dictionary that best
// matches a BCP 47 or POSIX language tag, by truncating the tag one subtag at
// a time (Unicode TR35, "Locale Inheritance"). It returns "" when none match.
func LanguageFallback(language string) string {
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(language, "-", "_")), "_")
	registry.Lock()
	defer registry.Unlock()
	for len(parts) > 0 {
		if name, ok := registry.lowercase[strings.Join(parts, "_")]; ok {
			return name
		}
		parts = parts[:len(parts)-1]
	}
	return ""
}

// Hyphenator splits words at the points allowed by a hyphenation dictionary.
// It is safe for concurrent use.
type Hyphenator struct {
	left, right int
	dict        *dictionary
}

// New returns a Hyphenator for a language tag using the bundled dictionaries.
// left and right are the minimum numbers of characters before the first and
// after the last hyphen; see DefaultLeft and DefaultRight.
func New(lang string, left, right int) (*Hyphenator, error) {
	code := LanguageFallback(lang)
	if code == "" {
		return nil, fmt.Errorf("%w: %q", ErrNoDictionary, lang)
	}
	registry.Lock()
	file := registry.languages[code]
	d := registry.loaded[file]
	registry.Unlock()
	if d == nil {
		data, err := dictionaryFS.ReadFile("dictionaries/" + file)
		if err != nil {
			return nil, fmt.Errorf("hyphen: reading %s: %w", file, err)
		}
		if d, err = parseDictionary(data); err != nil {
			return nil, fmt.Errorf("hyphen: parsing %s: %w", file, err)
		}
		registry.Lock()
		if prev := registry.loaded[file]; prev != nil {
			d = prev
		} else {
			registry.loaded[file] = d
		}
		registry.Unlock()
	}
	return &Hyphenator{left: left, right: right, dict: d}, nil
}

// NewFromData returns a Hyphenator for the contents of a hyph_*.dic file.
func NewFromData(data []byte, left, right int) (*Hyphenator, error) {
	d, err := parseDictionary(data)
	if err != nil {
		return nil, err
	}
	return &Hyphenator{left: left, right: right, dict: d}, nil
}

// NewFromFile returns a Hyphenator for the hyph_*.dic file at path.
func NewFromFile(path string, left, right int) (*Hyphenator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("hyphen: %w", err)
	}
	h, err := NewFromData(data, left, right)
	if err != nil {
		return nil, fmt.Errorf("hyphen: parsing %s: %w", path, err)
	}
	return h, nil
}

// Positions returns the points where word can be hyphenated, in increasing
// order. Points closer to the edges than the left and right limits are
// dropped.
func (h *Hyphenator) Positions(word string) []Position {
	right := utf8.RuneCountInString(word) - h.right
	var out []Position
	for _, p := range h.dict.positions(word) {
		if h.left <= p.Index && p.Index <= right {
			out = append(out, p)
		}
	}
	return out
}

// Split is a word cut at a hyphenation point. Head includes any characters
// that a non-standard hyphenation adds before the hyphen, but not the hyphen
// itself.
type Split struct {
	Head, Tail string
}

// Iterate returns all the ways to split word, the longest head first.
func (h *Hyphenator) Iterate(word string) []Split {
	positions := h.Positions(word)
	runes := []rune(word)
	upper := isUpper(word)
	splits := make([]Split, 0, len(positions))
	for i := len(positions) - 1; i >= 0; i-- {
		p := positions[i]
		if p.Alt == nil {
			splits = append(splits, Split{string(runes[:p.Index]), string(runes[p.Index:])})
			continue
		}
		change := p.Alt.Change
		if upper {
			change = strings.ToUpper(change)
		}
		c1, c2, _ := strings.Cut(change, "=")
		index := clamp(p.Alt.Index+p.Index, len(runes))
		cut := clamp(index+p.Alt.Cut, len(runes))
		splits = append(splits, Split{string(runes[:index]) + c1, c2 + string(runes[cut:])})
	}
	return splits
}

// Wrap returns the longest head of word that fits in width characters once
// hyphen is appended to it, and the remaining tail. ok is false when word has
// no hyphenation point that fits.
func (h *Hyphenator) Wrap(word string, width int, hyphen string) (head, tail string, ok bool) {
	width -= utf8.RuneCountInString(hyphen)
	for _, s := range h.Iterate(word) {
		if utf8.RuneCountInString(s.Head) <= width {
			return s.Head + hyphen, s.Tail, true
		}
	}
	return "", "", false
}

// Inserted returns word with hyphen inserted at every hyphenation point, as in
// "let-ter-gre-pen".
func (h *Hyphenator) Inserted(word, hyphen string) string {
	letters := []rune(word)
	upper := isUpper(word)
	positions := h.Positions(word)
	for i := len(positions) - 1; i >= 0; i-- {
		p := positions[i]
		if p.Alt == nil {
			at := clamp(p.Index, len(letters))
			letters = append(letters[:at], append([]rune(hyphen), letters[at:]...)...)
			continue
		}
		change := p.Alt.Change
		if upper {
			change = strings.ToUpper(change)
		}
		start := clamp(p.Alt.Index+p.Index, len(letters))
		end := clamp(start+p.Alt.Cut, len(letters))
		repl := []rune(strings.ReplaceAll(change, "=", hyphen))
		letters = append(letters[:start], append(repl, letters[end:]...)...)
	}
	return string(letters)
}

func clamp(i, n int) int {
	return max(0, min(i, n))
}
