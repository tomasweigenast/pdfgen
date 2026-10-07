package selector

import (
	"errors"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func parseTest(t *testing.T, css string, ns map[string]string) ([]*parsedSelector, error) {
	t.Helper()
	return parseSelectorList(syntax.ParseComponentValueList(css, false), ns, false, false)
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		selector string
		want     [3]int
	}{
		{"*", [3]int{0, 0, 0}},
		{"div", [3]int{0, 0, 1}},
		{"#a .b c", [3]int{1, 1, 1}},
		{"a[href]", [3]int{0, 1, 1}},
		{"a:hover", [3]int{0, 1, 1}},
		{"li:nth-child(2n+1)", [3]int{0, 1, 1}},
		{"a:not(#x)", [3]int{1, 0, 1}},
		{"a:not(#x, .y.z)", [3]int{1, 0, 1}},
		{":is(#a, .b)", [3]int{1, 0, 0}},
		{":is(div, #a > p)", [3]int{1, 0, 1}},
		{":is()", [3]int{0, 0, 0}},
		{":where(#a .b)", [3]int{0, 0, 0}},
		{"div:where(#a)", [3]int{0, 0, 1}},
		{"ol:has(> li.c, #x)", [3]int{1, 0, 1}},
		{"::before", [3]int{0, 0, 1}},
		{"p::first-line", [3]int{0, 0, 2}},
		{"p:after", [3]int{0, 0, 2}},
		{"a > b + c ~ d e", [3]int{0, 0, 5}},
		{"ns|a", [3]int{0, 0, 1}},
		{"[ns|a=b]", [3]int{0, 1, 0}},
	}
	ns := map[string]string{"ns": "http://example.org/ns"}
	for _, tt := range tests {
		list, err := parseTest(t, tt.selector, ns)
		if err != nil {
			t.Errorf("%q: %v", tt.selector, err)
			continue
		}
		if got := list[0].spec; got != tt.want {
			t.Errorf("%q: specificity %v, want %v", tt.selector, got, tt.want)
		}
	}
}

func TestPseudoElements(t *testing.T) {
	accepted := []string{
		"before", "after", "marker", "first-line", "first-letter",
		"footnote-call", "footnote-marker", "note-call", "note-marker",
		"note-callback", "selection", "placeholder",
	}
	for _, name := range accepted {
		list, err := parseTest(t, "p::"+name, nil)
		if err != nil || list[0].pseudoElement != name {
			t.Errorf("p::%s: got %v, %v", name, list, err)
		}
	}
	for _, name := range []string{"before", "after", "first-line", "first-letter"} {
		list, err := parseTest(t, "p:"+name, nil)
		if err != nil || list[0].pseudoElement != name {
			t.Errorf("p:%s: got %v, %v", name, list, err)
		}
	}
	list, err := parseTest(t, "P::BEFORE", nil)
	if err != nil || list[0].pseudoElement != "before" {
		t.Errorf("case-insensitive pseudo-element: got %v, %v", list, err)
	}
	for _, css := range []string{"p::unknown", "p::before:hover", "p::before b", "p::-x-foo", "p::before::after"} {
		if _, err := parseTest(t, css, nil); err == nil {
			t.Errorf("%q should be rejected", css)
		}
	}
}

func TestParseErrors(t *testing.T) {
	ns := map[string]string{"a": "urn:a"}
	for _, css := range []string{
		"", " ", ",", "a,", ",a", "a b,", "a >", "> a", "a > > b", "a +",
		"a[", "a[]", "a[=b]", "a[b=]", "a[b=c d]", "a[b=1]", "a[b c]",
		"a[b=c x]", "a[*]", "a[*|*]", "a[x|*]", "a[|]",
		"b|a", "a:", "a::", "a:::before", "a:()", "a.", "a.1", "a#", "a:1",
		"a|", "a|1", "*|", ":not()x", "a)", "a{}", "a:not(b,)",
	} {
		if _, err := parseTest(t, css, ns); err == nil {
			t.Errorf("%q should be rejected", css)
		} else {
			var e *Error
			if !errors.As(err, &e) {
				t.Errorf("%q: error %T is not *Error", css, err)
			}
		}
	}
}

func TestParseAccepts(t *testing.T) {
	ns := map[string]string{"a": "urn:a", "": "urn:default"}
	for _, css := range []string{
		"a", "*", "a b", "a>b", "a > b", "a+b", "a~b", "a  >  b",
		"a /**/ b", "a/**/>b", "/**/a", "a,b", "a , b", " a ", "a:hover",
		"[b]", "[ b ]", "[b=c]", "[b = c]", "[b=\"c\"]", "[b=c i]",
		"[b=c s]", "[b=c I]", "[b|=c]", "[b ~= c ]", "[a|b]", "[|b]", "[*|b]",
		"a|b", "a|*", "*|b", "*|*", "|b", "|*", "#id", ".c", "a.c.d#e",
		":is(a, b)", ":not(a)", ":has(> a)", ":has(a, + b, ~ c)",
		":nth-child(2n+1 of .x)", ":lang(en)", "a::before", "::before",
		"a:not(:not(b))", ":is()", ":where()", "a:unknown-yet",
	} {
		if _, err := parseTest(t, css, ns); err != nil {
			t.Errorf("%q: %v", css, err)
		}
	}
}

func TestForgivingSelectorLists(t *testing.T) {
	tests := []struct {
		css  string
		want int
	}{
		{"div", 1},
		{":::wrong", 0},
		{":::wrong, div", 0},
		{"div, :::wrong, fieldset", 2},
		{"div, :::wrong fieldset, p", 2},
		{"div, , p", 2},
		{"", 0},
		{"div p q r", 1},
	}
	for _, tt := range tests {
		list, err := parseSelectorList(syntax.ParseComponentValueList(tt.css, false), nil, true, false)
		if err != nil || len(list) != tt.want {
			t.Errorf("%q: got %d selectors, err %v; want %d", tt.css, len(list), err, tt.want)
		}
	}
}

func TestRelativeSelectorsKeepTheirCombinator(t *testing.T) {
	list, err := parseSelectorList(syntax.ParseComponentValueList("> a, b, + c, ~ d", false), nil, false, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{'>', ' ', '+', '~'}
	if len(list) != len(want) {
		t.Fatalf("got %d selectors", len(list))
	}
	for i, sel := range list {
		if sel.combinator != want[i] {
			t.Errorf("selector %d: combinator %q, want %q", i, sel.combinator, want[i])
		}
	}
}

func TestNamespaceResolution(t *testing.T) {
	ns := map[string]string{"x": "urn:x", "": "urn:default", "none": ""}
	tests := []struct {
		css       string
		wantNames []string
		wantNS    []string
	}{
		{"a", []string{"a"}, []string{"urn:default"}},
		{"*", nil, []string{"urn:default"}},
		{"*|a", []string{"a"}, nil},
		{"|a", []string{"a"}, []string{""}},
		{"x|a", []string{"a"}, []string{"urn:x"}},
		{"none|a", []string{"a"}, []string{""}},
		{"x|*", nil, []string{"urn:x"}},
	}
	for _, tt := range tests {
		list, err := parseTest(t, tt.css, ns)
		if err != nil {
			t.Errorf("%q: %v", tt.css, err)
			continue
		}
		var names, spaces []string
		for _, s := range list[0].tree.(*compound).simples {
			switch s := s.(type) {
			case *localName:
				names = append(names, s.name)
			case *namespaceSel:
				spaces = append(spaces, s.namespace)
			}
		}
		if !equalStrings(names, tt.wantNames) || !equalStrings(spaces, tt.wantNS) {
			t.Errorf("%q: names %q namespaces %q, want %q %q", tt.css, names, spaces, tt.wantNames, tt.wantNS)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
