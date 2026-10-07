package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/targets"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
)

func TestBoxesImplementTargetInterfaces(t *testing.T) {
	var box boxes.Box = boxes.NewBlockBox("div", plainStyle(), nil, nil)
	if _, ok := box.(targets.Source); !ok {
		t.Error("boxes are not target sources")
	}
	if _, ok := box.(targets.Target); !ok {
		t.Error("boxes are not target boxes")
	}
	if _, ok := box.(targets.Linked); !ok {
		t.Error("boxes are not linked boxes")
	}
}

func TestSourceReadsTheAttributesOfTheElement(t *testing.T) {
	s := newStyled(t, `<a href="#x" title=""></a>`)
	b := boxes.NewInlineBox("a", s.style("a"), s.el("a"), nil)
	if v, ok := b.Attr("href"); !ok || v != "#x" {
		t.Errorf("href = %q, %v", v, ok)
	}
	if v, ok := b.Attr("title"); !ok || v != "" {
		t.Errorf("title = %q, %v", v, ok)
	}
	if _, ok := b.Attr("rel"); ok {
		t.Error("found a missing attribute")
	}
	if b.BaseURL() != csstest.BaseURL(t) {
		t.Errorf("BaseURL = %q", b.BaseURL())
	}
	if _, ok := boxes.NewBlockBox("", plainStyle(), nil, nil).Attr("x"); ok {
		t.Error("a box without element has attributes")
	}
}

func TestCollectorStoresCounterValuesOnTheTargetBox(t *testing.T) {
	c := targets.NewCollector()
	c.CollectAnchor("top")
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	c.StoreTarget("top", targets.CounterValues{"n": {1, 2}}, box)

	got := box.CachedCounterValues()
	if len(got["n"]) != 2 || got["n"][1] != 2 {
		t.Errorf("counter values = %v", got)
	}
	if c.Target("top").TargetBox != targets.Target(box) {
		t.Error("the collector does not hold the box")
	}

	other := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	other.SetCachedCounterValues(targets.CounterValues{"m": {7}})
	c.CollectAnchor("other")
	c.StoreTarget("other", targets.CounterValues{"n": {9}}, other)
	if _, ok := other.CachedCounterValues()["n"]; ok {
		t.Error("counter values replaced the ones the box already had")
	}
}

func TestMissingCountersLinkTheBoxToItself(t *testing.T) {
	c := targets.NewCollector()
	box := boxes.NewBlockBox("div", plainStyle(), nil, nil)
	if box.MissingLink() != nil {
		t.Fatal("a new box has a missing link")
	}
	c.CollectMissingCounters(box, "content", func(targets.CounterValues) {}, map[string]bool{"page": true}, nil)
	if got, ok := box.MissingLink().(boxes.Box); !ok || got != boxes.Box(box) {
		t.Errorf("MissingLink = %v", box.MissingLink())
	}
	if c.CounterItem(box, "content") == nil {
		t.Error("the box was not collected")
	}
	box.SetMissingLink(nil)
	if box.MissingLink() != nil || box.MissingLinkBox != nil {
		t.Error("clearing the link failed")
	}
}

// childAt follows the children at the given indexes from box.
func childAt(t *testing.T, box boxes.Box, indexes ...int) boxes.Box {
	t.Helper()
	for _, i := range indexes {
		children := boxes.ChildrenOf(box)
		if i >= len(children) {
			t.Fatalf("%v has %d children, no %d", box, len(children), i)
		}
		box = children[i]
	}
	return box
}

func textOf(t *testing.T, box boxes.Box) string {
	t.Helper()
	text, ok := box.(*boxes.TextBox)
	if !ok {
		t.Fatalf("%v is not a text box", box)
	}
	return text.Text
}

func TestTargetCounter(t *testing.T) {
	testutil.AssertNoLogs(t)
	divs := body(t, boxtest.ParseAll(t, `
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
    `))
	for i, want := range []string{"4", "test 1", "iv", "3"} {
		if got := textAt(t, divs[i], 3); got != want {
			t.Errorf("div %d: %q, want %q", i+1, got, want)
		}
	}
}

func TestTargetCounterOfAttribute(t *testing.T) {
	testutil.AssertNoLogs(t)
	divs := body(t, boxtest.ParseAll(t, `
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
    `))
	for i, want := range []string{"4", "1", "2", "c"} {
		if got := textAt(t, divs[i], 3); got != want {
			t.Errorf("div %d: %q, want %q", i+1, got, want)
		}
	}
}

func TestTargetCounters(t *testing.T) {
	testutil.AssertNoLogs(t)
	divs := body(t, boxtest.ParseAll(t, `
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
    `))
	tests := []struct {
		box  boxes.Box
		want string
	}{
		{childAt(t, divs[0], 1), "4.2"},
		{childAt(t, divs[1], 0), "3"},
		{divs[2], "b.a"},
		{childAt(t, divs[3], 1), "12"},
	}
	for i, tt := range tests {
		if got := textAt(t, tt.box, 3); got != tt.want {
			t.Errorf("div %d: %q, want %q", i+1, got, tt.want)
		}
	}
}

func TestTargetText(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := body(t, boxtest.ParseAll(t, `
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
    `))
	if len(children) != 8 {
		t.Fatalf("body has %d children, want 8", len(children))
	}
	if got := textAt(t, children[0], 3); got != "test 4 Chapter 4" {
		t.Errorf("link 1: %q", got)
	}
	if got := textAt(t, children[2], 3); got != "wow" {
		t.Errorf("link 2: %q", got)
	}
	if got := boxes.ChildrenOf(childAt(t, children[5], 0, 0)); len(got) != 0 {
		t.Errorf("link 3 has content %v", got)
	}
	if got := textAt(t, children[7], 3); got != "1" {
		t.Errorf("link 4: %q", got)
	}
}

func TestTargetTextKeepsWhiteSpaceAroundTheLink(t *testing.T) {
	testutil.AssertNoLogs(t)
	children := body(t, boxtest.ParseAll(t, `
      <style>
        a::before { content: target-text(attr(href)) }
      </style>
      <p>before <a href="#ref"></a> after</p>
      <h2 id="ref">text</h2>
    `))
	line := onlyChild(t, children[0])
	parts := boxes.ChildrenOf(line)
	if len(parts) != 3 {
		t.Fatalf("line has %d children, want 3", len(parts))
	}
	if got := textOf(t, parts[0]); got != "before " {
		t.Errorf("before: %q", got)
	}
	if got := textOf(t, parts[2]); got != " after" {
		t.Errorf("after: %q", got)
	}
	if got := textAt(t, parts[1], 2); got != "text" {
		t.Errorf("link: %q", got)
	}
}
