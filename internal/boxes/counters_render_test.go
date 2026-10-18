package boxes_test

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// markerTexts returns the text of the marker of each item of the only list
// in the body of a laid out document.
func markerTexts(t *testing.T, source string) []string {
	t.Helper()
	list := renderBody(t, source)
	if len(list) != 1 {
		t.Fatalf("body has %d children, want 1", len(list))
	}
	var texts []string
	for _, li := range boxes.ChildrenOf(list[0]) {
		box := descend(t, li, 0, 0, 0)
		text, ok := box.(*boxes.TextBox)
		if !ok {
			t.Fatalf("%v is not a text box", box)
		}
		texts = append(texts, text.Text)
	}
	return texts
}

func TestRenderedCounterSymbols(t *testing.T) {
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
			got := markerTexts(t, `
              <style>
                ol { list-style-type: symbols(`+tt.arguments+`) }
              </style>
              <ol>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
              </ol>
            `)
			if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", tt.values) {
				t.Errorf("markers = %q, want %q", got, tt.values)
			}
		})
	}
}

func TestRenderedListStyleTypes(t *testing.T) {
	tests := []struct {
		styleType string
		values    []string
	}{
		{"decimal", []string{"1. ", "2. ", "3. ", "4. "}},
		{`"/"`, []string{"/", "/", "/", "/"}},
	}
	for _, tt := range tests {
		t.Run(tt.styleType, func(t *testing.T) {
			got := markerTexts(t, `
              <style>
                ol { list-style-type: `+tt.styleType+` }
              </style>
              <ol>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
                <li>abc</li>
              </ol>
            `)
			if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", tt.values) {
				t.Errorf("markers = %q, want %q", got, tt.values)
			}
		})
	}
}

func TestEmptyStringListStyleTypeRenders(t *testing.T) {
	pages := boxtest.RenderPages(t, `<ul><li style="list-style-type: ''">`)
	if len(pages) != 1 {
		t.Errorf("got %d pages, want 1", len(pages))
	}
}

func TestRenderedCounterSet(t *testing.T) {
	articles := renderBody(t, `
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
	for _, article := range articles {
		var texts []string
		for _, heading := range boxes.ChildrenOf(article) {
			text, ok := descend(t, heading, 0, 0, 0).(*boxes.TextBox)
			if !ok {
				t.Fatalf("%v has no text", heading)
			}
			texts = append(texts, text.Text)
		}
		got = append(got, texts)
	}
	if want := "[[1] [3 4] [5] [3] [4 5] [1 3 4]]"; fmt.Sprint(got) != want {
		t.Errorf("counters = %v, want %v", got, want)
	}
}

func TestRenderedCounterStylesWithCyclicExtends(t *testing.T) {
	got := markerTexts(t, `
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
	if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		t.Errorf("markers = %q, want %q", got, want)
	}
}

func TestNestedListWithOwnCounterResetNumbersItsItems(t *testing.T) {
	t.Skip("not supported: a counter-reset on a nested list replaces its implicit reset of list-item")
	list := renderBody(t, `
      <ol>
        <li></li>
        <li>
          <ol style="counter-reset: a">
            <li></li>
            <li></li>
          </ol>
        </li>
        <li></li>
      </ol>
    `)
	items := boxes.ChildrenOf(list[0])
	if len(items) != 3 {
		t.Fatalf("list has %d items, want 3", len(items))
	}
	nested := boxes.ChildrenOf(descend(t, items[1], 1))
	if len(nested) != 2 {
		t.Fatalf("nested list has %d items, want 2", len(nested))
	}
	expectText(t, descend(t, items[0], 0, 0, 0), "1. ")
	expectText(t, descend(t, items[1], 0, 0, 0), "2. ")
	expectText(t, descend(t, nested[0], 0, 0, 0), "1. ")
	expectText(t, descend(t, nested[1], 0, 0, 0), "2. ")
	expectText(t, descend(t, items[2], 0, 0, 0), "3. ")
}
