package media

import (
	"slices"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

func TestParseMediaTypes(t *testing.T) {
	tests := []struct {
		css  string
		want Query
		ok   bool
	}{
		{"", Query{"all"}, true},
		{"  ", Query{"all"}, true},
		{"print", Query{"print"}, true},
		{"PRINT", Query{"print"}, true},
		{"only screen", Query{"screen"}, true},
		{"screen, print", Query{"screen", "print"}, true},
		{"screen and (color)", nil, false},
		{`("unknown content")`, nil, false},
		{"screen,", nil, false},
	}
	for _, tt := range tests {
		c := logging.StartCapture()
		got, ok := Parse(syntax.ParseComponentValueList(tt.css, false))
		c.Close()
		if ok != tt.ok || !slices.Equal(got, tt.want) {
			t.Errorf("Parse(%q) = %q, %v; want %q, %v", tt.css, got, ok, tt.want, tt.ok)
		}
		if warned := len(c.Lines()) > 0; warned == tt.ok {
			t.Errorf("Parse(%q) logged %q", tt.css, c.Lines())
		}
	}
}

func TestInvalidMediaTypeWarning(t *testing.T) {
	c := logging.StartCapture()
	defer c.Close()
	Parse(syntax.ParseComponentValueList("screen and (color)", false))
	want := []string{"WARNING: Expected a media type, got 'screen/**/and/**/(color)'"}
	if got := c.Lines(); !slices.Equal(got, want) {
		t.Errorf("logs = %q, want %q", got, want)
	}
}

func TestEvaluateMatchesTypeOrAll(t *testing.T) {
	tests := []struct {
		q    Query
		want bool
	}{
		{Query{"all"}, true},
		{Query{"print"}, true},
		{Query{"screen"}, false},
		{Query{"screen", "print"}, true},
		{Query{}, false},
	}
	for _, tt := range tests {
		if got := Evaluate(tt.q, "print"); got != tt.want {
			t.Errorf("Evaluate(%q, print) = %v, want %v", tt.q, got, tt.want)
		}
	}
}
