package layout

import (
	"fmt"
	"math"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestIterLineBoxesResumesWhereEachLineEnds(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont; width: 30px">aa bb cc dd`, "p", 30)
	lines := f.layout(t, blockOptions{}, 0, 0)
	if len(lines) != 4 {
		t.Fatalf("%d lines, want 4", len(lines))
	}
	for i, want := range []string{"aa", "bb", "cc", "dd"} {
		if got := textAt(t, lines[i].Line, 0); got != want {
			t.Errorf("line %d = %q, want %q", i+1, got, want)
		}
		if y := lines[i].Line.PositionY; y != float64(i)*10 {
			t.Errorf("line %d y = %g, want %d", i+1, y, i*10)
		}
	}
	for i, want := range []ResumeAt{{0: {3: nil}}, {0: {6: nil}}, {0: {9: nil}}, nil} {
		if got := lines[i].ResumeAt; !got.Equal(want) {
			t.Errorf("line %d resumes at %v, want %v", i+1, got, want)
		}
	}
}

func TestIterLineBoxesContinuesFromASkipStack(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont; width: 30px">aa bb cc dd`, "p", 30)
	lines := f.layout(t, blockOptions{SkipStack: ResumeAt{0: {6: nil}}}, 0, 0)
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	if got := textAt(t, lines[0].Line, 0); got != "cc" {
		t.Errorf("first line = %q, want cc", got)
	}
}

func TestIterLineBoxesStopsWithTheCaller(t *testing.T) {
	const source = `<p style="font: 10px testfont; width: 40px">aaaa bbbb <span style="float: left; width: 5px; height: 5px"></span>cccc`
	f := newLineFixture(t, source, "p", 40)
	f.layout(t, blockOptions{}, 0, 1)
	if n := len(excludedShapes(f.ctx)); n != 0 {
		t.Fatalf("%d floats after the first line, want 0: the float is in the second", n)
	}

	f = newLineFixture(t, source, "p", 40)
	f.layout(t, blockOptions{}, 0, 2)
	if n := len(excludedShapes(f.ctx)); n != 1 {
		t.Fatalf("%d floats after the second line, want 1", n)
	}
}

func TestIterLineBoxesIndentsTheFirstLineOnly(t *testing.T) {
	const source = `<p style="font: 10px testfont; width: 100px; text-indent: 20px">aaaa bbbb cccc`
	f := newLineFixture(t, source, "p", 100)
	lines := f.layout(t, blockOptions{}, 0, 0)
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	if x := boxes.ChildrenOf(lines[0].Line)[0].Base().PositionX; x != 20 {
		t.Errorf("first line text x = %g, want 20", x)
	}
	if x := boxes.ChildrenOf(lines[1].Line)[0].Base().PositionX; x != 0 {
		t.Errorf("second line text x = %g, want 0", x)
	}

	// A block that continues on another page has no indent.
	f = newLineFixture(t, source, "p", 100)
	lines = f.layout(t, blockOptions{SkipStack: ResumeAt{0: {5: nil}}}, 0, 0)
	if x := boxes.ChildrenOf(lines[0].Line)[0].Base().PositionX; x != 0 {
		t.Errorf("continued text x = %g, want 0", x)
	}
}

func TestIterLineBoxesPercentageIndent(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont; text-indent: 10%">aa`, "p", 200)
	lines := f.layout(t, blockOptions{}, 0, 0)
	if x := boxes.ChildrenOf(lines[0].Line)[0].Base().PositionX; x != 20 {
		t.Errorf("text x = %g, want 20", x)
	}
}

func TestGetNextLineboxHasNoLineAfterTheLastText(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont">aa`, "p", 100)
	result := getNextLinebox(f.ctx, f.line, f.block, 0, 0, ResumeAt{0: {2: nil}}, &positionedBoxes{}, nil, nil)
	if result.Line != nil {
		t.Fatalf("got a line %v after all the text, want none", result.Line)
	}
}

func TestGetNextLineboxSkipsSpacesAfterTheResumePoint(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont">aa   bb`, "p", 100)
	result := getNextLinebox(f.ctx, f.line, f.block, 0, 0, ResumeAt{0: {2: nil}}, &positionedBoxes{}, nil, nil)
	if result.Line == nil {
		t.Fatal("no line")
	}
	if got := textAt(t, result.Line, 0); got != "bb" {
		t.Fatalf("line = %q, want bb: the spaces that start the line are removed", got)
	}
}

func TestPhantomLineboxesTakeNoSpace(t *testing.T) {
	for _, tt := range []struct {
		name, contents string
		phantom        bool
	}{
		{"empty inline", `<span></span>`, true},
		{"nested empty inlines", `<span><b></b></span>`, true},
		{"padding", `<span style="padding: 1px"></span>`, false},
		{"border", `<span style="border: 1px solid"></span>`, false},
		{"margin", `<span style="margin-left: 3px"></span>`, false},
		{"zero margin", `<span style="margin: 0"></span>`, true},
		{"nested padding", `<span><b style="padding-top: 1px"></b></span>`, false},
		{"text", `<span>a</span>`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := renderPage(t, `<style>body { font: 10px/10px testfont }</style><body><p>`+tt.contents)
			p := kids(t, d.body(t), 1)[0]
			if got := p.Base().Height.V == 0; got != tt.phantom {
				t.Fatalf("paragraph height = %g, want phantom = %v", p.Base().Height.V, tt.phantom)
			}
		})
	}
}

func TestPreservedBreakKeepsALineWithoutContentVisible(t *testing.T) {
	d := renderPage(t, `<style>body { font: 10px/10px testfont }</style><body><pre>a


b</pre>`)
	pre := kids(t, d.body(t), 1)[0]
	if h := pre.Base().Height.V; h != 40 {
		t.Fatalf("pre height = %g, want 40: the empty lines count", h)
	}
}

func TestRemoveLastWhitespace(t *testing.T) {
	for _, tt := range []struct {
		name, style, contents string
		width                 float64
	}{
		{"text", "", "a b  ", 30},
		{"in an inline box", "", "a <span>b  </span>", 30},
		{"in nested inline boxes", "", "a <span><b>b </b> </span>", 30},
		{"preserved", "white-space: pre", "a b  ", 50},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := renderPage(t, `<style>body { font: 10px/10px testfont }</style><body><p style="`+tt.style+`">`+tt.contents)
			line := kids(t, kids(t, d.body(t), 1)[0], 1)[0]
			if w := line.Base().Width.V; w != tt.width {
				t.Fatalf("line width = %g, want %g", w, tt.width)
			}
			if tt.style != "" {
				return
			}
			last := lastDescendantText(line)
			if got := last.Text; got != "" && got[len(got)-1] == ' ' {
				t.Fatalf("the line ends with %q", got)
			}
		})
	}
}

func lastDescendantText(box boxes.Box) *boxes.TextBox {
	var last *boxes.TextBox
	for d := range boxes.Descendants(box, false) {
		if text, ok := d.(*boxes.TextBox); ok {
			last = text
		}
	}
	return last
}

func TestRemoveLastWhitespaceMovesRightToLeftTextByTheSpace(t *testing.T) {
	d := renderPage(t, `
      <style>body { font: 10px/10px testfont; direction: rtl; width: 100px }</style>
      <body>ab `)
	body := d.body(t)
	text := at(t, body, 0, 0)
	if got := textOf(t, text); got != "ab" {
		t.Fatalf("text = %q, want ab", got)
	}
	// The text is shaped and drawn from its box with the space it had,
	// which is at its left: the box starts where "ab " would, so that the
	// letters end at the right edge.
	right := body.Base().PositionX + body.Base().Width.V
	if got, want := text.Base().PositionX, right-30; got != want {
		t.Errorf("text x = %g, want %g", got, want)
	}
	if got := text.Base().Width.V; got != 20 {
		t.Errorf("text width = %g, want 20", got)
	}
}

// linesOf returns the line boxes of a block, however deep their blocks are.
func linesOf(box boxes.Box) []boxes.Box {
	var lines []boxes.Box
	for d := range boxes.Descendants(box, false) {
		if d.Kind() == boxes.KindLine {
			lines = append(lines, d)
		}
	}
	return lines
}

func TestFloatsInALineReduceItsWidth(t *testing.T) {
	d := renderPage(t, `
      <style>body { font: 10px/10px testfont; width: 100px }
        span { float: left; width: 30px; height: 10px }</style>
      <body><span></span>aaaa bbbb cccc`)
	lines := linesOf(d.body(t))
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	// The float is next to the first line, which has 70px, and not to the
	// second one.
	if x := lines[0].Base().PositionX; x != 30 {
		t.Fatalf("first line x = %g, want 30", x)
	}
	if got := allText(lines[0]); got != "aaaa" {
		t.Fatalf("first line = %q", got)
	}
	if x := lines[1].Base().PositionX; x != 0 {
		t.Fatalf("second line x = %g, want 0, below the float", x)
	}
}

func TestFloatThatDoesNotFitWaitsForTheNextLine(t *testing.T) {
	d := renderPage(t, `
      <style>body { font: 10px/10px testfont; width: 100px }
        span { float: left; width: 30px; height: 10px }</style>
      <body>aaaa bbbb <span></span>cc`)
	lines := linesOf(d.body(t))
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	var float boxes.Box
	for d := range boxes.Descendants(d.body(t), false) {
		if d.IsFloated() {
			float = d
		}
	}
	if float == nil {
		t.Fatal("the float is not in the tree")
	}
	if y, want := float.Base().PositionY, lines[0].Base().PositionY+lines[0].Base().Height.V; y != want {
		t.Fatalf("float y = %g, want %g, below the first line", y, want)
	}
}

func TestJustifyLineSpreadsExtraWidthOverSpaces(t *testing.T) {
	d := renderPage(t, `
      <style>
        p { font: 10px testfont; width: 100px; text-align: justify }
      </style>
      <p>aa bb <em>cc</em> dd ee</p>`)
	lines := kids(t, kids(t, d.body(t), 1)[0], 2)
	first := kids(t, lines[0], 2)
	// "aa bb " and "cc" fit in 100px: 6 and 2 characters.
	text, em := first[0].(*boxes.TextBox), first[1]
	if em.Base().ElementTag != "em" {
		t.Fatalf("second child is %s", em.Base().ElementTag)
	}
	if lines[0].Base().Width.V != 100 {
		t.Errorf("justified line width = %g, want 100", lines[0].Base().Width.V)
	}
	// The 20px that were left go to the two spaces of the text before the
	// em.
	if text.JustificationSpacing != 10 {
		t.Errorf("spacing = %g, want 10", text.JustificationSpacing)
	}
	if got := text.Width.V; got != 60+20 {
		t.Errorf("justified text width = %g, want 80", got)
	}
	if x := em.Base().PositionX; x != 80 {
		t.Errorf("em x = %g, want 80", x)
	}
	// The last line is not stretched.
	if w := lines[1].Base().Width.V; w >= 100 {
		t.Errorf("last line width = %g, want less than 100", w)
	}
}

func TestVerticalAlignMathPercentageUsesLineHeight(t *testing.T) {
	d := renderPage(t, `
      <style>
        body { font: 10px/20px testfont }
        img { width: 10px; vertical-align: calc(5px + 50%) }
      </style>
      <body><img src="pattern.png">`)
	line := kids(t, d.body(t), 1)[0]
	img := kids(t, line, 1)[0]
	// 50% of the 20px line height and 5px: the image is raised by 15px and
	// is 10px high, so its top is 25px above the baseline, higher than the
	// strut.
	lineHeight, ascent := line.Base().Style.Strut()
	want := 25 + (lineHeight - ascent)
	if got := line.Base().Height.V; math.Abs(got-want) > 1e-9 {
		t.Errorf("line height = %g, want %g", got, want)
	}
	if y := img.Base().PositionY; y != 0 {
		t.Errorf("image y = %g, want 0", y)
	}
}

func TestAlignedSubtreeIncludesTheBoxItself(t *testing.T) {
	d := renderPage(t, `
      <style>body { font: 10px/10px testfont }</style>
      <body>x`)
	line := kids(t, d.body(t), 1)[0]
	var subtrees []boxes.Box
	e := alignedSubtreeVerticality(line, &subtrees, 100)
	top := 100 - baselineOf(line)
	if e.min != top || e.max != top+line.MarginHeight() || len(subtrees) != 0 {
		t.Fatalf("extent = [%g, %g] with %d subtrees, want [%g, %g]",
			e.min, e.max, len(subtrees), top, top+line.MarginHeight())
	}
}

func TestSplitInlineBoxResumesInsideNestedBoxes(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont; width: 30px"><span>aa bb</span> cc`, "p", 30)
	lines := f.layout(t, blockOptions{}, 0, 0)
	if len(lines) != 3 {
		t.Fatalf("%d lines, want 3", len(lines))
	}
	// The first line ends inside the text of the span, which is the first
	// child of the line.
	if got, want := lines[0].ResumeAt, (ResumeAt{0: {0: {3: nil}}}); !got.Equal(want) {
		t.Errorf("first resume = %v, want %v", got, want)
	}
	// The second one ends after the span.
	if got, want := lines[1].ResumeAt, (ResumeAt{1: {1: nil}}); !got.Equal(want) {
		t.Errorf("second resume = %v, want %v", got, want)
	}
	if got := textAt(t, lines[2].Line, 0); got != "cc" {
		t.Errorf("last line = %q, want cc", got)
	}
	if lines[2].ResumeAt != nil {
		t.Errorf("last resume = %v, want none", lines[2].ResumeAt)
	}
}

func TestSplitInlineBoxKeepsAnOverflowingWordWhole(t *testing.T) {
	f := newLineFixture(t, `<p style="font: 10px testfont; width: 20px">a <b>bbbbbb</b> c`, "p", 20)
	lines := f.layout(t, blockOptions{}, 0, 0)
	var got []string
	for _, line := range lines {
		got = append(got, allText(line.Line))
	}
	if want := []string{"a", "bbbbbb", "c"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
}

func TestInlineBlockBaseline(t *testing.T) {
	for _, tt := range []struct {
		name, style string
		lastLine    bool
	}{
		{"visible", "", true},
		{"hidden", "overflow: hidden", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := renderPage(t, `
      <style>body { font: 10px/20px testfont }</style>
      <body>a<div style="display: inline-block; `+tt.style+`">x<br>y</div>b`)
			line := kids(t, d.body(t), 1)[0]
			block := kids(t, line, 3)[1]
			lines := linesOf(block)
			last := lines[len(lines)-1]
			want := block.Base().PositionY + block.MarginHeight()
			if tt.lastLine {
				want = last.Base().PositionY + baselineOf(last)
			}
			// The baseline is relative to the margin box, which the
			// alignment of the line moved.
			offset := block.Base().PositionY - firstLinePosition(block)
			if got := baselineOf(block) + offset; got != want {
				t.Fatalf("baseline = %g, want %g", got, want)
			}
		})
	}
}

func TestFindInFlowBaseline(t *testing.T) {
	d := renderPage(t, `
      <style>body { font: 10px/20px testfont }</style>
      <body><div>
        <p style="float: left">float</p>
        <p style="position: absolute">absolute</p>
        <p>first</p>
        <p>second</p>
      </div>`)
	div := kids(t, d.body(t), 1)[0]
	lines := linesOf(div)
	var first, last boxes.Box
	for _, line := range lines {
		switch allText(line) {
		case "first":
			first = line
		case "second":
			last = line
		}
	}
	if first == nil || last == nil {
		t.Fatalf("lines not found in %d lines", len(lines))
	}
	if got := findInFlowBaseline(div, false); got == nil || *got != first.Base().PositionY+baselineOf(first) {
		t.Errorf("first baseline = %v, want that of the first paragraph", got)
	}
	if got := findInFlowBaseline(div, true); got == nil || *got != last.Base().PositionY+baselineOf(last) {
		t.Errorf("last baseline = %v, want that of the second paragraph", got)
	}
	empty := kids(t, renderPage(t, `<body><div></div>`).body(t), 1)[0]
	if got := findInFlowBaseline(empty, true); got != nil {
		t.Errorf("baseline of an empty block = %v, want none", *got)
	}
}

// firstLinePosition returns the position the first line of a box had
// before the box was moved: its top, as the box was laid out from y = 0.
func firstLinePosition(block boxes.Box) float64 {
	return linesOf(block)[0].Base().PositionY - block.Base().ContentBoxY() + block.Base().PositionY
}
