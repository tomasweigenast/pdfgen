package layout

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/text/linebreak"
)

func textBox(t *testing.T, source string) *boxes.TextBox {
	t.Helper()
	for _, box := range parseBoxes(t, source, "p") {
		if text, ok := box.(*boxes.TextBox); ok {
			return text
		}
	}
	t.Fatal("paragraph contains no text")
	return nil
}

func TestSplitTextResumesAtUTF8ByteOffsets(t *testing.T) {
	box := textBox(t, `<p style="font: 10px/2 testfont">é ab cd`)
	ctx := &Context{Text: linebreak.NewContext(box.Style.Context().FontConfig)}
	result := splitTextBox(ctx, box, 20, 3, true)
	if result.Box.Text != "ab" || result.ResumeIndex == nil || *result.ResumeIndex != 6 {
		t.Fatalf("split = %q at %v, want ab at byte 6", result.Box.Text, result.ResumeIndex)
	}
	if result.Box.Width.V != 20 || result.Box.Height.V != 10 || result.Box.MarginHeight() != 20 {
		t.Fatalf("text size = %v x %v, line height %g", result.Box.Width, result.Box.Height, result.Box.MarginHeight())
	}
	if box.Text != "é ab cd" || box.Layout != nil {
		t.Fatal("splitting changed the source text")
	}
}

func TestSplitTextDistinguishesPreservedBreaks(t *testing.T) {
	box := textBox(t, "<p style='font: 10px testfont; white-space: pre'>ab\ncd")
	ctx := &Context{Text: linebreak.NewContext(box.Style.Context().FontConfig)}
	result := splitTextBox(ctx, box, 100, 0, true)
	if result.Box.Text != "ab" || result.ResumeIndex == nil || *result.ResumeIndex != 3 || !result.PreservedLineBreak {
		t.Fatalf("preserved break: %#v", result)
	}
}

func TestSkipLeadingWhitespaceRetainsNestedContinuation(t *testing.T) {
	paragraph := parseBox(t, `<p>ab`, "p")
	line := boxes.ChildrenOf(paragraph)[0]
	text := boxes.ChildrenOf(line)[0].(*boxes.TextBox).CopyWithText(" ab")
	span := boxes.NewInlineBox("span", text.Style, text.Element, []boxes.Box{text})
	line.(boxes.Parent).ParentBase().Children = []boxes.Box{span}
	resume, exhausted := skipFirstWhitespace(line, nil)
	if exhausted {
		t.Fatal("nonempty line was exhausted")
	}
	if !resume.Equal(ResumeAt{0: {0: {1: nil}}}) {
		t.Fatalf("whitespace continuation = %v", resume)
	}
}

func TestSplitTextOfALoneSpaceWithWordSpacing(t *testing.T) {
	box := textBox(t, `<p style="font: 10px testfont; white-space: pre-wrap; word-spacing: 2px"> fhejaje if jdej `)
	ctx := &Context{Text: linebreak.NewContext(box.Style.Context().FontConfig)}
	result := splitTextBox(ctx, box, 50, 0, false)
	if result.ResumeIndex == nil || *result.ResumeIndex != 1 {
		t.Fatalf("resume = %v, want 1", result.ResumeIndex)
	}
	if result.PreservedLineBreak {
		t.Fatal("the zero-width space is not a preserved break")
	}
}
