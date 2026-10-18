package syntax

import "testing"

func FuzzTokenize(f *testing.F) {
	for _, seed := range []string{
		`a{color:red}`, `url("a b.png")`, `url(a\ b.png)`, `1e2px`, `/* open`, `"open`, `\\26 `,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64<<10 {
			t.Skip()
		}
		nodes := ParseComponentValueList(input, false)
		if containsParseError(nodes) {
			t.Skip()
		}
		serialized := Serialize(nodes)
		canonical := Serialize(ParseComponentValueList(serialized, false))
		if got := Serialize(ParseComponentValueList(canonical, false)); got != canonical {
			t.Fatalf("serialization does not converge: %q became %q", canonical, got)
		}
	})
}

func containsParseError(nodes []Node) bool {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ParseError:
			return true
		case *ParenthesesBlock:
			if containsParseError(n.Content) {
				return true
			}
		case *SquareBracketsBlock:
			if containsParseError(n.Content) {
				return true
			}
		case *CurlyBracketsBlock:
			if containsParseError(n.Content) {
				return true
			}
		case *FunctionBlock:
			if containsParseError(n.Arguments) {
				return true
			}
		}
	}
	return false
}

func FuzzParseStylesheet(f *testing.F) {
	for _, seed := range []string{`a { color: red }`, `@media print { p { margin: 1px } }`, `a{`, `/*x*/`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64<<10 {
			t.Skip()
		}
		ParseStylesheet(input, true, true)
	})
}

func FuzzParseDeclarationList(f *testing.F) {
	for _, seed := range []string{`color: red; margin: 1px`, `--x: {a:b};`, `!important`, `font: 12px serif`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64<<10 {
			t.Skip()
		}
		ParseDeclarationList(input, true, true)
	})
}
