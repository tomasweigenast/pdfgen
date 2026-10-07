package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil/csstest"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

func TestPositioningSchemes(t *testing.T) {
	s := newStyled(t, `<style>
.left{float:left} .end{float:inline-end} .note{float:footnote}
.abs{position:absolute} .fixed{position:fixed} .rel{position:relative}
.run{position:running(header)}
</style>
<p class=left></p><p class=end></p><p class=note></p><p class=abs></p><p class=fixed></p>
<p class=rel></p><p class=run></p><p></p>`)
	ps := csstest.Elements(s.doc, "p")
	tests := []struct {
		class                                        string
		floated, footnote, absolute, running, normal bool
	}{
		{"left", true, false, false, false, false},
		{"end", true, false, false, false, false},
		{"note", false, true, false, false, false},
		{"abs", false, false, true, false, false},
		{"fixed", false, false, true, false, false},
		{"rel", false, false, false, false, true},
		{"run", false, false, false, true, false},
		{"", false, false, false, false, true},
	}
	for i, tt := range tests {
		el := ps[i]
		b := boxes.NewBlockBox("p", s.sf.Get(el, ""), el, nil)
		got := []bool{b.IsFloated(), b.IsFootnote(), b.IsAbsolutelyPositioned(), b.IsRunning(), b.IsInNormalFlow()}
		want := []bool{tt.floated, tt.footnote, tt.absolute, tt.running, tt.normal}
		for j := range got {
			if got[j] != want[j] {
				t.Errorf("class %q: predicates %v, want %v", tt.class, got, want)
				break
			}
		}
	}
}

func TestFlexItemsIgnoreFloat(t *testing.T) {
	s := newStyled(t, `<style>p{float:left}</style><p></p>`)
	b := s.block("p")
	b.NotFloated = true
	if b.IsFloated() || !b.IsInNormalFlow() {
		t.Error("a box marked as not floated is floated")
	}
}

func TestMonolithicBoxes(t *testing.T) {
	s := newStyled(t, `<style>
.auto{overflow:auto} .scroll{overflow:scroll} .hidden{overflow:hidden}
.hidden-height{overflow:hidden;height:10px}
</style><div></div><div class=auto></div><div class=scroll></div><div class=hidden></div><div class=hidden-height></div>`)
	divs := csstest.Elements(s.doc, "div")
	want := []bool{false, true, true, false, true}
	for i, el := range divs {
		b := boxes.NewBlockBox("div", s.sf.Get(el, ""), el, nil)
		if got := b.IsMonolithic(); got != want[i] {
			t.Errorf("div %d (%q): monolithic = %v, want %v", i, el.Get("class"), got, want[i])
		}
	}
	plain := plainStyle()
	for _, b := range []boxes.Box{
		boxes.NewInlineBlockBox("", plain, nil, nil),
		boxes.NewInlineReplacedBox("", plain, nil, nil),
		boxes.NewBlockReplacedBox("", plain, nil, nil),
	} {
		if !b.IsMonolithic() {
			t.Errorf("%v is not monolithic", b.Kind())
		}
	}
	for _, b := range []boxes.Box{
		boxes.NewInlineBox("", plain, nil, nil),
		boxes.NewTableBox("", plain, nil, nil),
		boxes.NewFlexBox("", plain, nil, nil),
	} {
		if b.IsMonolithic() {
			t.Errorf("%v is monolithic", b.Kind())
		}
	}
}

func TestFormattingContexts(t *testing.T) {
	s := newStyled(t, `<style>.hidden{overflow:hidden} .root{display:flow-root} .f{float:left}</style>
<div></div><div class=hidden></div><div class=root></div><div class=f></div>`)
	divs := csstest.Elements(s.doc, "div")
	style := func(i int) *boxes.BlockBox {
		return boxes.NewBlockBox("div", s.sf.Get(divs[i], ""), divs[i], nil)
	}
	for i, want := range []bool{false, true, true, true} {
		if got := style(i).EstablishesFormattingContext(); got != want {
			t.Errorf("block %d establishes a context = %v, want %v", i, got, want)
		}
	}
	plain := plainStyle()
	tests := []struct {
		box  boxes.Box
		want bool
	}{
		{boxes.NewInlineBlockBox("", plain, nil, nil), true},
		{boxes.NewTableCellBox("", plain, nil, nil), true},
		{boxes.NewMarginBox("@top", plain), true},
		{boxes.NewFootnoteAreaBox(nil, plain), false},
		{boxes.NewTableCaptionBox("", plain, nil, nil), false},
		{boxes.NewInlineBox("", plain, nil, nil), false},
	}
	for _, tt := range tests {
		if got := tt.box.EstablishesFormattingContext(); got != tt.want {
			t.Errorf("%v establishes a context = %v, want %v", tt.box.Kind(), got, tt.want)
		}
	}
	col := boxes.NewBlockBox("div", plain, nil, nil)
	col.IsColumn = true
	if !col.EstablishesFormattingContext() {
		t.Error("a column does not establish a context")
	}
}

func TestMarginCollapsingBarriers(t *testing.T) {
	plain := plainStyle()
	b := boxes.NewBlockBox("div", plain, nil, nil)
	if !b.TopMarginCollapses() || !b.BottomMarginCollapses() {
		t.Error("a plain block does not collapse its margins")
	}
	for name, set := range map[string]func(*boxes.BlockBox){
		"flex item":     func(b *boxes.BlockBox) { b.IsFlexItem = true },
		"grid item":     func(b *boxes.BlockBox) { b.IsGridItem = true },
		"table wrapper": func(b *boxes.BlockBox) { b.IsTableWrapper = true },
		"root element":  func(b *boxes.BlockBox) { b.IsForRootElement = true },
		"column":        func(b *boxes.BlockBox) { b.IsColumn = true },
	} {
		b := boxes.NewBlockBox("div", plain, nil, nil)
		set(b)
		if b.TopMarginCollapses() || b.BottomMarginCollapses() {
			t.Errorf("%s collapses its margins", name)
		}
	}
	top := boxes.NewBlockBox("div", plain, nil, nil)
	top.PaddingTop = 1
	bottom := boxes.NewBlockBox("div", plain, nil, nil)
	bottom.BorderBottomWidth = 1
	if top.TopMarginCollapses() || !top.BottomMarginCollapses() || !bottom.TopMarginCollapses() || bottom.BottomMarginCollapses() {
		t.Error("padding or border on one side affects the other side")
	}
}

func TestRemoveDecoration(t *testing.T) {
	s := newStyled(t, `<style>.rtl{direction:rtl} .clone{box-decoration-break:clone}</style>
<div></div><span></span><span class=rtl></span><div class=clone></div>`)
	fill := func(b *boxes.BoxFields) {
		b.MarginTop, b.MarginRight, b.MarginBottom, b.MarginLeft = units.F(1), units.F(2), units.F(3), units.F(4)
		b.PaddingTop, b.PaddingRight, b.PaddingBottom, b.PaddingLeft = 1, 2, 3, 4
		b.BorderTopWidth, b.BorderRightWidth, b.BorderBottomWidth, b.BorderLeftWidth = 1, 2, 3, 4
	}
	spacing := func(b *boxes.BoxFields) [4]float64 {
		return [4]float64{b.MarginTop.V + b.PaddingTop + b.BorderTopWidth, b.MarginRight.V + b.PaddingRight + b.BorderRightWidth,
			b.MarginBottom.V + b.PaddingBottom + b.BorderBottomWidth, b.MarginLeft.V + b.PaddingLeft + b.BorderLeftWidth}
	}
	divs := csstest.Elements(s.doc, "div")
	spans := csstest.Elements(s.doc, "span")

	block := boxes.NewBlockBox("div", s.sf.Get(divs[0], ""), divs[0], nil)
	fill(block.Base())
	block.RemoveDecoration(true, false)
	if got := spacing(block.Base()); got != [4]float64{0, 6, 9, 12} || block.RemoveDecorationSides != boxes.SideTop {
		t.Errorf("block start: %v, sides %v", got, block.RemoveDecorationSides)
	}
	block.RemoveDecoration(false, true)
	if got := spacing(block.Base()); got != [4]float64{0, 6, 0, 12} {
		t.Errorf("block end: %v", got)
	}

	ltr := boxes.NewInlineBox("span", s.sf.Get(spans[0], ""), spans[0], nil)
	fill(ltr.Base())
	ltr.RemoveDecoration(true, false)
	if got := spacing(ltr.Base()); got != [4]float64{3, 6, 9, 0} {
		t.Errorf("ltr inline start: %v", got)
	}

	rtl := boxes.NewInlineBox("span", s.sf.Get(spans[1], ""), spans[1], nil)
	fill(rtl.Base())
	rtl.RemoveDecoration(true, false)
	if got := spacing(rtl.Base()); got != [4]float64{3, 0, 9, 12} {
		t.Errorf("rtl inline start: %v", got)
	}

	clone := boxes.NewBlockBox("div", s.sf.Get(divs[1], ""), divs[1], nil)
	fill(clone.Base())
	clone.RemoveDecoration(true, true)
	if got := spacing(clone.Base()); got != [4]float64{3, 6, 9, 12} || clone.RemoveDecorationSides != 0 {
		t.Errorf("clone: %v", got)
	}
}

func TestInlineBlockRemovesDecorationOnItsSides(t *testing.T) {
	b := boxes.NewInlineBlockBox("", plainStyle(), nil, nil)
	b.PaddingLeft, b.PaddingTop = 4, 5
	b.RemoveDecoration(true, false)
	if b.PaddingLeft != 0 || b.PaddingTop != 5 {
		t.Errorf("padding left %v, top %v", b.PaddingLeft, b.PaddingTop)
	}
	f := boxes.NewInlineFlexBox("", plainStyle(), nil, nil)
	f.PaddingLeft, f.PaddingTop = 4, 5
	f.RemoveDecoration(true, false)
	if f.PaddingLeft != 4 || f.PaddingTop != 0 {
		t.Errorf("inline flex padding left %v, top %v", f.PaddingLeft, f.PaddingTop)
	}
}

func TestInputsAndForms(t *testing.T) {
	s := newStyled(t, `<style>input,button,select,textarea{appearance:auto} .none{appearance:none}</style>
<form><input><input class=none><button></button><select></select><textarea></textarea><p></p></form>`)
	inputs := csstest.Elements(s.doc, "input")
	mk := func(name string, i int) boxes.Box {
		els := csstest.Elements(s.doc, name)
		return boxes.NewBlockBox(name, s.sf.Get(els[i], ""), els[i], nil)
	}
	if !mk("input", 0).IsInput() || mk("input", 1).IsInput() {
		t.Errorf("input appearance handling is wrong (%d inputs)", len(inputs))
	}
	for _, name := range []string{"button", "select", "textarea"} {
		if !mk(name, 0).IsInput() {
			t.Errorf("%s is not an input", name)
		}
	}
	if mk("p", 0).IsInput() || !mk("form", 0).IsForm() || mk("p", 0).IsForm() {
		t.Error("p or form misclassified")
	}
	els := csstest.Elements(s.doc, "button")
	line := boxes.NewLineBox("button", s.sf.Get(els[0], ""), els[0], nil)
	text := boxes.NewTextBox("button", s.sf.Get(els[0], ""), els[0], "x")
	if line.IsInput() || text.IsInput() {
		t.Error("the lines and text of a button are inputs")
	}
	if boxes.NewBlockBox("", plainStyle(), nil, nil).IsForm() {
		t.Error("a box without element is a form")
	}
}

func TestAttachmentLinks(t *testing.T) {
	s := newStyled(t, `<a rel="Attachment" href=x></a><a rel=next href=y></a><b></b>`)
	as := csstest.Elements(s.doc, "a")
	att := boxes.NewInlineBox("a", s.sf.Get(as[0], ""), as[0], nil)
	other := boxes.NewInlineBox("a", s.sf.Get(as[1], ""), as[1], nil)
	b := s.el("b")
	notLink := boxes.NewInlineBox("b", s.sf.Get(b, ""), b, nil)
	if !att.IsAttachment() || other.IsAttachment() || notLink.IsAttachment() {
		t.Error("attachment links misclassified")
	}
}

func TestPageBleed(t *testing.T) {
	page := boxes.NewPageBox(pageType(), plainStyle())
	page.Width, page.Height = units.F(100), units.F(200)
	if got := page.Bleed(); got != (boxes.Sides{}) {
		t.Errorf("default bleed = %+v", got)
	}
	if got, want := page.BleedArea(), (boxes.Rect{Width: 100, Height: 200}); got != want {
		t.Errorf("bleed area = %+v, want %+v", got, want)
	}
}
