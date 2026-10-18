package layout

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestEmptyLinebox(t *testing.T) {
	d := renderPage(t, `<p> </p>`)
	paragraph := kids(t, d.body(t), 1)[0]
	if n := len(boxes.ChildrenOf(paragraph)); n != 0 {
		t.Fatalf("paragraph has %d children, want 0", n)
	}
	if h := paragraph.Base().Height.V; h != 0 {
		t.Fatalf("paragraph height = %g, want 0", h)
	}
}

func TestEmptyLineboxRemovedSpace(t *testing.T) {
	t.Skip("not supported: the line left after a forced break keeps its box")
	d := renderPage(t, `
      <style>
        p { width: 1px }
      </style>
      <p><br>  </p>
    `)
	paragraph := kids(t, d.body(t), 1)[0]
	if n := len(boxes.ChildrenOf(paragraph)); n != 1 {
		t.Fatalf("paragraph has %d lines, want 1", n)
	}
}

func TestBreakingLinebox(t *testing.T) {
	d := renderPage(t, `
      <style>
      p { font-size: 13px;
          width: 300px;
          font-family: testfont;
          background-color: #393939;
          color: #FFFFFF;
          line-height: 1;
          text-decoration: underline overline line-through;}
      </style>
      <p><em>Lorem<strong> Ipsum <span>is very</span>simply</strong><em>
      dummy</em>text of the printing and. naaaa </em> naaaa naaaa naaaa
      naaaa naaaa naaaa naaaa naaaa</p>
    `)
	paragraph := kids(t, d.body(t), 1)[0]
	// Every glyph of the test font is one em wide, so a line of 300px holds
	// 23 characters of 13px.
	lines := kids(t, paragraph, 5)
	for _, line := range lines {
		if line.Base().Style.FontSize() != 13 || line.Base().ElementTag != "p" {
			t.Errorf("line %v: font size %v, tag %q", line, line.Base().Style.FontSize(), line.Base().ElementTag)
		}
		for _, child := range boxes.ChildrenOf(line) {
			if tag := child.Base().ElementTag; tag != "em" && tag != "p" {
				t.Errorf("line child tag %q", tag)
			}
			if child.Base().Style.FontSize() != 13 {
				t.Errorf("child font size %v", child.Base().Style.FontSize())
			}
		}
	}
}

func TestPositionXLTR(t *testing.T) {
	d := renderPage(t, `
      <style>
        span {
          padding: 0 10px 0 15px;
          margin: 0 2px 0 3px;
          border: 1px solid;
         }
      </style>
      <body><span>a<br>b<br>c</span>`)
	lines := kids(t, d.body(t), 3)
	span1 := kids(t, lines[0], 1)[0]
	if span1.Base().PositionX != 0 {
		t.Errorf("span 1 x = %g, want 0", span1.Base().PositionX)
	}
	text1 := kids(t, span1, 2)[0]
	if got, want := text1.Base().PositionX, float64(15+3+1); got != want {
		t.Errorf("text 1 x = %g, want %g", got, want)
	}
	span2 := kids(t, lines[1], 1)[0]
	if span2.Base().PositionX != 0 {
		t.Errorf("span 2 x = %g, want 0", span2.Base().PositionX)
	}
	text2 := kids(t, span2, 2)[0]
	if text2.Base().PositionX != 0 {
		t.Errorf("text 2 x = %g, want 0", text2.Base().PositionX)
	}
	span3 := kids(t, lines[2], 1)[0]
	if span3.Base().PositionX != 0 {
		t.Errorf("span 3 x = %g, want 0", span3.Base().PositionX)
	}
	text3 := kids(t, span3, 1)[0]
	if text3.Base().PositionX != 0 {
		t.Errorf("text 3 x = %g, want 0", text3.Base().PositionX)
	}
}

func TestPositionXRTL(t *testing.T) {
	d := renderPage(t, `
      <style>
        body {
          direction: rtl;
          width: 100px;
        }
        span {
          padding: 0 10px 0 15px;
          margin: 0 2px 0 3px;
          border: 1px solid;
         }
      </style>
      <body><span>a<br>b<br>c</span>`)
	lines := kids(t, d.body(t), 3)

	span1 := kids(t, lines[0], 1)[0]
	text1 := kids(t, span1, 2)[0]
	want := 100 - text1.Base().Width.V - (10 + 2 + 1)
	if got := span1.Base().PositionX; got != want {
		t.Errorf("span 1 x = %g, want %g", got, want)
	}
	if got := text1.Base().PositionX; got != want {
		t.Errorf("text 1 x = %g, want %g", got, want)
	}

	span2 := kids(t, lines[1], 1)[0]
	text2 := kids(t, span2, 2)[0]
	want = 100 - text2.Base().Width.V
	if got := span2.Base().PositionX; got != want {
		t.Errorf("span 2 x = %g, want %g", got, want)
	}
	if got := text2.Base().PositionX; got != want {
		t.Errorf("text 2 x = %g, want %g", got, want)
	}

	span3 := kids(t, lines[2], 1)[0]
	text3 := kids(t, span3, 1)[0]
	if got, want := span3.Base().PositionX, 100-text3.Base().Width.V-(15+3+1); got != want {
		t.Errorf("span 3 x = %g, want %g", got, want)
	}
	if got, want := text3.Base().PositionX, 100-text3.Base().Width.V; got != want {
		t.Errorf("text 3 x = %g, want %g", got, want)
	}
}

func TestBreakingLineboxRegression1(t *testing.T) {
	// See https://unicode.org/reports/tr14/
	d := renderPage(t, "<pre>a\nb\rc\r\nd e</pre>")
	pre := kids(t, d.body(t), 1)[0]
	var texts []string
	for _, line := range boxes.ChildrenOf(pre) {
		texts = append(texts, textOf(t, kids(t, line, 1)[0]))
	}
	if want := []string{"a", "b", "c", "d", "e"}; !slices.Equal(texts, want) {
		t.Fatalf("lines = %q, want %q", texts, want)
	}
}

func TestBreakingLineboxRegression2(t *testing.T) {
	const source = `
      <p style="width: %d.5em; font-family: testfont">ab
      <span style="padding-right: 1em; margin-right: 1em">c def</span>g
      hi</p>`
	for i := range 16 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(source, i))
			p := kids(t, d.body(t), 1)[0]
			text := func(box boxes.Box) string { return textOf(t, box) }
			var lines []boxes.Box
			switch {
			case i <= 3:
				lines = kids(t, p, 4)
				if got := text(kids(t, lines[0], 1)[0]); got != "ab" {
					t.Errorf("line 1 = %q", got)
				}
				span := kids(t, lines[1], 1)[0]
				if got := text(kids(t, span, 1)[0]); got != "c" {
					t.Errorf("line 2 = %q", got)
				}
				line3 := kids(t, lines[2], 2)
				if got := text(kids(t, line3[0], 1)[0]); got != "def" {
					t.Errorf("line 3 span = %q", got)
				}
				if got := text(line3[1]); got != "g" {
					t.Errorf("line 3 = %q", got)
				}
				if got := text(kids(t, lines[3], 1)[0]); got != "hi" {
					t.Errorf("line 4 = %q", got)
				}
			case i <= 8:
				lines = kids(t, p, 3)
				line1 := kids(t, lines[0], 2)
				if got := text(line1[0]); got != "ab " {
					t.Errorf("line 1 = %q", got)
				}
				if got := text(kids(t, line1[1], 1)[0]); got != "c" {
					t.Errorf("line 1 span = %q", got)
				}
				line2 := kids(t, lines[1], 2)
				if got := text(kids(t, line2[0], 1)[0]); got != "def" {
					t.Errorf("line 2 span = %q", got)
				}
				if got := text(line2[1]); got != "g" {
					t.Errorf("line 2 = %q", got)
				}
				if got := text(kids(t, lines[2], 1)[0]); got != "hi" {
					t.Errorf("line 3 = %q", got)
				}
			case i <= 10:
				lines = kids(t, p, 2)
				line1 := kids(t, lines[0], 2)
				if got := text(line1[0]); got != "ab " {
					t.Errorf("line 1 = %q", got)
				}
				if got := text(kids(t, line1[1], 1)[0]); got != "c" {
					t.Errorf("line 1 span = %q", got)
				}
				line2 := kids(t, lines[1], 2)
				if got := text(kids(t, line2[0], 1)[0]); got != "def" {
					t.Errorf("line 2 span = %q", got)
				}
				if got := text(line2[1]); got != "g hi" {
					t.Errorf("line 2 = %q", got)
				}
			case i <= 13:
				lines = kids(t, p, 2)
				line1 := kids(t, lines[0], 3)
				if got := text(line1[0]); got != "ab " {
					t.Errorf("line 1 = %q", got)
				}
				if got := text(kids(t, line1[1], 1)[0]); got != "c def" {
					t.Errorf("line 1 span = %q", got)
				}
				if got := text(line1[2]); got != "g" {
					t.Errorf("line 1 end = %q", got)
				}
				if got := text(kids(t, lines[1], 1)[0]); got != "hi" {
					t.Errorf("line 2 = %q", got)
				}
			default:
				lines = kids(t, p, 1)
				line1 := kids(t, lines[0], 3)
				if got := text(line1[0]); got != "ab " {
					t.Errorf("line 1 = %q", got)
				}
				if got := text(kids(t, line1[1], 1)[0]); got != "c def" {
					t.Errorf("line 1 span = %q", got)
				}
				if got := text(line1[2]); got != "g hi" {
					t.Errorf("line 1 end = %q", got)
				}
			}
		})
	}
}

func TestBreakingLineboxRegression3(t *testing.T) {
	d := renderPage(t, `<div style="width: 5.5em; font-family: testfont">aaaa aaaa a [<span>aaa</span>]`)
	div := kids(t, d.body(t), 1)[0]
	lines := kids(t, div, 4)
	if got := textOf(t, boxes.ChildrenOf(lines[0])[0]); got != "aaaa" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textOf(t, boxes.ChildrenOf(lines[1])[0]); got != "aaaa" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textOf(t, boxes.ChildrenOf(lines[2])[0]); got != "a" {
		t.Errorf("line 3 = %q", got)
	}
	line4 := kids(t, lines[3], 3)
	if got := textOf(t, line4[0]); got != "[" {
		t.Errorf("line 4 start = %q", got)
	}
	if got := textOf(t, line4[2]); got != "]" {
		t.Errorf("line 4 end = %q", got)
	}
	if got := textOf(t, kids(t, line4[1], 1)[0]); got != "aaa" {
		t.Errorf("line 4 span = %q", got)
	}
}

func TestBreakingLineboxRegression4(t *testing.T) {
	d := renderPage(t, `<div style="width: 5.5em; font-family: testfont">aaaa a <span>b c</span>d`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 3)
	if got := textOf(t, kids(t, lines[0], 1)[0]); got != "aaaa" {
		t.Errorf("line 1 = %q", got)
	}
	line2 := kids(t, lines[1], 2)
	if got := textOf(t, line2[0]); got != "a " {
		t.Errorf("line 2 = %q", got)
	}
	if got := textOf(t, kids(t, line2[1], 1)[0]); got != "b" {
		t.Errorf("line 2 span = %q", got)
	}
	line3 := kids(t, lines[2], 2)
	if got := textOf(t, kids(t, line3[0], 1)[0]); got != "c" {
		t.Errorf("line 3 span = %q", got)
	}
	if got := textOf(t, line3[1]); got != "d" {
		t.Errorf("line 3 = %q", got)
	}
}

func TestBreakingLineboxRegression5(t *testing.T) {
	d := renderPage(t, `<div style="width: 5.5em; font-family: testfont">`+
		`<span>aaaa aaaa a a a</span><span>bc</span>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 4)
	for i, want := range []string{"aaaa", "aaaa", "a a"} {
		span := boxes.ChildrenOf(lines[i])[0]
		if got := textOf(t, kids(t, span, 1)[0]); got != want {
			t.Errorf("line %d = %q, want %q", i+1, got, want)
		}
	}
	line4 := kids(t, lines[3], 2)
	if got := textOf(t, kids(t, line4[0], 1)[0]); got != "a" {
		t.Errorf("line 4 = %q", got)
	}
	if got := textOf(t, kids(t, line4[1], 1)[0]); got != "bc" {
		t.Errorf("line 4 end = %q", got)
	}
}

func TestBreakingLineboxRegression6(t *testing.T) {
	d := renderPage(t, `<div style="width: 5.5em; font-family: testfont">`+
		`a a <span style="white-space: nowrap">/ccc</span>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	if got := textOf(t, kids(t, lines[0], 1)[0]); got != "a a" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textOf(t, kids(t, kids(t, lines[1], 1)[0], 1)[0]); got != "/ccc" {
		t.Errorf("line 2 = %q", got)
	}
}

func TestBreakingLineboxRegression7(t *testing.T) {
	d := renderPage(t, `<div style="width: 3.5em; font-family: testfont">`+
		`<span><span>abc d e</span></span><span>f`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 3)
	for i, want := range []string{"abc", "d", "e"} {
		if got := textAt(t, lines[i], 0, 0, 0); got != want {
			t.Errorf("line %d = %q, want %q", i+1, got, want)
		}
	}
	if got := textAt(t, lines[2], 1, 0); got != "f" {
		t.Errorf("line 3 end = %q", got)
	}
}

func TestBreakingLineboxRegression8(t *testing.T) {
	d := renderPage(t, `<p style="font-family: testfont"><span>
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
bbbbbbbbbbb
<b>cccc</b></span>ddd</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	if got, want := textAt(t, lines[0], 0, 0), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbb"; got != want {
		t.Errorf("line 1 = %q, want %q", got, want)
	}
	if got := textAt(t, lines[1], 0, 0, 0); got != "cccc" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[1], 1); got != "ddd" {
		t.Errorf("line 2 end = %q", got)
	}
}

func TestBreakingLineboxRegression9(t *testing.T) {
	t.Skip("not supported: a line cannot break before the b element, as canBreakInside is false for the span")
	d := renderPage(t, `<p style="font-family: testfont"><span>
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbb
<b>cccc</b></span>ddd</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	if got, want := textAt(t, lines[0], 0, 0), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbb"; got != want {
		t.Errorf("line 1 = %q, want %q", got, want)
	}
	if got := textAt(t, lines[1], 0, 0, 0); got != "cccc" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[1], 1); got != "ddd" {
		t.Errorf("line 2 end = %q", got)
	}
}

func TestBreakingLineboxRegression10(t *testing.T) {
	d := renderPage(t, `<p style="width:195px; font-family: testfont">
  <span>
    <span>xxxxxx YYY yyyyyy yyy</span>
    ZZZZZZ zzzzz
  </span> )x
</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 4)
	if got := textAt(t, lines[0], 0, 0, 0); got != "xxxxxx YYY" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 0, 0); got != "yyyyyy yyy" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[2], 0, 0); got != "ZZZZZZ zzzzz" {
		t.Errorf("line 3 = %q", got)
	}
	if got := textAt(t, lines[3], 0); got != ")x" {
		t.Errorf("line 4 = %q", got)
	}
}

func TestBreakingLineboxRegression11(t *testing.T) {
	d := renderPage(t, `<p style="width:10em; font-family: testfont">  line 1<br><span>123 567 90</span>x</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 3)
	if got := textAt(t, lines[0], 0); got != "line 1" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 0); got != "123 567" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[2], 0, 0); got != "90" {
		t.Errorf("line 3 = %q", got)
	}
	if got := textAt(t, lines[2], 1); got != "x" {
		t.Errorf("line 3 end = %q", got)
	}
}

func TestBreakingLineboxRegression12(t *testing.T) {
	d := renderPage(t, `<p style="width:10em; font-family: testfont">  <br><span>123 567 90</span>x</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 3)
	if got := textAt(t, lines[1], 0, 0); got != "123 567" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[2], 0, 0); got != "90" {
		t.Errorf("line 3 = %q", got)
	}
	if got := textAt(t, lines[2], 1); got != "x" {
		t.Errorf("line 3 end = %q", got)
	}
}

func TestBreakingLineboxRegression13(t *testing.T) {
	d := renderPage(t, `<p style="width:10em; font-family: testfont">  123 567 90 <span>123 567 90</span>x</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 3)
	if got := textAt(t, lines[0], 0); got != "123 567 90" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 0); got != "123 567" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[2], 0, 0); got != "90" {
		t.Errorf("line 3 = %q", got)
	}
	if got := textAt(t, lines[2], 1); got != "x" {
		t.Errorf("line 3 end = %q", got)
	}
}

func TestBreakingLineboxRegression14(t *testing.T) {
	d := renderPage(t, `<style>  body {font-family: testfont; width: 3em}</style><span> <span>a</span> b</span><span>c</span>`)
	lines := kids(t, d.body(t), 2)
	if got := textAt(t, lines[0], 0, 0, 0); got != "a" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 0); got != "b" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[1], 1, 0); got != "c" {
		t.Errorf("line 2 end = %q", got)
	}
}

func TestLineboxText(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { width: 165px; font-family: testfont }
      </style>
      <p><em>Lorem Ipsum</em>is very <strong>coool</strong></p>
    `)
	lines := boxes.ChildrenOf(kids(t, d.body(t), 1)[0])
	if len(lines) != 3 {
		t.Fatalf("%d lines, want 3", len(lines))
	}
	var parts []string
	for _, line := range lines {
		parts = append(parts, allText(line))
	}
	if got := strings.Join(parts, " "); got != "Lorem Ipsumis very coool" {
		t.Fatalf("text = %q", got)
	}
}

func TestLineboxPositions(t *testing.T) {
	for _, tt := range []struct {
		width float64
		lines int
	}{{165, 3}, {1, 5}, {0, 5}} {
		t.Run(fmt.Sprint(tt.width), func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
          <style>
            p { width:%gpx; font-family: testfont;
                line-height: 20px }
          </style>
          <p>this is test for <strong>Wrapsinety</strong></p>`, tt.width))
			lines := boxes.ChildrenOf(kids(t, d.body(t), 1)[0])
			if len(lines) != tt.lines {
				t.Fatalf("%d lines, want %d", len(lines), tt.lines)
			}
			refY, refX := lines[0].Base().PositionY, lines[0].Base().PositionX
			for _, line := range lines {
				if line.Base().PositionY != refY || line.Base().PositionX != refX {
					t.Fatalf("line at (%g, %g), want (%g, %g)", line.Base().PositionX, line.Base().PositionY, refX, refY)
				}
				for _, box := range boxes.ChildrenOf(line) {
					if box.Base().PositionX != refX || box.Base().PositionY != refY {
						t.Fatalf("box at (%g, %g), want (%g, %g)", box.Base().PositionX, box.Base().PositionY, refX, refY)
					}
					refX += box.Base().Width.V
				}
				if refX-line.Base().PositionX > line.Base().Width.V {
					t.Fatalf("children are %g wide in a line of %g", refX-line.Base().PositionX, line.Base().Width.V)
				}
				refX = line.Base().PositionX
				refY += line.Base().Height.V
			}
		})
	}
}

func TestForcedLineBreaksPre(t *testing.T) {
	// These lines are small enough to fit on the default A4 page with the
	// default 12pt font-size.
	d := renderPage(t, `
      <style> pre { line-height: 42px }</style>
      <pre>Lorem ipsum dolor sit amet,
          consectetur adipiscing elit.


          Sed sollicitudin nibh

          et turpis molestie tristique.</pre>
    `)
	pre := kids(t, d.body(t), 1)[0]
	if pre.Base().ElementTag != "pre" {
		t.Fatalf("tag = %q", pre.Base().ElementTag)
	}
	for _, line := range kids(t, pre, 7) {
		if line.Kind() != boxes.KindLine || line.Base().Height.V != 42 {
			t.Errorf("%v has height %g", line, line.Base().Height.V)
		}
	}
}

func TestForcedLineBreaksParagraph(t *testing.T) {
	d := renderPage(t, `
      <style> p { line-height: 42px }</style>
      <p>Lorem ipsum dolor sit amet,<br>
        consectetur adipiscing elit.<br><br><br>
        Sed sollicitudin nibh<br>
        <br>

        et turpis molestie tristique.</p>
    `)
	paragraph := kids(t, d.body(t), 1)[0]
	if paragraph.Base().ElementTag != "p" {
		t.Fatalf("tag = %q", paragraph.Base().ElementTag)
	}
	for _, line := range kids(t, paragraph, 7) {
		if line.Kind() != boxes.KindLine || line.Base().Height.V != 42 {
			t.Errorf("%v has height %g", line, line.Base().Height.V)
		}
	}
}

func TestInlineBoxSplitting(t *testing.T) {
	// The text is strange to test some corner cases.
	for _, width := range []int{10000, 100, 10, 0} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
          <style>p { font-family: testfont; width: %dpx; }</style>
          <p><strong>Wrapsinety is a frée softwäre ./ visual rendèring enginè
                     for HTML !!! and CSS.</strong></p>
        `, width))
			lines := boxes.ChildrenOf(kids(t, d.body(t), 1)[0])
			if (width == 10000) != (len(lines) == 1) {
				t.Fatalf("%d lines at width %d", len(lines), width)
			}
			var parts []string
			for _, line := range lines {
				parts = append(parts, textAt(t, line, 0, 0))
			}
			want := "Wrapsinety is a frée softwäre ./ visual rendèring enginè for HTML !!! and CSS."
			if got := strings.Join(parts, " "); got != want {
				t.Fatalf("text = %q, want %q", got, want)
			}
		})
	}
}

func TestWhitespaceProcessing(t *testing.T) {
	for _, source := range []string{"a", "  a  ", " \n  \ta", " a\t "} {
		t.Run(fmt.Sprintf("%q", source), func(t *testing.T) {
			d := renderPage(t, "<p><em>"+source+"</em></p>")
			if got := textAt(t, kids(t, d.body(t), 1)[0], 0, 0, 0); got != "a" {
				t.Errorf("text = %q", got)
			}

			d = renderPage(t, `<p style="white-space: pre-line">

<em>`+strings.ReplaceAll(source, "\n", " ")+`</em></pre>`)
			lines := kids(t, kids(t, d.body(t), 1)[0], 3)
			if got := textAt(t, lines[2], 0, 0); got != "a" {
				t.Errorf("pre-line text = %q", got)
			}
		})
	}
}

func TestInlineReplacedAutoMargins(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 200px }
        img { display: inline; margin: auto; width: 50px }
      </style>
      <body><img src="pattern.png" />`)
	img := at(t, d.body(t), 0, 0).Base()
	for name, margin := range map[string]float64{
		"top": img.MarginTop.V, "right": img.MarginRight.V,
		"bottom": img.MarginBottom.V, "left": img.MarginLeft.V,
	} {
		if margin != 0 {
			t.Errorf("margin %s = %g, want 0", name, margin)
		}
	}
}

func TestEmptyInlineAutoMargins(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 200px }
        span { margin: auto }
      </style>
      <body><span></span>`)
	span := at(t, d.body(t), 0, 0).Base()
	if span.MarginTop.V == 0 || span.MarginBottom.V == 0 {
		t.Errorf("vertical margins = %g, %g, want nonzero half leading", span.MarginTop.V, span.MarginBottom.V)
	}
	if span.MarginRight.V != 0 || span.MarginLeft.V != 0 {
		t.Errorf("horizontal margins = %g, %g, want 0", span.MarginLeft.V, span.MarginRight.V)
	}
}

func TestLineCount(t *testing.T) {
	for _, tt := range []struct {
		source string
		lines  int
	}{
		{"<body>hyphénation", 1},                               // Default: no hyphenation
		{"<body lang=fr>hyphénation", 1},                       // lang only: no hyphenation
		{`<body style="hyphens: auto">hyphénation`, 1},         // hyphens only: no hyphenation
		{`<body style="hyphens: auto" lang=fr>hyphénation`, 4}, // both: hyphenation
		{"<body>hyp&shy;hénation", 2},                          // Hyphenation with soft hyphens
		{`<body style="hyphens: none">hyp&shy;hénation`, 1},    // unless disabled
	} {
		t.Run(tt.source, func(t *testing.T) {
			d := renderPage(t, `<html style="width: 5em; font-family: testfont">`+tt.source)
			if lines := boxes.ChildrenOf(d.body(t)); len(lines) != tt.lines {
				t.Fatalf("%d lines, want %d", len(lines), tt.lines)
			}
		})
	}
}

func TestVerticalAlign1(t *testing.T) {
	//            +-------+      <- position_y = 0
	//      +-----+       |
	// 40px |     |       | 60px
	//      |     |       |
	//      +-----+-------+      <- baseline
	d := renderPage(t, `
      <span>
        <img src="pattern.png" style="width: 40px"
        ><img src="pattern.png" style="width: 60px"
      ></span>`)
	body := d.body(t)
	line := kids(t, body, 1)[0]
	imgs := kids(t, kids(t, line, 1)[0], 2)
	img1, img2 := imgs[0].Base(), imgs[1].Base()
	if img1.Height.V != 40 || img2.Height.V != 60 {
		t.Errorf("heights = %g, %g, want 40, 60", img1.Height.V, img2.Height.V)
	}
	if img1.PositionY != 20 || img2.PositionY != 0 {
		t.Errorf("positions = %g, %g, want 20, 0", img1.PositionY, img2.PositionY)
	}
	// 60px + the descent of the font below the baseline
	if h := line.Base().Height.V; !(60 < h && h < 70) {
		t.Errorf("line height = %g, want between 60 and 70", h)
	}
	if body.Base().Height.V != line.Base().Height.V {
		t.Errorf("body height = %g, line height = %g", body.Base().Height.V, line.Base().Height.V)
	}
}

func TestVerticalAlign2(t *testing.T) {
	//            +-------+      <- position_y = 0
	//       35px |       |
	//      +-----+       | 60px
	// 40px |     |       |
	//      |     +-------+      <- baseline
	//      +-----+  15px
	d := renderPage(t, `
      <span>
        <img src="pattern.png" style="width: 40px; vertical-align: -15px"
        ><img src="pattern.png" style="width: 60px"></span>`)
	line := kids(t, d.body(t), 1)[0]
	imgs := kids(t, kids(t, line, 1)[0], 2)
	img1, img2 := imgs[0].Base(), imgs[1].Base()
	if img1.Height.V != 40 || img2.Height.V != 60 {
		t.Errorf("heights = %g, %g, want 40, 60", img1.Height.V, img2.Height.V)
	}
	if img1.PositionY != 35 || img2.PositionY != 0 {
		t.Errorf("positions = %g, %g, want 35, 0", img1.PositionY, img2.PositionY)
	}
	if line.Base().Height.V != 75 || d.body(t).Base().Height.V != 75 {
		t.Errorf("heights of the line and body = %g, %g, want 75", line.Base().Height.V, d.body(t).Base().Height.V)
	}
}

func TestVerticalAlign3(t *testing.T) {
	// Same as previously, but with percentages.
	d := renderPage(t, `
      <span style="line-height: 10px">
        <img src="pattern.png" style="width: 40px; vertical-align: -150%"
        ><img src="pattern.png" style="width: 60px"></span>`)
	line := kids(t, d.body(t), 1)[0]
	imgs := kids(t, kids(t, line, 1)[0], 2)
	img1, img2 := imgs[0].Base(), imgs[1].Base()
	if img1.Height.V != 40 || img2.Height.V != 60 {
		t.Errorf("heights = %g, %g, want 40, 60", img1.Height.V, img2.Height.V)
	}
	if img1.PositionY != 35 || img2.PositionY != 0 {
		t.Errorf("positions = %g, %g, want 35, 0", img1.PositionY, img2.PositionY)
	}
	if line.Base().Height.V != 75 || d.body(t).Base().Height.V != 75 {
		t.Errorf("heights of the line and body = %g, %g, want 75", line.Base().Height.V, d.body(t).Base().Height.V)
	}
}

func TestVerticalAlign4(t *testing.T) {
	// Same again, but have the vertical-align on an inline box.
	d := renderPage(t, `
      <span style="line-height: 10px">
        <span style="line-height: 10px; vertical-align: -15px">
          <img src="pattern.png" style="width: 40px"></span>
        <img src="pattern.png" style="width: 60px"></span>`)
	line := kids(t, d.body(t), 1)[0]
	span1 := kids(t, line, 1)[0]
	children := kids(t, span1, 3)
	img1 := kids(t, children[0], 1)[0].Base()
	img2 := children[2].Base()
	if img1.Height.V != 40 || img2.Height.V != 60 {
		t.Errorf("heights = %g, %g, want 40, 60", img1.Height.V, img2.Height.V)
	}
	if img1.PositionY != 35 || img2.PositionY != 0 {
		t.Errorf("positions = %g, %g, want 35, 0", img1.PositionY, img2.PositionY)
	}
	if line.Base().Height.V != 75 || d.body(t).Base().Height.V != 75 {
		t.Errorf("heights of the line and body = %g, %g, want 75", line.Base().Height.V, d.body(t).Base().Height.V)
	}
}

func TestVerticalAlign5(t *testing.T) {
	d := renderPage(t, `<span style="line-height: 12px; font-size: 12px; font-family: testfont">`+
		`<img src="pattern.png" style="width: 40px; vertical-align: middle">`+
		`<img src="pattern.png" style="width: 60px"></span>`)
	body := d.body(t)
	line := kids(t, body, 1)[0]
	imgs := kids(t, kids(t, line, 1)[0], 2)
	img1, img2 := imgs[0].Base(), imgs[1].Base()
	if img1.Height.V != 40 || img2.Height.V != 60 {
		t.Errorf("heights = %g, %g, want 40, 60", img1.Height.V, img2.Height.V)
	}
	// The middle of the image is half of the ex-height above the baseline of
	// the parent. Fonts declared by @font-face do not give the ex unit yet,
	// so the position of the first image is not checked.
	if img2.PositionY != 0 {
		t.Errorf("second image y = %g, want 0", img2.PositionY)
	}
	if body.Base().Height.V != line.Base().Height.V {
		t.Errorf("body height = %g, line height = %g", body.Base().Height.V, line.Base().Height.V)
	}
}

func TestVerticalAlign6(t *testing.T) {
	// sup and sub currently mean +/- 0.5 em. With the initial 16px font-size,
	// that is 8px.
	d := renderPage(t, `
      <span style="line-height: 10px">
        <img src="pattern.png" style="width: 60px"
        ><img src="pattern.png" style="width: 40px; vertical-align: super"
        ><img src="pattern.png" style="width: 40px; vertical-align: sub"
      ></span>`)
	body := d.body(t)
	line := kids(t, body, 1)[0]
	imgs := kids(t, kids(t, line, 1)[0], 3)
	for i, want := range []struct{ height, y float64 }{{60, 0}, {40, 12}, {40, 28}} {
		img := imgs[i].Base()
		if img.Height.V != want.height || img.PositionY != want.y {
			t.Errorf("image %d: height %g at y %g, want %g at %g", i+1, img.Height.V, img.PositionY, want.height, want.y)
		}
	}
	if line.Base().Height.V != 68 || body.Base().Height.V != 68 {
		t.Errorf("heights of the line and body = %g, %g, want 68", line.Base().Height.V, body.Base().Height.V)
	}
}

func TestVerticalAlign7(t *testing.T) {
	d := renderPage(t, `
      <body style="line-height: 10px">
        <span>
          <img src="pattern.png" style="vertical-align: text-top"
          ><img src="pattern.png" style="vertical-align: text-bottom"
        ></span>`)
	body := d.body(t)
	line := kids(t, body, 1)[0]
	imgs := kids(t, kids(t, line, 1)[0], 2)
	img1, img2 := imgs[0].Base(), imgs[1].Base()
	if img1.Height.V != 4 || img2.Height.V != 4 {
		t.Errorf("heights = %g, %g, want 4", img1.Height.V, img2.Height.V)
	}
	if img1.PositionY != 0 || img2.PositionY != 12 { // 16 - 4
		t.Errorf("positions = %g, %g, want 0, 12", img1.PositionY, img2.PositionY)
	}
	if line.Base().Height.V != 16 || body.Base().Height.V != 16 {
		t.Errorf("heights of the line and body = %g, %g, want 16", line.Base().Height.V, body.Base().Height.V)
	}
}

func TestVerticalAlign8(t *testing.T) {
	// This case used to cause an exception: the second span has no children
	// but counts for the line height since it has padding.
	d := renderPage(t, `<span style="line-height: 1.5">
      <span style="padding: 1px"></span></span>`)
	line := kids(t, d.body(t), 1)[0]
	span1 := kids(t, line, 1)[0]
	span2 := kids(t, span1, 1)[0]
	if span1.Base().Height.V != 16 || span2.Base().Height.V != 16 {
		t.Errorf("heights = %g, %g, want 16", span1.Base().Height.V, span2.Base().Height.V)
	}
	// The strut of the line does not have line-height: normal, but the
	// result must be smaller than 1.5.
	if span1.MarginHeight() != 24 || span2.MarginHeight() != 24 || line.Base().Height.V != 24 {
		t.Errorf("margin heights = %g, %g and line height = %g, want 24",
			span1.MarginHeight(), span2.MarginHeight(), line.Base().Height.V)
	}
}

func TestVerticalAlign10(t *testing.T) {
	// The first two images bring the top of the line box 30px above the
	// baseline and 10px below.
	d := renderPage(t, `
      <span style="font-size: 0">
        <img src="pattern.png" style="vertical-align: 26px">
        <img src="pattern.png" style="vertical-align: -10px">
        <span style="vertical-align: top">
          <img src="pattern.png" style="vertical-align: -10px">
          <span style="vertical-align: -10px">
            <img src="pattern.png" style="vertical-align: bottom">
          </span>
        </span>
        <span style="vertical-align: bottom">
          <img src="pattern.png" style="vertical-align: 6px">
        </span>
      </span>`)
	body := d.body(t)
	line := kids(t, body, 1)[0]
	span1 := kids(t, line, 1)[0]
	children := kids(t, span1, 4)
	img1, img2, span2, span4 := children[0], children[1], children[2], children[3]
	span2Children := kids(t, span2, 2)
	img3, span3 := span2Children[0], span2Children[1]
	img4 := kids(t, span3, 1)[0]
	img5 := kids(t, span4, 1)[0]
	if body.Base().Height.V != line.Base().Height.V || line.Base().Height.V != 40 {
		t.Errorf("heights of the body and line = %g, %g, want 40", body.Base().Height.V, line.Base().Height.V)
	}
	for name, want := range map[string]struct {
		box boxes.Box
		y   float64
	}{
		"img 1": {img1, 0}, "img 2": {img2, 36}, "img 3": {img3, 6}, "img 4": {img4, 36}, "img 5": {img5, 30},
	} {
		if got := want.box.Base().PositionY; got != want.y {
			t.Errorf("%s y = %g, want %g", name, got, want.y)
		}
	}
}

func TestVerticalAlign11(t *testing.T) {
	d := renderPage(t, `
      <span style="font-size: 0">
        <img src="pattern.png" style="vertical-align: bottom">
        <img src="pattern.png" style="vertical-align: top; height: 100px">
      </span>
    `)
	imgs := kids(t, at(t, d.body(t), 0, 0), 2)
	if y := imgs[0].Base().PositionY; y != 96 {
		t.Errorf("first image y = %g, want 96", y)
	}
	if y := imgs[1].Base().PositionY; y != 0 {
		t.Errorf("second image y = %g, want 0", y)
	}
}

func TestVerticalAlign12(t *testing.T) {
	// The baseline for the next test.
	d := renderPage(t, `
      <span style="font-size: 0; vertical-align: top">
        <img src="pattern.png">
      </span>
    `)
	img := at(t, d.body(t), 0, 0, 0)
	if y := img.Base().PositionY; y != 0 {
		t.Errorf("image y = %g, want 0", y)
	}
}

func TestBoxDecorationBreakInlineSlice(t *testing.T) {
	// https://www.w3.org/TR/css-backgrounds-3/#the-box-decoration-break
	d := renderPage(t, `
      <style>
        @page { size: 100px }
        span { font-family: testfont; box-decoration-break: slice;
               padding: 5px; border: 1px solid black }
      </style>
      <span>a<br/>b<br/>c</span>`)
	lines := kids(t, d.body(t), 3)

	span := kids(t, lines[0], 1)[0]
	if span.Base().Width.V != 16 || span.MarginWidth() != 16+5+1 {
		t.Errorf("span 1: width %g, margin width %g", span.Base().Width.V, span.MarginWidth())
	}
	if x := kids(t, span, 2)[0].Base().PositionX; x != 5+1 {
		t.Errorf("text 1 x = %g, want 6", x)
	}

	span = kids(t, lines[1], 1)[0]
	if span.Base().Width.V != 16 || span.MarginWidth() != 16 {
		t.Errorf("span 2: width %g, margin width %g", span.Base().Width.V, span.MarginWidth())
	}
	if x := kids(t, span, 2)[0].Base().PositionX; x != 0 {
		t.Errorf("text 2 x = %g, want 0", x)
	}

	span = kids(t, lines[2], 1)[0]
	if span.Base().Width.V != 16 || span.MarginWidth() != 16+5+1 {
		t.Errorf("span 3: width %g, margin width %g", span.Base().Width.V, span.MarginWidth())
	}
	if x := kids(t, span, 1)[0].Base().PositionX; x != 0 {
		t.Errorf("text 3 x = %g, want 0", x)
	}
}

func TestBoxDecorationBreakInlineClone(t *testing.T) {
	// https://www.w3.org/TR/css-backgrounds-3/#the-box-decoration-break
	d := renderPage(t, `
      <style>
        @page { size: 100px }
        span { font-size: 12pt; font-family: testfont;
               box-decoration-break: clone;
               padding: 5px; border: 1px solid black }
      </style>
      <span>a<br/>b<br/>c</span>`)
	lines := kids(t, d.body(t), 3)
	for i, line := range lines {
		span := kids(t, line, 1)[0]
		if span.Base().Width.V != 16 || span.MarginWidth() != 16+2*(5+1) {
			t.Errorf("span %d: width %g, margin width %g", i+1, span.Base().Width.V, span.MarginWidth())
		}
		if x := kids(t, span, len(boxes.ChildrenOf(span)))[0].Base().PositionX; x != 5+1 {
			t.Errorf("text %d x = %g, want 6", i+1, x)
		}
	}
}

func TestNestedWaitingChildrenWidth(t *testing.T) {
	d := renderPage(t, `<body style="width: 3em; font-family: testfont"><b><i style="width: 100%">a b</i>c`)
	lines := kids(t, d.body(t), 2)
	if got := textAt(t, lines[0], 0, 0, 0); got != "a" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 0, 0); got != "b" {
		t.Errorf("line 2 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 1); got != "c" {
		t.Errorf("line 2 end = %q", got)
	}
}

func TestVerticalAlign9(t *testing.T) {
	d := renderPage(t, `
      <span>
        <img src="pattern.png" style="width: 40px; vertical-align: -15px"
        ><img src="pattern.png" style="width: 60px"
      ></span><div style="display: inline-block; vertical-align: 3px">
        <div>
          <div style="height: 100px">foo</div>
          <div>
            <img src="pattern.png" style="
                 width: 40px; vertical-align: -15px"
            ><img src="pattern.png" style="width: 60px"
          ></div>
        </div>
      </div>`)
	body := d.body(t)
	line := kids(t, body, 1)[0]
	children := kids(t, line, 2)
	span, div1 := children[0], children[1]
	if line.Base().Height.V != 178 || body.Base().Height.V != 178 {
		t.Errorf("heights of the line and body = %g, %g, want 178", line.Base().Height.V, body.Base().Height.V)
	}

	// Same as earlier.
	imgs := kids(t, span, 2)
	if imgs[0].Base().Height.V != 40 || imgs[1].Base().Height.V != 60 {
		t.Errorf("heights = %g, %g, want 40, 60", imgs[0].Base().Height.V, imgs[1].Base().Height.V)
	}
	if imgs[0].Base().PositionY != 138 || imgs[1].Base().PositionY != 103 {
		t.Errorf("positions = %g, %g, want 138, 103", imgs[0].Base().PositionY, imgs[1].Base().PositionY)
	}

	div2 := kids(t, div1, 1)[0]
	inner := kids(t, div2, 2)
	div3, div4 := inner[0], inner[1]
	divLine := kids(t, div4, 1)[0]
	divImgs := kids(t, divLine, 2)
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"inline-block y", div1.Base().PositionY, 0},
		{"inline-block height", div1.Base().Height.V, 175},
		{"first block height", div3.Base().Height.V, 100},
		{"line height", divLine.Base().Height.V, 75},
		{"image 1 height", divImgs[0].Base().Height.V, 40},
		{"image 2 height", divImgs[1].Base().Height.V, 60},
		{"image 1 y", divImgs[0].Base().PositionY, 135},
		{"image 2 y", divImgs[1].Base().PositionY, 100},
	} {
		if c.got != c.want {
			t.Errorf("%s = %g, want %g", c.name, c.got, c.want)
		}
	}
}

func TestVerticalAlign13(t *testing.T) {
	// Should be the same as the span in the previous test.
	d := renderPage(t, `
      <span style="font-size: 0; vertical-align: top; display: inline-block">
        <img src="pattern.png">
      </span>`)
	img := at(t, d.body(t), 0, 0, 0, 0)
	if img.Base().ElementTag != "img" || img.Base().PositionY != 0 {
		t.Errorf("%s at y = %g, want img at 0", img.Base().ElementTag, img.Base().PositionY)
	}
}

func TestFontStretch(t *testing.T) {
	d := renderPage(t, fmt.Sprintf(`
      <style>
        @font-face { font-family: stretchy; src: url(testfont.otf) }
        @font-face {
          font-family: stretchy;
          src: url(%s);
          font-stretch: condensed;
        }
        p { float: left; font-family: stretchy }
      </style>
      <p>Hello, world!</p>
      <p style="font-stretch: condensed">Hello, world!</p>
    `, condensedTestFontURL(t)))
	ps := kids(t, d.body(t), 2)
	normal, condensed := ps[0].Base().Width.V, ps[1].Base().Width.V
	if condensed >= normal {
		t.Fatalf("condensed width %g is not less than normal %g", condensed, normal)
	}
}

func TestBreakingLineboxRegression15(t *testing.T) {
	d := renderPage(t, `
      <style>
        body {font-family: testfont; font-size: 4px}
        pre {float: left}
      </style>
      <pre>ab©
déf
ghïj
klm</pre>`)
	pre := kids(t, d.body(t), 1)[0]
	lines := kids(t, pre, 4)
	for i, want := range []struct {
		text  string
		width float64
	}{{"ab©", 4 * 3}, {"déf", 4 * 3}, {"ghïj", 4 * 4}, {"klm", 4 * 3}} {
		text := at(t, lines[i], 0)
		if textOf(t, text) != want.text || text.Base().Width.V != want.width {
			t.Errorf("line %d = %q wide %g, want %q wide %g", i+1, textOf(t, text), text.Base().Width.V, want.text, want.width)
		}
	}
	if w := pre.Base().Width.V; w != 4*4 {
		t.Errorf("pre width = %g, want 16", w)
	}
}

func TestBreakingLineboxRegression16(t *testing.T) {
	d := renderPage(t, `
      <style>
        body {font-family: testfont; font-size: 4px}
        p {float: left}
      </style>
      <p>tést</p>
      <pre>ab©
déf
ghïj
klm</pre>`)
	children := kids(t, d.body(t), 2)
	p, pre := children[0], children[1]
	if got := textAt(t, kids(t, p, 1)[0], 0); got != "tést" {
		t.Errorf("paragraph text = %q", got)
	}
	if w := p.Base().Width.V; w != 4*4 {
		t.Errorf("paragraph width = %g, want 16", w)
	}
	lines := kids(t, pre, 4)
	for i, want := range []struct {
		text  string
		width float64
	}{{"ab©", 4 * 3}, {"déf", 4 * 3}, {"ghïj", 4 * 4}, {"klm", 4 * 3}} {
		text := at(t, lines[i], 0)
		if textOf(t, text) != want.text || text.Base().Width.V != want.width {
			t.Errorf("line %d = %q wide %g, want %q wide %g", i+1, textOf(t, text), text.Base().Width.V, want.text, want.width)
		}
	}
}

func TestBidiPositionXInvariant(t *testing.T) {
	d := renderPage(t, `
      <style>
        .float-border {
          float: left;
          border-right: 100px solid black;
        }
      </style>
      <div class="float-border" style="direction: ltr">abc</div>
      <div>&nbsp;</div>
      <div class="float-border" style="direction: rtl">abc</div>
    `)
	blocks := kids(t, d.body(t), 3)
	ltr, rtl := blocks[0], blocks[2]
	lineLTR, lineRTL := kids(t, ltr, 1)[0], kids(t, rtl, 1)[0]
	textLTR, textRTL := kids(t, lineLTR, 1)[0], kids(t, lineRTL, 1)[0]

	if ltr.Base().PositionX != rtl.Base().PositionX {
		t.Errorf("blocks at x = %g and %g", ltr.Base().PositionX, rtl.Base().PositionX)
	}
	if lineLTR.Base().PositionX != lineRTL.Base().PositionX {
		t.Errorf("lines at x = %g and %g", lineLTR.Base().PositionX, lineRTL.Base().PositionX)
	}
	if textLTR.Base().PositionX != textRTL.Base().PositionX {
		t.Errorf("texts at x = %g and %g", textLTR.Base().PositionX, textRTL.Base().PositionX)
	}
}
