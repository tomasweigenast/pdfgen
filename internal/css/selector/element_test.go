package selector

import (
	"reflect"
	"testing"
)

func TestSplitWhitespace(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{" \t\r\n\f", []string{}},
		{"a", []string{"a"}},
		{" a  b\tc\nd\fe ", []string{"a", "b", "c", "d", "e"}},
		{"a b", []string{"a b"}},
	}
	for _, tt := range tests {
		if got := SplitWhitespace(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SplitWhitespace(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseContentLanguage(t *testing.T) {
	tests := []struct{ in, want string }{
		{"en", "en"},
		{" fr \t", "fr"},
		{"en, es", ""},
		{"en es", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ParseContentLanguage(tt.in); got != tt.want {
			t.Errorf("ParseContentLanguage(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
