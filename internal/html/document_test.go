package html

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/selector"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

func parseSource(t *testing.T, src *urls.Source, encoding string) *Document {
	t.Helper()
	defer src.Body.Close()
	var opts []Option
	if src.ProtocolEncoding != "" {
		opts = append(opts, WithTransportEncoding(src.ProtocolEncoding))
	}
	doc, err := Parse(src.Body, encoding, src.BaseURL, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func tagNames(elements []*Element) []string {
	names := make([]string, len(elements))
	for i, e := range elements {
		names[i] = e.LocalName()
	}
	return names
}

func checkDoc1(t *testing.T, doc *Document, hasBase bool) {
	t.Helper()
	root := doc.Root
	if root.LocalName() != "html" {
		t.Fatalf("root is %q", root.LocalName())
	}
	if got := strings.Join(tagNames(root.Children()), " "); got != "head body" {
		t.Fatalf("root children: %s", got)
	}
	body := root.Children()[1]
	if got := strings.Join(tagNames(body.Children()), " "); got != "h1 p ul div" {
		t.Fatalf("body children: %s", got)
	}
	if got := body.Children()[0].ChildText(); got != "Typesetter test document (with Ünicōde)" {
		t.Errorf("heading text = %q", got)
	}
	if !hasBase {
		if doc.BaseURL != "" {
			t.Errorf("BaseURL = %q, want none", doc.BaseURL)
		}
		return
	}
	url := urls.Join(doc.BaseURL, "pattern.png")
	if !strings.HasPrefix(url, "file:") || !strings.HasSuffix(url, "testdata/resources/pattern.png") {
		t.Errorf("pattern.png resolves to %q", url)
	}
}

func TestParseResourceFromEverySource(t *testing.T) {
	ctx := context.Background()
	abs := testutil.ResourcePath(t, "doc1.html")
	absURL := urls.PathToURL(abs)

	tests := []struct {
		name string
		open func(t *testing.T) *Document
	}{
		{"absolute path", func(t *testing.T) *Document {
			src, err := urls.OpenPath(abs, "")
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, src, "")
		}},
		{"guessed path", func(t *testing.T) *Document {
			src, err := urls.Guess(ctx, nil, abs, "")
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, src, "")
		}},
		{"guessed URL", func(t *testing.T) *Document {
			src, err := urls.Guess(ctx, nil, absURL, "")
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, src, "")
		}},
		{"URL", func(t *testing.T) *Document {
			src, err := urls.OpenURL(ctx, nil, absURL, "")
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, src, "")
		}},
		{"relative path", func(t *testing.T) *Document {
			t.Chdir(filepath.Dir(filepath.Dir(abs)))
			src, err := urls.OpenPath(filepath.Join("resources", "doc1.html"), "")
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, src, "")
		}},
		{"named reader", func(t *testing.T) *Document {
			f, err := os.Open(abs)
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, urls.FromReader(f, f.Name(), ""), "")
		}},
		{"bytes with relative base", func(t *testing.T) *Document {
			t.Chdir(filepath.Dir(filepath.Dir(abs)))
			content, err := os.ReadFile(abs)
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, urls.FromBytes(content, filepath.Join("resources", "doc1.html")), "")
		}},
		{"string with relative base", func(t *testing.T) *Document {
			t.Chdir(filepath.Dir(filepath.Dir(abs)))
			content, err := os.ReadFile(abs)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := ParseString(string(content), urls.EnsureURL(filepath.Join("resources", "doc1.html")))
			if err != nil {
				t.Fatal(err)
			}
			return doc
		}},
		{"UTF-16BE with forced encoding", func(t *testing.T) *Document {
			path := testutil.ResourcePath(t, "doc1_UTF-16BE.html")
			src, err := urls.OpenPath(path, "")
			if err != nil {
				t.Fatal(err)
			}
			return parseSource(t, src, "UTF-16BE")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkDoc1(t, tt.open(t), true)
		})
	}
}

func TestParseStringWithoutBase(t *testing.T) {
	content, err := os.ReadFile(testutil.ResourcePath(t, "doc1.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ParseString(string(content), "")
	if err != nil {
		t.Fatal(err)
	}
	checkDoc1(t, doc, false)
}

func TestParseBaseElement(t *testing.T) {
	const page = `<html><head>%s<meta charset=utf-8></head><body></body></html>`
	tests := []struct {
		name, base, html, want string
	}{
		{"relative href", "file:///dir/", `<base href="resources/">`, "file:///dir/resources/"},
		{"absolute href", "file:///dir/", `<base href="http://example.org/x/">`, "http://example.org/x/"},
		{"href without a source base", "", `<base href="http://example.org/x/">`, "http://example.org/x/"},
		{"relative href without a source base", "", `<base href="resources/">`, "resources/"},
		{"missing href", "file:///dir/", `<base>`, "file:///dir/"},
		{"empty href", "file:///dir/", `<base href="  ">`, "file:///dir/"},
		{"padded href", "file:///dir/", `<base href=" sub/ ">`, "file:///dir/sub/"},
		{"first base wins", "file:///dir/", `<base href="a/"><base href="b/">`, "file:///dir/a/"},
		{"first base decides even without href", "file:///dir/", `<base><base href="b/">`, "file:///dir/"},
		{"no base element", "file:///dir/", ``, "file:///dir/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseString(strings.Replace(page, "%s", tt.html, 1), tt.base)
			if err != nil {
				t.Fatal(err)
			}
			if doc.BaseURL != tt.want {
				t.Errorf("BaseURL = %q, want %q", doc.BaseURL, tt.want)
			}
		})
	}
}

func TestParseBaseElementInSVGIgnored(t *testing.T) {
	doc, err := ParseString(`<svg><base href="x/"></svg>`, "file:///dir/")
	if err != nil {
		t.Fatal(err)
	}
	if doc.BaseURL != "file:///dir/" {
		t.Errorf("BaseURL = %q", doc.BaseURL)
	}
}

func TestParseBuildsMissingStructure(t *testing.T) {
	doc, err := ParseString("<p>text", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tagNames(doc.Elements()), " "); got != "html head body p" {
		t.Errorf("elements: %s", got)
	}
}

func TestParseRootAttributes(t *testing.T) {
	doc, err := ParseString("<html test=ok>", "")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := doc.Root.Lookup("test"); v != "ok" {
		t.Errorf("test = %q", v)
	}
}

func TestElementTree(t *testing.T) {
	doc, err := ParseString(`<body><div id=a class=" x  y
z "><p>1</p><!-- c --><p>2</p>text<p>3</p></div><section id=b></section>`, "")
	if err != nil {
		t.Fatal(err)
	}
	var div, p1, p2, p3, section *Element
	for _, e := range doc.Elements() {
		switch {
		case e.ID() == "a":
			div = e
		case e.ID() == "b":
			section = e
		case e.LocalName() == "p":
			switch e.ChildText() {
			case "1":
				p1 = e
			case "2":
				p2 = e
			case "3":
				p3 = e
			}
		}
	}
	if got := div.Classes(); strings.Join(got, ",") != "x,y,z" {
		t.Errorf("classes = %q", got)
	}
	if got := section.Classes(); len(got) != 0 {
		t.Errorf("classes = %q", got)
	}
	if div.Parent() != selector.Element(doc.Root.Children()[1]) {
		t.Error("div parent is not body")
	}
	if doc.Root.Parent() != nil {
		t.Error("root has a parent")
	}
	if p1.PreviousSibling() != nil {
		t.Error("first child has a previous sibling")
	}
	if p3.NextSibling() != nil {
		t.Error("last child has a next sibling")
	}
	if p1.NextSibling() != selector.Element(p2) || p3.PreviousSibling() != selector.Element(p2) {
		t.Error("comment or text counted as sibling")
	}
	if div.FirstChild() != selector.Element(p1) || p1.FirstChild() != nil {
		t.Error("wrong first child")
	}
	for i, p := range []*Element{p1, p2, p3} {
		if p.Index() != i {
			t.Errorf("p%d index = %d", i+1, p.Index())
		}
	}
	if section.Index() != 1 || doc.Root.Index() != 0 {
		t.Errorf("section index %d, root index %d", section.Index(), doc.Root.Index())
	}
	if doc.Element(div.Node()) != div || doc.Element(div.Node().FirstChild) != p1 {
		t.Error("lookup by node failed")
	}
	if doc.Element(div.Node().FirstChild.NextSibling.NextSibling.NextSibling) != nil {
		t.Error("text node mapped to an element")
	}
	if div.Namespace() != XHTMLNamespace || !div.InHTMLDocument() {
		t.Error("HTML element not in the XHTML namespace")
	}
}

func TestElementIsEmpty(t *testing.T) {
	doc, err := ParseString(`<body><p id=a></p><p id=b><!-- c --></p><p id=c> </p><p id=d><b></b></p><p id=e>x</p>`, "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"a": true, "b": true, "c": false, "d": false, "e": false}
	for _, e := range doc.Elements() {
		if id := e.ID(); id != "" && e.IsEmpty() != want[id] {
			t.Errorf("#%s IsEmpty = %v", id, e.IsEmpty())
		}
	}
}

func TestElementAttributes(t *testing.T) {
	doc, err := ParseString(`<body><A HREF="x" Data-X=1 id=u><svg viewBox="0 0 1 1" xml:lang=EN-us><use xlink:href="#a"></use></svg>`, "")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*Element{}
	for _, e := range doc.Elements() {
		byName[e.LocalName()] = e
	}
	a := byName["a"]
	if v, ok := a.Attr("", "href"); !ok || v != "x" {
		t.Errorf("href = %q, %v", v, ok)
	}
	if v, ok := a.Attr("", "data-x"); !ok || v != "1" {
		t.Errorf("data-x = %q, %v", v, ok)
	}
	if _, ok := a.Attr("", "HREF"); ok {
		t.Error("attribute names are not folded at lookup")
	}
	if _, ok := a.Attr(XLinkNamespace, "href"); ok {
		t.Error("href found in the xlink namespace")
	}
	svg := byName["svg"]
	if svg.Namespace() != SVGNamespace {
		t.Errorf("svg namespace = %q", svg.Namespace())
	}
	if v, ok := svg.Attr("", "viewBox"); !ok || v != "0 0 1 1" {
		t.Errorf("viewBox = %q, %v", v, ok)
	}
	if v, ok := svg.Attr(XMLNamespace, "lang"); !ok || v != "EN-us" {
		t.Errorf("xml:lang = %q, %v", v, ok)
	}
	use := byName["use"]
	if v, ok := use.Attr(XLinkNamespace, "href"); !ok || v != "#a" {
		t.Errorf("xlink:href = %q, %v", v, ok)
	}
	if _, ok := use.Attr("", "href"); ok {
		t.Error("xlink:href found without a namespace")
	}
}

func TestElementLanguage(t *testing.T) {
	tests := []struct {
		name string
		html string
		opts []Option
		want map[string]string
	}{
		{
			name: "attribute inherited",
			html: `<html lang=EN-US><body><p id=a>x</p><p id=b lang=Fr>y<span id=c>z</span></p>`,
			want: map[string]string{"a": "en-us", "b": "fr", "c": "fr"},
		},
		{
			name: "unknown",
			html: `<body><p id=a>x</p>`,
			want: map[string]string{"a": ""},
		},
		{
			name: "empty attribute resets",
			html: `<html lang=en><body><p id=a lang="">x</p>`,
			want: map[string]string{"a": ""},
		},
		{
			name: "xml:lang takes precedence in foreign content",
			html: `<html lang=en><body><svg id=a xml:lang=DE lang=it></svg>`,
			want: map[string]string{"a": "de"},
		},
		{
			name: "meta content-language",
			html: `<html><head><meta http-equiv=Content-Language content="Es-AR"></head><body><p id=a>x</p>`,
			want: map[string]string{"a": "es-ar"},
		},
		{
			name: "attribute beats meta",
			html: `<html lang=en><head><meta http-equiv=content-language content=es></head><body><p id=a>x</p>`,
			want: map[string]string{"a": "en"},
		},
		{
			name: "meta with several languages is not a language",
			html: `<html><head><meta http-equiv=content-language content="en, fr"></head><body><p id=a>x</p>`,
			want: map[string]string{"a": ""},
		},
		{
			name: "last meta wins",
			html: `<html><head><meta http-equiv=content-language content=en><meta http-equiv=content-language content=fr></head><body><p id=a>x</p>`,
			want: map[string]string{"a": "fr"},
		},
		{
			name: "transport language",
			html: `<body><p id=a>x</p>`,
			opts: []Option{WithContentLanguage("PT-br")},
			want: map[string]string{"a": "pt-br"},
		},
		{
			name: "meta beats transport",
			html: `<html><head><meta http-equiv=content-language content=en></head><body><p id=a>x</p>`,
			opts: []Option{WithContentLanguage("pt")},
			want: map[string]string{"a": "en"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseString(tt.html, "", tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range doc.Elements() {
				if want, ok := tt.want[e.ID()]; ok && e.ID() != "" && e.Lang() != want {
					t.Errorf("#%s lang = %q, want %q", e.ID(), e.Lang(), want)
				}
			}
		})
	}
}

func TestElementsMatchSelectors(t *testing.T) {
	doc, err := ParseString(`<body><ul><li class=a>1<li class="a b">2<li lang=fr>3</ul><p></p>`, "")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		css  string
		want []string
	}{
		{"li", []string{"1", "2", "3"}},
		{"LI.a", []string{"1", "2"}},
		{".a.b", []string{"2"}},
		{"li + li", []string{"2", "3"}},
		{"li:nth-child(2) ~ li", []string{"3"}},
		{"ul > li:last-child", []string{"3"}},
		{":lang(fr)", []string{"3"}},
		{"ul > :first-child", []string{"1"}},
		{"body > :empty", []string{""}},
		{"[CLASS~=b]", []string{"2"}},
	}
	for _, tt := range tests {
		sels, err := selector.CompileString(tt.css, nil)
		if err != nil {
			t.Fatalf("%s: %v", tt.css, err)
		}
		var got []string
		for _, e := range doc.Elements() {
			for i := range sels {
				if sels[i].Match(e) {
					got = append(got, e.ChildText())
					break
				}
			}
		}
		if strings.Join(got, ",") != strings.Join(tt.want, ",") {
			t.Errorf("%s matched %q, want %q", tt.css, got, tt.want)
		}
	}
}

func TestElementChildText(t *testing.T) {
	doc, err := ParseString(`<body><p>a<b>inner</b>c<!-- x -->d</p>`, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range doc.Elements() {
		if e.LocalName() == "p" {
			if got := e.ChildText(); got != "acd" {
				t.Errorf("ChildText = %q", got)
			}
		}
	}
}

func TestHasLinkType(t *testing.T) {
	tests := []struct {
		rel  string
		want bool
	}{
		{"attachment", true},
		{"ATTACHMENT", true},
		{"  stylesheet\tAttachment  ", true},
		{"attachments", false},
		{"not_attachment", false},
		{"", false},
	}
	for _, tt := range tests {
		doc, err := ParseString(`<link rel="`+tt.rel+`">`, "")
		if err != nil {
			t.Fatal(err)
		}
		var link *Element
		for _, e := range doc.Elements() {
			if e.LocalName() == "link" {
				link = e
			}
		}
		if got := link.HasLinkType("attachment"); got != tt.want {
			t.Errorf("rel=%q: %v, want %v", tt.rel, got, tt.want)
		}
	}
}
