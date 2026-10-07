package html

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
)

var invalidIntegers = []string{"", "   ", "abc", "+", "-", "+ 1", "٠١"}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"42", 42}, {"0", 0}, {"100", 100}, {"+123", 123}, {"-42", -42},
		{"+0", 0}, {"-0", 0}, {"  42", 42}, {"\t7", 7}, {" \n -5", -5},
		{"   +0", 0}, {"100%", 100}, {"42px", 42}, {"100,000", 100},
		{"3.14", 3}, {"23.4", 23}, {"3٠", 3},
		{"99999999999999999999", 1<<31 - 1}, {"-99999999999999999999", -(1<<31 - 1)},
	}
	for _, tt := range tests {
		if got, ok := ParseInteger(tt.in); !ok || got != tt.want {
			t.Errorf("ParseInteger(%q) = %d, %v; want %d", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range invalidIntegers {
		if got, ok := ParseInteger(in); ok {
			t.Errorf("ParseInteger(%q) = %d, want failure", in, got)
		}
	}
}

func TestParseNonNegativeInteger(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"42", 42}, {"0", 0}, {"100", 100}, {"+123", 123}, {"+0", 0},
		{"-0", 0}, {"  42", 42}, {"\t7", 7}, {"   +0", 0}, {"100%", 100},
		{"42px", 42}, {"100,000", 100}, {"3.14", 3}, {"23.4", 23}, {"3٠", 3},
	}
	for _, tt := range tests {
		if got, ok := ParseNonNegativeInteger(tt.in); !ok || got != tt.want {
			t.Errorf("ParseNonNegativeInteger(%q) = %d, %v; want %d", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range append([]string{"-42", " \n -5"}, invalidIntegers...) {
		if got, ok := ParseNonNegativeInteger(in); ok {
			t.Errorf("ParseNonNegativeInteger(%q) = %d, want failure", in, got)
		}
	}
}

var invalidDimensions = append([]string{"-42", " \n -5", "+123", "+0", "-0", "   +0"}, invalidIntegers...)

func TestParseDimensionValue(t *testing.T) {
	tests := []struct {
		in   string
		want Dimension
	}{
		{"42", Dimension{42, "px"}}, {"0", Dimension{0, "px"}},
		{"100", Dimension{100, "px"}}, {"  42", Dimension{42, "px"}},
		{"\t7", Dimension{7, "px"}}, {"100%", Dimension{100, "%"}},
		{"12.3%", Dimension{12.3, "%"}}, {"42px", Dimension{42, "px"}},
		{"100,000", Dimension{100, "px"}}, {"3.14", Dimension{3.14, "px"}},
		{"23.4", Dimension{23.4, "px"}}, {"3٠", Dimension{3, "px"}},
		{"3.", Dimension{3, "px"}}, {"3.%", Dimension{3, "%"}},
	}
	for _, tt := range tests {
		if got, ok := ParseDimensionValue(tt.in); !ok || got != tt.want {
			t.Errorf("ParseDimensionValue(%q) = %v, %v; want %v", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range invalidDimensions {
		if got, ok := ParseDimensionValue(in); ok {
			t.Errorf("ParseDimensionValue(%q) = %v, want failure", in, got)
		}
	}
}

func TestMapToPixelLength(t *testing.T) {
	tests := []struct{ in, want string }{
		{"42", "42px"}, {"0", "0px"}, {"100", "100px"}, {"+123", "123px"},
		{"+0", "0px"}, {"-0", "0px"}, {"  42", "42px"}, {"\t7", "7px"},
		{"   +0", "0px"}, {"100%", "100px"}, {"12.3%", "12px"}, {"42px", "42px"},
		{"100,000", "100px"}, {"3.14", "3px"}, {"23.4", "23px"}, {"3٠", "3px"},
	}
	for _, tt := range tests {
		if got, ok := MapToPixelLength(tt.in); !ok || got != tt.want {
			t.Errorf("MapToPixelLength(%q) = %q, %v; want %q", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range append([]string{"-42", " \n -5"}, invalidIntegers...) {
		if got, ok := MapToPixelLength(in); ok {
			t.Errorf("MapToPixelLength(%q) = %q, want failure", in, got)
		}
	}
}

func TestMapToDimensionProperty(t *testing.T) {
	tests := []struct{ in, want string }{
		{"42", "42.0px"}, {"0", "0.0px"}, {"100", "100.0px"}, {"  42", "42.0px"},
		{"\t7", "7.0px"}, {"100%", "100.0%"}, {"12.3%", "12.3%"},
		{"42.0px", "42.0px"}, {"100,000", "100.0px"}, {"3.14", "3.14px"},
		{"23.4", "23.4px"}, {"3٠", "3.0px"},
		{"0.00001", "1e-05px"}, {"10000000000000000", "1e+16px"},
	}
	for _, tt := range tests {
		if got, ok := MapToDimensionProperty(tt.in); !ok || got != tt.want {
			t.Errorf("MapToDimensionProperty(%q) = %q, %v; want %q", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range invalidDimensions {
		if got, ok := MapToDimensionProperty(in); ok {
			t.Errorf("MapToDimensionProperty(%q) = %q, want failure", in, got)
		}
	}
}

func TestMapToDimensionPropertyIgnoringZero(t *testing.T) {
	tests := []struct{ in, want string }{
		{"42", "42.0px"}, {"100", "100.0px"}, {"  42", "42.0px"}, {"\t7", "7.0px"},
		{"100%", "100.0%"}, {"12.3%", "12.3%"}, {"42.0px", "42.0px"},
		{"100,000", "100.0px"}, {"3.14", "3.14px"}, {"23.4", "23.4px"},
		{"3٠", "3.0px"},
	}
	for _, tt := range tests {
		if got, ok := MapToDimensionPropertyIgnoringZero(tt.in); !ok || got != tt.want {
			t.Errorf("MapToDimensionPropertyIgnoringZero(%q) = %q, %v; want %q", tt.in, got, ok, tt.want)
		}
	}
	for _, in := range append([]string{"0", "0%", "0.0"}, invalidDimensions...) {
		if got, ok := MapToDimensionPropertyIgnoringZero(in); ok {
			t.Errorf("MapToDimensionPropertyIgnoringZero(%q) = %q, want failure", in, got)
		}
	}
}

func TestParseLegacyColor(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"red", "#ff0000", true},
		{"  Blue\n", "#0000ff", true},
		{"#0f0", "#00ff00", true},
		{"#336699", "#336699", true},
		{"rgb(10, 20, 30)", "#0a141e", true},
		{"rgba(10, 20, 30, 0.5)", "#0a141e", true},
		{"rgb(300, -5, 128)", "#ff0080", true},
		{"hsl(120, 100%, 50%)", "#00ff00", true},
		{"transparent", "", false},
		{"CurrentColor", "", false},
		{"not-a-color", "", false},
		{"", "", false},
		{"#12", "", false},
	}
	for _, tt := range tests {
		got, ok := ParseLegacyColor(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseLegacyColor(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParseURL(t *testing.T) {
	const base = "http://example.org/dir/page.html"
	tests := []struct {
		name, in, base string
		allowRelative  bool
		want           string
		ok             bool
	}{
		{"relative", "img.png", base, false, "http://example.org/dir/img.png", true},
		{"padded", " \t\n img.png\f\r", base, false, "http://example.org/dir/img.png", true},
		{"line breaks inside", "im\ng\t.p\rng", base, false, "http://example.org/dir/img.png", true},
		{"absolute without a base", "http://example.org/a b.png", "", false, "http://example.org/a%20b.png", true},
		{"non-ASCII", "é.png", base, false, "http://example.org/dir/%C3%A9.png", true},
		{"relative allowed without a base", "img.png", "", true, "img.png", true},
		{"fragment", "#top", base, true, "http://example.org/dir/page.html#top", true},
		{"empty", "", base, true, "", false},
		{"only whitespace", " \t\n", base, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseURL(tt.in, tt.base, tt.allowRelative)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ParseURL(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseURLRelativeWithoutBaseIsReported(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	if got, ok := ParseURL("img.png", "", false); ok || got != "" {
		t.Errorf("got %q, %v", got, ok)
	}
	lines := logs.Lines()
	if len(lines) != 1 || !strings.Contains(lines[0], "Relative URI reference without a base URI: img.png") {
		t.Errorf("logs: %q", lines)
	}
}
