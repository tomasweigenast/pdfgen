package boxes_test

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
)

func TestProcessWhitespace(t *testing.T) {
	tests := []struct {
		name       string
		whiteSpace string
		in         string
		following  bool
		want       string
		leading    bool
		trailing   bool
	}{
		{"collapse runs", "normal", "a \t  b", false, "a b", false, false},
		{"newlines are spaces", "normal", "a\n\nb\r\nc", false, "a b c", false, false},
		{"spaces around newlines", "normal", "a \n\t b", false, "a b", false, false},
		{"leading space kept at start", "normal", " a", false, " a", false, false},
		{"leading space dropped after a space", "normal", " a ", true, "a ", true, true},
		{"only a space after a space", "normal", " ", true, "", true, true},
		{"nowrap collapses like normal", "nowrap", "a  b\nc", false, "a b c", false, false},
		{"pre-line keeps newlines", "pre-line", "a  b \n  c", false, "a b\nc", false, false},
		{"pre-line normalizes carriage returns", "pre-line", "a\r\nb\rc", false, "a\nb\nc", false, false},
		{"pre keeps spaces", "pre", "a  b \n c", true, "a  b \n c", false, false},
		{"pre-wrap keeps spaces", "pre-wrap", " a  ", true, " a  ", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStyled(t, `<style>p{white-space:`+tt.whiteSpace+`}</style><p></p>`)
			b := boxes.NewTextBox("p", s.style("p"), s.el("p"), tt.in)
			got := b.ProcessWhitespace(tt.following)
			if b.Text != tt.want || b.LeadingCollapsibleSpace != tt.leading || got != tt.trailing {
				t.Errorf("text %q, leading %v, trailing %v; want %q, %v, %v",
					b.Text, b.LeadingCollapsibleSpace, got, tt.want, tt.leading, tt.trailing)
			}
		})
	}
}

func TestProcessWhitespaceFlowsAcrossInlineBoxes(t *testing.T) {
	s := plainStyle()
	a := boxes.NewTextBox("p", s, nil, "a ")
	b := boxes.NewTextBox("em", s, nil, " b ")
	em := boxes.NewInlineBox("em", s, nil, []boxes.Box{b})
	c := boxes.NewTextBox("p", s, nil, " c")
	line := boxes.NewLineBox("p", s, nil, []boxes.Box{a, em, c})

	if boxes.ProcessWhitespace(line, false) {
		t.Error("the line does not end with a collapsible space")
	}
	if a.Text != "a " || b.Text != "b " || c.Text != "c" {
		t.Errorf("texts %q, %q, %q", a.Text, b.Text, c.Text)
	}
	if !b.LeadingCollapsibleSpace || !c.LeadingCollapsibleSpace || a.LeadingCollapsibleSpace {
		t.Error("leading collapsible spaces were not recorded")
	}
}

func TestProcessWhitespaceResetsAfterBlocksInFlow(t *testing.T) {
	s := plainStyle()
	a := boxes.NewTextBox("p", s, nil, "a ")
	block := boxes.NewBlockBox("div", s, nil, nil)
	c := boxes.NewTextBox("p", s, nil, " c")
	line := boxes.NewLineBox("p", s, nil, []boxes.Box{a, block, c})
	boxes.ProcessWhitespace(line, false)
	if c.Text != " c" {
		t.Errorf("text after a block = %q, want the space kept", c.Text)
	}
}

func TestProcessWhitespaceSkipsOutOfFlowBoxes(t *testing.T) {
	s := newStyled(t, `<style>span{float:left}</style><span></span>`)
	a := boxes.NewTextBox("p", plainStyle(), nil, "a ")
	f := boxes.NewBlockBox("span", s.style("span"), s.el("span"), nil)
	c := boxes.NewTextBox("p", plainStyle(), nil, " c")
	line := boxes.NewLineBox("p", plainStyle(), nil, []boxes.Box{a, f, c})
	boxes.ProcessWhitespace(line, false)
	if c.Text != "c" {
		t.Errorf("text after a float = %q, want the space collapsed", c.Text)
	}
}

func TestIsWhitespace(t *testing.T) {
	s := plainStyle()
	tests := []struct {
		text string
		want bool
	}{{" \t\n", true}, {"\u00a0", true}, {"\u2003", true}, {" a ", false}, {"\u200b", false}}
	for _, tt := range tests {
		if got := boxes.IsWhitespace(boxes.NewTextBox("p", s, nil, tt.text)); got != tt.want {
			t.Errorf("IsWhitespace(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
	if boxes.IsWhitespace(boxes.NewBlockBox("p", s, nil, nil)) {
		t.Error("a block is white space")
	}
}

func TestProcessTextTransform(t *testing.T) {
	s := newStyled(t, `<style>
.up{text-transform:uppercase} .cap{text-transform:capitalize} .tr{text-transform:uppercase}
.nohyph{hyphens:none} .running{position:running(x);text-transform:uppercase}
</style><p class=up lang=en-US></p><p class=cap></p><p class=tr lang=tr-TR></p>
<p class=nohyph></p><p class=running></p>`)
	ps := s.doc.Elements()
	var up, cap, tr, nohyph, running *boxes.TextBox
	var runningBox *boxes.BlockBox
	for _, el := range ps {
		mk := func(in string) *boxes.TextBox { return boxes.NewTextBox("p", s.sf.Get(el, ""), el, in) }
		switch el.Get("class") {
		case "up":
			up = mk("straße i")
		case "cap":
			cap = mk("hello world")
		case "tr":
			tr = mk("iı")
		case "nohyph":
			nohyph = mk("co\u00adoperate")
		case "running":
			running = mk("keep")
			runningBox = boxes.NewBlockBox("p", s.sf.Get(el, ""), el, []boxes.Box{running})
		}
	}
	tree := boxes.NewBlockBox("body", plainStyle(), nil, []boxes.Box{up, cap, tr, nohyph, runningBox})
	boxes.ProcessTextTransform(tree)

	tests := []struct{ name, got, want string }{
		{"uppercase", up.Text, "STRASSE I"},
		{"capitalize", cap.Text, "Hello World"},
		{"tr uppercase", tr.Text, "İI"},
		{"hyphens none", nohyph.Text, "cooperate"},
		{"running element", running.Text, "keep"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestContentText(t *testing.T) {
	s := plainStyle()
	before := boxes.NewTextBox("p::before", s, nil, "<<")
	marker := boxes.NewTextBox("li::marker", s, nil, "1.")
	a := boxes.NewTextBox("p", s, nil, "  Hello\n  ")
	b := boxes.NewTextBox("em", s, nil, "  big world ")
	em := boxes.NewInlineBox("em", s, nil, []boxes.Box{b})
	after := boxes.NewTextBox("p::after", s, nil, ">>")
	line := boxes.NewLineBox("p", s, nil, []boxes.Box{before, marker, a, em, after})
	block := boxes.NewBlockBox("p", s, nil, []boxes.Box{line})

	if got := boxes.ContentText(block); got != "Hello big world" {
		t.Errorf("ContentText = %q", got)
	}
	if a.Text != "  Hello\n  " {
		t.Errorf("ContentText changed the text: %q", a.Text)
	}
	if got := boxes.ContentText(a); got != "Hello" {
		t.Errorf("text box ContentText = %q", got)
	}
	if got := boxes.ContentText(before); got != "<<" {
		t.Errorf("generated text box ContentText = %q", got)
	}
	if got := boxes.ContentText(boxes.NewInlineReplacedBox("img", s, nil, nil)); got != "" {
		t.Errorf("replaced box ContentText = %q", got)
	}
}

func TestExtractText(t *testing.T) {
	s := plainStyle()
	before := boxes.NewTextBox("p::before", s, nil, " <<")
	a := boxes.NewTextBox("p", s, nil, "“Hello world")
	after := boxes.NewTextBox("p::after", s, nil, ">>")
	nested := boxes.NewInlineBox("p::after", s, nil, []boxes.Box{boxes.NewTextBox("p::after", s, nil, "!")})
	block := boxes.NewBlockBox("p", s, nil, []boxes.Box{before, a, after, nested})

	tests := []struct{ part, want string }{
		{"text", "“Hello world"},
		{"content", "“Hello world"},
		{"before", "<<"},
		{"after", ">>!"},
		{"first-letter", "“H"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		if got := boxes.ExtractText(block, tt.part); got != tt.want {
			t.Errorf("ExtractText(%q) = %q, want %q", tt.part, got, tt.want)
		}
	}
}
