package syntax

import "testing"

func TestSerializeURLPreservesEscapedControlCharacters(t *testing.T) {
	nodes := ParseComponentValueList(`url(\1)`, false)
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes; want 1", len(nodes))
	}

	serialized := Serialize(nodes)
	reparsed := ParseComponentValueList(serialized, false)
	if got := Serialize(reparsed); got != serialized {
		t.Fatalf("serialization changed after tokenizing %q: got %q", serialized, got)
	}

	want := "url(\\1 )"
	if serialized != want {
		t.Fatalf("got %q; want %q", serialized, want)
	}
}
