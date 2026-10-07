package hyphen

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// ignoredPrefixes are the line prefixes of comments and of options that this
// package does not interpret.
var ignoredPrefixes = []string{
	"%", "#", "LEFTHYPHENMIN", "RIGHTHYPHENMIN",
	"COMPOUNDLEFTHYPHENMIN", "COMPOUNDRIGHTHYPHENMIN",
}

// Nonstandard describes a hyphenation point that changes the spelling of the
// word, as in Hungarian "kulissza" becoming "kulisz-sza".
type Nonstandard struct {
	// Change is a string like "ff=f"; the part before "=" ends the first line
	// and the part after it starts the next one.
	Change string
	// Index is where to substitute Change, counting from the position.
	Index int
	// Cut is the number of characters removed by the substitution.
	Cut int
}

// Position is a place where a word may be hyphenated.
type Position struct {
	// Index is the number of characters before the hyphen.
	Index int
	// Alt is non-nil for non-standard hyphenation.
	Alt *Nonstandard
}

// level is a pattern digit, possibly carrying non-standard data.
type level struct {
	value int
	alt   *Nonstandard
}

type pattern struct {
	offset int
	values []level
}

// dictionary holds the parsed patterns of one hyph_*.dic file.
type dictionary struct {
	patterns map[string]pattern
	maxLen   int

	mu    sync.RWMutex
	cache map[string][]Position
}

// alternativeParser tracks the position inside a pattern that carries a
// "/change,index,cut" suffix. Every digit slot consumes one index step, so an
// odd value records where in the pattern it sits.
type alternativeParser struct {
	change string
	index  int
	cut    int
}

func newAlternativeParser(pattern, alternative string) (*alternativeParser, error) {
	parts := strings.Split(alternative, ",")
	if len(parts) < 3 {
		return nil, fmt.Errorf("hyphen: malformed alternative %q", alternative)
	}
	index, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("hyphen: malformed alternative %q: %w", alternative, err)
	}
	cut, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("hyphen: malformed alternative %q: %w", alternative, err)
	}
	if strings.HasPrefix(pattern, ".") {
		index++
	}
	return &alternativeParser{change: parts[0], index: index, cut: cut}, nil
}

func (p *alternativeParser) level(value int) level {
	p.index--
	if value&1 == 1 {
		return level{value: value, alt: &Nonstandard{Change: p.change, Index: p.index, Cut: p.cut}}
	}
	return level{value: value}
}

// splitPattern splits a pattern into (digit, character) slots: an optional
// digit followed by an optional non-digit, plus a final empty slot. The final
// slot takes part in the alternative parser's index arithmetic.
func splitPattern(s string) (digits []int, chars []string) {
	for s != "" {
		d := 0
		if r, size := utf8.DecodeRuneInString(s); unicode.IsDigit(r) {
			d = digitValue(r)
			s = s[size:]
		}
		c := ""
		if r, size := utf8.DecodeRuneInString(s); size > 0 && !unicode.IsDigit(r) {
			c = s[:size]
			s = s[size:]
		}
		digits = append(digits, d)
		chars = append(chars, c)
	}
	return append(digits, 0), append(chars, "")
}

// digitValue returns the decimal value of a Unicode decimal digit.
func digitValue(r rune) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	// Decimal digit blocks are contiguous runs of ten code points.
	v := 0
	for c := r - 1; unicode.IsDigit(c) && v < 9; c-- {
		v++
	}
	return v
}

func parseHex(s string) string {
	if !strings.Contains(s, "^^") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if i+4 <= len(s) && s[i] == '^' && s[i+1] == '^' && isLowerHex(s[i+2]) && isLowerHex(s[i+3]) {
			v, _ := strconv.ParseUint(s[i+2:i+4], 16, 8)
			b.WriteRune(rune(v))
			i += 4
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func isLowerHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f'
}

// parseDictionary reads the contents of a hyph_*.dic file.
func parseDictionary(data []byte) (*dictionary, error) {
	first, rest, _ := strings.Cut(string(data), "\n")
	enc, err := lookupEncoding(strings.TrimSpace(first))
	if err != nil {
		return nil, err
	}
	decoded, err := enc.NewDecoder().String(rest)
	if err != nil {
		return nil, fmt.Errorf("hyphen: decoding dictionary: %w", err)
	}
	decoded = strings.ReplaceAll(decoded, "\r\n", "\n")
	decoded = strings.ReplaceAll(decoded, "\r", "\n")

	d := &dictionary{patterns: map[string]pattern{}, cache: map[string][]Position{}}
	for line := range strings.SplitSeq(decoded, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || hasAnyPrefix(line, ignoredPrefixes) {
			continue
		}
		if err := d.addPattern(parseHex(line)); err != nil {
			return nil, err
		}
	}
	if len(d.patterns) == 0 {
		return nil, fmt.Errorf("hyphen: dictionary has no patterns")
	}
	for key := range d.patterns {
		d.maxLen = max(d.maxLen, utf8.RuneCountInString(key))
	}
	return d, nil
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func (d *dictionary) addPattern(pat string) error {
	var alt *alternativeParser
	if strings.Contains(pat, "/") && strings.Contains(pat, "=") {
		var alternative string
		pat, alternative, _ = strings.Cut(pat, "/")
		var err error
		if alt, err = newAlternativeParser(pat, alternative); err != nil {
			return err
		}
	}

	digits, chars := splitPattern(pat)
	values := make([]level, len(digits))
	nonZero := false
	for i, v := range digits {
		if alt != nil {
			values[i] = alt.level(v)
		} else {
			values[i] = level{value: v}
		}
		nonZero = nonZero || v != 0
	}
	if !nonZero {
		return nil
	}

	start, end := 0, len(values)
	for values[start].value == 0 {
		start++
	}
	for values[end-1].value == 0 {
		end--
	}
	d.patterns[strings.Join(chars, "")] = pattern{offset: start, values: values[start:end]}
	return nil
}

// positions returns the hyphenation points of word, unfiltered by minimum
// syllable lengths.
func (d *dictionary) positions(word string) []Position {
	word = lower(word)
	d.mu.RLock()
	points, ok := d.cache[word]
	d.mu.RUnlock()
	if ok {
		return points
	}

	pointed := []rune("." + word + ".")
	refs := make([]level, len(pointed)+1)
	for i := 0; i < len(pointed)-1; i++ {
		stop := min(i+d.maxLen, len(pointed)) + 1
		for j := i + 1; j < stop; j++ {
			pat, ok := d.patterns[string(pointed[i:j])]
			if !ok {
				continue
			}
			from := i + pat.offset
			for k, v := range pat.values {
				if from+k >= len(refs) {
					break
				}
				// On ties the pattern's own value wins, which is what lets
				// non-standard data replace data from a shorter pattern.
				if v.value >= refs[from+k].value {
					refs[from+k] = v
				}
			}
		}
	}

	points = []Position{}
	for i, ref := range refs {
		if ref.value%2 == 1 {
			points = append(points, Position{Index: i - 1, Alt: ref.alt})
		}
	}
	d.mu.Lock()
	d.cache[word] = points
	d.mu.Unlock()
	return points
}
