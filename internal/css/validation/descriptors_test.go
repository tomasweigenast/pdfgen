package validation

import (
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// fontFaceDescriptors parses the single @font-face rule in css and validates
// its descriptors, returning them with the lines that were logged.
func fontFaceDescriptors(t *testing.T, css string) ([]Descriptor, []string) {
	t.Helper()
	rules := syntax.ParseStylesheet(css, false, false)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	rule, ok := rules[0].(*syntax.AtRule)
	if !ok || rule.AtKeyword != "font-face" {
		t.Fatalf("got %#v, want an @font-face rule", rules[0])
	}
	capture := logging.StartCapture()
	result := PreprocessDescriptors("font-face", testBaseURL, syntax.ParseBlocksContents(rule.Content, false, false))
	capture.Close()
	return result, capture.Lines()
}

func external(address string) urls.Link {
	return urls.Link{Kind: urls.LinkExternal, Target: address}
}

func assertDescriptors(t *testing.T, css string, wantLogs []string, want ...Descriptor) {
	t.Helper()
	got, logs := fontFaceDescriptors(t, css)
	if !reflect.DeepEqual(logs, wantLogs) && !(len(logs) == 0 && len(wantLogs) == 0) {
		t.Errorf("logs = %q, want %q", logs, wantLogs)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwant %#v", got, want)
	}
}

func TestFontFaceWithFamilyAndSource(t *testing.T) {
	assertDescriptors(t, `@font-face {
  font-family: Gentium Hard;
  src: url(https://example.com/fonts/Gentium.woff);
}`, nil,
		Descriptor{"font-family", "Gentium Hard"},
		Descriptor{"src", []FontSource{{Link: external("https://example.com/fonts/Gentium.woff")}}})
}

func TestFontFaceWithRelativeSourceAndStyle(t *testing.T) {
	assertDescriptors(t, `@font-face {
  font-family: "Fonty Smiley";
  src: url(Fonty-Smiley.woff);
  font-style: italic;
  font-weight: 200;
  font-stretch: condensed;
}`, nil,
		Descriptor{"font-family", "Fonty Smiley"},
		Descriptor{"src", []FontSource{{Link: external("https://example.org/foo/Fonty-Smiley.woff")}}},
		Descriptor{"font-style", props.Keyword("italic")},
		Descriptor{"font-weight", props.Integer(200)},
		Descriptor{"font-stretch", props.Keyword("condensed")})
}

func TestFontFaceWithEmptyLocalSource(t *testing.T) {
	assertDescriptors(t, `@font-face { font-family: Gentium Hard; src: local(); }`, nil,
		Descriptor{"font-family", "Gentium Hard"},
		Descriptor{"src", []FontSource{{Local: true}}})
}

func TestFontFaceWithLocalSourceName(t *testing.T) {
	assertDescriptors(t, `@font-face { font-family: Gentium Hard; src: local(Gentium Hard); }`, nil,
		Descriptor{"font-family", "Gentium Hard"},
		Descriptor{"src", []FontSource{{Local: true, Name: "Gentium Hard"}}})
}

func TestFontFaceKeepsEarlierSourceWhenALaterOneIsInvalid(t *testing.T) {
	got, logs := fontFaceDescriptors(t, `@font-face {
  font-family: Gentium Hard;
  src: local(Gentium Hard);
  src: local(Gentium Soft),
}`)
	want := []Descriptor{
		{"font-family", "Gentium Hard"},
		{"src", []FontSource{{Local: true, Name: "Gentium Hard"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	if len(logs) != 1 || !contains2(logs[0], "invalid value") {
		t.Errorf("logs = %q, want one about an invalid value", logs)
	}
}

func contains2(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestFontFaceIgnoresInvalidDescriptors(t *testing.T) {
	assertDescriptors(t, `@font-face {  font-family: "Bad Font";  src: url(BadFont.woff);  font-stretch: expanded;  font-style: wrong;  font-weight: bolder;  font-stretch: wrong;}`,
		[]string{
			"WARNING: Ignored `font-style: wrong` at 1:91, invalid value.",
			"WARNING: Ignored `font-weight: bolder` at 1:111, invalid value.",
			"WARNING: Ignored `font-stretch: wrong` at 1:133, invalid value.",
		},
		Descriptor{"font-family", "Bad Font"},
		Descriptor{"src", []FontSource{{Link: external("https://example.org/foo/BadFont.woff")}}},
		Descriptor{"font-stretch", props.Keyword("expanded")})
}

func TestFontFaceWarnsAboutUnsupportedDescriptor(t *testing.T) {
	_, logs := fontFaceDescriptors(t, `@font-face { font-family: test; unknown: 1; font-display: swap }`)
	if len(logs) != 1 || !contains2(logs[0], "descriptor not supported") {
		t.Errorf("logs = %q", logs)
	}
}

func TestFontFaceDescriptorsWithInvalidFamilyAndSource(t *testing.T) {
	tests := []struct {
		css  string
		logs []string
	}{
		{"@font-face { font-family: test; src: wrong }", []string{"WARNING: Ignored `src: wrong ` at 1:33, invalid value."}},
		{"@font-face { font-family: good, bad; src: url(test.woff) }", []string{"WARNING: Ignored `font-family: good, bad` at 1:14, invalid value."}},
		{"@font-face { font-family: good, bad; src: really bad }", []string{
			"WARNING: Ignored `font-family: good, bad` at 1:14, invalid value.",
			"WARNING: Ignored `src: really bad ` at 1:38, invalid value."}},
	}
	for _, tt := range tests {
		_, logs := fontFaceDescriptors(t, tt.css)
		if !reflect.DeepEqual(logs, tt.logs) {
			t.Errorf("%s\n logs %q\n want %q", tt.css, logs, tt.logs)
		}
	}
}

func counterStyleDescriptors(t *testing.T, css string) map[string]any {
	t.Helper()
	rule := syntax.ParseStylesheet(css, false, false)[0].(*syntax.AtRule)
	capture := logging.StartCapture()
	result := PreprocessDescriptors("counter-style", testBaseURL, syntax.ParseBlocksContents(rule.Content, false, false))
	capture.Close()
	if logs := capture.Lines(); len(logs) != 0 {
		t.Fatalf("%s: unexpected logs %q", css, logs)
	}
	m := map[string]any{}
	for _, d := range result {
		m[d.Name] = d.Value
	}
	return m
}

func TestCounterStyleDescriptors(t *testing.T) {
	got := counterStyleDescriptors(t, `@counter-style test {
  system: fixed 3; negative: "(" ")"; prefix: "<"; suffix: ">"; range: 1 5, infinite 0;
  pad: 3 "0"; fallback: lower-roman; symbols: "a" b;
}`)
	want := map[string]any{
		"system":   CounterSystem{Kind: "fixed", Number: 3},
		"negative": []CounterSymbol{str("("), str(")")},
		"prefix":   str("<"), "suffix": str(">"),
		"range":    []CounterRange{{Lower: 1, Upper: 5}, {Lower: inf(-1), Upper: 0}},
		"pad":      CounterPad{Width: 3, Symbol: str("0")},
		"fallback": "lower-roman",
		"symbols":  []CounterSymbol{str("a"), str("b")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v\nwant %#v", got, want)
	}

	got = counterStyleDescriptors(t, `@counter-style test { system: additive; additive-symbols: 10 x, 5 v, 1 i }`)
	if want := []CounterPad{{10, str("x")}, {5, str("v")}, {1, str("i")}}; !reflect.DeepEqual(got["additive-symbols"], want) {
		t.Errorf("additive-symbols = %#v", got["additive-symbols"])
	}
	if got["system"] != (CounterSystem{Kind: "additive"}) {
		t.Errorf("system = %#v", got["system"])
	}
}

func TestCounterStyleInvalidDescriptors(t *testing.T) {
	for _, decl := range []string{
		"additive-symbols: 10 x, 1 i, 5 v", "range: 5 1", "pad: -1 x",
		"fallback: none", "system: extends", "negative: a b c",
	} {
		rule := syntax.ParseStylesheet("@counter-style test {"+decl+"}", false, false)[0].(*syntax.AtRule)
		capture := logging.StartCapture()
		result := PreprocessDescriptors("counter-style", testBaseURL, syntax.ParseBlocksContents(rule.Content, false, false))
		capture.Close()
		if len(result) != 0 || len(capture.Lines()) != 1 {
			t.Errorf("%s: got %#v with logs %q", decl, result, capture.Lines())
		}
	}
}

func TestColorProfileDescriptors(t *testing.T) {
	rule := syntax.ParseStylesheet(`@color-profile --p { src: url(p.icc); rendering-intent: perceptual; components: a b, c }`, false, false)[0].(*syntax.AtRule)
	got := PreprocessDescriptors("color-profile", testBaseURL, syntax.ParseBlocksContents(rule.Content, false, false))
	want := []Descriptor{
		{"src", external("https://example.org/foo/p.icc")},
		{"rendering-intent", "perceptual"},
		{"components", [][]string{{"a", "b"}, {"c"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
