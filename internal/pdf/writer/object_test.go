package writer

import (
	"math"
	"strings"
	"testing"
)

func serialize(o Object) string { return string(appendObject(nil, o, false)) }

func TestRealFormatting(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{2, "2"},
		{2.3456, "2.3456"},
		{0.5, "0.5"},
		{-1.25, "-1.25"},
		{1e-7, "0."},
		{-1e-7, "-0."},
		{5e-7, "0."},
		{123456.7890123, "123456.789012"},
		{1e20, "100000000000000000000"},
		{0.1 + 0.2, "0.3"},
		{100.0000004, "100."},
		{0.999999999, "1."},
		{math.NaN(), "nan"},
		{math.Inf(1), "inf"},
		{math.Inf(-1), "-inf"},
	}
	for _, tt := range tests {
		if got := serialize(Real(tt.in)); got != tt.want {
			t.Errorf("Real(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestScalarSerialization(t *testing.T) {
	tests := []struct {
		in   Object
		want string
	}{
		{Null{}, "null"},
		{nil, "null"},
		{Bool(true), "true"},
		{Bool(false), "false"},
		{Int(-42), "-42"},
		{Ref{Num: 7}, "7 0 R"},
		{Name("Type"), "/Type"},
		{Name("A B"), "/A#20B"},
		{Name("a/b#c"), "/a#2Fb#23c"},
		{Name("é"), "/#C3#A9"},
		{Array{Int(1), Real(2.5), Name("X"), Ref{3}, Array{}}, "[1 2.5 /X 3 0 R []]"},
		{Array{}, "[]"},
	}
	for _, tt := range tests {
		if got := serialize(tt.in); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestStringEncoding(t *testing.T) {
	tests := []struct {
		in   String
		want string
	}{
		{Text("abc"), "(abc)"},
		{Text("déf"), "<feff006400e90066>"},
		{Text("♡"), "<feff2661>"},
		{Text("😀"), "<feffd83dde00>"},
		{Text(`\abc`), `(\\abc)`},
		{Text("abc("), `(abc\()`},
		{Text("ab)c"), `(ab\)c)`},
		{Text(""), "()"},
		{Literal([]byte{0xff, 'a'}), "(\xffa)"},
		{Hex([]byte{0, 0xab, 0x0f}), "<00ab0f>"},
	}
	for _, tt := range tests {
		if got := serialize(tt.in); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestDictKeepsInsertionOrder(t *testing.T) {
	d := NewDict("Type", Name("Page"), "Parent", Ref{1}, "Count", 3)
	d.Set("Type", Name("Pages"))
	d.Set("MediaBox", Array{Int(0), Int(0), Real(10.5), Int(10)})
	want := "<</Type /Pages/Parent 1 0 R/Count 3/MediaBox [0 0 10.5 10]>>"
	if got := serialize(d); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	d.Delete("Parent")
	d.Set("Parent", Ref{2})
	keys := d.Keys()
	wantKeys := []Name{"Type", "Count", "MediaBox", "Parent"}
	if len(keys) != len(wantKeys) {
		t.Fatalf("keys = %v, want %v", keys, wantKeys)
	}
	for i := range keys {
		if keys[i] != wantKeys[i] {
			t.Fatalf("keys = %v, want %v", keys, wantKeys)
		}
	}
}

func TestDictGetAndNil(t *testing.T) {
	var nilDict *Dict
	if _, ok := nilDict.Get("X"); ok || nilDict.Len() != 0 || nilDict.Keys() != nil {
		t.Error("nil dict should be empty")
	}
	if got := serialize(nilDict); got != "<<>>" {
		t.Errorf("nil dict = %q", got)
	}
	d := NewDict("A", nil)
	if v, ok := d.Get("A"); !ok || v != (Null{}) {
		t.Errorf("Get(A) = %v, %v", v, ok)
	}
}

func TestNestedDictionaryIsInline(t *testing.T) {
	d := NewDict("Resources", NewDict("Font", NewDict("F1", Ref{4})))
	want := "<</Resources <</Font <</F1 4 0 R>>>>>>"
	if got := serialize(d); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStreamSerialization(t *testing.T) {
	s := &Stream{Dict: NewDict("Type", Name("XObject")), Data: []byte("0 0 5 5 re\nf")}
	want := "<</Type /XObject/Length 12>>\nstream\n0 0 5 5 re\nf\nendstream"
	if got := serialize(s); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := serialize(&Stream{}); got != "<</Length 0>>\nstream\n\nendstream" {
		t.Errorf("empty stream = %q", got)
	}
}

func TestStreamCompressionOverride(t *testing.T) {
	yes, no := true, false
	data := []byte("2 2 5 6 re\nf 2 2 5 6 re\nf")
	tests := []struct {
		name       string
		override   *bool
		defaultOn  bool
		compressed bool
	}{
		{"default off", nil, false, false},
		{"default on", nil, true, true},
		{"forced on", &yes, false, true},
		{"forced off", &no, true, false},
	}
	for _, tt := range tests {
		s := &Stream{Data: data, Compress: tt.override}
		out := string(s.appendTo(nil, tt.defaultOn))
		has := strings.Contains(out, "/Filter /FlateDecode")
		if has != tt.compressed {
			t.Errorf("%s: compressed = %v, want %v: %q", tt.name, has, tt.compressed, out)
		}
		if strings.Contains(out, "2 2 5 6 re") == tt.compressed {
			t.Errorf("%s: plain text visibility does not match compression", tt.name)
		}
	}
}

func TestCompressedStreamOverridesFilterKeepingPosition(t *testing.T) {
	yes := true
	s := &Stream{Dict: NewDict("Filter", Name("DCTDecode"), "Width", 1), Data: []byte("x"), Compress: &yes}
	out := string(s.appendTo(nil, false))
	if !strings.HasPrefix(out, "<</Filter /FlateDecode/Width 1/Length ") {
		t.Errorf("got %q", out)
	}
}

func TestAppendHelpers(t *testing.T) {
	buf := AppendReal([]byte("x "), 1.50)
	buf = append(buf, ' ')
	buf = Append(buf, Array{Int(1), Name("A"), nil})
	if got, want := string(buf), "x 1.5 [1 /A null]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
