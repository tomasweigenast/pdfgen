package syntax

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStylesheetBytesEncodingDetection(t *testing.T) {
	cases := loadSuite(t, "stylesheet_bytes.json")
	for i, c := range cases {
		var in struct {
			CSSBytes    string  `json:"css_bytes"`
			Protocol    *string `json:"protocol_encoding"`
			Environment *string `json:"environment_encoding"`
		}
		if err := json.Unmarshal(c.input, &in); err != nil {
			t.Fatal(err)
		}
		raw := make([]byte, 0, len(in.CSSBytes))
		for _, r := range in.CSSBytes {
			raw = append(raw, byte(r))
		}
		var protocol string
		if in.Protocol != nil {
			protocol = *in.Protocol
		}
		var env *Encoding
		if in.Environment != nil {
			env = LookupEncoding(*in.Environment)
		}
		rules, enc := ParseStylesheetBytes(raw, protocol, env, true, true)
		got := normalize([]any{toJSON(rules), enc.Name})
		if !reflect.DeepEqual(got, c.expected) {
			t.Errorf("case %d %q:\n got  %#v\n want %#v", i, in.CSSBytes, got, c.expected)
		}
	}
	t.Logf("%d cases", len(cases))
}

func TestLookupEncodingUnknownLabel(t *testing.T) {
	if LookupEncoding("kamoulox") != nil {
		t.Error("unknown label resolved")
	}
	if enc := LookupEncoding(" Latin1 "); enc == nil || enc.Name != "windows-1252" {
		t.Errorf("got %v", enc)
	}
}
