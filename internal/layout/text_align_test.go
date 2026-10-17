package layout

import (
	"fmt"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// paragraphLine returns the line box of a paragraph, with the given width as
// if its content had been laid out.
func paragraphLine(t *testing.T, style, contents string, width float64) *boxes.LineBox {
	t.Helper()
	paragraph := parseBox(t, fmt.Sprintf(`<p style="font: 10px testfont; %s">%s`, style, contents), "p")
	line, ok := boxes.ChildrenOf(paragraph)[0].(*boxes.LineBox)
	if !ok {
		t.Fatalf("paragraph child is not a line: %v", boxes.ChildrenOf(paragraph))
	}
	line.Width = units.F(width)
	return line
}

func TestTextAlignOffset(t *testing.T) {
	// A line of 100px in 300px. The offset is in the direction of the text:
	// the caller mirrors it for right-to-left lines.
	for _, tt := range []struct {
		name, style string
		last        bool
		want        float64
	}{
		{"start", "text-align: start", false, 0},
		{"end", "text-align: end", false, 200},
		{"left", "text-align: left", false, 0},
		{"right", "text-align: right", false, 200},
		{"center", "text-align: center", false, 100},
		{"left in rtl", "direction: rtl; text-align: left", false, 200},
		{"right in rtl", "direction: rtl; text-align: right", false, 0},
		{"start in rtl", "direction: rtl; text-align: start", false, 0},
		{"end in rtl", "direction: rtl; text-align: end", false, 200},
		{"justify keeps its place", "text-align: justify", false, 0},
		{"last line", "text-align: center; text-align-last: right", true, 200},
		{"not the last line", "text-align: center; text-align-last: right", false, 100},
		{"last line with auto", "text-align: center; text-align-last: auto", true, 100},
		{"justify with a different last line", "text-align: justify; text-align-last: end", true, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			line := paragraphLine(t, tt.style, "a b", 100)
			if got := textAlign(&Context{}, line, 300, tt.last); got != tt.want {
				t.Fatalf("offset = %g, want %g", got, tt.want)
			}
		})
	}
}

func TestTextAlignNeedsRoomToDistribute(t *testing.T) {
	line := paragraphLine(t, "text-align: center", "a", 100)
	for _, available := range []float64{100, 50, 0} {
		if got := textAlign(&Context{}, line, available, false); got != 0 {
			t.Errorf("offset in %g px = %g, want 0", available, got)
		}
	}
}

func TestJustificationOnlyWidensCollapsingWhiteSpace(t *testing.T) {
	for _, tt := range []struct {
		whiteSpace string
		justified  bool
	}{
		{"normal", true},
		{"nowrap", true},
		{"pre-line", true},
		{"pre", false},
		{"pre-wrap", false},
	} {
		t.Run(tt.whiteSpace, func(t *testing.T) {
			line := paragraphLine(t, "text-align: justify; white-space: "+tt.whiteSpace, "a b c", 50)
			text := boxes.ChildrenOf(line)[0].(*boxes.TextBox)
			text.Width = units.F(50)
			ctx := &Context{Text: newLayoutText(text)}
			textAlign(ctx, line, 70, false)
			if got := text.JustificationSpacing != 0; got != tt.justified {
				t.Fatalf("justification spacing = %g, widened = %v, want %v", text.JustificationSpacing, got, tt.justified)
			}
		})
	}
}

func TestCountExpandableSpaces(t *testing.T) {
	line := paragraphLine(t, "", "a b&nbsp;c <em>d e</em><img src=pattern.png> f", 0)
	if got := countExpandableSpaces(line); got != 5 {
		t.Fatalf("expandable spaces = %d, want 5", got)
	}
	if got := countExpandableSpaces(boxes.ChildrenOf(line)[1]); got != 1 {
		t.Fatalf("expandable spaces in the em = %d, want 1", got)
	}
}
