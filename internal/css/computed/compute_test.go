package computed

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	"github.com/tomasweigenast/pdfgen/internal/css/validation"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

// declare builds the cascaded values of a style attribute.
func declare(css string) Cascaded {
	c := Cascaded{}
	for _, d := range validation.PreprocessDeclarations("", syntax.ParseBlocksContents(css, false, false)) {
		c[d.Name] = Declared{Value: d.Value, Weight: Weight{Precedence: 3, Layer: math.Inf(1)}}
	}
	return c
}

// family builds the styles of nested elements, each with its declarations.
func family(t *testing.T, css ...string) []*Style {
	t.Helper()
	return familyIn(t, NewContext(nil), css...)
}

// familyIn is like family, in a given context.
func familyIn(t testing.TB, ctx *Context, css ...string) []*Style {
	t.Helper()
	var styles []*Style
	var parent, root *Style
	for _, c := range css {
		s := NewStyle(ctx, parent, root, declare(c), "", "")
		if parent == nil {
			root = s
		}
		styles = append(styles, s)
		parent = s
	}
	return styles
}

func TestFontSizeKeywordsAndRelativeSizes(t *testing.T) {
	tests := []struct {
		parent, child string
		wantP, wantC  float64
	}{
		{"10px", "10px", 10, 10},
		{"x-small", "xx-large", 12, 32},
		{"x-large", "2em", 24, 48},
		{"1em", "1em", 16, 16},
		{"1em", "larger", 16, 6.0 / 5 * 16},
		{"medium", "larger", 16, 6.0 / 5 * 16},
		{"x-large", "larger", 24, 32},
		{"xx-large", "larger", 32, 1.2 * 32},
		{"1px", "larger", 1, 3.0 / 5 * 16},
		{"28px", "larger", 28, 32},
		{"100px", "larger", 100, 120},
		{"xx-small", "larger", 3.0 / 5 * 16, 12},
		{"1em", "smaller", 16, 8.0 / 9 * 16},
		{"medium", "smaller", 16, 8.0 / 9 * 16},
		{"x-large", "smaller", 24, 6.0 / 5 * 16},
		{"xx-large", "smaller", 32, 24},
		{"xx-small", "smaller", 3.0 / 5 * 16, 0.8 * 3 / 5 * 16},
		{"1px", "smaller", 1, 0.8},
		{"28px", "smaller", 28, 24},
		{"100px", "smaller", 100, 32},
		{"50%", "150%", 8, 12},
		{"calc(50% + 2px)", "calc(1em + 10%)", 10, 11},
	}
	for _, tt := range tests {
		s := family(t, "", "font-size:"+tt.parent, "font-size:"+tt.child)
		if got := float64(s[1].FontSize()); math.Abs(got-tt.wantP) > 1e-9 {
			t.Errorf("font-size: %s gives %v, want %v", tt.parent, got, tt.wantP)
		}
		if got := float64(s[2].FontSize()); math.Abs(got-tt.wantC) > 1e-9 {
			t.Errorf("font-size: %s in %s gives %v, want %v", tt.child, tt.parent, got, tt.wantC)
		}
	}
}

func TestFontWeightRelativeToParent(t *testing.T) {
	s := family(t, "font-weight: bold", "font-weight: bolder", "font-weight: lighter")
	for i, want := range []props.Integer{700, 900, 700} {
		if got := s[i].FontWeight(); got != want {
			t.Errorf("style %d: font-weight = %v, want %v", i, got, want)
		}
	}
}

func TestBorderWidthIsZeroWithoutStyle(t *testing.T) {
	s := family(t, "border-style: none solid hidden; border-width: thin thick 4px .25in")[0]
	got := []props.Pixels{s.BorderTopWidth(), s.BorderRightWidth(), s.BorderBottomWidth(), s.BorderLeftWidth()}
	want := []props.Pixels{0, 5, 0, 24}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("border widths = %v, want %v", got, want)
			break
		}
	}
	if w := s.OutlineWidth(); w != 0 {
		t.Errorf("outline-width = %v, want 0", w)
	}
}

func TestLengthsAreComputedToPixels(t *testing.T) {
	s := family(t, "font-size: 20px", "margin: 1em 2rem 1in 0; padding-left: 10%; width: calc(2em + 5px); height: calc(10% + 1em)")[1]
	if got := s.MarginTop(); got != props.Px(20) {
		t.Errorf("margin-top = %v", got)
	}
	if got := s.MarginRight(); got != props.Px(40) {
		t.Errorf("margin-right = %v", got)
	}
	if got := s.MarginBottom(); got != props.Px(96) {
		t.Errorf("margin-bottom = %v", got)
	}
	if got := s.MarginLeft(); got != props.Px(0) {
		t.Errorf("margin-left = %v", got)
	}
	if got := s.PaddingLeft(); got != (props.Length{Dimension: props.Dimension{Value: 10, Unit: "%"}}) {
		t.Errorf("padding-left = %v", got)
	}
	if got := s.Width(); got != props.Px(45) {
		t.Errorf("width = %v", got)
	}
	if got := s.Height(); got.Calc == nil {
		t.Errorf("height = %v, want a math function", got)
	}
}

func TestComputedValuesFromUnitsAndFunctions(t *testing.T) {
	s := familyIn(t, testFontContext(t), "font-family: testfont; font-size: 20px; line-height: 1.5", "font-size: 120%; line-height: normal; width: calc(2em + 5px); height: min(20px, 10rem); margin-left: max(2ex, 3ch); padding-top: clamp(1px, 2lh, 20px)")[1]
	if got := s.FontSize(); got != 24 {
		t.Errorf("font-size = %v, want 24px", got)
	}
	if got := s.LineHeight(); got.Keyword != "normal" {
		t.Errorf("line-height = %v, want normal", got)
	}
	if got := s.Width(); got != props.Px(53) {
		t.Errorf("width = %v, want 53px", got)
	}
	if got := s.Height(); got != props.Px(20) {
		t.Errorf("height = %v, want 20px", got)
	}
	if got := s.MarginLeft(); got != props.Px(72) {
		t.Errorf("margin-left = %v, want 72px", got)
	}
	if got := s.PaddingTop(); got != props.Px(20) {
		t.Errorf("padding-top = %v, want 20px", got)
	}
}

func TestGeneratedGettersMatchPropertyValues(t *testing.T) {
	s := family(t, "", "color: red; display: inline; width: 12px; opacity: .5")[1]
	value := reflect.ValueOf(s)
	checked := 0
	for id := props.PropID(0); int(id) < props.NumProps(); id++ {
		var name strings.Builder
		upper := true
		for _, r := range id.Name() {
			if r == '-' {
				upper = true
				continue
			}
			if upper {
				r = []rune(strings.ToUpper(string(r)))[0]
				upper = false
			}
			name.WriteRune(r)
		}
		method := value.MethodByName(name.String())
		if !method.IsValid() {
			continue
		}
		got := method.Call(nil)[0].Interface()
		want := s.Get(id)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s() = %#v, Get(%q) = %#v", name.String(), got, id.Name(), want)
		}
		checked++
	}
	if checked < props.NumProps()-10 {
		t.Fatalf("checked %d generated getters for %d properties", checked, props.NumProps())
	}
}

func TestComputedImagesPositionAndGeneratedContent(t *testing.T) {
	s := family(t, "", "background-image: linear-gradient(red, blue); background-position: right 2px bottom 3px; background-size: 10px auto; box-shadow: 1px 2px 3px red; color-scheme: light dark; break-before: page; bleed-top: 2mm; column-gap: 2em; border-image-outset: 2; border-image-source: url(https://example.com/thing.png); image-orientation: 90deg; object-position: right 2px bottom 3px; vertical-align: 2em; transform: translateX(1em); content: 'chapter'; bookmark-label: content(); string-set: title content(); grid-template-columns: 10px 1fr; grid-auto-rows: auto")[1]
	if got := s.BackgroundImage(); len(got) != 1 || got[0].Type != "linear-gradient" {
		t.Errorf("background-image = %v", got)
	}
	if got := s.BackgroundPosition(); len(got) != 1 || got[0].X != props.Px(2) {
		t.Errorf("background-position = %v", got)
	}
	if got := s.BackgroundSize(); len(got) != 1 || got[0].Width != props.Px(10) {
		t.Errorf("background-size = %v", got)
	}
	if got := s.BoxShadow(); len(got) != 1 || got[0].X != props.Px(1) {
		t.Errorf("box-shadow = %v", got)
	}
	if got := s.ColorScheme(); len(got) != 2 || got[0] != "light" || got[1] != "dark" {
		t.Errorf("color-scheme = %v", got)
	}
	if got := s.BreakBefore(); got != "page" || math.Abs(float64(s.BleedTop().Dimension.Value)-96.0/25.4*2) > 1e-9 || s.ColumnGap() != props.Px(32) {
		t.Errorf("break-before = %q, bleed-top = %v, column-gap = %v", got, s.BleedTop(), s.ColumnGap())
	}
	if got := s.BorderImageOutset(); got[0].Dimension.Value != 2 {
		t.Errorf("border-image-outset = %v", got)
	}
	if got := s.BorderImageSource(); got.URL != "https://example.com/thing.png" {
		t.Errorf("border-image-source = %v", got)
	}
	if got := s.ImageOrientation(); got.Angle != 90 {
		t.Errorf("image-orientation = %v", got)
	}
	if got := s.Content(); len(got.Items) != 1 || got.Items[0].Args[0] != "chapter" {
		t.Errorf("content = %v", got)
	}
	if got := s.BookmarkLabel(); len(got) == 0 || len(s.StringSet()) == 0 {
		t.Errorf("bookmark-label = %v, string-set = %v", got, s.StringSet())
	}
	if got := s.GridTemplateColumns(); len(got.Items) == 0 || len(s.GridAutoRows()) == 0 {
		t.Errorf("grid tracks = %v / %v", got, s.GridAutoRows())
	}
	if got := s.VerticalAlign(); got != props.Px(32) {
		t.Errorf("vertical-align = %v", got)
	}
	if got := s.Transform(); len(got) == 0 {
		t.Errorf("transform = %v", got)
	}
}

func TestInvalidMathFunctionFallsBack(t *testing.T) {
	logs := logging.StartCapture()
	defer logs.Close()
	s := family(t, "", "padding-left: calc(-1em)")[1]
	if got := s.PaddingLeft(); got != props.Px(0) {
		t.Errorf("padding-left = %v, want initial 0", got)
	}
	if lines := logs.Lines(); len(lines) != 1 || lines[0] != "WARNING: Invalid math function at 1:15: calc(-1em)" {
		t.Errorf("logs = %q", lines)
	}
}

func TestDisplayIsBlockified(t *testing.T) {
	tests := []struct {
		css  string
		want props.Display
	}{
		{"display: inline", props.Display{"inline", "flow"}},
		{"display: inline; float: left", props.Display{"block", "flow"}},
		{"display: inline-table; position: absolute", props.Display{"block", "flow"}},
		{"display: table-cell; position: fixed", props.Display{"block", "flow"}},
		{"display: inline list-item; float: right", props.Display{"block", "flow", "list-item"}},
		{"display: inline-block; float: left", props.Display{"block", "flow"}},
		{"display: flex; float: left", props.Display{"block", "flex"}},
	}
	for _, tt := range tests {
		got := family(t, "", tt.css)[1].Display()
		if len(got) != len(tt.want) {
			t.Errorf("%s: display = %v, want %v", tt.css, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%s: display = %v, want %v", tt.css, got, tt.want)
				break
			}
		}
	}
	if got := family(t, "display: inline")[0].Display(); got[0] != "block" {
		t.Errorf("root display = %v, want block", got)
	}
	if got := family(t, "", "float: left; position: absolute")[1].Float(); got != "none" {
		t.Errorf("float of absolute box = %v, want none", got)
	}
}

func TestLogicalPropertiesSetPhysicalOnes(t *testing.T) {
	s := family(t, "", "margin-inline-start: 5px; padding-block-end: 3px; inline-size: 7px; border-start-end-radius: 2px")[1]
	if got := s.MarginLeft(); got != props.Px(5) {
		t.Errorf("margin-left = %v", got)
	}
	if got := s.PaddingBottom(); got != props.Px(3) {
		t.Errorf("padding-bottom = %v", got)
	}
	if got := s.Width(); got != props.Px(7) {
		t.Errorf("width = %v", got)
	}
	if got := s.BorderTopRightRadius(); got.X != props.Px(2) {
		t.Errorf("border-top-right-radius = %v", got)
	}

	rtl := family(t, "", "direction: rtl; margin-inline-start: 5px")[1]
	if got := rtl.MarginRight(); got != props.Px(5) {
		t.Errorf("rtl margin-right = %v", got)
	}
}

func TestCustomPropertiesAreSubstituted(t *testing.T) {
	s := family(t, "--foo: 10px; --size: 3px", "--var: var(--foo); --SIZE: 4px", "width: var(--var); height: var(--x, var(--SIZE)); min-width: var(--size)")
	leaf := s[2]
	if got := leaf.Width(); got != props.Px(10) {
		t.Errorf("width = %v, want 10px", got)
	}
	if got := leaf.Height(); got != props.Px(4) {
		t.Errorf("height = %v, want 4px", got)
	}
	if got := leaf.MinWidth(); got != props.Px(3) {
		t.Errorf("min-width = %v, want 3px", got)
	}
}

func TestCustomPropertyCyclesAreInvalid(t *testing.T) {
	logs := logging.StartCapture()
	defer logs.Close()
	s := family(t, "--a: var(--b); --b: var(--a); padding-top: var(--a); --self: var(--self)")[0]
	if got := s.PaddingTop(); got != props.Px(0) {
		t.Errorf("padding-top = %v, want 0", got)
	}
	if got := s.Custom("--self"); len(got) != 0 {
		t.Errorf("--self = %v, want nothing", got)
	}
}

func TestInvalidVariableValueUsesInheritedOrInitial(t *testing.T) {
	logs := logging.StartCapture()
	defer logs.Close()
	s := family(t, "--v: blue; color: red", "margin: 0 0 0 var(--v); color: var(--v2)")[1]
	if got := s.MarginLeft(); got != props.Px(0) {
		t.Errorf("margin-left = %v, want 0", got)
	}
	if got := s.Color(); got.Coordinates[0] != 1 {
		t.Errorf("color = %v, want inherited red", got)
	}
	lines := logs.Lines()
	if len(lines) != 2 {
		t.Fatalf("logs = %q, want 2 warnings", lines)
	}
}

func TestTextDecorationsArePropagated(t *testing.T) {
	s := family(t, "text-decoration: underline red", "text-decoration-line: overline", "")
	if got := s[1].TextDecorationLine(); len(got) != 2 {
		t.Errorf("lines = %v, want overline and underline", got)
	}
	if got := s[2].TextDecorationLine(); len(got) != 2 {
		t.Errorf("propagated lines = %v", got)
	}
	if got := s[2].TextDecorationColor(); got.Coordinates[0] != 1 {
		t.Errorf("propagated color = %v, want red", got)
	}
}

func TestNamedPageIsTakenFromAncestors(t *testing.T) {
	s := family(t, "page: auto", "page: NARRow", "page: auto")
	for i, want := range []props.Keyword{"", "NARRow", "NARRow"} {
		if got := s[i].Page(); got != want {
			t.Errorf("style %d: page = %q, want %q", i, got, want)
		}
	}
}

func TestContentKeywords(t *testing.T) {
	ctx := NewContext(nil)
	root := NewStyle(ctx, nil, nil, nil, "", "")
	element := NewStyle(ctx, root, root, declare("content: normal"), "", "")
	before := NewStyle(ctx, element, root, declare("content: normal"), "before", "")
	none := NewStyle(ctx, element, root, declare("content: none"), "after", "")
	if got := none.Content(); got.Keyword != "inhibit" {
		t.Errorf("content: none = %v", got)
	}
	if got := NewStyle(ctx, root, root, nil, "", "").Content(); got.Keyword != "normal" {
		t.Errorf("initial content = %v, want normal", got)
	}
	if got := element.Content(); got.Keyword != "contents" {
		t.Errorf("element content = %v", got)
	}
	if got := before.Content(); got.Keyword != "inhibit" {
		t.Errorf("pseudo-element content = %v", got)
	}
}

func TestAnonymousStyleInheritsAndResets(t *testing.T) {
	s := family(t, "color: red; margin-top: 5px; border: 3px solid; --x: 1px")[0]
	a := s.Anonymous()
	if a != s.Anonymous() {
		t.Error("anonymous style is not shared")
	}
	if got := a.Color(); got.Coordinates[0] != 1 {
		t.Errorf("anonymous color = %v, want inherited red", got)
	}
	if got := a.MarginTop(); got != props.Px(0) {
		t.Errorf("anonymous margin-top = %v, want 0", got)
	}
	if got := a.BorderTopWidth(); got != 0 {
		t.Errorf("anonymous border width = %v, want 0", got)
	}
	if got := a.Custom("--x"); len(got) != 1 {
		t.Errorf("anonymous --x = %v", got)
	}
}

func TestViewportUnitsUsePageSizes(t *testing.T) {
	logs := logging.StartCapture()
	defer logs.Close()
	ctx := NewContext(nil)
	root := NewStyle(ctx, nil, nil, declare("width: 10vw"), "", "")
	if got := root.Width(); math.Abs(pixelsOf(got)-79.37) > 0.01 {
		t.Errorf("width before layout = %v, want 10%% of A4", got)
	}
	if lines := logs.Lines(); len(lines) != 1 || lines[0] != "WARNING: vw unit resolved before first page layout" {
		t.Errorf("logs = %q", lines)
	}
	ctx.PageBox = &Size{300, 200}
	ctx.PageArea = &Size{200, 100}
	s := NewStyle(ctx, root, root, declare("margin: 50vh 50pvh 20vmax 10pvmin"), "", "")
	got := []props.Length{s.MarginTop(), s.MarginRight(), s.MarginBottom(), s.MarginLeft()}
	want := []props.Length{props.Px(50), props.Px(100), props.Px(40), props.Px(20)}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("margins = %v, want %v", got, want)
			break
		}
	}
}
