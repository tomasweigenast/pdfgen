package selector

import (
	"encoding/xml"
	"testing"
)

func FuzzCompileAndMatch(f *testing.F) {
	for _, seed := range []string{"div", `#main > p.note`, `[lang|=en]`, `:nth-child(2n+1)`, `div:not(`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 16<<10 {
			t.Skip()
		}
		selectors, err := CompileString(input, nil)
		if err != nil {
			return
		}
		root := &testElement{name: xml.Name{Local: "main"}, attrs: map[xml.Name]string{}, html: true}
		child := &testElement{
			name: xml.Name{Local: "div"}, parent: root, index: 0, html: true,
			attrs: map[xml.Name]string{{Local: "id"}: "content", {Local: "class"}: "note"},
		}
		root.children = []*testElement{child}
		for i := range selectors {
			selectors[i].Match(root)
			selectors[i].Match(child)
		}
	})
}
