package validation

import (
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
	tk "github.com/tomasweigenast/pdfgen/internal/css/tokens"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

func preprocess1(t *testing.T, css string) []Declaration {
	t.Helper()
	return PreprocessDeclarations(testBaseURL, syntax.ParseBlocksContents(css, false, false))
}

func TestImportantAndUnknownProperties(t *testing.T) {
	got := preprocess1(t, "margin-top: 1px !important")
	if len(got) != 1 || !got[0].Important || got[0].Name != "margin-top" {
		t.Errorf("got %#v", got)
	}
	assertInvalid(t, "foo: 1", "unknown property")
	assertInvalid(t, "unicode-bidi: normal", "property not supported yet")
}

func TestCustomPropertyKeepsItsTokens(t *testing.T) {
	got := preprocess1(t, "--Foo-Bar: 1px  solid")
	if len(got) != 1 || got[0].Name != "--Foo-Bar" {
		t.Fatalf("got %#v", got)
	}
	if tokens, ok := got[0].Value.(props.Tokens); !ok || len(tokens) != 2 {
		t.Errorf("value = %#v", got[0].Value)
	}
}

func TestValueWithVarIsPending(t *testing.T) {
	got := preprocess1(t, "margin-top: var(--a)")
	pending, ok := got[0].Value.(*props.Pending)
	if !ok || pending.Name != "margin-top" {
		t.Fatalf("got %#v", got)
	}

	got = preprocess1(t, "margin: calc(var(--a) + 1px)")
	if len(got) != 4 {
		t.Fatalf("got %d declarations, want 4", len(got))
	}
	shared := got[0].Value.(*props.Pending)
	for _, d := range got {
		if d.Value != shared {
			t.Errorf("%s does not share the pending value", d.Name)
		}
	}
}

func TestSolvePendingValuesWarnsOnce(t *testing.T) {
	got := preprocess1(t, "margin: var(--a)")
	pending := got[0].Value.(*props.Pending)
	tokens := tk.RemoveWhitespace(syntax.ParseComponentValueList("red", true))

	capture := logging.StartCapture()
	for _, d := range got[:2] {
		if _, err := Solve(pending, tokens, d.Name, testBaseURL); err == nil {
			t.Errorf("%s: want an error", d.Name)
		}
	}
	capture.Close()
	if logs := capture.Lines(); len(logs) != 1 || !strings.Contains(logs[0], "Ignored `margin: red`") {
		t.Errorf("logs = %q", logs)
	}

	tokens = tk.RemoveWhitespace(syntax.ParseComponentValueList("1px 2px", true))
	v, err := Solve(pending, tokens, "margin-left", testBaseURL)
	if err != nil || v != px(2) {
		t.Errorf("Solve = %#v, %v", v, err)
	}
}

func TestSolvePendingProperty(t *testing.T) {
	pending := preprocess1(t, "width: var(--a)")[0].Value.(*props.Pending)
	v, err := Solve(pending, tk.RemoveWhitespace(syntax.ParseComponentValueList("3px", true)), "width", testBaseURL)
	if err != nil || v != px(3) {
		t.Errorf("Solve = %#v, %v", v, err)
	}
}

func TestParseErrorsAreLogged(t *testing.T) {
	capture := logging.StartCapture()
	preprocess1(t, "color red")
	capture.Close()
	if logs := capture.Lines(); len(logs) != 1 || !strings.HasPrefix(logs[0], "WARNING: Error: ") {
		t.Errorf("logs = %q", logs)
	}
}

func TestRuleDeclarationsCarryTheirSelectors(t *testing.T) {
	rule := syntax.ParseStylesheet("a, b > c { color: red; & d { color: blue } e { width: 1px } }", false, false)[0].(*syntax.QualifiedRule)
	got, err := PreprocessRuleDeclarations(testBaseURL, syntax.ParseBlocksContents(rule.Content, false, false), rule.Prelude)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d declarations, want 3", len(got))
	}
	for i, d := range got {
		if len(d.Selectors) < 1 {
			t.Errorf("declaration %d has no selectors", i)
		}
	}
	if len(got[0].Selectors) != 2 || got[1].Selectors[0].Specificity == [3]int{} {
		t.Errorf("selectors = %#v", got)
	}
}

func TestInvalidRulePreludeIsAnError(t *testing.T) {
	_, err := PreprocessRuleDeclarations(testBaseURL, nil, syntax.ParseComponentValueList("a >", false))
	if err == nil || err.Error() != "'a >'" {
		t.Errorf("err = %v", err)
	}
}
