package html

import "testing"

func FuzzParseDocument(f *testing.F) {
	for _, seed := range []string{
		`<!doctype html><title>sample</title><p>text`,
		`<html><body><img src="x"></body></html>`,
		`<div><span></div>`, `<!-- unfinished`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64<<10 {
			t.Skip()
		}
		ParseString(input, "")
	})
}
