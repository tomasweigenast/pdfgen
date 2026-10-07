package html

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

const testBase = "http://example.org/dir/page.html"

func find(t *testing.T, doc *Document, tag string) *Element {
	t.Helper()
	for _, e := range doc.Elements() {
		if e.LocalName() == tag {
			return e
		}
	}
	t.Fatalf("no <%s> element", tag)
	return nil
}

func parseWithBase(t *testing.T, markup string) *Document {
	t.Helper()
	doc, err := ParseString(markup, testBase)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestElementKind(t *testing.T) {
	doc := parseWithBase(t, `<body><img><embed><object></object><a></a><table><colgroup><col></colgroup></table><svg></svg><p></p><svg><a></a></svg>`)
	want := map[string]Kind{
		"img": KindImg, "embed": KindEmbed, "object": KindObject, "colgroup": KindColgroup,
		"col": KindCol, "svg": KindSVG, "p": KindNone, "html": KindNone,
	}
	for _, e := range doc.Elements() {
		if e.LocalName() == "a" {
			wantA := KindLink
			if e.Namespace() == SVGNamespace {
				wantA = KindNone
			}
			if e.Kind() != wantA {
				t.Errorf("<a> in %s: kind %d", e.Namespace(), e.Kind())
			}
			continue
		}
		if k, ok := want[e.LocalName()]; ok && e.Kind() != k {
			t.Errorf("<%s> kind = %d, want %d", e.LocalName(), e.Kind(), k)
		}
	}
}

func TestImg(t *testing.T) {
	tests := []struct {
		markup string
		want   ImgInfo
	}{
		{`<img src="pattern.png" alt="A pattern">`, ImgInfo{"http://example.org/dir/pattern.png", "A pattern"}},
		{`<img src="  pattern.png ">`, ImgInfo{"http://example.org/dir/pattern.png", ""}},
		{`<img src="data:image/svg+xml,<svg></svg>" alt=text>`, ImgInfo{"data:image/svg+xml,<svg></svg>", "text"}},
		{`<img src="" alt="fallback">`, ImgInfo{"", "fallback"}},
		{`<img alt="fallback">`, ImgInfo{"", "fallback"}},
		{`<img srcset="a.png 2x">`, ImgInfo{}},
		{`<img src="é.png">`, ImgInfo{"http://example.org/dir/%C3%A9.png", ""}},
	}
	for _, tt := range tests {
		doc := parseWithBase(t, tt.markup)
		if got := find(t, doc, "img").Img(); got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.markup, got, tt.want)
		}
	}
}

func TestImgRelativeWithoutBase(t *testing.T) {
	logs := testutil.CaptureLogs(t)
	doc, err := ParseString(`<img src="a.png" alt=x>`, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := find(t, doc, "img").Img(); got != (ImgInfo{"", "x"}) {
		t.Errorf("got %+v", got)
	}
	if len(logs.Lines()) != 1 {
		t.Errorf("logs: %q", logs.Lines())
	}
}

func TestEmbedAndObject(t *testing.T) {
	doc := parseWithBase(t, `<body>
		<embed src="a.svg" type=" image/svg+xml ">
		<embed>
		<object data="b.png"><p>fallback</p></object>
		<object data="  " type=image/png></object>`)
	var embeds, objects []*Element
	for _, e := range doc.Elements() {
		switch e.LocalName() {
		case "embed":
			embeds = append(embeds, e)
		case "object":
			objects = append(objects, e)
		}
	}
	if info, ok := embeds[0].Embed(); !ok || info != (ResourceInfo{"http://example.org/dir/a.svg", "image/svg+xml"}) {
		t.Errorf("embed: %+v, %v", info, ok)
	}
	if _, ok := embeds[1].Embed(); ok {
		t.Error("embed without src has a resource")
	}
	if info, ok := objects[0].Object(); !ok || info != (ResourceInfo{"http://example.org/dir/b.png", ""}) {
		t.Errorf("object: %+v, %v", info, ok)
	}
	if _, ok := objects[1].Object(); ok {
		t.Error("object with a blank data attribute has a resource")
	}
}

func TestLink(t *testing.T) {
	tests := []struct {
		name, markup, base string
		want               *urls.Link
	}{
		{"relative", `<a href="x.html#y">`, testBase, &urls.Link{Kind: urls.LinkExternal, Target: "http://example.org/dir/x.html#y"}},
		{"absolute", `<a href=" https://other.org/a b ">`, testBase, &urls.Link{Kind: urls.LinkExternal, Target: "https://other.org/a%20b"}},
		{"fragment", `<a href="#sec%201">`, testBase, &urls.Link{Kind: urls.LinkInternal, Target: "sec 1"}},
		{"fragment without base", `<a href="#top">`, "", &urls.Link{Kind: urls.LinkInternal, Target: "top"}},
		{"same document", `<a href="page.html#top">`, testBase, &urls.Link{Kind: urls.LinkInternal, Target: "top"}},
		{"relative without base", `<a href="x.html">`, "", &urls.Link{Kind: urls.LinkExternal, Target: "x.html"}},
		{"no href", `<a name=x>`, testBase, nil},
		{"empty href", `<a href="">`, testBase, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseString(tt.markup, tt.base)
			if err != nil {
				t.Fatal(err)
			}
			got := find(t, doc, "a").Link()
			if (got == nil) != (tt.want == nil) || got != nil && *got != *tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpans(t *testing.T) {
	doc := parseWithBase(t, `<table><colgroup span=3></colgroup><colgroup span=0><col></colgroup><colgroup span=-2></colgroup><colgroup></colgroup><colgroup span=x></colgroup>
		<tr><td colspan=2 rowspan=0><td colspan=0 rowspan=-1><td colspan=abc rowspan="3 rows"><td></table>`)
	var groups, cells []*Element
	for _, e := range doc.Elements() {
		switch e.LocalName() {
		case "colgroup":
			groups = append(groups, e)
		case "td":
			cells = append(cells, e)
		}
	}
	for i, want := range []int{3, 1, 1, 1, 1} {
		if got := groups[i].Span(); got != want {
			t.Errorf("colgroup %d span = %d, want %d", i, got, want)
		}
	}
	if groups[0].HasChildElement("col") || !groups[1].HasChildElement("col") {
		t.Error("HasChildElement wrong for colgroups")
	}
	colspans := []int{2, 1, 1, 1}
	rowspans := []int{0, 0, 3, 1}
	for i, c := range cells {
		if c.ColSpan() != colspans[i] || c.RowSpan() != rowspans[i] {
			t.Errorf("cell %d: colspan %d rowspan %d, want %d and %d", i, c.ColSpan(), c.RowSpan(), colspans[i], rowspans[i])
		}
	}
}

func TestColSpan(t *testing.T) {
	doc := parseWithBase(t, `<table><colgroup><col span=4><col></colgroup></table>`)
	var cols []*Element
	for _, e := range doc.Elements() {
		if e.LocalName() == "col" {
			cols = append(cols, e)
		}
	}
	if cols[0].Span() != 4 || cols[1].Span() != 1 {
		t.Errorf("spans: %d, %d", cols[0].Span(), cols[1].Span())
	}
}

func TestSVGMarkup(t *testing.T) {
	doc := parseWithBase(t, `<body><p>x</p><svg viewBox="0 0 10 10" width=10 xmlns:xlink="http://www.w3.org/1999/xlink">
		<!-- gone -->
		<defs><linearGradient id=g><stop offset=0></linearGradient></defs>
		<use xlink:href="#g" data-note='a "quoted" <b> & more'/>
		<style>rect > .a { fill: red }</style>
		<text xml:lang=en>1 &lt; 2 &amp; 3</text>
		<foreignObject><p class=x>html<br></p></foreignObject>
		</svg>`)
	svg := find(t, doc, "svg")
	markup := svg.SVGMarkup()

	d := xml.NewDecoder(strings.NewReader(markup))
	seen := map[string]string{}
	attrs := map[string]string{}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("not well-formed XML: %v\n%s", err, markup)
		}
		switch tk := tok.(type) {
		case xml.StartElement:
			seen[tk.Name.Local] = tk.Name.Space
			for _, a := range tk.Attr {
				attrs[tk.Name.Local+"@"+a.Name.Space+"|"+a.Name.Local] = a.Value
			}
		case xml.Comment:
			t.Error("comment kept")
		}
	}
	for tag, ns := range map[string]string{
		"svg": SVGNamespace, "defs": SVGNamespace, "use": SVGNamespace, "stop": SVGNamespace,
		"linearGradient": SVGNamespace, "foreignObject": SVGNamespace,
		"p": XHTMLNamespace, "br": XHTMLNamespace,
	} {
		if seen[tag] != ns {
			t.Errorf("<%s> in namespace %q, want %q", tag, seen[tag], ns)
		}
	}
	if got := attrs["use@"+XLinkNamespace+"|href"]; got != "#g" {
		t.Errorf("xlink:href = %q", got)
	}
	if got := attrs["use@|data-note"]; got != `a "quoted" <b> & more` {
		t.Errorf("data-note = %q", got)
	}
	if got := attrs["svg@|viewBox"]; got != "0 0 10 10" {
		t.Errorf("viewBox = %q", got)
	}
	if got := attrs["text@"+XMLNamespace+"|lang"]; got != "en" {
		t.Errorf("xml:lang = %q", got)
	}
	if !strings.Contains(markup, "1 &lt; 2 &amp; 3") || !strings.Contains(markup, "rect &gt; .a") {
		t.Errorf("text not escaped: %s", markup)
	}
	if strings.Count(markup, `xmlns="`+SVGNamespace+`"`) != 1 {
		t.Errorf("SVG namespace declared more or less than once: %s", markup)
	}
}

func TestSVGMarkupDropsInvalidAttributeNames(t *testing.T) {
	doc := parseWithBase(t, `<svg a@b="1" ok="2" 3x="4"></svg>`)
	markup := find(t, doc, "svg").SVGMarkup()
	if err := xml.Unmarshal([]byte(markup), new(struct{ XMLName xml.Name })); err != nil {
		t.Fatalf("%v\n%s", err, markup)
	}
	if !strings.Contains(markup, `ok="2"`) || strings.Contains(markup, "a@b") || strings.Contains(markup, "3x") {
		t.Errorf("markup = %s", markup)
	}
}
