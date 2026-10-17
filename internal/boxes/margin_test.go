package boxes_test

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
)

// marginText returns the text of the margin box that follows the content of
// a page: it has one line with one text box.
func marginText(t *testing.T, page *boxes.PageBox, margin int) string {
	t.Helper()
	children := boxes.ChildrenOf(page)
	if len(children) <= margin || children[0].Base().ElementTag != "html" {
		t.Fatalf("page has %v, want the html and at least %d margin boxes", children, margin)
	}
	lines := boxes.ChildrenOf(children[margin])
	if len(lines) != 1 {
		t.Fatalf("%v has %d lines, want 1", children[margin], len(lines))
	}
	texts := boxes.ChildrenOf(lines[0])
	if len(texts) != 1 {
		t.Fatalf("%v has %d children, want 1", lines[0], len(texts))
	}
	text, ok := texts[0].(*boxes.TextBox)
	if !ok {
		t.Fatalf("%v is not a text box", texts[0])
	}
	return text.Text
}

func expectMarginTexts(t *testing.T, pages []*boxes.PageBox, margin int, want ...string) {
	t.Helper()
	if len(pages) != len(want) {
		t.Fatalf("got %d pages, want %d", len(pages), len(want))
	}
	for i, page := range pages {
		if got := marginText(t, page, margin); got != want[i] {
			t.Errorf("page %d shows %q, want %q", i+1, got, want[i])
		}
	}
}

func atKeywords(page *boxes.PageBox) []string {
	var keywords []string
	for _, box := range boxes.ChildrenOf(page)[1:] {
		keywords = append(keywords, box.(*boxes.MarginBox).AtKeyword)
	}
	return keywords
}

func TestMarginBoxes(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          /* Make the page content area only 10px high and wide,
             so every word in <p> end up on a page of its own. */
          size: 30px;
          margin: 10px;
          @top-center { content: "Title" }
        }
        @page :first {
          @bottom-left { content: "foo" }
          @bottom-left-corner { content: "baz" }
        }
      </style>
      <p>lorem ipsum
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	for _, page := range pages {
		if tag := boxes.ChildrenOf(page)[0].Base().ElementTag; tag != "html" {
			t.Errorf("the first child of the page is %q, want html", tag)
		}
	}
	if got, want := fmt.Sprint(atKeywords(pages[0])), "[@top-center @bottom-left @bottom-left-corner]"; got != want {
		t.Errorf("margin boxes of the first page = %s, want %s", got, want)
	}
	if got, want := fmt.Sprint(atKeywords(pages[1])), "[@top-center]"; got != want {
		t.Errorf("margin boxes of the second page = %s, want %s", got, want)
	}
	if got := marginText(t, pages[1], 1); got != "Title" {
		t.Errorf("margin box shows %q, want Title", got)
	}
}

func TestStringSetFillsLaterPages(t *testing.T) {
	// Both pages get the string in the bottom center margin box.
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          @bottom-center { content: string(text_header) }
        }
        p {
          string-set: text_header content();
        }
        .page {
          page-break-before: always;
        }
      </style>
      <p>first assignment</p>
      <div class="page"></div>
    `)
	expectMarginTexts(t, pages, 1, "first assignment", "first assignment")
}

func TestStringSetContentValues(t *testing.T) {
	for _, value := range []string{"", "text", "before", "after"} {
		t.Run(value, func(t *testing.T) {
			testutil.AssertNoLogs(t)
			extra := ""
			want := "first assignment"
			if value == "before" || value == "after" {
				extra = fmt.Sprintf(`p:%s{content: "pseudo"}`, value)
				want = "pseudo"
			}
			pages := boxtest.RenderPages(t, fmt.Sprintf(`
          <style>
            @page {
              @top-center { content: string(text_header) }
            }
            p {
              string-set: text_header content(%s);
            }
            %s
          </style>
          <p>first assignment</p>
        `, value, extra))
			expectMarginTexts(t, pages, 1, want)
		})
	}
}

func TestStringSetFirstUsesTheFirstAssignmentOfThePage(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          @top-center { content: string(text_header, first) }
        }
        p {
          string-set: text_header content();
        }
      </style>
      <p>first assignment</p>
      <p>Second assignment</p>
    `)
	expectMarginTexts(t, pages, 1, "first assignment")
}

func TestStringSetFirstExceptExcludesThePageOfTheAssignment(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          @top-center { content: string(header_nofirst, first-except) }
        }
        p{
          string-set: header_nofirst content();
        }
        .page{
          page-break-before: always;
        }
      </style>
      <p>first_excepted</p>
      <div class="page"></div>
    `)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if lines := boxes.ChildrenOf(boxes.ChildrenOf(pages[0])[1]); len(lines) != 0 {
		t.Errorf("the first page shows %v, want nothing", lines)
	}
	if got := marginText(t, pages[1], 1); got != "first_excepted" {
		t.Errorf("the second page shows %q, want first_excepted", got)
	}
}

func TestStringSetLastUsesTheMostRecentAssignment(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          @top-center { content: string(header_last, last) }
        }
        p {
          string-set: header_last content();
        }
      </style>
      <p>String set</p>
      <p>Second assignment</p>
    `)
	expectMarginTexts(t, pages, 1, "Second assignment")
}

func TestStringSetWithCountersAndAttributes(t *testing.T) {
	testutil.AssertNoLogs(t)
	page := boxtest.RenderPages(t, `
      <style>
        @page {
          @top-center { content: string(text_header, first) }
          @bottom-center { content: string(text_footer, last) }
        }
        html { counter-reset: a }
        body { counter-increment: a }
        ul { counter-reset: b }
        li {
          counter-increment: b;
          string-set:
            text_header content(before) "-" content() "-" content(after)
                        counter(a, upper-roman) '.' counters(b, '|'),
            text_footer content(before) '-' attr(class)
                        counters(b, '|') "/" counter(a, upper-roman);
        }
        li:before { content: 'before!' }
        li:after { content: 'after!' }
        li:last-child:before { content: 'before!last' }
        li:last-child:after { content: 'after!last' }
      </style>
      <ul>
        <li class="firstclass">first
        <li>
          <ul>
            <li class="secondclass">second
    `)[0]
	if got := marginText(t, page, 1); got != "before!-first-after!I.1" {
		t.Errorf("header = %q, want %q", got, "before!-first-after!I.1")
	}
	if got := marginText(t, page, 2); got != "before!last-secondclass2|1/I" {
		t.Errorf("footer = %q, want %q", got, "before!last-secondclass2|1/I")
	}
}

func TestStringSetFromAttributesOfImagesThatFailToLoad(t *testing.T) {
	// The missing image logs an error, as the document may be at fault.
	testutil.CaptureLogs(t)
	page := boxtest.RenderPages(t, `
      <style>
        img { string-set: left attr(alt) }
        img + img { string-set: right attr(alt) }
        @page { @top-left  { content: '[' string(left)  ']' }
                @top-right { content: '{' string(right) '}' } }
      </style>
      <img src=pattern.png alt="Chocolate">
      <img src=no_such_file.png alt="Cake">
    `)[0]
	if got := marginText(t, page, 1); got != "[Chocolate]" {
		t.Errorf("left = %q, want [Chocolate]", got)
	}
	if got := marginText(t, page, 2); got != "{Cake}" {
		t.Errorf("right = %q, want {Cake}", got)
	}
}

func TestStringSetKeepsEmptyAndBlankStrings(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page { @top-left  { content: '[' string(left) ']' } }
        p { page-break-before: always }
        .initial { string-set: left 'initial' }
        .empty   { string-set: left ''        }
        .space   { string-set: left ' '       }
      </style>

      <p class="initial">Initial</p>
      <p class="empty">Empty</p>
      <p class="space">Space</p>
    `)
	expectMarginTexts(t, pages, 1, "[initial]", "[]", "[ ]")
}

func TestStringSetNamesAreCaseSensitive(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          @top-center {
            content: string(text_header, first)
                     ' ' string(TEXT_header, first)
          }
        }
        p { string-set: text_header content() }
        div { string-set: TEXT_header content() }
      </style>
      <p>first assignment</p>
      <div>second assignment</div>
    `)
	expectMarginTexts(t, pages, 1, "first assignment second assignment")
}

func TestStringSetStartUsesTheAssignmentAtTheStartOfThePage(t *testing.T) {
	testutil.AssertNoLogs(t)
	pages := boxtest.RenderPages(t, `
      <style>
        @page { @top-left  { content: '[' string(p, start) ']' } }
        p { string-set: p content(); page-break-after: always }
      </style>
      <article></article>
      <p>1</p>
      <article></article>
      <p>2</p>
      <p>3</p>
      <article></article>
    `)
	expectMarginTexts(t, pages, 1, "[]", "[1]", "[3]", "[3]")
}

func TestPageCounters(t *testing.T) {
	testutil.AssertNoLogs(t)
	// The spaces do not break, so that each text is a single line.
	const nbsp = "\u00a0"
	pages := boxtest.RenderPages(t, `
      <style>
        @page {
          /* Make the page content area only 10px high and wide,
             so every word in <p> end up on a page of its own. */
          size: 30px;
          margin: 10px;
          @bottom-center {
            content: "Page`+nbsp+`" counter(page) "`+nbsp+`of`+nbsp+`" counter(pages) ".";
          }
        }
      </style>
      <p>lorem ipsum dolor
    `)
	if len(pages) != 3 {
		t.Fatalf("got %d pages, want 3", len(pages))
	}
	want := make([]string, len(pages))
	for i := range want {
		want[i] = fmt.Sprintf("Page%s%d%sof%s3.", nbsp, i+1, nbsp, nbsp)
	}
	expectMarginTexts(t, pages, 1, want...)
}
