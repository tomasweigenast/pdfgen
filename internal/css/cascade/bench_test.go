package cascade_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/cascade"
	"github.com/tomasweigenast/pdfgen/internal/css/props"
	"github.com/tomasweigenast/pdfgen/internal/html"
)

// BenchmarkComputeAll cascades and computes every property of a document of
// a few thousand elements with the default user agent stylesheet.
func BenchmarkComputeAll(b *testing.B) {
	var src strings.Builder
	src.WriteString(`<style>
      body { font: 12px/1.4 serif; margin: 2cm }
      section > h2 { font-size: 1.5em; margin: 1em 0 .5em }
      p.note { color: #555; padding-left: calc(1em + 2%) }
      li:nth-child(2n) { background: #eee }
      table td, table th { border: 1px solid; padding: .2em .5em }
    </style>`)
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&src, `<section id="s%d"><h2>Section %d</h2><p class="note">Text <em>with</em> <a href="#s%d">link</a></p>`, i, i, i)
		src.WriteString(`<ul><li>one<li>two<li>three</ul><table><tr><th>a<th>b<tr><td>1<td>2</table></section>`)
	}
	doc, err := html.ParseString(src.String(), "")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sf, err := cascade.ComputeAll(doc, cascade.Options{})
		if err != nil {
			b.Fatal(err)
		}
		for _, el := range doc.Elements() {
			style := sf.Get(el, "")
			for id := 0; id < props.NumProps(); id++ {
				style.Get(props.PropID(id))
			}
		}
	}
}
