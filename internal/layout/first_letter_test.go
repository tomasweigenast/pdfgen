package layout

import (
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

// firstLetterLine returns the paragraph of a document, its line box and the
// style of its ::first-letter.
func firstLetterLine(t *testing.T, css, contents string) (*boxes.LineBox, boxes.Box) {
	t.Helper()
	paragraph := parseBox(t, `<style>p::first-letter { `+css+` }</style><p>`+contents, "p")
	line, ok := boxes.ChildrenOf(paragraph)[0].(*boxes.LineBox)
	if !ok || paragraph.Base().FirstLetterStyle == nil {
		t.Fatalf("paragraph %v has no line or first-letter style", paragraph)
	}
	return line, paragraph
}

func TestFirstLetterToBoxWrapsLeadingPunctuationWithTheLetter(t *testing.T) {
	line, paragraph := firstLetterLine(t, "font-size: 2em", `"Hello`)
	resume := firstLetterToBox(line, nil, paragraph.Base().FirstLetterStyle)
	if resume != nil {
		t.Fatalf("resume = %v, want none", resume)
	}
	children := kids(t, line, 2)
	letter, ok := children[0].(*boxes.InlineBox)
	if !ok || letter.ElementTag != "p::first-letter" {
		t.Fatalf("first child is %v with tag %q", children[0], children[0].Base().ElementTag)
	}
	if got := textAt(t, letter, 0); got != `"H` {
		t.Errorf("letter = %q, want %q", got, `"H`)
	}
	if got := textOf(t, children[1]); got != "ello" {
		t.Errorf("rest = %q, want ello", got)
	}
	if size := float64(letter.Style.FontSize()); size != 32 {
		t.Errorf("letter font size = %g, want 32", size)
	}
	if size := float64(children[1].Base().Style.FontSize()); size != 16 {
		t.Errorf("rest font size = %g, want 16", size)
	}
}

func TestFirstLetterToBoxFloatMakesABlock(t *testing.T) {
	line, paragraph := firstLetterLine(t, "float: left", "Hello")
	firstLetterToBox(line, nil, paragraph.Base().FirstLetterStyle)
	letter, ok := kids(t, line, 2)[0].(*boxes.BlockBox)
	if !ok {
		t.Fatalf("first child is %v, want a block", boxes.ChildrenOf(line)[0])
	}
	if !letter.IsFloated() {
		t.Error("the letter is not floated")
	}
	if got := textAt(t, letter, 0, 0); got != "H" {
		t.Errorf("letter = %q, want H", got)
	}
}

func TestFirstLetterToBoxIgnoresPosition(t *testing.T) {
	line, paragraph := firstLetterLine(t, "position: absolute; color: red", "Hello")
	firstLetterToBox(line, nil, paragraph.Base().FirstLetterStyle)
	letter := kids(t, line, 2)[0]
	if letter.IsAbsolutelyPositioned() {
		t.Error("the letter is positioned")
	}
	if got := letter.Base().Style.Position().Keyword; got != "static" {
		t.Errorf("position = %q, want static", got)
	}
	if reflect.DeepEqual(letter.Base().Style.Color(), paragraph.Base().Style.Color()) {
		t.Error("the letter did not take the color of the pseudo-element")
	}
}

func TestFirstLetterToBoxSearchesInlineChildren(t *testing.T) {
	line, paragraph := firstLetterLine(t, "font-size: 2em", "<em>abc</em> def")
	firstLetterToBox(line, nil, paragraph.Base().FirstLetterStyle)
	em := kids(t, line, 2)[0]
	children := kids(t, em, 2)
	if got := textAt(t, children[0], 0); got != "a" {
		t.Errorf("letter = %q, want a", got)
	}
	if got := textOf(t, children[1]); got != "bc" {
		t.Errorf("rest of the em = %q, want bc", got)
	}
}

func TestFirstLetterToBoxContinuesAfterTheSkippedText(t *testing.T) {
	line, paragraph := firstLetterLine(t, "font-size: 2em", "Hello")
	skip := ResumeAt{0: {2: nil}}
	if resume := firstLetterToBox(line, skip, paragraph.Base().FirstLetterStyle); resume != nil {
		t.Fatalf("resume = %v, want none: the skipped text is dropped", resume)
	}
	children := kids(t, line, 2)
	if got := textAt(t, children[0], 0); got != "l" {
		t.Errorf("letter = %q, want l", got)
	}
	if got := textOf(t, children[1]); got != "lo" {
		t.Errorf("rest = %q, want lo", got)
	}
}

func TestFirstLetterToBoxNeedsDeclarations(t *testing.T) {
	line, _ := firstLetterLine(t, "font-size: 2em", "Hello")
	if resume := firstLetterToBox(line, ResumeAt{0: nil}, nil); !resume.Equal(ResumeAt{0: nil}) {
		t.Fatalf("resume = %v, want it unchanged", resume)
	}
	if got := textOf(t, kids(t, line, 1)[0]); got != "Hello" {
		t.Fatalf("text = %q, want it unchanged", got)
	}
}

func TestApplyDeclaredCopiesInheritedPropertiesOnly(t *testing.T) {
	paragraph := parseBox(t, `<style>p::first-line { color: red; margin-left: 5px; font-weight: 700 }</style><p>a`, "p")
	pseudo := paragraph.Base().FirstLineStyle
	if !hasDeclarations(pseudo) {
		t.Fatal("the first line has no declarations")
	}

	target := paragraph.Base().Style.Copy()
	applyDeclared(target, pseudo, true)
	if !reflect.DeepEqual(target.Color(), pseudo.Color()) || float64(target.FontWeight()) != 700 {
		t.Error("inherited properties were not copied")
	}
	if target.MarginLeft() != paragraph.Base().Style.MarginLeft() {
		t.Error("an inherited-only copy changed margin-left")
	}

	target = paragraph.Base().Style.Copy()
	applyDeclared(target, pseudo, false)
	if got := target.MarginLeft(); got != pseudo.MarginLeft() || got == (props.Length{}) {
		t.Errorf("margin-left = %v, want %v", got, pseudo.MarginLeft())
	}
	if hasDeclarations(nil) || hasDeclarations(paragraph.Base().Style.Anonymous()) {
		t.Error("a missing style and a style without rules have declarations")
	}
}
