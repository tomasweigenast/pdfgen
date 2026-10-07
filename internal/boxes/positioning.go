package boxes

import "slices"

// IsFloated reports whether the box is floated.
func (b *BoxFields) IsFloated() bool {
	if b.NotFloated {
		return false
	}
	switch b.Style.Float() {
	case "left", "right", "inline-start", "inline-end":
		return true
	}
	return false
}

// IsFootnote reports whether the box is a footnote.
func (b *BoxFields) IsFootnote() bool { return b.Style.Float() == "footnote" }

// IsNote reports whether the box is a note.
func (b *BoxFields) IsNote() bool { return b.Style.Position().Keyword == "note()" }

// IsAbsolutelyPositioned reports whether the box is in the absolute
// positioning scheme.
func (b *BoxFields) IsAbsolutelyPositioned() bool {
	k := b.Style.Position().Keyword
	return k == "absolute" || k == "fixed"
}

// IsRunning reports whether the box is a running element.
func (b *BoxFields) IsRunning() bool { return b.Style.Position().Keyword == "running()" }

// IsInNormalFlow reports whether the box is in normal flow.
func (b *BoxFields) IsInNormalFlow() bool {
	return !(b.IsFloated() || b.IsAbsolutelyPositioned() || b.IsRunning() ||
		b.IsFootnote() || b.IsNote())
}

// IsMonolithic reports whether the box cannot be fragmented (CSS Fragmentation
// 3 §3.1).
func (b *BoxFields) IsMonolithic() bool {
	k := kinds[b.kind]
	if k.atomic || k.replaced {
		return true
	}
	switch b.Style.Overflow() {
	case "auto", "scroll":
		return true
	case "hidden":
		return !b.Style.Height().IsAuto()
	}
	return false
}

// EstablishesFormattingContext reports whether the box establishes a block
// formatting context (CSS 2.1 §9.4.1).
func (b *BoxFields) EstablishesFormattingContext() bool {
	k := kinds[b.kind]
	return b.IsFloated() ||
		b.IsAbsolutelyPositioned() ||
		b.IsColumn ||
		(k.blockContainer && !k.blockBox) ||
		(k.blockBox && b.Style.Overflow() != "visible") ||
		slices.Contains(b.Style.Display(), "flow-root")
}

// PageValues returns the page names at the start and at the end of the box
// (CSS Paged Media 3 §7.3.2). Empty means auto.
func (b *BoxFields) PageValues() (start, end string) {
	p := string(b.Style.Page())
	return p, p
}

// IsAttachment reports whether the link of the box is stored as a PDF
// attachment.
func (b *BoxFields) IsAttachment() bool {
	return b.Element != nil && b.Element.LocalName() == "a" && b.Element.HasLinkType("attachment")
}

// IsInput reports whether the box is a form input (HTML Living Standard
// §4.10.18.4).
func (b *BoxFields) IsInput() bool {
	if b.Style.Appearance() != "auto" || b.Element == nil {
		return false
	}
	switch b.Element.LocalName() {
	case "button", "input", "select", "textarea":
		return b.kind != KindLine && b.kind != KindText
	}
	return false
}

// IsForm reports whether the box is a form element.
func (b *BoxFields) IsForm() bool {
	return b.Element != nil && b.Element.LocalName() == "form"
}

// TopMarginCollapses reports whether the top margin of the box collapses
// with the one of its first child (CSS 2.1 §8.3.1).
func (b *BoxFields) TopMarginCollapses() bool {
	return !(b.BorderTopWidth != 0 || b.PaddingTop != 0 || b.marginBarrier())
}

// BottomMarginCollapses reports whether the bottom margin of the box
// collapses with the one of its last child (CSS 2.1 §8.3.1).
func (b *BoxFields) BottomMarginCollapses() bool {
	return !(b.BorderBottomWidth != 0 || b.PaddingBottom != 0 || b.marginBarrier())
}

func (b *BoxFields) marginBarrier() bool {
	return b.IsFlexItem || b.IsGridItem || b.EstablishesFormattingContext() ||
		b.IsTableWrapper || b.IsForRootElement
}
