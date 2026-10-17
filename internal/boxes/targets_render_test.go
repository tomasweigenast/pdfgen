package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// renderBody lays out a document that fits in one page and returns the
// children of its body.
func renderBody(t *testing.T, source string) []boxes.Box {
	t.Helper()
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, source)
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	html := boxes.ChildrenOf(pages[0])
	if len(html) != 1 {
		t.Fatalf("the page has %d children, want the html", len(html))
	}
	body := boxes.ChildrenOf(html[0])
	if len(body) != 1 {
		t.Fatalf("html has %d children, want the body", len(body))
	}
	return boxes.ChildrenOf(body[0])
}

// descend follows child indexes from a box.
func descend(t *testing.T, box boxes.Box, path ...int) boxes.Box {
	t.Helper()
	for _, i := range path {
		if ph, ok := box.(*boxes.AbsolutePlaceholder); ok {
			box = ph.Box
		}
		children := boxes.ChildrenOf(box)
		if i >= len(children) {
			t.Fatalf("%v has no child %d: %v", box, i, children)
		}
		box = children[i]
	}
	return box
}

func expectText(t *testing.T, box boxes.Box, want string) {
	t.Helper()
	text, ok := box.(*boxes.TextBox)
	if !ok {
		t.Fatalf("%v is not a text box", box)
	}
	if text.Text != want {
		t.Errorf("text = %q, want %q", text.Text, want)
	}
}

func TestTargetCounterOfPagesAfterLayout(t *testing.T) {
	list := renderBody(t, `
      <style>
        div:first-child { counter-reset: div }
        div { counter-increment: div }
        #id1::before { content: target-counter('#id4', div) }
        #id2::before { content: 'test ' target-counter('#id1', div) }
        #id3::before { content: target-counter(url(#id4), div, lower-roman) }
        #id4::before { content: target-counter('#id3', div) }
      </style>
      <body>
        <div id="id1"></div>
        <div id="id2"></div>
        <div id="id3"></div>
        <div id="id4"></div>
    `)
	if len(list) != 4 {
		t.Fatalf("body has %d children, want 4", len(list))
	}
	for i, want := range []string{"4", "test 1", "iv", "3"} {
		expectText(t, descend(t, list[i], 0, 0, 0), want)
	}
}

func TestTargetCounterOfAttributeAfterLayout(t *testing.T) {
	list := renderBody(t, `
      <style>
        div:first-child { counter-reset: div }
        div { counter-increment: div }
        div::before { content: target-counter(attr(data-count), div) }
        #id2::before { content: target-counter(attr(data-count url), div) }
        #id4::before {
          content: target-counter(attr(data-count), div, lower-alpha) }
      </style>
      <body>
        <div id="id1" data-count="#id4"></div>
        <div id="id2" data-count="#id1"></div>
        <div id="id3" data-count="#id2"></div>
        <div id="id4" data-count="#id3"></div>
    `)
	if len(list) != 4 {
		t.Fatalf("body has %d children, want 4", len(list))
	}
	for i, want := range []string{"4", "1", "2", "c"} {
		expectText(t, descend(t, list[i], 0, 0, 0), want)
	}
}

func TestTargetCountersAfterLayout(t *testing.T) {
	list := renderBody(t, `
      <style>
        div:first-child { counter-reset: div }
        div { counter-increment: div }
        #id1-2::before { content: target-counters('#id4-2', div, '.') }
        #id2-1::before { content: target-counters(url(#id3), div, '++') }
        #id3::before {
          content: target-counters('#id2-1', div, '.', lower-alpha) }
        #id4-2::before {
          content: target-counters(attr(data-count, url), div, '') }
      </style>
      <body>
        <div id="id1"><div></div><div id="id1-2"></div></div>
        <div id="id2"><div id="id2-1"></div><div></div></div>
        <div id="id3"></div>
        <div id="id4">
          <div></div><div id="id4-2" data-count="#id1-2"></div>
        </div>
    `)
	if len(list) != 4 {
		t.Fatalf("body has %d children, want 4", len(list))
	}
	expectText(t, descend(t, list[0], 1, 0, 0, 0), "4.2")
	expectText(t, descend(t, list[1], 0, 0, 0, 0), "3")
	expectText(t, descend(t, list[2], 0, 0, 0), "b.a")
	expectText(t, descend(t, list[3], 1, 0, 0, 0), "12")
}

func TestTargetTextAfterLayout(t *testing.T) {
	list := renderBody(t, `
      <style>
        a { display: block; color: red }
        div:first-child { counter-reset: div }
        div { counter-increment: div }
        #id2::before { content: 'wow' }
        #link1::before { content: 'test ' target-text('#id4') }
        #link2::before { content: target-text(attr(data-count, url), before) }
        #link3::before { content: target-text('#id3', after) }
        #link4::before { content: target-text(url(#id1), first-letter) }
      </style>
      <body>
        <a id="link1"></a>
        <div id="id1">1 Chapter 1</div>
        <a id="link2" data-count="#id2"></a>
        <div id="id2">2 Chapter 2</div>
        <div id="id3">3 Chapter 3</div>
        <a id="link3"></a>
        <div id="id4">4 Chapter 4</div>
        <a id="link4"></a>
    `)
	if len(list) != 8 {
		t.Fatalf("body has %d children, want 8", len(list))
	}
	a1, a2, a3, a4 := list[0], list[2], list[5], list[7]
	expectText(t, descend(t, a1, 0, 0, 0), "test 4 Chapter 4")
	expectText(t, descend(t, a2, 0, 0, 0), "wow")
	if n := len(boxes.ChildrenOf(descend(t, a3, 0, 0))); n != 0 {
		t.Errorf("the third link has %d boxes, want none", n)
	}
	expectText(t, descend(t, a4, 0, 0, 0), "1")
}

func TestTargetTextKeepsWhiteSpaceAroundTheLinkAfterLayout(t *testing.T) {
	list := renderBody(t, `
      <style>
        a::before { content: target-text(attr(href)) }
      </style>
      <p>before <a href="#ref"></a> after</p>
      <h2 id="ref">text</h2>
    `)
	if len(list) != 2 {
		t.Fatalf("body has %d children, want 2", len(list))
	}
	line := descend(t, list[0], 0)
	parts := boxes.ChildrenOf(line)
	if len(parts) != 3 {
		t.Fatalf("the line has %d children, want 3", len(parts))
	}
	expectText(t, parts[0], "before ")
	expectText(t, parts[2], " after")
	expectText(t, descend(t, parts[1], 0, 0), "text")
}

func TestTargetCounterOfPageInFloat(t *testing.T) {
	list := renderBody(t, `
      <style>
        a::after {
          content: target-counter('#h', page);
          float: right;
        }
      </style>
      <div><a id="span">link</a></div>
      <h1 id="h">abc</h1>
    `)
	if len(list) != 2 {
		t.Fatalf("body has %d children, want 2", len(list))
	}
	inline := descend(t, list[0], 0, 0)
	parts := boxes.ChildrenOf(inline)
	if len(parts) != 2 {
		t.Fatalf("the link has %d children, want 2", len(parts))
	}
	expectText(t, parts[0], "link")
	expectText(t, descend(t, parts[1], 0, 0), "1")
}

func TestTargetCounterOfPageInAbsoluteBox(t *testing.T) {
	list := renderBody(t, `
      <style>
        a::after {
          content: target-counter('#h', page);
        }
        div {
          position: absolute;
        }
      </style>
      <div><a id="span">link</a></div>
      <h1 id="h">abc</h1>
    `)
	if len(list) != 2 {
		t.Fatalf("body has %d children, want 2", len(list))
	}
	parts := boxes.ChildrenOf(descend(t, list[0], 0, 0))
	if len(parts) != 2 {
		t.Fatalf("the link has %d children, want 2", len(parts))
	}
	expectText(t, parts[0], "link")
	expectText(t, descend(t, parts[1], 0), "1")
}

func TestTargetCounterOfPageInNestedAbsoluteBox(t *testing.T) {
	list := renderBody(t, `
      <style>
        a::after {
          content: target-counter('#h', page);
        }
        section {
          position: relative;
        }
        div {
          position: absolute;
        }
      </style>
      <section><div><a id="span">link</a></div></section>
      <h1 id="h">abc</h1>
    `)
	if len(list) != 2 {
		t.Fatalf("body has %d children, want 2", len(list))
	}
	parts := boxes.ChildrenOf(descend(t, list[0], 0, 0, 0))
	if len(parts) != 2 {
		t.Fatalf("the link has %d children, want 2", len(parts))
	}
	expectText(t, parts[0], "link")
	expectText(t, descend(t, parts[1], 0), "1")
}

func TestTargetCounterOfPageIsKnownOnEarlierPages(t *testing.T) {
	// A link whose target is on a later page shows the page of the target,
	// once pagination knows it.
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page { size: 100px 60px }
        a::after { content: " " target-counter(attr(href), page) }
        h1 { break-before: page; margin: 0 }
      </style>
      <a href="#a">A</a><br><a href="#b">B</a><br><a href="#c">C</a>
      <h1 id="a">AA</h1><h1 id="b">BB</h1>
      <p style="break-before: page">pp</p><h1 id="c">CC</h1>
    `)
	first := pages[0]
	var got []string
	for box := range boxes.Descendants(first, true) {
		if box.Kind() == boxes.KindInline && box.Base().ElementTag == "a::after" {
			if text, ok := descend(t, box, 0).(*boxes.TextBox); ok {
				got = append(got, text.Text)
			}
		}
	}
	want := []string{" 2", " 3", " 5"}
	if len(got) != len(want) {
		t.Fatalf("got %d links, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link %d shows %q, want %q", i+1, got[i], want[i])
		}
	}
}
