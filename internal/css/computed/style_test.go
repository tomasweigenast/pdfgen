package computed

import (
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
)

func TestUnsetPropertiesFallBackToInitialValues(t *testing.T) {
	s := New(nil)
	if got := s.FontSize(); got != 16 {
		t.Errorf("FontSize() = %v, want 16", got)
	}
	if got := s.Width(); !got.IsAuto() {
		t.Errorf("Width() = %#v, want auto", got)
	}
	if got := s.Display(); len(got) != 2 || got[0] != "inline" || got[1] != "flow" {
		t.Errorf("Display() = %v, want [inline flow]", got)
	}
}

func TestInheritedPropertiesComeFromTheParent(t *testing.T) {
	parent := New(nil)
	parent.Set(props.PropFontSize, props.Pixels(20))
	parent.Set(props.PropMarginTop, props.Px(5))
	child := New(parent)

	if got := child.FontSize(); got != 20 {
		t.Errorf("inherited FontSize() = %v, want 20", got)
	}
	if got := child.MarginTop(); got != props.Px(0) {
		t.Errorf("non-inherited MarginTop() = %#v, want 0px", got)
	}
	if child.Parent() != parent {
		t.Error("Parent() does not return the parent style")
	}
}

func TestSetOverridesInheritance(t *testing.T) {
	parent := New(nil)
	parent.Set(props.PropFontSize, props.Pixels(20))
	child := New(parent)
	child.Set(props.PropFontSize, props.Pixels(10))
	if got := child.FontSize(); got != 10 {
		t.Errorf("FontSize() = %v, want 10", got)
	}
}

func TestGettersReturnInitialValuesOfTheirType(t *testing.T) {
	s := New(nil)
	if got := s.Color(); got.IsCurrentColor() || got.Alpha != 1 {
		t.Errorf("Color() = %#v, want opaque black", got)
	}
	if got := s.FontFamily(); len(got) != 1 || got[0] != "serif" {
		t.Errorf("FontFamily() = %v, want [serif]", got)
	}
}
