package draw

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/boxes"
	"github.com/tomasweigenast/pdfgen/internal/testutil/boxtest"
	"github.com/tomasweigenast/pdfgen/internal/units"
)

// columnBox returns a block laid out as one column at the given position.
func columnBox(parent boxes.Box, x, y, height float64) boxes.Box {
	box := boxes.NewBlockBox("column", parent.Base().Style, nil, nil)
	b := box.Base()
	b.PositionX, b.PositionY, b.Height = x, y, units.F(height)
	return box
}

func TestColumnRulesAreDrawnBetweenColumns(t *testing.T) {
	root := boxtest.Parse(t, `
		<html style="columns: 3; column-gap: 4px; column-rule: 2px solid red"></html>`)
	root.Base().Width = units.F(40)
	parent := root.(boxes.Parent).ParentBase()
	parent.Children = []boxes.Box{
		columnBox(root, 0, 3, 10),
		columnBox(root, 14, 3, 10),
		columnBox(root, 28, 3, 10),
	}

	s := newTestStream()
	drawBorder(s, root)

	// Each rule is centered in the 4px gap before the second and third
	// columns, and spans the height of the column.
	got := string(s.Bytes())
	for _, want := range []string{
		"1 0 0 rg\n11 3 2 10 re\n13 3 0 10 re\nf*",
		"1 0 0 rg\n25 3 2 10 re\n27 3 0 10 re\nf*",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stream does not contain %q:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "f*"); n != 2 {
		t.Errorf("got %d rules, want 2:\n%s", n, got)
	}
}

func TestSpanningElementResetsColumnRules(t *testing.T) {
	root := boxtest.Parse(t, `
		<html style="columns: 2; column-gap: 4px; column-rule: 2px solid red">
		<body style="column-span: all"></html>`)
	var spanner boxes.Box
	for _, child := range root.(boxes.Parent).ParentBase().Children {
		if child.Base().Style.ColumnSpan() == "all" {
			spanner = child
		}
	}
	if spanner == nil {
		t.Fatal("no spanning child")
	}
	root.Base().Width = units.F(40)
	root.(boxes.Parent).ParentBase().Children = []boxes.Box{
		columnBox(root, 0, 0, 10),
		columnBox(root, 20, 0, 10),
		spanner,
		columnBox(root, 0, 20, 10),
		columnBox(root, 20, 20, 10),
	}

	s := newTestStream()
	drawBorder(s, root)

	// The first column of each row has no rule before it.
	got := string(s.Bytes())
	if n := strings.Count(got, "f*"); n != 2 {
		t.Errorf("got %d rules, want 2:\n%s", n, got)
	}
}

func TestColumnRuleWithoutColumnsIsIgnored(t *testing.T) {
	root := boxtest.Parse(t, `<html style="column-rule: 2px solid red"></html>`)
	s := newTestStream()
	drawBorder(s, root)
	if got := s.Bytes(); len(got) != 0 {
		t.Errorf("stream = %q, want empty", got)
	}
}
