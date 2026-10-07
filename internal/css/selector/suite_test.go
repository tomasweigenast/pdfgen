package selector

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type suiteCases struct {
	Invalid []struct {
		Name     string
		Selector string
	}
	Valid []struct {
		Name        string
		Selector    string
		Expect      []string
		Unsupported bool
	}
	Select []struct {
		Selector    string
		Expect      []string // nil stands for every element of the document
		Unsupported bool
	}
	HTMLSelect []struct {
		Selector string
		Expect   []string
	} `json:"html_select"`
	Play []struct {
		Selector string
		Count    int
	}
}

func loadSuite(t *testing.T) *suiteCases {
	t.Helper()
	data, err := os.ReadFile("testdata/selector_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var c suiteCases
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	return &c
}

func parseFixture(t testing.TB, name string, html bool) *testElement {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return parseXML(t, string(data), html)
}

// namespaceTestDocument is the content document extended with elements in
// several namespaces under #root.
func namespaceTestDocument(t testing.TB) *testElement {
	t.Helper()
	doc := parseFixture(t, "document.xhtml", false)
	parent := doc.find("root")
	for _, id := range []string{"any-namespace", "no-namespace"} {
		div := parent.addChild(XHTMLNamespace, "div", id)
		div.addChild(XHTMLNamespace, "div", id+"-div1")
		div.addChild(XHTMLNamespace, "div", id+"-div2")
		div.addChild("", "div", id+"-div3")
		div.addChild("http://www.example.org/ns", "div", id+"-div4")
	}
	return doc
}

func TestInvalidSelectorsAreRejected(t *testing.T) {
	for _, c := range loadSuite(t).Invalid {
		if _, err := CompileString(c.Selector, nil); err == nil {
			t.Errorf("%q should be invalid (%s)", c.Selector, c.Name)
		}
	}
}

func TestSelectorsOnTheContentDocument(t *testing.T) {
	root := namespaceTestDocument(t)
	for _, c := range loadSuite(t).Valid {
		if c.Unsupported {
			continue
		}
		got := []string{}
		for _, el := range queryAll(t, root, c.Selector) {
			got = append(got, el.ID())
		}
		if !reflect.DeepEqual(got, c.Expect) {
			t.Errorf("%q: got %v, want %v (%s)", c.Selector, got, c.Expect, c.Name)
		}
	}
}

func TestSelectorsOnTheIDsDocument(t *testing.T) {
	suite := loadSuite(t)
	xmlRoot := parseFixture(t, "ids.xhtml", false)
	htmlRoot := parseFixture(t, "ids.xhtml", true)
	all := idsOf(queryAll(t, htmlRoot, "*"))

	for _, c := range suite.Select {
		if c.Unsupported {
			continue
		}
		want := c.Expect
		if want == nil {
			want = all
		}
		if got := idsOf(queryAll(t, xmlRoot, c.Selector)); !reflect.DeepEqual(got, want) {
			t.Errorf("xml %q: got %v, want %v", c.Selector, got, want)
		}
		if got := idsOf(queryAll(t, htmlRoot, c.Selector)); !reflect.DeepEqual(got, want) {
			t.Errorf("html %q: got %v, want %v", c.Selector, got, want)
		}
	}
}

func TestHTMLDocumentsMatchCaseInsensitively(t *testing.T) {
	xmlRoot := parseFixture(t, "ids.xhtml", false)
	htmlRoot := parseFixture(t, "ids.xhtml", true)
	for _, c := range loadSuite(t).HTMLSelect {
		if got := queryAll(t, xmlRoot, c.Selector); len(got) != 0 {
			t.Errorf("xml %q: got %v, want no match", c.Selector, idsOf(got))
		}
		if got := idsOf(queryAll(t, htmlRoot, c.Selector)); !reflect.DeepEqual(got, c.Expect) {
			t.Errorf("html %q: got %v, want %v", c.Selector, got, c.Expect)
		}
	}
}

func TestSelectorsOnALargeDocument(t *testing.T) {
	root := parseFixture(t, "play.xhtml", false)
	var found *testElement
	root.walk(func(el *testElement) {
		if found == nil && el.name.Space == XHTMLNamespace && el.name.Local == "body" {
			found = el
		}
	})
	if found == nil {
		t.Fatal("no body element")
	}
	// The body is matched as if it were the document root.
	found.parent, found.index = nil, 0

	for _, c := range loadSuite(t).Play {
		if got := len(queryAll(t, found, c.Selector)); got != c.Count {
			t.Errorf("%q: %d matches, want %d", c.Selector, got, c.Count)
		}
	}
}
