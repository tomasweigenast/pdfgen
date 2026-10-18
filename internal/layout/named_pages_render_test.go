package layout_test

import "testing"

func TestPageNameIsInheritedAndAutoComputesToEmpty(t *testing.T) {
	page := onlyPage(t, renderPages(t, `
      <style>
        @page NARRow { size: landscape }
        div { page: AUTO }
        p { page: NARRow }
      </style>
      <div><p><span>a</span></p></div>
    `))
	html, body := pageBody(t, page)
	div := child(t, body)
	p := child(t, div)
	span := child(t, p)
	for _, tt := range []struct {
		name string
		got  string
		want string
	}{
		{"html", string(html.Base().Style.Page()), ""},
		{"body", string(body.Base().Style.Page()), ""},
		{"div", string(div.Base().Style.Page()), ""},
		{"p", string(p.Base().Style.Page()), "NARRow"},
		{"span", string(span.Base().Style.Page()), "NARRow"},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: page = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}
