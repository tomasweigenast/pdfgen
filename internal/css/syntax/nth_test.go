package syntax

import (
	"encoding/json"
	"testing"
)

func TestParseNth(t *testing.T) {
	for i, c := range loadSuite(t, "An+B.json") {
		var css string
		if err := json.Unmarshal(c.input, &css); err != nil {
			t.Fatal(err)
		}
		a, b, ok := ParseNth(css)
		switch want := c.expected.(type) {
		case nil:
			if ok {
				t.Errorf("case %d %q: got (%d, %d), want invalid", i, css, a, b)
			}
		case []any:
			if !ok || float64(a) != want[0] || float64(b) != want[1] {
				t.Errorf("case %d %q: got (%d, %d, %v), want %v", i, css, a, b, ok, want)
			}
		}
	}
}

func TestParseNthIncompleteInput(t *testing.T) {
	for _, css := range []string{"+", "n+", "n-", "2n -", "2n +"} {
		if _, _, ok := ParseNth(css); ok {
			t.Errorf("%q should be invalid", css)
		}
	}
}
