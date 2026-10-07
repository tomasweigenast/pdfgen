package boxes_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// beforeTrees returns the trees of paragraphs that only have a ::before
// with the given texts.
func beforeTrees(texts ...string) []boxtest.Node {
	var nodes []boxtest.Node
	for _, text := range texts {
		nodes = append(nodes, P("p", "Block",
			P("p", "Line",
				P("p::before", "Inline", T("p::before", text)))))
	}
	return nodes
}

// markerTree is the tree of a list item with an inside marker.
func markerTree(tag, text string, children ...boxtest.Node) boxtest.Node {
	if len(children) == 0 {
		return P(tag, "Block",
			P(tag, "Line",
				P(tag+"::marker", "Inline", T(tag+"::marker", text))))
	}
	return P(tag, "Block", append([]boxtest.Node{
		P(tag, "Block",
			P(tag, "Line",
				P(tag+"::marker", "Inline", T(tag+"::marker", text))))}, children...)...)
}

// body returns the children of the body box of a document tree.
func body(t *testing.T, root boxes.Box) []boxes.Box {
	t.Helper()
	return boxes.ChildrenOf(onlyChild(t, root))
}

// textAt follows the first child of each box from box down to a text box
// and returns its text.
func textAt(t *testing.T, box boxes.Box, depth int) string {
	t.Helper()
	for range depth {
		children := boxes.ChildrenOf(box)
		if len(children) == 0 {
			t.Fatalf("%v has no children", box)
		}
		box = children[0]
	}
	text, ok := box.(*boxes.TextBox)
	if !ok {
		t.Fatalf("%v is not a text box", box)
	}
	return text.Text
}

func TestCountersAreResetAndIncremented(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p { counter-increment: p 2 }
        p:before { content: counter(p); }
        p:nth-child(1) { counter-increment: none; }
        p:nth-child(2) { counter-increment: p; }
      </style>
      <p></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p 117 p"></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p -13"></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p 42"></p>
      <p></p>
      <p></p>`), beforeTrees(strings.Fields("0 1 3  2 4 6  -11 -9 -7  44 46 48")...))
}

func TestListItemsIncrementTheListItemCounter(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <ol style="list-style-position: inside">
        <li></li>
        <li></li>
        <li></li>
        <li><ol>
          <li></li>
          <li style="counter-increment: none"></li>
          <li></li>
        </ol></li>
        <li></li>
      </ol>`), []boxtest.Node{
		P("ol", "Block",
			markerTree("li", "1. "),
			markerTree("li", "2. "),
			markerTree("li", "3. "),
			markerTree("li", "4. ",
				P("ol", "Block",
					markerTree("li", "1. "),
					markerTree("li", "1. "),
					markerTree("li", "2. "))),
			markerTree("li", "5. ")),
	})
}

func TestListItemCounterReset(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p { display: list-item; list-style: inside decimal }
      </style>
      <div>
        <p></p>
        <p></p>
        <p style="counter-reset: list-item 7 list-item -56"></p>
      </div>
      <p></p>`), []boxtest.Node{
		P("div", "Block",
			markerTree("p", "1. "),
			markerTree("p", "2. "),
			markerTree("p", "-55. ")),
		markerTree("p", "1. "),
	})
}

func TestNestedCountersInPseudoElements(t *testing.T) {
	testutil.AssertNoLogs(t)
	h1 := func(text string) boxtest.Node {
		return P("h1", "Block",
			P("h1", "Line",
				P("h1::before", "Inline", T("h1::before", text))))
	}
	section := func(children ...boxtest.Node) boxtest.Node {
		return P("section", "Block", append([]boxtest.Node{
			P("section", "Block",
				P("section", "Line",
					P("section::before", "Inline")))}, children...)...)
	}
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        section:before { counter-reset: h; content: '' }
        h1:before { counter-increment: h; content: counters(h, '.') }
      </style>
      <body>
        <section><h1></h1>
          <h1></h1>
          <section><h1></h1>
            <h1></h1>
          </section>
          <h1></h1>
        </section>
      </body>`), []boxtest.Node{
		section(h1("1"), h1("2"), section(h1("2.1"), h1("2.2")), h1("3")),
	})
}

func TestCounterScopeEndsWithTheParent(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p:before { content: counter(c) }
      </style>
      <div>
        <span style="counter-reset: c">
          Scope created now, deleted after the div
        </span>
      </div>
      <p></p>`), []boxtest.Node{
		P("div", "Block",
			P("div", "Line",
				P("span", "Inline",
					T("span", "Scope created now, deleted after the div ")))),
		beforeTrees("0")[0],
	})
}

func TestCounterIncrementReplacesTheListItemIncrement(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <p style="counter-increment: c;
                display: list-item; list-style: inside decimal">`), []boxtest.Node{
		markerTree("p", "0. "),
	})
}

func TestCounterNamesAreCaseSensitive(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p { counter-increment: p 2 }
        p:before { content: counter(p) '.' counter(P); }
      </style>
      <p></p>
      <p style="counter-increment: P 3"></p>
      <p></p>`), beforeTrees("2.0", "2.3", "4.3"))
}

func TestListItemPseudoElementsHaveMarkers(t *testing.T) {
	testutil.AssertNoLogs(t)
	p := P("p", "Block",
		P("p::before", "Block",
			P("p::marker", "Block",
				P("p::marker", "Line", T("p::marker", "• "))),
			P("p::before", "Block",
				P("p::before", "Line", T("p::before", "a")))))
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p:before { content: 'a'; display: list-item }
      </style>
      <p></p>
      <p></p>`), []boxtest.Node{p, p})
}

func TestCounterStylesInContent(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        body { --var: 'Counter'; counter-reset: p -12 }
        p { counter-increment: p }
        p:nth-child(1):before { content: '-' counter(p, none) '-'; }
        p:nth-child(2):before { content: counter(p, disc); }
        p:nth-child(3):before { content: counter(p, circle); }
        p:nth-child(4):before { content: counter(p, square); }
        p:nth-child(5):before { content: counter(p); }
        p:nth-child(6):before { content: var(--var) ':' counter(p); }
        p:nth-child(7):before { content: counter(p) ':' var(--var); }
      </style>
      <p></p>
      <p></p>
      <p></p>
      <p></p>
      <p></p>
      <p></p>
      <p></p>
    `), beforeTrees(strings.Fields("--  •  ◦  ▪  -7 Counter:-6 -5:Counter")...))
}

func TestDecimalLeadingZeroInContent(t *testing.T) {
	testutil.AssertNoLogs(t)
	boxtest.AssertTree(t, boxtest.ParseAll(t, `
      <style>
        p { counter-increment: p }
        p::before { content: counter(p, decimal-leading-zero); }
      </style>
      <p style="counter-reset: p -1987"></p>
      <p></p>
      <p style="counter-reset: p -12"></p>
      <p></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p -2"></p>
      <p></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p 8"></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p 98"></p>
      <p></p>
      <p></p>
      <p style="counter-reset: p 4134"></p>
      <p></p>
    `), beforeTrees(strings.Fields(`-1986 -1985  -11 -10 -9 -8  -1 00 01 02  09 10 11
                            99 100 101  4135 4136`)...))
}

// listMarkers returns the texts of the outside markers of the items of the
// only list of a document.
func listMarkers(t *testing.T, root boxes.Box) []string {
	t.Helper()
	list := body(t, root)
	if len(list) != 1 {
		t.Fatalf("body has %d children, want 1", len(list))
	}
	var texts []string
	for _, li := range boxes.ChildrenOf(list[0]) {
		texts = append(texts, textAt(t, li, 3))
	}
	return texts
}

func TestCounterSymbols(t *testing.T) {
	tests := []struct {
		arguments string
		values    []string
	}{
		{`cyclic "a" "b" "c"`, []string{"a ", "b ", "c ", "a "}},
		{`symbolic "a" "b"`, []string{"a ", "b ", "aa ", "bb "}},
		{`"a" "b"`, []string{"a ", "b ", "aa ", "bb "}},
		{`alphabetic "a" "b"`, []string{"a ", "b ", "aa ", "ab "}},
		{`fixed "a" "b"`, []string{"a ", "b ", "3 ", "4 "}},
		{`numeric "0" "1" "2"`, []string{"1 ", "2 ", "10 ", "11 "}},
	}
	for _, tt := range tests {
		t.Run(tt.arguments, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			got := listMarkers(t, boxtest.ParseAll(t, `
              <style>
                ol { list-style-type: symbols(`+tt.arguments+`) }
              </style>
              <ol>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
              </ol>
            `))
			if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", tt.values) {
				t.Errorf("markers = %q, want %q", got, tt.values)
			}
		})
	}
}

func TestListStyleTypes(t *testing.T) {
	tests := []struct {
		styleType string
		values    []string
	}{
		{"decimal", []string{"1. ", "2. ", "3. ", "4. "}},
		{`"/"`, []string{"/", "/", "/", "/"}},
	}
	for _, tt := range tests {
		t.Run(tt.styleType, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			got := listMarkers(t, boxtest.ParseAll(t, `
              <style>
                ol { list-style-type: `+tt.styleType+` }
              </style>
              <ol>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
              </ol>
            `))
			if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", tt.values) {
				t.Errorf("markers = %q, want %q", got, tt.values)
			}
		})
	}
}

func TestEmptyListStyleTypeMakesNoMarker(t *testing.T) {
	root := boxtest.ParseAll(t, `<ul><li style="list-style-type: ''">`)
	li := onlyChild(t, onlyChild(t, onlyChild(t, root)))
	for _, child := range boxes.ChildrenOf(li) {
		if strings.HasSuffix(child.Base().ElementTag, "::marker") {
			t.Errorf("list item has a marker %v", child)
		}
	}
}

func TestCounterSet(t *testing.T) {
	root := boxtest.ParseAll(t, `
      <style>
        body { counter-reset: h2 0 h3 4; font-size: 1px }
        article { counter-reset: h2 2 }
        h1 { counter-increment: h1 }
        h1::before { content: counter(h1) }
        h2 { counter-increment: h2; counter-set: h3 3 }
        h2::before { content: counter(h2) }
        h3 { counter-increment: h3 }
        h3::before { content: counter(h3) }
      </style>
      <article>
        <h1></h1>
      </article>
      <article>
        <h2></h2>
        <h3></h3>
      </article>
      <article>
        <h3></h3>
      </article>
      <article>
        <h2></h2>
      </article>
      <article>
        <h3></h3>
        <h3></h3>
      </article>
      <article>
        <h1></h1>
        <h2></h2>
        <h3></h3>
      </article>
    `)
	var got [][]string
	for _, article := range body(t, root) {
		var texts []string
		for _, heading := range boxes.ChildrenOf(article) {
			texts = append(texts, textAt(t, heading, 3))
		}
		got = append(got, texts)
	}
	if want := "[[1] [3 4] [5] [3] [4 5] [1 3 4]]"; fmt.Sprint(got) != want {
		t.Errorf("counters = %v, want %v", got, want)
	}
}

func TestCounterStylesWithCyclicExtends(t *testing.T) {
	root := boxtest.ParseAll(t, `
      <style>
        @counter-style a {
          system: extends b;
          prefix: a;
        }
        @counter-style b {
          system: extends c;
          suffix: b;
        }
        @counter-style c {
          system: extends b;
          pad: 2 c;
        }
        @counter-style d {
          system: extends d;
          prefix: d;
        }
        @counter-style e {
          system: extends unknown;
          prefix: e;
        }
        @counter-style f {
          system: extends decimal;
          symbols: a;
        }
        @counter-style g {
          system: extends decimal;
          additive-symbols: 1 a;
        }
      </style>
      <ol>
        <li style="list-style-type: a"></li>
        <li style="list-style-type: b"></li>
        <li style="list-style-type: c"></li>
        <li style="list-style-type: d"></li>
        <li style="list-style-type: e"></li>
        <li style="list-style-type: f"></li>
        <li style="list-style-type: g"></li>
        <li style="list-style-type: h"></li>
      </ol>
    `)
	want := []string{"a1b", "2b", "c3. ", "d4. ", "e5. ", "6. ", "7. ", "8. "}
	if got := listMarkers(t, root); fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		t.Errorf("markers = %q, want %q", got, want)
	}
}
