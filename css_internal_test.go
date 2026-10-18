package pdfgen

import (
	"strings"
	"testing"
)

func TestCSSDecoding(t *testing.T) {
	tests := []struct {
		name  string
		input string
		opts  []SourceOption
		want  string
	}{
		{"utf-8 by default", "a::before { content: \"é\" }", nil, "a::before { content: \"é\" }"},
		{"forced encoding", "a::before { content: \"\xe9\" }", []SourceOption{WithEncoding("iso-8859-1")}, "a::before { content: \"é\" }"},
		{"byte order mark", "\xef\xbb\xbfa { }", nil, "a { }"},
		{"@charset", "@charset \"iso-8859-1\"; a::before { content: \"\xe9\" }", nil, "@charset \"iso-8859-1\"; a::before { content: \"é\" }"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			css, err := CSSFromReader(strings.NewReader(tt.input), tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if css.text != tt.want {
				t.Errorf("text = %q, want %q", css.text, tt.want)
			}
		})
	}
}
