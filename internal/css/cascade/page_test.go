package cascade

import (
	"reflect"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/css/syntax"
)

func TestParsePageSelectors(t *testing.T) {
	nth := func(a, b int) *PageIndex { return &PageIndex{A: a, B: b} }
	tests := []struct {
		css  string
		want []PageSelector
	}{
		{"@page {}", []PageSelector{{}}},
		{"@page :left {}", []PageSelector{{Side: "left", Specificity: [3]int{0, 0, 1}}}},
		{"@page:first:left {}", []PageSelector{{Side: "left", First: true, Specificity: [3]int{0, 1, 1}}}},
		{"@page pagename {}", []PageSelector{{Name: "pagename", Specificity: [3]int{1, 0, 0}}}},
		{"@page pagename:first:right:blank {}", []PageSelector{{Side: "right", Blank: true, First: true, Name: "pagename", Specificity: [3]int{1, 2, 1}}}},
		{"@page pagename, :first {}", []PageSelector{
			{Name: "pagename", Specificity: [3]int{1, 0, 0}},
			{First: true, Specificity: [3]int{0, 1, 0}},
		}},
		{"@page :first:first {}", []PageSelector{{First: true, Specificity: [3]int{0, 2, 0}}}},
		{"@page :left:left {}", []PageSelector{{Side: "left", Specificity: [3]int{0, 0, 2}}}},
		{"@page :nth(2) {}", []PageSelector{{Index: nth(0, 2), Specificity: [3]int{0, 1, 0}}}},
		{"@page :nth(2n + 4) {}", []PageSelector{{Index: nth(2, 4), Specificity: [3]int{0, 1, 0}}}},
		{"@page :nth(3n) {}", []PageSelector{{Index: nth(3, 0), Specificity: [3]int{0, 1, 0}}}},
		{"@page :nth( n+2 ) {}", []PageSelector{{Index: nth(1, 2), Specificity: [3]int{0, 1, 0}}}},
		{"@page :nth(even) {}", []PageSelector{{Index: nth(2, 0), Specificity: [3]int{0, 1, 0}}}},
		{"@page pagename:nth(2) {}", []PageSelector{{Index: nth(0, 2), Name: "pagename", Specificity: [3]int{1, 1, 0}}}},
		{"@page :nth(2n of chapter) {}", []PageSelector{{Index: &PageIndex{A: 2, Of: "chapter"}, Specificity: [3]int{1, 1, 0}}}},
		{"@page page page {}", nil},
		{"@page :left page {}", nil},
		{"@page :left, {}", nil},
		{"@page , {}", nil},
		{"@page :left, test, {}", nil},
		{"@page :wrong {}", nil},
		{"@page :left:wrong {}", nil},
		{"@page :left:right {}", nil},
	}
	for _, tt := range tests {
		rule := syntax.ParseStylesheet(tt.css, false, false)[0].(*syntax.AtRule)
		got, ok := ParsePageSelectors(rule.Prelude)
		if ok != (tt.want != nil) || (ok && !reflect.DeepEqual(got, tt.want)) {
			t.Errorf("%s: got %+v, %v; want %+v", tt.css, got, ok, tt.want)
		}
	}
}

func TestPageSelectorMatches(t *testing.T) {
	tests := []struct {
		css  string
		page PageType
		want bool
	}{
		{"@page {}", PageType{Side: "left", Index: 3}, true},
		{"@page :first {}", PageType{Index: 0}, true},
		{"@page :first {}", PageType{Index: 1}, false},
		{"@page :right {}", PageType{Side: "left"}, false},
		{"@page :blank {}", PageType{Blank: true}, true},
		{"@page :blank {}", PageType{}, false},
		{"@page narrow {}", PageType{Name: "narrow"}, true},
		{"@page narrow {}", PageType{Name: "wide"}, false},
		{"@page :nth(2) {}", PageType{Index: 1}, true},
		{"@page :nth(2) {}", PageType{Index: 2}, false},
		{"@page :nth(2n+1) {}", PageType{Index: 4}, true},
		{"@page :nth(2n+1) {}", PageType{Index: 3}, false},
		{"@page :nth(-n+2) {}", PageType{Index: 1}, true},
		{"@page :nth(-n+2) {}", PageType{Index: 2}, false},
		{"@page :nth(1 of chapter) {}", PageType{Name: "chapter", Groups: []PageGroup{{"chapter", 0}}}, true},
		{"@page :nth(1 of chapter) {}", PageType{Name: "chapter", Groups: []PageGroup{{"chapter", 1}}}, false},
		{"@page :nth(1 of chapter) {}", PageType{Name: "other", Groups: []PageGroup{{"chapter", 0}}}, false},
	}
	for _, tt := range tests {
		rule := syntax.ParseStylesheet(tt.css, false, false)[0].(*syntax.AtRule)
		selectors, _ := ParsePageSelectors(rule.Prelude)
		if got := selectors[0].Matches(tt.page); got != tt.want {
			t.Errorf("%s matches %+v = %v, want %v", tt.css, tt.page, got, tt.want)
		}
	}
}

