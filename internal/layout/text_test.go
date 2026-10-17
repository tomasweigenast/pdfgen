package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestLineBreakingNBSP(t *testing.T) {
	d := renderPage(t, `
      <style>
        body { font-family: testfont; width: 7.5em }
      </style>
      <body>a <span>b</span> c d&nbsp;<span>ef
    `)
	lines := kids(t, d.body(t), 2)
	if got := textAt(t, lines[0], 0); got != "a " {
		t.Errorf("line 1 start = %q", got)
	}
	if got := textAt(t, lines[0], 1, 0); got != "b" {
		t.Errorf("line 1 span = %q", got)
	}
	if got := textAt(t, lines[0], 2); got != " c" {
		t.Errorf("line 1 end = %q", got)
	}
	if got := textAt(t, lines[1], 0); got != "d " {
		t.Errorf("line 2 start = %q", got)
	}
	if got := textAt(t, lines[1], 1, 0); got != "ef" {
		t.Errorf("line 2 span = %q", got)
	}
}

func TestLineBreakingTrailingInlineWidth(t *testing.T) {
	// The page is 520px wide with 20px margins.
	d := renderPage(t, `
      <style>
        @page { size: 520px 180px; margin: 20px }
        body {
          margin: 0;
          font-family: testfont;
          font-size: 16px;
          line-height: 1.25;
        }
        div {
          width: 27ch;
          text-align: right;
          border-right: 1px solid black;
        }
      </style>
      <div>
        <span>
          aaaaaaaaaa bbbbbbbbbb cccccccccc ddddd <span>e. 43</span>
          ff&nbsp;<span>ggggg</span>.
        </span>
      </div>
    `)
	div := kids(t, d.body(t), 1)[0]
	lines := kids(t, div, 3)
	span2 := kids(t, lines[1], 1)[0]
	nested2 := at(t, span2, 1)
	span3 := kids(t, lines[2], 1)[0]

	if got := textAt(t, span2, 0); got != "cccccccccc ddddd " {
		t.Errorf("text 2 = %q", got)
	}
	if got := textAt(t, nested2, 0); got != "e. 43" {
		t.Errorf("number = %q", got)
	}
	if got := textAt(t, span3, 0); got != "ff " {
		t.Errorf("text 3 = %q", got)
	}
	if got := textAt(t, at(t, span3, 1), 0); got != "ggggg" {
		t.Errorf("word = %q", got)
	}
	if got := textAt(t, span3, 2); got != "." {
		t.Errorf("dot = %q", got)
	}

	divRight := div.Base().PositionX + div.Base().Width.V
	numberRight := nested2.Base().PositionX + nested2.Base().Width.V
	spanRight := span2.Base().PositionX + span2.Base().Width.V
	if math.Abs(numberRight-divRight) > 1e-6*divRight {
		t.Errorf("number right edge = %g, want %g", numberRight, divRight)
	}
	if math.Abs(spanRight-numberRight) > 1e-6*numberRight {
		t.Errorf("span right edge = %g, want %g", spanRight, numberRight)
	}
}

func TestTextFontSizeZero(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { font-size: 0; }
      </style>
      <p>test font size zero</p>
    `)
	paragraph := kids(t, d.body(t), 1)[0]
	line := kids(t, paragraph, 1)[0]
	// Zero-sized text boxes are removed.
	if n := len(boxes.ChildrenOf(line)); n != 0 {
		t.Errorf("line has %d children, want 0", n)
	}
	if line.Base().Height.V != 0 || paragraph.Base().Height.V != 0 {
		t.Errorf("heights of the line and paragraph = %g, %g, want 0", line.Base().Height.V, paragraph.Base().Height.V)
	}
}

func TestTextFontSizeVerySmall(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { font-size: 0.00000001px }
      </style>
      <p>test font size zero</p>
    `)
	paragraph := kids(t, d.body(t), 1)[0]
	line := kids(t, paragraph, 1)[0]
	if line.Base().Height.V >= 0.001 || paragraph.Base().Height.V >= 0.001 {
		t.Errorf("heights of the line and paragraph = %g, %g, want less than 0.001",
			line.Base().Height.V, paragraph.Base().Height.V)
	}
}

func TestTextSpacedInlines(t *testing.T) {
	d := renderPage(t, `
      <p>start <i><b>bi1</b> <b>bi2</b></i> <b>b1</b> end</p>
    `)
	line := kids(t, kids(t, d.body(t), 1)[0], 1)[0]
	children := kids(t, line, 5)
	start, i, space, b, end := children[0], children[1], children[2], children[3], children[4]
	if textOf(t, start) != "start " || textOf(t, space) != " " || textOf(t, end) != " end" {
		t.Errorf("texts = %q, %q, %q", textOf(t, start), textOf(t, space), textOf(t, end))
	}
	if space.Base().Width.V <= 0 {
		t.Errorf("space width = %g, want positive", space.Base().Width.V)
	}

	inner := kids(t, i, 3)
	if got := textAt(t, inner[0], 0); got != "bi1" {
		t.Errorf("bi1 = %q", got)
	}
	if got := textOf(t, inner[1]); got != " " || inner[1].Base().Width.V <= 0 {
		t.Errorf("inner space = %q wide %g", got, inner[1].Base().Width.V)
	}
	if got := textAt(t, inner[2], 0); got != "bi2" {
		t.Errorf("bi2 = %q", got)
	}
	if got := textAt(t, b, 0); got != "b1" {
		t.Errorf("b1 = %q", got)
	}
}

func TestTextAlignLeft(t *testing.T) {
	// <-------------------->  page, body
	//     +-----+
	// +---+     |
	// |   |     |
	// +---+-----+
	//
	// ^   ^     ^          ^
	// x=0 x=40  x=100      x=200
	d := renderPage(t, `
      <style>
        @page { size: 200px }
      </style>
      <body>
        <img src="pattern.png" style="width: 40px"
        ><img src="pattern.png" style="width: 60px">`)
	imgs := kids(t, kids(t, d.body(t), 1)[0], 2)
	// The initial value of text-align is left in ltr text.
	if x := imgs[0].Base().PositionX; x != 0 {
		t.Errorf("image 1 x = %g, want 0", x)
	}
	if x := imgs[1].Base().PositionX; x != 40 {
		t.Errorf("image 2 x = %g, want 40", x)
	}
}

func TestTextAlignRight(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 200px }
        body { text-align: right }
      </style>
      <body>
        <img src="pattern.png" style="width: 40px"
        ><img src="pattern.png" style="width: 60px">`)
	imgs := kids(t, kids(t, d.body(t), 1)[0], 2)
	if x := imgs[0].Base().PositionX; x != 100 { // 200 - 60 - 40
		t.Errorf("image 1 x = %g, want 100", x)
	}
	if x := imgs[1].Base().PositionX; x != 140 { // 200 - 60
		t.Errorf("image 2 x = %g, want 140", x)
	}
}

func TestTextAlignCenter(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 200px }
        body { text-align: center }
      </style>
      <body>
        <img src="pattern.png" style="width: 40px"
        ><img src="pattern.png" style="width: 60px">`)
	imgs := kids(t, kids(t, d.body(t), 1)[0], 2)
	if x := imgs[0].Base().PositionX; x != 50 {
		t.Errorf("image 1 x = %g, want 50", x)
	}
	if x := imgs[1].Base().PositionX; x != 90 {
		t.Errorf("image 2 x = %g, want 90", x)
	}
}

// justified describes the boxes of the lines of the justification tests.
type justified struct {
	box  boxes.Box
	x    float64
	text string
}

func checkPositions(t *testing.T, expected map[string]justified) {
	t.Helper()
	for name, want := range expected {
		if got := want.box.Base().PositionX; got != want.x {
			t.Errorf("%s x = %g, want %g", name, got, want.x)
		}
		if want.text != "" {
			if got := textOf(t, want.box); got != want.text {
				t.Errorf("%s text = %q, want %q", name, got, want.text)
			}
		}
	}
}

func TestTextAlignJustify(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 300px 1000px }
        body { text-align: justify }
      </style>
      <p><img src="pattern.png" style="width: 40px">
        <strong>
          <img src="pattern.png" style="width: 60px">
          <img src="pattern.png" style="width: 10px">
          <img src="pattern.png" style="width: 100px"
        ></strong><img src="pattern.png" style="width: 290px"
        ><!-- Last image will be on its own line. -->`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	line1 := kids(t, lines[0], 3)
	image1, space1, strong := line1[0], line1[1], line1[2]
	inner := kids(t, strong, 5)
	image2, space2, image3, space3, image4 := inner[0], inner[1], inner[2], inner[3], inner[4]
	image5 := kids(t, lines[1], 1)[0]

	checkPositions(t, map[string]justified{
		"image 1": {image1, 0, ""},
		"space 1": {space1, 40, " "},
		"strong":  {strong, 70, ""},
		"image 2": {image2, 70, ""},
		"space 2": {space2, 130, " "},
		"image 3": {image3, 160, ""},
		"space 3": {space3, 170, " "},
		"image 4": {image4, 200, ""},
		"image 5": {image5, 0, ""},
	})
	if w := strong.Base().Width.V; w != 230 {
		t.Errorf("strong width = %g, want 230", w)
	}
}

func TestTextAlignJustifyAll(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 300px 1000px }
        body { text-align: justify-all }
      </style>
      <p><img src="pattern.png" style="width: 40px">
        <strong>
          <img src="pattern.png" style="width: 60px">
          <img src="pattern.png" style="width: 10px">
          <img src="pattern.png" style="width: 100px"
        ></strong><img src="pattern.png" style="width: 200px">
        <img src="pattern.png" style="width: 10px">`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	line1 := kids(t, lines[0], 3)
	image1, space1, strong := line1[0], line1[1], line1[2]
	inner := kids(t, strong, 5)
	image2, space2, image3, space3, image4 := inner[0], inner[1], inner[2], inner[3], inner[4]
	line2 := kids(t, lines[1], 3)
	image5, space4, image6 := line2[0], line2[1], line2[2]

	checkPositions(t, map[string]justified{
		"image 1": {image1, 0, ""},
		"space 1": {space1, 40, " "},
		"strong":  {strong, 70, ""},
		"image 2": {image2, 70, ""},
		"space 2": {space2, 130, " "},
		"image 3": {image3, 160, ""},
		"space 3": {space3, 170, " "},
		"image 4": {image4, 200, ""},
		"image 5": {image5, 0, ""},
		"space 4": {space4, 200, " "},
		"image 6": {image6, 290, ""},
	})
	if w := strong.Base().Width.V; w != 230 {
		t.Errorf("strong width = %g, want 230", w)
	}
}

func TestTextAlignAllLast(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 300px 1000px }
        body { text-align-all: justify; text-align-last: right }
      </style>
      <p><img src="pattern.png" style="width: 40px">
        <strong>
          <img src="pattern.png" style="width: 60px">
          <img src="pattern.png" style="width: 10px">
          <img src="pattern.png" style="width: 100px"
        ></strong><img src="pattern.png" style="width: 200px"
        ><img src="pattern.png" style="width: 10px">`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	line1 := kids(t, lines[0], 3)
	image1, space1, strong := line1[0], line1[1], line1[2]
	inner := kids(t, strong, 5)
	image2, space2, image3, space3, image4 := inner[0], inner[1], inner[2], inner[3], inner[4]
	line2 := kids(t, lines[1], 2)
	image5, image6 := line2[0], line2[1]

	checkPositions(t, map[string]justified{
		"image 1": {image1, 0, ""},
		"space 1": {space1, 40, " "},
		"strong":  {strong, 70, ""},
		"image 2": {image2, 70, ""},
		"space 2": {space2, 130, " "},
		"image 3": {image3, 160, ""},
		"space 3": {space3, 170, " "},
		"image 4": {image4, 200, ""},
		"image 5": {image5, 90, ""},
		"image 6": {image6, 290, ""},
	})
	if w := strong.Base().Width.V; w != 230 {
		t.Errorf("strong width = %g, want 230", w)
	}
}

func TestTextAlignNotEnoughSpace(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { text-align: center; width: 0 }
        span { display: inline-block }
      </style>
      <p><span>aaaaaaaaaaaaaaaaaaaaaaaaaa</span></p>`)
	span := at(t, d.body(t), 0, 0)
	if x := span.Base().PositionX; x != 0 {
		t.Errorf("span x = %g, want 0", x)
	}
}

func TestTextAlignJustifyNoSpace(t *testing.T) {
	// A line with a single word has no space to widen.
	d := renderPage(t, `
      <style>
        body { text-align: justify; width: 50px }
      </style>
      <p>Supercalifragilisticexpialidocious bar</p>
    `)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	text := kids(t, lines[0], 1)[0]
	if x := text.Base().PositionX; x != 0 {
		t.Errorf("text x = %g, want 0", x)
	}
}

func TestTextAlignJustifyTextIndent(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page { size: 300px 1000px }
        body { text-align: justify }
        p { text-indent: 3px }
      </style>
      <p><img src="pattern.png" style="width: 40px">
        <strong>
          <img src="pattern.png" style="width: 60px">
          <img src="pattern.png" style="width: 10px">
          <img src="pattern.png" style="width: 100px"
        ></strong><img src="pattern.png" style="width: 290px"
        ><!-- Last image will be on its own line. -->`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	line1 := kids(t, lines[0], 3)
	image1, space1, strong := line1[0], line1[1], line1[2]
	inner := kids(t, strong, 5)
	image2, space2, image3, space3, image4 := inner[0], inner[1], inner[2], inner[3], inner[4]
	image5 := kids(t, lines[1], 1)[0]

	checkPositions(t, map[string]justified{
		"image 1": {image1, 3, ""},
		"space 1": {space1, 43, " "},
		"strong":  {strong, 72, ""},
		"image 2": {image2, 72, ""},
		"space 2": {space2, 132, " "},
		"image 3": {image3, 161, ""},
		"space 3": {space3, 171, " "},
		"image 4": {image4, 200, ""},
		"image 5": {image5, 0, ""},
	})
	if w := strong.Base().Width.V; w != 228 {
		t.Errorf("strong width = %g, want 228", w)
	}
}

func TestTextAlignJustifyNoBreakBetweenChildren(t *testing.T) {
	// Justification when the line breaks between two inline children that
	// must stay together.
	d := renderPage(t, `
      <style>
        p { text-align: justify; font-family: testfont; width: 7em }
      </style>
      <p>
        <span>a</span>
        <span>b</span>
        <span>bla</span><span>,</span>
        <span>b</span>
      </p>
    `)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)

	line1 := kids(t, lines[0], 4)
	if x := line1[0].Base().PositionX; x != 0 {
		t.Errorf("span 1 x = %g, want 0", x)
	}
	if x := line1[2].Base().PositionX; x != 6*16 { // 1 character + 5 spaces
		t.Errorf("span 2 x = %g, want %d", x, 6*16)
	}
	if w := lines[0].Base().Width.V; w != 7*16 {
		t.Errorf("line width = %g, want %d", w, 7*16)
	}

	line2 := kids(t, lines[1], 5)
	if x := line2[0].Base().PositionX; x != 0 {
		t.Errorf("span 1 x = %g, want 0", x)
	}
	if x := line2[1].Base().PositionX; x != 3*16 { // 3 characters
		t.Errorf("span 2 x = %g, want %d", x, 3*16)
	}
	if x := line2[3].Base().PositionX; x != 5*16 { // 3 + 1 characters and a space
		t.Errorf("span 3 x = %g, want %d", x, 5*16)
	}
}

func TestWordSpacing(t *testing.T) {
	for _, text := range []string{
		"Lorem ipsum dolor<em>sit amet</em>",
		"Lorem ipsum <em>dolorsit</em> amet",
		"Lorem ipsum <em></em>dolorsit amet",
		"Lorem ipsum<em> </em>dolorsit amet",
		"Lorem ipsum<em> dolorsit</em> amet",
		"Lorem ipsum <em>dolorsit </em>amet",
	} {
		t.Run(text, func(t *testing.T) {
			d := renderPage(t, `
      <style></style> <!-- element.text is None -->
      <body><strong>Lorem ipsum dolor<em>sit amet</em></strong>`)
			strong1 := at(t, d.body(t), 0, 0)

			d = renderPage(t, `
      <style>strong { word-spacing: 11px }</style>
      <body><strong>`+text+`</strong>`)
			strong2 := at(t, d.body(t), 0, 0)

			if diff := strong2.Base().Width.V - strong1.Base().Width.V; diff != 33 {
				t.Fatalf("width grew by %g, want 33", diff)
			}
		})
	}
}

func TestLetterSpacing1(t *testing.T) {
	d := renderPage(t, `
        <body><strong>Supercalifragilisticexpialidocious</strong>`)
	strong1 := at(t, d.body(t), 0, 0)

	d = renderPage(t, `
        <style>strong { letter-spacing: 11px }</style>
        <body><strong>Supercalifragilisticexpialidocious</strong>`)
	strong2 := at(t, d.body(t), 0, 0)
	if diff := strong2.Base().Width.V - strong1.Base().Width.V; diff != 34*11 {
		t.Errorf("width grew by %g, want %d", diff, 34*11)
	}

	// An embedded tag does not affect the letter spacing of a single line.
	d = renderPage(t, `<style>strong { letter-spacing: 11px }</style>`+
		`<body><strong>Supercali<span>fragilistic</span>expialidocious</strong>`)
	strong3 := at(t, d.body(t), 0, 0)
	if strong3.Base().Width.V != strong2.Base().Width.V {
		t.Errorf("width with a span = %g, want %g", strong3.Base().Width.V, strong2.Base().Width.V)
	}

	// Wrapped lines have the same overall width. The words are inline
	// blocks so that they are not broken.
	d = renderPage(t, fmt.Sprintf(`<style>
  strong {
    letter-spacing: 11px;
    max-width: %gpx
  }
  span { display: inline-block }
</style>
<body><strong>
  <span>Supercali<i>fragilistic</i>expialidocious</span>
  <span>Supercali<i>fragilistic</i>expialidocious</span>
</strong>`, strong3.Base().Width.V*1.5))
	lines := kids(t, d.body(t), 2)
	w1 := kids(t, lines[0], 1)[0].Base().Width.V
	w2 := kids(t, lines[1], 1)[0].Base().Width.V
	if w1 != w2 || w1 != strong2.Base().Width.V {
		t.Errorf("widths = %g, %g, want %g", w1, w2, strong2.Base().Width.V)
	}
}

func TestSpacingEx(t *testing.T) {
	// Regression test.
	for _, spacing := range []string{"word-spacing", "letter-spacing"} {
		t.Run(spacing, func(t *testing.T) {
			renderPage(t, fmt.Sprintf(`<div style="%s: 2ex">abc def`, spacing))
		})
	}
}

func TestTextIndent(t *testing.T) {
	for _, indent := range []string{"12px", "6%"} {
		t.Run(indent, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
        <style>
            @page { size: 220px }
            body { margin: 10px; text-indent: %s }
        </style>
        <p>Some text that is long enough that it take at least three line,
           but maybe more.
    `, indent))
			lines := boxes.ChildrenOf(kids(t, d.body(t), 1)[0])
			if len(lines) < 3 {
				t.Fatalf("%d lines, want at least 3", len(lines))
			}
			for i, want := range []float64{22, 10, 10} { // 10px margin-left, then the 12px indent
				if x := kids(t, lines[i], 1)[0].Base().PositionX; x != want {
					t.Errorf("text %d x = %g, want %g", i+1, x, want)
				}
			}
		})
	}
}

func TestTextIndentInline(t *testing.T) {
	d := renderPage(t, `
        <style>
            p { display: inline-block; text-indent: 1em; font-family: testfont }
        </style>
        <p><span>text
    `)
	paragraph := at(t, d.body(t), 0, 0)
	line := kids(t, paragraph, 1)[0]
	if w := line.Base().Width.V; w != (4+1)*16 {
		t.Errorf("line width = %g, want %d", w, (4+1)*16)
	}
}

func TestHyphenateCharacter(t *testing.T) {
	for _, tt := range []struct {
		name, character string
		suffix          string
		strip           string
	}{
		{"exclamation mark", "!", "!", "!"},
		{"accented letter", "à", "à", "à"},
		{"several characters", "ù ù", "ù ù", "ù ù"},
		{"empty", "", "", ""},
		{"dashes", "———", "———", "—"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`<html style="width: 5em; font-family: testfont">`+
				`<body style="hyphens: auto; hyphenate-character: '%s'" lang=fr>hyphénation`, tt.character))
			lines := boxes.ChildrenOf(d.body(t))
			if len(lines) < 2 {
				t.Fatalf("%d lines, want more than one", len(lines))
			}
			if first := textAt(t, lines[0], 0); !strings.HasSuffix(first, tt.suffix) {
				t.Errorf("first line %q does not end with %q", first, tt.suffix)
			}
			var full strings.Builder
			for _, line := range lines {
				full.WriteString(textAt(t, line, 0))
			}
			got := strings.ReplaceAll(full.String(), tt.strip, "")
			if tt.name == "several characters" {
				got = strings.ReplaceAll(strings.ReplaceAll(full.String(), " ", ""), "ù", "")
			}
			if got != "hyphénation" {
				t.Errorf("text = %q, want hyphénation", got)
			}
		})
	}
}

// joinedText concatenates the text of the children of lines.
func joinedText(t *testing.T, lines []boxes.Box) string {
	t.Helper()
	var full strings.Builder
	for _, line := range lines {
		for _, child := range boxes.ChildrenOf(line) {
			full.WriteString(textOf(t, child))
		}
	}
	return full.String()
}

func TestHyphenateManual1(t *testing.T) {
	const hyphenated = "hyphénation"
	runes := []rune(hyphenated)
	for i := 1; i < len(runes); i++ {
		for _, character := range []string{"!", "ù ù"} {
			t.Run(fmt.Sprintf("%d %q", i, character), func(t *testing.T) {
				word := string(runes[:i]) + "\u00ad" + string(runes[i:])
				d := renderPage(t, `<html style="width: 5em; font-family: testfont">`+
					`<body style="hyphens: manual;  hyphenate-character: '`+character+`'"`+
					`  lang=fr>`+word)
				lines := kids(t, d.body(t), 2)
				if first := textAt(t, lines[0], 0); !strings.HasSuffix(first, character) {
					t.Errorf("first line %q does not end with %q", first, character)
				}
				if got := strings.ReplaceAll(joinedText(t, lines), character, ""); got != word {
					t.Errorf("text = %q, want %q", got, word)
				}
			})
		}
	}
}

func TestHyphenateManual2(t *testing.T) {
	const hyphenated = "hy phénation"
	runes := []rune(hyphenated)
	for i := 1; i < len(runes); i++ {
		for _, character := range []string{"!", "ù ù"} {
			t.Run(fmt.Sprintf("%d %q", i, character), func(t *testing.T) {
				word := string(runes[:i]) + "\u00ad" + string(runes[i:])
				d := renderPage(t, `<html style="width: 5em; font-family: testfont">`+
					`<body style="hyphens: manual;  hyphenate-character: '`+character+`'"`+
					`  lang=fr>`+word)
				lines := boxes.ChildrenOf(d.body(t))
				if len(lines) != 2 && len(lines) != 3 {
					t.Fatalf("%d lines, want 2 or 3", len(lines))
				}
				full := strings.ReplaceAll(joinedText(t, lines), character, "")
				first := textAt(t, lines[0], 0)
				if strings.HasSuffix(first, character) {
					if full != word {
						t.Errorf("text = %q, want %q", full, word)
					}
					return
				}
				if !strings.HasSuffix(strings.TrimRight(first, "\u00ad"), "y") {
					t.Errorf("first line %q does not end with y", first)
				}
				if len(lines) == 3 {
					if second := textAt(t, lines[1], 0); !strings.HasSuffix(strings.TrimRight(second, "\u00ad"), character) {
						t.Errorf("second line %q does not end with %q", second, character)
					}
				}
			})
		}
	}
}

func TestHyphenateManual3(t *testing.T) {
	// Automatic hyphenation opportunities within a word are ignored if the
	// word contains a conditional hyphen, in favor of the conditional
	// hyphens.
	d := renderPage(t, `<html style="width: 0.1em" lang="en">`+
		`<body style="hyphens: auto">in&shy;lighten&shy;lighten&shy;in`)
	lines := kids(t, d.body(t), 4)
	for i, want := range []string{"in\u00ad‐", "lighten\u00ad‐", "lighten\u00ad‐", "in"} {
		if got := textAt(t, lines[i], 0); got != want {
			t.Errorf("line %d = %q, want %q", i+1, got, want)
		}
	}
}

func TestHyphenateManual4(t *testing.T) {
	// The line should not end with a hyphen, but
	// that is not checked.
	d := renderPage(t, `<html style="width: 0.1em" lang="en">`+
		`<body style="hyphens: auto">test&shy;`)
	kids(t, d.body(t), 1)
}

func TestHyphenateManualKeepHyphen(t *testing.T) {
	d := renderPage(t, `<html style="width: 6em; font-family: testfont">`+
		`<body style="hyphens: manual">super&shy;aa bb`)
	lines := boxes.ChildrenOf(d.body(t))
	if got := textAt(t, lines[0], 0); got != "super\u00ad‐" {
		t.Errorf("first line = %q", got)
	}
}

func TestHyphenateLimitZone(t *testing.T) {
	for _, tt := range []struct {
		name, zone string
		hyphenated bool
	}{
		{"zero", "0", true},
		{"nine em", "9em", false},
		{"five percent", "5%", true},
		{"ninety-five percent", "95%", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := renderPage(t, `<html style="width: 12em; font-family: testfont">`+
				`<body style="hyphens: auto;hyphenate-limit-zone: `+tt.zone+`" lang=fr>mmmmm hyphénation`)
			lines := boxes.ChildrenOf(d.body(t))
			first := textAt(t, lines[0], 0)
			var full strings.Builder
			for _, line := range lines {
				full.WriteString(textAt(t, line, 0))
			}
			if tt.hyphenated {
				if len(lines) != 2 || !strings.HasSuffix(first, "‐") {
					t.Errorf("%d lines, the first %q", len(lines), first)
				}
				if got := strings.ReplaceAll(full.String(), "‐", ""); got != "mmmmm hyphénation" {
					t.Errorf("text = %q", got)
				}
				return
			}
			if len(lines) < 2 || !strings.HasSuffix(first, "mm") {
				t.Errorf("%d lines, the first %q", len(lines), first)
			}
			if got := full.String(); got != "mmmmmhyphénation" {
				t.Errorf("text = %q", got)
			}
		})
	}
}

func TestHyphenNBSP(t *testing.T) {
	d := renderPage(t, `
      <style>
        body {width: 20px; font: 2px testfont}
      </style>
      <body style="hyphens: auto;" lang="en">
        <span>this&nbsp;hyphenation
    `)
	lines := kids(t, d.body(t), 2)
	if got := textAt(t, lines[0], 0, 0); got != "this hy‐" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0, 0); got != "phenation" {
		t.Errorf("line 2 = %q", got)
	}
}

func TestHyphenateLimitChars(t *testing.T) {
	for _, tt := range []struct {
		css   string
		lines int
	}{
		{"auto", 2},
		{"auto auto 0", 2},
		{"0 0 0", 2},
		{"4 4 auto", 1},
		{"6 2 4", 2},
		{"auto 1 auto", 2},
		{"7 auto auto", 1},
		{"6 auto auto", 2},
		{"5 2", 2},
		{"3", 2},
		{"2 4 6", 1},
		{"auto 4", 1},
		{"auto 2", 2},
	} {
		t.Run(tt.css, func(t *testing.T) {
			d := renderPage(t, `<html style="width: 1em; font-family: testfont">`+
				`<body style="hyphens: auto;hyphenate-limit-chars: `+tt.css+`" lang=en>hyphen`)
			if lines := boxes.ChildrenOf(d.body(t)); len(lines) != tt.lines {
				t.Fatalf("%d lines, want %d", len(lines), tt.lines)
			}
		})
	}
}

func TestHyphenateLimitCharsPunctuation(t *testing.T) {
	// The word is light·en.
	for _, css := range []string{
		"3 3 3", // en is shorter than 3
		"3 6 2", // light is shorter than 6
		"8",     // lighten is shorter than 8
	} {
		t.Run(css, func(t *testing.T) {
			d := renderPage(t, `<html style="width: 1em; font-family: testfont">`+
				`<body style="hyphens: auto;hyphenate-limit-chars: `+css+`" lang=en>..lighten..`)
			if lines := boxes.ChildrenOf(d.body(t)); len(lines) != 1 {
				t.Fatalf("%d lines, want 1", len(lines))
			}
		})
	}
}

func TestOverflowWrap(t *testing.T) {
	for _, tt := range []struct {
		wrap, text string
		lines      func(int) bool
		fullText   string
	}{
		{"anywhere", "aaaaaaaa", func(n int) bool { return n > 1 }, "aaaaaaaa"},
		{"break-word", "aaaaaaaa", func(n int) bool { return n > 1 }, "aaaaaaaa"},
		{"normal", "aaaaaaaa", func(n int) bool { return n == 1 }, "aaaaaaaa"},
		{"break-word", "hyphenations", func(n int) bool { return n > 3 }, "hy‐phen‐ations"},
		{
			"break-word", "A splitted word.  An hyphenated word.", func(n int) bool { return n > 8 },
			"Asplittedword.Anhy‐phen‐atedword.",
		},
	} {
		t.Run(tt.wrap+" "+tt.text, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
      <style>
        body {width: 80px; overflow: hidden; font-family: testfont}
        span {overflow-wrap: %s}
      </style>
      <body style="hyphens: auto;" lang="en">
        <span>%s
    `, tt.wrap, tt.text))
			var full strings.Builder
			lines := boxes.ChildrenOf(d.body(t))
			for _, line := range lines {
				full.WriteString(textAt(t, kids(t, line, 1)[0], 0))
			}
			if !tt.lines(len(lines)) {
				t.Errorf("%d lines", len(lines))
			}
			if full.String() != tt.fullText {
				t.Errorf("text = %q, want %q", full.String(), tt.fullText)
			}
		})
	}
}

func TestWrapOverflowWordBreak(t *testing.T) {
	// overflow-wrap: anywhere and break-word are only allowed to break a word
	// if there are no otherwise-acceptable break points in the line, which
	// means they do not split a word if it fits cleanly into the next line.
	// This can be done accidentally if it is in its own inline element.
	for _, tt := range []struct {
		css   string
		lines []string
	}{
		{"overflow-wrap: anywhere", []string{"aaa", "bbb"}},
		{"overflow-wrap: break-word", []string{"aaa", "bbb"}},
		// On the other hand, word-break: break-all mandates a break anywhere
		// at the end of a line, even if the word could fit cleanly onto the
		// next line.
		{"word-break: break-all", []string{"aaa b", "bb"}},
	} {
		t.Run(tt.css, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
      <style>
        body {width: 80px; overflow: hidden; font-family: testfont}
        span {%s}
      </style>
      <body>
        <span>aaa </span><span>bbb
    `, tt.css))
			var lines []string
			for _, line := range boxes.ChildrenOf(d.body(t)) {
				var text string
				for _, span := range boxes.ChildrenOf(line) {
					text += textAt(t, span, 0)
				}
				lines = append(lines, text)
			}
			if strings.Join(lines, "|") != strings.Join(tt.lines, "|") {
				t.Fatalf("lines = %q, want %q", lines, tt.lines)
			}
		})
	}
}

func TestOverflowWrapNoBreakOnSpace(t *testing.T) {
	d := renderPage(t, `
      <style>
        body {width: 11px; font-family: testfont; font-size: 2px;
              overflow-wrap: anywhere}
      </style>.jpg, .png`)
	lines := kids(t, d.body(t), 2)
	if got := textAt(t, lines[0], 0); got != ".jpg," {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0); got != ".png" {
		t.Errorf("line 2 = %q", got)
	}
}

func TestLineBreakBeforeTrailingSpace(t *testing.T) {
	d := renderPage(t, `
        <p style="display: inline-block">test`+" "+` </p>a
        <p style="display: inline-block">test`+" "+`</p>a
    `)
	children := kids(t, kids(t, d.body(t), 1)[0], 4)
	if w1, w2 := children[0].Base().Width.V, children[2].Base().Width.V; w1 != w2 {
		t.Errorf("widths = %g, %g, want equal", w1, w2)
	}
}

func TestLineBreakBeforeNestedTrailingSpace(t *testing.T) {
	d := renderPage(t, `
    <div style="font: 2px testfont; width: 6px">
      AA <span><b><i>BBB</i></b> </span></div>
    `)
	kids(t, kids(t, d.body(t), 1)[0], 2)
}

func whiteSpaceLines(t *testing.T, width int, space string) []boxes.Box {
	t.Helper()
	d := renderPage(t, fmt.Sprintf(`
      <style>
        body { font-size: 100px; width: %dpx }
        span { white-space: %s }
      </style>
      <body><span>This +    
    is text`, width, space))
	return boxes.ChildrenOf(d.body(t))
}

func TestWhiteSpace(t *testing.T) {
	for _, tt := range []struct {
		width int
		space string
		lines []string
	}{
		{1, "normal", []string{"This", "+", "is", "text"}},
		{1, "pre", []string{"This +    ", "    is text"}},
		{1, "nowrap", []string{"This + is text"}},
		{1, "pre-wrap", []string{"This ", "+    ", "    ", "is ", "text"}},
		{1, "pre-line", []string{"This", "+", "is", "text"}},
		{1000000, "normal", []string{"This + is text"}},
		{1000000, "pre", []string{"This +    ", "    is text"}},
		{1000000, "nowrap", []string{"This + is text"}},
		{1000000, "pre-wrap", []string{"This +    ", "    is text"}},
		{1000000, "pre-line", []string{"This +", "is text"}},
	} {
		t.Run(fmt.Sprintf("%s at %d", tt.space, tt.width), func(t *testing.T) {
			lines := whiteSpaceLines(t, tt.width, tt.space)
			if len(lines) != len(tt.lines) {
				t.Fatalf("%d lines, want %d", len(lines), len(tt.lines))
			}
			for i, want := range tt.lines {
				if got := textAt(t, lines[i], 0, 0); got != want {
					t.Errorf("line %d = %q, want %q", i+1, got, want)
				}
			}
		})
	}
}

func TestWhiteSpace11(t *testing.T) {
	d := renderPage(t, `
      <style>
        pre { width: 0 }
      </style>
      <body><pre>This<br/>is text`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	line1 := kids(t, lines[0], 2)
	if got := textOf(t, line1[0]); got != "This" || line1[1].Base().ElementTag != "br" {
		t.Errorf("line 1 = %q and %s", got, line1[1].Base().ElementTag)
	}
	if got := textAt(t, lines[1], 0); got != "is text" {
		t.Errorf("line 2 = %q", got)
	}
}

func TestWhiteSpace12(t *testing.T) {
	d := renderPage(t, `
      <style>
        pre { width: 0 }
      </style>
      <body><pre>This is <span>lol</span> text`)
	line := kids(t, kids(t, d.body(t), 1)[0], 1)[0]
	children := kids(t, line, 3)
	if got := textOf(t, children[0]); got != "This is " {
		t.Errorf("text 1 = %q", got)
	}
	if children[1].Base().ElementTag != "span" {
		t.Errorf("middle box = %s", children[1].Base().ElementTag)
	}
	if got := textOf(t, children[2]); got != " text" {
		t.Errorf("text 2 = %q", got)
	}
}

func TestTabSize(t *testing.T) {
	for _, tt := range []struct {
		value string
		width float64
	}{
		{"8", 144},   // (2 + (8 - 1)) * 16
		{"4", 80},    // (2 + (4 - 1)) * 16
		{"3em", 64},  // (2 + (3 - 1)) * 16
		{"25px", 41}, // 2 * 16 + 25 - 1 * 16
	} {
		t.Run(tt.value, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
      <style>
        pre { tab-size: %s; font-family: testfont }
      </style>
      <pre>a&#9;a</pre>
    `, tt.value))
			line := kids(t, kids(t, d.body(t), 1)[0], 1)[0]
			if w := line.Base().Width.V; w != tt.width {
				t.Fatalf("line width = %g, want %g", w, tt.width)
			}
		})
	}
}

func TestTextTransform(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { text-transform: capitalize }
        p+p { text-transform: uppercase }
        p+p+p { text-transform: lowercase }
        p+p+p+p { text-transform: full-width }
        p+p+p+p+p { text-transform: none }
      </style>
<p>hé lO1</p><p>hé lO1</p><p>hé lO1</p><p>hé lO1</p><p>hé lO1</p>
    `)
	paragraphs := kids(t, d.body(t), 5)
	for i, want := range []string{"Hé LO1", "HÉ LO1", "hé lo1", "ｈé　ｌＯ１", "hé lO1"} {
		if got := textAt(t, paragraphs[i], 0, 0); got != want {
			t.Errorf("paragraph %d = %q, want %q", i+1, got, want)
		}
	}
}

func TestTextTransformLangUppercase(t *testing.T) {
	for _, tt := range []struct{ lang, lowercase, uppercase string }{
		{"az", "ı i", "I İ"},
		{"tr", "ı i", "I İ"},
		{"el", "καλημέρα αύριο θεϊκό ευφυΐα Νεράιδα", "ΚΑΛΗΜΕΡΑ ΑΥΡΙΟ ΘΕΪΚΟ ΕΥΦΥΪΑ ΝΕΡΑΪΔΑ"},
	} {
		t.Run(tt.lang, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
      <html lang="%s">
        <p style="text-transform: uppercase">%s
    `, tt.lang, tt.lowercase))
			if got := textAt(t, d.body(t), 0, 0, 0); got != tt.uppercase {
				t.Fatalf("text = %q, want %q", got, tt.uppercase)
			}
		})
	}
}

func TestTextTransformLangLowercase(t *testing.T) {
	for _, tt := range []struct{ lang, uppercase, lowercase string }{
		{"az", "İ I", "i ı"},
		{"tr", "İ I", "i ı"},
		{"lt", "Ì Í Ĩ", "i̇̀ i̇́ i̇̃"},
	} {
		t.Run(tt.lang, func(t *testing.T) {
			d := renderPage(t, fmt.Sprintf(`
      <html lang="%s">
        <p style="text-transform: lowercase">%s
    `, tt.lang, tt.uppercase))
			if got := textAt(t, d.body(t), 0, 0, 0); got != tt.lowercase {
				t.Fatalf("text = %q, want %q", got, tt.lowercase)
			}
		})
	}
}

func TestFirstLetterLineHeight(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { font-family: testfont }
        p:first-letter { line-height: 1; font-size: 20px }
      </style>
      <p>abc
    `)
	line := kids(t, kids(t, d.body(t), 1)[0], 1)[0]
	letter := kids(t, line, 2)[0]
	text := kids(t, letter, 1)[0]
	if got := textOf(t, text); got != "a" {
		t.Errorf("first letter = %q", got)
	}
	if h := text.Base().Height.V; h != 20 {
		t.Errorf("first letter height = %g, want 20", h)
	}
}

func TestFirstLetterTextTransform(t *testing.T) {
	d := renderPage(t, `
      <style>
        p::first-letter { text-transform: uppercase }
      </style>
      <p>abc
    `)
	line := kids(t, kids(t, d.body(t), 1)[0], 1)[0]
	children := kids(t, line, 2)
	if got := textAt(t, children[0], 0); got != "A" {
		t.Errorf("first letter = %q", got)
	}
	if got := textOf(t, children[1]); got != "bc" {
		t.Errorf("next text = %q", got)
	}
}

func TestFirstLetterNested(t *testing.T) {
	d := renderPage(t, `
      <style>
        body::first-letter { font-style: italic }
        p::first-letter { font-weight: 700 }
      </style>
      <p>abc</p>
      <p>abc</p>
    `)
	paragraphs := kids(t, d.body(t), 2)
	for i, want := range []struct {
		style  string
		weight float64
	}{{"italic", 700}, {"normal", 700}} {
		children := kids(t, kids(t, paragraphs[i], 1)[0], 2)
		letter, next := at(t, children[0], 0).Base().Style, children[1].Base().Style
		if got := string(letter.FontStyle()); got != want.style {
			t.Errorf("paragraph %d first letter style = %s, want %s", i+1, got, want.style)
		}
		if got := float64(letter.FontWeight()); got != want.weight {
			t.Errorf("paragraph %d first letter weight = %g, want %g", i+1, got, want.weight)
		}
		if got := string(next.FontStyle()); got != "normal" {
			t.Errorf("paragraph %d next text style = %s, want normal", i+1, got)
		}
		if got := float64(next.FontWeight()); got != 400 {
			t.Errorf("paragraph %d next text weight = %g, want 400", i+1, got)
		}
	}
}

func TestFirstLineLineHeight(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { font-family: testfont }
        p:first-line { line-height: 1; font-size: 20px }
      </style>
      <p>abc<br>def
    `)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	firstLine := kids(t, lines[0], 1)[0]
	children := kids(t, firstLine, 2)
	if got := textOf(t, children[0]); got != "abc" {
		t.Errorf("first line = %q", got)
	}
	if h := children[0].Base().Height.V; h != 20 {
		t.Errorf("first line text height = %g, want 20", h)
	}
}

func TestFirstLineTextTransform(t *testing.T) {
	d := renderPage(t, `
      <style>
        p::first-line { text-transform: uppercase }
      </style>
      <p>abc<br>def
    `)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	if got := textAt(t, kids(t, lines[0], 1)[0], 0); got != "ABC" {
		t.Errorf("first line = %q", got)
	}
	if got := textAt(t, lines[1], 0); got != "def" {
		t.Errorf("next line = %q", got)
	}
}

func TestFirstLineNested(t *testing.T) {
	d := renderPage(t, `
      <style>
        body::first-line { font-style: italic }
        p::first-line { font-weight: 700 }
      </style>
      <p>abc<br>def</p>
      <p>abc<br>def</p>
    `)
	paragraphs := kids(t, d.body(t), 2)
	for i, want := range []struct {
		style  string
		weight float64
	}{{"italic", 700}, {"normal", 700}} {
		lines := kids(t, paragraphs[i], 2)
		first := at(t, lines[0], 0, 0).Base().Style
		next := at(t, lines[1], 0).Base().Style
		if got := string(first.FontStyle()); got != want.style {
			t.Errorf("paragraph %d first line style = %s, want %s", i+1, got, want.style)
		}
		if got := float64(first.FontWeight()); got != want.weight {
			t.Errorf("paragraph %d first line weight = %g, want %g", i+1, got, want.weight)
		}
		if got := string(next.FontStyle()); got != "normal" {
			t.Errorf("paragraph %d next line style = %s, want normal", i+1, got)
		}
		if got := float64(next.FontWeight()); got != 400 {
			t.Errorf("paragraph %d next line weight = %g, want 400", i+1, got)
		}
	}
}

func TestFirstLineFirstLetter(t *testing.T) {
	d := renderPage(t, `
      <style>
        p::first-letter { font-style: italic }
        p::first-line { font-weight: 700 }
      </style>
      <p>abc<br>def</p>
    `)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	children := kids(t, kids(t, lines[0], 1)[0], 3)
	letter := at(t, children[0], 0).Base().Style
	text := children[1].Base().Style
	if got := string(letter.FontStyle()); got != "italic" {
		t.Errorf("first letter style = %s, want italic", got)
	}
	if got := float64(letter.FontWeight()); got != 700 {
		t.Errorf("first letter weight = %g, want 700", got)
	}
	if got := string(text.FontStyle()); got != "normal" {
		t.Errorf("first line text style = %s, want normal", got)
	}
	if got := float64(text.FontWeight()); got != 700 {
		t.Errorf("first line text weight = %g, want 700", got)
	}
	next := at(t, lines[1], 0).Base().Style
	if got := string(next.FontStyle()); got != "normal" {
		t.Errorf("next line style = %s, want normal", got)
	}
	if got := float64(next.FontWeight()); got != 400 {
		t.Errorf("next line weight = %g, want 400", got)
	}
}

func TestTextIndentMultipage(t *testing.T) {
	for _, indent := range []string{"12px", "6%"} {
		t.Run(indent, func(t *testing.T) {
			noLogs(t)
			pages := paginate(t, fmt.Sprintf(`
        <style>
            @page { size: 220px 1.5em; margin: 0 }
            body { margin: 10px; text-indent: %s }
        </style>
        <p>Some text that is long enough that it take at least three line,
           but maybe more.
    `, indent))
			if len(pages) < 2 {
				t.Fatalf("%d pages, want at least 2", len(pages))
			}
			for i, want := range []float64{22, 10} { // 10px margin-left, then the 12px indent
				_, body := pageBody(t, pages[i])
				line := kids(t, kids(t, body, 1)[0], 1)[0]
				if x := kids(t, line, 1)[0].Base().PositionX; x != want {
					t.Errorf("page %d: text x = %g, want %g", i+1, x, want)
				}
			}
		})
	}
}

func TestTextFloatingPreLine(t *testing.T) {
	renderPage(t, `
      <div style="float: left; white-space: pre-line">This is
      oh this end </div>
    `)
}

func TestMaxLines(t *testing.T) {
	t.Skip("not supported: max-lines does not move the remaining lines to the next page")
	noLogs(t)
	pages := paginate(t, `
      <style>
        @page {size: 10px 10px;}
        p {
          font-family: testfont;
          font-size: 2px;
          max-lines: 2;
        }
      </style>
      <p>
        abcd efgh ijkl
      </p>
    `)
	_, body := pageBody(t, pages[0])
	paragraphs := kids(t, body, 2)
	line1, line2 := kids(t, paragraphs[0], 2)[0], kids(t, paragraphs[0], 2)[1]
	line3 := kids(t, paragraphs[1], 1)[0]
	for line, want := range map[boxes.Box]string{line1: "abcd", line2: "efgh", line3: "ijkl"} {
		if got := textAt(t, line, 0); got != want {
			t.Errorf("line = %q, want %q", got, want)
		}
	}
}

func TestContinue(t *testing.T) {
	d := renderPage(t, `
      <style>
        @page {size: 10px 4px;}
        div {
          continue: discard;
          font-family: testfont;
          font-size: 2px;
        }
      </style>
      <div>
        abcd efgh ijkl
      </div>
    `)
	lines := kids(t, at(t, d.body(t), 0), 2)
	if got := textAt(t, lines[0], 0); got != "abcd" {
		t.Errorf("line 1 = %q", got)
	}
	if got := textAt(t, lines[1], 0); got != "efgh" {
		t.Errorf("line 2 = %q", got)
	}
}
