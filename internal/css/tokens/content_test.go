package tokens

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func TestURLAndContentParsing(t *testing.T) {
	for _, tt := range []struct {
		css, base string
		ok        bool
	}{
		{"url(https://example.com/a.png)", "", true},
		{"url(../a.png)", "https://example.com/css/main.css", true},
		{"url(../a.png)", "", false},
		{"attr(data-link url)", "", true},
		{"red", "", false},
	} {
		got, ok, err := GetURL(parseOne(t, tt.css), tt.base)
		if ok != tt.ok || (!tt.ok && tt.base == "" && tt.css != "red" && err == nil) {
			t.Errorf("GetURL(%q) = %v, %v; want valid=%v", tt.css, got, ok, tt.ok)
		}
		if ok && tt.base != "" && tt.css == "url(../a.png)" {
			if external, yes := got.External(); !yes || external != "https://example.com/a.png" {
				t.Errorf("resolved URL = %q, %v", external, yes)
			}
		}
	}
	if _, ok := GetString(parseOne(t, `"hello"`)); !ok {
		t.Fatal("quoted string was rejected")
	}
	for _, css := range []string{`attr(title)`, `counter(chapter)`, `counters(chapter, ".")`, `string(chapter)`, `content(text)`} {
		if _, ok := GetString(parseOne(t, css)); !ok {
			t.Errorf("GetString(%q) rejected a supported function", css)
		}
	}
}

func TestContentListsAndTargets(t *testing.T) {
	tokens := syntax.ParseComponentValueList(`"before" contents url(#target)`, true)
	items, ok, err := GetContentList(RemoveWhitespace(tokens), "https://example.com/page.html")
	if err != nil || !ok || len(items) != 3 || items[1].Type != "content()" || items[2].Args[0] != "internal" || items[2].Args[1] != "target" {
		t.Fatalf("GetContentList() = %#v, %v, %v", items, ok, err)
	}
	if _, ok, err := GetContentList([]syntax.Node{parseOne(t, "unknown-token")}, ""); err != nil || ok {
		t.Fatalf("invalid content list = %v, %v; want false without error", ok, err)
	}
	for _, tt := range []struct {
		css string
		ok  bool
	}{
		{`target-text(url(#x))`, true},
		{`target-text(url(#x), before)`, true},
		{`target-text(url(#x), invalid)`, false},
		{`target-counter(url(#x), chapter, upper-roman)`, true},
		{`target-counters(url(#x), chapter, ".", decimal)`, true},
		{`counter(chapter)`, false},
	} {
		_, got, err := GetTarget(parseOne(t, tt.css), "https://example.com/")
		if err != nil || got != tt.ok {
			t.Errorf("GetTarget(%q) = %v, %v; want %v, nil", tt.css, got, err, tt.ok)
		}
	}
}

func TestImageAndListStyleParsing(t *testing.T) {
	for _, css := range []string{
		`url("image.png")`,
		`linear-gradient(to right, red, blue)`,
		`linear-gradient(red, blue)`,
		`radial-gradient(circle at center, red 0%, blue 100%)`,
		`radial-gradient(red, blue)`,
	} {
		img, ok, err := GetImage(parseOne(t, css), "https://example.com/css/")
		if err != nil || !ok || img.Type == "" {
			t.Errorf("GetImage(%q) = %#v, %v, %v", css, img, ok, err)
		}
	}
	for _, css := range []string{"none", "disc", "decimal", `"*"`, "upper-roman"} {
		if _, ok := ParseListStyleType(parseOne(t, css)); !ok {
			t.Errorf("ParseListStyleType(%q) rejected a valid value", css)
		}
	}
	for _, css := range []string{`symbols("*" "†")`, `symbols(numeric "0" "1")`, `symbols(alphabetic "a" "b")`} {
		if _, ok := ParseListStyleType(parseOne(t, css)); !ok {
			t.Errorf("ParseListStyleType(%q) rejected symbol notation", css)
		}
	}
	for _, css := range []string{`symbols()`, `symbols(numeric "0")`, `symbols(invalid "x")`, `symbols("x" 1)`} {
		if _, ok := ParseListStyleType(parseOne(t, css)); ok {
			t.Errorf("ParseListStyleType(%q) accepted invalid symbols", css)
		}
	}
	if _, ok := ParseListStyleType(parseOne(t, "12px")); ok {
		t.Error("ParseListStyleType accepted an unknown value")
	}
	if _, ok, err := GetImage(parseOne(t, "url(relative.png)"), ""); err == nil || ok {
		t.Errorf("relative image without a base = %v, %v", ok, err)
	}
	external, ok, err := GetURL(parseOne(t, "url(https://example.com/a.png)"), "")
	if err != nil || !ok {
		t.Fatalf("GetURL(external) = %v, %v", ok, err)
	}
	if address, ok := external.External(); !ok || address != "https://example.com/a.png" {
		t.Errorf("External() = %q, %v", address, ok)
	}
	if _, ok, err := GetContentListToken(parseOne(t, "close-quote"), ""); err != nil || !ok {
		t.Errorf("quote content item = %v, %v", ok, err)
	}
}

func TestGradientParametersAndColorStops(t *testing.T) {
	for _, tt := range []struct {
		css       string
		linear    bool
		repeating bool
		ok        bool
	}{
		{`linear-gradient(to top left, red 10%, 30%, blue)`, true, false, true},
		{`repeating-linear-gradient(45deg, red, blue)`, true, true, true},
		{`radial-gradient(circle closest-side at right top, red, blue)`, false, false, true},
		{`radial-gradient(10px 20px at 25% 75%, red, blue)`, false, false, true},
		{`repeating-radial-gradient(ellipse farthest-corner, red, blue)`, false, true, true},
		{`linear-gradient(red 20%)`, true, false, true},
		{`linear-gradient(red, 20%, blue)`, true, false, true},
		{`image-set(url(a.png) 1x)`, false, false, false},
	} {
		img, ok, err := GetImage(parseOne(t, tt.css), "https://example.com/")
		if err != nil || ok != tt.ok {
			t.Errorf("GetImage(%q) = %v, %v", tt.css, ok, err)
			continue
		}
		if !ok {
			continue
		}
		if tt.linear && (img.Linear == nil || img.Linear.Gradient.Repeating != tt.repeating) {
			t.Errorf("linear gradient = %#v", img.Linear)
		}
		if !tt.linear && img.Radial != nil && img.Radial.Gradient.Repeating != tt.repeating {
			t.Errorf("radial gradient = %#v", img.Radial)
		}
	}
	for _, css := range []string{"leader(space)", "open-quote", "no-close-quote", `"literal"`, "contents"} {
		if _, ok, err := GetContentListToken(parseOne(t, css), "https://example.com/"); err != nil || !ok {
			t.Errorf("GetContentListToken(%q) = %v, %v", css, ok, err)
		}
	}
}

func TestTokenTextExtraction(t *testing.T) {
	for _, tt := range []struct {
		token syntax.Node
		want  string
	}{
		{parseOne(t, `"value"`), "value"},
		{parseOne(t, "name"), "name"},
		{syntax.ParseComponentValueList(" ", false)[0], " "},
		{parseOne(t, "url(image.png)"), "image.png"},
		{parseOne(t, "!"), "!"},
		{parseOne(t, "func(value)"), ""},
	} {
		if got := valueOfNode(tt.token); got != tt.want {
			t.Errorf("valueOfNode(%s) = %q, want %q", tt.token.Serialize(), got, tt.want)
		}
	}
}
