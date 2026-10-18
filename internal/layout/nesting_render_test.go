package layout_test

import "testing"

func TestNestedRulesDecideTheWidthOfLaidOutBoxes(t *testing.T) {
	styles := []string{
		`div { p { width: 10px } }`,
		`p { div & { width: 10px } }`,
		`p { width: 20px; div & { width: 10px } }`,
		`p { div & { width: 10px } width: 20px }`,
		`div { & { & { p { & { width: 10px } } } } }`,
		`@media print { div { p { width: 10px } } }`,
		`div { em, p { width: 10px } }`,
		`p { a, div & { width: 10px } }`,
	}
	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			_, body := pageBody(t, onlyPage(t, renderPages(t, `
              <style>`+style+`</style>
              <div><p></p></div><p></p>
            `)))
			list := children(t, body, 2)
			if got := child(t, list[0]).Base().Width.V; got != 10 {
				t.Errorf("nested paragraph width = %g, want 10", got)
			}
			if got := list[1].Base().Width.V; got == 10 {
				t.Errorf("paragraph outside the div has width %g", got)
			}
		})
	}
}
