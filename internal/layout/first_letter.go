package layout

import (
	"strings"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/css/computed"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/text/transform"
)

// hasDeclarations reports whether the style of a ::first-line or
// ::first-letter pseudo-element sets any property.
func hasDeclarations(style *computed.Style) bool {
	return style != nil && len(style.Cascaded()) != 0
}

// applyDeclared copies to dst the computed value of every property that the
// rules of a pseudo-element style declare. With inheritedOnly, only the
// inherited properties are copied: the first line of a box is not a box of
// its own, so what it can change in the boxes it contains is what they
// inherit (CSS Pseudo-Elements 4 §3).
func applyDeclared(dst, pseudo *computed.Style, inheritedOnly bool) {
	for name := range pseudo.Cascaded() {
		id, ok := props.ID(name)
		if !ok || inheritedOnly && !id.Inherited() {
			continue
		}
		dst.Set(id, pseudo.Get(id))
	}
}

// firstLetterToBox wraps the first letter of the text of a line in a box
// that has the style of the ::first-letter pseudo-element, and returns the
// skip stack that resumes after it (CSS Pseudo-Elements 4 §2). The letter
// is an inline box or, when the pseudo-element is floated, a block.
func firstLetterToBox(box boxes.Parent, skip ResumeAt, style *computed.Style) ResumeAt {
	children := box.ParentBase().Children
	if !hasDeclarations(style) || len(children) == 0 {
		return skip
	}
	b := box.Base()
	tag := b.ElementTag + "::first-letter"

	switch child := children[0].(type) {
	case *boxes.TextBox:
		letterStyle := b.Style.Copy()
		applyDeclared(letterStyle, style, false)
		// A positioned letter would be taken out of the line; the property
		// does not apply to the pseudo-element.
		letterStyle.Set(props.PropPosition, props.Positioning{Keyword: "static"})

		if strings.HasSuffix(child.ElementTag, "::first-letter") {
			letter := boxes.NewInlineBox(tag, letterStyle, b.Element, []boxes.Box{child})
			box.ParentBase().Children = append([]boxes.Box{letter}, children[1:]...)
			return skip
		}
		if child.Text == "" {
			return skip
		}
		if len(skip) != 0 {
			if _, inner := firstResume(skip); len(inner) != 0 {
				index, _ := firstResume(inner)
				child.Text = child.Text[min(index, len(child.Text)):]
				skip = nil
			}
		}
		letterText := transform.FirstLetter(child.Text)
		child.Text = child.Text[len(letterText):]
		if strings.TrimLeft(letterText, "\n") == "" {
			return skip
		}

		// The initial letter is similar to an inline-level element if its
		// float is none, otherwise to a floated element.
		textStyle := letterStyle.Anonymous()
		text := boxes.NewTextBox(tag, textStyle, b.Element, letterText)
		var letter boxes.Box
		if letterStyle.Float() == "none" {
			letter = boxes.NewInlineBox(tag, letterStyle, b.Element, []boxes.Box{text})
		} else {
			line := boxes.NewLineBox(tag, textStyle, b.Element, []boxes.Box{text})
			letter = boxes.NewBlockBox(tag, letterStyle, b.Element, []boxes.Box{line})
		}
		box.ParentBase().Children = append([]boxes.Box{letter}, children...)
		boxes.ProcessTextTransform(text)
	case boxes.Parent:
		var childSkip ResumeAt
		if len(skip) != 0 {
			_, childSkip = firstResume(skip)
		}
		childSkip = firstLetterToBox(child, childSkip, style)
		if len(skip) != 0 {
			index, _ := firstResume(skip)
			skip = ResumeAt{index: childSkip}
		}
	}
	return skip
}
