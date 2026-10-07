package logging

import "testing"

func TestReprQuotesLikeMessages(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "''"},
		{"print", "'print'"},
		{"it's", `"it's"`},
		{`it's "x"`, `'it\'s "x"'`},
		{`a\b`, `'a\\b'`},
		{"a\nb\tc", `'a\nb\tc'`},
		{"\x01", `'\x01'`},
		{"ünicōde", "'ünicōde'"},
	}
	for _, tt := range tests {
		if got := Repr(tt.in); got != tt.want {
			t.Errorf("Repr(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
