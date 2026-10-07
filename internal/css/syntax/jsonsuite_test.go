package syntax

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// toJSON renders nodes in the shape used by the shared parsing test suite.
func toJSON(v any) any {
	list := func(nodes []Node) []any {
		out := make([]any, 0, len(nodes))
		for _, n := range nodes {
			out = append(out, toJSON(n))
		}
		return out
	}
	numeric := func(n Numeric) []any {
		kind := "number"
		if n.IsInteger {
			kind = "integer"
		}
		return []any{n.Representation, n.Value, kind}
	}
	switch n := v.(type) {
	case nil:
		return nil
	case []Node:
		if n == nil {
			return nil
		}
		return list(n)
	case *ParseError:
		return []any{"error", n.Kind}
	case *Comment:
		return "/* … */"
	case *WhitespaceToken:
		return " "
	case *LiteralToken:
		return n.Value
	case *IdentToken:
		return []any{"ident", n.Value}
	case *AtKeywordToken:
		return []any{"at-keyword", n.Value}
	case *HashToken:
		kind := "unrestricted"
		if n.IsIdentifier {
			kind = "id"
		}
		return []any{"hash", n.Value, kind}
	case *StringToken:
		return []any{"string", n.Value}
	case *URLToken:
		return []any{"url", n.Value}
	case *NumberToken:
		return append([]any{"number"}, numeric(n.Numeric)...)
	case *PercentageToken:
		return append([]any{"percentage"}, numeric(n.Numeric)...)
	case *DimensionToken:
		return append(append([]any{"dimension"}, numeric(n.Numeric)...), n.Unit)
	case *UnicodeRangeToken:
		return []any{"unicode-range", n.Start, n.End}
	case *CurlyBracketsBlock:
		return append([]any{"{}"}, list(n.Content)...)
	case *SquareBracketsBlock:
		return append([]any{"[]"}, list(n.Content)...)
	case *ParenthesesBlock:
		return append([]any{"()"}, list(n.Content)...)
	case *FunctionBlock:
		return append([]any{"function", n.Name}, list(n.Arguments)...)
	case *Declaration:
		return []any{"declaration", n.Name, list(n.Value), n.Important}
	case *AtRule:
		return []any{"at-rule", n.AtKeyword, list(n.Prelude), toJSON(n.Content)}
	case *QualifiedRule:
		return []any{"qualified rule", list(n.Prelude), list(n.Content)}
	}
	return v
}

// normalize converts every number to float64 so that values can be compared
// with decoded JSON.
func normalize(v any) any {
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalize(x[i])
		}
		return out
	case int:
		return float64(x)
	}
	return v
}

type jsonCase struct {
	input    json.RawMessage
	expected any
}

func loadSuite(t *testing.T, name string) []jsonCase {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatal(err)
	}
	var cases []jsonCase
	for i := 0; i+1 < len(items); i += 2 {
		var expected any
		if err := json.Unmarshal(items[i+1], &expected); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, jsonCase{items[i], expected})
	}
	return cases
}

// runSuite runs fn over every string input of a suite file and compares the
// result with the expected value.
func runSuite(t *testing.T, name string, fn func(css string) any) {
	t.Helper()
	cases := loadSuite(t, name)
	if len(cases) == 0 {
		t.Fatalf("%s: no cases", name)
	}
	for i, c := range cases {
		var css string
		if err := json.Unmarshal(c.input, &css); err != nil {
			t.Fatal(err)
		}
		got := normalize(toJSON(fn(css)))
		if !reflect.DeepEqual(got, c.expected) {
			t.Errorf("%s case %d %q:\n got  %#v\n want %#v", name, i, css, got, c.expected)
		}
	}
	t.Logf("%s: %d cases", name, len(cases))
}

func TestComponentValueList(t *testing.T) {
	runSuite(t, "component_value_list.json", func(css string) any {
		return ParseComponentValueList(css, true)
	})
}

func TestOneComponentValue(t *testing.T) {
	runSuite(t, "one_component_value.json", func(css string) any {
		return ParseOneComponentValue(css, true)
	})
}

func TestDeclarationList(t *testing.T) {
	runSuite(t, "declaration_list.json", func(css string) any {
		return ParseDeclarationList(css, true, true)
	})
}

func TestBlocksContents(t *testing.T) {
	runSuite(t, "blocks_contents.json", func(css string) any {
		return ParseBlocksContents(css, true, true)
	})
}

func TestOneDeclaration(t *testing.T) {
	runSuite(t, "one_declaration.json", func(css string) any {
		return ParseOneDeclaration(css, true)
	})
}

func TestStylesheet(t *testing.T) {
	runSuite(t, "stylesheet.json", func(css string) any {
		return ParseStylesheet(css, true, true)
	})
}

func TestRuleList(t *testing.T) {
	runSuite(t, "rule_list.json", func(css string) any {
		return ParseRuleList(css, true, true)
	})
}

func TestOneRule(t *testing.T) {
	runSuite(t, "one_rule.json", func(css string) any {
		return ParseOneRule(css, true)
	})
}

func TestSerializationRoundTrip(t *testing.T) {
	runSuite(t, "component_value_list.json", func(css string) any {
		parsed := ParseComponentValueList(css, true)
		return ParseComponentValueList(Serialize(parsed), true)
	})
}
