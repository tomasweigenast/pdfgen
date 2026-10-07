package syntax

import "testing"

func TestSkipOptionsDropOnlyWhatTheyName(t *testing.T) {
	source := `
    /* foo */
    @media print {
        #foo {
            width: /* bar*/4px;
            color: green;
        }
    }
    `
	noWS := ParseStylesheet(source, false, true)
	noComment := ParseStylesheet(source, true, false)
	all := ParseComponentValueList(source, false)
	if Serialize(noWS) == source {
		t.Error("skipping whitespace did not change the output")
	}
	if Serialize(noComment) == source {
		t.Error("skipping comments did not change the output")
	}
	if got := Serialize(all); got != source {
		t.Errorf("round trip = %q", got)
	}
}

func TestUnterminatedCommentIsClosedOnSerialization(t *testing.T) {
	got := Serialize(ParseComponentValueList("/* foo ", false))
	if got != "/* foo */" {
		t.Errorf("got %q", got)
	}
}

func TestDeclarationSerializesBackToSource(t *testing.T) {
	source := "color:#369"
	decl, ok := ParseOneDeclaration(source, false).(*Declaration)
	if !ok {
		t.Fatal("not a declaration")
	}
	if len(decl.Value) != 1 {
		t.Fatalf("value = %v", decl.Value)
	}
	if got := decl.Serialize(); got != source {
		t.Errorf("got %q", got)
	}
}

func TestSerializeRoundTrips(t *testing.T) {
	tests := []struct {
		name  string
		parse func(string) []Node
		src   string
	}{
		{"rules", func(s string) []Node { return ParseRuleList(s, false, false) },
			`@import "a.css"; foo#bar.baz { color: red } /**/ @media print{}`},
		{"declarations", func(s string) []Node { return ParseBlocksContents(s, false, false) },
			`color: #123; /**/ @top-left {} width:7px !important;`},
		{"nested functions", func(s string) []Node { return ParseRuleList(s, false, false) }, `
        foo#bar.baz {
            background: url();
            color: rgb(0, 0, 0);
            width: calc(calc());
            height: calc(calc(calc()));
        }
    `},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Serialize(tc.parse(tc.src)); got != tc.src {
				t.Errorf("got %q, want %q", got, tc.src)
			}
		})
	}
}

func TestBackslashDelimiterBeforeNewline(t *testing.T) {
	source := "\\\nfoo"
	tokens := ParseComponentValueList(source, false)
	types := func() (s []string) {
		for _, n := range tokens {
			s = append(s, n.Type())
		}
		return
	}
	if got := types(); len(got) != 3 || got[0] != "literal" || got[1] != "whitespace" || got[2] != "ident" {
		t.Fatalf("types = %v", got)
	}
	if !IsLiteral(tokens[0], `\`) {
		t.Error("first token is not a backslash")
	}
	tokens = append(tokens[:1], tokens[2:]...)
	if got := Serialize(tokens); got != source {
		t.Errorf("got %q, want %q", got, source)
	}
}

func TestEscapesAreDecoded(t *testing.T) {
	dim := ParseComponentValueList(`0\e9 x`, false)
	if len(dim) != 1 {
		t.Fatalf("tokens = %v", dim)
	}
	d, ok := dim[0].(*DimensionToken)
	if !ok || !d.IsInteger || d.IntValue != 0 || d.Unit != "éx" {
		t.Errorf("got %#v", dim[0])
	}
	fn, ok := ParseComponentValueList(`\e9()`, false)[0].(*FunctionBlock)
	if !ok || fn.Name != "é" {
		t.Errorf("got %#v", fn)
	}
	at, ok := ParseRuleList("@\\e9 ", false, false)[0].(*AtRule)
	if !ok || at.AtKeyword != "é" {
		t.Errorf("got %#v", at)
	}
}

func TestSurrogateEscapeGivesReplacementCharacter(t *testing.T) {
	id := ParseComponentValueList(`\d800 `, false)[0].(*IdentToken)
	if id.Value != "�" {
		t.Errorf("got %q", id.Value)
	}
}

func TestPositionsCountCodePoints(t *testing.T) {
	nodes := ParseComponentValueList("é a\r\n  b\n\tc", false)
	want := [][2]int{{1, 1}, {1, 2}, {1, 3}, {1, 4}, {2, 3}, {2, 4}, {3, 2}}
	if len(nodes) != len(want) {
		t.Fatalf("got %d nodes", len(nodes))
	}
	for i, n := range nodes {
		if l, c := n.Pos(); [2]int{l, c} != want[i] {
			t.Errorf("node %d (%s) at %d:%d, want %v", i, n.Type(), l, c, want[i])
		}
	}
}

func TestParseNodeListInput(t *testing.T) {
	tokens := ParseComponentValueList("a: b; c: d", false)
	got := ParseBlocksContents(tokens, false, true)
	if len(got) != 2 {
		t.Fatalf("got %d nodes", len(got))
	}
	for _, n := range got {
		if _, ok := n.(*Declaration); !ok {
			t.Errorf("not a declaration: %s", n.Type())
		}
	}
}

func TestNestedRuleInBlock(t *testing.T) {
	got := ParseBlocksContents("a: b; &:hover { c: d } e: f", true, true)
	var types []string
	for _, n := range got {
		types = append(types, n.Type())
	}
	want := []string{"declaration", "qualified-rule", "declaration"}
	if len(types) != len(want) {
		t.Fatalf("types = %v", types)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("types = %v, want %v", types, want)
		}
	}
}
