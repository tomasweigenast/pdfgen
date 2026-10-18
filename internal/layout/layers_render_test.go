package layout_test

import (
	"testing"
)

func TestLayersDecideTheWidthOfLaidOutBoxes(t *testing.T) {
	styles := []string{
		`@layer { div { width: 100px } }`,
		`@layer a { div { width: 100px } }`,
		`
        div { width: 100px }
        @layer a { div { width: 200px } }`,
		`
        @layer { div { width: 200px } }
        @layer a { div { width: 100px } }`,
		`
        @layer a { div { width: 200px } }
        @layer { div { width: 100px } }`,
		`
        @layer a { div { width: 200px } }
        @layer a { div { width: 100px } }`,
		`
        @layer a { div { width: 100px } }
        @layer a.b { div { width: 200px } }`,
		`
        @layer a.b { div { width: 200px } }
        @layer a { div { width: 100px } }`,
		`
        @layer a { div { width: 200px } }
        @layer b { div { width: 100px } }`,
		`
        @layer b, a;
        @layer a { div { width: 100px } }
        @layer b { div { width: 200px } }`,
		`
        @layer b;
        @layer a { div { width: 100px } }
        @layer b { div { width: 200px } }`,
		`
        @import url(data:text/css,div{width:100px});
        @layer a { div { width: 200px } }`,
		`
        @import url(data:text/css,div{width:200px}) layer;
        @layer a { div { width: 100px } }`,
		`
        @import url(data:text/css,div{width:200px}) layer(b);
        @layer a { div { width: 100px } }`,
		`
        @import url(data:text/css,div{width:100px}) layer(a);
        @layer a.b { div { width: 200px } }`,
		`
        @import url(data:text/css,div{width:200px}) layer(a.b);
        @layer a { div { width: 100px } }`,
		`
        @layer a { div { width: 100px } }
        @import url(data:text/css,div{width:200px}) layer(a.b);`,
	}
	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			_, body := pageBody(t, onlyPage(t, renderPages(t, `
              <style>
                `+style+`
              </style>
              <div>abc</div>
            `)))
			div := child(t, body)
			if got := div.Base().Width.V; got != 100 {
				t.Errorf("width = %g, want 100", got)
			}
		})
	}
}
